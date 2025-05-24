package gwyneth

import (
	"fmt"
	"time"
	"regexp"
)

import (
	"github.com/l4go/task"
)

import (
	"github.com/hinoshiba/gwyneth/slog"
	"github.com/hinoshiba/gwyneth/config"
	"github.com/hinoshiba/gwyneth/tv"
	"github.com/hinoshiba/gwyneth/model"
	"github.com/hinoshiba/gwyneth/filter"

	"github.com/hinoshiba/gwyneth/tv/errors"

	"github.com/hinoshiba/gwyneth/collector/rss"
)

const (
	COLLECTOR_RSS_POOL_SIZE = 10
	FILTER_ACTION_POOL_SIZE = 10
)

type Gwyneth struct {
	tv  *tv.TimeVortex
	msn *task.Mission

	lm  *slog.LogManager

	new_src       *noticer
	update_filter *noticer

	artcl_ch     chan *model.Article
	do_filter_ch chan *model.Article

	default_source_type map[string]struct{}
}

func New(msn *task.Mission, lm *slog.LogManager, cfg *config.Config) (*Gwyneth, error) {
	t, err := tv.New(msn.New(), cfg)
	if err != nil {
		return nil, err
	}
	self := &Gwyneth {
		tv: t,
		msn: msn,

		lm: lm,

		artcl_ch: make(chan *model.Article),
		do_filter_ch: make(chan *model.Article),

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
	go self.run_article_recoder(self.msn.New())

	self.new_src.Notice()
	self.update_filter.Notice()
	return nil
}

func (self *Gwyneth) run_core(msn *task.Mission) {
	defer msn.Done()

	var msn_clctr *task.Mission
	var msn_filtr *task.Mission
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
					slog.Error("failed: wakeup rss collector: %s", err)
				}
			}(msn_clctr)
		case <- self.update_filter.Recv():
			if msn_filtr != nil {
				msn_filtr.Cancel()
			}

			msn_filtr = msn.New()
			go func(msn_filtr *task.Mission){
				defer msn_filtr.Done()

				if err := self.run_filter_engine(msn_filtr.New()); err !=nil {
					slog.Error("failed: wakeup article recorder: %s", err)
				}
			}(msn_filtr)
		}
	}
}

func (self *Gwyneth) run_article_recoder(msn *task.Mission) error {
	defer msn.Done()

	slog.Debug("start article_recoder")

	for {
		select {
		case <- msn.RecvCancel():
			return nil
		case artcl := <- self.artcl_ch:
			added_artcl, err := self.addArticle(artcl.Title(), artcl.Body(), artcl.Link(), artcl.Unixtime(), artcl.Raw(), artcl.Src().Id())
			if err != nil {
				if err == errors.ERR_ALREADY_EXIST_ARTICLE {
					continue
				}
				slog.Warn("failed: addArticle: %s", err)
				continue
			}

			select {
			case <- msn.RecvCancel():
				return nil
			case self.do_filter_ch <- added_artcl:
			}
		}
	}
}

func (self *Gwyneth) run_filter_engine(msn *task.Mission) error {
	defer msn.Done()

	slog.Debug("start filter engine")

	f_buf := make(map[string][]*filter.Filter)

	p := task.NewPool(msn.New(), FILTER_ACTION_POOL_SIZE)
	defer p.Close()

	for {
		select {
		case <- msn.RecvCancel():
			return nil
		case artcl := <- self.do_filter_ch:
			fs, ok := f_buf[artcl.Src().Id().String()]
			if !ok {
				new_fs, err := self.getFilterOnSource(artcl.Src().Id())
				if err != nil {
					slog.Warn("failed: cannot get filters for '%s': %s", artcl.Src().Id().String(), err)
					continue
				}

				fs = new_fs
				f_buf[artcl.Src().Id().String()] = new_fs
			}

			go func (msn *task.Mission, artcl *model.Article) {
				defer msn.Done()

				for _, f := range fs {
					go func(msn *task.Mission, f filter.Filter) {
						defer msn.Done()
						if f.IsMatch(artcl) {
							p.Do(action_filter, msn.New(), self.lm.GetActionsLogger(), f.Action(), artcl)
						}
					}(msn.New(), *f)
				}
			}(msn.New(), artcl)
		}
	}
}

