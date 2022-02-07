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

		// IgnoringRequestsBy contains nodes which must ignore Replica0.
		IgnoringRequestsBy Nodes `json:"ignoring_requests_by" yaml:"ignoring_requests_by" mapstructure:"ignoring_requests_by"`

		// CorrectResponseBy contains nodes which must response correctly to Replica0.
		CorrectResponseBy Nodes `json:"correct_response_by" yaml:"correct_response_by" mapstructure:"correct_response_by"`

		Configured bool

		Ignored int

		statsCollector *stats.NodesClientStats

		mu sync.Mutex
	}
)

const (
	BlockStateChangeRequestorName = "attack BlockStateChangeRequestor"
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
	return cases.NewBlockStateChangeRequestor(n.statsCollector, n.getType())
}

// Name implements TestCaseConfigurator interface.
func (n *BlockStateChangeRequestor) Name() string {
	postfix := ""
	switch n.getType() {
	case cases.BSCRNoReplies:
		postfix = "neither node reply"

	case cases.BSCROnlyOneRepliesCorrectly:
		postfix = "only one node replies correctly"

	default:
		postfix = "unknown"
	}
	return BlockStateChangeRequestorName + ": " + postfix
}

// Decode implements MapDecoder interface.
func (n *BlockStateChangeRequestor) Decode(val interface{}) error {
	return mapstructure.Decode(val, n)
}

func (n *BlockStateChangeRequestor) getType() cases.BlockStateChangeRequestorCaseType {
	switch {
	case n.IgnoringRequestsBy.Num() > 1 && n.CorrectResponseBy.Num() == 0:
		return cases.BSCRNoReplies

	case n.IgnoringRequestsBy.Num() > 0 && n.CorrectResponseBy.Num() == 1:
		return cases.BSCROnlyOneRepliesCorrectly

	default:
		return -1
	}
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
