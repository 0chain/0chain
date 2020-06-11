package config

import (
	"time"
)

type flowExecuteFunc func(f Flow, name string, ex Executor, val interface{},
	tm time.Duration) (err error)

var flowRegistry = make(map[string]flowExecuteFunc)

func register(name string, fn flowExecuteFunc) {
	flowRegistry[name] = fn
}

func init() {

	// common setups

	register("set_monitor", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return f.setMonitor(ex, val, tm)
	})
	register("cleanup_bc", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return ex.CleanupBC(tm)
	})

	// common nodes control (start / stop, lock / unlock)

	register("start", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return f.start(name, ex, val, false, tm)
	})
	register("start_lock", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return f.start(name, ex, val, true, tm)
	})
	register("unlock", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return f.unlock(ex, val, tm)
	})
	register("stop", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return f.stop(ex, val, tm)
	})

	// wait for an event of the monitor

	register("wait_view_change", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return f.waitViewChange(ex, val, tm)
	})
	register("wait_phase", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return f.waitPhase(ex, val, tm)
	})
	register("wait_round", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return f.waitRound(ex, val, tm)
	})
	register("wait_contribute_mpk", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return f.waitContributeMpk(ex, val, tm)
	})
	register("wait_share_signs_or_shares", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return f.waitShareSignsOrShares(ex, val, tm)
	})
	register("wait_add", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return f.waitAdd(ex, val, tm)
	})
	register("wait_no_progress", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return f.waitNoProgress(ex, tm)
	})

	// control nodes behavior / misbehavior (view change)

	register("send_share_only", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return f.sendShareOnly(ex, val, tm)
	})
	register("send_share_bad", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return f.sendShareBad(ex, val, tm)
	})
	register("set_revealed", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return f.setRevealed(name, ex, val, true, tm)
	})
	register("unset_revealed", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return f.setRevealed(name, ex, val, false, tm)
	})

}
