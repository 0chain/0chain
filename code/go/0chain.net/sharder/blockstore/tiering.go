//All the tiering works will go here

package blockstore

import "time"

func MoveToColdTier() {

}

func MoveToBlobber() {

}

func TierWorker() {
	done := make(chan struct{})
	for {
		ticker := time.NewTicker(time.Hour * 720)
		select {
		case <-ticker.C:
			//
		case <-done:
		}
	}
}
