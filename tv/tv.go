package tv

import (
	"sync"
//	"log/slog"
)

import (
	"github.com/l4go/task"
)

import (
	"github.com/hinoshiba/gwyneth/config"
	"github.com/hinoshiba/gwyneth/tv/database"
	"github.com/hinoshiba/gwyneth/structs"
)

type TimeVortex struct {
	db  database.Session

	msn *task.Mission
	mtx *sync.RWMutex
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
		mtx: new(sync.RWMutex),
	}, nil
}

func (self *TimeVortex) Close() error {
	defer self.msn.Done()

	self.msn.Cancel()

	self.mtx.Lock()
	defer self.mtx.Unlock()

	return self.db.Close()
}

func (self *TimeVortex) AddSourceType(name string, cmd string, is_user_creation bool) (*structs.SourceType, error) {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	return self.addSourceType(name, cmd, is_user_creation)
}

func (self *TimeVortex) addSourceType(name string, cmd string, is_user_creation bool) (*structs.SourceType, error) {
	return self.db.AddSourceType(name, cmd, is_user_creation)
}

func (self *TimeVortex) GetSourceTypes() ([]*structs.SourceType, error) {
	self.mtx.RLock()
	defer self.mtx.RUnlock()

	return self.getSourceTypes()
}

func (self *TimeVortex) getSourceTypes() ([]*structs.SourceType, error) {
	return self.db.GetSourceTypes()
}

func (self *TimeVortex) GetSourceType(id *structs.Id) (*structs.SourceType, error) {
	self.mtx.RLock()
	defer self.mtx.RUnlock()

	return self.getSourceType(id)
}

func (self *TimeVortex) getSourceType(id *structs.Id) (*structs.SourceType, error) {
	return self.db.GetSourceType(id)
}

func (self *TimeVortex) DeleteSourceType(id *structs.Id) error {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	return self.deleteSourceType(id)
}

func (self *TimeVortex) deleteSourceType(id *structs.Id) error {
	return self.db.DeleteSourceType(id)
}

func (self *TimeVortex) AddSource(title string, src_type *structs.SourceType, val string) (*structs.Source, error) {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	return self.addSource(title, src_type, val)
}

func (self *TimeVortex) addSource(title string, src_type *structs.SourceType, val string) (*structs.Source, error) {
	return self.db.AddSource(title, src_type.Id(), val)
}

func (self *TimeVortex) GetSources() ([]*structs.Source, error) {
	self.mtx.RLock()
	defer self.mtx.RUnlock()

	return self.getSources()
}

func (self *TimeVortex) getSources() ([]*structs.Source, error) {
	return self.db.GetSources()
}

func (self *TimeVortex) FindSource(kw string) ([]*structs.Source, error) {
	self.mtx.RLock()
	defer self.mtx.RUnlock()

	return self.findSource(kw)
}

func (self *TimeVortex) findSource(kw string) ([]*structs.Source, error) {
	return self.db.FindSource(kw)
}

func (self *TimeVortex) GetSource(id *structs.Id) (*structs.Source, error) {
	self.mtx.RLock()
	defer self.mtx.RUnlock()

	return self.getSource(id)
}

func (self *TimeVortex) getSource(id *structs.Id) (*structs.Source, error) {
	return self.db.GetSource(id)
}

func (self *TimeVortex) DeleteSource(id *structs.Id) error {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	return self.deleteSource(id)
}

func (self *TimeVortex) deleteSource(id *structs.Id) error {
	return self.db.DeleteSource(id)
}
