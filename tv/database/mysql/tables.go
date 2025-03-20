package mysql

func make_table_dict() ([]string, map[string]string) {
	d := make(map[string]string)
	order := []string{
		"source_type", "source",
		"action", "filter", "src_filter_map",
		"article", "feed",
	}

	d["source_type"] = TABLE_SOURCE_TYPE
	d["source"] = TABLE_SOURCE

	d["filter"] = TABLE_FILTER
	d["action"] = TABLE_ACTION
	d["src_filter_map"] = TABLE_SOURCE_FILTER_MAP

	d["article"] = TABLE_ARTICLE
	d["feed"] = TABLE_FEED

	return order, d
}

const TABLE_ACTION string = `
id BINARY(16) NOT NULL,
name VARCHAR(255) UNIQUE NOT NULL,
command TEXT,
PRIMARY KEY (id)
`
// 1 is true at boolean

const TABLE_FILTER string = `
id BINARY(16) NOT NULL,
val_title TEXT NOT NULL,
is_regex_title BOOLEAN NOT NULL DEFAULT 0,
val_body TEXT NOT NULL,
is_regex_body BOOLEAN NOT NULL DEFAULT 0,
action_id BINARY(16) NOT NULL,
PRIMARY KEY (id),
FOREIGN KEY (action_id) REFERENCES action(id)
`

const TABLE_SOURCE_FILTER_MAP string = `
filter_id BINARY(16) NOT NULL,
src_id BINARY(16) NOT NULL,
PRIMARY KEY (filter_id, src_id),
FOREIGN KEY (filter_id) REFERENCES filter(id),
FOREIGN KEY (src_id) REFERENCES source(id)
`

const TABLE_SOURCE_TYPE string = `
id BINARY(16) NOT NULL,
name VARCHAR(255) NOT NULL,
command TEXT,
user_create BOOLEAN NOT NULL DEFAULT 1,
PRIMARY KEY (id)
`
// 1 is true at boolean

const TABLE_SOURCE string = `
id BINARY(16) NOT NULL,
title VARCHAR(255) NOT NULL,
type BINARY(16) NOT NULL,
source TEXT NOT NULL,
pause BOOLEAN NOT NULL DEFAULT 0,
disable BOOLEAN NOT NULL DEFAULT 0,
PRIMARY KEY (id),
FOREIGN KEY (type) REFERENCES source_type(id)
`

const TABLE_ARTICLE string = `
id BINARY(16) NOT NULL,
src_id BINARY(16) NOT NULL,
title LONGTEXT NOT NULL,
body LONGTEXT NOT NULL,
link TEXT NOT NULL,
timestamp TIMESTAMP NOT NULL,
raw LONGTEXT NOT NULL,
disable BOOLEAN NOT NULL DEFAULT 0,
PRIMARY KEY (id),
FOREIGN KEY (src_id) REFERENCES source(id)
`
// 0 is false at boolean

const TABLE_FEED string = `
src_id BINARY(16) NOT NULL,
article_id BINARY(16) NOT NULL,
timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
disable BOOLEAN NOT NULL DEFAULT 0,
PRIMARY KEY (src_id, article_id),
FOREIGN KEY (src_id) REFERENCES source(id),
FOREIGN KEY (article_id) REFERENCES article(id)
`
