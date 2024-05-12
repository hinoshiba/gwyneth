package gwyneth

import (
	"fmt"
	"log/slog"
	"time"
	"regexp"
)

import (
	"github.com/l4go/task"
)

import (
	"github.com/hinoshiba/gwyneth/config"
	"github.com/hinoshiba/gwyneth/tv"
	"github.com/hinoshiba/gwyneth/structs"
	"github.com/hinoshiba/gwyneth/filter"

	"github.com/hinoshiba/gwyneth/tv/errors"

	"github.com/hinoshiba/gwyneth/collector/rss"
)

const (
	COLLECTOR_RSS_POOL_SIZE = 10
)

type Gwyneth struct {
	tv  *tv.TimeVortex
	msn *task.Mission

	new_src       *noticer
	update_filter *noticer

	artcl_ch chan *structs.Article

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

		artcl_ch: make(chan *structs.Article),

		new_src:       newNoticer(msn.NewCancel()),
		update_filter: newNoticer(msn.NewCancel()),
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
	go self.run_core(self.msn.New())

	self.new_src.Notice()
	self.update_filter.Notice()
	return nil
}

func (self *Gwyneth) run_core(msn *task.Mission) {
	defer msn.Done()

	var msn_clctr *task.Mission
	var msn_rcdr  *task.Mission
	for {
		select {
		case <- msn.RecvCancel():
			return
		case <- self.new_src.Recv():
			if msn_clctr != nil {
				msn_clctr.Cancel()
			}

			msn_clctr = msn.New()
			go func(msn_clctr *task.Mission){
				defer msn_clctr.Done()

				if err := self.run_rss_collector(msn_clctr.New()); err != nil {
					slog.Error(fmt.Sprintf("failed: wakeup rss collector: %s", err))
				}
			}(msn_clctr)
		case <- self.update_filter.Recv():
			if msn_rcdr != nil {
				msn_rcdr.Cancel()
			}

			msn_rcdr = msn.New()
			go func(msn_rcdr *task.Mission){
				defer msn_rcdr.Done()

				if err := self.run_article_recoder(msn_rcdr.New()); err !=nil {
					slog.Error(fmt.Sprintf("failed: wakeup article recorder: %s", err))
				}
			}(msn_rcdr)
		}
	}
}

func (self *Gwyneth) run_article_recoder(msn *task.Mission) error {
	defer msn.Done()
	fmt.Println("run article recorder")

	f_buf := make(map[string][]*filter.Filter)

	for {
		select {
		case <- msn.RecvCancel():
			return nil
		case artcl := <- self.artcl_ch:
			a, err := self.addArticle(artcl.Title(), artcl.Body(), artcl.Link(), artcl.Unixtime(), artcl.Raw(), artcl.Src().Id())
			if err != nil {
				/* //debug all do filter
				if err != errors.ERR_ALREADY_EXIST_ARTICLE {
					slog.Warn(fmt.Sprintf("failed: addArticle: %s", err))
					continue
				}
				*/
				if err == errors.ERR_ALREADY_EXIST_ARTICLE {
					continue
				}
				slog.Warn(fmt.Sprintf("failed: addArticle: %s", err))
				continue
			}

			fs, ok := f_buf[a.Src().Id().String()]
			if !ok {
				new_fs, err := self.getFilterOnSource(a.Src().Id())
				if err != nil {
					slog.Warn(fmt.Sprintf("failed: cannot get filters for '%s': %s", a.Src().Id().String(), err))
					continue
				}

				fs = new_fs
				f_buf[a.Src().Id().String()] = new_fs
			}

			func (msn *task.Mission) {
				defer msn.Done()

				for _, f := range fs {
					go func(msn *task.Mission, f filter.Filter) {
						defer msn.Done()
						if f.IsMatch(a) {
							action := f.Action()

							if err := action.Do(msn.New(), a); err != nil {
								slog.Error(fmt.Sprintf("failed: execute filter: %s", err))
							}
						}
					}(msn.New(), *f)
				}
			}(msn.New())
		}
	}
}

