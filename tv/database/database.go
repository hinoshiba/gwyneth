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

	"github.com/hinoshiba/gwyneth/model"
	"github.com/hinoshiba/gwyneth/filter"
)

var (
	ErrNotConnect error = fmt.Errorf("not connect to a storage.")
)

type Session interface {
	Close() error
	AddSourceType(string, string, bool) (*model.SourceType, error)
	GetSourceType(*model.Id) (*model.SourceType, error)
	GetSourceTypes() ([]*model.SourceType, error)
	DeleteSourceType(*model.Id) error

	AddSource(string, *model.Id, string) (*model.Source, error)
	GetSource(*model.Id) (*model.Source, error)
	GetSources() ([]*model.Source, error)
	FindSource(string) ([]*model.Source, error)
	RemoveSource(*model.Id) error
	PauseSource(*model.Id) error
	ResumeSource(*model.Id) error

	AddArticle(string, string, string, int64, string, *model.Id) (*model.Article, error)
	LookupArticles(string, string, []*model.Id, int64, int64, int64) ([]*model.Article, error)
	RemoveArticle(*model.Id) error

	GetFeed(*model.Id, int64) ([]*model.Article, error)
	BindFeed(*model.Id, *model.Id) error
	RemoveFeedEntry(*model.Id, *model.Id) error

	AddAction(name string, cmd string) (*filter.Action, error)
	GetAction(id *model.Id) (*filter.Action, error)
	GetActions() ([]*filter.Action, error)
	DeleteAction(id *model.Id) error

	AddFilter(title string, regex_title bool, body string, regex_body bool, action_id *model.Id) (*filter.Filter, error)
	UpdateFilterAction(id *model.Id, action_id *model.Id) (*filter.Filter, error)
	GetFilter(id *model.Id) (*filter.Filter, error)
	GetFilters() ([]*filter.Filter, error)
	DeleteFilter(id *model.Id) error

	BindFilter(src_id *model.Id, f_id *model.Id) error
	UnBindFilter(src_id *model.Id, f_id *model.Id) error
	GetFilterOnSource(src_id *model.Id) ([]*filter.Filter, error)
	GetSourceWithEnabledFilter(f_id *model.Id) ([]*model.Source, error)
}

func Connect(msn *task.Mission, cfg *config.Database) (Session, error) {
	return mysql.NewSession(msn, cfg)
}
