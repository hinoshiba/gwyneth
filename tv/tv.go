package tv

import (
	"sync"
)

import (
	"github.com/l4go/task"
)

import (
	"github.com/hinoshiba/gwyneth/config"
	"github.com/hinoshiba/gwyneth/tv/database"
	"github.com/hinoshiba/gwyneth/model"
	"github.com/hinoshiba/gwyneth/filter"
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

func (self *TimeVortex) AddSourceType(name string, cmd string, is_user_creation bool) (*model.SourceType, error) {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	return self.addSourceType(name, cmd, is_user_creation)
}

func (self *TimeVortex) addSourceType(name string, cmd string, is_user_creation bool) (*model.SourceType, error) {
	return self.db.AddSourceType(name, cmd, is_user_creation)
}

func (self *TimeVortex) GetSourceTypes() ([]*model.SourceType, error) {
	self.mtx.RLock()
	defer self.mtx.RUnlock()

	return self.getSourceTypes()
}

func (self *TimeVortex) getSourceTypes() ([]*model.SourceType, error) {
	return self.db.GetSourceTypes()
}

func (self *TimeVortex) GetSourceType(id *model.Id) (*model.SourceType, error) {
	self.mtx.RLock()
	defer self.mtx.RUnlock()

	return self.getSourceType(id)
}

func (self *TimeVortex) getSourceType(id *model.Id) (*model.SourceType, error) {
	return self.db.GetSourceType(id)
}

func (self *TimeVortex) DeleteSourceType(id *model.Id) error {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	return self.deleteSourceType(id)
}

func (self *TimeVortex) deleteSourceType(id *model.Id) error {
	return self.db.DeleteSourceType(id)
}

func (self *TimeVortex) AddSource(title string, src_type_id *model.Id, val string) (*model.Source, error) {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	return self.addSource(title, src_type_id, val)
}

func (self *TimeVortex) addSource(title string, src_type_id *model.Id, val string) (*model.Source, error) {
	return self.db.AddSource(title, src_type_id, val)
}

func (self *TimeVortex) GetSources() ([]*model.Source, error) {
	self.mtx.RLock()
	defer self.mtx.RUnlock()

	return self.getSources()
}

func (self *TimeVortex) getSources() ([]*model.Source, error) {
	return self.db.GetSources()
}

func (self *TimeVortex) FindSource(kw string) ([]*model.Source, error) {
	self.mtx.RLock()
	defer self.mtx.RUnlock()

	return self.findSource(kw)
}

func (self *TimeVortex) findSource(kw string) ([]*model.Source, error) {
	return self.db.FindSource(kw)
}

func (self *TimeVortex) GetSource(id *model.Id) (*model.Source, error) {
	self.mtx.RLock()
	defer self.mtx.RUnlock()

	return self.getSource(id)
}

func (self *TimeVortex) getSource(id *model.Id) (*model.Source, error) {
	return self.db.GetSource(id)
}

func (self *TimeVortex) RemoveSource(id *model.Id) error {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	return self.removeSource(id)
}

func (self *TimeVortex) removeSource(id *model.Id) error {
	return self.db.RemoveSource(id)
}

func (self *TimeVortex) PauseSource(id *model.Id) error {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	return self.pauseSource(id)
}

func (self *TimeVortex) pauseSource(id *model.Id) error {
	return self.db.PauseSource(id)
}

func (self *TimeVortex) ResumeSource(id *model.Id) error {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	return self.resumeSource(id)
}

func (self *TimeVortex) resumeSource(id *model.Id) error {
	return self.db.ResumeSource(id)
}

func (self *TimeVortex) AddArticle(title string, body string, link string, utime int64, raw string, src_id *model.Id) (*model.Article, error){
	self.mtx.Lock()
	defer self.mtx.Unlock()

	return self.db.AddArticle(title, body, link, utime, raw, src_id)
}

func (self *TimeVortex) RemoveArticle(id *model.Id) error {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	return self.db.RemoveArticle(id)
}

func (self *TimeVortex) LookupArticles(t_kw string, b_kw string, src_ids []*model.Id, start int64, end int64, limit int64) ([]*model.Article, error) {
	self.mtx.RLock()
	defer self.mtx.RUnlock()

	return self.db.LookupArticles(t_kw, b_kw, src_ids, start, end, limit)
}

func (self *TimeVortex) GetFeed(src_id *model.Id, limit int64) ([]*model.Article, error) {
	self.mtx.RLock()
	defer self.mtx.RUnlock()

	return self.db.GetFeed(src_id, limit)
}

func (self *TimeVortex) BindFeed(src_id *model.Id, article_id *model.Id) error {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	return self.db.BindFeed(src_id, article_id)
}

func (self *TimeVortex) RemoveFeedEntry(src_id *model.Id, article_id *model.Id) error {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	return self.db.RemoveFeedEntry(src_id, article_id)
}

func (self *TimeVortex) AddAction(name string, cmd string) (*filter.Action, error) {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	return self.db.AddAction(name, cmd)
}

func (self *TimeVortex) GetAction(id *model.Id) (*filter.Action, error) {
	self.mtx.RLock()
	defer self.mtx.RUnlock()

	return self.db.GetAction(id)
}

func (self *TimeVortex) GetActions() ([]*filter.Action, error) {
	self.mtx.RLock()
	defer self.mtx.RUnlock()

	return self.db.GetActions()
}

func (self *TimeVortex) DeleteAction(id *model.Id) error {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	return self.db.DeleteAction(id)
}

func (self *TimeVortex) AddFilter(title string, regex_title bool, body string, regex_body bool, action_id *model.Id) (*filter.Filter, error) {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	return self.db.AddFilter(title, regex_title, body, regex_body, action_id)
}

func (self *TimeVortex) UpdateFilterAction(id *model.Id, action_id *model.Id) (*filter.Filter, error) {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	return self.db.UpdateFilterAction(id, action_id)
}

func (self *TimeVortex) GetFilter(id *model.Id) (*filter.Filter, error) {
	self.mtx.RLock()
	defer self.mtx.RUnlock()

	return self.db.GetFilter(id)
}

func (self *TimeVortex) GetFilters() ([]*filter.Filter, error) {
	self.mtx.RLock()
	defer self.mtx.RUnlock()

	return self.db.GetFilters()
}

func (self *TimeVortex) DeleteFilter(id *model.Id) error {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	return self.db.DeleteFilter(id)
}

func (self *TimeVortex) BindFilter(src_id *model.Id, f_id *model.Id) error {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	return self.db.BindFilter(src_id, f_id)
}

func (self *TimeVortex) UnBindFilter(src_id *model.Id, f_id *model.Id) error {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	return self.db.UnBindFilter(src_id, f_id)
}

func (self *TimeVortex) GetFilterOnSource(src_id *model.Id) ([]*filter.Filter, error) {
	self.mtx.RLock()
	defer self.mtx.RUnlock()

	return self.db.GetFilterOnSource(src_id)
}

func (self *TimeVortex) GetSourceWithEnabledFilter(f_id *model.Id) ([]*model.Source, error) {
	self.mtx.RLock()
	defer self.mtx.RUnlock()

	return self.db.GetSourceWithEnabledFilter(f_id)
}
