package common

import (
	"time"
)

// TODO: Move to a config file
const (
	healthCheckDelayLimit = Timestamp(30 * time.Second)
)

func Downtime(prevHealthCheck, curHealthCheck Timestamp, healthCheckPeriod time.Duration) uint64 {
	period := Timestamp(healthCheckPeriod)
	if (curHealthCheck - prevHealthCheck) > (period + healthCheckDelayLimit) {
		return uint64(curHealthCheck - prevHealthCheck - period)
	}

	return 0
}
