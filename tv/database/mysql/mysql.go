package mysql

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
	"strings"
	"database/sql"
)

import (
	"github.com/l4go/task"
	_ "github.com/go-sql-driver/mysql"
)

import (
	"github.com/hinoshiba/gwyneth/config"
	"github.com/hinoshiba/gwyneth/structs"

	"github.com/hinoshiba/gwyneth/tv/errors"
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
	rows, err := self.db.Query("SELECT * FROM source_type WHERE id = ? LIMIT 1", id.Value())
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

	return self.getSource(id)
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
	rows, err := self.db.Query("SELECT * FROM source WHERE id = ? LIMIT 1", id.Value())
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

func (self *Session) getArticle(id *structs.Id) (*structs.Article, error) {
	as, err := self.query4article("SELECT id, src_id, title, body, link, timestamp, raw FROM article WHERE id = ? AND disable <> 1 LIMIT 1", id.Value())
	if err != nil {
		return nil, err
	}
	return as[0], nil
}

func (self *Session) AddArticle(title string, body string, link string, unixtime int64, raw string, src_id *structs.Id) (*structs.Article, error){
	self.mtx.Lock()
	defer self.mtx.Unlock()

	q := "SELECT id, src_id, title, body, link, timestamp, raw FROM article WHERE title = ? AND body = ? AND link = ? AND src_id = ?"
	as, err := self.query4article(q, title, body, link, src_id.Value())
	if err != nil {
		return nil, err
	}
	if !(len(as) < 1) {
		return as[0], errors.ERR_ALREADY_EXIST_ARTICLE
	}

	id := structs.NewId(nil)
	_, err = self.db.ExecContext(self.msn.AsContext(),
		"INSERT INTO article (id, src_id, title, body, link, timestamp, raw) VALUES (?, ?, ?, ?, ?, FROM_UNIXTIME(?), ?)",
					id.Value(), src_id.Value(), title, body, link, unixtime, raw)
	if err != nil {
		return nil, err
	}
	if err := self.addFeed(src_id, id); err != nil {
		return nil, err
	}

	return self.getArticle(id)
}

func (self *Session) LookupArticles(t_kw string, b_kw string, src_ids []*structs.Id, start int64, end int64, limit int64) ([]*structs.Article, error) {
	self.mtx.RLock()
	defer self.mtx.RUnlock()

	q := "SELECT id, src_id, title, body, link, timestamp, raw FROM article WHERE disable <> 1"
	args := make([]any, 0)
	if t_kw != "" {
		q += " AND LIKE ?"
		args = append(args, "%" + t_kw + "%")
	}
	if b_kw != "" {
		q += " AND LIKE ?"
		args = append(args, "%" + b_kw + "%")
	}
	if start > 0 {
		q += " AND timestamp >= ?"
		args = append(args, start)
	}
	if end > 0 {
		q += " AND timestamp <= ?"
		args = append(args, end)
	}
	if len(src_ids) > 0 {
		q += " AND src_id IN (?" + strings.Repeat(", ?", len(src_ids) - 1) + ")"
		for _, src_id := range src_ids {
			args = append(args, src_id)
		}
	}
	if limit > 0 {
		q += " LIMIT ?"
		args = append(args, limit)
	}

	return self.query4article(q, args...)
}

func (self *Session) RemoveArticle(id *structs.Id) error {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	if _, err := self.getArticle(id); err != nil {
		return err
	}
	_, err := self.db.ExecContext(self.msn.AsContext(),
		"UPDATE article SET disable = 1 WHERE id = ?", id.Value())
	return err
}

func (self *Session) GetFeed(src_id *structs.Id, limit int64) ([]*structs.Article, error) {
	self.mtx.RLock()
	defer self.mtx.RUnlock()

	if _, err := self.getSource(src_id); err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = 100
	}

	var q string = `
SELECT a.id, f.src_id, a.title, a.body, a.link, a.timestamp, a.raw
FROM article a
JOIN feed f ON a.id = f.article_id
WHERE f.src_id = ? AND a.disable <> 1
ORDER BY f.timestamp DESC
LIMIT ?
`

	return self.query4article(q, src_id.Value(), limit)
}

func (self *Session) RemoveFeedEntry(src_id *structs.Id, article_id *structs.Id) error {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	_, err := self.db.ExecContext(self.msn.AsContext(),
		"UPDATE feed SET disable = 1 WHERE src_id = ? AND article_id = ?", src_id.Value(), article_id.Value())
	return err
}

