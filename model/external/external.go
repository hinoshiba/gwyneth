package external

type SourceType struct {
	Id         string `json:"id"`
	Name       string `json:"name"`
	Cmd        string `json:"command"`
	UserCreate bool   `json:"user_create"`
}

type Source struct {
	Id    string      `json:"id"`
	Title string      `json:"title"`
	Type  *SourceType `json:"type"`
	Value string      `json:"value"`
	Pause bool        `json:"pause"`
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

type Action struct {
	Id    string `json:"id"`
	Name  string `json:"name"`
	Cmd   string `json:"command"`
}

type Filter struct {
	Id            string       `json:"id"`

	Title         *FilterValue `json:"title"`
	Body          *FilterValue `json:"body"`

	Action        *Action      `json:"action"`
}

type FilterValue struct {
	Value   string `json:"value"`
	IsRegex bool   `json:"regex"`
}
