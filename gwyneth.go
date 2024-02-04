package gwyneth

import (
	"github.com/l4go/task"
)

import (
	"github.com/hinoshiba/gwyneth/config"
	"github.com/hinoshiba/gwyneth/tv"
)

type Gwyneth struct {
	tv  *tv.TimeVortex
	msn *task.Mission
}

func New(msn *task.Mission, cfg *config.Config) (*Gwyneth, error) {
	t, err := tv.New(msn.New(), cfg)
	if err != nil {
		return nil, err
	}
	return &Gwyneth {
		tv: t,
		msn: msn,
	}, nil
}

func (self *Gwyneth) Close() error {
	defer self.msn.Done()

	self.msn.Cancel()

	return self.tv.Close()
}

func (self *Gwyneth) Test() error {
	return self.tv.Test()
}
