package cases

import (
	"sync"

	"github.com/mitchellh/mapstructure"

	"0chain.net/conductor/cases"
	"0chain.net/conductor/conductrpc/stats"
)

type (
	// FBRequestor represents TestCaseConfigurator implementation.
	FBRequestor struct {
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

		Configured bool

		Ignored int

		statsCollector *stats.NodesClientStats

		mu sync.Mutex
	}
)

const (
	// FBRequestorName represents base part of the FBRequestor name.
	FBRequestorName = "attack FBRequestor"
)

var (
	// Ensure FBRequestor implements TestCaseConfigurator.
	_ TestCaseConfigurator = (*FBRequestor)(nil)
)

// NewFBRequestor creates initialised FBRequestor.
func NewFBRequestor(statsCollector *stats.NodesClientStats) *FBRequestor {
	return &FBRequestor{
		statsCollector: statsCollector,
	}
}

// TestCase implements TestCaseConfigurator interface.
func (n *FBRequestor) TestCase() cases.TestCase {
	return cases.NewFBRequestor(n.statsCollector, n.getType())
}

// Name implements TestCaseConfigurator interface.
func (n *FBRequestor) Name() string {
	postfix := ""
	switch n.getType() {
	case cases.FBRNoReplies:
		postfix = "neither node reply"

	case cases.FBROnlyOneRepliesCorrectly:
		postfix = "only one node replies"

	case cases.FBRValidBlockWithChangedHash:
		postfix = "only one node sends valid block (with changed hash)"

	case cases.FBRInvalidBlockWithChangedHash:
		postfix = "only one node sends invalid block (with changed hash)"

	case cases.FBRBlockWithoutVerTickets:
		postfix = "only one node sends block without verification tickets"

	default:
		postfix = "unknown"
	}
	return FBRequestorName + ": " + postfix
}

// Decode implements MapDecoder interface.
func (n *FBRequestor) Decode(val interface{}) error {
	return mapstructure.Decode(val, n)
}

func (n *FBRequestor) getType() cases.FBRequestorType {
	switch {
	case !n.IgnoringRequestsBy.IsEmpty() &&
		n.CorrectResponseBy.IsEmpty() && n.ValidBlockWithChangedHashBy.IsEmpty() && n.InvalidBlockWithChangedHashBy.IsEmpty() && n.BlockWithoutVerTicketsBy.IsEmpty():

		return cases.FBRNoReplies

	case !n.IgnoringRequestsBy.IsEmpty() && n.CorrectResponseBy.Num() == 1 &&
		n.ValidBlockWithChangedHashBy.IsEmpty() && n.InvalidBlockWithChangedHashBy.IsEmpty() && n.BlockWithoutVerTicketsBy.IsEmpty():

		return cases.FBROnlyOneRepliesCorrectly

	case !n.IgnoringRequestsBy.IsEmpty() && n.ValidBlockWithChangedHashBy.Num() == 1 &&
		n.CorrectResponseBy.IsEmpty() && n.InvalidBlockWithChangedHashBy.IsEmpty() && n.BlockWithoutVerTicketsBy.IsEmpty():

		return cases.FBRValidBlockWithChangedHash

	case !n.IgnoringRequestsBy.IsEmpty() && n.InvalidBlockWithChangedHashBy.Num() == 1 &&
		n.CorrectResponseBy.IsEmpty() && n.ValidBlockWithChangedHashBy.IsEmpty() && n.BlockWithoutVerTicketsBy.IsEmpty():

		return cases.FBRInvalidBlockWithChangedHash

	case !n.IgnoringRequestsBy.IsEmpty() && n.BlockWithoutVerTicketsBy.Num() == 1 &&
		n.CorrectResponseBy.IsEmpty() && n.ValidBlockWithChangedHashBy.IsEmpty() && n.InvalidBlockWithChangedHashBy.IsEmpty():

		return cases.FBRBlockWithoutVerTickets

	default:
		return -1
	}
}

func (n *FBRequestor) Lock() {
	if n == nil {
		return
	}
	n.mu.Lock()
}

func (n *FBRequestor) Unlock() {
	if n == nil {
		return
	}
	n.mu.Unlock()
}
