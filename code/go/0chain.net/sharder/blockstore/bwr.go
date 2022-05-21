package blockstore

import "time"

type WhichTier uint8

const (
	DiskTier        WhichTier = 2 // Block is in disk only
	ColdTier        WhichTier = 4 // Block is in cold storage only
	DiskAndColdTier WhichTier = 8 // Block is in both disk and cold storages
)

type blockWhereRecord struct {
	Hash    string    `json:"-"`
	Tiering WhichTier `json:"tr"`
	//For disk volume it is simple unix path. For cold storage it is "storageUrl:bucketName"
	BlockPath string `json:"vp,omitempty"`
	ColdPath  string `json:"cp,omitempty"`
}

func (bwr *blockWhereRecord) addOrUpdate() error {
	return nil
}

func getBWR(hash string) (*blockWhereRecord, error) {
	return nil, nil
}

/**********************************Unmoved block record***************************************/
type unmovedBlockRecord struct {
	Hash string `json:"h"`
	// CreateAt duration passed from epoch date
	CreatedAt time.Duration `json:"c"`
}

func (ubr *unmovedBlockRecord) Add() error {
	return nil
}

func (ubr *unmovedBlockRecord) Delete() error {
	return nil
}

// getUnmovedBlockRecords will return a channel where it will pass
// all the unmoved blocks
func getUnmovedBlockRecords() <-chan *unmovedBlockRecord {
	ch := make(chan *unmovedBlockRecord)

	go func() {
		defer close(ch)
	}()
	return ch
}
