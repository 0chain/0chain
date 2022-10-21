package cases

import (
	"0chain.net/conductor/cases"
	"github.com/mitchellh/mapstructure"
)

type (
	// SendDifferentBlocksFromAllGenerators represents TestCaseConfigurator implementation.
	SendDifferentBlocksFromAllGenerators struct {
		Round int `json:"round" yaml:"round" mapstructure:"round"`

		minersNum int
	}
)

const (
	SendDifferentBlocksFromAllGeneratorsName = "send different blocks from all generators"
)

var (
	// Ensure SendDifferentBlocksFromAllGenerators implements TestCaseConfigurator.
	_ TestCaseConfigurator = (*SendDifferentBlocksFromAllGenerators)(nil)
)

// NewSendDifferentBlocksFromAllGenerators creates initialised SendDifferentBlocksFromAllGenerators.
func NewSendDifferentBlocksFromAllGenerators(minersNum int) *SendDifferentBlocksFromAllGenerators {
	return &SendDifferentBlocksFromAllGenerators{
		minersNum: minersNum,
	}
}

// TestCase implements TestCaseConfigurator interface.
func (n *SendDifferentBlocksFromAllGenerators) TestCase() cases.TestCase {
	return cases.NewSendDifferentBlocksFromAllGenerators(n.minersNum)
}

// Name implements TestCaseConfigurator interface.
func (n *SendDifferentBlocksFromAllGenerators) Name() string {
	return SendDifferentBlocksFromAllGeneratorsName
}

// Decode implements MapDecoder interface.
func (n *SendDifferentBlocksFromAllGenerators) Decode(val interface{}) error {
	return mapstructure.Decode(val, n)
}

func (n *SendDifferentBlocksFromAllGenerators) IsTesting(round int64, generator bool, nodeTypeRank int) bool {
	return int64(n.Round) == round && generator && nodeTypeRank == 0
}
