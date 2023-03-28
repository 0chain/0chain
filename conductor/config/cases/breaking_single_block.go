package cases

import (
	"0chain.net/conductor/cases"

	"sync"

	"github.com/mitchellh/mapstructure"
)

type (
	// BreakingSingleBlock represents TestCaseConfigurator implementation.
	BreakingSingleBlock struct {
		TestReport `json:"test_report" yaml:"test_report" mapstructure:"test_report"`

		Sent bool

		mu sync.Mutex
	}
)

const (
	BreakingSingleBlockName = "breaking single block"
)

var (
	// Ensure BreakingSingleBlock implements TestCaseConfigurator.
	_ TestCaseConfigurator = (*BreakingSingleBlock)(nil)
)

// NewBreakingSingleBlock creates initialised BreakingSingleBlock.
func NewBreakingSingleBlock() *BreakingSingleBlock {
	return new(BreakingSingleBlock)
}

// TestCase implements TestCaseConfigurator interface.
func (n *BreakingSingleBlock) TestCase() cases.TestCase {
	return cases.NewBreakingSingleBlock()
}

// Name implements TestCaseConfigurator interface.
func (n *BreakingSingleBlock) Name() string {
	return BreakingSingleBlockName
}

// Decode implements MapDecoder interface.
func (n *BreakingSingleBlock) Decode(val interface{}) error {
	return mapstructure.Decode(val, n)
}

func (n *BreakingSingleBlock) Lock() {
	if n == nil {
		return
	}
	n.mu.Lock()
}

func (n *BreakingSingleBlock) Unlock() {
	if n == nil {
		return
	}
	n.mu.Unlock()
}
