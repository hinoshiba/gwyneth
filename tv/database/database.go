package database

import (
	"fmt"
)

import (
	"github.com/l4go/task"
)

import (
	"github.com/hinoshiba/gwyneth/config"
	"github.com/hinoshiba/gwyneth/tv/database/mysql"

	"github.com/hinoshiba/gwyneth/structs"
)

var (
	ErrNotConnect error = fmt.Errorf("not connect to a storage.")
)

type Session interface {
	Close() error
	AddSourceType(string, string, bool) (*structs.SourceType, error)
	GetSourceType(*structs.Id) (*structs.SourceType, error)
	GetSourceTypes() ([]*structs.SourceType, error)
	DeleteSourceType(*structs.Id) error

	AddSource(string, *structs.Id, string) (*structs.Source, error)
	GetSource(*structs.Id) (*structs.Source, error)
	GetSources() ([]*structs.Source, error)
	FindSource(string) ([]*structs.Source, error)
	DeleteSource(*structs.Id) error

	AddArticle(string, string, string, int64, string, *structs.Id) (*structs.Article, error)
	LookupArticles(string, string, []*structs.Id, int64, int64, int64) ([]*structs.Article, error)
	RemoveArticle(*structs.Id) error

	GetFeed(*structs.Id, int64) ([]*structs.Article, error)
	BindFeed(*structs.Id, *structs.Id) error
	RemoveFeedEntry(*structs.Id, *structs.Id) error

	AddAction(name string, cmd string) (*structs.Action, error)
	GetAction(id *structs.Id) (*structs.Action, error)
	GetActions() ([]*structs.Action, error)
	DeleteAction(id *structs.Id) error

	AddFilter(title string, regex_title bool, body string, regex_body bool, action_id *structs.Id) (*structs.Filter, error)
	UpdateFilterAction(id *structs.Id, action_id *structs.Id) (*structs.Filter, error)
	GetFilter(id *structs.Id) (*structs.Filter, error)
	GetFilters() ([]*structs.Filter, error)
	DeleteFilter(id *structs.Id) error
}

func Connect(msn *task.Mission, cfg *config.Database) (Session, error) {
	return mysql.NewSession(msn, cfg)
}
