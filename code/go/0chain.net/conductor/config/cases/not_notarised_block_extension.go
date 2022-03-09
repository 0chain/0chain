package cases

import (
	"0chain.net/conductor/cases"

	"sync"

	"github.com/mitchellh/mapstructure"
)

type (
	// NotNotarisedBlockExtension represents TestCaseConfigurator implementation.
	NotNotarisedBlockExtension struct {
		TestReport `json:"test_report" yaml:"test_report" mapstructure:"test_report"`

		Configured bool

		mu sync.Mutex
	}
)

const (
	NotNotarisedBlockExtensionName = "not notarised block extension"
)

var (
	// Ensure NotNotarisedBlockExtension implements TestCaseConfigurator.
	_ TestCaseConfigurator = (*NotNotarisedBlockExtension)(nil)
)

// NewNotNotarisedBlockExtension creates initialised NotNotarisedBlockExtension.
func NewNotNotarisedBlockExtension() *NotNotarisedBlockExtension {
	return new(NotNotarisedBlockExtension)
}

// TestCase implements TestCaseConfigurator interface.
func (n *NotNotarisedBlockExtension) TestCase() cases.TestCase {
	return cases.NewNotNotarisedBlockExtension()
}

// Name implements TestCaseConfigurator interface.
func (n *NotNotarisedBlockExtension) Name() string {
	return NotNotarisedBlockExtensionName
}

// Decode implements MapDecoder interface.
func (n *NotNotarisedBlockExtension) Decode(val interface{}) error {
	return mapstructure.Decode(val, n)
}

func (n *NotNotarisedBlockExtension) Lock() {
	if n == nil {
		return
	}
	n.mu.Lock()
}

func (n *NotNotarisedBlockExtension) Unlock() {
	if n == nil {
		return
	}
	n.mu.Unlock()
}
