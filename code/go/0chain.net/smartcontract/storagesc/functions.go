package storagesc

import (
	"fmt"

	cstate "github.com/0chain/0chain/code/go/0chain.net/chaincore/chain/state"
	"github.com/0chain/0chain/code/go/0chain.net/chaincore/state"
	"github.com/0chain/0chain/code/go/0chain.net/chaincore/tokenpool"
)

func moveServiceCharge(sscKey string, zcnPool tokenpool.ZcnPool, sp *stakePool,
	value state.Balance, balances cstate.StateContextI) (err error) {

	if value == 0 {
		return // avoid insufficient transfer
	}

	var (
		dw       = sp.Settings.DelegateWallet
		transfer *state.Transfer
	)
	transfer, _, err = zcnPool.DrainPool(sscKey, dw, value, nil)
	if err != nil {
		return fmt.Errorf("transferring tokens challenge_pool() -> "+
			"blobber_charge(%s): %v", dw, err)
	}
	if err = balances.AddTransfer(transfer); err != nil {
		return fmt.Errorf("adding transfer: %v", err)
	}

	// blobber service charge
	sp.Rewards.Charge += value
	return
}

// moveToBlobber moves tokens to blobber or validator
func moveReward(sscKey string, zcnPool tokenpool.ZcnPool, sp *stakePool,
	value state.Balance, balances cstate.StateContextI) (moved state.Balance, err error) {

	if value == 0 {
		return // nothing to move
	}

	if zcnPool.Balance < value {
		return 0, fmt.Errorf("not enough tokens in pool %s: %d < %d",
			zcnPool.ID, zcnPool.Balance, value)
	}

	var serviceCharge state.Balance
	serviceCharge = state.Balance(sp.Settings.ServiceCharge * float64(value))

	err = moveServiceCharge(sscKey, zcnPool, sp, serviceCharge, balances)
	if err != nil {
		return
	}

	value = value - serviceCharge

	if value == 0 {
		return // nothing to move
	}

	if len(sp.Pools) == 0 {
		return 0, fmt.Errorf("no stake pools to move tokens to %s", zcnPool.ID)
	}

	var stake = float64(sp.stake())
	for _, dp := range sp.orderedPools() {
		var ratio float64

		if stake == 0.0 {
			ratio = 1.0 / float64(len(sp.Pools))
		} else {
			ratio = float64(dp.Balance) / stake
		}
		var move = state.Balance(float64(value) * ratio)
		if move == 0 {
			continue
		}
		var transfer *state.Transfer
		transfer, _, err = zcnPool.DrainPool(sscKey, dp.DelegateID, move, nil)
		if err != nil {
			return 0, fmt.Errorf("transferring tokens challenge_pool(%s) -> "+
				"stake_pool_holder(%s): %v", zcnPool.ID, dp.DelegateID, err)
		}
		if err = balances.AddTransfer(transfer); err != nil {
			return 0, fmt.Errorf("adding transfer: %v", err)
		}
		// stat
		dp.Rewards += move // add to stake_pool_holder rewards
		moved += move
	}

	return
}
