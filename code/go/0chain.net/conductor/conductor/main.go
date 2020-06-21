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
		testsFile  string = "conductor.view-change-1.yaml"
		verbose    bool   = true
	)
	flag.StringVar(&configFile, "config", configFile, "configurations file")
	flag.StringVar(&testsFile, "tests", testsFile, "tests file")
	flag.BoolVar(&verbose, "verbose", verbose, "verbose output")
	flag.Parse()

	println("CONFIG FILE PATH:", configFile)
	println("TESTS FILE PATH:", testsFile)

	log.Print("read configurations files: ", configFile, ", ", testsFile)
	var (
		conf = readConfigs(configFile, testsFile)
		r    Runner
		err  error
	)

	if len(conf.Nodes) == 0 {
		panic("NO NODES")
	}

	for _, n := range conf.Nodes {
		println(" - NODE", n.Name)
	}

	log.Print("create worker instance")
	r.conf = conf
	r.verbose = verbose
	r.server, err = conductrpc.NewServer(conf.Bind, conf.Nodes.Names())
	if err != nil {
		log.Fatal("[ERR]", err)
	}

	log.Print("(rpc) start listening on:", conf.Bind)
	go func() {
		if err := r.server.Serve(); err != nil {
			log.Fatal("staring RPC server:", err)
		}
	}()
	defer r.server.Close()

	r.waitNodes = make(map[config.NodeName]struct{})
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

func readConfigs(configFile, testsFile string) (conf *config.Config) {
	conf = readConfig(configFile)
	var tests = readConfig(testsFile)
	conf.Tests = tests.Tests   // set
	conf.Enable = tests.Enable // set
	conf.Sets = tests.Sets     // set
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
	waitNodes              map[config.NodeName]struct{}  // (start a node)
	waitRound              config.WaitRound              //
	waitContributeMPK      config.WaitContributeMpk      //
	waitShareSignsOrShares config.WaitShareSignsOrShares //
	waitAdd                config.WaitAdd                // add_miner, add_sharder
	waitNoProgressUntil    time.Time                     //
	waitNoViewChainge      config.WaitNoViewChainge      // no VC expected
	// timeout and monitor
	timer   *time.Timer // waiting timer
	monitor NodeName    // monitor node

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
	case !r.waitNoProgressUntil.IsZero():
		return tm, true
	case !r.waitNoViewChainge.IsZero():
		return tm, true
	}

	return tm, false
}

// func (r *Runner) toIDs(names []NodeName) (ids []NodeID, err error) {
// 	ids = make([]NodeID, 0, len(names))
// 	for _, name := range names {
// 		var n, ok = r.conf.Nodes.NodeByName(name)
// 		if !ok {
// 			return nil, fmt.Errorf("unknown node %q", name)
// 		}
// 		ids = append(ids, n.ID)
// 	}
// 	return
// }

