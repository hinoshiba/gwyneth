package errors

import (
	"fmt"
)

var (
	ERR_ALREADY_EXIST_ARTICLE = fmt.Errorf("the article is already exist.")
)
