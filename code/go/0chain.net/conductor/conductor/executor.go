package main

import (
	"fmt"
	"log"
	"time"

	"0chain.net/conductor/conductrpc"
	"0chain.net/conductor/config"
)

//
// execute (the config.Executor implementation)
//

func (r *Runner) setupTimeout(tm time.Duration) {
	r.timer = time.NewTimer(tm)
	if tm <= 0 {
		<-r.timer.C // drain zero timeout
	}
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
	return r.conf.CleanupBC()
}

//
// control nodes
//

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

		// miners and sharders, but skip blobbers
		if !r.conf.IsSkipWait(name) {
			r.server.AddNode(name, lock)   // expected server interaction
			r.waitNodes[name] = struct{}{} // wait list
		}

		if err = n.Start(r.conf.Logs); err != nil {
			return fmt.Errorf("starting %s: %v", n.Name, err)
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
			n.Kill()
		}
		log.Print(n.Name, " stopped")
	}
	return
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
			wr.Round = r.lastRound + wr.Shift
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
		log.Printf(" [INF] wait add miners: %s, sharders: %s, blobbers: %s",
			wadd.Miners, wadd.Sharders, wadd.Blobbers)
	}

	r.setupTimeout(tm)
	r.waitAdd = wadd
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

	r.waitNoProgressUntil = time.Now().Add(wait)
	r.setupTimeout(wait)
	return
}

//
// Byzantine blockchain miners.
//

func (r *Runner) VRFS(vrfs *config.Bad) (err error) {
	if r.verbose {
		log.Printf(" [INF] set VRFS of %s: good %s, bad %s", vrfs.By,
			vrfs.Good, vrfs.Bad)
	}

	err = r.server.UpdateStates(vrfs.By, func(state *conductrpc.State) {
		state.VRFS = vrfs
	})
	if err != nil {
		return fmt.Errorf("setting VRFS: %v", err)
	}
	return
}

func (r *Runner) RoundTimeout(rt *config.Bad) (err error) {

	if r.verbose {
		log.Printf(" [INF] set 'wrong round timeout' "+
			"of %s: good %s, bad %s", rt.By, rt.Good, rt.Bad)
	}
	err = r.server.UpdateStates(rt.By, func(state *conductrpc.State) {
		state.RoundTimeout = rt
	})
	if err != nil {
		return fmt.Errorf("setting 'wrong round timeout': %v", err)
	}
	return
}

func (r *Runner) CompetingBlock(cb *config.Bad) (err error) {

	if r.verbose {
		log.Printf(" [INF] set 'competing block' "+
			"of %s: good %s, bad %s", cb.By, cb.Good, cb.Bad)
	}
	err = r.server.UpdateStates(cb.By, func(state *conductrpc.State) {
		state.CompetingBlock = cb
	})
	if err != nil {
		return fmt.Errorf("setting 'competing block': %v", err)
	}
	return
}

func (r *Runner) SignOnlyCompetingBlocks(socb *config.Bad) (err error) {

	if r.verbose {
		log.Printf(" [INF] set 'sign only competing block' "+
			"of %s: good %s, bad %s", socb.By, socb.Good, socb.Bad)
	}
	err = r.server.UpdateStates(socb.By, func(state *conductrpc.State) {
		state.SignOnlyCompetingBlocks = socb
	})
	if err != nil {
		return fmt.Errorf("setting 'sign only competing block': %v", err)
	}
	return
}

func (r *Runner) DoubleSpendTransaction(dst *config.Bad) (err error) {

	if r.verbose {
		log.Printf(" [INF] set 'double spend transaction' "+
			"of %s: good %s, bad %s", dst.By, dst.Good, dst.Bad)
	}
	err = r.server.UpdateStates(dst.By, func(state *conductrpc.State) {
		state.DoubleSpendTransaction = dst
	})
	if err != nil {
		return fmt.Errorf("setting 'double spend transaction': %v", err)
	}
	return
}

func (r *Runner) WrongBlockSignHash(wbsh *config.Bad) (err error) {

	if r.verbose {
		log.Printf(" [INF] set 'wrong block sign hash' "+
			"of %s: good %s, bad %s", wbsh.By, wbsh.Good, wbsh.Bad)
	}
	err = r.server.UpdateStates(wbsh.By, func(state *conductrpc.State) {
		state.WrongBlockSignHash = wbsh
	})
	if err != nil {
		return fmt.Errorf("setting 'wrong block sign hash': %v", err)
	}
	return
}

