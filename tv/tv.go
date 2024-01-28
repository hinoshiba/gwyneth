package tv

import (
	"fmt"
	"sync"
	"log/slog"
)

import (
	"github.com/l4go/task"
)

import (
	"github.com/hinoshiba/gwyneth/config"
	"github.com/hinoshiba/gwyneth/tv/database"
)

type TimeVortex struct {
	db  database.Session

	msn *task.Mission
	mtx *sync.Mutex
}

func New(msn *task.Mission, cfg *config.Config) (*TimeVortex, error) {
	db, err := database.Connect(msn.New(), cfg.Database)
	if err != nil {
		msn.Done()
		return nil, err
	}

	return &TimeVortex{
		db: db,
		msn: msn,
		mtx: new(sync.Mutex),
	}, nil
}

func (self *TimeVortex) Close() error {
	defer self.msn.Done()

	self.msn.Cancel()

	return self.db.Close()
}

func (self *TimeVortex) Test() error {
	st, err := self.db.AddSourceType("john", "", false)
	if err != nil {
		return err
	}
	slog.Debug(fmt.Sprintf("%s", st))
	sts, err := self.db.GetSourceTypes()
	if err != nil {
		return err
	}
	slog.Debug(fmt.Sprintf("%s", sts))
	if err := self.db.DeleteSourceType(st.Id()); err != nil {
		return err
	}
	slog.Debug("deleted")
	sts, err = self.db.GetSourceTypes()
	if err != nil {
		return err
	}
	slog.Debug(fmt.Sprintf("%s", sts))
	return nil
}
