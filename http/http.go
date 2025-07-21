package http

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"html/template"
)

import (
	"github.com/l4go/task"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
)

import (
	"github.com/hinoshiba/gwyneth"
	"github.com/hinoshiba/gwyneth/slog"
	"github.com/hinoshiba/gwyneth/consts"
	"github.com/hinoshiba/gwyneth/config"
	"github.com/hinoshiba/gwyneth/model"
	"github.com/hinoshiba/gwyneth/model/external"
)

func init() {
	gin.SetMode(gin.ReleaseMode)
}

type Router struct {
	engine *gin.Engine

	cfg    *config.Config

	msn    *task.Mission
}

func New(msn *task.Mission, cfg *config.Config, g *gwyneth.Gwyneth) (*Router, error) {
	self := &Router{
		engine: gin.Default(),
		cfg: cfg,
		msn: msn,
	}

	self.mapRoute(g)
	if err := self.run(); err != nil {
		self.Close()
		return nil, err
	}
	return self, nil
}

func (self *Router) Close() error {
	defer self.msn.Done()

	self.msn.Cancel()
	return nil
}

func (self *Router) makeRenderHtml(fname string, param *gin.H) *render.HTML {
	app_root := self.cfg.Http.Root
	if !strings.HasSuffix(app_root, "/") {
		app_root += "/"
	}
	data := gin.H{
		"Version": consts.VERSION,
		"AppRoot": app_root,
	}
	if param != nil {
		for k, v := range *param {
			data[k] = v
		}
	}

	tmpl := loadTemplate(fname)
	return &render.HTML{
		Template: tmpl,
		Name: "contents",
		Data: data,
	}
}

func loadTemplate(fname string) *template.Template {
	files := []string{
		"/usr/local/src/http/templates/layout/main.html",
		filepath.Join("/usr/local/src/http/templates/pages/", fname),
	}

	tmpl := template.New("").Funcs(template.FuncMap{
		"eq": func(a, b string) bool { return a == b },

	})
	return template.Must(tmpl.ParseFiles(files...))
}

