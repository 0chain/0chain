package blockstore

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
