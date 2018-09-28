package chain

//Stats - a struct to store various runtime stats of the chain
type Stats struct {
	MissedBlocks              int64
	RollbackCount             int64
	LongestRollbackLength     int64
	ZeroNotarizedBlocksCount  int64
	MultiNotarizedBlocksCount int64
}
