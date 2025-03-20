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
	"github.com/hinoshiba/gwyneth/filter"
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
	RemoveSource(*structs.Id) error
	PauseSource(*structs.Id) error
	ResumeSource(*structs.Id) error

	AddArticle(string, string, string, int64, string, *structs.Id) (*structs.Article, error)
	LookupArticles(string, string, []*structs.Id, int64, int64, int64) ([]*structs.Article, error)
	RemoveArticle(*structs.Id) error

	GetFeed(*structs.Id, int64) ([]*structs.Article, error)
	BindFeed(*structs.Id, *structs.Id) error
	RemoveFeedEntry(*structs.Id, *structs.Id) error

	AddAction(name string, cmd string) (*filter.Action, error)
	GetAction(id *structs.Id) (*filter.Action, error)
	GetActions() ([]*filter.Action, error)
	DeleteAction(id *structs.Id) error

	AddFilter(title string, regex_title bool, body string, regex_body bool, action_id *structs.Id) (*filter.Filter, error)
	UpdateFilterAction(id *structs.Id, action_id *structs.Id) (*filter.Filter, error)
	GetFilter(id *structs.Id) (*filter.Filter, error)
	GetFilters() ([]*filter.Filter, error)
	DeleteFilter(id *structs.Id) error

	BindFilter(src_id *structs.Id, f_id *structs.Id) error
	UnBindFilter(src_id *structs.Id, f_id *structs.Id) error
	GetFilterOnSource(src_id *structs.Id) ([]*filter.Filter, error)
	GetSourceWithEnabledFilter(f_id *structs.Id) ([]*structs.Source, error)
}

func Connect(msn *task.Mission, cfg *config.Database) (Session, error) {
	return mysql.NewSession(msn, cfg)
}
