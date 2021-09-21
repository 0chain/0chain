package chain

import (
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/logging"
	"0chain.net/smartcontract/storagesc/blockrewards"
	"go.uber.org/zap"
)

/*
func (c *Chain) updateBlockRewardTotals(sctx bcstate.StateContextI) error {
	b := sctx.GetBlock()
	clientState := sctx.GetState()
	toClient := sctx.GetTransaction().ClientID
	ts, err := c.getState(clientState, toClient)
	if !isValid(err) {
		if state.DebugTxn() {
			logging.Logger.Error("transfer amount - to_client get", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash), zap.Error(err))
			for _, txn := range b.Txns {
				if txn == nil {
					break
				}
				fmt.Fprintf(block.StateOut, "transfer amount r=%v b=%v t=%+v\n", b.Round, b.Hash, txn)
			}
			fmt.Fprintf(block.StateOut, "transfer amount - error getting state value: %v %v\n", toClient, err)
			block.PrintStates(clientState, b.ClientState)
			logging.Logger.DPanic(fmt.Sprintf("transfer amount - error getting state value: %v %v", toClient, err))
		}
		if state.Debug() {
			logging.Logger.Error("transfer amount - error", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Error(err))
		}
		return err
	}
	sctx.SetStateContext(ts)
	_ = UpdateRewardTotalList(sctx)
	_, err = clientState.Insert(util.Path(toClient), ts)
	if err != nil {
		if state.DebugTxn() {
			if config.DevConfiguration.State {
				logging.Logger.Error("transfer amount - to_client get", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash), zap.Error(err))
				for _, txn := range b.Txns {
					if txn == nil {
						break
					}
					fmt.Fprintf(block.StateOut, "transfer amount r=%v b=%v t=%+v\n", b.Round, b.Hash, txn)
				}
				fmt.Fprintf(block.StateOut, "transfer amount - error getting state value: %v  %v\n", toClient, err)
				block.PrintStates(clientState, b.ClientState)
				logging.Logger.DPanic("transfer amount - error", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Error(err))
			}
		}
		if state.Debug() {
			logging.Logger.Error("transfer amount - error", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Error(err))
		}
		return err
	}
	return nil
}
*/
func UpdateRewardTotalList(balances cstate.StateContextI) error {
	round := balances.GetBlock().Round
	if round < 0 {
		return nil
	}

	logging.Logger.Info("piers start UpdateRewardTotalList",
		zap.Int64("round", round))

	var qtl *blockrewards.QualifyingTotalsList
	qtl, err := blockrewards.GetQualifyingTotalsList(balances)
	if err != nil {
		return fmt.Errorf("getting qualifying totals list: %v", err)
	}

	var nextQt blockrewards.QualifyingTotals
	if len(qtl.Totals) > 0 {
		nextQt = qtl.Totals[len(qtl.Totals)-1]
	} else {
		if round > 0 {
			return fmt.Errorf("update_block_rewards, "+
				"corrupted chain, list empty on round %v", round)
		}
	}

	if nextQt.Round != round-1 {
		if nextQt.Round != round {
			return fmt.Errorf("update_block_rewards, currupt chain,"+
				" rounds not sequental %v, and %v", nextQt.Round, round)
		}
		return nil
	}
	if int64(len(qtl.Totals)) != round {
		return fmt.Errorf("update_block_rewards, currupt chain block reward entries %d "+
			"do not much round number %d", len(qtl.Totals), round)
	}

	nextQt.Round = round
	settings, changed, err := qtl.HasBlockRewardsSettingsChanged(balances)
	if err != nil {
		return fmt.Errorf("update_block_rewards: %v", err)
	}
	if changed {
		nextQt.SettingsChange = settings
		nextQt.LastSettingsChange = nextQt.Round
	} else {
		nextQt.SettingsChange = nil
	}
	deltaCapacity, deltaUsed := balances.GetBlockRewardDeltas()
	nextQt.Capacity += deltaCapacity
	nextQt.Used += deltaUsed

	qtl.Totals = append(qtl.Totals, nextQt)

	logging.Logger.Info("piers added qt of UpdateRewardTotalList",
		zap.Int64("round number", round),
		zap.Any("new qualifying totals", nextQt),
	)

	if err := qtl.Save(balances); err != nil {
		return fmt.Errorf("update_block_rewards, saving qualifying totals list: %v", err)
	}

	if len(qtl.Totals) > 3 {
		logging.Logger.Info("piers end UpdateRewardTotalList",
			zap.Int("length qtl", len(qtl.Totals)),
			zap.Any("new list", qtl.Totals[len(qtl.Totals)-3:]),
		)
	} else {
		logging.Logger.Info("piers end UpdateRewardTotalList",
			zap.Int("length qtl", len(qtl.Totals)),
			zap.Any("new list", qtl),
		)
	}

	return nil
}
