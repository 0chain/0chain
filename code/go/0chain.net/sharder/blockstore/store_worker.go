package blockstore

import (
	"time"

	. "0chain.net/core/logging"
)

func cacheTiering() {
	for {
		ticker := time.NewTicker(time.Second * 60)
		select {
		case <-ticker.C:
			Logger.Info("Tiering blocks to warm/cold tier")
		}
	}
}
