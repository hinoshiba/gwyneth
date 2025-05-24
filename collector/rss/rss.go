package rss

import (
	"time"
	"encoding/json"
)

import (
	"github.com/l4go/task"
	"github.com/mmcdole/gofeed"
)

import (
	"github.com/hinoshiba/gwyneth/slog"
	"github.com/hinoshiba/gwyneth/structs"
)

func GetFeed(msn *task.Mission, src *structs.Source, artcl_ch chan <- *structs.Article) error {
	defer msn.Done()

	fp := gofeed.NewParser()

	url := src.Value()
	feed, err := fp.ParseURLWithContext(url, msn.AsContext())
	if err != nil {
		return err
	}

	now := time.Now()
	for _, item := range feed.Items {
		var pubdate time.Time = now
		if item.UpdatedParsed != nil {
			pubdate = *item.UpdatedParsed
		}
		if item.UpdatedParsed == nil && item.PublishedParsed != nil {
			pubdate = *item.PublishedParsed
		}

		raw_j, err := json.Marshal(item)
		if err != nil {
			slog.Warn("convert errror: cannot convert to json str from item struct. : '%s', '%s'", item.Title, url)
			continue
		}

		artcl := structs.NewArticle(nil, src, item.Title, item.Description, item.Link, pubdate.Unix(), string(raw_j))

		go func(msn *task.Mission, artcl *structs.Article) {
			defer msn.Done()

			select {
			case <- msn.RecvCancel():
			case artcl_ch <- artcl:
			}
		}(msn.New(), artcl)
	}
	return nil
}
