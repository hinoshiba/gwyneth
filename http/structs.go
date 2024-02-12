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
