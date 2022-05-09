package common

import (
	"context"
	"strconv"
	"time"
)

//DateTimeFormat - the format in which the date time fields should be displayed in the UI
var DateTimeFormat = "2006-01-02T15:04:05+00:00"

//go:generate msgp -io=false -tests=false -v
/*Timestamp - just a wrapper to control the json encoding */
type Timestamp int64

/*Now - current datetime */
func Now() Timestamp {
	return Timestamp(time.Now().Unix())
}

// Duration returns the Timestamp as time.Duration. Used where the Timestamp
// represents a duration.
func (t Timestamp) Duration() time.Duration {
	return time.Second * time.Duration(t)
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

// SleepOrDone sleeps given timeout and returns true. But, if given context
// expired, then it returns false immediately.
func SleepOrDone(ctx context.Context, sleep time.Duration) (done bool) {
	var tm = time.NewTimer(sleep)
	defer tm.Stop()
	select {
	case <-ctx.Done():
		done = true
	case <-tm.C:
	}
	return
}

// ToSeconds converts the time.Duration to Timestamp(time.Seconds)
func ToSeconds(duration time.Duration) Timestamp {
	return Timestamp(duration / time.Second)
}
