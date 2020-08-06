package config

import (
	"fmt"
	"time"

	"github.com/mitchellh/mapstructure"
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
	register("wait_no_view_change", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return f.waitNoViewChainge(ex, val, tm)
	})
	register("wait_sharder_keep", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return f.waitSharderKeep(ex, val, tm)
	})

	// control nodes behavior / misbehavior (view change)

	register("set_revealed", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return f.setRevealed(name, ex, val, true, tm)
	})
	register("unset_revealed", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return f.setRevealed(name, ex, val, false, tm)
	})

	// Byzantine blockchain.

	register("vrfs", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var vrfs Bad
		if err = vrfs.Unmarshal(name, val); err != nil {
			return
		}
		return ex.VRFS(&vrfs)
	})

	register("round_timeout", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var rt Bad
		if err = rt.Unmarshal(name, val); err != nil {
			return
		}
		return ex.RoundTimeout(&rt)
	})

	register("competing_block", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var cb Bad
		if err = cb.Unmarshal(name, val); err != nil {
			return
		}
		return ex.CompetingBlock(&cb)
	})

	register("sign_only_competing_blocks", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var socb Bad
		if err = socb.Unmarshal(name, val); err != nil {
			return
		}
		return ex.SignOnlyCompetingBlocks(&socb)
	})

	register("double_spend_transaction", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var dst Bad
		if err = dst.Unmarshal(name, val); err != nil {
			return
		}
		return ex.DoubleSpendTransaction(&dst)
	})

	register("wrong_block_sign_hash", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var wbsh Bad
		if err = wbsh.Unmarshal(name, val); err != nil {
			return
		}
		return ex.WrongBlockSignHash(&wbsh)
	})

	register("wrong_block_sign_key", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var wbsk Bad
		if err = wbsk.Unmarshal(name, val); err != nil {
			return
		}
		return ex.WrongBlockSignKey(&wbsk)
	})

	register("wrong_block_hash", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var wbh Bad
		if err = wbh.Unmarshal(name, val); err != nil {
			return
		}
		return ex.WrongBlockHash(&wbh)
	})

	register("verification_ticket_group", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var vtg Bad
		if err = vtg.Unmarshal(name, val); err != nil {
			return
		}
		return ex.VerificationTicketGroup(&vtg)
	})

	register("wrong_verification_ticket_hash", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var wvth Bad
		if err = wvth.Unmarshal(name, val); err != nil {
			return
		}
		return ex.WrongVerificationTicketHash(&wvth)
	})

	register("wrong_verification_ticket_key", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var wvtk Bad
		if err = wvtk.Unmarshal(name, val); err != nil {
			return
		}
		return ex.WrongVerificationTicketKey(&wvtk)
	})

	register("wrong_notarized_block_hash", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var wnth Bad
		if err = wnth.Unmarshal(name, val); err != nil {
			return
		}
		return ex.WrongNotarizedBlockHash(&wnth)
	})

	register("wrong_notarized_block_key", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var wnbk Bad
		if err = wnbk.Unmarshal(name, val); err != nil {
			return
		}
		return ex.WrongNotarizedBlockKey(&wnbk)
	})

	register("notarize_only_competing_block", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var nocb Bad
		if err = nocb.Unmarshal(name, val); err != nil {
			return
		}
		return ex.NotarizeOnlyCompetingBlock(&nocb)
	})

	register("notarized_block", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var nb Bad
		if err = nb.Unmarshal(name, val); err != nil {
			return
		}
		return ex.NotarizedBlock(&nb)
	})

	// Byzantine blockchain sharders

	register("finalized_block", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var fb Bad
		if err = fb.Unmarshal(name, val); err != nil {
			return
		}
		return ex.FinalizedBlock(&fb)
	})

	register("magic_block", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var mb Bad
		if err = mb.Unmarshal(name, val); err != nil {
			return
		}
		return ex.MagicBlock(&mb)
	})

	register("verify_transaction", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var vt Bad
		if err = vt.Unmarshal(name, val); err != nil {
			return
		}
		return ex.VerifyTransaction(&vt)
	})

	// Byzantine view change

	register("mpk", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var mpk Bad
		if err = mpk.Unmarshal(name, val); err != nil {
			return
		}
		return ex.MPK(&mpk)
	})

	register("share", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var shares Bad
		if err = shares.Unmarshal(name, val); err != nil {
			return
		}
		return ex.Shares(&shares)
	})

	register("signature", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var signatures Bad
		if err = signatures.Unmarshal(name, val); err != nil {
			return
		}
		return ex.Signatures(&signatures)
	})

	register("publish", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var publish Bad
		if err = publish.Unmarshal(name, val); err != nil {
			return
		}
		return ex.Publish(&publish)
	})

	// a system command

	register("command", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var cn CommandName
		if err = mapstructure.Decode(val, &cn); err != nil {
			return fmt.Errorf("decoding '%s': %v", name, err)
		}
		ex.Command(cn.Name, tm) // async command
		return nil
	})

	// Blobber related executors

	register("storage_tree", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var st Bad
		if err = mapstructure.Decode(val, &st); err != nil {
			return fmt.Errorf("decoding '%s': %v", name, err)
		}
		return ex.StorageTree(&st)
	})

	register("validator_proof", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var vp Bad
		if err = mapstructure.Decode(val, &vp); err != nil {
			return fmt.Errorf("decoding '%s': %v", name, err)
		}
		return ex.ValidatorProof(&vp)
	})

	register("challenges", func(f Flow, name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var cs Bad
		if err = mapstructure.Decode(val, &cs); err != nil {
			return fmt.Errorf("decoding '%s': %v", name, err)
		}
		return ex.Challenges(&cs)
	})

}
