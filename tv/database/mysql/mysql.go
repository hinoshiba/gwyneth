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
	"github.com/hinoshiba/gwyneth/tv/database/structs"
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

	self.mtx.Lock()
	defer self.mtx.Unlock()

	self.msn.Cancel()

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

func (self *Session) DeleteSourceType(id *structs.Id) error {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	_, err := self.db.ExecContext(self.msn.AsContext(),
		"DELETE FROM source_type WHERE id = ?", id.Value())
	return err
}