func (self *Router) mapRoute(g *gwyneth.Gwyneth) error {
	self.engine.Static("/static", "/usr/local/src/http/static")

	self.engine.GET("/", func(c *gin.Context) {
		app_root := self.cfg.Http.Root
		if !strings.HasSuffix(app_root, "/") {
			app_root += "/"
		}
		path := fmt.Sprintf("%ssource", app_root)
		c.Redirect(http.StatusMovedPermanently, path)
	})

	self.engine.GET("/search", func(c *gin.Context) {
		c.Render(http.StatusOK, self.makeRenderHtml(
			"search.html",
			&gin.H{"Page": "search"},
		))
	})

	self.engine.GET("/source_type", func(c *gin.Context) {
		c.Render(http.StatusOK, self.makeRenderHtml(
			"source_type.html",
			&gin.H{"Page": "source_type"},
		))
	})

	self.engine.GET("/source", func(c *gin.Context) {
		c.Render(http.StatusOK, self.makeRenderHtml(
			"source.html",
			&gin.H{"Page": "source"},
		))
	})

	self.engine.GET("/source/:id", func(c *gin.Context) {
		src_id := c.Param("id")

		c.Render(http.StatusOK, self.makeRenderHtml(
			"source_detail.html",
			&gin.H{
				"Page": "source_detail",
				"src_id": fmt.Sprintf("%s", src_id),
			},
		))
	})

	self.engine.GET("/action", func(c *gin.Context) {
		c.Render(http.StatusOK, self.makeRenderHtml(
			"action.html",
			&gin.H{"Page": "action"},
		))
	})

	self.engine.GET("/action/:id", func(c *gin.Context) {
		action_id := c.Param("id")

		c.Render(http.StatusOK, self.makeRenderHtml(
			"action_detail.html",
			&gin.H{
				"Page": "action_detail",
				"action_id": fmt.Sprintf("%s", action_id),
			},
		))
	})

	self.engine.GET("/filter", func(c *gin.Context) {
		c.Render(http.StatusOK, self.makeRenderHtml(
			"filter.html",
			&gin.H{"Page": "filter"},
		))
	})

	self.engine.GET("/filter/:id", func(c *gin.Context) {
		filter_id := c.Param("id")

		c.Render(http.StatusOK, self.makeRenderHtml(
			"filter_detail.html",
			&gin.H{
				"Page": "filter_detail",
				"filter_id": fmt.Sprintf("%s", filter_id),
			},
		))
	})

	api := self.engine.Group("/api")
	api.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	api.GET("/source_type", getHandlerGetSourceTypes(g))
	api.POST("/source_type", getHandlerAddSourceType(g))
	api.DELETE("source_type", getHandlerDeleteSourceType(g))

	api.GET("/source", getHandlerGetSources(g))
	api.POST("/source", getHandlerAddSource(g))
	api.DELETE("/source", getHandlerRemoveSource(g))

	api.GET("/source/:id", getHandlerGetSource(g))
	api.POST("/source/:id/filter", getHandlerBindFilter(g))
	api.GET("/source/:id/filter", getHandlerGetFilterOnSource(g))
	api.DELETE("/source/:id/filter", getHandlerUnBindFilter(g))
	api.POST("/source/:id/pause", getHandlerPauseSource(g))
	api.POST("/source/:id/resume", getHandlerResumeSource(g))

	api.GET("/article", getHandlerLookupArticles(self.cfg.Feed, g))
	api.POST("/article", getHandlerAddArticle(g))
	api.DELETE("/article", getHandlerRemoveArticle(g))

	api.GET("/feed/:id", getHandlerGetFeed(self.cfg.Feed, g))
	api.POST("/feed/:id", getHandlerPostFeed(self.cfg.Feed, g))
	api.DELETE("/feed/:id", getHandlerDeleteFeed(self.cfg.Feed, g))
	api.POST("/feed/:id/refilter", getHandlerReFilter(g))

	api.GET("/action", getHandlerGetActions(g))
	api.POST("/action", getHandlerAddAction(g))
	api.DELETE("/action", getHandlerDeleteAction(g))

	api.POST("/action/:id/restart", getHandlerRestartAction(g))
	api.POST("/action/:id/cancel", getHandlerCancelAction(g))
	api.GET("/action/:id/queue", getHandlerGetQeueMessages(g))
	api.GET("/action/:id/dlqueue", getHandlerGetDlqMessages(g))
	api.DELETE("/action/:id/queue/:msg_id", getHandlerDeleteActionQueueMessage(g))
	api.DELETE("/action/:id/dlqueue/:msg_id", getHandlerDeleteActionDlqMessage(g))
	api.POST("/action/:id/dlqueue/:msg_id/redrive", getHandlerRedriveActionMessage(g))

	api.GET("/filter", getHandlerGetFilters(g))
	api.POST("/filter", getHandlerAddFilter(g))
	api.PATCH("/filter", getHandlerUpdateFilter(g))
	api.DELETE("/filter", getHandlerDeleteFilter(g))

	return nil
}

func (self *Router) run() error {
	c_msn := self.msn.New()

	lc := net.ListenConfig{}
	ln, err := lc.Listen(c_msn.AsContext(), "tcp", self.cfg.Http.GetAddr())
	if err != nil {
		return err
	}

	go func(c_msn *task.Mission, ln net.Listener) {
		defer c_msn.Done()

		self.engine.RunListener(ln)
	}(c_msn, ln)
	return nil
}

func getHandlerAddSourceType(g *gwyneth.Gwyneth) func(*gin.Context) {
	return func(c *gin.Context) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "this api is not published yet."})
		return

		var st external.SourceType
		if err := c.ShouldBindJSON(&st); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.Debug("AddSourceType: request is '%v'", st)

		added_st, err := g.AddSourceType(st.Name, st.Cmd, true)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.IndentedJSON(http.StatusOK, added_st.ConvertExternal())
	}
}

