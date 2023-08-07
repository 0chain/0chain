package chain

// Stats - a struct to store various runtime stats of the chain
type Stats struct {
	ZeroNotarizedBlocksCount  int64
	MultiNotarizedBlocksCount int64
	NotarizedBlocksCounts     []int64
	RoundTimeoutsCount        int64
}
