package config

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
)

// CleanupBC represents blockchain cleaning.
type CleanupBC struct {
	Timeout time.Duration `json:"timeout" yaml:"timeout" mapstructure:"timeout"`
}

// MagicBlock represents expected magic block.
type MagicBlock struct {
	// Round ignored if it's zero. If set a positive value, then this
	// round is expected.
	Round int64 `json:"round" yaml:"round" mapstructure:"round"`
	// RoundNextVCAfter used in combination with wait_view_change.remember_round
	// that remember round with some name. This directive expects next VC round
	// after the remembered one. For example, if round 340 has remembered as
	// "enter_miner5", then "round_next_vc_after": "enter_miner5", expects
	// 500 round (next VC after the remembered round). Empty string ignored.
	RoundNextVCAfter string `json:"round_next_vc_after" yaml:"round_next_vc_after" mapstructure:"round_next_vc_after"`
	// Sharders expected in MB.
	Sharders []string `json:"sharders" yaml:"sharders" mapstructure:"sharders"`
	// Miners expected in MB.
	Miners []string `json:"miners" yaml:"miners" mapstructure:"miners"`
}

// IsZero returns true if the MagicBlock is empty.
func (mb *MagicBlock) IsZero() bool {
	return mb.Round == 0 &&
		mb.RoundNextVCAfter == "" &&
		len(mb.Sharders) == 0 &&
		len(mb.Miners) == 0
}

// ViewChange flow configuration.
type ViewChange struct {
	RememberRound    string     `json:"remember_round" yaml:"remember_round" mapstructure:"remember_round"`
	ExpectMagicBlock MagicBlock `json:"expect_magic_block" yaml:"expect_magic_block" mapstructure:"expect_magic_block"`
}

// IsZero returns true if the ViewChagne is empty.
func (vc *ViewChange) IsZero() bool {
	return vc.RememberRound == "" &&
		vc.ExpectMagicBlock.IsZero()
}

type Phase int

const (
	Unknown    = iota // illegal
	Start             //
	Contribute        //
	Share             //
	Publish           //
	Wait              //
)

func (p *Phase) UnmarshalYAML(unmarshal func(interface{}) error) (err error) {
	var ps string
	if err = unmarshal(&ps); err != nil {
		return
	}
	switch strings.ToLower(ps) {
	case "start":
		(*p) = Start
	case "contribute":
		(*p) = Contribute
	case "share":
		(*p) = Share
	case "publish":
		(*p) = Publish
	case "wait":
		(*p) = Wait
	default:
		return fmt.Errorf("unknown phase: %q", ps)
	}
	return // nil
}

// WaitPhase flow configuration.
type WaitPhase struct {
	// Phase to wait for (number).
	Phase Phase `json:"phase" yaml:"phase" mapstructure:"phase"`
	// Node of which the phase expected.
	Node string `json:"node" yaml:"node" mapstructure:"node"`
	// ViewChangeRound after which the phase expected (and before next VC),
	// value can be an empty string for any VC.
	ViewChangeRound string `json:"view_change_round" yaml:"view_change_round" mapstructure:"view_change_round"`
}

// IsZero returns true if the WaitPhase is empty.
func (wp *WaitPhase) IsZero() bool {
	return wp.Phase == 0 && wp.Node == "" && wp.ViewChangeRound == ""
}

// Executor used by a Flow to perform a flow directive.
type Executor interface {
	Start(names []string, lock bool, timeout time.Duration) (err error)
	WaitViewChange(vc ViewChange, timeout time.Duration) (err error)
	WaitPhase(wp WaitPhase, timeout time.Duration) (err error)
	Unlock(names []string, timeout time.Duration) (err error)
	Stop(names []string, timeout time.Duration) (err error)
	CleanupBC(timeout time.Duration) (err error)
}

// The Flow represents single value map.
//
//     start            - list of 'sharder 1', 'miner 1', etc
//     wait_view_change - remember_round and/or expect_magic_block
//     wait_phase       - wait for a phase
//     unlock           - see start
//     stop             - see start
//
// See below for a possible map formats.
type Flow map[string]interface{}

func (f Flow) getFirst() (name string, val interface{}, ok bool) {
	for name, val = range f {
		ok = true
		return
	}
	return
}

func getStrings(val interface{}) (ss []string, ok bool) {
	switch tt := val.(type) {
	case string:
		return []string{tt}, true
	case []string:
		return tt, true
	}
	return // nil, false
}

