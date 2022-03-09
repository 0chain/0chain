package cases

import (
	"0chain.net/conductor/cases"
	"github.com/mitchellh/mapstructure"
)

type (
	// SendDifferentBlocksFromFirstGenerator represents TestCaseConfigurator implementation.
	SendDifferentBlocksFromFirstGenerator struct {
		TestReport `json:"test_report" yaml:"test_report" mapstructure:"test_report"`

		minersNum int
	}
)

const (
	SendDifferentBlocksFromFirstGeneratorName = "send different blocks from first generator"
)

var (
	// Ensure SendDifferentBlocksFromFirstGenerator implements TestCaseConfigurator.
	_ TestCaseConfigurator = (*SendDifferentBlocksFromFirstGenerator)(nil)
)

// NewSendDifferentBlocksFromFirstGenerator creates initialised SendDifferentBlocksFromFirstGenerator.
func NewSendDifferentBlocksFromFirstGenerator(minersNum int) *SendDifferentBlocksFromFirstGenerator {
	return &SendDifferentBlocksFromFirstGenerator{
		minersNum: minersNum,
	}
}

// TestCase implements TestCaseConfigurator interface.
func (n *SendDifferentBlocksFromFirstGenerator) TestCase() cases.TestCase {
	return cases.NewSendDifferentBlocksFromFirstGenerator(n.minersNum)
}

// Name implements TestCaseConfigurator interface.
func (n *SendDifferentBlocksFromFirstGenerator) Name() string {
	return SendDifferentBlocksFromFirstGeneratorName
}

// Decode implements MapDecoder interface.
func (n *SendDifferentBlocksFromFirstGenerator) Decode(val interface{}) error {
	return mapstructure.Decode(val, n)
}
