package cases

import (
	"0chain.net/conductor/cases"
	"0chain.net/conductor/conductrpc/stats"
	"github.com/mitchellh/mapstructure"
)

type (
	// BadTimeoutVRFS represents TestCaseConfigurator implementation.
	BadTimeoutVRFS struct {
		TestReport `json:"test_report" yaml:"test_report" mapstructure:"test_report"`

		monitorID string

		statsCollector *stats.NodesServerStats
	}
)

const (
	BadTimeoutVRFSName = "bad timeout vrfs"
)

var (
	// Ensure BadTimeoutVRFS implements TestCaseConfigurator.
	_ TestCaseConfigurator = (*BadTimeoutVRFS)(nil)
)

// NewBadTimeoutVRFS creates initialised BadTimeoutVRFS.
func NewBadTimeoutVRFS(statsCollector *stats.NodesServerStats, monitorID string) *BadTimeoutVRFS {
	return &BadTimeoutVRFS{
		statsCollector: statsCollector,
		monitorID:      monitorID,
	}
}

// TestCase implements TestCaseConfigurator interface.
func (n *BadTimeoutVRFS) TestCase() cases.TestCase {
	return cases.NewBadTimeoutVRFS(n.statsCollector, n.monitorID)
}

// Name implements TestCaseConfigurator interface.
func (n *BadTimeoutVRFS) Name() string {
	return BadTimeoutVRFSName
}

// Decode implements MapDecoder interface.
func (n *BadTimeoutVRFS) Decode(val interface{}) error {
	return mapstructure.Decode(val, n)
}
