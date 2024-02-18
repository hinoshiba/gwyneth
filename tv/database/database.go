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
	/*
	AddNoticer()
	GetNoticers()
	DeleteNoticer()

	AddFilterType()
	GetFilterTypes()
	GetFilterType()
	DeleteFilterType()

	AddFilter()
	GetFilters()
	GetFilter()
	DeleteFilter()
	*/
}

func Connect(msn *task.Mission, cfg *config.Database) (Session, error) {
	return mysql.NewSession(msn, cfg)
}
