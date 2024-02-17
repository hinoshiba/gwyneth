package mysql

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
	"database/sql"
)

import (
	"github.com/l4go/task"
	_ "github.com/go-sql-driver/mysql"
)

import (
	"github.com/hinoshiba/gwyneth/config"
	"github.com/hinoshiba/gwyneth/structs"
)

const (
	MAX_RETRY int = 7
)

type Session struct {
	db      *sql.DB
	msn     *task.Mission
	mtx     *sync.RWMutex
}

func NewSession(msn *task.Mission, cfg *config.Database) (*Session, error) {
	db_auth := fmt.Sprintf("%s:%s@(%s:%d)/%s?parseTime=true", cfg.User, cfg.Pass,
														cfg.Host, cfg.Port, cfg.DB)

	self := &Session {
		msn: msn,
		mtx: new(sync.RWMutex),
	}
	if err := self.connect(db_auth); err != nil {
		self.Close()
		return nil, err
	}

	return self, nil
}

func (self *Session) connect(db_auth string) error {
	wait_sec := 15

	self.mtx.Lock()
	defer self.mtx.Unlock()

	cnt := 0
	tc := time.NewTicker(time.Second)
	defer tc.Stop()

	for {
		self.db = nil

		if cnt >= MAX_RETRY {
			self.msn.Cancel()
			break
		}
		cnt++

		select {
		case <- self.msn.RecvDone():
			return nil
		case <- self.msn.RecvCancel():
			return nil
		case <- tc.C:
			tc.Reset(time.Second * time.Duration(wait_sec))
		}

		db, err := sql.Open("mysql", db_auth)
		if err != nil {
			msg := fmt.Sprintf("Failed: connect to DB: %s, trying reconnect (%d/%d) after %d sec.",
																		err, cnt, MAX_RETRY, wait_sec)
			slog.Warn(msg)
			continue
		}
		self.db = db

		if err := self.init(); err != nil {
			msg := fmt.Sprintf("Failed: connect to a DB: %s, trying reconnect (%d/%d) after %d sec.",
											err, cnt, MAX_RETRY, wait_sec)
			slog.Warn(msg)
			continue
		}

		slog.Info("database connected.")
		self.wakeup_pinger()
		return nil
	}
	return fmt.Errorf("cannot connect to a DB.")
}

func (self *Session) Close() error {
	defer self.msn.Done()

	self.msn.Cancel()

	self.mtx.Lock()
	defer self.mtx.Unlock()

	err := self.db.Close()
	self.db = nil
	return err
}

func (self *Session) wakeup_pinger() {
	go func(msn *task.Mission) {
		defer msn.Done()

		tc := time.NewTicker(time.Second * time.Duration(30))
		defer tc.Stop()

		for {
			select {
			case <- msn.RecvCancel():
				return
			case <- tc.C:
				self.mtx.Lock()
				go func() {
					defer self.mtx.Unlock()

					if err := self.db.PingContext(msn.AsContext()); err != nil {
						if task.IsCanceled(msn) {
							return
						}
						slog.Warn("DB connection is lost.")
						msn.Cancel()
						return
					}
				}()
			}
		}
	}(self.msn.New())
}

func (self *Session) init() error {
	rows, err := self.db.QueryContext(self.msn.AsContext(), "SHOW TABLES")
	if err != nil {
		return err
	}

	order, d := make_table_dict()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return err
		}

		if _, ok := d[name]; !ok {
			continue
		}
		delete(d, name)
	}

	for _, name := range order {
		structs, ok := d[name]
		if !ok {
			continue
		}

		param := "ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci"
		query := fmt.Sprintf("CREATE TABLE %s (%s) %s", name, structs, param)
		if _, err := self.db.ExecContext(self.msn.AsContext(), query); err != nil {
			return fmt.Errorf("%s: '%s'", err, query)
		}
	}
	return nil
}

func (self *Session) AddSourceType(name string, command string, is_user_creation bool) (*structs.SourceType, error) {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	id := structs.NewId(nil)

	_, err := self.db.ExecContext(self.msn.AsContext(),
		"INSERT INTO source_type (id, name, command, user_create) VALUES (?, ?, ?, ?)",
		id.Value(), name, command, is_user_creation)
	if err != nil {
		return nil, err
	}

	return structs.NewSourceType(id, name, command, is_user_creation), nil
}

