package config

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// common types
type (
	NodeName  string // node name used in configurations
	NodeID    string // a NodeID is ID of a miner or a sharder
	Round     int64  // a Round number
	RoundName string // round name (remember round, round next after VC)
)

// CleanupBC represents blockchain cleaning.
type CleanupBC struct {
	Timeout time.Duration `json:"timeout" yaml:"timeout" mapstructure:"timeout"`
}

type Phase int

const (
	Start      = iota //
	Contribute        //
	Share             //
	Publish           //
	Wait              //
)

func ParsePhase(ps string) (ph Phase, err error) {
	switch strings.ToLower(ps) {
	case "start":
		return Start, nil
	case "contribute":
		return Contribute, nil
	case "share":
		return Share, nil
	case "publish":
		return Publish, nil
	case "wait":
		return Wait, nil
	}
	return 0, fmt.Errorf("unknown phase: %q", ps)
}

// String implements standard fmt.Stringer interface.
func (p Phase) String() string {
	switch p {
	case Start:
		return "start"
	case Contribute:
		return "contribute"
	case Share:
		return "share"
	case Publish:
		return "publish"
	case Wait:
		return "wait"
	}
	return fmt.Sprintf("Phase<%d>", int(p))
}

// A Case represents a test case.
type Case struct {
	Name string `json:"name" yaml:"name" mapstructure:"name"`
	Flow Flows  `json:"flow" yaml:"flow" mapstructure:"flow"`
}

// Set of tests.
type Set struct {
	// Name of the Set that used in the 'Config.Enable'
	Name string `json:"name" yaml:"name" mapstructure:"name"`
	// Tests names of the Set.
	Tests []string `json:"tests" yaml:"tests" mapstructure:"tests"`
}

// A Config represents conductor testing configurations.
type Config struct {
	// BindRunner endpoint to connect to.
	BindRunner string `json:"bind_runner" yaml:"bind_runner" mapstructure:"bind_runner"`
	// Runner endpoint to connect to.
	Runner string `json:"runner" yaml:"runner" mapstructure:"runner"`
	// Bind is address to start RPC server.
	Bind string `json:"bind" yaml:"bind" mapstructure:"bind"`
	// Address is address of RPC server in docker network (e.g.
	// address to connect to).
	Address string `json:"address" yaml:"address" mapstructure:"address"`
	// Logs is directory for stdin and stdout logs.
	Logs string `json:"logs" yaml:"logs" mapstructure:"logs"`
	// Nodes for tests.
	Nodes Nodes `json:"nodes" yaml:"nodes" mapstructure:"nodes"`
	// Tests cases and related.
	Tests []Case `json:"tests" yaml:"tests" mapstructure:"tests"`
	// CleanupCommand used to cleanup BC. All nodes should be stopped before.
	CleanupCommand string `json:"cleanup_command" yaml:"cleanup_command" mapstructure:"cleanup_command"`
	// ViewChange is number of rounds for a view change (e.g. 250, 50 per phase).
	ViewChange Round `json:"view_change" yaml:"view_change" mapstructure:"view_change"`
	// Enable sets of tests.
	Enable []string `json:"enable" yaml:"enable" mapstructure:"enable"`
	// Sets or tests.
	Sets []Set `json:"sets" yaml:"sets" mapstructure:"sets"`
}

// TestsOfSet returns test cases of given Set.
func (c *Config) TestsOfSet(set *Set) (cs []Case) {
	for _, name := range set.Tests {
		for _, testCase := range c.Tests {
			if testCase.Name == name {
				cs = append(cs, testCase)
			}
		}
	}
	return
}

// IsEnabled returns true it given Set enabled.
func (c *Config) IsEnabled(set *Set) bool {
	for _, name := range c.Enable {
		if set.Name == name {
			return true
		}
	}
	return false
}

// CleanupBC used to execute the configured cleanup_command.
func (c *Config) CleanupBC() (err error) {
	if c.CleanupCommand == "" {
		return errors.New("no cleanup_command given in conductor.yaml")
	}

	var (
		ss  = strings.Fields(c.CleanupCommand)
		cmd = exec.Command(ss[0], ss[1:]...)
	)

	return cmd.Run()
}
