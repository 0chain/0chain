package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"0chain.net/conductor/conductrpc"
	"0chain.net/conductor/conductrpc/stats"
	"0chain.net/conductor/config"
	"0chain.net/conductor/config/cases"
	"0chain.net/conductor/dirs"
	"0chain.net/conductor/services"
	"0chain.net/conductor/types"
	"0chain.net/conductor/utils"
)

//
// execute (the config.Executor implementation)
//

func (r *Runner) setupTimeout(tm time.Duration) {
	r.timer = time.NewTimer(tm)
	if tm <= 0 {
		<-r.timer.C // drain zero timeout so that wherever it is waited upon it waits indefinitely
	}
}

func (r *Runner) doStart(name NodeName, lock, errIfAlreadyStarted bool) (err error) {
	var n, ok = r.conf.Nodes.NodeByName(name)
	if !ok {
		return fmt.Errorf("(doStart): unknown node: %q", name)
	}
	if n.IsStarted() {
		if errIfAlreadyStarted {
			return fmt.Errorf("(doStart): node already started: %s", n.Name)
		} else {
			return nil
		}
	}
	// miners and sharders, but skip blobbers
	if !r.conf.IsSkipWait(name) {
		r.server.AddNode(name, lock)   // expected server interaction
		r.waitNodes[name] = struct{}{} // wait list
	}
	if err := n.Start(r.conf.Logs, r.conf.Env); err != nil {
		return fmt.Errorf("starting %s: %v", n.Name, err)
	}

	r.nodeHistory[n.Name] = n
	return nil
}

//
// control the conductor (entire tests controls)
//

// SetMonitor for phases and view changes.
func (r *Runner) SetMonitor(name NodeName) (err error) {
	r.server.AddNode(name, false) // the node must exists to be a monitor
	err = r.server.UpdateState(name, func(state *conductrpc.State) {
		state.IsMonitor = true
	})
	if err != nil {
		return
	}
	r.monitor = name // monitor node
	return           // ok
}

// CleanupBC cleans up blockchain.
func (r *Runner) CleanupBC(tm time.Duration) (err error) {
	r.stopAll()
	r.resetRounds()
	err = r.conf.CleanupBC()
	if err != nil {
		log.Printf("Cleanup_BC: do cleanup result %v", err)
	}
	return err
}

// SaveLogs copies current execution logs contained in the "workDir/node-n/log" to the
// "docker.local/conductor.backup-logs/testCaseName".
func (r *Runner) SaveLogs() error {
	now := time.Now().Format(time.RFC822)
	for _, node := range r.conf.Nodes {
		var (
			source       = filepath.Join(node.WorkDir, "log")
			testCaseName = strings.Replace(r.currTestCaseName, " ", "_", -1) + "_" + now
			destination  = filepath.Join("docker.local", "conductor.backup_logs", testCaseName, string(node.Name))
		)
		if err := os.MkdirAll(destination, 0755); err != nil {
			return err
		}
		if err := dirs.CopyDir(source, destination); err != nil {
			return err
		}
	}

	return nil
}

// set additional environment variables
func (r *Runner) SetEnv(env map[string]string) (err error) {
	if r.verbose {
		keys := make([]string, len(env))
		i := 0
		for k := range env {
			keys[i] = k
			i++
		}
		log.Printf(" [INF] setting test-specific environment variables: %s", strings.Join(keys, ","))
	}
	r.conf.Env = env
	return nil
}

//
// control nodes
//

func (r *Runner) GetNodes() map[config.NodeName]config.NodeID {
	m := make(map[config.NodeName]config.NodeID)
	for _, n := range r.conf.Nodes {
		m[n.Name] = n.ID
	}

	return m
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
		if err := r.doStart(name, lock, true); err != nil {
			return err
		}
	}
	return
}

func (r *Runner) Unlock(names []NodeName, tm time.Duration) (err error) {

	if r.verbose {
		log.Print(" [INF] unlock ", names)
	}

	r.setupTimeout(tm)
	err = r.server.UpdateStates(names, func(state *conductrpc.State) {
		state.IsLock = false
	})
	if err != nil {
		return fmt.Errorf("unlocking nodes: %v", err)
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
			if err := n.Kill(); err != nil {
				log.Printf("kill failed: %v", err)
			}
		}
		log.Print(n.Name, " stopped")
	}
	return
}

//
// checks
//

func (r *Runner) ExpectActiveSet(emb config.ExpectMagicBlock) (
	err error) {

	if r.verbose {
		log.Print(" [INF] checking the active set ")
	}
	if r.lastVC == nil {
		return errors.New("no VC info yet!")
	}
	err = r.checkMagicBlock(&emb, r.lastVC)
	if err == nil {
		log.Println("[OK] active set")
	}
	return err
}

//
// waiters
//

