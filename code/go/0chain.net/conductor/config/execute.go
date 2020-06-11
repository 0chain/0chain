package config

import (
	"errors"
	"fmt"
	"time"

	"github.com/mitchellh/mapstructure"
)

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
	SendShareOnly(miner NodeName, only []NodeName) (err error)
	SendShareBad(miner NodeName, bad []NodeName) (err error)
	SetRevealed(miners []NodeName, pin bool, tm time.Duration) (err error)
	WaitNoProgress(wait time.Duration) (err error)
}

//
// common setups
//

func (f Flow) setMonitor(ex Executor, val interface{}, tm time.Duration) (
	err error) {

	if ss, ok := getNodeNames(val); ok && len(ss) == 1 {
		return ex.SetMonitor(ss[0])
	}
	return fmt.Errorf("invalid 'set_monitor' argument type: %T", val)
}

//
// common nodes control (start / stop, lock / unlock)
//

func (f Flow) start(name string, ex Executor, val interface{}, lock bool,
	tm time.Duration) (err error) {

	if ss, ok := getNodeNames(val); ok {
		return ex.Start(ss, lock, tm)
	}
	return fmt.Errorf("invalid '%s' argument type: %T", name, val)
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

//
// wait for an event of the monitor
//

func (f Flow) waitViewChange(ex Executor, val interface{}, tm time.Duration) (
	err error) {

	var vc WaitViewChange
	if err = mapstructure.Decode(val, &vc); err != nil {
		return fmt.Errorf("invalid 'wait_view_change' argument type: %T, "+
			"decoding error: %v", val, err)
	}
	return ex.WaitViewChange(vc, tm)
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

func (f Flow) waitNoProgress(ex Executor, tm time.Duration) (err error) {
	return ex.WaitNoProgress(tm)
}

//
// control nodes behavior / misbehavior (view change)
//

func (f Flow) sendShareOnly(ex Executor, val interface{}, tm time.Duration) (
	err error) {

	type sendShareOnly struct {
		Miner NodeName   `mapstructure:"miner"`
		Only  []NodeName `mapstructure:"only"`
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
		Miner NodeName   `mapstructure:"miner"`
		Bad   []NodeName `mapstructure:"bad"`
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

func (f Flow) setRevealed(name string, ex Executor, val interface{}, pin bool,
	tm time.Duration) (err error) {

	if ss, ok := getNodeNames(val); ok {
		return ex.SetRevealed(ss, pin, tm)
	}
	return fmt.Errorf("invalid '%s' argument type: %T", name, val)
}
