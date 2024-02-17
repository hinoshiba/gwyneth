package http

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
)

import (
	"github.com/l4go/task"
	"github.com/gin-gonic/gin"
)

import (
	"github.com/hinoshiba/gwyneth"
	"github.com/hinoshiba/gwyneth/config"
	"github.com/hinoshiba/gwyneth/structs"
)

func init() {
	gin.SetMode(gin.ReleaseMode)
}

type Router struct {
	engine *gin.Engine

	cfg    *config.Http

	msn    *task.Mission
}

func New(msn *task.Mission, cfg *config.Http, g *gwyneth.Gwyneth) (*Router, error) {
	self := &Router{
		engine: gin.Default(),
		cfg: cfg,
		msn: msn,
	}

	self.map_route(g)
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

func (self *Router) map_route(g *gwyneth.Gwyneth) error {
	self.engine.LoadHTMLGlob("/usr/local/src/http/templates/*")
	self.engine.Static("/static", "/usr/local/src/http/static")
	self.engine.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
		 "message": "Hi, i'm gwyneth.",
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
	return nil

}

func (self *Router) run() error {
	c_msn := self.msn.New()

	lc := net.ListenConfig{}
	ln, err := lc.Listen(c_msn.AsContext(), "tcp", self.cfg.GetAddr())
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
