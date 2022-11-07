package cases

import (
	"sync"

	"github.com/mitchellh/mapstructure"

	"0chain.net/conductor/cases"
	"0chain.net/conductor/conductrpc/stats"
)

type (
	// MinerNotarisedBlockRequestor represents TestCaseConfigurator implementation.
	MinerNotarisedBlockRequestor struct {
		TestReport `json:"test_report" yaml:"test_report" mapstructure:"test_report"`

		// IgnoringRequestsBy contains nodes which must ignore Replica0.
		IgnoringRequestsBy Nodes `json:"ignoring_requests_by" yaml:"ignoring_requests_by" mapstructure:"ignoring_requests_by"`

		// CorrectResponseBy contains nodes which must response correctly to Replica0.
		CorrectResponseBy Nodes `json:"correct_response_by" yaml:"correct_response_by" mapstructure:"correct_response_by"`

		// ValidBlockWithChangedHashBy contains nodes which must response Replica0 with valid block but changed hash.
		ValidBlockWithChangedHashBy Nodes `json:"valid_block_with_changed_hash_by" yaml:"valid_block_with_changed_hash_by" mapstructure:"valid_block_with_changed_hash_by"`

		// InvalidBlockWithChangedHashBy contains nodes which must response Replica0 with invalid block with changed hash.
		InvalidBlockWithChangedHashBy Nodes `json:"invalid_block_with_changed_hash_by" yaml:"invalid_block_with_changed_hash_by" mapstructure:"invalid_block_with_changed_hash_by"`

		// BlockWithoutVerTicketsBy contains nodes which must response Replica0 with block without verification tickets.
		BlockWithoutVerTicketsBy Nodes `json:"block_without_ver_tickets_by" yaml:"block_without_ver_tickets_by" mapstructure:"block_without_ver_tickets_by"`

		// BlockWithoutVerTicketsBy contains nodes which must response Replica0 with block without verification tickets.
		BlockWithInvalidTicketsBy Nodes `json:"block_with_invalid_tickets_by" yaml:"block_with_invalid_tickets_by" mapstructure:"block_with_invalid_tickets_by"`

		// BlockWithoutVerTicketsBy contains nodes which must response Replica0 with block without verification tickets.
		BlockWithValidTicketsForOldRoundBy Nodes `json:"block_with_valid_tickets_for_old_round_by" yaml:"block_with_valid_tickets_for_old_round_by" mapstructure:"block_with_valid_tickets_for_old_round_by"`

		Configured bool

		Ignored int

		statsCollector *stats.NodesClientStats

		mu sync.Mutex
	}
)

const (
	// MinerNotarisedBlockRequestorName represents base part of the MinerNotarisedBlockRequestor name.
	MinerNotarisedBlockRequestorName = "attack MinerNotarisedBlockRequestor"
)

var (
	// Ensure BlockStateChangeRequestor implements TestCaseConfigurator.
	_ TestCaseConfigurator = (*MinerNotarisedBlockRequestor)(nil)
)

// NewMinerNotarisedBlockRequestor creates initialised BlockStateChangeRequestor.
func NewMinerNotarisedBlockRequestor(statsCollector *stats.NodesClientStats) *MinerNotarisedBlockRequestor {
	return &MinerNotarisedBlockRequestor{
		statsCollector: statsCollector,
	}
}

// TestCase implements TestCaseConfigurator interface.
func (n *MinerNotarisedBlockRequestor) TestCase() cases.TestCase {
	return cases.NewMinerNotarisedBlockRequestor(n.statsCollector, n.getType())
}

// Name implements TestCaseConfigurator interface.
func (n *MinerNotarisedBlockRequestor) Name() string {
	postfix := ""
	switch n.getType() {
	case cases.MSBRNoReplies:
		postfix = "neither node reply"

	case cases.MSBROnlyOneRepliesCorrectly:
		postfix = "only one node replies"

	case cases.MSBRValidBlockWithChangedHash:
		postfix = "only one node sends valid block (with changed hash)"

	case cases.MSBRInvalidBlockWithChangedHash:
		postfix = "only one node sends invalid block (with changed hash)"

	case cases.MSBRBlockWithoutVerTickets:
		postfix = "only one node sends block without verification tickets"

	default:
		postfix = "unknown"
	}
	return MinerNotarisedBlockRequestorName + ": " + postfix
}

// Decode implements MapDecoder interface.
func (n *MinerNotarisedBlockRequestor) Decode(val interface{}) error {
	return mapstructure.Decode(val, n)
}

func (n *MinerNotarisedBlockRequestor) getType() cases.MinerNotarisedBlockRequestorType {
	switch {
	case !n.IgnoringRequestsBy.IsEmpty() &&
		n.CorrectResponseBy.IsEmpty() && n.ValidBlockWithChangedHashBy.IsEmpty() && n.InvalidBlockWithChangedHashBy.IsEmpty() && n.BlockWithoutVerTicketsBy.IsEmpty():

		return cases.MSBRNoReplies

	case !n.IgnoringRequestsBy.IsEmpty() && n.CorrectResponseBy.Num() == 1 &&
		n.ValidBlockWithChangedHashBy.IsEmpty() && n.InvalidBlockWithChangedHashBy.IsEmpty() && n.BlockWithoutVerTicketsBy.IsEmpty():

		return cases.MSBROnlyOneRepliesCorrectly

	case !n.IgnoringRequestsBy.IsEmpty() && n.ValidBlockWithChangedHashBy.Num() == 1 &&
		n.CorrectResponseBy.IsEmpty() && n.InvalidBlockWithChangedHashBy.IsEmpty() && n.BlockWithoutVerTicketsBy.IsEmpty():

		return cases.MSBRValidBlockWithChangedHash

	case !n.IgnoringRequestsBy.IsEmpty() && n.InvalidBlockWithChangedHashBy.Num() == 1 &&
		n.CorrectResponseBy.IsEmpty() && n.ValidBlockWithChangedHashBy.IsEmpty() && n.BlockWithoutVerTicketsBy.IsEmpty():

		return cases.MSBRInvalidBlockWithChangedHash

	case !n.IgnoringRequestsBy.IsEmpty() && n.BlockWithoutVerTicketsBy.Num() == 1 &&
		n.CorrectResponseBy.IsEmpty() && n.ValidBlockWithChangedHashBy.IsEmpty() && n.InvalidBlockWithChangedHashBy.IsEmpty():

		return cases.MSBRBlockWithoutVerTickets

	default:
		return -1
	}
}

func (n *MinerNotarisedBlockRequestor) Lock() {
	if n == nil {
		return
	}
	n.mu.Lock()
}

func (n *MinerNotarisedBlockRequestor) Unlock() {
	if n == nil {
		return
	}
	n.mu.Unlock()
}
