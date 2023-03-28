package config

import (
	"fmt"
	"io"
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
	// StopCommand to start the node.
	StopCommand string `json:"stop_command" yaml:"stop_command" mapstructure:"stop_command"`

	// internals
	Command *exec.Cmd `json:"-" yaml:"-" mapstructure:"-"`
}

// Start the Node.
func (n *Node) Start(logsDir string, env map[string]string) (err error) {
	if n.WorkDir == "" {
		n.WorkDir = "."
	}
	var (
		ss      = strings.Fields(n.StartCommand)
		command string
	)
	command = ss[0]
	//if filepath.Base(command) != command {
	// 	command = "./" + filepath.Join(n.WorkDir, command)
	// }
	var cmd = exec.Command(command, ss[1:]...)
	cmd.Dir = n.WorkDir
	if n.Env != "" {
		cmd.Env = append(os.Environ(), strings.Split(n.Env, ",")...)
	}

	for key, value := range env {
		pair := key + "=" + value
		cmd.Env = append(cmd.Env, pair)
	}

	logsDir = filepath.Join(logsDir, string(n.Name))
	if err = os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("creating logs directory %s: %v", logsDir, err)
	}

	cmd.Stdout, err = os.Create(filepath.Join(logsDir, "stdout.log"))
	if err != nil {
		return fmt.Errorf("creating STDOUT file: %v", err)
	}

	cmd.Stderr, err = os.Create(filepath.Join(logsDir, "stderr.log"))
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
		if err := n.Command.Process.Kill(); err != nil {
			return err
		}
	}
	n.Command = nil
	return
}

func (n *Node) IsStarted() bool {
	return n.Command != nil
}

// Stop interrupts command and waits it. Then it closes STDIN and STDOUT
// files (logs).
func (n *Node) Stop() (err error) {
	startCmd := n.Command
	if startCmd == nil {
		return fmt.Errorf("command %v not started", n.Name)
	}
	if err = n.Kill(); err != nil {
		return fmt.Errorf("command %v: kill: %v", n.Name, err)
	}
	if stdin, ok := startCmd.Stdin.(*os.File); ok {
		if err := stdin.Close(); err != nil {
			return err
		}
	}
	if stderr, ok := startCmd.Stderr.(*os.File); ok {
		if err := stderr.Close(); err != nil {
			return err
		}
	}

	if n.WorkDir == "" {
		n.WorkDir = "."
	}
	var (
		ss      = strings.Fields(n.StopCommand)
		command string
	)
	command = ss[0]
	var cmd = exec.Command(command, ss[1:]...)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stop command %v: failed creating error pipe: %+v", n.Name, err)
	}

	cmd.Dir = n.WorkDir
	if n.Env != "" {
		cmd.Env = append(os.Environ(), strings.Split(n.Env, ",")...)
	}

	if err = cmd.Start(); err != nil {
		return fmt.Errorf("stop command %v: run: %+v", n.Name, err)
	}

	if err = cmd.Wait(); err != nil {
		slurp, _ := io.ReadAll(stderr)
		if slurp != nil {
			fmt.Printf("%s\n", slurp)
		}
		return fmt.Errorf("stop command %v: wait: %+v", n.Name, err)
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
