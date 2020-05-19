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
	RememberRound    string        `json:"remember_round" yaml:"remember_round" mapstructure:"remember_round"`
	ExpectMagicBlock MagicBlock    `json:"expect_magic_block" yaml:"expect_magic_block" mapstructure:"expect_magic_block"`
	Timeout          time.Duration `json:"timeout" yaml:"timeout" mapstructure:"timeout"`
}

// IsZero returns true if the ViewChagne is empty.
func (vc *ViewChange) IsZero() bool {
	return vc.RememberRound == "" &&
		vc.ExpectMagicBlock.IsZero() &&
		vc.Timeout == 0
}

// WaitPhase flow configuration.
type WaitPhase struct {
	// Phase to wait for (number.
	Phase int `json:"phase" yaml:"phase" mapstructure:"phase"`
	// Timeout to wait (to fail).
	Timeout time.Duration `json:"timeout" yaml:"timeout" mapstructure:"timeout"`
}

// IsZero returns true if the WaitPhase is empty.
func (wp *WaitPhase) IsZero() bool {
	return wp.Phase == 0 && wp.Timeout == 0
}

// Executor used by a Flow to perform a flow directive.
type Executor interface {
	Start(names []string, lock bool) (err error)
	WaitViewChange(vc ViewChange) (err error)
	WaitPhase(phase int, timeout time.Duration) (err error)
	Unlock(names []string) (err error)
	Stop(names []string) (err error)
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
	switch name {
	case "start":
		if ss, ok := getStrings(val); ok {
			return ex.Start(ss, false)
		}
	case "wait_view_change":
		var vc ViewChange
		if err = mapstructure.Decode(val, &vc); err != nil {
			return fmt.Errorf("invalid '%s' argument type: %T, "+
				"decoding error: %v", name, val, err)
		}
		return ex.WaitViewChange(vc)
	case "start_lock":
		if ss, ok := getStrings(val); ok {
			return ex.Start(ss, true)
		}
	case "wait_phase":
		var wp WaitPhase
		if err = mapstructure.Decode(val, &wp); err != nil {
			return fmt.Errorf("invalid '%s' argument type: %T, "+
				"decoding error: %v", name, val, err)
		}
		return ex.WaitPhase(wp.Phase, wp.Timeout)
	case "unlock":
		if ss, ok := getStrings(val); ok {
			return ex.Unlock(ss)
		}
	case "stop":
		if ss, ok := getStrings(val); ok {
			return ex.Stop(ss)
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

type Test struct {
	// CleanupBlockchain used to rest BC to initial
	// to make all nodes not registered.
	CleanupBlockchain bool `json:"cleanup_blockchain" yaml:"cleanup_blockchain" mapstructure:"cleanup_blockchain"`
	// Cases contains test cases.
	Cases []Case `json:"cases" yaml:"cases" mapstructure:"cases"`
}

// A Config represents conductor testing configurations.
type Config struct {
	// Address is RPC server address
	Address string `json:"address" yaml:"address" mapstructure:"address"`
	// WorkDir relative or absolute.
	WorkDir string `json:"work_dir" yaml:"work_dir" mapstructure:"work_dir"`
	// Logs is directory for stdin and stdout logs.
	Logs string `json:"logs" yaml:"logs" mapstructure:"logs"`
	// Nodes for tests.
	Nodes []Node `json:"nodes" yaml:"nodes" mapstructure:"nodes"`
	// Test cases and related.
	Test Test `json:"cases" yaml:"cases" mapstructure:"cases"`
}

/*

Example configurations
======================

#
# conductor BC testing configurations
#
---

# RPC server address (listen on)
address: 127.0.0.1:15210
# RPC server working directory (creates logs)
working_directory: ".."
# place for stdin and stdout logs of nodes (relative to the working_directory)
logs: "logs"

#
# nodes used in tests
#
nodes:
  - name: "sharder 1"
    id: 57b416fcda1cf82b8a7e1fc3a47c68a94e617be873b5383ea2606bda757d3ce4
    work_dir: "docker.local/sharder1"
    start_command: "../bin/start.b0sharder.sh"

  - name: "sharder 2"
    id: b098d2d56b087ee910f3ee2d2df173630566babb69f0be0e2e9a0c98d63f0b0b
    work_dir: "docker.local/sharder2"
    start_command: "../bin/start.b0sharder.sh"

  - name: "sharder 3"
    id: d9558143f8e976126367603bff34125f5eb94720df8d7acefffdd66675d134c2
    work_dir: "docker.local/sharder3"
    start_command: "../bin/start.b0sharder.sh"

  - name: "miner 1"
    id: 31810bd1258ae95955fb40c7ef72498a556d3587121376d9059119d280f34929
    work_dir: "docker.local/miner1"
    start_command: "../bin/start.b0miner.sh"

  - name: "miner 2"
    id: 585732eb076d07455fbebcf3388856b6fd00449a25c47c0f72d961c7c4e7e7c2
    work_dir: "docker.local/miner2"
    start_command: "../bin/start.b0miner.sh"

  - name: "miner 3"
    id: bfa64c67f49bceec8be618b1b6f558bdbaf9c100fd95d55601fa2190a4e548d8
    work_dir: "docker.local/miner3"
    start_command: "../bin/start.b0miner.sh"

  - name: "miner 4"
    id: 8877e3da19b4cb51e59b4646ec7c0cf4849bc7b860257d69ddbf753b9a981e1b
    work_dir: "docker.local/miner4"
    start_command: "../bin/start.b0miner.sh"

  - name: "miner 5"
    id: 53add50ff9501014df2cbd698c673f85e5785281cebba8772a64a6e74057d328
    work_dir: "docker.local/miner5"
    start_command: "../bin/start.b0miner.sh"

#
# test cases
#
test:
  # cleanup blockchain to make all nodes not registered
  cleanup_blockchain: true
  # test cases
  cases:
    # preparation
    - name: "prepare"
      flow:
      - start:
        - sharder 1
      - start:
        - miner 1
        - miner 2
        - miner 3

    # sharder 1, miner 1, 2, 3 is running
    # on a phase 0 miner 4 comes up BC
    # (miner 4 is not registered if BC cleaned before)
    - name: "miner 4 comes up on phase 0"
      flow:
      - wait_view_change:
          timeout: 10m
          remember_round: "starting_round"
      - start_lock:
        - miner 4
      - wait_phase:
          phase: 0
          timeout: 1 minute
      - unlock:
          - miner 4
      - wait_view_change:
          timeout: 10 minute
          expect_magic_block:
            round_next_vc_after: "starting_round"
            sharders:
              - sharder 1
            miners:
              - miner 1
              - miner 2
              - miner 3
              - miner 4

    - name: "miner 4 goes down until next view change"
      flow:
      - wait_view_change:
          timeout: 10m
          remember_round: "starting_round"
      - wait_phase:
          phase: 0
          timeout: 1 minute
      - stop:
          - miner 4
      - wait_view_change:
          timeout: 10 minute
          expect_magic_block:
            round_next_vc_after: "starting_round"
            sharders:
              - sharder 1
            miners:
              - miner 1
              - miner 2
              - miner 3
...

*/
