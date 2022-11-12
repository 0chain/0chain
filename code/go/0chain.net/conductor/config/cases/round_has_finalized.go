package cases

import (
	"0chain.net/conductor/cases"
	"github.com/mitchellh/mapstructure"
)

type (
	// NodeTypeTypeRank holds the information about the node type and the type rank of the miner
	// If node type is 0 the miner is a generator/leader. Otherwise, it is a replica.
	NodeTypeTypeRank struct {
		NodeType int `json:"node_type" yaml:"node_type" mapstructure:"node_type"`
		TypeRank int `json:"type_rank" yaml:"type_rank" mapstructure:"type_rank"`
	}
	// RoundHasFinalized represents TestCaseConfigurator implementation.
	RoundHasFinalized struct {
		Spammers         []NodeTypeTypeRank `json:"spammers" yaml:"spammers" mapstructure:"spammers"`
		SpammingReceiver NodeTypeTypeRank   `json:"spamming_receiver" yaml:"spamming_receiver" mapstructure:"spamming_receiver"`
		Round            int                `json:"round" yaml:"round" mapstructure:"round"`
	}
)

const (
	RoundHasFinalizedName = "round has finalized"
)

var (
	// Ensure RoundHasFinalized implements TestCaseConfigurator.
	_ TestCaseConfigurator = (*RoundHasFinalized)(nil)
)

// NewRoundHasFinalized creates initialised RoundHasFinalized.
func NewRoundHasFinalized() *RoundHasFinalized {
	return &RoundHasFinalized{}
}

// TestCase implements TestCaseConfigurator interface.
func (n *RoundHasFinalized) TestCase() cases.TestCase {
	return cases.NewRoundHasFinalized()
}

// Name implements TestCaseConfigurator interface.
func (n *RoundHasFinalized) Name() string {
	return RoundHasFinalizedName
}

// Decode implements MapDecoder interface.
func (n *RoundHasFinalized) Decode(val interface{}) error {
	return mapstructure.Decode(val, n)
}