func (self *Gwyneth) run_rss_collector(msn *task.Mission) error {
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
					p.Do(rss_collector, msn.New(), tgt, self.artcl_ch)
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
	s, err := self.tv.AddSource(title, src_type_id, source)
	if err != nil {
		return nil, err
	}

	self.new_src.Notice()
	return s, nil
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
	if err := self.tv.DeleteSource(id); err != nil {
		return err
	}

	self.new_src.Notice()
	return nil
}

func (self *Gwyneth) AddArticle(title string, body string, link string, utime int64, raw string, src_id *structs.Id) (*structs.Article, error){
	a, err := self.addArticle(title, body, link, utime, raw, src_id)
	if err != nil {
		if err != errors.ERR_ALREADY_EXIST_ARTICLE {
			return nil, err
		}
	}
	return a, nil
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

func (self *Gwyneth) BindFeed(src_id *structs.Id, artcl_id *structs.Id) error {
	return self.bindFeed(src_id, artcl_id)
}

func (self *Gwyneth) bindFeed(src_id *structs.Id, artcl_id *structs.Id) error {
	return self.tv.BindFeed(src_id, artcl_id)
}

func (self *Gwyneth) RemoveFeedEntry(src_id *structs.Id, article_id *structs.Id) error {
	return self.removeFeedEntry(src_id, article_id)
}

func (self *Gwyneth) removeFeedEntry(src_id *structs.Id, article_id *structs.Id) error {
	return self.tv.RemoveFeedEntry(src_id, article_id)
}

func (self *Gwyneth) AddAction(name string, cmd string) (*filter.Action, error) {
	return self.addAction(name, cmd)
}

func (self *Gwyneth) addAction(name string, cmd string) (*filter.Action, error) {
	action, err := self.tv.AddAction(name, cmd)
	if err != nil {
		return nil, err
	}

	self.update_filter.Notice()
	return action, nil
}

func (self *Gwyneth) GetActions() ([]*filter.Action, error) {
	return self.getActions()
}

func (self *Gwyneth) getActions() ([]*filter.Action, error) {
	return self.tv.GetActions()
}

func (self *Gwyneth) GetAction(id *structs.Id) (*filter.Action, error) {
	return self.getAction(id)
}

func (self *Gwyneth) getAction(id *structs.Id) (*filter.Action, error) {
	return self.tv.GetAction(id)
}

func (self *Gwyneth) DeleteAction(id *structs.Id) error {
	return self.deleteAction(id)
}

func (self *Gwyneth) deleteAction(id *structs.Id) error {
	if err := self.tv.DeleteAction(id); err != nil {
		return err
	}

	self.update_filter.Notice()
	return nil
}

func (self *Gwyneth) AddFilter(title string, regex_title bool, body string, regex_body bool, action_id *structs.Id) (*filter.Filter, error) {
	return self.addFilter(title, regex_title, body, regex_body, action_id)
}

func (self *Gwyneth) addFilter(title string, regex_title bool, body string, regex_body bool, action_id *structs.Id) (*filter.Filter, error) {
	if regex_title {
		_, err := regexp.Compile(title)
		if err != nil {
			return nil, fmt.Errorf("cannot compile regex at title :'%s'", err)
		}
	}
	if regex_body {
		_, err := regexp.Compile(body)
		if err != nil {
			return nil, fmt.Errorf("cannot compile regex at body :'%s'", err)
		}
	}

	f, err := self.tv.AddFilter(title, regex_title, body, regex_body, action_id)
	if err != nil {
		return nil, err
	}

	self.update_filter.Notice()
	return f, nil
}

func (self *Gwyneth) UpdateFilterAction(id *structs.Id, action_id *structs.Id) (*filter.Filter, error) {
	return self.updateFilterAction(id, action_id)
}

func (self *Gwyneth) updateFilterAction(id *structs.Id, action_id *structs.Id) (*filter.Filter, error) {
	f, err := self.tv.UpdateFilterAction(id, action_id)
	if err != nil {
		return nil, err
	}

	self.update_filter.Notice()
	return f, nil
}

func (self *Gwyneth) GetFilters() ([]*filter.Filter, error) {
	return self.getFilters()
}

func (self *Gwyneth) getFilters() ([]*filter.Filter, error) {
	return self.tv.GetFilters()
}

func (self *Gwyneth) GetFilter(id *structs.Id) (*filter.Filter, error) {
	return self.getFilter(id)
}

func (self *Gwyneth) getFilter(id *structs.Id) (*filter.Filter, error) {
	return self.tv.GetFilter(id)
}

func (self *Gwyneth) DeleteFilter(id *structs.Id) error {
	return self.deleteFilter(id)
}

func (self *Gwyneth) deleteFilter(id *structs.Id) error {
	if err := self.tv.DeleteFilter(id); err != nil {
		return err
	}

	self.update_filter.Notice()
	return nil
}

func (self *Gwyneth) BindFilter(src_id *structs.Id, f_id *structs.Id) error {
	return self.bindFilter(src_id, f_id)
}

func (self *Gwyneth) bindFilter(src_id *structs.Id, f_id *structs.Id) error {
	if err := self.tv.BindFilter(src_id, f_id); err != nil {
		return err
	}

	self.update_filter.Notice()
	return nil
}

func (self *Gwyneth) UnBindFilter(src_id *structs.Id, f_id *structs.Id) error {
	return self.unBindFilter(src_id, f_id)
}

func (self *Gwyneth) unBindFilter(src_id *structs.Id, f_id *structs.Id) error {
	if err := self.tv.UnBindFilter(src_id, f_id); err != nil {
		return err
	}

	self.update_filter.Notice()
	return nil
}

func (self *Gwyneth) GetFilterOnSource(src_id *structs.Id) ([]*filter.Filter, error) {
	return self.getFilterOnSource(src_id)
}

func (self *Gwyneth) getFilterOnSource(src_id *structs.Id) ([]*filter.Filter, error) {
	return self.tv.GetFilterOnSource(src_id)
}

func (self *Gwyneth) GetSourceWithEnabledFilter(f_id *structs.Id) ([]*structs.Source, error) {
	return self.getSourceWithEnabledFilter(f_id)
}

func (self *Gwyneth) getSourceWithEnabledFilter(f_id *structs.Id) ([]*structs.Source, error) {
	return self.tv.GetSourceWithEnabledFilter(f_id)
}

func rss_collector(msn *task.Mission, args ...any) {
	defer msn.Done()

	src := args[0].(*structs.Source)
	artcl_ch := args[1].(chan *structs.Article)

	if task.IsCanceled(msn) {
		slog.Info(fmt.Sprintf("the collector of '%s' is canceld", src.Title()))
		return
	}

	slog.Debug(fmt.Sprintf("the collector of '%s' is running... :'%s'", src.Title(), src.Value()))
	if err := rss.GetFeed(msn.New(), src, artcl_ch); err != nil {
		slog.Warn(fmt.Sprintf("cannot collect '%s/%s': %s", src.Title(), src.Value(), err))
	}
	slog.Debug(fmt.Sprintf("the collector of '%s' done!!!", src.Title()))
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

type noticer struct {
	msg_ch chan struct{}
	cc     task.Canceller
}

func newNoticer(cc task.Canceller) *noticer {
	return &noticer{
		msg_ch: make(chan struct{}),
		cc: cc,
	}
}

func (self *noticer) Recv() <- chan struct{} {
	return self.msg_ch
}

func (self *noticer) Notice() {
	go func () {
		select {
		case <-self.cc.RecvCancel():
		case self.msg_ch <- struct{}{}:
		}
	}()
}
