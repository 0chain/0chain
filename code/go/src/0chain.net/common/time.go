package common

import (
	"math/rand"
	"strconv"
	"time"

	"0chain.net/config"
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

/*Within ensures a given timestamp is within (+/- inclusive) certain number of seconds w.r.t current time */
func Within(ts int64, seconds int64) bool {
	return WithinTime(time.Now().Unix(), ts, seconds)
}

/*WithinTime ensures a given timestamp is within (+/- inclusive) certain number of seconds w.r.t to the reference time */
func WithinTime(o int64, ts int64, seconds int64) bool {
	return ts >= o-seconds && ts <= o+seconds
}

var randGenerator = rand.New(rand.NewSource(time.Now().UnixNano()))

/*InduceDelay - induces some random delay - useful to test resilience */
func InduceDelay() int {
	if config.Development() && config.MaxDelay() > 0 {
		r := randGenerator.Intn(config.MaxDelay())
		if r < 500 {
			time.Sleep(time.Duration(r) * time.Millisecond)
			return r
		}
	}
	return 0
}
