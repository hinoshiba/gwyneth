package gwyneth

import (
	"fmt"
)

import (
	"github.com/l4go/task"
)

import (
	"github.com/hinoshiba/gwyneth/config"
	"github.com/hinoshiba/gwyneth/tv"
	"github.com/hinoshiba/gwyneth/structs"
)

type Gwyneth struct {
	tv  *tv.TimeVortex
	msn *task.Mission

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
	return nil
}

func (self *Gwyneth) checkAndInitSourceTypes() error {
	defaults := map[string]string{
		"rss": "rss",
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

/*
	AddSource(string, *structs.Id, string) (*structs.Source, error)
	GetSource(*structs.Id) (*structs.Source, error)
	GetSources() ([]*structs.Source, error)
	FindSource(string) ([]*structs.Source, error)
	DeleteSource(*structs.Id) error
*/
