package consts

import "time"

const (
	VERSION = "v0.0.1"
)

var (
	TZ_JST *time.Location
)

func init() {
	TZ_JST = time.FixedZone("Asia/Tokyo", 9*60*60)
}