func (self *Gwyneth) run_rss_collector(msn *task.Mission) error {
	defer msn.Done()

	slog.Debug("start collector")

	p := task.NewPool(msn.New(), COLLECTOR_RSS_POOL_SIZE)
	defer p.Close()

	src_s, err := self.tv.GetSources()
	if err != nil {
		return err
	}

	tgts := []*model.Source{}
	for _, src := range src_s {
		if src.Type().IsUserCreate() {
			continue
		}
		if !(src.Type().Name() == "rss") {
			continue
		}
		if src.IsPause() {
			continue
		}

		tgts = append(tgts, src)
	}
	if len(tgts) < 1 {
		slog.Info("rss collector is zero")
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
					p.Do(rss_collector, msn.New(), self.lm.GetCollectorsLogger(), tgt, self.artcl_ch)
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

func (self *Gwyneth) AddSourceType(name string, cmd string, is_user_creation bool) (*model.SourceType, error) {
	return self.tv.AddSourceType(name, cmd, is_user_creation)
}

func (self *Gwyneth) GetSourceType(id *model.Id) (*model.SourceType, error) {
	return self.tv.GetSourceType(id)
}

func (self *Gwyneth) GetSourceTypes() ([]*model.SourceType, error) {
	return self.tv.GetSourceTypes()
}

func (self *Gwyneth) DeleteSourceType(id *model.Id) error {
	if _, ok := self.default_source_type[id.String()]; ok {
		return fmt.Errorf("cannot delete a default's source type.")
	}
	return self.tv.DeleteSourceType(id)
}

func (self *Gwyneth) AddSource(title string, src_type_id *model.Id, source string) (*model.Source, error) {
	s, err := self.tv.AddSource(title, src_type_id, source)
	if err != nil {
		return nil, err
	}

	self.new_src.Notice()
	return s, nil
}

func (self *Gwyneth) GetSource(id *model.Id) (*model.Source, error) {
	return self.tv.GetSource(id)
}

func (self *Gwyneth) GetSources() ([]*model.Source, error) {
	return self.tv.GetSources()
}

func (self *Gwyneth) FindSource(kw string) ([]*model.Source, error) {
	return self.tv.FindSource(kw)
}

func (self *Gwyneth) RemoveSource(id *model.Id) error {
	if err := self.tv.RemoveSource(id); err != nil {
		return err
	}

	self.new_src.Notice()
	return nil
}

func (self *Gwyneth) PauseSource(id *model.Id) error {
	if err := self.tv.PauseSource(id); err != nil {
		return err
	}

	self.new_src.Notice()
	return nil
}

func (self *Gwyneth) ResumeSource(id *model.Id) error {
	if err := self.tv.ResumeSource(id); err != nil {
		return err
	}

	self.new_src.Notice()
	return nil
}

func (self *Gwyneth) AddArticle(title string, body string, link string, utime int64, raw string, src_id *model.Id) (*model.Article, error){
	a, err := self.addArticle(title, body, link, utime, raw, src_id)
	if err != nil {
		if err != errors.ERR_ALREADY_EXIST_ARTICLE {
			return nil, err
		}
	}
	select {
	case <- self.msn.RecvCancel():
	case self.do_filter_ch <- a:
	}
	return a, nil
}

func (self *Gwyneth) addArticle(title string, body string, link string, utime int64, raw string, src_id *model.Id) (*model.Article, error){
	return self.tv.AddArticle(title, body, link, utime, raw, src_id)
}

func (self *Gwyneth) RemoveArticle(id *model.Id) error {
	return self.removeArticle(id)
}

func (self *Gwyneth) removeArticle(id *model.Id) error {
	return self.tv.RemoveArticle(id)
}

func (self *Gwyneth) LookupArticles(t_kw string, b_kw string, src_ids []*model.Id, start int64, end int64, limit int64) ([]*model.Article, error) {
	return self.lookupArticles(t_kw, b_kw, src_ids, start, end, limit)
}

func (self *Gwyneth) lookupArticles(t_kw string, b_kw string, src_ids []*model.Id, start int64, end int64, limit int64) ([]*model.Article, error) {
	return self.tv.LookupArticles(t_kw, b_kw, src_ids, start, end, limit)
}

func (self *Gwyneth) GetFeed(src_id *model.Id, limit int64) ([]*model.Article, error) {
	return self.getFeed(src_id, limit)
}

func (self *Gwyneth) getFeed(src_id *model.Id, limit int64) ([]*model.Article, error) {
	return self.tv.GetFeed(src_id, limit)
}

func (self *Gwyneth) BindFeed(src_id *model.Id, artcl_id *model.Id) error {
	return self.bindFeed(src_id, artcl_id)
}

func (self *Gwyneth) bindFeed(src_id *model.Id, artcl_id *model.Id) error {
	return self.tv.BindFeed(src_id, artcl_id)
}

func (self *Gwyneth) RemoveFeedEntry(src_id *model.Id, article_id *model.Id) error {
	return self.removeFeedEntry(src_id, article_id)
}

func (self *Gwyneth) removeFeedEntry(src_id *model.Id, article_id *model.Id) error {
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

func (self *Gwyneth) GetAction(id *model.Id) (*filter.Action, error) {
	return self.getAction(id)
}

func (self *Gwyneth) getAction(id *model.Id) (*filter.Action, error) {
	return self.tv.GetAction(id)
}

func (self *Gwyneth) DeleteAction(id *model.Id) error {
	return self.deleteAction(id)
}

func (self *Gwyneth) deleteAction(id *model.Id) error {
	if err := self.tv.DeleteAction(id); err != nil {
		return err
	}

	self.update_filter.Notice()
	return nil
}

func (self *Gwyneth) AddFilter(title string, regex_title bool, body string, regex_body bool, action_id *model.Id) (*filter.Filter, error) {
	return self.addFilter(title, regex_title, body, regex_body, action_id)
}

func (self *Gwyneth) addFilter(title string, regex_title bool, body string, regex_body bool, action_id *model.Id) (*filter.Filter, error) {
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

func (self *Gwyneth) UpdateFilterAction(id *model.Id, action_id *model.Id) (*filter.Filter, error) {
	return self.updateFilterAction(id, action_id)
}

func (self *Gwyneth) updateFilterAction(id *model.Id, action_id *model.Id) (*filter.Filter, error) {
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

func (self *Gwyneth) GetFilter(id *model.Id) (*filter.Filter, error) {
	return self.getFilter(id)
}

func (self *Gwyneth) getFilter(id *model.Id) (*filter.Filter, error) {
	return self.tv.GetFilter(id)
}

func (self *Gwyneth) DeleteFilter(id *model.Id) error {
	return self.deleteFilter(id)
}

func (self *Gwyneth) deleteFilter(id *model.Id) error {
	if err := self.tv.DeleteFilter(id); err != nil {
		return err
	}

	self.update_filter.Notice()
	return nil
}

func (self *Gwyneth) BindFilter(src_id *model.Id, f_id *model.Id) error {
	return self.bindFilter(src_id, f_id)
}

func (self *Gwyneth) bindFilter(src_id *model.Id, f_id *model.Id) error {
	if err := self.tv.BindFilter(src_id, f_id); err != nil {
		return err
	}

	self.update_filter.Notice()
	return nil
}

func (self *Gwyneth) UnBindFilter(src_id *model.Id, f_id *model.Id) error {
	return self.unBindFilter(src_id, f_id)
}

func (self *Gwyneth) unBindFilter(src_id *model.Id, f_id *model.Id) error {
	if err := self.tv.UnBindFilter(src_id, f_id); err != nil {
		return err
	}

	self.update_filter.Notice()
	return nil
}

func (self *Gwyneth) GetFilterOnSource(src_id *model.Id) ([]*filter.Filter, error) {
	return self.getFilterOnSource(src_id)
}

func (self *Gwyneth) getFilterOnSource(src_id *model.Id) ([]*filter.Filter, error) {
	return self.tv.GetFilterOnSource(src_id)
}

func (self *Gwyneth) GetSourceWithEnabledFilter(f_id *model.Id) ([]*model.Source, error) {
	return self.getSourceWithEnabledFilter(f_id)
}

func (self *Gwyneth) getSourceWithEnabledFilter(f_id *model.Id) ([]*model.Source, error) {
	return self.tv.GetSourceWithEnabledFilter(f_id)
}

func (self *Gwyneth) ReFilter(src_id *model.Id, limit int64) error {
	articles, err := self.getFeed(src_id, limit)
	if err != nil {
		return err
	}

	func (msn *task.Mission) {
		defer msn.Done()

		for _, article := range articles {
			go func (msn *task.Mission, article *model.Article) {
				defer msn.Done()

				select {
				case <- msn.RecvCancel():
				case self.do_filter_ch <- article:
				}
			}(msn.New(), article)
		}
	}(self.msn.New())
	return nil
}

func action_filter(msn *task.Mission, args ...any) {
	defer msn.Done()

	logger := args[0].(*slog.Logger)
	action := args[1].(*filter.Action)
	artcl := args[2].(*model.Article)

	if task.IsCanceled(msn) {
		logger.Info("the filter's action is canceld")
		return
	}
	if err := action.Do(msn.New(), logger, artcl); err != nil {
		logger.Error("failed: execute filter: %s", err)
	}
}

func rss_collector(msn *task.Mission, args ...any) {
	defer msn.Done()

	logger := args[0].(*slog.Logger)
	src := args[1].(*model.Source)
	artcl_ch := args[2].(chan *model.Article)

	if task.IsCanceled(msn) {
		logger.Info("the collector of '%s' is canceld", src.Title())
		return
	}

	logger.Debug("the collector of '%s' is running... :'%s'", src.Title(), src.Value())
	if err := rss.GetFeed(msn.New(), logger, src, artcl_ch); err != nil {
		logger.Warn("cannot collect '%s/%s': %s", src.Title(), src.Value(), err)
	}
	logger.Debug("the collector of '%s' done!!!", src.Title())
}

func split_src(size int, src_s []*model.Source) [][]*model.Source {
	if len(src_s) < 1 {
		return make([][]*model.Source, 0, 0)
	}

	bkt_size := len(src_s) / size
	if bkt_size < 1 {
		bkt_size = 1
	}

	ret := make([][]*model.Source, 0, size)
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