func getHandlerGetSourceTypes(g *gwyneth.Gwyneth) func(*gin.Context) {
	return func(c *gin.Context) {
		id_base := c.Query("id")
		if id_base != "" {
			id, err := model.ParseStringId(id_base)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			st, err := g.GetSourceType(id)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			c.IndentedJSON(http.StatusOK, []*external.SourceType{st.ConvertExternal()})
			return
		}

		sts, err := g.GetSourceTypes()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ret_sts := []*external.SourceType{}
		for _, st := range sts {
			ret_sts = append(ret_sts, st.ConvertExternal())
		}
		c.IndentedJSON(http.StatusOK, ret_sts)
	}
}

func getHandlerDeleteSourceType(g *gwyneth.Gwyneth) func(*gin.Context) {
	return func(c *gin.Context) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "this api is not published yet."})
		return

		var st external.SourceType
		if err := c.ShouldBindJSON(&st); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.Debug("DeleteSourceType: request is '%v'", st)

		id, err := model.ParseStringId(st.Id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := g.DeleteSourceType(id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"id": st.Id,
		})
	}
}

func getHandlerAddSource(g *gwyneth.Gwyneth) func(*gin.Context) {
	return func(c *gin.Context) {
		var src external.Source
		if err := c.ShouldBindJSON(&src); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.Debug("AddSource: request is '%v'", src)

		src_type_id, err := model.ParseStringId(src.Type.Id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		added_src, err := g.AddSource(src.Title, src_type_id, src.Value)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.IndentedJSON(http.StatusOK, added_src.ConvertExternal())
	}
}

func getHandlerGetSources(g *gwyneth.Gwyneth) func(*gin.Context) {
	return func(c *gin.Context) {
		id_base := c.Query("id")
		if id_base != "" {
			id, err := model.ParseStringId(id_base)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			src, err := g.GetSource(id)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			c.IndentedJSON(http.StatusOK, []*external.Source{src.ConvertExternal()})
			return
		}

		srcs, err := g.GetSources()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ret_src := []*external.Source{}
		for _, src := range srcs {
			ext_src := src.ConvertExternal()

			sts := g.GetSourceStatus(src.Id())
			for _, st := range sts {
				ext_src.Status = append(ext_src.Status, st.ConvertExternal())
			}
			ret_src = append(ret_src, ext_src)
		}
		c.IndentedJSON(http.StatusOK, ret_src)
	}
}

func getHandlerRemoveSource(g *gwyneth.Gwyneth) func(*gin.Context) {
	return func(c *gin.Context) {
		var src external.Source
		if err := c.ShouldBindJSON(&src); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		id, err := model.ParseStringId(src.Id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := g.RemoveSource(id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"id": src.Id,
		})
	}
}

func getHandlerAddArticle(g *gwyneth.Gwyneth) func(*gin.Context) {
	return func(c *gin.Context) {
		var article external.Article
		if err := c.ShouldBindJSON(&article); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.Debug("AddArticle: request is '%v'", article)

		src_id, err := model.ParseStringId(article.Src.Id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("cannot parse src id('%s'): %s", article.Src.Id, err)})
			return
		}

		if _, err := g.GetSource(src_id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("source id is not exist: '%s'", article.Src.Id)})
			return
		}

		added_article, err := g.AddArticle(article.Title, article.Body, article.Link, int64(article.Timestamp), article.Raw, src_id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.IndentedJSON(http.StatusOK, added_article.ConvertExternal())
	}
}

func getHandlerRemoveArticle(g *gwyneth.Gwyneth) func(*gin.Context) {
	return func(c *gin.Context) {
		var article external.Article
		if err := c.ShouldBindJSON(&article); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		id, err := model.ParseStringId(article.Id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := g.RemoveArticle(id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"id": article.Id,
		})
	}
}

