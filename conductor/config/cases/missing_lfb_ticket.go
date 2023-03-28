package cases

import (
	"github.com/mitchellh/mapstructure"

	"0chain.net/conductor/cases"
)

type (
	// MissingLFBTickets represents TestCaseConfigurator implementation.
	MissingLFBTickets struct {
		TestReport `json:"test_report" yaml:"test_report" mapstructure:"test_report"`

		minersNum int
	}
)

const (
	MissingLFBTicketsName = "missing lfb tickets"
)

var (
	// Ensure MissingLFBTickets implements TestCaseConfigurator.
	_ TestCaseConfigurator = (*MissingLFBTickets)(nil)
)

// NewMissingLFBTickets creates initialised MissingLFBTickets.
func NewMissingLFBTickets(minersNum int) *MissingLFBTickets {
	return &MissingLFBTickets{
		minersNum: minersNum,
	}
}

// TestCase implements TestCaseConfigurator interface.
func (n *MissingLFBTickets) TestCase() cases.TestCase {
	return cases.NewMissingLFBTickets(n.minersNum)
}

// Name implements TestCaseConfigurator interface.
func (n *MissingLFBTickets) Name() string {
	return MissingLFBTicketsName
}

// Decode implements MapDecoder interface.
func (n *MissingLFBTickets) Decode(val interface{}) error {
	return mapstructure.Decode(val, n)
}
