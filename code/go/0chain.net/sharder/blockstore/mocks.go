package blockstore

import (
	"encoding/hex"
	"golang.org/x/crypto/sha3"
	"time"
)

func mockBWR() *BlockWhereRecord {
	bin, _ := time.Now().MarshalBinary()
	hash := sha3.Sum256(bin)
	return &BlockWhereRecord{
		Hash:      hex.EncodeToString(hash[:]),
		BlockPath: hex.EncodeToString(hash[:]),
		ColdPath:  hex.EncodeToString(hash[:]),
		Tiering:   0,
	}
}

func mockUBR() *UnmovedBlockRecord {
	now := time.Now()
	bin, _ := time.Now().MarshalBinary()
	hash := sha3.Sum256(bin)
	return &UnmovedBlockRecord{
		Hash:      hex.EncodeToString(hash[:]),
		CreatedAt: now,
	}
}
