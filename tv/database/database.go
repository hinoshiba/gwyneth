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
	GetSourceTypes() ([]*structs.SourceType, error)
	DeleteSourceType(*structs.Id) error
	/*
	GetSourceType()
	DeleteSourceType()

	AddSource()
	GetSources()
	FindSource()
	GetSource()
	DeleteSource()

	AddArticle()
	BatchAddArticle()
	LookupArticles()
	RemoveArticle()

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
