package config

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/mitchellh/mapstructure"

	"0chain.net/conductor/config/cases"
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
	register("env", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return env(ex, val)
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

	// checks
	register("expect_active_set", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return expectActiveSet(ex, val)
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

	// control nodes behavior / misbehavior

	register("generators_failure", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		return configureGeneratorsFailure(name, ex, val)
	})

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

	register("wrong_block_random_seed", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var wb Bad
		if err = wb.Unmarshal(name, val); err != nil {
			return
		}
		return ex.WrongBlockRandomSeed(&wb)
	})

	register("wrong_block_ddos", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		var wb Bad
		if err = wb.Unmarshal(name, val); err != nil {
			return
		}
		return ex.WrongBlockDDoS(&wb)
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

	register("collect_verification_tickets_when_missing_vrf", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		cfg := NewCollectVerificationTicketsWhenMissedVRF()
		if err = cfg.Decode(val); err != nil {
			return
		}
		return ex.SetServerState(cfg)
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

		var dur time.Duration
		if cn.FailureThreshold == "" {
			dur = 0
		} else {
			dur, err = time.ParseDuration(cn.FailureThreshold)
			if err != nil {
				return fmt.Errorf("decoding '%s': %v", name, err)
			}
		}

		ex.Command(cn.Name, cn.Params, dur, tm) // async command
		return nil
	})

	register("sleep", func(_ string,
		_ Executor, val interface{}, _ time.Duration) (err error) {
		var d time.Duration
		switch v := val.(type) {
		case string:
			d, err = time.ParseDuration(v)
			if err != nil {
				return
			}
		case time.Duration:
			d = v
		case int:
			d = time.Duration(v)
		default:
			return fmt.Errorf("Invalid duration argument: %v", val)
		}
		time.Sleep(d)
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

	register(saveLogsDirectiveName, func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {

		if err := ex.SaveLogs(); err != nil {
			log.Printf("Warning, logs are not saved, err: %v", err)
		}

		return nil
	})

	// checks

	register("configure_not_notarised_block_extension_test_case", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {

		cfg := cases.NewNotNotarisedBlockExtension()
		if err := cfg.Decode(val); err != nil {
			return err
		}

		return ex.ConfigureTestCase(cfg)
	})

	register("configure_send_different_blocks_from_first_generator_test_case", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {

		cfg := cases.NewSendDifferentBlocksFromFirstGenerator(ex.MinersNum())
		if err := cfg.Decode(val); err != nil {
			return err
		}

		return ex.ConfigureTestCase(cfg)
	})

	register("configure_send_different_blocks_from_all_generators_test_case", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {

		cfg := cases.NewSendDifferentBlocksFromAllGenerators(ex.MinersNum())
		if err := cfg.Decode(val); err != nil {
			return err
		}
		return ex.ConfigureTestCase(cfg)
	})

	register("configure_breaking_single_block", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {

		cfg := cases.NewBreakingSingleBlock()
		if err := cfg.Decode(val); err != nil {
			return err
		}
		return ex.ConfigureTestCase(cfg)
	})

	register("configure_send_insufficient_proposals_test_case", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {

		cfg := cases.NewSendInsufficientProposals()
		if err := cfg.Decode(val); err != nil {
			return err
		}
		return ex.ConfigureTestCase(cfg)
	})

	register("configure_verifying_non_existent_block_test_case", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {

		if err := ex.EnableServerStatsCollector(); err != nil {
			return fmt.Errorf("error while enabling server stats collector: %v", err)
		}

		cfg := cases.NewVerifyingNonExistentBlock(ex.GetServerStatsCollector())
		if err := cfg.Decode(val); err != nil {
			return err
		}
		return ex.ConfigureTestCase(cfg)
	})

	register("configure_notarising_non_existent_block_test_case", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {

		if err := ex.EnableServerStatsCollector(); err != nil {
			return fmt.Errorf("error while enabling server stats collector: %v", err)
		}

		cfg := cases.NewNotarisingNonExistentBlock(ex.GetServerStatsCollector())
		if err := cfg.Decode(val); err != nil {
			return err
		}
		return ex.ConfigureTestCase(cfg)
	})

	register("configure_resend_proposed_block_test_case", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {

		cfg := cases.NewResendProposedBlock()
		if err := cfg.Decode(val); err != nil {
			return err
		}
		return ex.ConfigureTestCase(cfg)
	})

	register("configure_resend_notarisation_test_case", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {

		cfg := cases.NewResendNotarisation()
		if err := cfg.Decode(val); err != nil {
			return err
		}
		return ex.ConfigureTestCase(cfg)
	})

	register("configure_bad_timeout_vrfs_test_case", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {

		if err := ex.EnableServerStatsCollector(); err != nil {
			return fmt.Errorf("error while enabling server stats collector: %v", err)
		}

		cfg := cases.NewBadTimeoutVRFS(ex.GetServerStatsCollector(), ex.GetMonitorID())
		if err := cfg.Decode(val); err != nil {
			return err
		}
		return ex.ConfigureTestCase(cfg)
	})

	register("configure_half_nodes_down_test_case", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {

		cfg := cases.NewHalfNodesDown(ex.MinersNum())
		if err := cfg.Decode(val); err != nil {
			return err
		}
		return ex.ConfigureTestCase(cfg)
	})

	register("configure_block_state_change_requestor_test_case", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {

		if err := ex.EnableClientStatsCollector(); err != nil {
			return fmt.Errorf("error while enabling server stats collector: %v", err)
		}

		cfg := cases.NewBlockStateChangeRequestor(ex.GetClientStatsCollector())
		if err := cfg.Decode(val); err != nil {
			return err
		}
		return ex.ConfigureTestCase(cfg)
	})

	register("configure_miner_notarised_block_requestor_test_case", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {

		if err := ex.EnableClientStatsCollector(); err != nil {
			return fmt.Errorf("error while enabling server stats collector: %v", err)
		}

		cfg := cases.NewMinerNotarisedBlockRequestor(ex.GetClientStatsCollector())
		if err := cfg.Decode(val); err != nil {
			return err
		}
		return ex.ConfigureTestCase(cfg)
	})

	register("configure_fb_requestor_test_case", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {

		if err := ex.EnableClientStatsCollector(); err != nil {
			return fmt.Errorf("error while enabling server stats collector: %v", err)
		}

		cfg := cases.NewFBRequestor(ex.GetClientStatsCollector())
		if err := cfg.Decode(val); err != nil {
			return err
		}
		return ex.ConfigureTestCase(cfg)
	})

	register("configure_missing_lfb_tickets_test_case", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {

		cfg := cases.NewMissingLFBTickets(ex.MinersNum())
		if err := cfg.Decode(val); err != nil {
			return err
		}
		return ex.ConfigureTestCase(cfg)
	})

	register("configure_check_challenge_is_valid_test_case", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		cfg := cases.NewCheckChallengeIsValid()
		if err := cfg.Decode(val); err != nil {
			return err
		}
		return ex.ConfigureTestCase(cfg)
	})

	register("lock_notarization_and_send_next_round_vrf", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		cfg := NewLockNotarizationAndSendNextRoundVRF()
		if err := cfg.Decode(val); err != nil {
			return err
		}

		return ex.SetServerState(cfg)
	})

	register("round_has_finalized", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		cfg := cases.NewRoundHasFinalized()
		if err := cfg.Decode(val); err != nil {
			return err
		}

		return ex.ConfigureTestCase(cfg)
	})

	register("round_random_seed", func(name string,
		ex Executor, val interface{}, tm time.Duration) (err error) {
		cfg := cases.NewRoundRandomSeed()
		if err := cfg.Decode(val); err != nil {
			return err
		}

		return ex.ConfigureTestCase(cfg)
	})

	register("make_test_case_check", func(name string, ex Executor, val interface{}, tm time.Duration) (err error) {
		cfg := &TestCaseCheck{}
		if err := cfg.Decode(val); err != nil {
			return err
		}
		return ex.MakeTestCaseCheck(cfg)
	})

	register("blobber_list", func(name string, ex Executor, val interface{}, tm time.Duration) (err error) {
		cfg := NewBlobberList()
		if err := cfg.Decode(val); err != nil {
			return err
		}

		return ex.SetServerState(cfg)
	})

	register("blobber_download", func(name string, ex Executor, val interface{}, tm time.Duration) (err error) {
		cfg := NewBlobberDownload()
		if err := cfg.Decode(val); err != nil {
			return err
		}

		return ex.SetServerState(cfg)
	})

	register("blobber_upload", func(name string, ex Executor, val interface{}, tm time.Duration) (err error) {
		cfg := NewBlobberUpload()
		if err := cfg.Decode(val); err != nil {
			return err
		}

		return ex.SetServerState(cfg)
	})

	register("adversarial_validator", func(name string, ex Executor, val interface{}, tm time.Duration) (err error) {
		cfg := NewAdversarialValidator()
		if err := cfg.Decode(val); err != nil {
			return err
		}

		return ex.SetServerState(cfg)
	})

	register("adversarial_authorizer", func(name string, ex Executor, val interface{}, tm time.Duration) (err error) {
		cfg := NewAdversarialAuthorizer()
		if err := cfg.Decode(val); err != nil {
			return err
		}

		return ex.SetServerState(cfg)
	})

	register("magic_block_config", func(name string, ex Executor, val interface{}, tm time.Duration) (err error) {
		cfg := val.(string)
		return ex.SetMagicBlock(cfg)
	})

	register("blobber_list", func(name string, ex Executor, val interface{}, tm time.Duration) (err error) {
		cfg := NewBlobberList()
		if err := cfg.Decode(val); err != nil {
			return err
		}

		return ex.SetServerState(cfg)
	})

	register("blobber_download", func(name string, ex Executor, val interface{}, tm time.Duration) (err error) {
		cfg := NewBlobberDownload()
		if err := cfg.Decode(val); err != nil {
			return err
		}

		return ex.SetServerState(cfg)
	})

	register("blobber_upload", func(name string, ex Executor, val interface{}, tm time.Duration) (err error) {
		cfg := NewBlobberUpload()
		if err := cfg.Decode(val); err != nil {
			return err
		}

		return ex.SetServerState(cfg)
	})

	register("blobber_delete", func(name string, ex Executor, val interface{}, tm time.Duration) (err error) {
		cfg := NewBlobberDelete()
		if err := cfg.Decode(val); err != nil {
			return err
		}

		return ex.SetServerState(cfg)
	})

	register("check_miner_generates_blocks", func(name string, ex Executor, val interface{}, tm time.Duration) (err error) {
		var cfg WaitMinerGeneratesBlock
		if err := mapstructure.Decode(val, &cfg); err != nil {
			return err
		}
		return ex.WaitMinerGeneratesBlock(cfg, tm)
	})

	register("wait_sharder_lfb", func(name string, ex Executor, val interface{}, tm time.Duration) (err error) {
		var cfg WaitSharderLFB
		if err := mapstructure.Decode(val, &cfg); err != nil {
			return err
		}
		return ex.WaitSharderLFB(cfg, tm)
	})

	register("sleep", func(_ string,
		_ Executor, val interface{}, _ time.Duration) (err error) {
		var d time.Duration
		switch v := val.(type) {
		case string:
			d, err = time.ParseDuration(v)
			if err != nil {
				return
			}
		case time.Duration:
			d = v
		case int:
			d = time.Duration(v)
		default:
			return fmt.Errorf("invalid duration argument: %v", val)
		}
		time.Sleep(d)
		return nil
	})

	register("generate_all_challenges", func(name string, ex Executor, val interface{}, tm time.Duration) (err error) {
		cfg, ok := val.(bool)
		if !ok {
			return fmt.Errorf("invalid value. Required type bool, got %T", val)
		}

		log.Printf("[INF] generate_all_challenges: %v", cfg)

		return ex.SetServerState(GenerateAllChallenges(cfg))
	})

	register("generate_challenge", func(name string, ex Executor, val interface{}, tm time.Duration) (err error) {
		cfg := NewGenerateChallenge()
		if err := cfg.Decode(val); err != nil {
			return err
		}

		err = ex.GenerateChallenge(cfg)
		if err != nil {
			return err
		}

		return ex.SetServerState(cfg)
	})

	register("wait_blobber_commit", func(_ string, ex Executor, _ interface{}, tm time.Duration) (err error) {
		ex.WaitOnBlobberCommit(tm)
		return nil

	})

	// waits for miner to generate challenge-generate transaction
	register("wait_challenge_generation", func(name string, ex Executor, val interface{}, tm time.Duration) (err error) {
		ex.WaitForChallengeGeneration(tm)
		return nil
	})

	// waits for blobber to submit challenge and miner to send status of this challenge
	register("wait_challenge_status", func(_ string, ex Executor, _ interface{}, tm time.Duration) (err error) {
		ex.WaitForChallengeStatus(tm)
		return nil
	})

	register("wait_validator_ticket", func(_ string, ex Executor, val interface{}, tm time.Duration) (err error) {
		cfg := NewWaitValidatorTicket()
		err = mapstructure.Decode(val, cfg)
		if err != nil {
			return
		}

		ex.WaitValidatorTicket(*cfg, tm)
		return nil
	})

	// stop_challenge_generation directs miner to stop/resume generating challenge for any blobber
	register("stop_challenge_generation", func(_ string, ex Executor, val interface{}, _ time.Duration) (err error) {
		stopChalGen, ok := val.(bool)
		if !ok {
			return fmt.Errorf("invalid value. Required type bool, got %T", val)
		}
		cfg := StopChallengeGeneration(stopChalGen)
		return ex.SetServerState(cfg)
	})

	register("stop_wm_commit", func(name string, ex Executor, val interface{}, tm time.Duration) (err error) {
		cfg := StopWMCommit(true)
		return ex.SetServerState(cfg)
	})

	register("resume_wm_commit", func(name string, ex Executor, val interface{}, tm time.Duration) (err error) {
		cfg := StopWMCommit(false)
		return ex.SetServerState(cfg)
	})

	register("fail_rename_commit", func(name string, ex Executor, val interface{}, tm time.Duration) (err error) {
		s, ok := getNodeNames(val)
		if !ok {
			return fmt.Errorf("required type slice but got %T", val)
		}

		nodes := ex.GetNodes()
		var nodeIds []NodeID
		for _, name := range s {
			id, ok := nodes[name]
			if !ok {
				return fmt.Errorf("node id for %s not found", name)
			}
			nodeIds = append(nodeIds, id)
		}

		return ex.SetServerState(BuildFailRenameCommit(nodeIds))
	})

	register("disable_fail_rename_commit", func(name string, ex Executor, val interface{}, tm time.Duration) (err error) {
		s, ok := getNodeNames(val)
		if !ok {
			return fmt.Errorf("required type slice but got %T", val)
		}

		nodes := ex.GetNodes()
		var nodeIds []NodeID
		for _, name := range s {
			id, ok := nodes[name]
			if !ok {
				return fmt.Errorf("node id for %s not found", name)
			}
			nodeIds = append(nodeIds, id)
		}

		return ex.SetServerState(BuildDisableFailRenameCommit(nodeIds))
	})

	register("fail_upload_commit", func(name string, ex Executor, val interface{}, tm time.Duration) (err error) {
		s, ok := getNodeNames(val)
		if !ok {
			return fmt.Errorf("required type slice but got %T", val)
		}

		nodes := ex.GetNodes()
		var nodeIds []NodeID
		for _, name := range s {
			id, ok := nodes[name]
			if !ok {
				return fmt.Errorf("node id for %s not found", name)
			}
			nodeIds = append(nodeIds, id)
		}

		return ex.SetServerState(BuildFailUploadCommit(nodeIds))
	})

	register("disable_fail_upload_commit", func(name string, ex Executor, val interface{}, tm time.Duration) (err error) {
		s, ok := getNodeNames(val)
		if !ok {
			return fmt.Errorf("required type slice but got %T", val)
		}

		nodes := ex.GetNodes()
		var nodeIds []NodeID
		for _, name := range s {
			id, ok := nodes[name]
			if !ok {
				return fmt.Errorf("node id for %s not found", name)
			}
			nodeIds = append(nodeIds, id)
		}

		return ex.SetServerState(BuildDisableFailUploadCommit(nodeIds))
	})

	register("wait_for_file_meta_root", func(name string, ex Executor, val interface{}, tm time.Duration) (err error) {
		ex.WaitForFileMetaRoot()
		cfg := GetFileMetaRoot(true)
		return ex.SetServerState(cfg)
	})

	register("check_file_meta_root", func(name string, ex Executor, val interface{}, tm time.Duration) (err error) {
		var command CheckFileMetaRoot
		err = command.Decode(val)
		if err != nil {
			return fmt.Errorf("error decoding directive data: %v", err)
		}
		return ex.CheckFileMetaRoot(&command)
	})

	register("check_aggregate_value_change", func(name string, ex Executor, val interface{}, tm time.Duration) (err error) {
		var cfg CheckAggregateChange
		err = mapstructure.Decode(val, &cfg)
		if err != nil {
			return
		}

		return ex.CheckAggregateValueChange(&cfg, tm)
	})

	register("check_aggregate_value_comparison", func(name string, ex Executor, val interface{}, tm time.Duration) (err error) {
		var cfg CheckAggregateComparison
		err = mapstructure.Decode(val, &cfg)
		if err != nil {
			return
		}

		return ex.CheckAggregateValueComparison(&cfg, tm)
	})

	register("store_allocations_data", func(name string, ex Executor, val interface{}, tm time.Duration) (err error) {
		return ex.StoreAllocationsData()
	})

	register("check_rollback_tokenomics", func(name string, ex Executor, val interface{}, tm time.Duration) (err error) {
		return ex.CheckRollbackTokenomicsComparison()
	})

	register("set_node_config", func(name string, ex Executor, val interface{}, tm time.Duration) (err error) {
		var cfg NodeCustomConfig
		err = mapstructure.Decode(val, &cfg)
		if err != nil {
			return
		}

		return ex.SetNodeCustomConfig(&cfg)
	})

	register("sync_latest_aggregates", func(name string, ex Executor, val interface{}, tm time.Duration) (err error) {
		var cfg SyncAggregates
		err = mapstructure.Decode(val, &cfg)
		if err != nil {
			return fmt.Errorf("error decoding directive data: %v", err)
		}

		return ex.SyncLatestAggregates(&cfg)
	})

	register("pause", func(name string, ex Executor, val interface{}, tm time.Duration) (err error) {
		// pause execution until the user presses enter
		log.Println("Press enter to continue...")
		_, err = bufio.NewReader(os.Stdin).ReadBytes('\n')
		if err != nil {
			return
		}
		log.Printf("Continuing execution...")
		return nil
	})

	register("set_miss_up_download", func(name string, ex Executor, val interface{}, tm time.Duration) (err error) {
		input, ok := val.(bool)
		if !ok {
			return fmt.Errorf("invalid value. Required type MissUpDownload, got %T", input)
		}
		cfg := MissUpDownload(input)
		return ex.SetMissUpDownload(cfg)
	})

	register("wait_sharders_finalize_near_blocks", func(name string, ex Executor, val interface{}, tm time.Duration) (err error) {
		var command WaitShardersFinalizeNearBlocks
		err = mapstructure.Decode(val, &command)
		if err != nil {
			return fmt.Errorf("error decoding directive data: %v", err)
		}

		ex.WaitShardersFinalizeNearBlocks(command, tm)
		return nil
	})
}