// is equal set of elements in both slices (an order doesn't matter)
func isEqual(a, b []NodeName) (ok bool) {
	if len(a) != len(b) {
		return false
	}
	var am = make(map[NodeName]struct{})
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

func (r *Runner) printNodes(list []NodeName) {
	for _, x := range list {
		var n, ok = r.conf.Nodes.NodeByName(x)
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
		if !r.conf.Nodes.Has(mn) {
			log.Print("   - ", mn, " (unknown node)")
			continue
		}
		log.Print("   - ", mn)
	}
	log.Print(" [INF] VC MB sharders:")
	for _, sh := range vce.Sharders {
		if !r.conf.Nodes.Has(sh) {
			log.Print("   - ", sh, " (unknown node)")
			continue
		}
		log.Print("   - ", sh)
	}
}

func (r *Runner) acceptViewChange(vce *conductrpc.ViewChangeEvent) (err error) {
	if vce.Sender != r.monitor {
		return // not the monitor node
	}

	if !r.waitNoProgressUntil.IsZero() {
		return fmt.Errorf("got VC %d, but 'no progress' is expected", vce.Round)
	}

	if !r.waitNoViewChainge.IsZero() {
		if r.waitNoViewChainge.Round <= vce.Round {
			return fmt.Errorf("no VC until %d round is expected, but got on %d",
				r.waitNoViewChainge.Round, vce.Round)
		}
	}

	r.printViewChange(vce) // if verbose
	if !r.conf.Nodes.Has(vce.Sender) {
		return fmt.Errorf("unknown node %q sends view change", vce.Sender)
	}
	log.Println("view change:", vce.Round, vce.Sender)
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
	var okm, oks bool
	// check miners
	if okm = isEqual(emb.Miners, vce.Miners); !okm {
		fmt.Println("[ERR] expected miners list:")
		r.printNodes(emb.Miners)
		fmt.Println("[ERR] got miners")
		r.printNodes(vce.Miners)
	}

	// check sharders
	if oks = isEqual(emb.Sharders, vce.Sharders); !oks {
		fmt.Println("[ERR] expected sharders list:")
		r.printNodes(emb.Sharders)
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
	if !r.conf.Nodes.Has(pe.Sender) {
		return fmt.Errorf("unknown 'phase' sender: %s", pe.Sender)
	}
	if r.verbose {
		log.Print(" [INF] phase ", pe.Phase.String(), " ", pe.Sender)
	}
	if r.waitPhase.IsZero() {
		return // doesn't wait for a phase
	}
	if r.waitPhase.Phase != pe.Phase {
		return // not this phase
	}
	var (
		vcr Round
		ok  bool
	)
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
	log.Printf("[OK] accept phase %s by %s", pe.Phase.String(), pe.Sender)
	r.waitPhase = config.WaitPhase{} // reset
	return
}

func (r *Runner) acceptAddMiner(addm *conductrpc.AddMinerEvent) (err error) {
	if addm.Sender != r.monitor {
		return // not the monitor node
	}
	var (
		sender, sok = r.conf.Nodes.NodeByName(addm.Sender)
		added, aok  = r.conf.Nodes.NodeByName(addm.Miner)
	)
	if !sok {
		return fmt.Errorf("unexpected add_miner sender: %q", addm.Sender)
	}
	if !aok {
		return fmt.Errorf("unexpected miner %q added by add_miner of %q",
			addm.Miner, sender.Name)
	}

	if r.verbose {
		log.Print(" [INF] add_mienr ", added.Name)
	}

	if r.waitAdd.IsZero() {
		return // doesn't wait for a node
	}

	if r.waitAdd.TakeMiner(added.Name) {
		log.Print("[OK] add_miner ", added.Name)
	}
	return
}

func (r *Runner) acceptAddSharder(adds *conductrpc.AddSharderEvent) (err error) {
	if adds.Sender != r.monitor {
		return // not the monitor node
	}
	var (
		sender, sok = r.conf.Nodes.NodeByName(adds.Sender)
		added, aok  = r.conf.Nodes.NodeByName(adds.Sharder)
	)
	if !sok {
		return fmt.Errorf("unexpected add_sharder sender: %q", adds.Sender)
	}
	if !aok {
		return fmt.Errorf("unexpected sharder %q added by add_sharder of %q",
			adds.Sharder, sender.Name)
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

func (r *Runner) acceptNodeReady(nodeName NodeName) (err error) {
	if _, ok := r.waitNodes[nodeName]; !ok {
		var n, ok = r.conf.Nodes.NodeByName(nodeName)
		if !ok {
			return fmt.Errorf("unexpected and unknown node: %s", nodeName)
		}
		return fmt.Errorf("unexpected node: %s (%s)", n.Name, nodeName)
	}
	delete(r.waitNodes, nodeName)
	var n, ok = r.conf.Nodes.NodeByName(nodeName)
	if !ok {
		return fmt.Errorf("unknown node: %s", nodeName)
	}
	log.Println("[OK] node ready", nodeName, n.Name)
	return
}

func (r *Runner) acceptRound(re *conductrpc.RoundEvent) (err error) {
	if re.Sender != r.monitor {
		return // not the monitor node
	}

	if !r.waitNoViewChainge.IsZero() {
		if re.Round > r.waitNoViewChainge.Round {
			r.waitNoViewChainge = config.WaitNoViewChainge{} // reset
		}
	}

	var _, ok = r.conf.Nodes.NodeByName(re.Sender)
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
	_, ok = r.conf.Nodes.NodeByName(cmpke.Sender)
	if !ok {
		return fmt.Errorf("unknown 'c mpk' sender: %s", cmpke.Sender)
	}
	miner, ok = r.conf.Nodes.NodeByName(cmpke.Miner)
	if !ok {
		return fmt.Errorf("unknown 'c mpk' miner: %s", cmpke.Miner)
	}

	if r.verbose {
		log.Print(" [INF] contribute mpk ", miner.Name)
	}

	if r.waitContributeMPK.IsZero() {
		return // doesn't wait for a contribute MPK
	}

	if r.waitContributeMPK.Miner != miner.Name {
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
	_, ok = r.conf.Nodes.NodeByName(sosse.Sender)
	if !ok {
		return fmt.Errorf("unknown 'soss' sender: %s", sosse.Sender)
	}
	miner, ok = r.conf.Nodes.NodeByName(sosse.Miner)
	if !ok {
		return fmt.Errorf("unknown 'soss' miner: %s", sosse.Miner)
	}

	if r.verbose {
		log.Print(" [INF] share or sign shares ", miner.Name)
	}

	if r.waitShareSignsOrShares.IsZero() {
		return // doesn't wait for a soss
	}

	if r.waitShareSignsOrShares.Miner != miner.Name {
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
		case timeout := <-tm.C:
			if !r.waitNoProgressUntil.IsZero() {
				if timeout.UnixNano() >= r.waitNoProgressUntil.UnixNano() {
					r.waitNoProgressUntil = time.Time{} // reset
					return
				}
			}
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