// Execute the flow directive.
func (f Flow) Execute(ex Executor) (err error) {
	var name, val, ok = f.getFirst()
	if !ok {
		return errors.New("invalid empty flow")
	}

	var tm time.Duration

	// extract timeout
	if msi, ok := val.(map[string]interface{}); ok {
		if tmsi, ok := msi["timeout"]; ok {
			tms, ok := tmsi.(string)
			if !ok {
				return fmt.Errorf("invalid 'timeout' type: %T", tmsi)
			}
			if tm, err = time.ParseDuration(tms); err != nil {
				return fmt.Errorf("paring 'timeout' %q: %v", tms, err)
			}
			delete(msi, "timeout")
		}
	}

	switch name {
	case "cleanup_bc":
		return ex.CleanupBC(tm)
	case "start":
		if ss, ok := getStrings(val); ok {
			return ex.Start(ss, false, tm)
		}
	case "wait_view_change":
		var vc ViewChange
		if err = mapstructure.Decode(val, &vc); err != nil {
			return fmt.Errorf("invalid '%s' argument type: %T, "+
				"decoding error: %v", name, val, err)
		}
		return ex.WaitViewChange(vc, tm)
	case "start_lock":
		if ss, ok := getStrings(val); ok {
			return ex.Start(ss, true, tm)
		}
	case "wait_phase":
		var wp WaitPhase
		if err = mapstructure.Decode(val, &wp); err != nil {
			return fmt.Errorf("invalid '%s' argument type: %T, "+
				"decoding error: %v", name, val, err)
		}
		return ex.WaitPhase(wp, tm)
	case "unlock":
		if ss, ok := getStrings(val); ok {
			return ex.Unlock(ss, tm)
		}
	case "stop":
		if ss, ok := getStrings(val); ok {
			return ex.Stop(ss, tm)
		}
	default:
		return fmt.Errorf("unknown flow directive: %q", name)
	}
	return fmt.Errorf("invalid '%s' argument type: %T", name, val)
}

// Flows represents order of start/stop miners/sharder and other BC events.
type Flows []Flow

// A Case represents a test case.
type Case struct {
	Name string `json:"name" yaml:"name" mapstructure:"name"`
	Flow Flows  `json:"flow" yaml:"flow" mapstructure:"flow"`
}

// A Node used in tests.
type Node struct {
	// Name used in flow configurations and logs.
	Name string `json:"name" yaml:"name" mapstructure:"name"`
	// ID used in RPC.
	ID string `json:"id" yaml:"id" mapstructure:"id"`
	// WorkDir to start the node in.
	WorkDir string `json:"work_dir" yaml:"work_dir" mapstructure:"work_dir"`
	// StartCommand to start the node.
	StartCommand string `json:"start_command" yaml:"start_command" mapstructure:"start_command"`

	// internals
	Command *exec.Cmd `json:"-" yaml:"-" mapstructure:"-"`
}

// Start the Node.
func (n *Node) Start(logsDir string) (err error) {
	if n.WorkDir == "" {
		n.WorkDir = "."
	}
	var (
		ss      = strings.Fields(n.StartCommand)
		command string
	)
	command = ss[0]
	if filepath.Base(command) != command {
		command = filepath.Join(n.WorkDir, command)
	}
	var cmd = exec.Command(command, ss[1:]...)
	cmd.Dir = n.WorkDir

	logsDir = filepath.Join(logsDir, n.Name)
	if err = os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("creating logs directory %s: %v", logsDir, err)
	}

	cmd.Stdout, err = os.Create(filepath.Join(logsDir, "STDOUT.log"))
	if err != nil {
		return fmt.Errorf("creating STDOUT file: %v", err)
	}

	cmd.Stderr, err = os.Create(filepath.Join(logsDir, "STDERR.log"))
	if err != nil {
		return fmt.Errorf("creating STDERR file: %v", err)
	}

	n.Command = cmd
	return cmd.Start()
}

// Interrupt sends SIGINT to the command if its running.
func (n *Node) Interrupt() (err error) {
	if n.Command == nil {
		return fmt.Errorf("command %v not started", n.Name)
	}
	var proc = n.Command.Process
	if proc == nil {
		return fmt.Errorf("missing command %v process", n.Name)
	}
	if err = proc.Signal(os.Interrupt); err != nil {
		return fmt.Errorf("command %v: sending SIGINT: %v", n.Name, err)
	}
	return
}

// Kill the command if started.
func (n *Node) Kill() (err error) {
	if n.Command != nil && n.Command.Process != nil {
		return n.Command.Process.Kill()
	}
	return
}

// Stop interrupts command and waits it. Then it closes STDIN and STDOUT
// files (logs).
func (n *Node) Stop() (err error) {
	if err = n.Interrupt(); err != nil {
		return fmt.Errorf("interrupting: %v", err)
	}
	if err = n.Command.Wait(); err != nil {
		err = fmt.Errorf("waiting the command: %v", err) // don't return
	}
	if stdin, ok := n.Command.Stdin.(*os.File); ok {
		stdin.Close() // ignore error
	}
	if stderr, ok := n.Command.Stderr.(*os.File); ok {
		stderr.Close() // ignore error
	}
	return // nil or error
}

// Nodes is list of nodes.
type Nodes []*Node

// NodeByName returns node by name.
func (ns Nodes) NodeByName(name string) (n *Node, ok bool) {
	for _, x := range ns {
		if x.Name == name {
			return x, true
		}
	}
	return // nil, false
}

// A Config represents conductor testing configurations.
type Config struct {
	// Address is RPC server address
	Address string `json:"address" yaml:"address" mapstructure:"address"`
	// Logs is directory for stdin and stdout logs.
	Logs string `json:"logs" yaml:"logs" mapstructure:"logs"`
	// Nodes for tests.
	Nodes Nodes `json:"nodes" yaml:"nodes" mapstructure:"nodes"`
	// Tests cases and related.
	Tests []Case `json:"tests" yaml:"tests" mapstructure:"tests"`
}
