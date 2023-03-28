package cases

import (
	"0chain.net/conductor/cases"
	"0chain.net/conductor/conductrpc/stats"
	"github.com/mitchellh/mapstructure"
)

type (
	// NotarisingNonExistentBlock represents TestCaseConfigurator implementation.
	NotarisingNonExistentBlock struct {
		Hash string `json:"hash" yaml:"hash" mapstructure:"hash"`

		TestReport `json:"test_report" yaml:"test_report" mapstructure:"test_report"`

		statsCollector *stats.NodesServerStats
	}
)

const (
	NotarisingNonExistentBlockName = "notarising non existent block"
)

var (
	// Ensure NotarisingNonExistentBlock implements TestCaseConfigurator.
	_ TestCaseConfigurator = (*NotarisingNonExistentBlock)(nil)
)

// NewNotarisingNonExistentBlock creates initialised NotarisingNonExistentBlock.
func NewNotarisingNonExistentBlock(statsCollector *stats.NodesServerStats) *NotarisingNonExistentBlock {
	return &NotarisingNonExistentBlock{
		statsCollector: statsCollector,
	}
}

// TestCase implements TestCaseConfigurator interface.
func (n *NotarisingNonExistentBlock) TestCase() cases.TestCase {
	return cases.NewNotarisingNonExistentBlock(n.statsCollector)
}

// Name implements TestCaseConfigurator interface.
func (n *NotarisingNonExistentBlock) Name() string {
	return NotarisingNonExistentBlockName
}

// Decode implements MapDecoder interface.
func (n *NotarisingNonExistentBlock) Decode(val interface{}) error {
	return mapstructure.Decode(val, n)
}
