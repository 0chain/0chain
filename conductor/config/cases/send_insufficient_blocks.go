package cases

import (
	"0chain.net/conductor/cases"

	"sync"

	"github.com/mitchellh/mapstructure"
)

type (
	// SendInsufficientProposals represents TestCaseConfigurator implementation.
	SendInsufficientProposals struct {
		TestReport `json:"test_report" yaml:"test_report" mapstructure:"test_report"`

		Sent bool

		mu sync.Mutex
	}
)

const (
	SendInsufficientProposalsName = "send insufficient proposals"
)

var (
	// Ensure SendInsufficientProposals implements TestCaseConfigurator.
	_ TestCaseConfigurator = (*SendInsufficientProposals)(nil)
)

// NewSendInsufficientProposals creates initialised SendInsufficientProposals.
func NewSendInsufficientProposals() *SendInsufficientProposals {
	return new(SendInsufficientProposals)
}

// TestCase implements TestCaseConfigurator interface.
func (n *SendInsufficientProposals) TestCase() cases.TestCase {
	return cases.NewSendInsufficientProposals()
}

// Name implements TestCaseConfigurator interface.
func (n *SendInsufficientProposals) Name() string {
	return SendInsufficientProposalsName
}

// Decode implements MapDecoder interface.
func (n *SendInsufficientProposals) Decode(val interface{}) error {
	return mapstructure.Decode(val, n)
}

func (n *SendInsufficientProposals) Lock() {
	if n == nil {
		return
	}
	n.mu.Lock()
}

func (n *SendInsufficientProposals) Unlock() {
	if n == nil {
		return
	}
	n.mu.Unlock()
}
