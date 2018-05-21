package common

import (
	"strings"
	"time"
)

/*Time - just a wrapper to control the json encoding */
type Time struct {
	time.Time
}

var timeFormat = time.RFC3339Nano

/*UnmarshalJSON - to control how the timestamp will be received */
func (t *Time) UnmarshalJSON(buf []byte) error {
	tt, err := time.Parse(timeFormat, strings.Trim(string(buf), `"`))
	if err != nil {
		return err
	}
	t.Time = tt.UTC()
	return nil
}

/*MarshalJSON - to control how the timestamp will be sent */
func (t *Time) MarshalJSON() ([]byte, error) {
	return []byte(`"` + t.Time.Format(timeFormat) + `"`), nil
}

/*Now - current datetime */
func Now() Time {
	now := Time{}
	now.Time = time.Now().UTC()
	return now
}
