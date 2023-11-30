package cmd

import (
	"fmt"
	"log"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/smartcontract/minersc"
	"github.com/0chain/common/core/currency"
	"github.com/0chain/common/core/util"
)

func mockUpdateState(
	name string,
	txn *transaction.Transaction,
	balances cstate.StateContextI) {
	_ = balances.AddTransfer(state.NewTransfer(
		txn.ClientID, txn.ToClientID, txn.Value),
	)
	_ = balances.AddTransfer(state.NewTransfer(
		txn.ClientID, minersc.ADDRESS, txn.Fee),
	)

	for _, transfer := range balances.GetTransfers() {
		mockTransferAmount(
			name,
			transfer.ClientID,
			transfer.ToClientID,
			transfer.Amount,
			balances,
		)
	}

	for _, signedTransfer := range balances.GetSignedTransfers() {
		mockTransferAmount(
			name,
			signedTransfer.ClientID,
			signedTransfer.ToClientID,
			signedTransfer.Amount,
			balances,
		)
	}
}

func mockMint(
	to string,
	amount currency.Coin,
	balances cstate.StateContextI,
) {
	toState, err := balances.GetClientState(to)
	if err != nil && err != util.ErrValueNotPresent {
		log.Fatal(err)
		return
	}

	newBal, err := currency.AddCoin(toState.Balance, amount)
	if err != nil {
		return
	}
	//fmt.Printf("mint %v to %s, new balance: %v\n", amount, to, newBal)
	toState.Balance = newBal
	if _, err := balances.SetClientState(to, toState); err != nil {
		log.Fatal(err)
		return
	}
}

func mockTransferAmount(
	name, from, to string,
	amount currency.Coin,
	balances cstate.StateContextI,
) {
	fromState, err := balances.GetClientState(from)
	if err != nil && err != util.ErrValueNotPresent {
		log.Fatal(err)
		return
	}

	v, err := currency.MinusCoin(fromState.Balance, amount)
	if err != nil {
		fmt.Printf("transfer: %v: from: %v, balance: %v, to: %v, amount: %v\n",
			name, from, fromState.Balance, to, amount)
		panic(err)
	}

	fromState.Balance = v
	_, err = balances.SetClientState(from, fromState)
	if err != nil {
		log.Fatal(err)
		return
	}

	toState, err := balances.GetClientState(to)
	if err != nil && err != util.ErrValueNotPresent {
		log.Fatal(err)
		return
	}

	newBal, err := currency.AddCoin(toState.Balance, amount)
	if err != nil {
		return
	}
	toState.Balance = newBal
	_, err = balances.SetClientState(to, toState)
	if err != nil {
		log.Fatal(err)
	}
}
