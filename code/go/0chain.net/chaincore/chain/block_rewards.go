package chain

import (
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/logging"
	"0chain.net/smartcontract/storagesc/blockrewards"
	"go.uber.org/zap"
)

func UpdateRewardTotalList(balances cstate.StateContextI) error {
	round := balances.GetBlock().Round
	logging.Logger.Info("start piers  UpdateRewardTotalList",
		zap.Int64("round", round),
	)
	if round < 1 {
		return nil
	}

	logging.Logger.Info("piers start UpdateRewardTotalList",
		zap.Int64("round", round),
	)
	qtl, err := blockrewards.GetQualifyingTotalsList(0, balances)
	if err != nil {
		return fmt.Errorf("getting qualifying totals list: %v", err)
	}
	logStart := len(qtl.Totals) - 5
	if logStart < 0 {
		logStart = 0
	}
	logging.Logger.Info("piers start sjpw qt; UpdateRewardTotalList",
		zap.Int64("round", round),
		zap.Int("length qtl", len(qtl.Totals)),
		zap.Any("qtl", qtl.Totals[logStart:]),
	)

	if len(qtl.Totals) == 0 {
		return fmt.Errorf("update_block_rewards, currupt chain,"+
			" round %d empty qualifying totals list", round)
	}
	var newQt = qtl.Totals[len(qtl.Totals)-1]
	deltaCapacity, deltaUsed := balances.GetBlockRewardDeltas()
	newQt.Round = round
	newQt.Capacity += deltaCapacity
	newQt.Used += deltaUsed

	switch int64(len(qtl.Totals)) {
	case round:
		qtl.Totals = append(qtl.Totals, newQt)
	case round + 1:
		qtl.Totals[len(qtl.Totals)-1] = newQt
	default:
		return fmt.Errorf("update_block_rewards, currupt chain,"+
			"qualifing totals list length %d, wrong length for round %d", len(qtl.Totals), round)
	}

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
