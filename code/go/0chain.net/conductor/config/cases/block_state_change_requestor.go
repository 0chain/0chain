package cases

import (
	"sync"

	"0chain.net/conductor/cases"
	"0chain.net/conductor/conductrpc/stats"
	"github.com/mitchellh/mapstructure"
)

type (
	// BlockStateChangeRequestor represents TestCaseConfigurator implementation.
	BlockStateChangeRequestor struct {
		TestReport `json:"test_report" yaml:"test_report" mapstructure:"test_report"`

		Configured bool

		Ignored int

		statsCollector *stats.NodesClientStats

		mu sync.Mutex
	}
)

const (
	BlockStateChangeRequestorName = "attack BlockStateChangeRequestor: neither node reply"
)

var (
	// Ensure BlockStateChangeRequestor implements TestCaseConfigurator.
	_ TestCaseConfigurator = (*BlockStateChangeRequestor)(nil)
)

// NewBlockStateChangeRequestor creates initialised BlockStateChangeRequestor.
func NewBlockStateChangeRequestor(statsCollector *stats.NodesClientStats) *BlockStateChangeRequestor {
	return &BlockStateChangeRequestor{
		statsCollector: statsCollector,
	}
}

// TestCase implements TestCaseConfigurator interface.
func (n *BlockStateChangeRequestor) TestCase() cases.TestCase {
	return cases.NewBlockStateChangeRequestor(n.statsCollector)
}

// Name implements TestCaseConfigurator interface.
func (n *BlockStateChangeRequestor) Name() string {
	return BlockStateChangeRequestorName
}

// Decode implements MapDecoder interface.
func (n *BlockStateChangeRequestor) Decode(val interface{}) error {
	return mapstructure.Decode(val, n)
}

func (n *BlockStateChangeRequestor) Lock() {
	if n == nil {
		return
	}
	n.mu.Lock()
}

func (n *BlockStateChangeRequestor) Unlock() {
	if n == nil {
		return
	}
	n.mu.Unlock()
}
