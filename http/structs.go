package http

import (
	"github.com/hinoshiba/gwyneth/structs"
)

type SourceType struct {
	Id         string `json:"id"`
	Name       string `json:"name"`
	Cmd        string `json:"command"`
	UserCreate bool   `json:"user_create"`
}

func convSourceType(st *structs.SourceType) *SourceType {
	return &SourceType{
		Id: st.Id().String(),
		Name: st.Name(),
		Cmd: st.Command(),
		UserCreate: st.IsUserCreate(),
	}
}

type Source struct {
	Id    string      `json:"id"`
	Title string      `json:"title"`
	Type  *SourceType `json:"type"`
	Value string      `json:"value"`
}

func convSource(src *structs.Source) *Source {
	return &Source {
		Id: src.Id().String(),
		Title: src.Title(),
		Type: convSourceType(src.Type()),
		Value: src.Value(),
	}
}

type Article struct {
	Id         string  `json:"id"`
	Src        *Source `json:"src"`
	Title      string  `json:"title"`
	Body       string  `json:"body"`
	Link       string  `json:"link"`
	Timestamp  int     `json:"timestamp"`
	Raw        string  `json:"raw"`
}

func convArticle(artcl *structs.Article) *Article {
	return &Article{
		Id: artcl.Id().String(),
		Src: convSource(artcl.Src()),
		Title: artcl.Title(),
		Body: artcl.Body(),
		Link: artcl.Link(),
		Timestamp: int(artcl.Unixtime()),
		Raw: artcl.Raw(),
	}
}

type Action struct {
	Id    string `json:"id"`
	Name  string `json:"name"`
	Cmd   string `json:"command"`
}

func convAction(action *structs.Action) *Action {
	return &Action{
		Id: action.Id().String(),
		Name: action.Name(),
		Cmd: action.Command(),
	}
}

type Filter struct {
	Id            string       `json:"id"`

	Title         *FilterValue `json:"title"`
	Body          *FilterValue `json:"body"`

	Action        *Action      `json:"action"`
}

type FilterValue struct {
	Value   string `json:"value"`
	IsRegex bool `json:"is_regex"`
}

func convFilter(f *structs.Filter) *Filter {
	return &Filter{
		Id: f.Id().String(),
		Title: &FilterValue{
			Value: f.ValTitle(),
			IsRegex: f.IsRegexTitle(),
		},
		Body: &FilterValue{
			Value: f.ValBody(),
			IsRegex: f.IsRegexBody(),
		},
		Action: convAction(f.Action()),
	}
}
