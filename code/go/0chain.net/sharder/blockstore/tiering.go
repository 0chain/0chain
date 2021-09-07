//All the tiering works will go here

package blockstore

import (
	"io/fs"
	"path/filepath"
	"time"
)

var TieringInterval = time.Hour * 720 //take from config

func MoveToColdTier() {

}

func MoveToBlobber() {

}

func TierWorker() {
	uBlockCh := make(chan *ColdBlock) //Unmoved block channel
	done := make(chan struct{})
	for {
		ticker := time.NewTicker(time.Hour * 720)
		select {
		case <-ticker.C:
			prefix := ""
			for {
				err := GetUnmovedBlocks(uBlockCh, prefix)
				if err != nil {
					break
				}
				coldBlock := <-uBlockCh
				if coldBlock.CreatedAt.Sub(time.Now()) > TieringInterval {
					MoveToColdTier()
					MoveToBlobber()
					RemoveMovedBlocks(coldBlock.HashKey)
				}

				prefix = coldBlock.HashKey
			}
		case <-done:
			//Sharder is exiting
		}
	}
}

func ManageHotBlocks() {
	//May be have blocks count and total blocks size to be in hot tier to be in config
	//check SSD capacity
	rootPath := "/data/blocks"
	filepath.Walk(rootPath, func(path string, info fs.FileInfo, err error) error {
		if !info.IsDir() {
			//check created date
			//check access count in bmr

		}
		return nil
	})
}

/*
there is round number for each block

*/
