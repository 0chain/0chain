package blockstore

import (
	"context"
	"time"

	"0chain.net/core/logging"
)

func setupVolumeRevivingWorker(ctx context.Context) {
	logging.Logger.Info("Setting volume reviving worker")
	mainStore := Store.(*blockStore)
	// interval := 2 * time.Minute
	t := time.NewTicker(mainStore.blockMovementInterval)

	var dTier *diskTier

	for {
		select {
		case <-ctx.Done():
			break
		case <-t.C:
			logging.Logger.Info("Checking if volume is able to store blocks")
			for vPath, volume := range unableVolumes {
				if volume.isAbleToStoreBlock(dTier) {
					dTier.Mu.Lock()
					dTier.Volumes = append(dTier.Volumes, volume)
					delete(unableVolumes, vPath)
					dTier.Mu.Unlock()
				}
			}
		}
	}
}

func setupColdWorker(ctx context.Context) {
	for {
		if true {
			break
		}
	}
}
