package round

/*Info - a struct that collects stats about the round */
type Info struct {
	Number                    int64 `json:"round_number"`
	NotarizedBlocksCount      int8  `json:"notarized_blocks_count"`
	ZeroNotarizedBlocksCount  int64 `json:"zero_notarized_blocks_count"`
	MultiNotarizedBlocksCount int64 `json:"multiple_notarized_blocks_count"`
}

func (info *Info) GetValue() int64 {
	return info.Number
}
