// Package conductor represents 0chain BC testing conductor that
// maintain BC joining and leaving. It's introduced to test some
// view change cases where a miner comes up and goes down.
//
// The conductor uses RPC to control nodes. It starts and stops
// miners and sharders. It controls their lifecycle and entire
// system state. There is internal BC monitoring to generate
// events depending BC state (view change, view change phase,
// nodes registration, etc).
//
// All the cases uses b0magicBlock_4_miners_1_sharder.json where
// there is 1 genesis sharder and 4 genesis miners. Also, there is
// 2 non-genesis sharders and 1 non-genesis miner.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"io"
	"net"

	"gopkg.in/yaml.v2"

	"0chain.net/conductor/conductrpc"
	"0chain.net/conductor/config"

	"github.com/kr/pretty"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

// type aliases
type (
	NodeID    = config.NodeID
	NodeName  = config.NodeName
	Round     = config.Round
	RoundName = config.RoundName
)

func fuckingEcho() {
	println("FUCKING ECHO START")
	defer println("FUCKING ECHO STOP")
	l, err := net.Listen("tcp", "0.0.0.0:1515")
	if err != nil {
		log.Fatal(err)
	}
	println("FUCKING ECHO START LISTENER")
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Print("FUCKING ECHO ACCEPTING: ", err)
			continue
		}
		println("FUCKING ECHO ACCEPTED")
		go func(conn net.Conn) {
			defer conn.Close()
			go io.Copy(os.Stdout, conn)
			conn.Write([]byte("I AM FUCKING ECHO SERVER!!!!! FUCK YOU!!!!\n"))
		}(conn)
	}
}

func main() {
	log.Print("start the conductor")

	go fuckingEcho()

	var configFile string = "conductor.yaml"
	flag.StringVar(&configFile, "config", configFile, "configurations file")
	flag.Parse()

	log.Print("read configurations file: ", configFile)
	var (
		conf = readConfig(configFile)
		r    Runner
		err  error
	)

	log.Print("create worker instance")
	r.conf = conf
	r.server = conductrpc.NewServer(conf.Bind)
	if r.runner, err = newRunner(conf.BindRunner); err != nil {
		log.Fatalf("creating runner listener: %v", err)
	}

	log.Print("(rpc) start listening on:", conf.Bind)
	go func() {
		if err := r.server.Serve(); err != nil {
			log.Fatal("staring RPC server:", err)
		}
	}()
	defer r.server.Close()

	r.nodes = make(map[config.NodeID]struct{})
	r.rounds = make(map[config.RoundName]config.Round)
	r.setupTimeout(0)

	if err = r.Run(); err != nil {
		log.Print("[ERR] ", err)
	}

	_ = pretty.Print
}

func readConfig(configFile string) (conf *config.Config) {
	conf = new(config.Config)
	var fl, err = os.Open(configFile)
	if err != nil {
		log.Fatalf("opening configurations file %s: %v", configFile, err)
	}
	defer fl.Close()
	if err = yaml.NewDecoder(fl).Decode(conf); err != nil {
		log.Fatalf("decoding configurations file %s: %v", configFile, err)
	}
	return
}

type Runner struct {
	server *conductrpc.Server
	conf   *config.Config
	runner *runner

	// state

	lastVCRound Round // last view change round

	// wait for
	phase      config.WaitPhase           // wait for a phase
	viewChange config.WaitViewChange      // wait for a view change
	nodes      map[config.NodeID]struct{} // wait starting nodes
	timer      *time.Timer                // waiting timer
	monitor    NodeID                     // monitor node

	// remembered rounds: name -> round number
	rounds map[config.RoundName]config.Round // named rounds (the remember_round)
}

func (r *Runner) isWaiting() (tm *time.Timer, ok bool) {
	return r.timer, !r.phase.IsZero() || !r.viewChange.IsZero() ||
		len(r.nodes) > 0
}

func (r *Runner) toIDs(names []NodeName) (ids []NodeID, err error) {
	ids = make([]NodeID, 0, len(names))
	for _, name := range names {
		var n, ok = r.conf.Nodes.NodeByName(name)
		if !ok {
			return nil, fmt.Errorf("unknown node %q", name)
		}
		ids = append(ids, n.ID)
	}
	return
}

func isEqual(a, b []NodeID) (ok bool) {
	if len(a) != len(b) {
		return false
	}
	var am = make(map[NodeID]struct{})
	for _, ax := range a {
		am[ax] = struct{}{}
	}
	if len(am) != len(a) {
		return false // duplicate node id
	}
	for _, bx := range b {
		if _, ok := am[bx]; !ok {
			return false
		}
		delete(am, bx)
	}
	return true
}

func (r *Runner) printNodes(list []NodeID) {
	for _, x := range list {
		var n, ok = r.conf.Nodes.NodeByID(x)
		if !ok {
			fmt.Println("  - ", x, "(unknown node)")
			continue
		}
		fmt.Println("  - ", n.Name, x)
	}
}

