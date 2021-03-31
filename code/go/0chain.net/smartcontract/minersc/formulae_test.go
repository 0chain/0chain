package minersc

import (
	"github.com/stretchr/testify/require"
	"testing"
)

type EarningsType int

const (
	EtFees EarningsType = iota
	EtBlockReward
	EtBoth
)

// Calculates important 0chain values defined from config
// logs and cli input parameters.
// sc = sc.yaml
// lockFlags input to ./zwallet lock
//
type formulae struct {
	zChain           mock0ChainYaml
	sc               mockScYaml
	runtime          runtimeValues
	minerDelegates   []float64
	sharderDelegates [][]float64
}

func (f formulae) tokensEarned(et EarningsType) int64 {
	var totalFees int64 = 0
	for _, fee := range f.runtime.fees {
		totalFees += int64(fee)
	}
	var blockReward = f.sc.blockReward * f.sc.rewardRate
	switch et {
	case EtFees:
		return totalFees
	case EtBlockReward:
		return int64(zcnToBalance(blockReward))
	case EtBoth:
		return totalFees + int64(zcnToBalance(blockReward))
	default:
		panic("Invalid earnings type")
	}
}

func (f formulae) minerRevenue(et EarningsType) int64 {
	var totalEarned = float64(f.tokensEarned(et))

	return int64(totalEarned * f.sc.shareRatio)
}

func (f formulae) sharderRevenue(t *testing.T, et EarningsType) int64 {
	var totalEarned = float64(f.tokensEarned(et))
	var ratio = 1 - f.sc.shareRatio
	require.True(t, len(f.sharderDelegates) > 0)
	var numberOfSharders = len(f.sharderDelegates)

	return int64(totalEarned * ratio / float64(numberOfSharders))
}

// miner gets any extra reward from rounding errors after paying delegates
func (f formulae) minerReward(et EarningsType) int64 {
	var minerRevenue = float64(f.minerRevenue(et))
	var areDelegates = len(f.minerDelegates) > 0
	var serviceCharge = f.zChain.ServiceCharge

	if areDelegates {
		return int64(minerRevenue * serviceCharge)
	} else {
		return int64(minerRevenue)
	}
}

// sharders get any extra reward from rounding errors after paying delegates
func (f formulae) sharderReward(t *testing.T, et EarningsType, sharderId int) int64 {
	var sharderRevenue = float64(f.sharderRevenue(t, et))
	var areDelegates = len(f.sharderDelegates[sharderId]) > 0
	var serviceCharge = f.zChain.ServiceCharge

	if areDelegates {
		return int64(sharderRevenue * serviceCharge)
	} else {
		return int64(sharderRevenue)
	}
}

func (f formulae) minerDelegateReward(t *testing.T, et EarningsType, delegateId int) int64 {
	require.True(t, len(f.minerDelegates) > 0)
	var total = 0.0
	for i := 0; i < len(f.minerDelegates); i++ {
		total += float64(zcnToBalance(float64(f.minerDelegates[i])))
	}
	require.True(t, total > 0.0)
	var ratio = float64(zcnToBalance(f.minerDelegates[delegateId])) / total
	var minerRevenue = float64(f.minerRevenue(et))
	var minerReward = float64(f.minerReward(et))

	return int64((minerRevenue - minerReward) * ratio)
}

func (f formulae) sharderDelegateReward(t *testing.T, et EarningsType, delegateId, sharderId int) int64 {
	require.True(t, len(f.sharderDelegates) > sharderId)
	require.True(t, len(f.sharderDelegates[sharderId]) >= delegateId)
	var total = 0.0
	for i := 0; i < len(f.sharderDelegates[sharderId]); i++ {
		total += float64(zcnToBalance(f.sharderDelegates[sharderId][i]))
	}
	require.True(t, total > 0.0)
	var ratio = float64(zcnToBalance(f.sharderDelegates[sharderId][delegateId])) / total
	var sharderRevenue = float64(f.sharderRevenue(t, et))
	var sharderReward = float64(f.sharderReward(t, et, sharderId))

	return int64((sharderRevenue - sharderReward) * ratio)
}

func (f formulae) minerDelegateInterest(delegateId int) int64 {
	var investment = float64(zcnToBalance(float64(f.minerDelegates[delegateId])))
	var interestRate = f.sc.interestRate

	return int64(investment * interestRate)
}

func (f formulae) sharderDelegateInterest(delegateId, sharderId int) int64 {
	var investment = float64(zcnToBalance(float64(f.sharderDelegates[sharderId][delegateId])))
	var interestRate = f.sc.interestRate

	return int64(investment * interestRate)
}

func (f formulae) totalInterest() int64 {
	var totalInterest = 0.0
	for md := range f.minerDelegates {
		totalInterest += float64(f.minerDelegateInterest(md))
	}
	for s := range f.sharderDelegates {
		for d := range f.sharderDelegates[s] {
			totalInterest += float64(f.sharderDelegateInterest(d, s))
		}
	}

	return int64(totalInterest)
}
