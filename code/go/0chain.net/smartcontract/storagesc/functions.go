package storagesc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/core/datastore"
	"fmt"
)

type payment struct {
	to     datastore.Key
	amount state.Balance
}

func transferReward(
	sscKey string,
	zcnPool tokenpool.ZcnPool,
	sp *stakePool,
	value state.Balance,
	balances cstate.StateContextI,
) (state.Balance, error) {
	if zcnPool.Balance < value {
		return 0, fmt.Errorf("not enough tokens in pool %s: %d < %d",
			zcnPool.ID, zcnPool.Balance, value)
	}

	payments, moved, err := getPayments(sp, float64(value))
	if err != nil {
		return 0, err
	}
	for _, payment := range payments {
		var transfer *state.Transfer
		transfer, _, err = zcnPool.DrainPool(sscKey, payment.to, payment.amount, nil)
		if err != nil {
			return 0, fmt.Errorf("transferring tokens challenge_pool(%s) -> "+
				"stake_pool_holder(%s): %v", zcnPool.ID, payment.to, err)
		}
		if err = balances.AddTransfer(transfer); err != nil {
			return 0, fmt.Errorf("adding transfer: %v", err)
		}
	}
	return state.Balance(moved), nil
}

func mintReward(
	sp *stakePool,
	value float64,
	balances cstate.StateContextI,
) error {
	payments, _, err := getPayments(sp, value)
	if err != nil {
		return err
	}
	for _, payment := range payments {
		if err := balances.AddMint(&state.Mint{
			Minter:     ADDRESS,        // storage SC
			ToClientID: payment.to,     // delegate wallet
			Amount:     payment.amount, // move total mints at once
		}); err != nil {
			return fmt.Errorf("minting rewards: %v", err)
		}
	}
	return nil
}

// moveToBlobber moves tokens to blobber or validator
func getPayments(sp *stakePool, value float64) ([]payment, float64, error) {
	var payments []payment

	if value == 0 {
		return nil, 0, nil // nothing to move
	}

	var serviceCharge float64
	serviceCharge = sp.Settings.ServiceCharge * value
	if state.Balance(serviceCharge) > 0 {
		sp.Rewards.Charge += state.Balance(serviceCharge)
		payments = append(payments, payment{
			to:     sp.Settings.DelegateWallet,
			amount: state.Balance(serviceCharge),
		})
	}

	if state.Balance(value-serviceCharge) == 0 {
		return nil, 0, nil // nothing to move
	}

	if len(sp.Pools) == 0 {
		return nil, 0, fmt.Errorf("no stake pools to move tokens to")
	}

	valueLeft := float64(value) - serviceCharge
	var stake = float64(sp.stake())

	var moved = 0.0
	for _, dp := range sp.orderedPools() {
		var ratio float64

		if stake == 0.0 {
			ratio = 1.0 / float64(len(sp.Pools))
		} else {
			ratio = float64(dp.Balance) / stake
		}
		var move = valueLeft * ratio
		if move == 0 {
			continue
		}

		payments = append(payments, payment{
			to:     dp.DelegateID,
			amount: state.Balance(move),
		})

		// stat
		dp.Rewards += state.Balance(move)
		moved += move
	}
	return payments, moved, nil
}
