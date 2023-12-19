package cases

import (
	"0chain.net/conductor/cases"
	"github.com/mitchellh/mapstructure"
)

type (
	// RoundRandomSeed represents TestCaseConfigurator implementation.
	RoundRandomSeed struct {
		RandomSeed int64 `json:"random_seed" yaml:"random_seed" mapstructure:"random_seed"`
	}
)

const (
	RoundRandomSeedName = "round random seed"
)

var (
	// Ensure RoundRandomSeed implements TestCaseConfigurator.
	_ TestCaseConfigurator = (*RoundRandomSeed)(nil)
)

// NewRoundRandomSeed creates initialised RoundRandomSeed.
func NewRoundRandomSeed() *RoundRandomSeed {
	return &RoundRandomSeed{}
}

// TestCase implements TestCaseConfigurator interface.
func (n *RoundRandomSeed) TestCase() cases.TestCase {
	return cases.NewRoundHasFinalized()
}

// Name implements TestCaseConfigurator interface.
func (n *RoundRandomSeed) Name() string {
	return RoundRandomSeedName
}

// Decode implements MapDecoder interface.
func (n *RoundRandomSeed) Decode(val interface{}) error {
	return mapstructure.Decode(val, n)
}
