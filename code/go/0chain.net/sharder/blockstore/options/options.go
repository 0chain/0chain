package options

import (
	"time"
)

type ColdFilterOptions struct {
	Prefix    string
	startDate time.Time
	endDate   time.Time
}
