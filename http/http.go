package http

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"
)

import (
	"github.com/l4go/task"
	"github.com/gin-gonic/gin"
)

import (
	"github.com/hinoshiba/gwyneth"
	"github.com/hinoshiba/gwyneth/consts"
	"github.com/hinoshiba/gwyneth/config"
	"github.com/hinoshiba/gwyneth/structs"
	"github.com/hinoshiba/gwyneth/structs/external"
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

func (self *Router) mapRoute(g *gwyneth.Gwyneth) error {
	self.engine.LoadHTMLGlob("/usr/local/src/http/templates/*")
	self.engine.Static("/static", "/usr/local/src/http/static")
	self.engine.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"message": fmt.Sprintf("Welcome to Gwyneth %s", consts.VERSION),
		})
	})

	self.engine.GET("/search", func(c *gin.Context) {
		c.HTML(http.StatusOK, "search.html", gin.H{
			"message": fmt.Sprintf("Gwyneth %s", consts.VERSION),
		})
	})

	self.engine.GET("/source_type", func(c *gin.Context) {
		c.HTML(http.StatusOK, "source_type.html", gin.H{
			"message": fmt.Sprintf("Gwyneth %s", consts.VERSION),
		})
	})

	self.engine.GET("/source", func(c *gin.Context) {
		c.HTML(http.StatusOK, "source.html", gin.H{
			"message": fmt.Sprintf("Gwyneth %s", consts.VERSION),
		})
	})

	self.engine.GET("/source/:id", func(c *gin.Context) {
		src_id := c.Param("id")

		c.HTML(http.StatusOK, "source_detail.html", gin.H{
			"message": fmt.Sprintf("Gwyneth %s", consts.VERSION),
			"src_id": fmt.Sprintf("%s", src_id),
		})
	})

	self.engine.GET("/action", func(c *gin.Context) {
		c.HTML(http.StatusOK, "action.html", gin.H{
			"message": fmt.Sprintf("Gwyneth %s", consts.VERSION),
		})
	})

	self.engine.GET("/filter", func(c *gin.Context) {
		c.HTML(http.StatusOK, "filter.html", gin.H{
			"message": fmt.Sprintf("Gwyneth %s", consts.VERSION),
		})
	})

	self.engine.GET("/filter/:id", func(c *gin.Context) {
		filter_id := c.Param("id")

		c.HTML(http.StatusOK, "filter_detail.html", gin.H{
			"message": fmt.Sprintf("Gwyneth %s", consts.VERSION),
			"filter_id": fmt.Sprintf("%s", filter_id),
		})
	})

	self.engine.GET("/api/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	self.engine.GET("/api/source_type", getHandlerGetSourceTypes(g))
	self.engine.POST("/api/source_type", getHandlerAddSourceType(g))
	self.engine.DELETE("api/source_type", getHandlerDeleteSourceType(g))

	self.engine.GET("/api/source", getHandlerGetSources(g))
	self.engine.POST("/api/source", getHandlerAddSource(g))
	self.engine.DELETE("/api/source", getHandlerRemoveSource(g))

	self.engine.GET("/api/source/:id", getHandlerGetSource(g))
	self.engine.POST("/api/source/:id/filter", getHandlerBindFilter(g))
	self.engine.GET("/api/source/:id/filter", getHandlerGetFilterOnSource(g))
	self.engine.DELETE("/api/source/:id/filter", getHandlerUnBindFilter(g))
	self.engine.POST("/api/source/:id/pause", getHandlerPauseSource(g))
	self.engine.POST("/api/source/:id/resume", getHandlerResumeSource(g))

	self.engine.GET("/api/article", getHandlerLookupArticles(self.cfg.Feed, g))
	self.engine.POST("/api/article", getHandlerAddArticle(g))
	self.engine.DELETE("/api/article", getHandlerRemoveArticle(g))

	self.engine.GET("/api/feed/:id", getHandlerGetFeed(self.cfg.Feed, g))
	self.engine.POST("/api/feed/:id", getHandlerPostFeed(self.cfg.Feed, g))
	self.engine.DELETE("/api/feed/:id", getHandlerDeleteFeed(self.cfg.Feed, g))
	self.engine.POST("/api/feed/:id/refilter", getHandlerReFilter(g))

	self.engine.GET("/api/action", getHandlerGetActions(g))
	self.engine.POST("/api/action", getHandlerAddAction(g))
	self.engine.DELETE("/api/action", getHandlerDeleteAction(g))

	self.engine.GET("/api/filter", getHandlerGetFilters(g))
	self.engine.POST("/api/filter", getHandlerAddFilter(g))
	self.engine.PATCH("/api/filter", getHandlerUpdateFilter(g))
	self.engine.DELETE("/api/filter", getHandlerDeleteFilter(g))

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
		slog.Debug(fmt.Sprintf("AddSourceType: request is '%v'", st))

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
			id, err := structs.ParseStringId(id_base)
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
		slog.Debug(fmt.Sprintf("DeleteSourceType: request is '%v'", st))

		id, err := structs.ParseStringId(st.Id)
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
		slog.Debug(fmt.Sprintf("AddSource: request is '%v'", src))

		src_type_id, err := structs.ParseStringId(src.Type.Id)
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
			id, err := structs.ParseStringId(id_base)
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
			ret_src = append(ret_src, src.ConvertExternal())
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

		id, err := structs.ParseStringId(src.Id)
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
		slog.Debug(fmt.Sprintf("AddArticle: request is '%v'", article))

		src_id, err := structs.ParseStringId(article.Src.Id)
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

		id, err := structs.ParseStringId(article.Id)
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

		src_ids := []*structs.Id{}
		for _, src_id_base := range src_id_base_s {
			src_id, err := structs.ParseStringId(src_id_base)
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
		src_id, err := structs.ParseStringId(id_base)
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
		src_id, err := structs.ParseStringId(id_base)
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
		slog.Debug(fmt.Sprintf("BindFeed: request is '%v'", article))

		article_id, err := structs.ParseStringId(article.Id)
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
		src_id, err := structs.ParseStringId(id_base)
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
		slog.Debug(fmt.Sprintf("UnBindFeed: request is '%v'", article))

		article_id, err := structs.ParseStringId(article.Id)
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

func doResponseFeed(cfg *config.Feed, c *gin.Context, as []*structs.Article, feed_type string) {
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
			id, err := structs.ParseStringId(id_base)
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
		slog.Debug(fmt.Sprintf("AddAciton: request is '%v'", action))

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

		id, err := structs.ParseStringId(action.Id)
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

func getHandlerGetFilters(g *gwyneth.Gwyneth) func(*gin.Context) {
	return func(c *gin.Context) {
		id_base := c.Query("id")
		if id_base != "" {
			id, err := structs.ParseStringId(id_base)
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
		slog.Debug(fmt.Sprintf("AddFilter: request is '%v'", f))

		action_id, err := structs.ParseStringId(f.Action.Id)
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

		id, err := structs.ParseStringId(f.Id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		action_id, err := structs.ParseStringId(f.Action.Id)
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

		id, err := structs.ParseStringId(f.Id)
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
		id, err := structs.ParseStringId(id_base)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		src, err := g.GetSource(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.IndentedJSON(http.StatusOK, src.ConvertExternal())
	}
}

func getHandlerPauseSource(g *gwyneth.Gwyneth) func(*gin.Context) {
	return func(c *gin.Context) {
		id_base := c.Param("id")
		id, err := structs.ParseStringId(id_base)
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
		id, err := structs.ParseStringId(id_base)
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
		id, err := structs.ParseStringId(id_base)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.Debug(fmt.Sprintf("ReFilter: request is '%v'", id))

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
		id, err := structs.ParseStringId(id_base)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var f external.Filter
		if err := c.ShouldBindJSON(&f); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.Debug(fmt.Sprintf("BindFilter: request is '%v'", f))

		f_id, err := structs.ParseStringId(f.Id)
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
		id, err := structs.ParseStringId(id_base)
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
		id, err := structs.ParseStringId(id_base)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var f external.Filter
		if err := c.ShouldBindJSON(&f); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.Debug(fmt.Sprintf("BindFilter: request is '%v'", f))

		f_id, err := structs.ParseStringId(f.Id)
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
