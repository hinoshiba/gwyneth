package mysql

func make_table_dict() ([]string, map[string]string) {
	d := make(map[string]string)
	order := []string{
		"source_type", "source", "article", "feed", "noticer",
		"filter_action", "filter", "filter_entry",
	}

	d["source_type"] = TABLE_SOURCE_TYPE
	d["source"] = TABLE_SOURCE
	d["article"] = TABLE_ARTICLE
	d["feed"] = TABLE_FEED

	d["noticer"] = TABLE_NOTICER

	d["filter"] = TABLE_FILTER
	d["filter_action"] = TABLE_FILTER_ACTION
	d["filter_entry"] = TABLE_FILTER_ENTRY

	return order, d
}

const TABLE_FILTER_ACTION string = `
id BINARY(16) NOT NULL,
name TEXT NOT NULL,
command TEXT,
user_create BOOLEAN NOT NULL DEFAULT 1,
PRIMARY KEY (id)
`
// 1 is true at boolean

const TABLE_FILTER string = `
id BINARY(16) NOT NULL,
title_filter TEXT NOT NULL,
body_filter TEXT NOT NULL,
action_id BINARY(16) NOT NULL,
PRIMARY KEY (id),
FOREIGN KEY (action_id) REFERENCES filter_action(id)
`

const TABLE_FILTER_ENTRY string = `
filter_id BINARY(16) NOT NULL,
action_id BINARY(16) NOT NULL,
dst_id BINARY(16) NOT NULL,
PRIMARY KEY (filter_id, action_id, dst_id),
FOREIGN KEY (filter_id) REFERENCES filter(id),
FOREIGN KEY (action_id) REFERENCES filter_action(id),
FOREIGN KEY (dst_id) REFERENCES source(id)
`

const TABLE_NOTICER string = `
id BINARY(16) NOT NULL,
filter TEXT NOT NULL,
dst_id BINARY(16) NOT NULL,
PRIMARY KEY (id),
FOREIGN KEY (dst_id) REFERENCES source(id)
`

const TABLE_SOURCE_TYPE string = `
id BINARY(16) NOT NULL,
name TEXT NOT NULL,
command TEXT,
user_create BOOLEAN NOT NULL DEFAULT 1,
PRIMARY KEY (id)
`
// 1 is true at boolean

const TABLE_SOURCE string = `
id BINARY(16) NOT NULL,
title TEXT NOT NULL,
type BINARY(16) NOT NULL,
source TEXT NOT NULL,
PRIMARY KEY (id),
FOREIGN KEY (type) REFERENCES source_type(id)
`

const TABLE_ARTICLE string = `
id BINARY(16) NOT NULL,
src_id BINARY(16) NOT NULL,
title LONGTEXT NOT NULL,
body LONGTEXT NOT NULL,
link TEXT NOT NULL,
timestap TIMESTAMP NOT NULL,
raw LONGTEXT NOT NULL,
disable BOOLEAN NOT NULL DEFAULT 0,
PRIMARY KEY (id),
FOREIGN KEY (src_id) REFERENCES source(id)
`
// 0 is false at boolean

const TABLE_FEED string = `
src_id BINARY(16) NOT NULL,
article_id BINARY(16) NOT NULL,
timestap TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
disable BOOLEAN NOT NULL DEFAULT 0,
PRIMARY KEY (src_id, article_id),
FOREIGN KEY (src_id) REFERENCES source(id),
FOREIGN KEY (article_id) REFERENCES article(id)
`