func (r *Runner) acceptViewChange(vce *conductrpc.ViewChangeEvent) (err error) {
	if vce.Sender != r.monitor {
		return // not the monitor node
	}
	var sender, ok = r.conf.Nodes.NodeByID(vce.Sender)
	if !ok {
		return fmt.Errorf("unknown node %q sends view change", vce.Sender)
	}
	log.Print("view change:", vce.Round, sender.Name)
	// don't wait a VC
	if r.viewChange.IsZero() {
		r.lastVCRound = vce.Round // keep last round number
		return
	}
	// remember the round
	if rrn := r.viewChange.RememberRound; rrn != "" {
		log.Printf("[OK] remember round %q: %d", rrn, vce.Round)
		r.rounds[r.viewChange.RememberRound] = vce.Round
	}
	var emb = r.viewChange.ExpectMagicBlock
	if emb.IsZero() {
		r.lastVCRound = vce.Round // keep last round number
		return                    // nothing more is here
	}
	if rnan := emb.RoundNextVCAfter; rnan != "" {
		var rna, ok = r.rounds[rnan]
		if !ok {
			return fmt.Errorf("unknown round name: %q", rnan)
		}
		var vcr = emb.Round // VC round
		if vcr != r.conf.ViewChange+rna {
			return fmt.Errorf("VC expected at %d, but given at %d",
				r.conf.ViewChange+rna, vcr)
		}
		// ok, accept
	}
	if len(emb.Miners) == 0 && len(emb.Sharders) == 0 {
		r.lastVCRound = vce.Round // keep the last VC round
		return                    // doesn't check MB for nodes
	}
	// check for nodes

	var miners, sharders []NodeID
	if miners, err = r.toIDs(emb.Miners); err != nil {
		return fmt.Errorf("unknown miner: %v", err)
	}
	if sharders, err = r.toIDs(emb.Sharders); err != nil {
		return fmt.Errorf("unknown sharder: %v", err)
	}

	// check miners
	if !isEqual(miners, vce.Miners) {
		fmt.Println("expected miners list:")
		r.printNodes(vce.Miners)
		fmt.Println("got miners")
		r.printNodes(miners)
	}

	// check sharders
	if !isEqual(sharders, vce.Sharders) {
		fmt.Println("expected sharders list:")
		r.printNodes(vce.Sharders)
		fmt.Println("got sharders")
		r.printNodes(sharders)
	}

	log.Print("[OK] view change", vce.Round)

	r.lastVCRound = vce.Round // keep the last VC round
	return
}

func (r *Runner) acceptPhase(pe *conductrpc.PhaseEvent) (err error) {
	if pe.Sender != r.monitor {
		return // not the monitor node
	}
	var n, ok = r.conf.Nodes.NodeByID(pe.Sender)
	if !ok {
		return fmt.Errorf("unknown 'phase' sender: %s", pe.Sender)
	}
	log.Print("phase: ", pe.Phase.String(), n.Name)
	if r.phase.IsZero() {
		return // doesn't wait for a phase
	}
	if r.phase.Phase != pe.Phase {
		return // not this phase
	}
	var vcr Round
	if vcrn := r.phase.ViewChangeRound; vcrn != "" {
		if vcr, ok = r.rounds[vcrn]; !ok {
			return fmt.Errorf("unknown view_change_round of phase: %s", vcrn)
		}
		if vcr < r.lastVCRound {
			return // wait one more view change
		}
		if vcr >= r.lastVCRound+r.conf.ViewChange {
			return fmt.Errorf("got phase %s, but after %s (%d) view change, "+
				"last known view change: %d", pe.Phase.String(), vcrn, vcr,
				r.lastVCRound)
		}
		// ok, accept it
	}
	log.Printf("[OK] accept phase %s by %s", pe.Phase.String(), n.Name)
	return
}

func (r *Runner) acceptAddMiner(addm *conductrpc.AddMinerEvent) (err error) {
	if addm.Sender != r.monitor {
		return // not the monitor node
	}
	var (
		sender, sok = r.conf.Nodes.NodeByID(addm.Sender)
		added, aok  = r.conf.Nodes.NodeByID(addm.MinerID)
	)
	if !sok {
		return fmt.Errorf("unexpected add_miner sender: %q", addm.Sender)
	}
	if !aok {
		return fmt.Errorf("unexpected miner %q added by add_miner of %q",
			addm.MinerID, sender.Name)
	}
	log.Printf("%s add_miner: %s", sender.Name, added.Name)
	return
}

func (r *Runner) acceptAddSharder(adds *conductrpc.AddSharderEvent) (err error) {
	if adds.Sender != r.monitor {
		return // not the monitor node
	}
	var (
		sender, sok = r.conf.Nodes.NodeByID(adds.Sender)
		added, aok  = r.conf.Nodes.NodeByID(adds.SharderID)
	)
	if !sok {
		return fmt.Errorf("unexpected add_sharder sender: %q", adds.Sender)
	}
	if !aok {
		return fmt.Errorf("unexpected sharder %q added by add_sharder of %q",
			adds.SharderID, sender.Name)
	}
	log.Printf("%s add_sharder: %s", sender.Name, added.Name)
	return
}

