package stats

type (
	BlockRequestor int
)

const (
	BRMinerNotarisedBlock BlockRequestor = iota
	BRBlockStateChange
	BRFB
)