func (r *Runner) WaitViewChange(vc config.WaitViewChange, tm time.Duration) (
	err error) {

	if r.verbose {
		log.Print(" [INF] wait for VC ", vc.ExpectMagicBlock.Round, "/",
			vc.ExpectMagicBlock.Number)
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

func (r *Runner) WaitRound(wr config.WaitRound, tm time.Duration) (err error) {
	if wr.Round == 0 {
		if wr.Name != "" {
			// by a named round
			var rx, ok = r.rounds[wr.Name]
			if !ok {
				return fmt.Errorf(
					"wait_round: no round with %q name is registered",
					wr.Name)
			}
			wr.Round = rx // by the named round
			if wr.Shift != 0 {
				wr.Round += wr.Shift // shift the named round
			} else {
				return fmt.Errorf(
					"wait_round: wait named round %q without a shift",
					wr.Name)
			}
		} else if wr.Shift != 0 {
			// shift without a name means shift from current round
			wr.Round = r.lastAcceptedRound.Round + wr.Shift
		}
	}

	// reset all fields excluding the 'Round'
	wr.Name, wr.Shift = "", 0

	r.setupTimeout(tm)
	r.waitRound = wr

	if r.verbose {
		log.Print(" [INF] wait for round ", wr.Round)
	}

	return
}

func (r *Runner) WaitContributeMpk(wcmpk config.WaitContributeMpk,
	tm time.Duration) (err error) {

	var miner, ok = r.conf.Nodes.NodeByName(wcmpk.Miner)
	if !ok {
		return fmt.Errorf("unknown miner: %q", wcmpk.Miner)
	}

	if r.verbose {
		log.Print(" [INF] wait for contribute MPK by ", miner.Name)
	}

	r.setupTimeout(tm)
	r.waitContributeMPK = wcmpk
	return
}

func (r *Runner) WaitShareSignsOrShares(ssos config.WaitShareSignsOrShares,
	tm time.Duration) (err error) {

	if r.verbose {
		log.Printf(" [INF] wait for SOSS of %s", ssos.Miner)
	}

	var miner, ok = r.conf.Nodes.NodeByName(ssos.Miner)
	if !ok {
		return fmt.Errorf("unknown miner: %v", ssos.Miner)
	}

	if r.verbose {
		log.Print(" [INF] wait for SOSS by ", miner.Name)
	}

	r.setupTimeout(tm)
	r.waitShareSignsOrShares = ssos
	return
}

func (r *Runner) WaitAdd(wadd config.WaitAdd, tm time.Duration) (err error) {
	if r.verbose {
		log.Printf(" [INF] wait add miners: %s, sharders: %s, blobbers: %s, validators: %s, authorizers %s",
			wadd.Miners, wadd.Sharders, wadd.Blobbers, wadd.Validators, wadd.Authorizers)
	}

	r.setupTimeout(tm)
	r.waitAdd = wadd
	if wadd.Start {
		// start nodes that haven't been started yet
		allNodes := append(wadd.Sharders, wadd.Miners...)
		allNodes = append(allNodes, wadd.Blobbers...)
		allNodes = append(allNodes, wadd.Validators...)
		allNodes = append(allNodes, wadd.Authorizers...)
		allNodes = append(allNodes, wadd.Validators...)

		for _, name := range allNodes {
			if err := r.doStart(name, false, false); err != nil {
				return err
			}
		}
	}

	// it is not necessary to wait for authorizers because they are registered previously
	r.waitAdd.Authorizers = []config.NodeName{}

	return
}

func (r *Runner) WaitSharderKeep(wsk config.WaitSharderKeep,
	tm time.Duration) (err error) {

	if r.verbose {
		log.Printf(" [INF] wait shader keep: %s", wsk.Sharders)
	}

	r.setupTimeout(tm)
	r.waitSharderKeep = wsk
	return
}

func (r *Runner) WaitNoProgress(wait time.Duration) (err error) {
	if r.verbose {
		log.Print(" [INF] wait no progress ", wait.String())
	}

	r.waitNoProgress = config.WaitNoProgress{Start: time.Now().Add(noProgressSeconds * time.Second), Until: time.Now().Add(wait)}
	r.setupTimeout(wait)
	return
}

func (r *Runner) WaitMinerGeneratesBlock(wmgb config.WaitMinerGeneratesBlock, timeout time.Duration) (err error) {
	if r.verbose {
		log.Printf("[INF] start waiting until miner generates block: %s", wmgb.MinerName)
	}

	r.setupTimeout(timeout)

	err = r.SetServerState(&config.NotifyOnBlockGeneration{
		Enable: true,
	})
	if err != nil {
		return
	}

	r.waitMinerGeneratesBlock = wmgb
	return
}

func (r *Runner) WaitSharderLFB(conf config.WaitSharderLFB, timeout time.Duration) (err error) {
	if r.verbose {
		log.Printf(" [INF] Watching for sharders blocks to check LFB for %v\n", conf.Target)
	}

	r.setupTimeout(timeout)

	err = r.SetServerState(&config.NotifyOnBlockGeneration{
		Enable: true,
	})
	if err != nil {
		return
	}

	r.waitSharderLFB = conf
	return
}

func (r *Runner) WaitShardersFinalizeNearBlocks(command config.WaitShardersFinalizeNearBlocks, timeout time.Duration) {
	if r.verbose {
		log.Printf(" [INF] Watching for sharders blocks to check LFB for %v\n", command.Sharders)
	}

	r.setupTimeout(timeout)

	err := r.SetServerState(&config.NotifyOnBlockGeneration{
		Enable: true,
	})
	if err != nil {
		return
	}

	r.waitShardersFinalizeNearBlocks = command
}

func (r *Runner) GenerateChallenge(c *config.GenerateChallege) error {
	if r.verbose {
		log.Print(" [INF] setting generate challenge info")
	}

	r.chalConf = c
	return nil
}

func (r *Runner) WaitForChallengeGeneration(timeout time.Duration) {
	if r.verbose {
		log.Print(" [INF] waiting for blockchain to generate challenge")
	}

	if r.chalConf == nil {
		log.Printf(" [ERR] challenge config is not set")
		return
	}

	r.setupTimeout(timeout)
	r.chalConf.WaitOnChallengeGeneration = true
}

func (r *Runner) WaitOnBlobberCommit(timeout time.Duration) {
	if r.verbose {
		log.Print(" [INF] waiting for blobber to commit writemarker")
	}

	if r.chalConf == nil {
		log.Printf(" [ERR] challenge config is not set")
		return
	}

	r.setupTimeout(timeout)
	r.chalConf.WaitOnBlobberCommit = true
}

func (r *Runner) WaitForChallengeStatus(timeout time.Duration) {
	if r.verbose {
		log.Print(" [INF] waiting for challenge status from chain")
	}

	if r.chalConf == nil {
		log.Printf(" [ERR] challenge config is not set")
		return
	}

	r.setupTimeout(timeout)
	r.chalConf.WaitForChallengeStatus = true
}

func (r *Runner) WaitValidatorTicket(wvt config.WaitValidatorTicket, timeout time.Duration) {
	validator, ok := r.conf.Nodes.NodeByName(config.NodeName(wvt.ValidatorName))
	if !ok {
		log.Printf("[ERR] Validator with name %v not found", wvt.ValidatorName)
	}

	if r.verbose {
		log.Printf(" [INF] waiting for ticket from validator %v (%v)", wvt.ValidatorName, validator.ID)
	}

	err := r.SetServerState(config.NotifyOnValidationTicketGeneration(true))
	if err != nil {
		log.Printf("[ERR] setting notify on validation ticket generation: %v", err)
	}

	r.setupTimeout(timeout)
	r.waitValidatorTicket.ValidatorName = wvt.ValidatorName
	r.waitValidatorTicket.ValidatorId = string(validator.ID)
}

func (r *Runner) WaitForFileMetaRoot() {
	if r.verbose {
		log.Print(" [INF] waiting for file meta root")
	}
	count := 0
	for name := range r.server.Nodes() {
		if strings.Contains(string(name), "blobber") {
			count++
		}
	}

	f := fileMetaRoot{
		shouldWait:   true,
		totalBlobers: count,
	}
	r.fileMetaRoot = f
}

func (r *Runner) CheckFileMetaRoot(cfg *config.CheckFileMetaRoot) error {
	if r.verbose {
		log.Print(" [INF] checking file meta root")
	}

	var fmrs []string
	for _, fmr := range r.fileMetaRoot.fmrs {
		fmrs = append(fmrs, fmr)
	}

	curFmr := fmrs[0]
	allEqual := true
	for i := 1; i < len(fmrs); i++ {
		allEqual = allEqual && curFmr == fmrs[i]
		curFmr = fmrs[i]
	}

	fmt.Printf("RequiredSameRoot = %v, allEqual = %v\n", cfg.RequireSameRoot, allEqual)

	if cfg.RequireSameRoot && !allEqual {
		return fmt.Errorf("required all file meta root to be same")
	}

	if !cfg.RequireSameRoot && allEqual {
		return fmt.Errorf("required some file meta root to be different")
	}

	return nil
}

//
// Byzantine blockchain miners.
//

func (r *Runner) VRFS(vrfs *config.Bad) (err error) {
	r.verbosePrintByGoodBad("VRFS", vrfs)

	err = r.server.UpdateStates(vrfs.By, func(state *conductrpc.State) {
		state.VRFS = vrfs
	})
	if err != nil {
		return fmt.Errorf("setting VRFS: %v", err)
	}
	return
}

func (r *Runner) RoundTimeout(rt *config.Bad) (err error) {
	r.verbosePrintByGoodBad("wrong round timeout", rt)

	err = r.server.UpdateStates(rt.By, func(state *conductrpc.State) {
		state.RoundTimeout = rt
	})
	if err != nil {
		return fmt.Errorf("setting 'wrong round timeout': %v", err)
	}
	return
}

func (r *Runner) CompetingBlock(cb *config.Bad) (err error) {
	r.verbosePrintByGoodBad("competing block", cb)

	err = r.server.UpdateStates(cb.By, func(state *conductrpc.State) {
		state.CompetingBlock = cb
	})
	if err != nil {
		return fmt.Errorf("setting 'competing block': %v", err)
	}
	return
}

func (r *Runner) SignOnlyCompetingBlocks(socb *config.Bad) (err error) {
	r.verbosePrintByGoodBad("sign only competing block", socb)

	err = r.server.UpdateStates(socb.By, func(state *conductrpc.State) {
		state.SignOnlyCompetingBlocks = socb
	})
	if err != nil {
		return fmt.Errorf("setting 'sign only competing block': %v", err)
	}
	return
}

func (r *Runner) DoubleSpendTransaction(dst *config.Bad) (err error) {
	r.verbosePrintByGoodBad("double spend transaction", dst)

	err = r.server.UpdateStates(dst.By, func(state *conductrpc.State) {
		state.DoubleSpendTransaction = dst
	})
	if err != nil {
		return fmt.Errorf("setting 'double spend transaction': %v", err)
	}
	return
}

func (r *Runner) WrongBlockSignHash(wbsh *config.Bad) (err error) {
	r.verbosePrintByGoodBad("wrong block sign hash", wbsh)

	err = r.server.UpdateStates(wbsh.By, func(state *conductrpc.State) {
		state.WrongBlockSignHash = wbsh
	})
	if err != nil {
		return fmt.Errorf("setting 'wrong block sign hash': %v", err)
	}
	return
}

func (r *Runner) WrongBlockSignKey(wbsk *config.Bad) (err error) {
	r.verbosePrintByGoodBad("wrong block sign key", wbsk)

	err = r.server.UpdateStates(wbsk.By, func(state *conductrpc.State) {
		state.WrongBlockSignKey = wbsk
	})
	if err != nil {
		return fmt.Errorf("setting 'wrong block sign key': %v", err)
	}
	return
}

func (r *Runner) WrongBlockHash(wbh *config.Bad) (err error) {
	r.verbosePrintByGoodBad("wrong block hash", wbh)

	err = r.server.UpdateStates(wbh.By, func(state *conductrpc.State) {
		state.WrongBlockHash = wbh
	})
	if err != nil {
		return fmt.Errorf("setting 'wrong block hash': %v", err)
	}
	return
}

func (r *Runner) WrongBlockRandomSeed(wb *config.Bad) (err error) {
	r.verbosePrintByGoodBad("wrong block random seed", wb)

	err = r.server.UpdateStates(wb.By, func(state *conductrpc.State) {
		state.WrongBlockRandomSeed = wb
	})
	if err != nil {
		return fmt.Errorf("setting 'wrong block random seed': %v", err)
	}
	return
}

func (r *Runner) WrongBlockDDoS(wb *config.Bad) (err error) {
	r.verbosePrintByGoodBad("wrong block ddos", wb)

	err = r.server.UpdateStates(wb.By, func(state *conductrpc.State) {
		state.WrongBlockDDoS = wb
	})
	if err != nil {
		return fmt.Errorf("setting 'wrong block ddos': %v", err)
	}
	return
}

func (r *Runner) VerificationTicketGroup(vtg *config.Bad) (err error) {
	r.verbosePrintByGoodBad("verification ticket group", vtg)

	err = r.server.UpdateStates(vtg.By, func(state *conductrpc.State) {
		state.VerificationTicketGroup = vtg
	})
	if err != nil {
		return fmt.Errorf("setting 'verification_ticket_group': %v", err)
	}
	return
}

func (r *Runner) WrongVerificationTicketHash(wvth *config.Bad) (err error) {
	r.verbosePrintByGoodBad("wrong verification ticket hash", wvth)

	err = r.server.UpdateStates(wvth.By, func(state *conductrpc.State) {
		state.WrongVerificationTicketHash = wvth
	})
	if err != nil {
		return fmt.Errorf("setting 'wrong verification ticket hash': %v", err)
	}
	return
}

func (r *Runner) WrongVerificationTicketKey(wvtk *config.Bad) (err error) {
	r.verbosePrintByGoodBad("wrong verification ticket key", wvtk)

	err = r.server.UpdateStates(wvtk.By, func(state *conductrpc.State) {
		state.WrongVerificationTicketKey = wvtk
	})
	if err != nil {
		return fmt.Errorf("setting 'wrong verification ticket key': %v", err)
	}
	return
}

func (r *Runner) WrongNotarizedBlockHash(wnbh *config.Bad) (err error) {
	r.verbosePrintByGoodBad("wrong notarized block hash", wnbh)

	err = r.server.UpdateStates(wnbh.By, func(state *conductrpc.State) {
		state.WrongNotarizedBlockHash = wnbh
	})
	if err != nil {
		return fmt.Errorf("setting 'wrong notarized block hash': %v", err)
	}
	return
}

func (r *Runner) WrongNotarizedBlockKey(wnbk *config.Bad) (err error) {
	r.verbosePrintByGoodBad("wrong notarized block key", wnbk)

	err = r.server.UpdateStates(wnbk.By, func(state *conductrpc.State) {
		state.WrongNotarizedBlockKey = wnbk
	})
	if err != nil {
		return fmt.Errorf("setting 'wrong notarized block key': %v", err)
	}
	return
}

func (r *Runner) NotarizeOnlyCompetingBlock(ncb *config.Bad) (err error) {
	r.verbosePrintByGoodBad("notarize only competing block", ncb)

	err = r.server.UpdateStates(ncb.By, func(state *conductrpc.State) {
		state.NotarizeOnlyCompetingBlock = ncb
	})
	if err != nil {
		return fmt.Errorf("setting 'notarized only competing block': %v", err)
	}
	return
}

func (r *Runner) NotarizedBlock(nb *config.Bad) (err error) {
	r.verbosePrintByGoodBad("notarized block", nb)

	err = r.server.UpdateStates(nb.By, func(state *conductrpc.State) {
		state.NotarizedBlock = nb
	})
	if err != nil {
		return fmt.Errorf("setting 'notarized block': %v", err)
	}
	return
}

//
// Misbehavior
//

func (r *Runner) ConfigureGeneratorsFailure(round Round) (
	err error) {

	if r.verbose {
		log.Printf(" [INF] configure generators failure for round %v", round)
	}

	err = r.server.UpdateAllStates(func(state *conductrpc.State) {
		state.GeneratorsFailureRoundNumber = round
	})
	if err != nil {
		return fmt.Errorf("configuring generators failure for round %v: %v", round, err)
	}
	return
}

func (r *Runner) SetRevealed(ss []NodeName, pin bool, tm time.Duration) (
	err error) {

	if r.verbose {
		log.Printf(" [INF] set revealed of %s to %t", ss, pin)
	}

	err = r.server.UpdateStates(ss, func(state *conductrpc.State) {
		state.IsRevealed = pin
	})
	if err != nil {
		return fmt.Errorf("setting revealed to %t nodes: %v", pin, err)
	}
	return
}

//
// Byzantine VC miners.
//

func (r *Runner) MPK(mpk *config.Bad) (err error) {
	r.verbosePrintByGoodBad("MPK", mpk)

	err = r.server.UpdateStates(mpk.By, func(state *conductrpc.State) {
		state.MPK = mpk
	})
	if err != nil {
		return fmt.Errorf("setting 'MPK': %v", err)
	}
	return
}

func (r *Runner) Shares(s *config.Bad) (err error) {
	r.verbosePrintByGoodBad("shares", s)

	err = r.server.UpdateStates(s.By, func(state *conductrpc.State) {
		state.Shares = s
	})
	if err != nil {
		return fmt.Errorf("setting 'shares': %v", err)
	}
	return
}

func (r *Runner) Signatures(s *config.Bad) (err error) {
	r.verbosePrintByGoodBad("signatures", s)

	err = r.server.UpdateStates(s.By, func(state *conductrpc.State) {
		state.Signatures = s
	})
	if err != nil {
		return fmt.Errorf("setting 'signatures': %v", err)
	}
	return
}

func (r *Runner) Publish(p *config.Bad) (err error) {
	r.verbosePrintByGoodBad("publish", p)

	err = r.server.UpdateStates(p.By, func(state *conductrpc.State) {
		state.Publish = p
	})
	if err != nil {
		return fmt.Errorf("setting 'publish': %v", err)
	}
	return
}

//
// Byzantine blockchain sharders
//

func (r *Runner) FinalizedBlock(fb *config.Bad) (err error) {
	r.verbosePrintByGoodBad("finalized block", fb)

	err = r.server.UpdateStates(fb.By, func(state *conductrpc.State) {
		state.FinalizedBlock = fb
	})
	if err != nil {
		return fmt.Errorf("setting 'finalized block': %v", err)
	}
	return
}

func (r *Runner) MagicBlock(mb *config.Bad) (err error) {
	r.verbosePrintByGoodBad("magic block", mb)

	err = r.server.UpdateStates(mb.By, func(state *conductrpc.State) {
		state.MagicBlock = mb
	})
	if err != nil {
		return fmt.Errorf("setting 'magic block': %v", err)
	}
	return
}

func (r *Runner) VerifyTransaction(vt *config.Bad) (err error) {
	r.verbosePrintByGoodBad("verify transaction", vt)

	err = r.server.UpdateStates(vt.By, func(state *conductrpc.State) {
		state.VerifyTransaction = vt
	})
	if err != nil {
		return fmt.Errorf("setting bad 'verify transaction': %v", err)
	}
	return
}

func (r *Runner) WaitNoViewChainge(wnvc config.WaitNoViewChainge,
	tm time.Duration) (err error) {

	if r.verbose {
		log.Printf(" [INF] wait no view change until %d round", wnvc.Round)
	}

	r.setupTimeout(tm)
	r.waitNoViewChange = wnvc
	return
}

// Command executing.
func (r *Runner) Command(name string, params map[string]interface{}, retryCount int, failureThreshold, tm time.Duration) {
	r.setupTimeout(tm)

	if r.verbose {
		log.Printf(" [INF] command %q", name)
	}

	stringParams := make(map[string]string, len(params))
	for k, v := range params {
		switch tv := v.(type) {
		case string, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool:
			stringParams[k] = fmt.Sprintf("%v", tv)
		case []interface{}:
			stringSlice, err := utils.StringSlice(tv)
			if err != nil {
				r.waitCommand = make(chan error)
				r.waitCommand <- err
				return
			}
			stringParams[k] = strings.Join(stringSlice, ",")
		}
	}

	r.waitCommand = r.asyncCommand(name, stringParams, retryCount, failureThreshold)
}

func (r *Runner) asyncCommand(name string, params map[string]string, retryCount int, failureThreshold time.Duration) (reply chan error) {
	reply = make(chan error)
	fmt.Println("async command")
	go r.runAsyncCommand(reply, name, params, retryCount, failureThreshold)
	return
}

func (r *Runner) runAsyncCommand(reply chan error, name string, params map[string]string, retryCount int, failureThreshold time.Duration) {
	var err error
	if retryCount == 0 {
		retryCount = 1
	}
	
	for i := 0; i < retryCount; i++ {
		err = r.conf.Execute(name, params, failureThreshold)
		if err == nil {
			reply <- nil
			return
		}
		log.Printf(" [ERR] command %q failed: %v", name, err)
	}
	
	reply <- err // nil or error (of the latest run)
}

//
// blobber related commands
//

func (r *Runner) StorageTree(st *config.Bad) (err error) {
	if r.verbose {
		log.Printf(" [INF] set bad 'storage_tree' of %s", st.Bad)
	}

	err = r.server.UpdateStates(st.Bad, func(state *conductrpc.State) {
		state.StorageTree = st
	})
	if err != nil {
		return fmt.Errorf("setting bad 'storage_tree': %v", err)
	}
	return
}

func (r *Runner) ValidatorProof(vp *config.Bad) (err error) {
	if r.verbose {
		log.Printf(" [INF] set bad 'validator_proof' of %s", vp.Bad)
	}

	err = r.server.UpdateStates(vp.Bad, func(state *conductrpc.State) {
		state.ValidatorProof = vp
	})
	if err != nil {
		return fmt.Errorf("setting bad 'storage_tree': %v", err)
	}
	return
}

func (r *Runner) Challenges(cs *config.Bad) (err error) {
	if r.verbose {
		log.Printf(" [INF] set bad 'challenges' of %s", cs.Bad)
	}

	err = r.server.UpdateStates(cs.Bad, func(state *conductrpc.State) {
		state.Challenges = cs
	})
	if err != nil {
		return fmt.Errorf("setting bad 'challenges': %v", err)
	}
	return
}

func (r *Runner) verbosePrintByGoodBad(label string, bad *config.Bad) {
	if r.verbose {
		log.Printf(" [INF] set '%s' of %s: good %s, bad %s",
			label, bad.By, bad.Good, bad.Bad)
	}
}

// MinersNum implements config.Executor interface.
func (r *Runner) MinersNum() int {
	return r.server.GetMinersNum()
}

// GetMonitorID implements config.Executor interface.
func (r *Runner) GetMonitorID() string {
	monitorName := r.monitor
	for _, node := range r.conf.Nodes {
		if node.Name == monitorName {
			return string(node.ID)
		}
	}
	return ""
}

// EnableServerStatsCollector implements config.Executor interface.
func (r *Runner) EnableServerStatsCollector() error {
	return r.server.EnableServerStatsCollector()
}

// EnableClientStatsCollector implements config.Executor interface.
func (r *Runner) EnableClientStatsCollector() error {
	return r.server.EnableClientStatsCollector()
}

// GetServerStatsCollector implements config.Executor interface.
func (r *Runner) GetServerStatsCollector() *stats.NodesServerStats {
	return r.server.NodesServerStatsCollector
}

// GetClientStatsCollector implements config.Executor interface.
func (r *Runner) GetClientStatsCollector() *stats.NodesClientStats {
	return r.server.NodesClientStatsCollector
}

//
// Checks
//

// ConfigureTestCase implements config.Executor interface.
func (r *Runner) ConfigureTestCase(configurator cases.TestCaseConfigurator) error {
	if r.verbose {
		log.Printf(" [INF] configuring \"%s\" test case", configurator.Name())
	}

	err := r.server.UpdateAllStates(func(state *conductrpc.State) {
		switch cfg := configurator.(type) {
		case *cases.NotNotarisedBlockExtension:
			state.ExtendNotNotarisedBlock = cfg

		case *cases.SendDifferentBlocksFromFirstGenerator:
			state.SendDifferentBlocksFromFirstGenerator = cfg

		case *cases.SendDifferentBlocksFromAllGenerators:
			state.SendDifferentBlocksFromAllGenerators = cfg

		case *cases.BreakingSingleBlock:
			state.BreakingSingleBlock = cfg

		case *cases.SendInsufficientProposals:
			state.SendInsufficientProposals = cfg

		case *cases.VerifyingNonExistentBlock:
			state.VerifyingNonExistentBlock = cfg

		case *cases.NotarisingNonExistentBlock:
			state.NotarisingNonExistentBlock = cfg

		case *cases.ResendProposedBlock:
			state.ResendProposedBlock = cfg

		case *cases.ResendNotarisation:
			state.ResendNotarisation = cfg

		case *cases.BadTimeoutVRFS:
			state.BadTimeoutVRFS = cfg

		case *cases.HalfNodesDown:
			state.HalfNodesDown = cfg

		case *cases.BlockStateChangeRequestor:
			state.BlockStateChangeRequestor = cfg

		case *cases.MinerNotarisedBlockRequestor:
			state.MinerNotarisedBlockRequestor = cfg

		case *cases.FBRequestor:
			state.FBRequestor = cfg

		case *cases.MissingLFBTickets:
			state.MissingLFBTicket = cfg

		case *cases.CheckChallengeIsValid:
			state.CheckChallengeIsValid = cfg

		case *cases.RoundHasFinalized:
			state.RoundHasFinalizedConfig = cfg

		case *cases.RoundRandomSeed:
			state.RoundRandomSeed = cfg

		default:
			log.Panicf("unknown test case name: %s", configurator.Name())
		}
	})
	if err != nil {
		return fmt.Errorf("error while updating all states on \"%s\" test case: %v", configurator.Name(), err)
	}

	r.server.CurrentTest = configurator.TestCase()
	r.currTestCaseName = configurator.Name()

	switch cfg := configurator.(type) {
	case *cases.RoundHasFinalized:
		_ = r.server.CurrentTest.Configure([]byte(strconv.Itoa(cfg.Round)))
	}

	return nil
}

// MakeTestCaseCheck implements config.Executor interface.
func (r *Runner) MakeTestCaseCheck(cfg *config.TestCaseCheck) error {
	if r.verbose {
		log.Print(" [INF] making test case check")
	}

	if r.server.CurrentTest == nil {
		return errors.New("check is not set up")
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.WaitTime)
	defer cancel()
	success, err := r.server.CurrentTest.Check(ctx)
	if err != nil {
		return err
	}
	if !success {
		return errors.New("check failed")
	}
	return nil
}

// SetServerState implements config.Executor interface.
func (r *Runner) SetServerState(update interface{}) error {
	err := r.server.UpdateAllStates(func(state *conductrpc.State) {
		switch update := update.(type) {
		case *config.BlobberList:
			state.BlobberList = update
		case *config.BlobberDownload:
			state.BlobberDownload = update
		case *config.BlobberUpload:
			state.BlobberUpload = update
		case *config.BlobberDelete:
			state.BlobberDelete = update
		case *config.AdversarialValidator:
			state.AdversarialValidator = update
		case *config.LockNotarizationAndSendNextRoundVRF:
			state.LockNotarizationAndSendNextRoundVRF = update
		case *config.CollectVerificationTicketsWhenMissedVRF:
			state.CollectVerificationTicketsWhenMissedVRF = update
		case *config.AdversarialAuthorizer:
			state.AdversarialAuthorizer = update
		case *config.NotifyOnBlockGeneration:
			state.NotifyOnBlockGeneration = update.Enable
		case config.StopChallengeGeneration:
			state.StopChallengeGeneration = bool(update)
		case config.StopWMCommit:
			state.StopWMCommit = true
		case config.BlobberCommittedWM:
			state.BlobberCommittedWM = true
		case *config.GenerateChallege:
			state.GenerateChallenge = update
		case config.GetFileMetaRoot:
			state.GetFileMetaRoot = bool(update)
		case config.GenerateAllChallenges:
			state.GenerateAllChallenges = bool(update)
		case *config.RenameCommitControl:
			if update.Fail {
				state.FailRenameCommit = utils.SliceUnion(state.FailRenameCommit, update.Nodes)
			} else {
				state.FailRenameCommit = utils.SliceDifference(state.FailRenameCommit, update.Nodes)
			}
			fmt.Printf("state.FailRenameCommit = %v\n", state.FailRenameCommit)
		case *config.UploadCommitControl:
			if update.Fail {
				state.FailUploadCommit = utils.SliceUnion(state.FailUploadCommit, update.Nodes)
			} else {
				state.FailUploadCommit = utils.SliceDifference(state.FailUploadCommit, update.Nodes)
			}
			fmt.Printf("state.FailUploadCommit = %v\n", state.FailUploadCommit)
		case config.NotifyOnValidationTicketGeneration:
			state.NotifyOnValidationTicketGeneration = bool(update)
		case config.MissUpDownload:
			state.MissUpDownload = bool(update)
		}
	})

	return err
}

// SetMagicBlock implements config.Executor interface.
func (r *Runner) SetMagicBlock(configFile string) error {
	if r.verbose {
		log.Print(" [INF] Setting magic block configuration file ", configFile)
	}

	r.server.SetMagicBlock(configFile)

	return nil
}

func (r *Runner) SyncLatestAggregates(cfg *config.SyncAggregates) error {
	if r.verbose {
		log.Printf("[INF] syncing aggregates, %+v\n", cfg)
	}

	aggService := services.NewAggregateService(r.conf.AggregatesBaseUrl)

	if len(cfg.MinerIds) > 0 {
		err := aggService.SyncLatestAggregates(types.Miner, cfg.MinerIds)
		if err != nil && cfg.Required {
			return fmt.Errorf("error syncing miner aggregates: %v", err)
		}
	}

	if len(cfg.SharderIds) > 0 {
		err := aggService.SyncLatestAggregates(types.Sharder, cfg.SharderIds)
		if err != nil && cfg.Required {
			return fmt.Errorf("error syncing sharder aggregates: %v", err)
		}
	}

	if len(cfg.BlobberIds) > 0 {
		err := aggService.SyncLatestAggregates(types.Blobber, cfg.BlobberIds)
		if err != nil && cfg.Required {
			return fmt.Errorf("error syncing blobber aggregates: %v", err)
		}
	}

	if len(cfg.ValidatorIds) > 0 {
		err := aggService.SyncLatestAggregates(types.Validator, cfg.ValidatorIds)
		if err != nil && cfg.Required {
			return fmt.Errorf("error syncing validator aggregates: %v", err)
		}
	}

	if len(cfg.AuthorizerIds) > 0 {
		err := aggService.SyncLatestAggregates(types.Authorizer, cfg.AuthorizerIds)
		if err != nil && cfg.Required {
			return fmt.Errorf("error syncing authorizer aggregates: %v", err)
		}
	}

	if len(cfg.UserIds) > 0 {
		err := aggService.SyncLatestAggregates(types.User, cfg.UserIds)
		if err != nil && cfg.Required {
			return fmt.Errorf("error syncing user aggregates: %v", err)
		}
	}

	if cfg.MonitorGlobal {
		err := aggService.SyncLatestAggregates(types.Global, []string{})
		if err != nil && cfg.Required {
			return fmt.Errorf("error syncing monitor aggregates: %v", err)
		}
	}

	return nil
}

func (r *Runner) CheckAggregateValueChange(cfg *config.CheckAggregateChange, tm time.Duration) error {
	if r.verbose {
		log.Printf("[INF] checking aggregate value change: %+v", cfg)
	}

	aggService := services.NewAggregateService(r.conf.AggregatesBaseUrl)

	check, err := aggService.CheckAggregateValueChange(cfg.ProviderType, cfg.ProviderId, cfg.Key, cfg.Monotonicity, tm)
	if err != nil {
		return err
	}

	if !check {
		return fmt.Errorf("aggregate value not changed: %v", cfg)
	}

	return nil
}

func (r *Runner) CheckAggregateValueComparison(cfg *config.CheckAggregateComparison, tm time.Duration) error {
	if r.verbose {
		log.Printf("[INF] checking aggregate value comparison: %+v", cfg)
	}

	aggService := services.NewAggregateService(r.conf.AggregatesBaseUrl)

	check, err := aggService.CompareAggregateValue(cfg.ProviderType, cfg.ProviderId, cfg.Key, cfg.Comparison, cfg.RValue, tm)
	if err != nil {
		return err
	}

	if !check {
		return fmt.Errorf("aggregate comparison failed: %v", cfg)
	}

	return nil
}

func (r *Runner) CheckRollbackTokenomicsComparison() error {
	if r.verbose {
		log.Printf("[INF] checking rollback tokenomics comparison")
	}

	allocationService := services.NewAllocationService(r.conf.Sharder1BaseURL)

	check, err := allocationService.CompareRollBackTokens()
	if err != nil {
		return err
	}

	if !check {
		return fmt.Errorf("aggregate comparison failed")
	}

	return nil
}

func (r *Runner) StoreAllocationsData() error {
	if r.verbose {
		log.Printf("[INF] storing allocations data : " + r.conf.Sharder1BaseURL)
	}

	allocationService := services.NewAllocationService(r.conf.Sharder1BaseURL)

	err := allocationService.StoreAllocationsData()
	if err != nil {
		return err
	}

	return nil
}

func (r *Runner) SetNodeCustomConfig(cfg *config.NodeCustomConfig) error {
	if r.verbose {
		log.Printf("[INF] setting node custom config: %+v", cfg)
	}

	node, ok := r.conf.Nodes.NodeByName(cfg.NodeName)
	if !ok {
		return fmt.Errorf("node not found: %v", cfg.NodeName)
	}

	return r.server.SetNodeConfig(node.ID, cfg.Config)
}

func (r *Runner) SetMissUpDownload(cfg config.MissUpDownload) error {
	if r.verbose {
		log.Printf("[INF] setting miss up download: %+v", cfg)
	}

	return r.SetServerState(cfg)
}