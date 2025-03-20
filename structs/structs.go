package structs

import (
	"fmt"
)

import (
	"github.com/google/uuid"
)

import (
	"github.com/hinoshiba/gwyneth/structs/external"
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

func ParseStringId(id_base string) (*Id, error) {
	id_buf, err := uuid.Parse(id_base)
	if err != nil {
		return nil, fmt.Errorf("id size error: %s", err)
	}
	id, err := id_buf.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("id size error: %s", err)
	}
	if len(id) != 16 {
		return nil, fmt.Errorf("id size error.")
	}

	return &Id { id: string(id) }, nil
}

func (self *Id) Value() []byte {
	return []byte(self.id)
}

func (self *Id) String() string {
	uuid_base, err := uuid.FromBytes([]byte(self.id))
	if err != nil {
		panic(fmt.Sprintf("id size error: %s, '%v'", err, []byte(self.id)))
	}
	return uuid_base.String()

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

func (self *SourceType) ConvertExternal() *external.SourceType {
	return &external.SourceType{
		Id: self.id.String(),
		Name: self.name,
		Cmd: self.cmd,
		UserCreate: self.user_create,
	}
}

type Source struct {
	id       *Id
	title    string
	src_type *SourceType
	val      string
	pause    bool
}

func NewSource(id *Id, title string, src_type *SourceType, val string, pause bool) *Source {
	return &Source {
		id: id,
		title: title,
		src_type: src_type,
		val: val,
		pause: pause,
	}
}

func (self *Source) Id() *Id {
	return self.id
}

func (self *Source) Title() string {
	return self.title
}

func (self *Source) Type() *SourceType {
	return self.src_type
}

func (self *Source) Value() string {
	return self.val
}

func (self *Source) IsPause() bool {
	return self.pause
}

func (self *Source) ConvertExternal() *external.Source {
	return &external.Source {
		Id: self.id.String(),
		Title: self.title,
		Type: self.src_type.ConvertExternal(),
		Value: self.val,
		Pause: self.pause,
	}
}

type Article struct {
	id    *Id
	src   *Source
	title string
	body  string
	link  string
	utime int64
	raw   string
}

func NewArticle(id *Id, src *Source, title string, body string, link string, utime int64, raw string) *Article {
	return &Article{
		id: id,
		src: src,
		title: title,
		body: body,
		link: link,
		utime: utime,
		raw: raw,
	}
}

func (self *Article) Id() *Id {
	return self.id
}

func (self *Article) Src() *Source {
	return self.src
}

func (self *Article) Title() string {
	return self.title
}

func (self *Article) Body() string {
	return self.body
}

func (self *Article) Link() string {
	return self.link
}

func (self *Article) Unixtime() int64 {
	return self.utime
}

func (self *Article) Raw() string {
	return self.raw
}

func (self *Article) ConvertExternal() *external.Article {
	return &external.Article{
		Id: self.id.String(),
		Src: self.src.ConvertExternal(),
		Title: self.title,
		Body: self.body,
		Link: self.link,
		Timestamp: int(self.utime),
		Raw: self.raw,
	}
}
