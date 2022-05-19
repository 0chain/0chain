package cases

import (
	"0chain.net/conductor/cases"
	"0chain.net/conductor/conductrpc/stats"
	"github.com/mitchellh/mapstructure"
)

type (
	// RoundHasFinalized represents TestCaseConfigurator implementation.
	RoundHasFinalized struct {
		Spammers []string `json:"spammers" yaml:"spammers" mapstructure:"spammers"`

		monitorID string

		statsCollector *stats.NodesServerStats
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