func (self *Session) GetSourceTypes() ([]*structs.SourceType, error) {
	self.mtx.RLock()
	defer self.mtx.RUnlock()

	rows, err := self.db.Query("SELECT * FROM source_type")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	source_types := []*structs.SourceType{}
	for rows.Next() {
		var id_base []byte
		var name string
		var cmd string
		var user_create bool

		err := rows.Scan(&id_base, &name, &cmd, &user_create)
		if err != nil {
			return nil, err
		}
		id := structs.NewId(id_base)
		st := structs.NewSourceType(id, name, cmd, user_create)

		source_types = append(source_types, st)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return source_types, nil
}

func (self *Session) GetSourceType(id *structs.Id) (*structs.SourceType, error) {
	self.mtx.RLock()
	defer self.mtx.RUnlock()

	return self.getSourceType(id)
}

func (self *Session) getSourceType(id *structs.Id) (*structs.SourceType, error) {
	rows, err := self.db.Query("SELECT * FROM source_type WHERE id = ?", id.Value())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var id_base []byte
	var name string
	var cmd string
	var user_create bool

	for rows.Next() {
		if err = rows.Scan(&id_base, &name, &cmd, &user_create); err != nil {
			return nil, err
		}
		id = structs.NewId(id_base)
		st := structs.NewSourceType(id, name, cmd, user_create)

		if err = rows.Err(); err != nil {
			return nil, err
		}
		return st, nil
	}
	return nil, fmt.Errorf("cannot find the source type.")
}

func (self *Session) DeleteSourceType(id *structs.Id) error {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	if _, err := self.getSourceType(id); err != nil {
		return err
	}

	_, err := self.db.ExecContext(self.msn.AsContext(),
		"DELETE FROM source_type WHERE id = ?", id.Value())
	return err
}

func (self *Session) AddSource(title string, src_type_id *structs.Id, source string) (*structs.Source, error) {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	id := structs.NewId(nil)

	_, err := self.db.ExecContext(self.msn.AsContext(),
		"INSERT INTO source (id, title, type, source) VALUES (?, ?, ?, ?)",
								id.Value(), title, src_type_id.Value(), source)
	if err != nil {
		return nil, err
	}

	src_type, err := self.getSourceType(src_type_id)
	if err != nil {
		return nil, err
	}

	return structs.NewSource(id, title, src_type, source), nil
}

func (self *Session) GetSources() ([]*structs.Source, error) {
	self.mtx.RLock()
	defer self.mtx.RUnlock()

	rows, err := self.db.Query("SELECT * FROM source")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	srcs := []*structs.Source{}
	st_cache := make(map[string]*structs.SourceType)
	for rows.Next() {
		var id_base []byte
		var title string
		var source_type_id_base []byte
		var source string

		err := rows.Scan(&id_base, &title, &source_type_id_base, &source)
		if err != nil {
			return nil, err
		}
		id := structs.NewId(id_base)
		source_type_id := structs.NewId(source_type_id_base)

		st, ok := st_cache[source_type_id.String()]
		if !ok {
			var err error
			st, err = self.getSourceType(source_type_id)
			if err != nil {
				return nil, err
			}

			st_cache[source_type_id.String()] = st
		}

		srcs = append(srcs, structs.NewSource(id, title, st, source))
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return srcs, nil
}

func (self *Session) FindSource(kw string) ([]*structs.Source, error) {
	self.mtx.RLock()
	defer self.mtx.RUnlock()

	rows, err := self.db.Query("SELECT * FROM source WHERE title = ?", kw)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	srcs := []*structs.Source{}
	st_cache := make(map[string]*structs.SourceType)
	for rows.Next() {
		var id_base []byte
		var title string
		var source_type_id_base []byte
		var source string

		err := rows.Scan(&id_base, &title, &source_type_id_base, &source)
		if err != nil {
			return nil, err
		}
		id := structs.NewId(id_base)
		source_type_id := structs.NewId(source_type_id_base)

		st, ok := st_cache[source_type_id.String()]
		if !ok {
			var err error
			st, err = self.getSourceType(source_type_id)
			if err != nil {
				return nil, err
			}

			st_cache[source_type_id.String()] = st
		}

		srcs = append(srcs, structs.NewSource(id, title, st, source))
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return srcs, nil
}

func (self *Session) GetSource(id *structs.Id) (*structs.Source, error) {
	self.mtx.RLock()
	defer self.mtx.RUnlock()

	return self.getSource(id)
}

func (self *Session) getSource(id *structs.Id) (*structs.Source, error) {
	rows, err := self.db.Query("SELECT * FROM source WHERE id = ?", id.Value())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var id_base []byte
	var title string
	var source_type_id_base []byte
	var source string

	for rows.Next() {
		if err = rows.Scan(&id_base, &title, &source_type_id_base, &source); err != nil {
			return nil, err
		}
		id = structs.NewId(id_base)
		source_type_id := structs.NewId(source_type_id_base)

		st, err := self.getSourceType(source_type_id)
		if err != nil {
			return nil, err
		}

		if err = rows.Err(); err != nil {
			return nil, err
		}
		return structs.NewSource(id, title, st, source), nil
	}
	return nil, fmt.Errorf("cannot find the source.")
}

func (self *Session) DeleteSource(id *structs.Id) error {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	if _, err := self.getSource(id); err != nil {
		return err
	}

	_, err := self.db.ExecContext(self.msn.AsContext(),
		"DELETE FROM source WHERE id = ?", id.Value())
	return err
}

func (self *Session) AddArticle(title string, body string, link string, timestamp uint64, raw string, src_id *structs.Id) (*structs.Article, error){
//too: WIP
}

func (self *Session) LookupArticles(kw string, src_ids []*structs.Id, start uint64, end uint64, limit int64) ([]*structs.Article, error) {
}

func (self *Session) RemoveArticle(id *structs.Id) error {
}
/*
	AddArticle()
	BatchAddArticle()
	LookupArticles()
	RemoveArticle()
	AddArticle(string, string, string, uint64, string, *structs.Source) (*structs.Article, error)
	//BatchAddArticle()
	LookupArticles(string)
	RemoveArticle(*structs.Id) error

id BINARY(16) NOT NULL,
src_id BINARY(16) NOT NULL,
title LONGTEXT NOT NULL,
body LONGTEXT NOT NULL,
link TEXT NOT NULL,
timestap TIMESTAMP NOT NULL,
raw LONGTEXT NOT NULL,
disable BOOLEAN NOT NULL DEFAULT 0,
PRIMARY KEY (id),
FOREIGN KEY (src_id) REFERENCES source(id)

	*/
