package cases

import (
	"0chain.net/conductor/cases"
	"github.com/mitchellh/mapstructure"
)

type (
	// HalfNodesDown represents TestCaseConfigurator implementation.
	HalfNodesDown struct {
		TestReport `json:"test_report" yaml:"test_report" mapstructure:"test_report"`

		minersNum int
	}
)

const (
	HalfNodesDownName = "half nodes down"
)

var (
	// Ensure HalfNodesDown implements TestCaseConfigurator.
	_ TestCaseConfigurator = (*HalfNodesDown)(nil)
)

// NewHalfNodesDown creates initialised HalfNodesDown.
func NewHalfNodesDown(minersNum int) *HalfNodesDown {
	return &HalfNodesDown{
		minersNum: minersNum,
	}
}

// TestCase implements TestCaseConfigurator interface.
func (n *HalfNodesDown) TestCase() cases.TestCase {
	return cases.NewHalfNodesDown(n.minersNum)
}

// Name implements TestCaseConfigurator interface.
func (n *HalfNodesDown) Name() string {
	return HalfNodesDownName
}

// Decode implements MapDecoder interface.
func (n *HalfNodesDown) Decode(val interface{}) error {
	return mapstructure.Decode(val, n)
}
