package common

import (
	"time"
)

/*Timestamp - just a wrapper to control the json encoding */
type Timestamp int64

/*Now - current datetime */
func Now() Timestamp {
	return Timestamp(time.Now().Unix())
}

/*Within ensures a given timestamp is within certain number of seconds */
func Within(ts int64, seconds int64) bool {
	now := time.Now().Unix()
	return now > ts-seconds && now < ts+seconds
}
