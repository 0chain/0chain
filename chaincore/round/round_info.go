package round

import "time"

/*Info - a struct that collects stats about the round */
type Info struct {
	TimeStamp                 *time.Time `json:"ts"`
	Number                    int64      `json:"round_number"`
	NotarizedBlocksCount      int8       `json:"notarized_blocks_count"`
	ZeroNotarizedBlocksCount  int64      `json:"zero_notarized_blocks_count"`
	MultiNotarizedBlocksCount int64      `json:"multiple_notarized_blocks_count"`
	MissedBlocks              int64      `json:"missed_blocks_count"`
	RollbackCount             int64      `json:"rollback_count"`
	LongestRollbackLength     int8       `json:"longest_rollback_length"`
}

//GetKey - implements Metric Interface
func (info *Info) GetKey() int64 {
	return info.Number
}

//GetTime - implements Metric Interface
func (info *Info) GetTime() *time.Time {
	return info.TimeStamp
}
