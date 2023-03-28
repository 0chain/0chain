package cases

import (
	"sync"

	"github.com/mitchellh/mapstructure"

	"0chain.net/conductor/cases"
	"0chain.net/conductor/conductrpc/stats"
)

type (
	// BlockStateChangeRequestor represents TestCaseConfigurator implementation.
	BlockStateChangeRequestor struct {
		TestReport `json:"test_report" yaml:"test_report" mapstructure:"test_report"`

		// IgnoringRequestsBy contains nodes which must ignore Replica0.
		IgnoringRequestsBy Nodes `json:"ignoring_requests_by" yaml:"ignoring_requests_by" mapstructure:"ignoring_requests_by"`

		// CorrectResponseBy contains nodes which must response correctly to Replica0.
		CorrectResponseBy Nodes `json:"correct_response_by" yaml:"correct_response_by" mapstructure:"correct_response_by"`

		ChangedMPTNodeBy Nodes `json:"changed_mpt_node_by" yaml:"changed_mpt_node_by" mapstructure:"changed_mpt_node_by"`

		DeletedMPTNodeBy Nodes `json:"deleted_mpt_node_by" yaml:"deleted_mpt_node_by" mapstructure:"deleted_mpt_node_by"`

		AddedMPTNodeBy Nodes `json:"added_mpt_node_by" yaml:"added_mpt_node_by" mapstructure:"added_mpt_node_by"`

		PartialStateFromAnotherBlockBy Nodes `json:"partial_state_from_another_block_by" yaml:"partial_state_from_another_block_by" mapstructure:"partial_state_from_another_block_by"`

		Configured bool

		Ignored int

		Resulted bool

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
	return cases.NewBlockStateChangeRequestor(n.statsCollector, n.GetType())
}

// Name implements TestCaseConfigurator interface.
func (n *BlockStateChangeRequestor) Name() string {
	postfix := ""
	switch n.GetType() {
	case cases.BSCRNoReplies:
		postfix = "neither node reply"

	case cases.BSCROnlyOneRepliesCorrectly:
		postfix = "only one node replies correctly"

	case cases.BSCRChangeNode:
		postfix = "one node sends state change with changed mpt node"

	case cases.BSCRDeleteNode:
		postfix = "one node sends state change with deleted mpt node"

	case cases.BSCRAddNode:
		postfix = "one node sends state change with added mpt node"

	case cases.BSCRAnotherPartialState:
		postfix = "one node sends partial state from another block"

	default:
		postfix = "unknown"
	}
	return BlockStateChangeRequestorName + ": " + postfix
}

// Decode implements MapDecoder interface.
func (n *BlockStateChangeRequestor) Decode(val interface{}) error {
	return mapstructure.Decode(val, n)
}

func (n *BlockStateChangeRequestor) GetType() cases.BlockStateChangeRequestorCaseType {
	switch {
	case !n.IgnoringRequestsBy.IsEmpty() &&
		n.CorrectResponseBy.IsEmpty() && n.ChangedMPTNodeBy.IsEmpty() && n.DeletedMPTNodeBy.IsEmpty() &&
		n.PartialStateFromAnotherBlockBy.IsEmpty() && n.AddedMPTNodeBy.IsEmpty():

		return cases.BSCRNoReplies

	case !n.IgnoringRequestsBy.IsEmpty() && n.CorrectResponseBy.Num() == 1 &&
		n.ChangedMPTNodeBy.IsEmpty() && n.DeletedMPTNodeBy.IsEmpty() && n.AddedMPTNodeBy.IsEmpty() && n.PartialStateFromAnotherBlockBy.IsEmpty():

		return cases.BSCROnlyOneRepliesCorrectly

	case n.ChangedMPTNodeBy.Num() == 1 && !n.IgnoringRequestsBy.IsEmpty() &&
		n.CorrectResponseBy.IsEmpty() && n.DeletedMPTNodeBy.IsEmpty() && n.AddedMPTNodeBy.IsEmpty() && n.PartialStateFromAnotherBlockBy.IsEmpty():

		return cases.BSCRChangeNode

	case n.DeletedMPTNodeBy.Num() == 1 && !n.IgnoringRequestsBy.IsEmpty() &&
		n.CorrectResponseBy.IsEmpty() && n.ChangedMPTNodeBy.IsEmpty() && n.AddedMPTNodeBy.IsEmpty() && n.PartialStateFromAnotherBlockBy.IsEmpty():

		return cases.BSCRDeleteNode

	case n.AddedMPTNodeBy.Num() == 1 && !n.IgnoringRequestsBy.IsEmpty() &&
		n.CorrectResponseBy.IsEmpty() && n.ChangedMPTNodeBy.IsEmpty() && n.DeletedMPTNodeBy.IsEmpty() && n.PartialStateFromAnotherBlockBy.IsEmpty():

		return cases.BSCRAddNode

	case n.PartialStateFromAnotherBlockBy.Num() == 1 && !n.IgnoringRequestsBy.IsEmpty() &&
		n.CorrectResponseBy.IsEmpty() && n.ChangedMPTNodeBy.IsEmpty() && n.DeletedMPTNodeBy.IsEmpty() && n.AddedMPTNodeBy.IsEmpty():

		return cases.BSCRAnotherPartialState

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
