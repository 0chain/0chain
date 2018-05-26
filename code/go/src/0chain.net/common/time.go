package common

import (
	"time"
)

/*Timestamp - just a wrapper to control the json encoding */
type Timestamp int64

/*Now - current datetime */
func Now() Timestamp {
	return Timestamp(time.Now().UTC().Unix())
}