func (r *Runner) acceptNodeReady(nodeID NodeID) (err error) {
	if _, ok := r.nodes[nodeID]; !ok {
		var n, ok = r.conf.Nodes.NodeByID(nodeID)
		if !ok {
			return fmt.Errorf("unexpected and unknown node: %s", nodeID)
		}
		return fmt.Errorf("unexpected node: %s (%s)", n.Name, nodeID)
	}
	delete(r.nodes, nodeID)
	var n, ok = r.conf.Nodes.NodeByID(nodeID)
	if !ok {
		return fmt.Errorf("unknown node: %s", nodeID)
	}
	log.Print("[OK] node ready", nodeID, n.Name)
	return
}

func (r *Runner) stopAll() {
	log.Print("stop all nodes")
	for _, n := range r.conf.Nodes {
		log.Printf("stop %s", n.Name)
		r.runner.Stop(n)
	}
}

// Run the tests.
func (r *Runner) Run() (err error) {

	log.Println("start testing...")
	defer log.Println("end of testing")

	// stop all nodes after all
	defer r.stopAll()

	// for every test case
	for i, testCase := range r.conf.Tests {
		log.Printf("start %d %s test case", i, testCase.Name)
		for j, f := range testCase.Flow {
			log.Printf("  %d flow step", j)

			// proceed
			println("FUCK IS WAITING?")
			for tm, ok := r.isWaiting(); ok; {
				println("YES IT IS?")
				select {
				case vce := <-r.server.OnViewChange():
					err = r.acceptViewChange(vce)
				case pe := <-r.server.OnPhase():
					err = r.acceptPhase(pe)
				case addm := <-r.server.OnAddMiner():
					err = r.acceptAddMiner(addm)
				case adds := <-r.server.OnAddSharder():
					err = r.acceptAddSharder(adds)
				case nid := <-r.server.OnNodeReady():
					err = r.acceptNodeReady(nid)
				case <-tm.C:
					return fmt.Errorf("timeout error")
				}
				if err != nil {
					return
				}
			}

			// execute
			println("EXECUTE")
			if err = f.Execute(r); err != nil {
				return // fatality
			}
		}

		log.Printf("end of %d %s test case", i, testCase.Name)
	}

	return
}

//
// execute
//

func (r *Runner) setupTimeout(tm time.Duration) {
	r.timer = time.NewTimer(tm)
	if tm <= 0 {
		<-r.timer.C // drain zero timeout
	}
}

// SetMonitor for phases and view changes.
func (r *Runner) SetMonitor(name NodeName) (err error) {
	var n, ok = r.conf.Nodes.NodeByName(name)
	if !ok {
		return fmt.Errorf("unknown node: %s", name)
	}
	r.monitor = n.ID
	return // ok
}

// CleanupBC cleans up blockchain.
func (r *Runner) CleanupBC(tm time.Duration) (err error) {
	r.stopAll()
	r.runner.Clean()
	return // nil
}

// Start nodes, or start and lock them.
func (r *Runner) Start(names []NodeName, lock bool,
	tm time.Duration) (err error) {

	r.setupTimeout(tm)

	// start nodes
	for _, name := range names {
		var n, ok = r.conf.Nodes.NodeByName(name) //
		if !ok {
			return fmt.Errorf("(start): unknown node: %q", name)
		}
		r.server.AddNode(n.ID, lock)
		r.runner.Start(n)
		r.nodes[n.ID] = struct{}{} // wait list
	}
	return
}

func (r *Runner) WaitViewChange(vc config.WaitViewChange, tm time.Duration) (
	err error) {

	r.setupTimeout(tm)
	r.viewChange = vc
	return
}

func (r *Runner) WaitPhase(pe config.WaitPhase, tm time.Duration) (err error) {
	r.setupTimeout(tm)
	r.phase = pe
	return
}

func (r *Runner) Unlock(names []NodeName, tm time.Duration) (err error) {
	r.setupTimeout(0)
	for _, name := range names {
		var n, ok = r.conf.Nodes.NodeByName(name) //
		if !ok {
			return fmt.Errorf("(start): unknown node: %q", name)
		}
		log.Print("node", n.Name, "unlock")
		r.server.UnlockNode(n.ID)
	}
	return
}

func (r *Runner) Stop(names []NodeName, tm time.Duration) (err error) {
	for _, name := range names {
		var n, ok = r.conf.Nodes.NodeByName(name) //
		if !ok {
			return fmt.Errorf("(start): unknown node: %q", name)
		}
		log.Print("stop", n.Name)
		r.runner.Stop(n)
	}
	return
}
