package http

import (
	"time"
)

import (
	"github.com/gorilla/feeds"
)

import (
	"github.com/hinoshiba/gwyneth/config"
	"github.com/hinoshiba/gwyneth/consts"
	"github.com/hinoshiba/gwyneth/structs"
)

func makeFeed(cfg *config.Feed, as []*structs.Article) (*feeds.Feed, error) {
	var lt int64 = 0

	items := make([]*feeds.Item, len(as), len(as))
	for i, article := range as {
		t := time.Unix(article.Unixtime(), 0)
		t_jst := t.In(consts.TZ_JST)

		items[i] = &feeds.Item{
			Title: article.Title(),
			Description: article.Body(),
			Link: &feeds.Link{Href: article.Link()},
			Source: &feeds.Link{Href: article.Src().Value()},
			Id: article.Id().String(),
			Created: t_jst,
			Content: article.Raw(),
		}

		if !(lt < article.Unixtime()) {
			continue
		}
		lt = article.Unixtime()
	}

	if lt == 0 {
		lt = time.Now().Unix()
	}
	t := time.Unix(lt, 0)
	t_jst := t.In(consts.TZ_JST)
	return &feeds.Feed{
		Title:       cfg.Title,
		Description: cfg.Description,
		Link:        &feeds.Link{Href: cfg.Link},
		Author:      &feeds.Author {
			Name: cfg.AuthorName,
			Email:cfg.AuthorEmail,
		},
		Created:     t_jst,
		Items:       items,
	}, nil
}
