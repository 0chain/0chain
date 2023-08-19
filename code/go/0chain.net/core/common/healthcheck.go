package common

import (
	"time"

	"0chain.net/core/config"
)

// TODO: Move to a config file
const (
	healthCheckPeriod     = Timestamp(5 * time.Minute)
	healthCheckDelayLimit = Timestamp(30 * time.Second)
)

func Downtime(prevHealthCheck, curHealthCheck Timestamp) uint64 {
	config.GetMainChainID()
	if (curHealthCheck - prevHealthCheck) > (healthCheckPeriod + healthCheckDelayLimit) {
		return uint64(curHealthCheck - prevHealthCheck - healthCheckPeriod)
	}

	return 0
}
