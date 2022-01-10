package cases

import (
	"0chain.net/conductor/cases"
	"github.com/mitchellh/mapstructure"
)

type (
	// SendInsufficientProposals represents TestCaseConfigurator implementation.
	SendInsufficientProposals struct {
		TestReport `json:"test_report" yaml:"test_report" mapstructure:"test_report"`
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
