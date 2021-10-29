//cache
package smartblockstore

import "0chain.net/chaincore/block"

type cache struct {
	blocksDir            string
	sizeToMaintain       uint64
	allowedBlocksNumbers uint64
	allowedBlockSize     uint64
	blocksCount          uint64
	blocksSize           uint64
	//This field will determine when to poll and clean cache's blocks.
	pollInterval int
}

func getFromCache() (*block.Block, error) {
	//
	return nil, nil
}

func writeToCache() (err error) {
	if err != nil {
		//log error
	}
	return
}

func deleteFromCache() {
	//
}

//Check for old blocks and clean cache
func pollCache() {
	//
}