func getHandlerLookupArticles(cfg *config.Feed, g *gwyneth.Gwyneth) func(*gin.Context) {
	return func(c *gin.Context) {
		title_urlencode := c.Query("title")
		body_urlencode := c.Query("body")
		src_id_base_s := c.QueryArray("src_id")
		s_start := c.DefaultQuery("start", "-1")
		s_end := c.DefaultQuery("end", "-1")
		s_limit := c.DefaultQuery("limit", "30")
		feed_type := c.DefaultQuery("type", cfg.DefaultType)

		title, err := url.QueryUnescape(title_urlencode)
		if err != nil {
			err_msg := fmt.Sprintf("cannot parse title ('%s'): %s", title_urlencode, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err_msg})
			return
		}

		body, err := url.QueryUnescape(body_urlencode)
		if err != nil {
			err_msg := fmt.Sprintf("cannot parse body ('%s'): %s", body_urlencode, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err_msg})
			return
		}

		if s_start == "" {
			s_start = "-1"
		}
		start, err := strconv.ParseInt(s_start, 10, 64)
		if err != nil {
			err_msg := fmt.Sprintf("unkown fmt the parameter of start('%s'): %s", s_start, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err_msg})
			return
		}

		if s_end == "" {
			s_end = "-1"
		}
		end, err := strconv.ParseInt(s_end, 10, 64)
		if err != nil {
			err_msg := fmt.Sprintf("unkown fmt the parameter of end('%s'): %s", s_end, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err_msg})
			return
		}
		limit, err := strconv.ParseInt(s_limit, 10, 64)
		if err != nil {
			err_msg := fmt.Sprintf("unkown fmt the parameter of limit('%s'): %s", s_limit, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err_msg})
			return
		}

		src_ids := []*model.Id{}
		for _, src_id_base := range src_id_base_s {
			src_id, err := model.ParseStringId(src_id_base)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("cannot parse src id('%s'): %s", src_id_base, err)})
				return
			}

			src_ids = append(src_ids, src_id)
		}

		as, err := g.LookupArticles(title, body, src_ids, start, end, limit)
		if err != nil {
			err_msg := fmt.Sprintf("lookup failed: %s", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err_msg})
			return
		}

		doResponseFeed(cfg, c, as, feed_type)
	}
}

func getHandlerGetFeed(cfg *config.Feed, g *gwyneth.Gwyneth) func(*gin.Context) {
	return func(c *gin.Context) {
		id_base := c.Param("id")
		src_id, err := model.ParseStringId(id_base)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		s_limit := c.DefaultQuery("limit", "30")
		feed_type := c.DefaultQuery("type", cfg.DefaultType)

		limit, err := strconv.ParseInt(s_limit, 10, 64)
		if err != nil {
			err_msg := fmt.Sprintf("unkown fmt the parameter of limit('%s'): %s", s_limit, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err_msg})
			return
		}

		as, err := g.GetFeed(src_id, limit)
		if err != nil {
			err_msg := fmt.Sprintf("get failed: %s", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err_msg})
			return
		}

		doResponseFeed(cfg, c, as, feed_type)
	}
}

func getHandlerPostFeed(cfg *config.Feed, g *gwyneth.Gwyneth) func(*gin.Context) {
	return func(c *gin.Context) {
		id_base := c.Param("id")
		src_id, err := model.ParseStringId(id_base)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if _, err := g.GetSource(src_id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("source id is not exist: '%s'", id_base)})
			return
		}

		var article external.Article
		if err := c.ShouldBindJSON(&article); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.Debug("BindFeed: request is '%v'", article)

		article_id, err := model.ParseStringId(article.Id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("cannot parse article id('%s'): %s", article.Id, err)})
			return
		}

		if err := g.BindFeed(src_id, article_id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "success",
		})
	}
}

func getHandlerDeleteFeed(cfg *config.Feed, g *gwyneth.Gwyneth) func(*gin.Context) {
	return func(c *gin.Context) {
		id_base := c.Param("id")
		src_id, err := model.ParseStringId(id_base)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if _, err := g.GetSource(src_id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("source id is not exist: '%s'", id_base)})
			return
		}

		var article external.Article
		if err := c.ShouldBindJSON(&article); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.Debug("UnBindFeed: request is '%v'", article)

		article_id, err := model.ParseStringId(article.Id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("cannot parse article id('%s'): %s", article.Id, err)})
			return
		}

		if err := g.RemoveFeedEntry(src_id, article_id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "success",
		})
	}
}