func (self *Session) BindFeed(src_id *structs.Id, article_id *structs.Id) error {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	return self.addFeed(src_id, article_id)
}

func (self *Session) addFeed(src_id *structs.Id, article_id *structs.Id) error {
	_, err := self.db.ExecContext(self.msn.AsContext(),
					"INSERT INTO feed (src_id, article_id) VALUES (?, ?)",
											src_id.Value(), article_id.Value())
	return err
}

func (self *Session) query4article(q string, args ...any) ([]*structs.Article, error) {
	rows, err := self.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	articles := []*structs.Article{}
	src_cache := make(map[string]*structs.Source)
	for rows.Next() {
		var id_base []byte
		var src_id_base []byte
		var title string
		var body string
		var link string
		var t_stamp time.Time
		var raw string

		if err = rows.Scan(&id_base, &src_id_base, &title, &body, &link, &t_stamp, &raw); err != nil {
			return nil, err
		}
		id := structs.NewId(id_base)
		src_id := structs.NewId(src_id_base)

		src, ok := src_cache[src_id.String()]
		if !ok {
			var err error
			src, err = self.getSource(src_id)
			if err != nil {
				return nil, err
			}

			src_cache[src_id.String()] = src
		}

		if err = rows.Err(); err != nil {
			return nil, err
		}
		articles = append(articles, structs.NewArticle(id, src, title, body, link, t_stamp.Unix(), raw))
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return articles, nil
}

func (self *Session) AddAction(name string, cmd string) (*structs.Action, error) {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	id := structs.NewId(nil)

	_, err := self.db.ExecContext(self.msn.AsContext(),
		"INSERT INTO action (id, name, command) VALUES (?, ?, ?)", id.Value(), name, cmd)
	if err != nil {
		return nil, err
	}

	return structs.NewAction(id, name, cmd), nil
}

func (self *Session) GetActions() ([]*structs.Action, error) {
	self.mtx.RLock()
	defer self.mtx.RUnlock()

	rows, err := self.db.Query("SELECT * FROM action")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	actions := []*structs.Action{}
	for rows.Next() {
		var id_base []byte
		var name string
		var cmd string

		err := rows.Scan(&id_base, &name, &cmd)
		if err != nil {
			return nil, err
		}
		id := structs.NewId(id_base)
		action := structs.NewAction(id, name, cmd)

		actions = append(actions, action)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return actions, nil
}

func (self *Session) GetAction(id *structs.Id) (*structs.Action, error) {
	self.mtx.RLock()
	defer self.mtx.RUnlock()

	return self.getAction(id)
}

func (self *Session) getAction(id *structs.Id) (*structs.Action, error) {
	rows, err := self.db.Query("SELECT * FROM action WHERE id = ? LIMIT 1", id.Value())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var id_base []byte
	var name string
	var cmd string

	for rows.Next() {
		if err = rows.Scan(&id_base, &name, &cmd); err != nil {
			return nil, err
		}
		id = structs.NewId(id_base)
		action := structs.NewAction(id, name, cmd)

		if err = rows.Err(); err != nil {
			return nil, err
		}
		return action, nil
	}
	return nil, fmt.Errorf("cannot find the action")
}

func (self *Session) DeleteAction(id *structs.Id) error {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	if _, err := self.getAction(id); err != nil {
		return err
	}

	_, err := self.db.ExecContext(self.msn.AsContext(),
		"DELETE FROM action WHERE id = ?", id.Value())
	return err
}

func (self *Session) AddFilter(title string, regex_title bool, body string, regex_body bool, action_id *structs.Id) (*structs.Filter, error) {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	id := structs.NewId(nil)

	_, err := self.db.ExecContext(self.msn.AsContext(),
		"INSERT INTO filter (id, val_title, is_regex_title, val_body, is_regex_body, action_id) VALUES (?, ?, ?, ?, ?, ?)",
			id.Value(), title, regex_title, body, regex_body, action_id.Value())
	if err != nil {
		return nil, err
	}

	return self.getFilter(id)
}

func (self *Session) UpdateFilterAction(id *structs.Id, action_id *structs.Id) (*structs.Filter, error) {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	if _, err := self.getFilter(id); err != nil {
		return nil, err
	}

	_, err := self.db.ExecContext(self.msn.AsContext(),
		"UPDATE filter SET action_id = ? WHERE id = ?", action_id.Value(), id.Value())
	if err != nil {
		return nil, err
	}
	return self.getFilter(id)
}

func (self *Session) GetFilter(id *structs.Id) (*structs.Filter, error) {
	self.mtx.RLock()
	defer self.mtx.RUnlock()

	return self.getFilter(id)
}

func (self *Session) getFilter(id *structs.Id) (*structs.Filter, error) {
	rows, err := self.db.Query("SELECT * FROM filter WHERE id = ? LIMIT 1", id.Value())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var id_base        []byte
	var val_title      string
	var is_regex_title bool
	var val_body       string
	var is_regex_body  bool
	var action_id_base []byte
	for rows.Next() {
		err := rows.Scan(&id_base, &val_title, &is_regex_title, &val_body, &is_regex_body, &action_id_base)
		if err != nil {
			return nil, err
		}
		id := structs.NewId(id_base)

		action_id := structs.NewId(action_id_base)
		action, err := self.getAction(action_id)
		if err != nil {
			return nil, err
		}

		f := structs.NewFilter(id, val_title, is_regex_title, val_body, is_regex_body, action)

		if err = rows.Err(); err != nil {
			return nil, err
		}
		return f, nil
	}
	return nil, fmt.Errorf("cannot find the filter.")
}

func (self *Session) GetFilters() ([]*structs.Filter, error) {
	self.mtx.RLock()
	defer self.mtx.RUnlock()

	rows, err := self.db.Query("SELECT * FROM filter")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	f_s := []*structs.Filter{}
	action_cache := make(map[string]*structs.Action)
	for rows.Next() {
		var id_base        []byte
		var val_title      string
		var is_regex_title bool
		var val_body       string
		var is_regex_body  bool
		var action_id_base []byte

		err := rows.Scan(&id_base, &val_title, &is_regex_title, &val_body, &is_regex_body, &action_id_base)
		if err != nil {
			return nil, err
		}
		id := structs.NewId(id_base)
		action_id := structs.NewId(action_id_base)

		action, ok := action_cache[action_id.String()]
		if !ok {
			var err error
			action, err = self.getAction(action_id)
			if err != nil {
				return nil, err
			}

			action_cache[action_id.String()] = action
		}

		f_s = append(f_s, structs.NewFilter(id, val_title, is_regex_title, val_body, is_regex_body, action))
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return f_s, nil
}

func (self *Session) DeleteFilter(id *structs.Id) error {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	if _, err := self.getFilter(id); err != nil {
		return err
	}

	_, err := self.db.ExecContext(self.msn.AsContext(),
		"DELETE FROM filter WHERE id = ?", id.Value())
	return err
}

func (self *Session) BindFilter(src_id *structs.Id, f_id *structs.Id) error {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	_, err := self.db.ExecContext(self.msn.AsContext(),
		"INSERT INTO src_filter_map (src_id, filter_id) VALUES (?, ?)",
		src_id.Value(), f_id.Value())
	return err
}

func (self *Session) UnBindFilter(src_id *structs.Id, f_id *structs.Id) error {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	_, err := self.db.ExecContext(self.msn.AsContext(),
			"DELETE FROM src_filter_map WHERE src_id = ? AND filter_id = ?",
												src_id.Value(), f_id.Value())
	return err
}

func (self *Session) GetFilterOnSource(src_id *structs.Id) ([]*structs.Filter, error) {
	self.mtx.RLock()
	defer self.mtx.RUnlock()

	rows, err := self.db.Query("SELECT filter_id FROM src_filter_map WHERE src_id = ?", src_id.Value())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fs := []*structs.Filter{}
	for rows.Next() {
		var id_base        []byte

		if err := rows.Scan(&id_base); err != nil {
			return nil, err
		}

		id := structs.NewId(id_base)
		f, err := self.getFilter(id)
		if err != nil {
			return nil, err
		}
		fs = append(fs, f)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return fs, nil
}

func (self *Session) GetSourceWithEnabledFilter(f_id *structs.Id) ([]*structs.Source, error) {
	self.mtx.RLock()
	defer self.mtx.RUnlock()

	rows, err := self.db.Query("SELECT filter_id FROM src_filter_map WHERE filter_id = ?", f_id.Value())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	srcs := []*structs.Source{}
	for rows.Next() {
		var id_base        []byte

		if err := rows.Scan(&id_base); err != nil {
			return nil, err
		}

		id := structs.NewId(id_base)
		src, err := self.getSource(id)
		if err != nil {
			return nil, err
		}
		srcs = append(srcs, src)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return srcs, nil
}
