package main

//
// execute (the config.Executor implementation)
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
		log.Printf(" [INF] wait add miners: %s, sharders: %s",
			wadd.Miners, wadd.Sharders)
	}

	r.setupTimeout(tm)
	r.waitAdd = wadd
	return
}

func (r *Runner) SetRevealed(ss []NodeName, pin bool, tm time.Duration) (
	err error) {

	var (
		ssIDs = make([]NodeID, 0, len(ss))
	)

	for _, name := range ss {
		var m, ok = r.conf.Nodes.NodeByName(name)
		if !ok {
			return fmt.Errorf("SetRevealed (%t): unexpected node: %q", pin,
				name)
		}
		ssIDs = append(ssIDs, m.ID)
	}

	r.server.SetRevealed(ssIDs, pin)
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

func (r *Runner) VRFS(vrfs *VRFS) (err error) {
	//
	return
}

func (r *Runner) RoundTimeout(rt *RoundTimeout) (err error) {
	//
	return
}

func (r *Runner) CompetingBlock(cb *CompetingBlock) (err error) {
	//
	return
}

func (r *Runner) SignOnlyCompetingBlocks(socb *SignOnlyCompetingBlocks) (
	err error) {

	return
}

func (r *Runner) DoubleSpendTransaction(dst *DoubleSpendTransaction) (
	err error) {

	//
	return
}

func (r *Runner) WrongBlockSignHash(wbsh *WrongBlockSignHash) (err error) {
	//
	return
}

func (r *Runner) WrongBlockSignKey(wbsk *WrongBlockSignKey) (err error) {
	//
	return
}

func (r *Runner) WrongBlockHash(wbh *WrongBlockHash) (err error) {
	//
	return
}

func (r *Runner) VerificationTicket(vt *VerificationTicket) (err error) {
	//
	return
}

func (r *Runner) WrongVerificationTicketHash(
	wvth *WrongVerificationTicketHash) (err error) {

	//
	return
}

func (r *Runner) WrongVerificationTicketKey(wvtk *WrongVerificationTicketKey) (
	err error) {

	//
	return
}

func (r *Runner) WrongNotarizedBlockHash(wnbh *WrongNotarizedBlockHash) (
	err error) {

	//
	return
}

func (r *Runner) WrongNotarizedBlockKey(wnbk *WrongNotarizedBlockKey) (
	err error) {

	//
	return
}

func (r *Runner) NotarizeOnlyCompetingBlock(ncb *NotarizeOnlyCompetingBlock) (
	err error) {

	//
	return
}

func (r *Runner) NotarizedBlock(nb *NotarizedBlock) (err error) {
	//
	return
}

func (r *Runner) MPK(mpk *MPK) (err error) {
	//
	return
}

func (r *Runner) Shares(s *Shares) (err error) {
	//
	return
}

func (r *Runner) Signatures(s *Signatures) (err error) {
	//
	return
}

func (r *Runner) Publish(p *Publish) (err error) {
	//
	return
}
