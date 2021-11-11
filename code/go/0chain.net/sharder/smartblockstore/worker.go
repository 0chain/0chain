package smartblockstore

import (
	"context"
	"fmt"
	"sync"
	"time"

	. "0chain.net/core/logging"
)

func SetupWorkers(ctx context.Context)

/*
Todo assign correct index for selectDir function; restartVolumes, recoverMetadata
Todo Think more about caching
Todo
*/

func moveToColdTier(smartStore *SmartStore, ctx context.Context) {
	pollInterval := time.Hour * time.Duration(smartStore.ColdTier.pollInterval)
	t := time.NewTicker(pollInterval)
	errCh := make(chan error, 1)
	for {
		select {
		case <-ctx.Done():
			break
		case err := <-errCh:
			Logger.Error(fmt.Sprintf("Error occurred while moving objects to cold tier. Error: %v", err))
			break
		case <-t.C:
			Logger.Info("Moving blocks to cold tier")
			minPrefix := []byte(time.Now().Add(-pollInterval).Format(time.RFC3339))
			var newColdPath string

			guideChannel := make(chan struct{}, 10)
			wg := sync.WaitGroup{}

			var ubr *UnmovedBlockRecord
			var prevKey []byte
			var err error
			for ubr, prevKey, err = GetUnmovedBlock(prevKey, minPrefix); err == nil && ubr != nil; {
				guideChannel <- struct{}{}
				wg.Add(1)

				go func() {
					defer func() {
						<-guideChannel
						wg.Done()
					}()

					bwr, err := GetBlockWhereRecord(ubr.Hash)
					if err != nil {
						Logger.Error(fmt.Sprintf("Unexpected error; Error: %v", err))
						return
					}

					newColdPath, err = smartStore.ColdTier.moveBlock(bwr.Hash, bwr.BlockPath)
					if err != nil {
						Logger.Error(err.Error())
						return
					}

					Logger.Info(fmt.Sprintf("Block %v is moved to %v", bwr.Hash, newColdPath))
					switch bwr.Tiering {
					case HotTier:
						bwr.Tiering = newTiering(HotTier, HotTier, smartStore.ColdTier.deleteLocal)
					case WarmTier:
						bwr.Tiering = newTiering(WarmTier, WarmTier, smartStore.ColdTier.deleteLocal)
					case CacheAndWarmTier:
						bwr.Tiering = newTiering(CacheAndWarmTier, WarmTier, smartStore.ColdTier.deleteLocal)
					case CacheAndHotTier:
						bwr.Tiering = newTiering(CacheAndHotTier, HotTier, smartStore.ColdTier.deleteLocal)
					}

					if smartStore.ColdTier.deleteLocal {
						bwr.BlockPath = ""
					}
					bwr.ColdPath = newColdPath

					if err := bwr.AddOrUpdate(); err != nil {
						Logger.Error(fmt.Sprintf("Block %v is moved to %v but could not update meta record. Error: %v", bwr.Hash, newColdPath, err))
						errCh <- err
					}
				}()
			}

			wg.Wait()
		}
	}
}

func newTiering(previousTiering WhichTier, skipOrSubtractTier WhichTier, deleteLocal bool) (nt WhichTier) {
	nt = previousTiering + ColdTier
	if deleteLocal {
		nt -= skipOrSubtractTier
	}
	return
}
