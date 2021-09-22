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
		zap.Any("viper block rewards", blockrewards.GetSettingsFromFile()),
	)
	if round < 1 {
		return nil
	}

	logging.Logger.Info("piers start UpdateRewardTotalList",
		zap.Int64("round", round),
	)
	var qtl *blockrewards.QualifyingTotalsList
	qtl, err := blockrewards.GetQualifyingTotalsList(balances)
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

	var lastQt blockrewards.QualifyingTotals
	if len(qtl.Totals) == 0 {
		qtl.Totals = append(qtl.Totals, blockrewards.QualifyingTotals{})
	}
	lastQt = qtl.Totals[len(qtl.Totals)-1]

	var repeat = false
	if lastQt.Round != round-1 {
		if lastQt.Round != round {
			logging.Logger.Info("lastQt.Round != round",
				zap.Int64("lastQt.Round", lastQt.Round))
			return fmt.Errorf("update_block_rewards, currupt chain,"+
				" rounds not sequental %v, and %v", lastQt.Round, round)
		}
		// we are on a repeat for this round
		repeat = true
		if len(qtl.Totals) > 1 {
			lastQt = qtl.Totals[len(qtl.Totals)-2]
		} else {
			lastQt = blockrewards.QualifyingTotals{}
		}
	}
	logging.Logger.Info("piers UpdateRewardTotalList setting round",
		zap.Any("lastQt", lastQt),
		zap.Int64("round", round),
		zap.Bool("repeat", repeat),
	)
	if int64(len(qtl.Totals)) != round && int64(len(qtl.Totals)) != round+1 {
		logging.Logger.Info("int64(len(qtl.Totals)) != round && int64(len(qtl.Totals)) != round+1",
			zap.Int64("lastQt.Round", lastQt.Round))
		return fmt.Errorf("update_block_rewards, currupt chain block reward entries %d "+
			"do not much round number %d", len(qtl.Totals), round)
	}

	var newQt = lastQt
	newQt.Round = round
	settings, changed, err := qtl.HasBlockRewardsSettingsChanged(balances)
	if err != nil {
		return fmt.Errorf("update_block_rewards: %v", err)
	}
	if changed {
		newQt.SettingsChange = settings
		newQt.LastSettingsChange = newQt.Round
	} else {
		newQt.SettingsChange = nil
	}
	deltaCapacity, deltaUsed := balances.GetBlockRewardDeltas()
	newQt.Capacity += deltaCapacity
	newQt.Used += deltaUsed

	if repeat {
		logging.Logger.Info("piers UpdateRewardTotalList about to change",
			zap.Int("old size", len(qtl.Totals)),
			zap.Int64("round", round),
			zap.Any("change", qtl.Totals[round]),
			zap.Any("to qt", newQt),
		)
		qtl.Totals[round] = newQt
	} else {
		logging.Logger.Info("piers UpdateRewardTotalList about to append to list",
			zap.Int("old size", len(qtl.Totals)),
			zap.Int64("round", round),
			zap.Any("new qt", newQt),
		)
		qtl.Totals = append(qtl.Totals, newQt)
	}

	logging.Logger.Info("piers added qt of UpdateRewardTotalList",
		zap.Int64("round number", round),
		zap.Any("new qualifying totals", newQt),
	)

	logStart = len(qtl.Totals) - 5
	if logStart < 0 {
		logStart = 0
	}
	logging.Logger.Info("piers about to save qtl; UpdateRewardTotalList",
		zap.Int64("round", round),
		zap.Int("length qtl", len(qtl.Totals)),
		zap.Any("qtl", qtl.Totals[logStart:]),
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