func (r *Runner) WrongBlockSignKey(wbsk *config.Bad) (err error) {

	if r.verbose {
		log.Printf(" [INF] set 'wrong block sign key' "+
			"of %s: good %s, bad %s", wbsk.By, wbsk.Good, wbsk.Bad)
	}
	err = r.server.UpdateStates(wbsk.By, func(state *conductrpc.State) {
		state.WrongBlockSignKey = wbsk
	})
	if err != nil {
		return fmt.Errorf("setting 'wrong block sign key': %v", err)
	}
	return
}

func (r *Runner) WrongBlockHash(wbh *config.Bad) (err error) {

	if r.verbose {
		log.Printf(" [INF] set 'wrong block hash' "+
			"of %s: good %s, bad %s", wbh.By, wbh.Good, wbh.Bad)
	}
	err = r.server.UpdateStates(wbh.By, func(state *conductrpc.State) {
		state.WrongBlockHash = wbh
	})
	if err != nil {
		return fmt.Errorf("setting 'wrong block hash': %v", err)
	}
	return
}

func (r *Runner) VerificationTicketGroup(vtg *config.Bad) (err error) {

	if r.verbose {
		log.Printf(" [INF] set 'verification_ticket_group' of %s", vtg.By)
	}
	err = r.server.UpdateStates(vtg.By, func(state *conductrpc.State) {
		state.VerificationTicketGroup = vtg
	})
	if err != nil {
		return fmt.Errorf("setting 'verification_ticket_group': %v", err)
	}
	return
}

func (r *Runner) WrongVerificationTicketHash(wvth *config.Bad) (err error) {

	if r.verbose {
		log.Printf(" [INF] set 'wrong verification ticket hash' "+
			"of %s: good %s, bad %s", wvth.By, wvth.Good, wvth.Bad)
	}
	err = r.server.UpdateStates(wvth.By, func(state *conductrpc.State) {
		state.WrongVerificationTicketHash = wvth
	})
	if err != nil {
		return fmt.Errorf("setting 'wrong verification ticket hash': %v", err)
	}
	return
}

func (r *Runner) WrongVerificationTicketKey(wvtk *config.Bad) (err error) {

	if r.verbose {
		log.Printf(" [INF] set 'wrong verification ticket key' "+
			"of %s: good %s, bad %s", wvtk.By, wvtk.Good, wvtk.Bad)
	}
	err = r.server.UpdateStates(wvtk.By, func(state *conductrpc.State) {
		state.WrongVerificationTicketKey = wvtk
	})
	if err != nil {
		return fmt.Errorf("setting 'wrong verification ticket key': %v", err)
	}
	return
}

func (r *Runner) WrongNotarizedBlockHash(wnbh *config.Bad) (err error) {

	if r.verbose {
		log.Printf(" [INF] set 'wrong notarized block hash' "+
			"of %s: good %s, bad %s", wnbh.By, wnbh.Good, wnbh.Bad)
	}
	err = r.server.UpdateStates(wnbh.By, func(state *conductrpc.State) {
		state.WrongNotarizedBlockHash = wnbh
	})
	if err != nil {
		return fmt.Errorf("setting 'wrong notarized block hash': %v", err)
	}
	return
}

func (r *Runner) WrongNotarizedBlockKey(wnbk *config.Bad) (err error) {

	if r.verbose {
		log.Printf(" [INF] set 'wrong notarized block key' "+
			"of %s: good %s, bad %s", wnbk.By, wnbk.Good, wnbk.Bad)
	}
	err = r.server.UpdateStates(wnbk.By, func(state *conductrpc.State) {
		state.WrongNotarizedBlockKey = wnbk
	})
	if err != nil {
		return fmt.Errorf("setting 'wrong notarized block key': %v", err)
	}
	return
}

func (r *Runner) NotarizeOnlyCompetingBlock(ncb *config.Bad) (err error) {

	if r.verbose {
		log.Printf(" [INF] set 'notarize only competing block' "+
			"of %s: good %s, bad %s", ncb.By, ncb.Good, ncb.Bad)
	}
	err = r.server.UpdateStates(ncb.By, func(state *conductrpc.State) {
		state.NotarizeOnlyCompetingBlock = ncb
	})
	if err != nil {
		return fmt.Errorf("setting 'notarized only competing block': %v", err)
	}
	return
}

