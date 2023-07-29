package config

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type StopChallengeGeneration bool
type StopWMCommit bool
type BlobberCommittedWM bool

// common types
type (
	NodeName  string // node name used in configurations
	NodeID    string // a NodeID is ID of a miner or a sharder
	Round     int64  // a Round number
	RoundName string // round name (remember round, round next after VC)
	Number    int64  // magic block number
)

// CleanupBC represents blockchain cleaning.
type CleanupBC struct {
	Timeout time.Duration `json:"timeout" yaml:"timeout" mapstructure:"timeout"`
}

type Phase int

const (
	PhaseStart      = iota //
	PhaseContribute        //
	PhaseShare             //
	PhasePublish           //
	PhaseWait              //
)

func ParsePhase(ps string) (ph Phase, err error) {
	switch strings.ToLower(ps) {
	case "start":
		return PhaseStart, nil
	case "contribute":
		return PhaseContribute, nil
	case "share":
		return PhaseShare, nil
	case "publish":
		return PhasePublish, nil
	case "wait":
		return PhaseWait, nil
	}
	return 0, fmt.Errorf("unknown phase: %q", ps)
}

// String implements standard fmt.Stringer interface.
func (p Phase) String() string {
	switch p {
	case PhaseStart:
		return "start"
	case PhaseContribute:
		return "contribute"
	case PhaseShare:
		return "share"
	case PhasePublish:
		return "publish"
	case PhaseWait:
		return "wait"
	}
	return fmt.Sprintf("Phase<%d>", int(p))
}

// A Case represents a test case.
type Case struct {
	Name string `json:"name" yaml:"name" mapstructure:"name"`
	Flow Flow   `json:"flow" yaml:"flow" mapstructure:"flow"`
}

// Set of tests.
type Set struct {
	// Name of the Set that used in the 'Config.Enable'
	Name string `json:"name" yaml:"name" mapstructure:"name"`
	// Tests names of the Set.
	Tests []string `json:"tests" yaml:"tests" mapstructure:"tests"`
}

// A system command.
type Command struct {
	WorkDir    string `json:"work_dir" yaml:"work_dir" mapstructure:"work_dir"`
	Exec       string `json:"exec" yaml:"exec" mapstructure:"exec"`
	ShouldFail bool   `json:"should_fail" yaml:"should_fail" mapstructure:"should_fail"`
	CanFail    bool   `json:"can_fail" yaml:"can_fail" mapstructure:"can_fail"`
}

// CommandName
type CommandName struct {
	Name string `json:"name" yaml:"name" mapstructure:"name"`
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
	// Commands is list of system commands to perform.
	Commands map[string]*Command `json:"commands" yaml:"commands" mapstructure:"commands"`
	// SkipWait nodes waiting (blobbers)
	SkipWait              []NodeName `json:"skip_wait" yaml:"skip_wait" mapstructure:"skip_wait"`
	StuckWarningThreshold string     `json:"stuck_warning_threshold" yaml:"stuck_warning_threshold" mapstructure:"stuck_warning_threshold"`
	Env                   map[string]string
	stuckWarningThreshold *time.Duration
}

// cleaning up custom environment variables before each test
func (c *Config) CleanupEnv() {
	c.Env = nil
}

// IsSkipWait skips waiting node initialization message.
func (c *Config) IsSkipWait(name NodeName) (ok bool) {
	for _, skip := range c.SkipWait {
		if name == skip {
			return true
		}
	}
	return // wait
}

func (c *Config) GetStuckWarningThreshold() time.Duration {
	if c.stuckWarningThreshold == nil {
		if tm, err := time.ParseDuration(c.StuckWarningThreshold); err == nil {
			c.stuckWarningThreshold = &tm
		} else {
			var tm time.Duration = 0
			c.stuckWarningThreshold = &tm
		}
	}
	return *c.stuckWarningThreshold
}

// Execute system command by its name.
func (c *Config) Execute(name string) (err error) {
	var n, ok = c.Commands[name]
	if !ok {
		return fmt.Errorf("unknown system command: %q", name)
	}

	if n.WorkDir == "" {
		n.WorkDir = "."
	}

	var (
		ss      = strings.Fields(n.Exec)
		command string
	)
	command = ss[0]
	if filepath.Base(command) != command && !strings.HasPrefix(command, ".") {
		command = "./" + filepath.Join(n.WorkDir, command)
	}
	var cmd = exec.Command(command, ss[1:]...)
	cmd.Dir = n.WorkDir

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if n.CanFail {
		return nil // ignore an error
	}

	if err == nil {
		if n.ShouldFail {
			return fmt.Errorf("command %q success (but should fail)", name)
		}
		return nil // ok
	}

	if _, ok := err.(*exec.ExitError); !ok {
		return // not exit status error
	}

	if n.ShouldFail {
		return nil // ok
	}

	return err
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

// IsEnabled returns true if given set is included in `enable“ list.
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
