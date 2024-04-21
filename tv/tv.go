package tv

import (
	"sync"
//	"log/slog"
)

import (
	"github.com/l4go/task"
)

import (
	"github.com/hinoshiba/gwyneth/config"
	"github.com/hinoshiba/gwyneth/tv/database"
	"github.com/hinoshiba/gwyneth/structs"
)

type TimeVortex struct {
	db  database.Session

	msn *task.Mission
	mtx *sync.RWMutex
}

func New(msn *task.Mission, cfg *config.Config) (*TimeVortex, error) {
	db, err := database.Connect(msn.New(), cfg.Database)
	if err != nil {
		msn.Done()
		return nil, err
	}

	return &TimeVortex{
		db: db,
		msn: msn,
		mtx: new(sync.RWMutex),
	}, nil
}

func (self *TimeVortex) Close() error {
	defer self.msn.Done()

	self.msn.Cancel()

	self.mtx.Lock()
	defer self.mtx.Unlock()

	return self.db.Close()
}

func (self *TimeVortex) AddSourceType(name string, cmd string, is_user_creation bool) (*structs.SourceType, error) {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	return self.addSourceType(name, cmd, is_user_creation)
}

func (self *TimeVortex) addSourceType(name string, cmd string, is_user_creation bool) (*structs.SourceType, error) {
	return self.db.AddSourceType(name, cmd, is_user_creation)
}

func (self *TimeVortex) GetSourceTypes() ([]*structs.SourceType, error) {
	self.mtx.RLock()
	defer self.mtx.RUnlock()

	return self.getSourceTypes()
}

func (self *TimeVortex) getSourceTypes() ([]*structs.SourceType, error) {
	return self.db.GetSourceTypes()
}

func (self *TimeVortex) GetSourceType(id *structs.Id) (*structs.SourceType, error) {
	self.mtx.RLock()
	defer self.mtx.RUnlock()

	return self.getSourceType(id)
}

func (self *TimeVortex) getSourceType(id *structs.Id) (*structs.SourceType, error) {
	return self.db.GetSourceType(id)
}

func (self *TimeVortex) DeleteSourceType(id *structs.Id) error {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	return self.deleteSourceType(id)
}

func (self *TimeVortex) deleteSourceType(id *structs.Id) error {
	return self.db.DeleteSourceType(id)
}

func (self *TimeVortex) AddSource(title string, src_type_id *structs.Id, val string) (*structs.Source, error) {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	return self.addSource(title, src_type_id, val)
}

func (self *TimeVortex) addSource(title string, src_type_id *structs.Id, val string) (*structs.Source, error) {
	return self.db.AddSource(title, src_type_id, val)
}

func (self *TimeVortex) GetSources() ([]*structs.Source, error) {
	self.mtx.RLock()
	defer self.mtx.RUnlock()

	return self.getSources()
}

func (self *TimeVortex) getSources() ([]*structs.Source, error) {
	return self.db.GetSources()
}

func (self *TimeVortex) FindSource(kw string) ([]*structs.Source, error) {
	self.mtx.RLock()
	defer self.mtx.RUnlock()

	return self.findSource(kw)
}

func (self *TimeVortex) findSource(kw string) ([]*structs.Source, error) {
	return self.db.FindSource(kw)
}

func (self *TimeVortex) GetSource(id *structs.Id) (*structs.Source, error) {
	self.mtx.RLock()
	defer self.mtx.RUnlock()

	return self.getSource(id)
}

func (self *TimeVortex) getSource(id *structs.Id) (*structs.Source, error) {
	return self.db.GetSource(id)
}

func (self *TimeVortex) DeleteSource(id *structs.Id) error {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	return self.deleteSource(id)
}

func (self *TimeVortex) deleteSource(id *structs.Id) error {
	return self.db.DeleteSource(id)
}

func (self *TimeVortex) AddArticle(title string, body string, link string, utime int64, raw string, src_id *structs.Id) (*structs.Article, error){
	self.mtx.Lock()
	defer self.mtx.Unlock()

	return self.db.AddArticle(title, body, link, utime, raw, src_id)
}

func (self *TimeVortex) RemoveArticle(id *structs.Id) error {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	return self.db.RemoveArticle(id)
}

func (self *TimeVortex) LookupArticles(t_kw string, b_kw string, src_ids []*structs.Id, start int64, end int64, limit int64) ([]*structs.Article, error) {
	self.mtx.RLock()
	defer self.mtx.RUnlock()

	return self.db.LookupArticles(t_kw, b_kw, src_ids, start, end, limit)
}

func (self *TimeVortex) GetFeed(src_id *structs.Id, limit int64) ([]*structs.Article, error) {
	self.mtx.RLock()
	defer self.mtx.RUnlock()

	return self.db.GetFeed(src_id, limit)
}

func (self *TimeVortex) RemoveFeedEntry(src_id *structs.Id, article_id *structs.Id) error {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	return self.db.RemoveFeedEntry(src_id, article_id)
}

func (self *TimeVortex) AddAction(name string, cmd string) (*structs.Action, error) {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	return self.db.AddAction(name, cmd)
}

func (self *TimeVortex) GetAction(id *structs.Id) (*structs.Action, error) {
	self.mtx.RLock()
	defer self.mtx.RUnlock()

	return self.db.GetAction(id)
}

func (self *TimeVortex) GetActions() ([]*structs.Action, error) {
	self.mtx.RLock()
	defer self.mtx.RUnlock()

	return self.db.GetActions()
}

func (self *TimeVortex) DeleteAction(id *structs.Id) error {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	return self.db.DeleteAction(id)
}

func (self *TimeVortex) AddFilter(title string, regex_title bool, body string, regex_body bool, action_id *structs.Id) (*structs.Filter, error) {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	return self.db.AddFilter(title, regex_title, body, regex_body, action_id)
}

func (self *TimeVortex) UpdateFilterAction(id *structs.Id, action_id *structs.Id) (*structs.Filter, error) {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	return self.db.UpdateFilterAction(id, action_id)
}

func (self *TimeVortex) GetFilter(id *structs.Id) (*structs.Filter, error) {
	self.mtx.RLock()
	defer self.mtx.RUnlock()

	return self.db.GetFilter(id)
}

func (self *TimeVortex) GetFilters() ([]*structs.Filter, error) {
	self.mtx.RLock()
	defer self.mtx.RUnlock()

	return self.db.GetFilters()
}

func (self *TimeVortex) DeleteFilter(id *structs.Id) error {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	return self.db.DeleteFilter(id)
}