func (r *Runner) NotarizedBlock(nb *config.Bad) (err error) {

	if r.verbose {
		log.Printf(" [INF] set 'notarized block' of %s: good %s, bad %s", nb.By,
			nb.Good, nb.Bad)
	}
	err = r.server.UpdateStates(nb.By, func(state *conductrpc.State) {
		state.NotarizedBlock = nb
	})
	if err != nil {
		return fmt.Errorf("setting 'notarized block': %v", err)
	}
	return
}

//
// Byzantine VC miners.
//

func (r *Runner) SetRevealed(ss []NodeName, pin bool, tm time.Duration) (
	err error) {

	if r.verbose {
		log.Printf(" [INF] set reveled of %s to %t", ss, pin)
	}

	err = r.server.UpdateStates(ss, func(state *conductrpc.State) {
		state.IsRevealed = pin
	})
	if err != nil {
		return fmt.Errorf("setting revealed to %t nodes: %v", pin, err)
	}
	return
}

func (r *Runner) MPK(mpk *config.Bad) (err error) {
	if r.verbose {
		log.Printf(" [INF] set 'MPK' of %s: good %s, bad %s", mpk.By,
			mpk.Good, mpk.Bad)
	}
	err = r.server.UpdateStates(mpk.By, func(state *conductrpc.State) {
		state.MPK = mpk
	})
	if err != nil {
		return fmt.Errorf("setting 'MPK': %v", err)
	}
	return
}

func (r *Runner) Shares(s *config.Bad) (err error) {
	if r.verbose {
		log.Printf(" [INF] set 'shares' of %s: good %s, bad %s", s.By,
			s.Good, s.Bad)
	}
	err = r.server.UpdateStates(s.By, func(state *conductrpc.State) {
		state.Shares = s
	})
	if err != nil {
		return fmt.Errorf("setting 'shares': %v", err)
	}
	return
}

func (r *Runner) Signatures(s *config.Bad) (err error) {
	if r.verbose {
		log.Printf(" [INF] set 'signatures' of %s: good %s, bad %s", s.By,
			s.Good, s.Bad)
	}
	err = r.server.UpdateStates(s.By, func(state *conductrpc.State) {
		state.Signatures = s
	})
	if err != nil {
		return fmt.Errorf("setting 'signatures': %v", err)
	}
	return
}

func (r *Runner) Publish(p *config.Bad) (err error) {
	if r.verbose {
		log.Printf(" [INF] set 'publish' of %s: good %s, bad %s", p.By,
			p.Good, p.Bad)
	}

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
	if r.verbose {
		log.Printf(" [INF] set 'finalized block' of %s: good %s, bad %s", fb.By,
			fb.Good, fb.Bad)
	}

	err = r.server.UpdateStates(fb.By, func(state *conductrpc.State) {
		state.FinalizedBlock = fb
	})
	if err != nil {
		return fmt.Errorf("setting 'finalized block': %v", err)
	}
	return
}

func (r *Runner) MagicBlock(mb *config.Bad) (err error) {
	if r.verbose {
		log.Printf(" [INF] set 'magic block' of %s: good %s, bad %s", mb.By,
			mb.Good, mb.Bad)
	}

	err = r.server.UpdateStates(mb.By, func(state *conductrpc.State) {
		state.MagicBlock = mb
	})
	if err != nil {
		return fmt.Errorf("setting 'magic block': %v", err)
	}
	return
}

func (r *Runner) VerifyTransaction(vt *config.Bad) (err error) {
	if r.verbose {
		log.Printf(" [INF] set bad 'verify transaction' of %s to clients",
			vt.By)
	}

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
func (r *Runner) Command(name string, tm time.Duration) {
	r.setupTimeout(tm)

	if r.verbose {
		log.Printf(" [INF] command %q", name)
	}

	r.waitCommand = r.asyncCommand(name)
}

func (r *Runner) asyncCommand(name string) (reply chan error) {
	reply = make(chan error)
	go r.runAsyncCommand(reply, name)
	return
}

func (r *Runner) runAsyncCommand(reply chan error, name string) {
	var err = r.conf.Execute(name)
	if err != nil {
		err = fmt.Errorf("%q: %v", name, err)
	}
	reply <- err // nil or error
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
