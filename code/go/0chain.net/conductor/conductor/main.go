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

func main() {
	log.Print("start the conductor")

	var (
		configFile string = "conductor.yaml"
		verbose    bool   = true
	)
	flag.StringVar(&configFile, "config", configFile, "configurations file")
	flag.BoolVar(&verbose, "verbose", verbose, "verbose output")
	flag.Parse()

	log.Print("read configurations file: ", configFile)
	var (
		conf = readConfig(configFile)
		r    Runner
		err  error
	)

	log.Print("create worker instance")
	r.conf = conf
	r.verbose = verbose
	if r.server, err = conductrpc.NewServer(conf.Bind); err != nil {
		log.Fatal("[ERR]", err)
	}

	log.Print("(rpc) start listening on:", conf.Bind)
	go func() {
		if err := r.server.Serve(); err != nil {
			log.Fatal("staring RPC server:", err)
		}
	}()
	defer r.server.Close()

	r.waitNodes = make(map[config.NodeID]struct{})
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
	server  *conductrpc.Server
	conf    *config.Config
	verbose bool

	// state

	lastVCRound Round // last view change round

	// wait for
	waitPhase              config.WaitPhase              //
	waitViewChange         config.WaitViewChange         //
	waitNodes              map[config.NodeID]struct{}    // (start a node)
	waitRound              config.WaitRound              //
	waitContributeMPK      config.WaitContributeMpk      //
	waitShareSignsOrShares config.WaitShareSignsOrShares //
	waitAdd                config.WaitAdd                // add_miner, add_sharder
	// timeout and monitor
	timer   *time.Timer // waiting timer
	monitor NodeID      // monitor node

	// remembered rounds: name -> round number
	rounds map[config.RoundName]config.Round // named rounds (the remember_round)
}

