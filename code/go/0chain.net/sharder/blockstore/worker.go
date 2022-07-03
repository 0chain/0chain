package blockstore

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"0chain.net/core/logging"
)

func setupVolumeRevivingWorker(ctx context.Context) {
	logging.Logger.Info("Setting volume reviving worker")
	t := time.NewTicker(time.Hour)

	dTier := GetStore().(*blockStore).diskTier

	for {
		select {
		case <-ctx.Done():
			break
		case <-t.C:
			logging.Logger.Info("Checking if volume is able to store blocks")
			for vPath, volume := range unableVolumes {
				if volume.isAbleToStoreBlock() {
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
	store := Store.(*blockStore)
	ticker := time.NewTicker(store.blockMovementInterval)

	for {
		select {
		case <-ctx.Done():
			break
		case <-ticker.C:
			upto := time.Now().Add(-store.blockMovementInterval).Unix()
			maxPrefix := strconv.FormatInt(upto, 10)
			ch := getUnmovedBlockRecords([]byte(maxPrefix))
			guideCh := make(chan struct{}, 10)
			wg := &sync.WaitGroup{}
			for ubr := range ch {
				guideCh <- struct{}{}
				wg.Add(1)

				go func(ubr *unmovedBlockRecord) {
					defer func() {
						<-guideCh
						wg.Done()
					}()

					bwr, err := getBWR(ubr.Hash)
					if err != nil {
						logging.Logger.Error(fmt.Sprintf("Unexpected error; Error: %v", err))
						return
					}
					logging.Logger.Info("Moving block " + bwr.Hash)
					newColdPath, err := store.coldTier.moveBlock(bwr.Hash, bwr.BlockPath)
					if err != nil {
						logging.Logger.Error(err.Error())
						return
					}
					logging.Logger.Info(fmt.Sprintf("Block %v is moved to %v", bwr.Hash, newColdPath))

					bwr.Tiering = newTiering(store.coldTier.DeleteLocal)
					if store.coldTier.DeleteLocal {
						bwr.BlockPath = ""
					}
					bwr.ColdPath = newColdPath

					if err := ubr.Delete(); err != nil {
						logging.Logger.Error(fmt.Sprintf("Block %v is moved to %v but could not delete meta record from unmoved block bucket. Error: %v", bwr.Hash, newColdPath, err))
					}

					if err := bwr.addOrUpdate(); err != nil {
						logging.Logger.Error(fmt.Sprintf("Block %v is moved to %v but could not update meta record. Error: %v", bwr.Hash, newColdPath, err))
					}

					logging.Logger.Info(fmt.Sprintf("Block meta data for bmr for block %v is updated successfully", bwr.Hash))

				}(ubr)
			}

		}
	}
}

func newTiering(deleteLocal bool) (nt WhichTier) {
	nt = DiskAndColdTier
	if deleteLocal {
		nt = ColdTier
	}
	return
}
