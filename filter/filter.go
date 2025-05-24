package filter

import (
	"regexp"
	"strings"
)

import (
	"github.com/hinoshiba/gwyneth/model"
	"github.com/hinoshiba/gwyneth/model/external"
)

type Filter struct {
	id           *model.Id

	val_title      string
	is_regex_title bool

	val_body       string
	is_regex_body  bool

	action       *Action
}

func NewFilter(id *model.Id, val_title string, is_regex_title bool, val_body string, is_regex_body bool, action *Action) *Filter {
	return &Filter{
		id: id,

		val_title: val_title,
		is_regex_title: is_regex_title,

		val_body: val_body,
		is_regex_body: is_regex_body,

		action: action,
	}
}

func (self *Filter) Id() *model.Id {
	return self.id
}

func (self *Filter) ValTitle() string {
	return self.val_title
}

func (self *Filter) IsRegexTitle() bool {
	return self.is_regex_title
}

func (self *Filter) ValBody() string {
	return self.val_body
}

func (self *Filter) IsRegexBody() bool {
	return self.is_regex_body
}

func (self *Filter) Action() *Action {
	return self.action
}

func (self *Filter) IsMatch(artlc *model.Article) bool {
	if self.is_regex_title {
		match, _ := regexp.MatchString(self.val_title, artlc.Title())
		if match {
			return true
		}
	} else {
		if strings.Contains(artlc.Title(), self.val_title) {
			return true
		}
	}

	if self.is_regex_body {
		match, _ := regexp.MatchString(self.val_body, artlc.Body())
		if match {
			return true
		}
	} else {
		if strings.Contains(artlc.Body(), self.val_body) {
			return true
		}
	}

	return false
}

func (self *Filter) ConvertExternal() *external.Filter {
	return &external.Filter{
		Id: self.id.String(),
		Title: &external.FilterValue{
			Value: self.val_title,
			IsRegex: self.is_regex_title,
		},
		Body: &external.FilterValue{
			Value: self.val_body,
			IsRegex: self.is_regex_body,
		},
		Action: self.action.ConvertExternal(),
	}
}
