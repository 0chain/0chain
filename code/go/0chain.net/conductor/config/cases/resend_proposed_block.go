package cases

import (
	"sync"

	"0chain.net/conductor/cases"
	"github.com/mitchellh/mapstructure"
)

type (
	// ResendProposedBlock represents TestCaseConfigurator implementation.
	ResendProposedBlock struct {
		TestReport `json:"test_report" yaml:"test_report" mapstructure:"test_report"`

		Resent bool

		mutex sync.Mutex
	}
)

const (
	ResendProposedBlockName = "resend proposed block"
)

var (
	// Ensure ResendProposedBlock implements TestCaseConfigurator.
	_ TestCaseConfigurator = (*ResendProposedBlock)(nil)
)

// NewResendProposedBlock creates initialised ResendProposedBlock.
func NewResendProposedBlock() *ResendProposedBlock {
	return new(ResendProposedBlock)
}

// TestCase implements TestCaseConfigurator interface.
func (n *ResendProposedBlock) TestCase() cases.TestCase {
	return cases.NewResendProposedBlock()
}

// Name implements TestCaseConfigurator interface.
func (n *ResendProposedBlock) Name() string {
	return ResendProposedBlockName
}

// Decode implements MapDecoder interface.
func (n *ResendProposedBlock) Decode(val interface{}) error {
	return mapstructure.Decode(val, n)
}

func (n *ResendProposedBlock) Lock() {
	if n == nil {
		return
	}
	n.mutex.Lock()
}

func (n *ResendProposedBlock) Unlock() {
	if n == nil {
		return
	}
	n.mutex.Unlock()
}
