package filler

import (
	"errors"

	zmc "github.com/0chain/gosdk/zmagmacore/magmasc"

	chain "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	tx "0chain.net/chaincore/transaction"
)

// spend tries to spend the token pool by given amount.
func spend(txn *tx.Transaction, bill *zmc.Billing, sci chain.StateContextI, tp zmc.TokenPool) error {
	if bill.Amount < 0 {
		return errors.New("billing amount is negative")
	}

	payee, amount, poolBalance := tp.PayeeID, state.Balance(bill.Amount), state.Balance(tp.Balance)
	switch {
	case amount == 0: // refund token pool to payer
		payee = tp.PayerID

	case amount < poolBalance: // spend part of token pool to payee
		if err := sci.AddTransfer(state.NewTransfer(txn.ToClientID, payee, amount)); err != nil {
			return err
		}
		poolBalance -= amount
		payee = tp.PayerID // refund remaining token pool balance to payer
	}

	// spend token pool by balance
	if err := sci.AddTransfer(state.NewTransfer(txn.ToClientID, payee, poolBalance)); err != nil {
		return err
	}

	tp.Balance = 0
	tp.Transfers = append(tp.Transfers, zmc.TokenPoolTransfer{
		TxnHash:    txn.Hash,
		FromPool:   tp.ID,
		Value:      bill.Amount,
		FromClient: tp.PayerID,
		ToClient:   tp.PayeeID,
	})

	return nil
}
