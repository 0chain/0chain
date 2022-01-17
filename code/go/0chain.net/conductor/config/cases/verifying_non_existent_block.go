package cases

import (
	"sync"

	"0chain.net/conductor/cases"
	"0chain.net/conductor/conductrpc/stats"
	"github.com/mitchellh/mapstructure"
)

type (
	// VerifyingNonExistentBlock represents TestCaseConfigurator implementation.
	VerifyingNonExistentBlock struct {
		Hash       string `json:"hash" yaml:"hash" mapstructure:"hash"`
		TestReport `json:"test_report" yaml:"test_report" mapstructure:"test_report"`

		IgnoredVerificationTicketsNum int

		Sent bool

		statsCollector *stats.NodesServerStats

		mutex sync.Mutex
	}
)

const (
	VerifyingNonExistentBlockName = "verifying non existent block"
)

var (
	// Ensure VerifyingNonExistentBlock implements TestCaseConfigurator.
	_ TestCaseConfigurator = (*VerifyingNonExistentBlock)(nil)
)

// NewVerifyingNonExistentBlock creates initialised VerifyingNonExistentBlock.
func NewVerifyingNonExistentBlock(statsCollector *stats.NodesServerStats) *VerifyingNonExistentBlock {
	return &VerifyingNonExistentBlock{
		statsCollector: statsCollector,
	}
}

// TestCase implements TestCaseConfigurator interface.
func (n *VerifyingNonExistentBlock) TestCase() cases.TestCase {
	return cases.NewVerifyingNonExistentBlock(n.Hash, int(n.TestReport.OnRound), n.statsCollector)
}

// Name implements TestCaseConfigurator interface.
func (n *VerifyingNonExistentBlock) Name() string {
	return VerifyingNonExistentBlockName
}

// Decode implements MapDecoder interface.
func (n *VerifyingNonExistentBlock) Decode(val interface{}) error {
	return mapstructure.Decode(val, n)
}

func (n *VerifyingNonExistentBlock) Lock() {
	if n == nil {
		return
	}
	n.mutex.Lock()
}

func (n *VerifyingNonExistentBlock) Unlock() {
	if n == nil {
		return
	}
	n.mutex.Unlock()
}
