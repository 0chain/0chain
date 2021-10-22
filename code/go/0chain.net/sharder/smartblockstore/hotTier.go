package blockstore

var hotTier hTier

type hTier struct { //Hot Tier
	hVolumes []hotVolumes //List of hot volumes
}

type hotVolumes struct {
	path                    string
	allowedBlockNumbers     uint64
	allowedBlockSize        uint64
	sizeToMaintain          uint64
	blocksSize, blocksCount uint64
	availableSize           uint64
	curKInd                 uint32
	curDirInd               uint32
	curDirBlockNums         uint32
}
