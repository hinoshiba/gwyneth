package gwyneth

import (
	"fmt"
	"log/slog"
	"time"
)

import (
	"github.com/l4go/task"
)

import (
	"github.com/hinoshiba/gwyneth/config"
	"github.com/hinoshiba/gwyneth/tv"
	"github.com/hinoshiba/gwyneth/structs"
	//"github.com/hinoshiba/gwyneth/consts"

	"github.com/hinoshiba/gwyneth/collector/rss"
)

const (
	COLLECTOR_RSS_POOL_SIZE = 10
)

type Gwyneth struct {
	tv  *tv.TimeVortex
	msn *task.Mission

	new_src chan struct{}

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

		new_src: make(chan struct{}),
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
	go self.run_collector(self.msn.New())
	self.reload_collector()
	return nil
}

func (self *Gwyneth) run_collector(msn *task.Mission) {
	defer msn.Done()

	artcl_ch := make(chan *structs.Article)
	defer close(artcl_ch)

	var p_msn *task.Mission
	for {
		select {
		case <- self.new_src:
			if p_msn != nil {
				p_msn.Cancel()
			}

			p_msn = msn.New()
			go func(p_msn *task.Mission){
				defer p_msn.Done()

				self.run_rss_collector(p_msn.New(), artcl_ch)
			}(p_msn)
		case <- msn.RecvCancel():
			return
		case artcl := <- artcl_ch:
			if _, err := self.addArticle(artcl.Title(), artcl.Body(), artcl.Link(), artcl.Unixtime(), artcl.Raw(), artcl.Src().Id()); err != nil {
				slog.Warn(fmt.Sprintf("failed: addArticle: %s", err))
				continue
			}
		}
	}
}

func (self *Gwyneth) reload_collector() {
	go func(cc task.Canceller) {
		select {
		case <-cc.RecvCancel():
		case self.new_src <- struct{}{}:
		}
	}(self.msn.NewCancel())
}

func (self *Gwyneth) run_rss_collector(msn *task.Mission, artcl_ch chan <- *structs.Article) error {
	defer msn.Done()

	p := task.NewPool(msn.New(), COLLECTOR_RSS_POOL_SIZE)
	defer p.Close()

	src_s, err := self.tv.GetSources()
	if err != nil {
		return err
	}

	tgts := []*structs.Source{}
	for _, src := range src_s {
		if src.Type().IsUserCreate() {
			continue
		}
		if !(src.Type().Name() == "rss") {
			continue
		}

		tgts = append(tgts, src)
	}
	if len(tgts) < 1 {
		slog.Info(fmt.Sprintf("rss collector is zero"))
		return nil
	}

	loop_sec := 60 * 5 //WIP: to config

	tgts_s := split_src(loop_sec, tgts)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	now := 0
	for {
		select {
		case <- msn.RecvCancel():
			return nil
		case <- ticker.C:
			func () {
				defer func(){
					now++
					if now >= loop_sec {
						now = 0
					}
				}()

				if len(tgts_s) <= now {
					return
				}

				tgts := tgts_s[now]
				if tgts == nil {
					return
				}

				for _, tgt := range tgts {
					p.Do(rss_collector, msn.New(), tgt, artcl_ch)
				}
			}()
		}
	}
}

func (self *Gwyneth) checkAndInitSourceTypes() error {
	defaults := map[string]string{
		"rss": "rss",
		"noop": "noop",
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
	self.reload_collector()
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

func (self *Gwyneth) AddAction(name string, cmd string) (*structs.Action, error) {
	return self.addAction(name, cmd)
}

func (self *Gwyneth) addAction(name string, cmd string) (*structs.Action, error) {
	return self.tv.AddAction(name, cmd)
}

func (self *Gwyneth) GetActions() ([]*structs.Action, error) {
	return self.getActions()
}

func (self *Gwyneth) getActions() ([]*structs.Action, error) {
	return self.tv.GetActions()
}

func (self *Gwyneth) GetAction(id *structs.Id) (*structs.Action, error) {
	return self.getAction(id)
}

func (self *Gwyneth) getAction(id *structs.Id) (*structs.Action, error) {
	return self.tv.GetAction(id)
}

func (self *Gwyneth) DeleteAction(id *structs.Id) error {
	return self.deleteAction(id)
}

func (self *Gwyneth) deleteAction(id *structs.Id) error {
	return self.tv.DeleteAction(id)
}

func (self *Gwyneth) AddFilter(title string, regex_title bool, body string, regex_body bool, action_id *structs.Id) (*structs.Filter, error) {
	return self.addFilter(title, regex_title, body, regex_body, action_id)
}

func (self *Gwyneth) addFilter(title string, regex_title bool, body string, regex_body bool, action_id *structs.Id) (*structs.Filter, error) {
	return self.tv.AddFilter(title, regex_title, body, regex_body, action_id)
}

func (self *Gwyneth) UpdateFilterAction(id *structs.Id, action_id *structs.Id) (*structs.Filter, error) {
	return self.updateFilterAction(id, action_id)
}

func (self *Gwyneth) updateFilterAction(id *structs.Id, action_id *structs.Id) (*structs.Filter, error) {
	return self.tv.UpdateFilterAction(id, action_id)
}

func (self *Gwyneth) GetFilters() ([]*structs.Filter, error) {
	return self.getFilters()
}

func (self *Gwyneth) getFilters() ([]*structs.Filter, error) {
	return self.tv.GetFilters()
}

func (self *Gwyneth) GetFilter(id *structs.Id) (*structs.Filter, error) {
	return self.getFilter(id)
}

func (self *Gwyneth) getFilter(id *structs.Id) (*structs.Filter, error) {
	return self.tv.GetFilter(id)
}

func (self *Gwyneth) DeleteFilter(id *structs.Id) error {
	return self.deleteFilter(id)
}

func (self *Gwyneth) deleteFilter(id *structs.Id) error {
	return self.tv.DeleteFilter(id)
}

func rss_collector(msn *task.Mission, args ...any) {
	defer msn.Done()

	src := args[0].(*structs.Source)
	artcl_ch := args[1].(chan <- *structs.Article)

	if task.IsCanceled(msn) {
		slog.Info(fmt.Sprintf("the collector of '%s' is canceld", src.Title()))
		return
	}

	slog.Debug(fmt.Sprintf("the collector of '%s' is running... :'%s'", src.Title(), src.Value()))
	if err := rss.GetFeed(msn.New(), src, artcl_ch); err != nil {
		slog.Warn(fmt.Sprintf("cannot collect '%s/%s': %s", src.Title(), src.Value(), err))
	}
	slog.Debug(fmt.Sprintf("the collector of '%s' done!!!"))
}

func split_src(size int, src_s []*structs.Source) [][]*structs.Source {
	if len(src_s) < 1 {
		return make([][]*structs.Source, 0, 0)
	}

	bkt_size := len(src_s) / size
	if bkt_size < 1 {
		bkt_size = 1
	}

	ret := make([][]*structs.Source, 0, size)
	for i := 0; i < len(src_s); i += bkt_size {
		end := i + bkt_size
		if end > len(src_s) {
			end = len(src_s)
		}
		ret = append(ret, src_s[i:end])
	}
	return ret
}
