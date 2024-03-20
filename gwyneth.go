package gwyneth

import (
	"fmt"
)

import (
	"github.com/l4go/task"
)

import (
	"github.com/hinoshiba/gwyneth/config"
	"github.com/hinoshiba/gwyneth/tv"
	"github.com/hinoshiba/gwyneth/structs"
)

type Gwyneth struct {
	tv  *tv.TimeVortex
	msn *task.Mission

	default_source_type map[string]struct{}
}

func New(msn *task.Mission, cfg *config.Config) (*Gwyneth, error) {
	t, err := tv.New(msn.New(), cfg)
	if err != nil {
		return nil, err
	}
	self := &Gwyneth {
		tv: t,
		msn: msn,
	}

	if err := self.init(); err != nil {
		return nil, err
	}
	return self, nil
}

func (self *Gwyneth) Close() error {
	defer self.msn.Done()

	self.msn.Cancel()

	return self.tv.Close()
}

func (self *Gwyneth) init() error {
	if err := self.checkAndInitSourceTypes(); err != nil {
		return err
	}
	return nil
}

func (self *Gwyneth) checkAndInitSourceTypes() error {
	defaults := map[string]string{
		"rss": "rss",
	}
	self.default_source_type = make(map[string]struct{})

	sts, err := self.tv.GetSourceTypes()
	if err != nil {
		return err
	}

	for _, st := range sts {
		if st.IsUserCreate() {
			continue
		}
		self.default_source_type[st.Id().String()] = struct{}{}
		delete(defaults, st.Name())
	}
	for name, cmd := range defaults {
		st, err := self.tv.AddSourceType(name, cmd, false)
		if err != nil {
			return err
		}
		self.default_source_type[st.Id().String()] = struct{}{}
	}
	return nil
}

func (self *Gwyneth) AddSourceType(name string, cmd string, is_user_creation bool) (*structs.SourceType, error) {
	return self.tv.AddSourceType(name, cmd, is_user_creation)
}

func (self *Gwyneth) GetSourceType(id *structs.Id) (*structs.SourceType, error) {
	return self.tv.GetSourceType(id)
}

func (self *Gwyneth) GetSourceTypes() ([]*structs.SourceType, error) {
	return self.tv.GetSourceTypes()
}

func (self *Gwyneth) DeleteSourceType(id *structs.Id) error {
	if _, ok := self.default_source_type[id.String()]; ok {
		return fmt.Errorf("cannot delete a default's source type.")
	}
	return self.tv.DeleteSourceType(id)
}

func (self *Gwyneth) AddSource(title string, src_type_id *structs.Id, source string) (*structs.Source, error) {
	return self.tv.AddSource(title, src_type_id, source)
}

func (self *Gwyneth) GetSource(id *structs.Id) (*structs.Source, error) {
	return self.tv.GetSource(id)
}

func (self *Gwyneth) GetSources() ([]*structs.Source, error) {
	return self.tv.GetSources()
}

func (self *Gwyneth) FindSource(kw string) ([]*structs.Source, error) {
	return self.tv.FindSource(kw)
}

func (self *Gwyneth) DeleteSource(id *structs.Id) error {
	return self.tv.DeleteSource(id)
}

func (self *Gwyneth) AddArticle(title string, body string, link string, utime int64, raw string, src_id *structs.Id) (*structs.Article, error){
	return self.addArticle(title, body, link, utime, raw, src_id)
}

func (self *Gwyneth) addArticle(title string, body string, link string, utime int64, raw string, src_id *structs.Id) (*structs.Article, error){
	return self.tv.AddArticle(title, body, link, utime, raw, src_id)
}

func (self *Gwyneth) RemoveArticle(id *structs.Id) error {
	return self.removeArticle(id)
}

func (self *Gwyneth) removeArticle(id *structs.Id) error {
	return self.tv.RemoveArticle(id)
}

func (self *Gwyneth) LookupArticles(t_kw string, b_kw string, src_ids []*structs.Id, start int64, end int64, limit int64) ([]*structs.Article, error) {
	return self.lookupArticles(t_kw, b_kw, src_ids, start, end, limit)
}

func (self *Gwyneth) lookupArticles(t_kw string, b_kw string, src_ids []*structs.Id, start int64, end int64, limit int64) ([]*structs.Article, error) {
	return self.tv.LookupArticles(t_kw, b_kw, src_ids, start, end, limit)
}

func (self *Gwyneth) GetFeed(src_id *structs.Id, limit int64) ([]*structs.Article, error) {
	return self.getFeed(src_id, limit)
}

func (self *Gwyneth) getFeed(src_id *structs.Id, limit int64) ([]*structs.Article, error) {
	return self.tv.GetFeed(src_id, limit)
}

func (self *Gwyneth) RemoveFeedEntry(src_id *structs.Id, article_id *structs.Id) error {
	return self.removeFeedEntry(src_id, article_id)
}

func (self *Gwyneth) removeFeedEntry(src_id *structs.Id, article_id *structs.Id) error {
	return self.tv.RemoveFeedEntry(src_id, article_id)
}
