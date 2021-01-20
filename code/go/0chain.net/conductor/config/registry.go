package config

import (
	"fmt"
	"time"

	"github.com/mitchellh/mapstructure"
)

type flowExecuteFunc func(name string, ex Executor, val interface{},
	tm time.Duration) (err error)

var flowRegistry = make(map[string]flowExecuteFunc)

func register(name string, fn flowExecuteFunc) {
	flowRegistry[name] = fn
}

func init() {

	// common setups

	register("set_monitor", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return setMonitor(ex, val, tm)
	})
	register("cleanup_bc", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return ex.CleanupBC(tm)
	})

	// common nodes control (start / stop, lock / unlock)

	register("start", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return start(name, ex, val, false, tm)
	})
	register("start_lock", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return start(name, ex, val, true, tm)
	})
	register("unlock", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return unlock(ex, val, tm)
	})
	register("stop", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return stop(ex, val, tm)
	})

	// wait for an event of the monitor

	register("wait_view_change", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return waitViewChange(ex, val, tm)
	})
	register("wait_phase", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return waitPhase(ex, val, tm)
	})
	register("wait_round", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return waitRound(ex, val, tm)
	})
	register("wait_contribute_mpk", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return waitContributeMpk(ex, val, tm)
	})
	register("wait_share_signs_or_shares", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return waitShareSignsOrShares(ex, val, tm)
	})
	register("wait_add", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return waitAdd(ex, val, tm)
	})
	register("wait_no_progress", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return waitNoProgress(ex, tm)
	})
	register("wait_no_view_change", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return waitNoViewChainge(ex, val, tm)
	})
	register("wait_sharder_keep", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return waitSharderKeep(ex, val, tm)
	})

	// control nodes behavior / misbehavior (view change)

	register("set_revealed", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return setRevealed(name, ex, val, true, tm)
	})
	register("unset_revealed", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return setRevealed(name, ex, val, false, tm)
	})

	// Byzantine blockchain.

	register("vrfs", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var vrfs Bad
		if err = vrfs.Unmarshal(name, val); err != nil {
			return
		}
		return ex.VRFS(&vrfs)
	})

	register("round_timeout", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var rt Bad
		if err = rt.Unmarshal(name, val); err != nil {
			return
		}
		return ex.RoundTimeout(&rt)
	})

	register("competing_block", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var cb Bad
		if err = cb.Unmarshal(name, val); err != nil {
			return
		}
		return ex.CompetingBlock(&cb)
	})

	register("sign_only_competing_blocks", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var socb Bad
		if err = socb.Unmarshal(name, val); err != nil {
			return
		}
		return ex.SignOnlyCompetingBlocks(&socb)
	})

	register("double_spend_transaction", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var dst Bad
		if err = dst.Unmarshal(name, val); err != nil {
			return
		}
		return ex.DoubleSpendTransaction(&dst)
	})

	register("wrong_block_sign_hash", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var wbsh Bad
		if err = wbsh.Unmarshal(name, val); err != nil {
			return
		}
		return ex.WrongBlockSignHash(&wbsh)
	})

	register("wrong_block_sign_key", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var wbsk Bad
		if err = wbsk.Unmarshal(name, val); err != nil {
			return
		}
		return ex.WrongBlockSignKey(&wbsk)
	})

	register("wrong_block_hash", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var wbh Bad
		if err = wbh.Unmarshal(name, val); err != nil {
			return
		}
		return ex.WrongBlockHash(&wbh)
	})

	register("verification_ticket_group", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var vtg Bad
		if err = vtg.Unmarshal(name, val); err != nil {
			return
		}
		return ex.VerificationTicketGroup(&vtg)
	})

	register("wrong_verification_ticket_hash", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var wvth Bad
		if err = wvth.Unmarshal(name, val); err != nil {
			return
		}
		return ex.WrongVerificationTicketHash(&wvth)
	})

	register("wrong_verification_ticket_key", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var wvtk Bad
		if err = wvtk.Unmarshal(name, val); err != nil {
			return
		}
		return ex.WrongVerificationTicketKey(&wvtk)
	})

	register("wrong_notarized_block_hash", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var wnth Bad
		if err = wnth.Unmarshal(name, val); err != nil {
			return
		}
		return ex.WrongNotarizedBlockHash(&wnth)
	})

	register("wrong_notarized_block_key", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var wnbk Bad
		if err = wnbk.Unmarshal(name, val); err != nil {
			return
		}
		return ex.WrongNotarizedBlockKey(&wnbk)
	})

	register("notarize_only_competing_block", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var nocb Bad
		if err = nocb.Unmarshal(name, val); err != nil {
			return
		}
		return ex.NotarizeOnlyCompetingBlock(&nocb)
	})

	register("notarized_block", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var nb Bad
		if err = nb.Unmarshal(name, val); err != nil {
			return
		}
		return ex.NotarizedBlock(&nb)
	})

	// Byzantine blockchain sharders

	register("finalized_block", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var fb Bad
		if err = fb.Unmarshal(name, val); err != nil {
			return
		}
		return ex.FinalizedBlock(&fb)
	})

	register("magic_block", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var mb Bad
		if err = mb.Unmarshal(name, val); err != nil {
			return
		}
		return ex.MagicBlock(&mb)
	})

	register("verify_transaction", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var vt Bad
		if err = vt.Unmarshal(name, val); err != nil {
			return
		}
		return ex.VerifyTransaction(&vt)
	})

	// Byzantine view change

	register("mpk", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var mpk Bad
		if err = mpk.Unmarshal(name, val); err != nil {
			return
		}
		return ex.MPK(&mpk)
	})

	register("share", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var shares Bad
		if err = shares.Unmarshal(name, val); err != nil {
			return
		}
		return ex.Shares(&shares)
	})

	register("signature", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var signatures Bad
		if err = signatures.Unmarshal(name, val); err != nil {
			return
		}
		return ex.Signatures(&signatures)
	})

	register("publish", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var publish Bad
		if err = publish.Unmarshal(name, val); err != nil {
			return
		}
		return ex.Publish(&publish)
	})

	// a system command

	register("command", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var cn CommandName
		if err = mapstructure.Decode(val, &cn); err != nil {
			return fmt.Errorf("decoding '%s': %v", name, err)
		}
		ex.Command(cn.Name, tm) // async command
		return nil
	})

	// Blobber related executors

	register("storage_tree", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var st Bad
		if err = mapstructure.Decode(val, &st); err != nil {
			return fmt.Errorf("decoding '%s': %v", name, err)
		}
		return ex.StorageTree(&st)
	})

	register("validator_proof", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var vp Bad
		if err = mapstructure.Decode(val, &vp); err != nil {
			return fmt.Errorf("decoding '%s': %v", name, err)
		}
		return ex.ValidatorProof(&vp)
	})

	register("challenges", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var cs Bad
		if err = mapstructure.Decode(val, &cs); err != nil {
			return fmt.Errorf("decoding '%s': %v", name, err)
		}
		return ex.Challenges(&cs)
	})

}
