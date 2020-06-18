package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// A Node used in tests.
type Node struct {
	// Name used in flow configurations and logs.
	Name NodeName `json:"name" yaml:"name" mapstructure:"name"`
	// ID used in RPC.
	ID NodeID `json:"id" yaml:"id" mapstructure:"id"`
	// WorkDir to start the node in.
	WorkDir string `json:"work_dir" yaml:"work_dir" mapstructure:"work_dir"`
	// Env added to os.Environ
	Env string `json:"env" yaml:"env" mapstructure:"env"`
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
	// if filepath.Base(command) != command {
	// 	command = "./" + filepath.Join(n.WorkDir, command)
	// }
	var cmd = exec.Command(command, ss[1:]...)
	cmd.Dir = n.WorkDir
	if n.Env != "" {
		cmd.Env = append(os.Environ(), n.Env)
	}

	logsDir = filepath.Join(logsDir, string(n.Name))
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

func (n *Node) IsStarted() bool {
	return n.Command != nil
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
func (ns Nodes) NodeByName(name NodeName) (n *Node, ok bool) {
	for _, x := range ns {
		if x.Name == name {
			return x, true
		}
	}
	return // nil, false
}

// NodeByID returns node by ID.
func (ns Nodes) NodeByID(id NodeID) (n *Node, ok bool) {
	for _, x := range ns {
		if x.ID == id {
			return x, true
		}
	}
	return // nil, false
}

// Names returns map node_id -> node_name
func (ns Nodes) Names() (names map[NodeID]NodeName) {
	names = make(map[NodeID]NodeName, len(ns))
	for _, n := range ns {
		names[n.ID] = n.Name
	}
	return
}

func (ns Nodes) Has(name NodeName) (ok bool) {
	_, ok = ns.NodeByName(name)
	return
}