func doResponseFeed(cfg *config.Feed, c *gin.Context, as []*model.Article, feed_type string) {
	f, err := makeFeed(cfg, as)
	if err != nil {
		err_msg := fmt.Sprintf("cannot make feed: %s", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err_msg})
		return
	}

	switch feed_type {
	case "rss":
		rss_str, err := f.ToRss()
		if err != nil {
			err_msg := fmt.Sprintf("cannot make feed: %s", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err_msg})
			return
		}
		c.Data(http.StatusOK, "application/rss+xml", []byte(rss_str))
		return

	case "atom":
		atom_str, err := f.ToAtom()
		if err != nil {
			err_msg := fmt.Sprintf("cannot make feed: %s", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err_msg})
			return
		}
		c.Data(http.StatusOK, "application/atom+xml", []byte(atom_str))
		return

	case "json":
		j_str, err := f.ToJSON()
		if err != nil {
			err_msg := fmt.Sprintf("cannot make feed: %s", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err_msg})
			return
		}
		c.Data(http.StatusOK, "application/json", []byte(j_str))
		return

	}
	err_msg := fmt.Sprintf("unsupported type: '%s'", feed_type)
	c.JSON(http.StatusBadRequest, gin.H{"error": err_msg})
	return
}

func getHandlerGetActions(g *gwyneth.Gwyneth) func(*gin.Context) {
	return func(c *gin.Context) {
		id_base := c.Query("id")
		if id_base != "" {
			id, err := model.ParseStringId(id_base)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			action, err := g.GetAction(id)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			c.IndentedJSON(http.StatusOK, []*external.Action{action.ConvertExternal()})
			return
		}

		actions, err := g.GetActions()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ret_actions := []*external.Action{}
		for _, action := range actions {
			ret_actions = append(ret_actions, action.ConvertExternal())
		}
		c.IndentedJSON(http.StatusOK, ret_actions)
	}
}

func getHandlerAddAction(g *gwyneth.Gwyneth) func(*gin.Context) {
	return func(c *gin.Context) {
		var action external.Action
		if err := c.ShouldBindJSON(&action); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.Debug("AddAciton: request is '%v'", action)

		added_action, err := g.AddAction(action.Name, action.Cmd)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.IndentedJSON(http.StatusOK, added_action.ConvertExternal())
	}
}

func getHandlerDeleteAction(g *gwyneth.Gwyneth) func(*gin.Context) {
	return func(c *gin.Context) {
		var action external.Action
		if err := c.ShouldBindJSON(&action); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		id, err := model.ParseStringId(action.Id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := g.DeleteAction(id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"id": action.Id,
		})
	}
}

func getHandlerRestartAction(g *gwyneth.Gwyneth) func(*gin.Context) {
	return func(c *gin.Context) {
		id_base := c.Param("id")
		if id_base == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id is empty"})
			return
		}
		id, err := model.ParseStringId(id_base)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := g.RestartAction(id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "success",
		})
	}
}

func getHandlerCancelAction(g *gwyneth.Gwyneth) func(*gin.Context) {
	return func(c *gin.Context) {
		id_base := c.Param("id")
		if id_base == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id is empty"})
			return
		}
		id, err := model.ParseStringId(id_base)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := g.CancelAction(id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "success",
		})
	}
}

func getHandlerGetQeueMessages(g *gwyneth.Gwyneth) func(*gin.Context) {
	return func(c *gin.Context) {
		id_base := c.Param("id")
		if id_base == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id is empty"})
			return
		}
		id, err := model.ParseStringId(id_base)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		articles, err := g.GetActionQueueItems(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ext_artcles := make([]*external.Article, len(articles), len(articles))
		for i, article := range articles {
			ext_artcles[i] = article.ConvertExternal()
		}
		c.IndentedJSON(http.StatusOK, ext_artcles)
	}
}

