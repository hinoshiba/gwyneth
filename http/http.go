package http

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
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
	self.engine.GET("/source_type", func(c *gin.Context) {
		c.HTML(http.StatusOK, "source_type.html", gin.H{})
	})
	self.engine.GET("/source", func(c *gin.Context) {
		c.HTML(http.StatusOK, "source.html", gin.H{})
	})

	self.engine.GET("/api/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	self.engine.GET("/api/source_type", getHandlerGetSourceType(g))
	self.engine.POST("/api/source_type", getHandlerAddSourceType(g))
	self.engine.DELETE("api/source_type", getHandlerDeleteSourceType(g))

	self.engine.GET("/api/source", getHandlerGetSource(g))
	self.engine.POST("/api/source", getHandlerAddSource(g))
	self.engine.DELETE("/api/source", getHandlerDeleteSource(g))

	self.engine.GET("/api/article", getHandlerLookupArticles(self.cfg.Feed, g))
	self.engine.POST("/api/article", getHandlerAddArticle(g))
	self.engine.DELETE("/api/article", getHandlerRemoveArticle(g))

	self.engine.GET("/api/feed", getHandlerGetFeed(self.cfg.Feed, g))
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
		var st SourceType
		if err := c.ShouldBindJSON(&st); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.Debug(fmt.Sprintf("request is '%v'", st))

		added_st, err := g.AddSourceType(st.Name, st.Cmd, true)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.IndentedJSON(http.StatusOK, convSourceType(added_st))
	}
}

func getHandlerGetSourceType(g *gwyneth.Gwyneth) func(*gin.Context) {
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

			c.IndentedJSON(http.StatusOK, []*SourceType{convSourceType(st)})
			return
		}

		sts, err := g.GetSourceTypes()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ret_sts := []*SourceType{}
		for _, st := range sts {
			ret_sts = append(ret_sts, convSourceType(st))
		}
		c.IndentedJSON(http.StatusOK, ret_sts)
	}
}

func getHandlerDeleteSourceType(g *gwyneth.Gwyneth) func(*gin.Context) {
	return func(c *gin.Context) {
		var st SourceType
		if err := c.ShouldBindJSON(&st); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.Debug(fmt.Sprintf("request is '%v'", st))

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
		var src Source
		if err := c.ShouldBindJSON(&src); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.Debug(fmt.Sprintf("request is '%v'", src))

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

		c.IndentedJSON(http.StatusOK, convSource(added_src))
	}
}

func getHandlerGetSource(g *gwyneth.Gwyneth) func(*gin.Context) {
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

			c.IndentedJSON(http.StatusOK, []*Source{convSource(src)})
			return
		}

		srcs, err := g.GetSources()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ret_src := []*Source{}
		for _, src := range srcs {
			ret_src = append(ret_src, convSource(src))
		}
		c.IndentedJSON(http.StatusOK, ret_src)
	}
}

func getHandlerDeleteSource(g *gwyneth.Gwyneth) func(*gin.Context) {
	return func(c *gin.Context) {
		var src Source
		if err := c.ShouldBindJSON(&src); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		id, err := structs.ParseStringId(src.Id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := g.DeleteSource(id); err != nil {
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
		var article Article
		if err := c.ShouldBindJSON(&article); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		slog.Debug(fmt.Sprintf("request is '%v'", article))

		src_id, err := structs.ParseStringId(article.Src.Id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("cannot parse src id('%s'): %s", article.Src.Id, err)})
			return
		}

		added_article, err := g.AddArticle(article.Title, article.Body, article.Link, int64(article.Timestamp), article.Raw, src_id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.IndentedJSON(http.StatusOK, convArticle(added_article))
	}
}

func getHandlerRemoveArticle(g *gwyneth.Gwyneth) func(*gin.Context) {
	return func(c *gin.Context) {
		var article Article
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
		title := c.Query("title")
		body := c.Query("body")
		s_start := c.DefaultQuery("start", "-1")
		s_end := c.DefaultQuery("end", "-1")
		s_limit := c.DefaultQuery("limit", "-1")
		feed_type := c.DefaultQuery("type", cfg.DefaultType)

		start, err := strconv.ParseInt(s_start, 10, 64)
		if err != nil {
			err_msg := fmt.Sprintf("unkown fmt the parameter of start('%s'): %s", s_start, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err_msg})
			return
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

		as, err := g.LookupArticles(title, body, nil, start, end, limit)
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
		src_id_base := c.Query("src_id")
		s_limit := c.DefaultQuery("limit", "30")
		feed_type := c.DefaultQuery("type", cfg.DefaultType)

		src_id, err := structs.ParseStringId(src_id_base)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("cannot parse src id('%s'): %s", src_id_base, err)})
			return
		}
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
