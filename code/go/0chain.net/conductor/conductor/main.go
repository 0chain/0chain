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
	"errors"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	"0chain.net/conductor/conductrpc"
	"0chain.net/conductor/conductrpc/stats"
	"0chain.net/conductor/config"
	"0chain.net/conductor/utils"
)

const noProgressSeconds = 10

func init() {
	log.SetFlags(log.Lshortfile | log.Ldate | log.Ltime | log.Lmicroseconds)
}

// type aliases
type (
	NodeName         = config.NodeName
	Round            = config.Round
	Number           = config.Number
	ExpectMagicBlock = config.ExpectMagicBlock
)

type VCInfo struct {
	MagicBlockNumber Number
	Round            Round
	Miners           []NodeName
	Sharders         []NodeName
}

func main() {
	log.Print("start the conductor")

	var (
		configFile string = "conductor.yaml"
		testsFile  string = "conductor.view-change.fault-tolerance.yaml"
		verbose    bool   = true
	)
	flag.StringVar(&configFile, "config", configFile, "configurations file")
	flag.StringVar(&testsFile, "tests", testsFile, "tests file")
	flag.BoolVar(&verbose, "verbose", verbose, "verbose output")
	flag.Parse()

	log.Print("read configurations files: ", configFile, ", ", testsFile)
	var (
		conf = readConfigs(configFile, strings.Fields(testsFile))
		r    Runner
		err  error
	)

	if len(conf.Nodes) == 0 {
		panic("NO NODES")
	}

	log.Print("create worker instance")
	r.conf = conf
	r.verbose = verbose
	r.server, err = conductrpc.NewServer(conf.Bind, conf.Nodes.Names())
	if err != nil {
		log.Fatal("[ERR]", err)
		os.Exit(1)
	}

	log.Print("(rpc) start listening on:", conf.Bind)
	go func() {
		if err := r.server.Serve(); err != nil {
			log.Fatal("Error while starting RPC server:", err)
			os.Exit(1)
		}
	}()
	defer r.server.Close()

	r.waitNodes = make(map[config.NodeName]struct{})
	r.latestBlock = make(map[NodeName]Number)
	r.rounds = make(map[config.RoundName]config.Round)
	r.nodeHistory = make(map[NodeName]*config.Node)
	r.setupTimeout(0)

	var success bool
	// not always error means failure

	err, success = r.Run()
	if err != nil {
		log.Print("[ERR] ", err)
	}

	if success {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
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

func readConfigs(configFile string, testsFilesArr []string) (conf *config.Config) {
	conf = readConfig(configFile)

	for _, testsFile := range testsFilesArr {
		matches, err := filepath.Glob(testsFile)
		if err != nil {
			panic(err)
		}
		for _, filename := range matches {
			log.Printf("Adding tests of %s", filename)
			appendTests(conf, readConfig(filename))
		}
	}
	return
}

func appendTests(conf *config.Config, tests *config.Config) {
	conf.Tests = append(conf.Tests, tests.Tests...)
	conf.Enable = append(conf.Enable, tests.Enable...)
	conf.Sets = append(conf.Sets, tests.Sets...)
}

type reportTestCase struct {
	name               string
	startedAt, endedAt time.Time // start at, end at
	directives         []reportFlowDirective
}

type reportFlowDirective struct {
	success   bool
	err       error
	directive string
}

type fileMetaRoot struct {
	fmrs         map[string]string // blobberID:fileMetaRoot
	totalBlobers int
	shouldWait   bool
}

type Runner struct {
	server  *conductrpc.Server
	conf    *config.Config
	verbose bool

	currTestCaseName string

	// state

	lastVC            *VCInfo // last view change
	lastAcceptedRound struct {
		Round     // last accepted round
		time.Time // timestamp of acceptance
	}

	// wait for
	waitPhase              config.WaitPhase              //
	waitViewChange         config.WaitViewChange         //
	waitNodes              map[config.NodeName]struct{}  // (start a node)
	waitRound              config.WaitRound              //
	waitContributeMPK      config.WaitContributeMpk      //
	waitShareSignsOrShares config.WaitShareSignsOrShares //
	waitAdd                config.WaitAdd                // add_miner, add_sharder
	waitSharderKeep        config.WaitSharderKeep        // sharder_keep
	waitNoProgress         config.WaitNoProgress         // no new rounds expected
	waitNoViewChange       config.WaitNoViewChainge      // no VC expected
	waitShardersFinalizeNearBlocks config.WaitShardersFinalizeNearBlocks // wait for sharders to finalize blocks near each other
	waitCommand            chan error                    // wait a command
	waitMinerGeneratesBlock config.WaitMinerGeneratesBlock
	waitSharderLFB	config.WaitSharderLFB	
	waitValidatorTicket   config.WaitValidatorTicket
	chalConf               *config.GenerateChallege
	fileMetaRoot           fileMetaRoot
	// timeout and monitor
	timer   *time.Timer // waiting timer
	monitor NodeName    // monitor node

	// remembered rounds: name -> round number
	rounds map[config.RoundName]config.Round // named rounds (the remember_round)

	// List of latest block numbers received from sharders
	latestBlock map[NodeName]Number

	// final report
	report []reportTestCase

	// history of all nodes spawned during each test case. Should be cleared after each test case. Used to store the logs for each case
	nodeHistory map[NodeName]*config.Node
}

func (r *Runner) isWaiting() (tm *time.Timer, ok bool) {
	tm = r.timer

	switch {
	case len(r.waitNodes) > 0:
		log.Printf("wait for %d nodes", len(r.waitNodes))
		return tm, true
	case !r.waitRound.IsZero():
		log.Println("wait for round")
		return tm, true
	case !r.waitPhase.IsZero():
		log.Println("wait for phase")
		return tm, true
	case !r.waitContributeMPK.IsZero():
		log.Println("wait for mpk contributes")
		return tm, true
	case !r.waitShareSignsOrShares.IsZero():
		log.Println("wait for share signs")
		return tm, true
	case !r.waitViewChange.IsZero():
		fmt.Printf("wait for view change %v\n", r.waitViewChange)
		return tm, true
	case !r.waitAdd.IsZero():
		log.Printf("wait for adding sharders (%+v), miners (%+v), blobbers (%+v), validators (%+v) and authorizers (%+v)", r.waitAdd.Sharders, r.waitAdd.Miners, r.waitAdd.Blobbers, r.waitAdd.Validators, r.waitAdd.Authorizers)
		return tm, true
	case !r.waitSharderKeep.IsZero():
		log.Println("wait for sharder keep")
		return tm, true
	case !r.waitNoProgress.IsZero():
		log.Println("wait for no progress")
		return tm, true
	case !r.waitNoViewChange.IsZero():
		log.Println("wait for no view change")
		return tm, true
	case r.waitMinerGeneratesBlock.MinerName != "":
		log.Printf("wait until miner %v generates block\n", r.waitMinerGeneratesBlock.MinerName)
		return tm, true
	case r.waitSharderLFB.Target != "":
		log.Printf("wait to check sharder %v got LFB\n", r.waitSharderLFB.Target)
		return tm, true
	case r.waitCommand != nil:
		// log.Println("wait for command")
		return tm, true
	case r.chalConf != nil && r.chalConf.WaitOnBlobberCommit:
		return tm, true
	case r.chalConf != nil && r.chalConf.WaitOnChallengeGeneration:
		return tm, true
	case r.chalConf != nil && r.chalConf.WaitForChallengeStatus:
		return tm, true
	case r.fileMetaRoot.shouldWait:
		return tm, true
	case r.waitValidatorTicket.ValidatorName != "":
		return tm, true
	case len(r.waitShardersFinalizeNearBlocks.Sharders) > 0:
		return tm, true
	}

	return tm, false
}

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
	log.Print(" [INF] VC round: ", vce.Round, ", number: ", vce.Number)
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

	if !r.waitNoViewChange.IsZero() {
		if r.waitNoViewChange.Round <= vce.Round {
			return fmt.Errorf("no VC until %d round is expected, but got on %d",
				r.waitNoViewChange.Round, vce.Round)
		}
	}

	vci := VCInfo{
		Round:            vce.Round,
		MagicBlockNumber: vce.Number,
		Miners:           vce.Miners,
		Sharders:         vce.Sharders,
	}

	r.printViewChange(vce) // if verbose
	if !r.conf.Nodes.Has(vce.Sender) {
		return fmt.Errorf("unknown node %q sends view change", vce.Sender)
	}
	log.Println("view change:", vce.Round, vce.Sender)
	// don't wait a VC
	if r.waitViewChange.IsZero() {
		r.lastVC = &vci
		return
	}
	// remember the round
	if rrn := r.waitViewChange.RememberRound; rrn != "" {
		log.Printf("[OK] remember round %q: %d", rrn, vce.Round)
		r.rounds[r.waitViewChange.RememberRound] = vce.Round
	}
	err = r.checkMagicBlock(&r.waitViewChange.ExpectMagicBlock, &vci)

	log.Println("[OK] view change", vce.Round)

	r.lastVC = &vci
	r.waitViewChange = config.WaitViewChange{} // reset
	return
}

func (r *Runner) checkMagicBlock(emb *ExpectMagicBlock, vci *VCInfo) (err error) {
	if emb.IsZero() {
		return // nothing more is here
	}
	if rnan := emb.RoundNextVCAfter; rnan != "" {
		var rna, ok = r.rounds[rnan]
		if !ok {
			return fmt.Errorf("unknown round name: %q", rnan)
		}
		var vcr = vci.Round // VC round
		if vcr != r.conf.ViewChange+rna {
			return fmt.Errorf("VC expected at %d, but given at %d",
				r.conf.ViewChange+rna, vcr)
		}
		// ok, accept
	} else if emb.Round != 0 && vci.Round != emb.Round {
		return fmt.Errorf("VC expected at %d, but given at %d",
			emb.Round, vci.Round)
	} else if emb.Number != 0 && vci.MagicBlockNumber != emb.Number {
		return fmt.Errorf("VC expected with %d number, but given number is %d",
			emb.Number, vci.MagicBlockNumber)
	}
	if len(emb.Miners) == 0 && len(emb.Sharders) == 0 && emb.MinersCount == 0 && emb.ShardersCount == 0 {
		return // don't check MB for nodes
	}
	// check for nodes
	var okm, oks bool
	// check miners
	if emb.MinersCount > 0 && len(emb.Miners) == 0 {
		// check count only
		if okm = (emb.MinersCount == len(vci.Miners)); !okm {
			fmt.Println("[ERR] expected miners count:", emb.MinersCount)
			fmt.Println("[ERR] got miners")
			r.printNodes(vci.Miners)
		}
	} else {
		if okm = isEqual(emb.Miners, vci.Miners); !okm {
			fmt.Println("[ERR] expected miners list:")
			r.printNodes(emb.Miners)
			fmt.Println("[ERR] got miners")
			r.printNodes(vci.Miners)
		}
	}
	// check sharders
	if emb.ShardersCount > 0 && len(emb.Sharders) == 0 {
		// check count only
		if oks = (emb.ShardersCount == len(vci.Sharders)); !oks {
			fmt.Println("[ERR] expected sharders count:", emb.ShardersCount)
			fmt.Println("[ERR] got sharders")
			r.printNodes(vci.Sharders)
		}
	} else {
		if oks = isEqual(emb.Sharders, vci.Sharders); !oks {
			fmt.Println("[ERR] expected sharders list:")
			r.printNodes(emb.Sharders)
			fmt.Println("[ERR] got sharders")
			r.printNodes(vci.Sharders)
		}
	}

	if !okm || !oks {
		return fmt.Errorf("unexpected MB miners/sharders (see logs)")
	}
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
	var lastVCRound Round = 0
	if r.lastVC != nil {
		lastVCRound = r.lastVC.Round
	}
	if vcrn := r.waitPhase.ViewChangeRound; vcrn != "" {
		if vcr, ok = r.rounds[vcrn]; !ok {
			return fmt.Errorf("unknown view_change_round of phase: %s", vcrn)
		}
		if vcr < lastVCRound {
			return // wait one more view change
		}
		if vcr >= lastVCRound+r.conf.ViewChange {
			log.Printf("got phase %s, but after %s (%d) view change, "+
				"last known view change: %d", pe.Phase.String(), vcrn, vcr,
				lastVCRound)
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
		log.Print(" [INF] add_miner ", added.Name)
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

func (r *Runner) acceptAddBlobber(addb *conductrpc.AddBlobberEvent) (
	err error) {

	if addb.Sender != r.monitor {
		return // not the monitor node
	}
	var (
		sender, sok = r.conf.Nodes.NodeByName(addb.Sender)
		added, aok  = r.conf.Nodes.NodeByName(addb.Blobber)
	)
	if !sok {
		return fmt.Errorf("unexpected add_miner sender: %q", addb.Sender)
	}
	if !aok {
		return fmt.Errorf("unexpected blobber %q added by add_blobber of %q",
			addb.Blobber, sender.Name)
	}

	if r.verbose {
		log.Print(" [INF] add_blobber ", added.Name)
	}

	if r.waitAdd.IsZero() {
		return // doesn't wait for a node
	}

	if r.waitAdd.TakeBlobber(added.Name) {
		log.Print("[OK] add_blobber ", added.Name)
	}
	return
}

func (r *Runner) acceptAddValidator(addv *conductrpc.AddValidatorEvent) (
	err error) {

	if addv.Sender != r.monitor {
		return // not the monitor node
	}
	var (
		sender, sok = r.conf.Nodes.NodeByName(addv.Sender)
		added, aok  = r.conf.Nodes.NodeByName(addv.Validator)
	)
	if !sok {
		return fmt.Errorf("unexpected add_validator sender: %q", addv.Sender)
	}
	if !aok {
		return fmt.Errorf("unexpected validator %q added by add_validator of %q",
			addv.Validator, sender.Name)
	}

	if r.verbose {
		log.Print(" [INF] add_validator ", added.Name)
	}

	if r.waitAdd.IsZero() {
		return // doesn't wait for a node
	}

	if r.waitAdd.TakeValidator(added.Name) {
		log.Print("[OK] add_validator ", added.Name)
	}
	return
}

func (r *Runner) acceptAddAuthorizer(addb *conductrpc.AddAuthorizerEvent) (
	err error) {
	if addb.Sender != r.monitor {
		return // not the monitor node
	}
	var (
		sender, sok = r.conf.Nodes.NodeByName(addb.Sender)
		added, aok  = r.conf.Nodes.NodeByName(addb.Authorizer)
	)
	if !sok {
		return fmt.Errorf("unexpected add_miner sender: %q", addb.Sender)
	}
	if !aok {
		return fmt.Errorf("unexpected authorizer %q added by add_authorizer of %q",
			addb.Authorizer, sender.Name)
	}

	if r.verbose {
		log.Print(" [INF] add_authorizer ", added.Name)
	}

	if r.waitAdd.IsZero() {
		return // doesn't wait for a node
	}

	fmt.Printf("Take authorizer %v %v\n", added.Name, r.waitAdd.Authorizers)
	if r.waitAdd.TakeAuthorizer(added.Name) {
		log.Print("[OK] add_authorizer ", added.Name)
	}
	return
}

func (r *Runner) acceptSharderKeep(ske *conductrpc.SharderKeepEvent) (
	err error) {

	if ske.Sender != r.monitor {
		return // not the monitor node
	}
	var (
		sender, sok = r.conf.Nodes.NodeByName(ske.Sender)
		added, aok  = r.conf.Nodes.NodeByName(ske.Sharder)
	)
	if !sok {
		return fmt.Errorf("unexpected sharder_keep sender: %q", ske.Sender)
	}
	if !aok {
		return fmt.Errorf("unexpected sharder %q added by sharder_keep of %q",
			ske.Sharder, sender.Name)
	}

	if r.verbose {
		log.Print(" [INF] sharder_keep ", added.Name)
	}

	if r.waitSharderKeep.IsZero() {
		return // doesn't wait for a node
	}

	if r.waitSharderKeep.TakeSharder(added.Name) {
		log.Print("[OK] sharder_keep ", added.Name)
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

	r.waitAdd.Take(nodeName)

	return
}

func (r *Runner) acceptRound(re *conductrpc.RoundEvent) (err error) {
	if re.Sender != r.monitor {
		return // not the monitor node
	}

	if r.lastAcceptedRound.Round > 0 && re.Round > r.lastAcceptedRound.Round {
		threshold := r.conf.GetStuckWarningThreshold()
		if threshold > 0 {
			duration := time.Since(r.lastAcceptedRound.Time)
			if duration > threshold {
				log.Print("[WARN] chain was stuck for ", duration)
			}
		}
	}

	if !r.waitNoProgress.IsZero() {
		if r.lastAcceptedRound.Round < re.Round && time.Now().After(r.waitNoProgress.Start) {
			return fmt.Errorf("got round %d, but 'no progress' is expected", re.Round)
		}
	}

	if !r.waitNoViewChange.IsZero() {
		if re.Round > r.waitNoViewChange.Round {
			r.waitNoViewChange = config.WaitNoViewChainge{} // reset
		}
	}

	if !r.waitViewChange.IsZero() {
		var vcr = r.waitViewChange.ExpectMagicBlock.Round
		if vcr != 0 && vcr < re.Round {
			return fmt.Errorf("missing VC at %d", vcr)
		}
	}

	var _, ok = r.conf.Nodes.NodeByName(re.Sender)
	if !ok {
		return fmt.Errorf("unknown 'round' sender: %s", re.Sender)
	}

	// set last round
	r.lastAcceptedRound = struct {
		Round
		time.Time
	}{
		re.Round,
		time.Now(),
	}

	if r.waitRound.IsZero() {
		return // doesn't wait for a round
	}

	switch {
	case r.waitRound.Round > re.Round:
		log.Printf("Got round %v\n", re.Round)
		return // not this round
	case r.waitRound.ForbidBeyond && r.waitRound.Round < re.Round:
		return fmt.Errorf("missing round: %d, got %d", r.waitRound.Round,
			re.Round)
	}
	log.Print("[OK] accept round ", re.Round)
	r.waitRound = config.WaitRound{} // don't wait anymore
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

func (r *Runner) acceptSharderBlockForMiner(block *stats.BlockFromSharder) (err error) {
	if r.verbose {
		log.Printf(" [INF] Recieved new sharder block: %+v\n", block)
	}
	switch {
	case r.waitMinerGeneratesBlock.MinerName != "":
		miner, ok := r.conf.Nodes.NodeByName(r.waitMinerGeneratesBlock.MinerName)
		if !ok {
			return fmt.Errorf("expecting block from unknown miner: %s", miner.ID)
		}
	
		if r.verbose {
			log.Printf(" [INF] got sharder block for miner %v, looking for miner %v\n", block.GeneratorId, miner.ID)
		}
	
		err = r.handleNewBlockWaitingForMinerBlockGeneration(block, string(miner.ID))
		return
	case r.waitSharderLFB.Target != "":
		sharder, ok := r.conf.Nodes.NodeByName(r.waitSharderLFB.Target)
		if !ok {
			return fmt.Errorf("expecting block from unknown sharder: %s", r.waitSharderLFB.Target)
		}

		err = r.handleNewBlockWaitingForSharderLFB(block, string(sharder.ID))
		return
	case len(r.waitShardersFinalizeNearBlocks.Sharders) > 0:
		err = r.handleNewBlockWaitingForShardersFinalizeNearBlocks(block)
	}

	return
}

func (r *Runner) acceptValidatorTicket(vt *conductrpc.ValidtorTicket) (err error) {
	if r.verbose {
		log.Printf("[INF] got validator ticket from %v\n", vt.ValidatorId)
	}

	if vt.ValidatorId != r.waitValidatorTicket.ValidatorId {
		return nil
	}

	if r.verbose {
		log.Printf(" [INF] ✅ Got validator ticket from the required validator %v\n", r.waitValidatorTicket.ValidatorId)
	}

	r.waitValidatorTicket = config.WaitValidatorTicket{}
	err = r.SetServerState(config.NotifyOnValidationTicketGeneration(false))
	return err
}

func (r *Runner) handleNewBlockWaitingForShardersFinalizeNearBlocks(block *stats.BlockFromSharder) (err error) {
	if r.verbose {
		log.Printf(" [INF] got block %v from sharder %v\n", block.Round, block.SenderId)
	}

	node, ok := r.conf.Nodes.NodeByID(config.NodeID(block.SenderId))
	if !ok {
		log.Printf(" [WARN] received block from unknown sharder %v\n", block.SenderId)
		return
	}

	r.latestBlock[node.Name] = config.Number(block.Round)

	if len(r.latestBlock) < len(r.waitShardersFinalizeNearBlocks.Sharders) {
		return
	}

	max := config.Number(0)
	min := config.Number(math.MaxInt64)
	for _, v := range r.latestBlock {
		if v > max {
			max = v
		}
		if v < min {
			min = v
		}
	}

	if max - min < 5 {
		if r.verbose {
			log.Printf(" [INF] ✅ sharders finalized near blocks\n")
		}

		r.waitShardersFinalizeNearBlocks = config.WaitShardersFinalizeNearBlocks{}

		err = r.SetServerState(&config.NotifyOnBlockGeneration{
			Enable: false,
		})
	}

	return
}

func (r *Runner) handleNewBlockWaitingForMinerBlockGeneration(block *stats.BlockFromSharder, minerId string) (err error) {
	if block.GeneratorId != string(minerId) {
		return 
	}

	if r.verbose {
		log.Printf(" [INF] ✅ found sharder block %v\n", minerId)
	}

	r.waitMinerGeneratesBlock = config.WaitMinerGeneratesBlock{}

	err = r.SetServerState(&config.NotifyOnBlockGeneration{
		Enable: false,
	})

	return
}

func (r *Runner) handleNewBlockWaitingForSharderLFB(block *stats.BlockFromSharder, sharderId string) (err error) {
	if block.SenderId == sharderId {
		minDiff := int64(6) // well, 6 is infinity if the max allowed is 5
		targetRound := block.Round
		var curDiff int64
		for sid, blk := range r.waitSharderLFB.LFBs {
			if sid == config.NodeID(sharderId) {
				continue
			}
			curDiff = blk.Round - targetRound
			if curDiff < minDiff {
				minDiff = curDiff
			}
		}

		if minDiff <=5 {
			if r.verbose {
				log.Printf(" [INF] ✅ sharder sent LFB %+v\n", block)
			}

			r.waitSharderLFB = config.WaitSharderLFB{}
			
			err = r.SetServerState(&config.NotifyOnBlockGeneration{
				Enable: false,
			})

			return
		}
	}

	if r.waitSharderLFB.LFBs == nil {
		r.waitSharderLFB.LFBs = make(map[config.NodeID]*stats.BlockFromSharder)
	}
	r.waitSharderLFB.LFBs[config.NodeID(block.SenderId)] = block
	return
}

func (r *Runner) onChallengeGeneration(txnHash string) {
	log.Printf("Challenge has been generated in txn: %v\n", txnHash)

	if r.chalConf != nil {
		r.chalConf.WaitOnChallengeGeneration = false
	}
}

func (r *Runner) onChallengeStatus(m map[string]interface{}) error {
	blobberID, ok := m["blobber_id"].(string)
	if !ok {
		return errors.New("invalid map on challenge status")
	}

	if r.chalConf != nil {
		if blobberID != r.chalConf.BlobberID {
			return nil
		}

		r.chalConf.WaitForChallengeStatus = false

		status := m["status"].(int)
		if r.chalConf.ExpectedStatus != status {
			return fmt.Errorf("expected challenge status %d, got %d", r.chalConf.ExpectedStatus, status)
		}	
	}

	return nil
}

func (r *Runner) onGettingFileMetaRoot(m map[string]string) error {	
	blobberId, ok := m["blobber_id"]
	if !ok {
		return fmt.Errorf("onGettingFileMetaRoot error: response lacks blobber_id")
	}

	fileMetaRoot, ok := m["file_meta_root"]
	if !ok {
		return fmt.Errorf("onGettingFileMetaRoot error: response lacks file_meta_root")
	}

	if r.fileMetaRoot.fmrs == nil {
		r.fileMetaRoot.fmrs = make(map[string]string)
	}

	r.fileMetaRoot.fmrs[blobberId] = fileMetaRoot

	if len(r.fileMetaRoot.fmrs) >= r.fileMetaRoot.totalBlobers {
		r.fileMetaRoot.shouldWait = false
		cfg := config.GetFileMetaRoot(false)
		return r.SetServerState(cfg)
	}
	return nil
}

func (r *Runner) onBlobberCommit(blobberID string) {
	if r.chalConf != nil {
		if blobberID != r.chalConf.BlobberID {
			log.Printf("Ignoring blobber: %s\n", blobberID)
			return
		}
		log.Printf("Value of waitonblobbercommit %v\n", r.chalConf.WaitOnBlobberCommit)
		r.chalConf.WaitOnBlobberCommit = false
	}

	err := r.SetServerState(config.BlobberCommittedWM(true))
	if err != nil {
		log.Printf("error: %s", err.Error())
	}
}

func (r *Runner) stopAll() {
	log.Print("stop all nodes")
	for _, n := range r.conf.Nodes {
		log.Printf("stop %s", n.Name)
		if err := n.Stop(); err != nil && !strings.Contains(err.Error(), "not started") {
			log.Printf("[INF] stop node error %v", err)
		}
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
		case addb := <-r.server.OnAddBlobber():
			err = r.acceptAddBlobber(addb)
		case addv := <-r.server.OnAddValidator():
			err = r.acceptAddValidator(addv)
		case adda := <-r.server.OnAddAuthorizer():
			err = r.acceptAddAuthorizer(adda)
		case sk := <-r.server.OnSharderKeep():
			err = r.acceptSharderKeep(sk)
		case nid := <-r.server.OnNodeReady():
			err = r.acceptNodeReady(nid)
		case re := <-r.server.OnRound():
			err = r.acceptRound(re)
		case cmpke := <-r.server.OnContributeMPK():
			err = r.acceptContributeMPK(cmpke)
		case sosse := <-r.server.OnShareOrSignsShares():
			err = r.acceptShareOrSignsShares(sosse)
		case block := <- r.server.OnSharderBlock():
			err = r.acceptSharderBlockForMiner(block)
		case blobberID := <-r.server.OnBlobberCommit():
			r.onBlobberCommit(blobberID)
		case blobberID := <-r.server.OnGenerateChallenge():
			r.onChallengeGeneration(blobberID)
		case m := <-r.server.OnChallengeStatus():
			err = r.onChallengeStatus(m)
		case m := <-r.server.OnGettingFileMetaRoot():
			err = r.onGettingFileMetaRoot(m)
		case vt := <-r.server.OnValidatorTicket():
			err = r.acceptValidatorTicket(vt)
		case err = <-r.waitCommand:
			if err != nil {
				err = fmt.Errorf("executing command: %v", err)
			}
			r.waitCommand = nil // reset
		case timeout := <-tm.C:
			if !r.waitNoProgress.IsZero() {
				if timeout.UnixNano() >= r.waitNoProgress.Until.UnixNano() {
					r.waitNoProgress = config.WaitNoProgress{} // reset
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

func okString(t bool) string {
	if t {
		return "[PASS]"
	}
	return "[FAIL]"
}

func (r *Runner) processReport() (success bool) {
	success = true

	var totalDuration time.Duration

	fmt.Println("........................ R E P O R T ........................")

	for _, testCase := range r.report {
		fmt.Println("- ", testCase.name)

		var caseError error = nil
		var caseSuccess bool = true
		totalDuration += testCase.endedAt.Sub(testCase.startedAt)

		for i, flowDirective := range testCase.directives {
			caseSuccess = caseSuccess && flowDirective.success

			if flowDirective.err != nil {
				caseError = flowDirective.err
				fmt.Printf("Failed on index %d directive. Directive name: %s\n",
					i, flowDirective.directive)
				break
			}
		}

		fmt.Printf("  %s after %s\n", okString(caseSuccess),
			testCase.endedAt.Sub(testCase.startedAt).Round(time.Second))

		success = success && caseSuccess

		if caseError != nil {
			fmt.Printf("  - [ERR] %v\n", caseError)
		}
	}

	fmt.Println("total duration:", totalDuration.Round(time.Second))
	fmt.Println("overall success:", success)
	fmt.Println(".............................................................")

	return success
}

func (r *Runner) resetWaiters() {
	r.waitNodes = make(map[NodeName]struct{})                  //
	r.waitRound = config.WaitRound{}                           //
	r.waitPhase = config.WaitPhase{}                           //
	r.waitContributeMPK = config.WaitContributeMpk{}           //
	r.waitShareSignsOrShares = config.WaitShareSignsOrShares{} //
	r.waitViewChange = config.WaitViewChange{}                 //
	r.waitAdd = config.WaitAdd{}                               //
	r.waitNoProgress = config.WaitNoProgress{}                 //
	r.waitNoViewChange = config.WaitNoViewChainge{}            //
	r.waitSharderKeep = config.WaitSharderKeep{}               //
	r.waitMinerGeneratesBlock = config.WaitMinerGeneratesBlock{}
	if r.waitCommand != nil {
		go func(wc chan error) { <-wc }(r.waitCommand)
		r.waitCommand = nil
	}

}

func (r *Runner) resetRounds() {
	r.lastAcceptedRound.Round = 0
	r.lastVC = nil
}

// Run the tests.
// Not always presence of an error means failure
func (r *Runner) Run() (err error, success bool) {
	log.Println("start testing...")
	defer log.Println("end of testing")

	// stop all nodes after all
	defer r.stopAll()

	// clean full logs directory
	if err = os.RemoveAll(r.conf.FullLogsDir); err != nil {
		log.Printf("[WARN] couldn't clean full logs dir")
	}

	// for every enabled set
	for _, set := range r.conf.Sets {
		if !r.conf.IsEnabled(&set) {
			continue
		}
		log.Print("...........................................................")
		log.Print("start set ", set.Name)
		log.Print("...........................................................")

	cases:
		for i, testCase := range r.conf.TestsOfSet(&set) {
			_ = r.SetMagicBlock("")
			r.conf.CleanupEnv()
			var report reportTestCase
			report.name = testCase.Name
			report.startedAt = time.Now()

			log.Print("=======================================================")
			log.Printf("Test case %d: %s", i, testCase.Name)
			for j, d := range testCase.Flow {
				log.Print("---------------------------------------------------")
				log.Printf("  %d/%d step", i, j)

				err, mustFail := d.Execute(r)
				if err == nil {
					err = r.proceedWaiting()
				}

				if err != nil {
					// this is a failure, but might be an expected one
					report.directives = append(report.directives, reportFlowDirective{
						success:   mustFail,
						err:       err,
						directive: d.GetName(),
					})

					report.endedAt = time.Now()
					r.report = append(r.report, report) // add to report
					r.stopAll()
					r.resetWaiters()

					log.Printf("[ERR] at the end of %d test case: %v", i, err)
					if mustFail {
						log.Printf("[The error is expected result of the test case]")
					}

					if testCase.Flow.IsSavingLogs() {
						if err := r.SaveLogs(); err != nil {
							log.Printf("Warning: error while saving logs: %v", err)
						}
					}

					continue cases
				}

				// test case succeeded
				report.directives = append(report.directives, reportFlowDirective{
					success: !mustFail,
					err:     err,
				})
			}

			report.endedAt = time.Now()
			r.report = append(r.report, report)
			
			// Export all logs
			if errors := r.ExportFullLogs(testCase.Name); len(errors) > 0 {
				log.Printf("[WARN] ⚠️ errors while exporting full logs for this test case: %v", errors)
			} else {
				log.Printf("[INF] ✅ all logs saved to the full logs dir successfully")
			}

			// clear node history
			r.nodeHistory = make(map[NodeName]*config.Node)

			log.Printf("end of %d %s test case", i, testCase.Name)
		}
	}

	success = r.processReport()
	return err, success
}

func (r *Runner) ExportFullLogs(testCaseName string) (errors []error) {
	log.Printf("[INF] exporting full logs for the test case")
	fullLogsPathForTheCase := filepath.Join(r.conf.FullLogsDir, strings.ReplaceAll(testCaseName, " ", "-"))

	// copy current case conductor logs in conductor logs backup dir
	condcutorLogsDstPath := filepath.Join(fullLogsPathForTheCase, "conductor")
	log.Printf("[INF] saving conductor logs from path %v", condcutorLogsDstPath)
	if err := utils.CopyDir(r.conf.Logs, condcutorLogsDstPath); err != nil {
		errors = append(errors, err)
	}

	// copy all spawned nodes logs during the case
	for _, node := range r.nodeHistory {
		nodeLogsSrcPath := filepath.Join(node.WorkDir, node.LogsDir)
		nodeLogsDstPath := filepath.Join(fullLogsPathForTheCase, string(node.Name))
		log.Printf("[INF] saving logs for node %v from path %v", node.Name, nodeLogsSrcPath)

		if err := utils.CopyDir(nodeLogsSrcPath, nodeLogsDstPath); err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}