package http

import (
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
	self.engine.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Hello World",
		})
	})
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
