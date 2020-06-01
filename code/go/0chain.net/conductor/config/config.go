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

// MagicBlock represents expected magic block.
type MagicBlock struct {
	// Round ignored if it's zero. If set a positive value, then this
	// round is expected.
	Round Round `json:"round" yaml:"round" mapstructure:"round"`
	// RoundNextVCAfter used in combination with wait_view_change.remember_round
	// that remember round with some name. This directive expects next VC round
	// after the remembered one. For example, if round 340 has remembered as
	// "enter_miner5", then "round_next_vc_after": "enter_miner5", expects
	// 500 round (next VC after the remembered round). Empty string ignored.
	RoundNextVCAfter RoundName `json:"round_next_vc_after" yaml:"round_next_vc_after" mapstructure:"round_next_vc_after"`
	// Sharders expected in MB.
	Sharders []NodeName `json:"sharders" yaml:"sharders" mapstructure:"sharders"`
	// Miners expected in MB.
	Miners []NodeName `json:"miners" yaml:"miners" mapstructure:"miners"`
}

// IsZero returns true if the MagicBlock is empty.
func (mb *MagicBlock) IsZero() bool {
	return mb.Round == 0 &&
		mb.RoundNextVCAfter == "" &&
		len(mb.Sharders) == 0 &&
		len(mb.Miners) == 0
}

// WaitViewChange flow configuration.
type WaitViewChange struct {
	RememberRound    RoundName  `json:"remember_round" yaml:"remember_round" mapstructure:"remember_round"`
	ExpectMagicBlock MagicBlock `json:"expect_magic_block" yaml:"expect_magic_block" mapstructure:"expect_magic_block"`
}

