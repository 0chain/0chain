package chain

import (
	"fmt"

	"0chain.net/chaincore/block"
	bcstate "0chain.net/chaincore/chain/state"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/state"
	"0chain.net/core/logging"
	"0chain.net/core/util"
	"0chain.net/smartcontract/storagesc/blockrewards"
	"go.uber.org/zap"
)

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
	_ = updateRewardTotalList(sctx)
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

func updateRewardTotalList(balances cstate.StateContextI) error {
	qt, err := blockrewards.GetQualifyingTotals(balances)
	if err != nil {
		if err != util.ErrValueNotPresent {
			return err
		}
		qt = new(blockrewards.QualifyingTotals)
	}
	qt.Round = balances.GetBlock().Round
	deltaCapacity, deltaUsage := balances.GetBlockRewardDeltas()
	qt.Capacity += deltaCapacity
	qt.Used += deltaUsage
	if qt.Capacity < 0 || qt.Used < 0 {
		return fmt.Errorf("negative capaciy %d or used %d", qt.Capacity, qt.Used)
	}
	// todo have to handle case where block reward settings are changed
	var qtl blockrewards.QualifyingTotalsSlice
	qtl, err = blockrewards.GetQualifyingTotalsList(balances)
	if err != nil {
		return err
	}
	//qtl[balances.GetBlock().Round] = *qt
	qtl = append(qtl, *qt)
	logging.Logger.Info("piers added qt of UpdateRewardTotalList",
		zap.Int64("round number", balances.GetBlock().Round),
		zap.Any("new qualifying totals", qt),
		zap.Any("new entry totals", qtl[balances.GetBlock().Round]),
	)
	if err := qtl.Save(balances); err != nil {
		return err
	}
	if len(qtl) > 3 {
		logging.Logger.Info("piers end UpdateRewardTotalList",
			zap.Int("length qtl", len(qtl)),
			zap.Any("new list", qtl[len(qtl)-3:]),
		)
	} else {
		logging.Logger.Info("piers end UpdateRewardTotalList",
			zap.Int("length qtl", len(qtl)),
			zap.Any("new list", qtl),
		)
	}
	return nil
}
