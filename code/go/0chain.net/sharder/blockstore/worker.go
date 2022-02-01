package blockstore

import (
	"context"
	"fmt"
	"sync"
	"time"

	. "0chain.net/core/logging"
)

func setupColdWorker(ctx context.Context) {
	Logger.Info("Setting up cold worker")
	pollInterval := time.Hour * time.Duration(Store.ColdTier.PollInterval)
	// pollInterval := time.Minute
	t := time.NewTicker(pollInterval)
	errCh := make(chan error, 10)

	for {
		select {
		case <-ctx.Done():
			break
		case err := <-errCh:
			Logger.Error(fmt.Sprintf("Error occurred while moving objects to cold tier. Error: %v", err))
			break
		case <-t.C:
			Logger.Info("Moving blocks to cold tier")
			maxPrefix := time.Now().Add(-pollInterval).UnixMicro()
			var newColdPath string

			guideChannel := make(chan struct{}, 10)
			wg := sync.WaitGroup{}

			var errorOccurred bool
			for ubrs := GetUnmovedBlocks(maxPrefix, 1000); ubrs != nil; ubrs = GetUnmovedBlocks(maxPrefix, 1000) {
				for _, ubr := range ubrs {
					if errorOccurred {
						break
					}

					guideChannel <- struct{}{}
					wg.Add(1)

					Logger.Info(fmt.Sprintf("Moving block %v to cold tier", ubr.Hash))
					go func(ubr *UnmovedBlockRecord) {
						defer func() {
							<-guideChannel
							wg.Done()
						}()

						bwr, err := GetBlockWhereRecord(ubr.Hash)
						if err != nil {
							errorOccurred = true
							Logger.Error(fmt.Sprintf("Unexpected error; Error: %v", err))
							errCh <- err
							return
						}

						newColdPath, err = Store.ColdTier.moveBlock(bwr.Hash, bwr.BlockPath)
						if err != nil {
							errorOccurred = true
							Logger.Error(err.Error())
							errCh <- err
							return
						}

						Logger.Info(fmt.Sprintf("Block %v is moved to %v", bwr.Hash, newColdPath))
						switch bwr.Tiering {
						case HotTier:
							bwr.Tiering = newTiering(HotTier, HotTier, Store.ColdTier.DeleteLocal)
						case WarmTier:
							bwr.Tiering = newTiering(WarmTier, WarmTier, Store.ColdTier.DeleteLocal)
						}

						if Store.ColdTier.DeleteLocal {
							bwr.BlockPath = ""
						}
						bwr.ColdPath = newColdPath

						if err := ubr.Delete(); err != nil {
							errorOccurred = true
							Logger.Error(fmt.Sprintf("Block %v is moved to %v but could not delete meta record from unmoved block bucket. Error: %v", bwr.Hash, newColdPath, err))
							errCh <- err
							return
						}
						Logger.Info(fmt.Sprintf("Block meta data for %v from unmoved bucket is removed", ubr.Hash))

						if err := bwr.AddOrUpdate(); err != nil {
							errorOccurred = true
							Logger.Error(fmt.Sprintf("Block %v is moved to %v but could not update meta record. Error: %v", bwr.Hash, newColdPath, err))
							errCh <- err
						}
						Logger.Info(fmt.Sprintf("Block meta data for bmr for block %v is updated successfully", bwr.Hash))
					}(ubr)
				}

				wg.Wait()
			}
		}
	}
}

func setupVolumeRevivingWorker(ctx context.Context) {
	Logger.Info("Setting volume reviving worker")
	interval := 2 * time.Hour * time.Duration(Store.ColdTier.PollInterval)
	// interval := 2 * time.Minute
	t := time.NewTicker(interval)

	var dTier *diskTier

	if Store.HotTier != nil {
		dTier = Store.HotTier
	} else if Store.WarmTier != nil {
		dTier = Store.WarmTier
	} else {
		Logger.Debug("Volume reviving worker not set up")
		return
	}
	for {
		select {
		case <-ctx.Done():
			break
		case <-t.C:
			Logger.Info("Checking if volume is able to store blocks")
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

func setupCacheReplacement(ctx context.Context, cacheI cacher) {
	Logger.Info("Setting cache replacement worker")
	cache := cacheI.(*diskCache)
	t := time.NewTicker(cache.ReplaceInterval)
	// t := time.NewTicker(time.Minute * 2)
	for {
		select {
		case <-ctx.Done():
			break
		case <-t.C:
			Logger.Info("Replacing old cache")
			cache.Replace()
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
