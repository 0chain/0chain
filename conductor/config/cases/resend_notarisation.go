package cases

import (
	"sync"

	"0chain.net/conductor/cases"
	"github.com/mitchellh/mapstructure"
)

type (
	// ResendNotarisation represents TestCaseConfigurator implementation.
	ResendNotarisation struct {
		TestReport `json:"test_report" yaml:"test_report" mapstructure:"test_report"`

		Notarisation []byte
		Resent       bool

		mutex sync.Mutex
	}
)

const (
	ResendNotarisationName = "resend notarisation"
)

var (
	// Ensure ResendNotarisation implements TestCaseConfigurator.
	_ TestCaseConfigurator = (*ResendNotarisation)(nil)
)

// NewResendNotarisation creates initialised ResendNotarisation.
func NewResendNotarisation() *ResendNotarisation {
	return new(ResendNotarisation)
}

// TestCase implements TestCaseConfigurator interface.
func (n *ResendNotarisation) TestCase() cases.TestCase {
	return cases.NewResendNotarisation()
}

// Name implements TestCaseConfigurator interface.
func (n *ResendNotarisation) Name() string {
	return ResendNotarisationName
}

// Decode implements MapDecoder interface.
func (n *ResendNotarisation) Decode(val interface{}) error {
	return mapstructure.Decode(val, n)
}

func (n *ResendNotarisation) Lock() {
	if n == nil {
		return
	}
	n.mutex.Lock()
}

func (n *ResendNotarisation) Unlock() {
	if n == nil {
		return
	}
	n.mutex.Unlock()
}
