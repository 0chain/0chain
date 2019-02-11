package common

import (
	"strconv"
	"time"
)

//DateTimeFormat - the format in which the date time fields should be displayed in the UI
var DateTimeFormat = "2006-01-02T15:04:05+00:00"

/*Timestamp - just a wrapper to control the json encoding */
type Timestamp int64

/*Now - current datetime */
func Now() Timestamp {
	return Timestamp(time.Now().Unix())
}

//TimeToString - return the time stamp as a string
func TimeToString(ts Timestamp) string {
	return strconv.FormatInt(int64(ts), 10)
}

//ToTime - converts the common.Timestamp to time.Time
func ToTime(ts Timestamp) time.Time {
	return time.Unix(int64(ts), 0)
}

/*Within ensures a given timestamp is within (+/- inclusive) certain number of seconds w.r.t current time */
func Within(ts int64, seconds int64) bool {
	return WithinTime(time.Now().Unix(), ts, seconds)
}

/*WithinTime ensures a given timestamp is within (+/- inclusive) certain number of seconds w.r.t to the reference time */
func WithinTime(o int64, ts int64, seconds int64) bool {
	return ts >= o-seconds && ts <= o+seconds
}