func getHandlerGetDlqMessages(g *gwyneth.Gwyneth) func(*gin.Context) {
	return func(c *gin.Context) {
		id_base := c.Param("id")
		if id_base == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id is empty"})
			return
		}
		id, err := model.ParseStringId(id_base)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		articles, err := g.GetActionDlqItems(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ext_artcles := make([]*external.Article, len(articles), len(articles))
		for i, article := range articles {
			ext_artcles[i] = article.ConvertExternal()
		}
		c.IndentedJSON(http.StatusOK, ext_artcles)
	}
}

func getHandlerDeleteActionQueueMessage(g *gwyneth.Gwyneth) func(*gin.Context) {
	return func(c *gin.Context) {
		action_id_base := c.Param("id")
		if action_id_base == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id is empty"})
			return
		}
		action_id, err := model.ParseStringId(action_id_base)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		msg_id_base := c.Param("msg_id")
		if msg_id_base == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id is empty"})
			return
		}
		msg_id, err := model.ParseStringId(msg_id_base)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := g.DeleteActionQueueItem(action_id, msg_id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "success",
		})
	}
}

func getHandlerDeleteActionDlqMessage(g *gwyneth.Gwyneth) func(*gin.Context) {
	return func(c *gin.Context) {
		action_id_base := c.Param("id")
		if action_id_base == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id is empty"})
			return
		}
		action_id, err := model.ParseStringId(action_id_base)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		msg_id_base := c.Param("msg_id")
		if msg_id_base == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id is empty"})
			return
		}
		msg_id, err := model.ParseStringId(msg_id_base)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := g.DeleteActionDlqItem(action_id, msg_id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "success",
		})
	}
}

func getHandlerRedriveActionMessage(g *gwyneth.Gwyneth) func(*gin.Context) {
	return func(c *gin.Context) {
		action_id_base := c.Param("id")
		if action_id_base == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id is empty"})
			return
		}
		action_id, err := model.ParseStringId(action_id_base)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		msg_id_base := c.Param("msg_id")
		if msg_id_base == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id is empty"})
			return
		}
		msg_id, err := model.ParseStringId(msg_id_base)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := g.RedriveActionDlqItem(action_id, msg_id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "success",
		})
	}
}

func getHandlerGetFilters(g *gwyneth.Gwyneth) func(*gin.Context) {
	return func(c *gin.Context) {
		id_base := c.Query("id")
		if id_base != "" {
			id, err := model.ParseStringId(id_base)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			f, err := g.GetFilter(id)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			c.IndentedJSON(http.StatusOK, []*external.Filter{f.ConvertExternal()})
			return
		}

		fs, err := g.GetFilters()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ret_fs := []*external.Filter{}
		for _, f := range fs {
			ret_fs = append(ret_fs, f.ConvertExternal())
		}
		c.IndentedJSON(http.StatusOK, ret_fs)
	}
}

