package cases

import (
	"log"

	"github.com/mitchellh/mapstructure"

	"0chain.net/conductor/cases"
)

type (
	// CollectVerificationTicket represents TestCaseConfigurator implementation.
	CollectVerificationTicket struct {
		TestReport `json:"test_report" yaml:"test_report" mapstructure:"test_report"`
	}
)

const (
	CollectVerificationTicketName = "collect verification ticket"
)

var (
	// Ensure HalfNodesDown implements TestCaseConfigurator.
	_ TestCaseConfigurator = (*CollectVerificationTicket)(nil)
)

// NewCollectVerification creates initialised CollectVerificationTicket.
func NewCollectVerification() *CollectVerificationTicket {
	return new(CollectVerificationTicket)
}

// TestCase implements TestCaseConfigurator interface.
func (n *CollectVerificationTicket) TestCase() cases.TestCase {
	return cases.NewCollectVerificationTicket()
}

// Name implements TestCaseConfigurator interface.
func (n *CollectVerificationTicket) Name() string {
	return CollectVerificationTicketName
}

// Decode implements MapDecoder interface.
func (n *CollectVerificationTicket) Decode(val interface{}) error {
	return mapstructure.Decode(val, n)
}

func (n *CollectVerificationTicket) IsTesting(round int64, generator bool, nodeTypeRank int, isMonitor bool) bool {
	log.Printf("onround %v + 1 and round %v, is generator %v", n.OnRound, round, generator)
	return n.OnRound+1 == round && isMonitor
}
