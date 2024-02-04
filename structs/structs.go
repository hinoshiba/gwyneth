package structs

import (
	"fmt"
)

import (
	"github.com/google/uuid"
)

type Id struct {
	id string
}

func NewId(id []byte) *Id {
	if id == nil {
		id_base := uuid.New()

		var err error
		id, err = id_base.MarshalBinary()
		if err != nil {
			panic(fmt.Sprintf("id size error: %s", err))
		}
	}
	if len(id) != 16 {
		panic("id size error.")
	}

	return &Id { id: string(id) }
}

func (self *Id) Value() []byte {
	return []byte(self.id)
}

type SourceType struct {
	id   *Id
	name string
	cmd  string
	user_create bool
}

func NewSourceType(id *Id, name string, cmd string, user_create bool) *SourceType {
	return &SourceType {
		id: id,
		name: name,
		cmd: cmd,
		user_create: user_create,
	}
}

func (self *SourceType) Id() *Id {
	return self.id
}

func (self *SourceType) Name() string {
	return self.name
}

func (self *SourceType) Command() string {
	return self.cmd
}

func (self *SourceType) IsUserCreate() bool {
	return self.user_create
}

type Source struct {}
type Article struct {}
type Noticer struct {}
type FilterAction struct {}
type Filter struct {}