func getHandlerAddFilter(g *gwyneth.Gwyneth) func(*gin.Context) {
	return func(c *gin.Context) {
		var f external.Filter
		if err := c.ShouldBindJSON(&f); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.Debug("AddFilter: request is '%v'", f)

		action_id, err := model.ParseStringId(f.Action.Id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		added_f, err := g.AddFilter(f.Title.Value, f.Title.IsRegex,
									f.Body.Value, f.Body.IsRegex, action_id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.IndentedJSON(http.StatusOK, added_f.ConvertExternal())
	}
}

func getHandlerUpdateFilter(g *gwyneth.Gwyneth) func(*gin.Context) {
	return func(c *gin.Context) {
		var f external.Filter
		if err := c.ShouldBindJSON(&f); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		id, err := model.ParseStringId(f.Id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		action_id, err := model.ParseStringId(f.Action.Id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		updated_f, err := g.UpdateFilterAction(id, action_id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, updated_f.ConvertExternal())
	}
}

func getHandlerDeleteFilter(g *gwyneth.Gwyneth) func(*gin.Context) {
	return func(c *gin.Context) {
		var f external.Filter
		if err := c.ShouldBindJSON(&f); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		id, err := model.ParseStringId(f.Id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := g.DeleteFilter(id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"id": f.Id,
		})
	}
}

func getHandlerGetSource(g *gwyneth.Gwyneth) func(*gin.Context) {
	return func(c *gin.Context) {
		id_base := c.Param("id")
		id, err := model.ParseStringId(id_base)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		src, err := g.GetSource(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ext_src := src.ConvertExternal()
		sts := g.GetSourceStatus(id)
		for _, st := range sts {
			ext_src.Status = append(ext_src.Status, st.ConvertExternal())
		}

		c.IndentedJSON(http.StatusOK, ext_src)
	}
}

func getHandlerPauseSource(g *gwyneth.Gwyneth) func(*gin.Context) {
	return func(c *gin.Context) {
		id_base := c.Param("id")
		id, err := model.ParseStringId(id_base)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := g.PauseSource(id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "success",
		})
	}
}
func getHandlerResumeSource(g *gwyneth.Gwyneth) func(*gin.Context) {
	return func(c *gin.Context) {
		id_base := c.Param("id")
		id, err := model.ParseStringId(id_base)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := g.ResumeSource(id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "success",
		})
	}
}

func getHandlerReFilter(g *gwyneth.Gwyneth) func(*gin.Context) {
	return func(c *gin.Context) {
		id_base := c.Param("id")
		id, err := model.ParseStringId(id_base)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.Debug("ReFilter: request is '%v'", id)

		var json_data struct {
			Limit int `json:"limit"`
		}
		if err := c.ShouldBindJSON(&json_data); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		limit := 50
		if json_data.Limit > 0 {
			limit = json_data.Limit
		}
		if err := g.ReFilter(id, int64(limit)); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}

		c.JSON(http.StatusOK, gin.H{})
	}
}

func getHandlerBindFilter(g *gwyneth.Gwyneth) func(*gin.Context) {
	return func(c *gin.Context) {
		id_base := c.Param("id")
		id, err := model.ParseStringId(id_base)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var f external.Filter
		if err := c.ShouldBindJSON(&f); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.Debug("BindFilter: request is '%v'", f)

		f_id, err := model.ParseStringId(f.Id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := g.BindFilter(id, f_id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		fs, err := g.GetFilterOnSource(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ret_fs := []*external.Filter{}
		for _, f := range fs {
			ret_fs = append(ret_fs, f.ConvertExternal())
		}
		c.IndentedJSON(http.StatusOK, ret_fs)
	}
}

func getHandlerGetFilterOnSource(g *gwyneth.Gwyneth) func(*gin.Context) {
	return func(c *gin.Context) {
		id_base := c.Param("id")
		id, err := model.ParseStringId(id_base)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		fs, err := g.GetFilterOnSource(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ret_fs := []*external.Filter{}
		for _, f := range fs {
			ret_fs = append(ret_fs, f.ConvertExternal())
		}
		c.IndentedJSON(http.StatusOK, ret_fs)
	}
}

func getHandlerUnBindFilter(g *gwyneth.Gwyneth) func(*gin.Context) {
	return func(c *gin.Context) {
		id_base := c.Param("id")
		id, err := model.ParseStringId(id_base)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var f external.Filter
		if err := c.ShouldBindJSON(&f); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.Debug("BindFilter: request is '%v'", f)

		f_id, err := model.ParseStringId(f.Id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := g.UnBindFilter(id, f_id); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		fs, err := g.GetFilterOnSource(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ret_fs := []*external.Filter{}
		for _, f := range fs {
			ret_fs = append(ret_fs, f.ConvertExternal())
		}
		c.IndentedJSON(http.StatusOK, ret_fs)
	}
}
