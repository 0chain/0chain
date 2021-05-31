package config

import (
	"fmt"
	"time"

	"github.com/mitchellh/mapstructure"
)

// Executor used by a Flow to perform a flow directive.
type Executor interface {

	// common setups

	SetMonitor(name NodeName) (err error)
	CleanupBC(timeout time.Duration) (err error)

	// common control

	Start(names []NodeName, lock bool, timeout time.Duration) (err error)
	Unlock(names []NodeName, timeout time.Duration) (err error)
	Stop(names []NodeName, timeout time.Duration) (err error)

	// VC misbehavior

	SetRevealed(miners []NodeName, pin bool, tm time.Duration) (err error)

	// waiting

	WaitViewChange(vc WaitViewChange, timeout time.Duration) (err error)
	WaitPhase(wp WaitPhase, timeout time.Duration) (err error)
	WaitRound(wr WaitRound, timeout time.Duration) (err error)
	WaitContributeMpk(wcmpk WaitContributeMpk, timeout time.Duration) (err error)
	WaitShareSignsOrShares(ssos WaitShareSignsOrShares, timeout time.Duration) (err error)
	WaitAdd(wadd WaitAdd, timeout time.Duration) (err error)
	WaitNoProgress(wait time.Duration) (err error)
	WaitNoViewChainge(wnvc WaitNoViewChainge, timeout time.Duration) (err error)
	WaitSharderKeep(wsk WaitSharderKeep, timeout time.Duration) (err error)

	// Byzantine: BC, sharders

	FinalizedBlock(fb *Bad) (err error)
	MagicBlock(mb *Bad) (err error)
	VerifyTransaction(vt *Bad) (err error)

	// Byzantine: BC tests, miners misbehavior

	VRFS(vrfs *Bad) (err error)
	RoundTimeout(rt *Bad) (err error)
	CompetingBlock(cb *Bad) (err error)
	SignOnlyCompetingBlocks(socb *Bad) (err error)
	DoubleSpendTransaction(dst *Bad) (err error)
	WrongBlockSignHash(wbsh *Bad) (err error)
	WrongBlockSignKey(wbsk *Bad) (err error)
	WrongBlockHash(wbh *Bad) (err error)
	VerificationTicketGroup(vtg *Bad) (err error)
	WrongVerificationTicketHash(wvth *Bad) (err error)
	WrongVerificationTicketKey(wvtk *Bad) (err error)
	WrongNotarizedBlockHash(wnbh *Bad) (err error)
	WrongNotarizedBlockKey(wnbk *Bad) (err error)
	NotarizeOnlyCompetingBlock(ncb *Bad) (err error)
	NotarizedBlock(nb *Bad) (err error)
	MPK(mpk *Bad) (err error)
	Shares(s *Bad) (err error)
	Signatures(s *Bad) (err error)
	Publish(p *Bad) (err error)

	// system command (a bash script, etc)
	Command(name string, timeout time.Duration)

	// Blobber related executors
	StorageTree(st *Bad) (err error)
	ValidatorProof(vp *Bad) (err error)
	Challenges(cs *Bad) (err error)
}

//
// common setups
//

func setMonitor(ex Executor, val interface{}, tm time.Duration) (
	err error) {

	if ss, ok := getNodeNames(val); ok && len(ss) == 1 {
		return ex.SetMonitor(ss[0])
	}
	return fmt.Errorf("invalid 'set_monitor' argument type: %T", val)
}

//
// common nodes control (start / stop, lock / unlock)
//

func start(name string, ex Executor, val interface{}, lock bool,
	tm time.Duration) (err error) {

	if ss, ok := getNodeNames(val); ok {
		return ex.Start(ss, lock, tm)
	}
	return fmt.Errorf("invalid '%s' argument type: %T", name, val)
}

func unlock(ex Executor, val interface{}, tm time.Duration) (
	err error) {

	if ss, ok := getNodeNames(val); ok {
		return ex.Unlock(ss, tm)
	}
	return fmt.Errorf("invalid 'unlock' argument type: %T", val)
}

func stop(ex Executor, val interface{}, tm time.Duration) (
	err error) {

	if ss, ok := getNodeNames(val); ok {
		return ex.Stop(ss, tm)
	}
	return fmt.Errorf("invalid 'stop' argument type: %T", val)
}

//
// wait for an event of the monitor
//

func waitViewChange(ex Executor, val interface{}, tm time.Duration) (
	err error) {

	var vc WaitViewChange
	if err = mapstructure.Decode(val, &vc); err != nil {
		return fmt.Errorf("invalid 'wait_view_change' argument type: %T, "+
			"decoding error: %v", val, err)
	}
	return ex.WaitViewChange(vc, tm)
}

func waitPhase(ex Executor, val interface{}, tm time.Duration) (
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

func waitRound(ex Executor, val interface{}, tm time.Duration) (
	err error) {

	var wr WaitRound
	if err = mapstructure.Decode(val, &wr); err != nil {
		return fmt.Errorf("decoding 'wait_round': %v", err)
	}
	return ex.WaitRound(wr, tm)
}

func waitContributeMpk(ex Executor, val interface{}, tm time.Duration) (
	err error) {

	var wcmpk WaitContributeMpk
	if err = mapstructure.Decode(val, &wcmpk); err != nil {
		return fmt.Errorf("decoding 'wait_contribute_mpk': %v", err)
	}
	return ex.WaitContributeMpk(wcmpk, tm)
}

func waitShareSignsOrShares(ex Executor, val interface{}, tm time.Duration) (
	err error) {

	var wsoss WaitShareSignsOrShares
	if err = mapstructure.Decode(val, &wsoss); err != nil {
		return fmt.Errorf("decoding 'wait_share_signs_or_shares': %v", err)
	}
	return ex.WaitShareSignsOrShares(wsoss, tm)
}

func waitAdd(ex Executor, val interface{}, tm time.Duration) (
	err error) {

	var wa WaitAdd
	if err = mapstructure.Decode(val, &wa); err != nil {
		return fmt.Errorf("decoding 'wait_add': %v", err)
	}
	return ex.WaitAdd(wa, tm)
}

func waitNoViewChainge(ex Executor, val interface{},
	tm time.Duration) (err error) {

	var wnvc WaitNoViewChainge
	if err = mapstructure.Decode(val, &wnvc); err != nil {
		return fmt.Errorf("decoding 'wait_no_view_change': %v", err)
	}
	return ex.WaitNoViewChainge(wnvc, tm)
}

func waitNoProgress(ex Executor, tm time.Duration) (err error) {
	return ex.WaitNoProgress(tm)
}

func waitSharderKeep(ex Executor, val interface{},
	tm time.Duration) (err error) {

	var wsk WaitSharderKeep
	if err = mapstructure.Decode(val, &wsk); err != nil {
		return fmt.Errorf("decoding 'wait_sharder_keep': %v", err)
	}
	return ex.WaitSharderKeep(wsk, tm)
}

//
// control nodes behavior / misbehavior (view change)
//

func setRevealed(name string, ex Executor, val interface{}, pin bool,
	tm time.Duration) (err error) {

	if ss, ok := getNodeNames(val); ok {
		return ex.SetRevealed(ss, pin, tm)
	}
	return fmt.Errorf("invalid '%s' argument type: %T", name, val)
}