// IsZero returns true if the ViewChagne is empty.
func (vc *WaitViewChange) IsZero() bool {
	return vc.RememberRound == "" &&
		vc.ExpectMagicBlock.IsZero()
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

// WaitPhase flow configuration.
type WaitPhase struct {
	// Phase to wait for (number).
	Phase Phase `json:"phase" yaml:"phase" mapstructure:"phase"`
	// ViewChangeRound after which the phase expected (and before next VC),
	// value can be an empty string for any VC.
	ViewChangeRound RoundName `json:"view_change_round" yaml:"view_change_round" mapstructure:"view_change_round"`
}

// IsZero returns true if the WaitPhase is empty.
func (wp *WaitPhase) IsZero() bool {
	return wp.Phase == 0 && wp.ViewChangeRound == ""
}

// WaitRound waits a round.
type WaitRound struct {
	Round Round `json:"round" yaml:"round" mapstructure:"round"`
}

func (wr *WaitRound) IsZero() bool {
	return wr.Round == 0
}

// WaitContibuteMpk wait for MPK contributing of a node.
type WaitContributeMpk struct {
	MinerID NodeID `json:"miner_id" yaml:"miner_id" mapstructure:"miner_id"`
}

func (wcm *WaitContributeMpk) IsZero() bool {
	return wcm.MinerID == ""
}

// WaitShareSignsOrShares waits for SOSS of a node.
type WaitShareSignsOrShares struct {
	MinerID NodeID `json:"miner_id" yaml:"miner_id" mapstructure:"miner_id"`
}

func (wssos *WaitShareSignsOrShares) IsZero() bool {
	return wssos.MinerID == ""
}

// WaitAdd used to wait for add_mienr and add_sharder SC calls.
type WaitAdd struct {
	Miners   []NodeName `json:"miners" yaml:"miners" mapstructure:"miners"`
	Sharders []NodeName `json:"sharders" yaml:"sharders" mapstructure:"sharders"`
}

func (wa *WaitAdd) IsZero() bool {
	return len(wa.Miners) == 0 && len(wa.Sharders) == 0
}

func (wa *WaitAdd) TakeMiner(name NodeName) (ok bool) {
	for i, minerName := range wa.Miners {
		if minerName == name {
			wa.Miners = append(wa.Miners[:i], wa.Miners[i+1:]...)
			return true
		}
	}
	return // false
}

func (wa *WaitAdd) TakeSharder(name NodeName) (ok bool) {
	for i, sharderName := range wa.Sharders {
		if sharderName == name {
			wa.Sharders = append(wa.Sharders[:i], wa.Sharders[i+1:]...)
			return true
		}
	}
	return
}

// Executor used by a Flow to perform a flow directive.
type Executor interface {
	SetMonitor(name NodeName) (err error)
	Start(names []NodeName, lock bool, timeout time.Duration) (err error)
	WaitViewChange(vc WaitViewChange, timeout time.Duration) (err error)
	WaitPhase(wp WaitPhase, timeout time.Duration) (err error)
	WaitRound(wr WaitRound, timeout time.Duration) (err error)
	WaitContributeMpk(wcmpk WaitContributeMpk, timeout time.Duration) (err error)
	WaitShareSignsOrShares(ssos WaitShareSignsOrShares, timeout time.Duration) (err error)
	WaitAdd(wadd WaitAdd, timeout time.Duration) (err error)
	Unlock(names []NodeName, timeout time.Duration) (err error)
	Stop(names []NodeName, timeout time.Duration) (err error)
	CleanupBC(timeout time.Duration) (err error)
	SendShareOnly(miner NodeID, only []NodeID) (err error)
	SendShareBad(miner NodeID, bad []NodeID) (err error)
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

func getNodeNames(val interface{}) (ss []NodeName, ok bool) {
	switch tt := val.(type) {
	case string:
		return []NodeName{NodeName(tt)}, true
	case []interface{}:
		ss = make([]NodeName, 0, len(tt))
		for _, t := range tt {
			if ts, ok := t.(string); ok {
				ss = append(ss, NodeName(ts))
			} else {
				return nil, false
			}
		}
		return ss, true
	case []string:
		ss = make([]NodeName, 0, len(tt))
		for _, t := range tt {
			ss = append(ss, NodeName(t))
		}
		return ss, true
	}
	return // nil, false
}

func (f Flow) setMonitor(ex Executor, val interface{}, tm time.Duration) (
	err error) {

	if ss, ok := getNodeNames(val); ok && len(ss) == 1 {
		return ex.SetMonitor(ss[0])
	}
	return fmt.Errorf("invalid 'set_monitor' argument type: %T", val)
}

func (f Flow) waitViewChange(ex Executor, val interface{}, tm time.Duration) (
	err error) {

	var vc WaitViewChange
	if err = mapstructure.Decode(val, &vc); err != nil {
		return fmt.Errorf("invalid 'wait_view_change' argument type: %T, "+
			"decoding error: %v", val, err)
	}
	return ex.WaitViewChange(vc, tm)
}

func (f Flow) start(name string, ex Executor, val interface{}, lock bool,
	tm time.Duration) (err error) {

	if ss, ok := getNodeNames(val); ok {
		return ex.Start(ss, lock, tm)
	}
	return fmt.Errorf("invalid '%s' argument type: %T", name, val)
}

func (f Flow) waitPhase(ex Executor, val interface{}, tm time.Duration) (
	err error) {

	type waitPhase struct {
		Phase           string    `mapstructure:"phase"`
		ViewChangeRound RoundName `mapstructure:"view_change_round"`
	}
	var wps waitPhase
	if err = mapstructure.Decode(val, &wps); err != nil {
		return fmt.Errorf("invalid 'wait_phase' argument type: %T, "+
			"decoding error: %v", val, err)
	}
	var wp WaitPhase
	if wp.Phase, err = ParsePhase(wps.Phase); err != nil {
		return fmt.Errorf("parsing phase: %v", err)
	}
	return ex.WaitPhase(wp, tm)
}

func (f Flow) unlock(ex Executor, val interface{}, tm time.Duration) (
	err error) {

	if ss, ok := getNodeNames(val); ok {
		return ex.Unlock(ss, tm)
	}
	return fmt.Errorf("invalid 'unlock' argument type: %T", val)
}

func (f Flow) stop(ex Executor, val interface{}, tm time.Duration) (
	err error) {

	if ss, ok := getNodeNames(val); ok {
		return ex.Stop(ss, tm)
	}
	return fmt.Errorf("invalid 'stop' argument type: %T", val)
}

func (f Flow) waitRound(ex Executor, val interface{}, tm time.Duration) (
	err error) {

	var wr WaitRound
	if err = mapstructure.Decode(val, &wr); err != nil {
		return fmt.Errorf("decoding 'wait_round': %v", err)
	}
	return ex.WaitRound(wr, tm)
}

func (f Flow) waitContributeMpk(ex Executor, val interface{},
	tm time.Duration) (err error) {

	var wcmpk WaitContributeMpk
	if err = mapstructure.Decode(val, &wcmpk); err != nil {
		return fmt.Errorf("decoding 'wait_contribute_mpk': %v", err)
	}
	return ex.WaitContributeMpk(wcmpk, tm)
}

func (f Flow) waitShareSignsOrShares(ex Executor, val interface{},
	tm time.Duration) (err error) {

	var wsoss WaitShareSignsOrShares
	if err = mapstructure.Decode(val, &wsoss); err != nil {
		return fmt.Errorf("decoding 'wait_share_signs_or_shares': %v", err)
	}
	return ex.WaitShareSignsOrShares(wsoss, tm)
}

func (f Flow) waitAdd(ex Executor, val interface{}, tm time.Duration) (
	err error) {

	var wa WaitAdd
	if err = mapstructure.Decode(val, &wa); err != nil {
		return fmt.Errorf("decoding 'wait_add': %v", err)
	}
	return ex.WaitAdd(wa, tm)
}

func (f Flow) sendShareOnly(ex Executor, val interface{}, tm time.Duration) (
	err error) {

	type sendShareOnly struct {
		Miner NodeID   `mapstructure:"miner"`
		Only  []NodeID `mapstructure:"only"`
	}

	var sso sendShareOnly
	if err = mapstructure.Decode(val, &sso); err != nil {
		return fmt.Errorf("invalid 'send_share_only' argument type: %T, "+
			"decoding error: %v", val, err)
	}

	if sso.Miner == "" {
		return errors.New("'seed_share_only' missing miner ID")
	}

	return ex.SendShareOnly(sso.Miner, sso.Only)
}

func (f Flow) sendShareBad(ex Executor, val interface{}, tm time.Duration) (
	err error) {

	type sendShareBad struct {
		Miner NodeID   `mapstructure:"miner"`
		Bad   []NodeID `mapstructure:"bad"`
	}

	var ssb sendShareBad
	if err = mapstructure.Decode(val, &ssb); err != nil {
		return fmt.Errorf("invalid 'send_share_bad' argument type: %T, "+
			"decoding error: %v", val, err)
	}

	if ssb.Miner == "" {
		return errors.New("'seed_share_bad' missing miner ID")
	}

	return ex.SendShareBad(ssb.Miner, ssb.Bad)
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
	case "set_monitor":
		return f.setMonitor(ex, val, tm)
	case "cleanup_bc":
		return ex.CleanupBC(tm)
	case "start":
		return f.start(name, ex, val, false, tm)
	case "wait_view_change":
		return f.waitViewChange(ex, val, tm)
	case "start_lock":
		return f.start(name, ex, val, true, tm)
	case "wait_phase":
		return f.waitPhase(ex, val, tm)
	case "unlock":
		return f.unlock(ex, val, tm)
	case "stop":
		return f.stop(ex, val, tm)
	case "wait_round":
		return f.waitRound(ex, val, tm)
	case "wait_contribute_mpk":
		return f.waitContributeMpk(ex, val, tm)
	case "wait_share_signs_or_shares":
		return f.waitShareSignsOrShares(ex, val, tm)
	case "wait_add":
		return f.waitAdd(ex, val, tm)
	case "send_share_only":
		return f.sendShareOnly(ex, val, tm)
	case "send_share_bad":
		return f.sendShareBad(ex, val, tm)
	default:
		return fmt.Errorf("unknown flow directive: %q", name)
	}
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
