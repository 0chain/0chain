package blockstore

import (
	"time"

	"0chain.net/core/encryption"
)

func mockBlockWhereRecord() *BlockWhereRecord {
	return &BlockWhereRecord{
		Hash:      encryption.Hash(time.Now().String()),
		Tiering:   WarmTier,
		BlockPath: "block-path",
		CachePath: "cache-path",
		ColdPath:  "cold-path",
	}
}

func mockUnmovedBlockRecord() *UnmovedBlockRecord {
	return &UnmovedBlockRecord{
		CreatedAt: time.Now(),
		Hash:      encryption.Hash(time.Now().String()),
	}
}