func (r *Runner) isWaiting() (tm *time.Timer, ok bool) {
	tm = r.timer

	switch {
	case len(r.waitNodes) > 0:
		log.Printf("wait for %d nodes", len(r.waitNodes))
		return tm, true
	case !r.waitRound.IsZero():
		return tm, true
	case !r.waitPhase.IsZero():
		return tm, true
	case !r.waitContributeMPK.IsZero():
		return tm, true
	case !r.waitShareSignsOrShares.IsZero():
		return tm, true
	case !r.waitViewChange.IsZero():
		return tm, true
	case !r.waitAdd.IsZero():
		return tm, true
	}

	return tm, false
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

func (r *Runner) printViewChange(vce *conductrpc.ViewChangeEvent) {
	if !r.verbose {
		return
	}
	log.Print(" [INF] VC ", vce.Round)
	log.Print(" [INF] VC MB miners:")
	for _, mn := range vce.Miners {
		var n, ok = r.conf.Nodes.NodeByID(mn)
		if !ok {
			log.Print("   - ", mn, " (unknown)")
			continue
		}
		log.Print("   - ", n.Name)
	}
	log.Print(" [INF] VC MB sharders:")
	for _, sh := range vce.Sharders {
		var n, ok = r.conf.Nodes.NodeByID(sh)
		if !ok {
			log.Print("   - ", sh, " (unknown)")
			continue
		}
		log.Print("   - ", n.Name)
	}
}

func (r *Runner) acceptViewChange(vce *conductrpc.ViewChangeEvent) (err error) {
	if vce.Sender != r.monitor {
		return // not the monitor node
	}
	r.printViewChange(vce) // if verbose
	var sender, ok = r.conf.Nodes.NodeByID(vce.Sender)
	if !ok {
		return fmt.Errorf("unknown node %q sends view change", vce.Sender)
	}
	log.Println("view change:", vce.Round, sender.Name)
	// don't wait a VC
	if r.waitViewChange.IsZero() {
		r.lastVCRound = vce.Round // keep last round number
		return
	}
	// remember the round
	if rrn := r.waitViewChange.RememberRound; rrn != "" {
		log.Printf("[OK] remember round %q: %d", rrn, vce.Round)
		r.rounds[r.waitViewChange.RememberRound] = vce.Round
	}
	var emb = r.waitViewChange.ExpectMagicBlock
	if emb.IsZero() {
		r.lastVCRound = vce.Round                  // keep last round number
		r.waitViewChange = config.WaitViewChange{} // reset
		return                                     // nothing more is here
	}
	if rnan := emb.RoundNextVCAfter; rnan != "" {
		var rna, ok = r.rounds[rnan]
		if !ok {
			return fmt.Errorf("unknown round name: %q", rnan)
		}
		var vcr = vce.Round // VC round
		if vcr != r.conf.ViewChange+rna {
			return fmt.Errorf("VC expected at %d, but given at %d",
				r.conf.ViewChange+rna, vcr)
		}
		// ok, accept
	} else if emb.Round != 0 && vce.Round != emb.Round {
		return fmt.Errorf("VC expected at %d, but given at %d",
			emb.Round, vce.Round)
	}
	if len(emb.Miners) == 0 && len(emb.Sharders) == 0 {
		r.lastVCRound = vce.Round                  // keep the last VC round
		r.waitViewChange = config.WaitViewChange{} // reset
		return                                     // doesn't check MB for nodes
	}
	// check for nodes

	var miners, sharders []NodeID
	if miners, err = r.toIDs(emb.Miners); err != nil {
		return fmt.Errorf("unknown miner: %v", err)
	}
	if sharders, err = r.toIDs(emb.Sharders); err != nil {
		return fmt.Errorf("unknown sharder: %v", err)
	}

	var okm, oks bool

	// check miners
	if okm = isEqual(miners, vce.Miners); !okm {
		fmt.Println("[ERR] expected miners list:")
		r.printNodes(miners)
		fmt.Println("[ERR] got miners")
		r.printNodes(vce.Miners)
	}

	// check sharders
	if oks = isEqual(sharders, vce.Sharders); !oks {
		fmt.Println("[ERR] expected sharders list:")
		r.printNodes(sharders)
		fmt.Println("[ERR] got sharders")
		r.printNodes(vce.Sharders)
	}

	if !okm || !oks {
		return fmt.Errorf("unexpected MB miners/sharders (see logs)")
	}

	log.Println("[OK] view change", vce.Round)

	r.lastVCRound = vce.Round                  // keep the last VC round
	r.waitViewChange = config.WaitViewChange{} // reset
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
	if r.verbose {
		log.Print(" [INF] phase ", pe.Phase.String(), " ", n.Name)
	}
	if r.waitPhase.IsZero() {
		return // doesn't wait for a phase
	}
	if r.waitPhase.Phase != pe.Phase {
		return // not this phase
	}
	var vcr Round
	if vcrn := r.waitPhase.ViewChangeRound; vcrn != "" {
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
	r.waitPhase = config.WaitPhase{} // reset
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

	if r.verbose {
		log.Print(" [INF] add_mienr ", added.Name)
	}

	if r.waitAdd.IsZero() {
		return // doesn't wait for a node
	}

	println(fmt.Sprint("[B] W ADD MIENRS: ", r.waitAdd.Miners), "{")
	if r.waitAdd.TakeMiner(added.Name) {
		log.Print("[OK] add_miner ", added.Name)
	}
	println(fmt.Sprint("[A] W ADD MIENRS: ", r.waitAdd.Miners), "}")
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

	if r.verbose {
		log.Print(" [INF] add_sharder ", added.Name)
	}

	if r.waitAdd.IsZero() {
		return // doesn't wait for a node
	}

	if r.waitAdd.TakeSharder(added.Name) {
		log.Print("[OK] add_sharder ", added.Name)
	}
	return
}

func (r *Runner) acceptNodeReady(nodeID NodeID) (err error) {
	if _, ok := r.waitNodes[nodeID]; !ok {
		var n, ok = r.conf.Nodes.NodeByID(nodeID)
		if !ok {
			return fmt.Errorf("unexpected and unknown node: %s", nodeID)
		}
		return fmt.Errorf("unexpected node: %s (%s)", n.Name, nodeID)
	}
	delete(r.waitNodes, nodeID)
	var n, ok = r.conf.Nodes.NodeByID(nodeID)
	if !ok {
		return fmt.Errorf("unknown node: %s", nodeID)
	}
	log.Println("[OK] node ready", nodeID, n.Name)
	return
}

func (r *Runner) acceptRound(re *conductrpc.RoundEvent) (err error) {
	if re.Sender != r.monitor {
		return // not the monitor node
	}

	var _, ok = r.conf.Nodes.NodeByID(re.Sender)
	if !ok {
		return fmt.Errorf("unknown 'round' sender: %s", re.Sender)
	}
	if r.verbose {
		// log.Print(" [INF] round ", re.Round, " ", n.Name)
	}
	if r.waitRound.IsZero() {
		return // doesn't wait for a round
	}

	switch {
	case r.waitRound.Round > re.Round:
		return // not this round
	case r.waitRound.Round == re.Round:
		log.Print("[OK] accept round", re.Round)
		r.waitRound.Round = 0 // doesn't wait anymore
	case r.waitRound.Round < re.Round:
		return fmt.Errorf("missing round: %d, got %d", r.waitRound, re.Round)
	}

	return
}

func (r *Runner) acceptContributeMPK(cmpke *conductrpc.ContributeMPKEvent) (
	err error) {

	if cmpke.Sender != r.monitor {
		return // not the monitor node
	}

	var (
		miner *config.Node
		ok    bool
	)
	_, ok = r.conf.Nodes.NodeByID(cmpke.Sender)
	if !ok {
		return fmt.Errorf("unknown 'c mpk' sender: %s", cmpke.Sender)
	}
	miner, ok = r.conf.Nodes.NodeByID(cmpke.MinerID)
	if !ok {
		return fmt.Errorf("unknown 'c mpk' miner: %s", cmpke.MinerID)
	}

	if r.verbose {
		log.Print(" [INF] contribute mpk ", miner.Name)
	}

	if r.waitContributeMPK.IsZero() {
		return // doesn't wait for a contribute MPK
	}

	if r.waitContributeMPK.MinerID != cmpke.MinerID {
		return // not the miner waiting for
	}

	log.Print("[OK] accept contribute MPK", miner.Name)
	r.waitContributeMPK = config.WaitContributeMpk{}

	return
}

func (r *Runner) acceptShareOrSignsShares(
	sosse *conductrpc.ShareOrSignsSharesEvent) (err error) {

	if sosse.Sender != r.monitor {
		return // not the monitor node
	}

	var (
		miner *config.Node
		ok    bool
	)
	_, ok = r.conf.Nodes.NodeByID(sosse.Sender)
	if !ok {
		return fmt.Errorf("unknown 'soss' sender: %s", sosse.Sender)
	}
	miner, ok = r.conf.Nodes.NodeByID(sosse.MinerID)
	if !ok {
		return fmt.Errorf("unknown 'soss' miner: %s", sosse.MinerID)
	}

	if r.verbose {
		log.Print(" [INF] share or sign shares ", miner.Name)
	}

	if r.waitShareSignsOrShares.IsZero() {
		return // doesn't wait for a soss
	}

	if r.waitShareSignsOrShares.MinerID != sosse.MinerID {
		return // not the miner waiting for
	}

	log.Print("[OK] accept share or signs shares", miner.Name)
	r.waitShareSignsOrShares = config.WaitShareSignsOrShares{}

	return
}

func (r *Runner) stopAll() {
	log.Print("stop all nodes")
	for _, n := range r.conf.Nodes {
		log.Printf("stop %s", n.Name)
		n.Stop()
	}
}

func (r *Runner) killAll() {
	log.Print("kill all nodes")
	for _, n := range r.conf.Nodes {
		log.Printf("kill %s", n.Name)
		n.Kill()
	}
}

func (r *Runner) proceedWaiting() (err error) {
	for tm, ok := r.isWaiting(); ok; tm, ok = r.isWaiting() {
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
		case re := <-r.server.OnRound():
			err = r.acceptRound(re)
		case cmpke := <-r.server.OnContributeMPK():
			err = r.acceptContributeMPK(cmpke)
		case sosse := <-r.server.OnShareOrSignsShares():
			err = r.acceptShareOrSignsShares(sosse)
		case <-tm.C:
			return fmt.Errorf("timeout error")
		}
		if err != nil {
			return
		}
	}
	return
}

// Run the tests.
func (r *Runner) Run() (err error) {

	log.Println("start testing...")
	defer log.Println("end of testing")

	// stop all nodes after all
	defer r.stopAll()

	// for every enabled set
	for _, set := range r.conf.Sets {
		if !r.conf.IsEnabled(&set) {
			continue
		}
		log.Print("...........................................................")
		log.Print("start set ", set.Name)
		log.Print("...........................................................")
		// for every test case
		for i, testCase := range r.conf.TestsOfSet(&set) {
			log.Print("=======================================================")
			log.Printf("%d %s test case", i, testCase.Name)
			for j, f := range testCase.Flow {
				log.Print("---------------------------------------------------")
				log.Printf("  %d/%d step", i, j)
				// execute
				if err = f.Execute(r); err != nil {
					return // fatality
				}
				if err = r.proceedWaiting(); err != nil {
					return
				}
			}
			log.Printf("end of %d %s test case", i, testCase.Name)
		}
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
	return r.conf.CleanupBC()
}

// Start nodes, or start and lock them.
func (r *Runner) Start(names []NodeName, lock bool,
	tm time.Duration) (err error) {

	if r.verbose {
		log.Print(" [INF] start ", names)
	}

	r.setupTimeout(tm)

	// start nodes
	for _, name := range names {
		var n, ok = r.conf.Nodes.NodeByName(name) //
		if !ok {
			return fmt.Errorf("(start): unknown node: %q", name)
		}
		r.server.AddNode(n.ID, lock)   // lock list
		r.waitNodes[n.ID] = struct{}{} // wait list
		if err = n.Start(r.conf.Logs); err != nil {
			return fmt.Errorf("starting %s: %v", n.Name, err)
		}
	}
	return
}

func (r *Runner) WaitViewChange(vc config.WaitViewChange, tm time.Duration) (
	err error) {

	if r.verbose {
		log.Print(" [INF] wait for VC ", vc.ExpectMagicBlock.Round)
	}

	r.setupTimeout(tm)
	r.waitViewChange = vc
	return
}

func (r *Runner) WaitPhase(pe config.WaitPhase, tm time.Duration) (err error) {

	if r.verbose {
		log.Print(" [INF] wait phase ", pe.Phase.String())
	}

	r.setupTimeout(tm)
	r.waitPhase = pe
	return
}

func (r *Runner) Unlock(names []NodeName, tm time.Duration) (err error) {

	if r.verbose {
		log.Print(" [INF] unlock ", names)
	}

	r.setupTimeout(0)
	for _, name := range names {
		var n, ok = r.conf.Nodes.NodeByName(name) //
		if !ok {
			return fmt.Errorf("(unlock): unknown node: %q", name)
		}
		log.Print("unlock ", n.Name)
		if err = r.server.UnlockNode(n.ID); err != nil {
			return
		}
	}
	return
}

func (r *Runner) Stop(names []NodeName, tm time.Duration) (err error) {

	if r.verbose {
		log.Print(" [INF] stop ", names)
	}

	for _, name := range names {
		var n, ok = r.conf.Nodes.NodeByName(name) //
		if !ok {
			return fmt.Errorf("(stop): unknown node: %q", name)
		}
		log.Print("stopping ", n.Name, "...")
		if err := n.Stop(); err != nil {
			log.Printf("stopping %s: %v", n.Name, err)
			n.Kill()
		}
		log.Print(n.Name, " stopped")
	}
	return
}

func (r *Runner) WaitRound(wr config.WaitRound, tm time.Duration) (err error) {

	if r.verbose {
		log.Print(" [INF] wait for round ", wr.Round)
	}

	r.setupTimeout(tm)
	r.waitRound = wr
	return
}

func (r *Runner) WaitContributeMpk(wcmpk config.WaitContributeMpk,
	tm time.Duration) (err error) {

	if r.verbose {
		var miner, ok = r.conf.Nodes.NodeByID(wcmpk.MinerID)
		if ok {
			log.Print(" [INF] wait for contribute MPK by ", miner.Name)
		} else {
			log.Print(" [INF] wait for contribute MPK", wcmpk.MinerID,
				"(unknown)")
		}
	}

	r.setupTimeout(tm)
	r.waitContributeMPK = wcmpk
	return
}

func (r *Runner) WaitShareSignsOrShares(ssos config.WaitShareSignsOrShares,
	tm time.Duration) (err error) {

	if r.verbose {
		var miner, ok = r.conf.Nodes.NodeByID(ssos.MinerID)
		if ok {
			log.Print(" [INF] wait for SOSS by ", miner.Name)
		} else {
			log.Print(" [INF] wait for SOSS by ", ssos.MinerID, "(unknown)")
		}
	}

	r.setupTimeout(tm)
	r.waitShareSignsOrShares = ssos
	return
}

func (r *Runner) WaitAdd(wadd config.WaitAdd, tm time.Duration) (err error) {

	if r.verbose {
		log.Printf(" [INF] wait add miners: %s, sharders: %s",
			wadd.Miners, wadd.Sharders)
	}

	r.setupTimeout(tm)
	r.waitAdd = wadd
	return
}

func (r *Runner) SendShareOnly(miner NodeID, only []NodeID) (err error) {
	r.server.SetSendShareOnly(miner, only)
	return
}

func (r *Runner) SendShareBad(miner NodeID, bad []NodeID) (err error) {
	r.server.SetSendShareBad(miner, bad)
	return
}
