package magmasc

import (
	"encoding/json"

	"github.com/0chain/gosdk/zmagmacore/errors"
	zmc "github.com/0chain/gosdk/zmagmacore/magmasc"

	chain "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	tx "0chain.net/chaincore/transaction"
	"0chain.net/core/util"
)

type (
	// tokenPool represents token pool wrapper implementation.
	tokenPool struct {
		zmc.TokenPool
	}
)

var (
	// Make sure tokenPool implements Serializable interface.
	_ util.Serializable = (*tokenPool)(nil)
)

// newTokenPool returns a new constructed token pool.
func newTokenPool() *tokenPool {
	return &tokenPool{}
}

// Decode implements util.Serializable interface.
func (m *tokenPool) Decode(blob []byte) error {
	var pool tokenPool
	if err := json.Unmarshal(blob, &pool); err != nil {
		return errDecodeData.Wrap(err)
	}

	m.ID = pool.ID
	m.Balance = pool.Balance
	m.PayerID = pool.PayerID
	m.PayeeID = pool.PayeeID

	return nil
}

// Encode implements util.Serializable interface.
func (m *tokenPool) Encode() []byte {
	blob, _ := json.Marshal(m)
	return blob
}

// create tries to create a new token poll by given acknowledgment.
func (m *tokenPool) create(txn *tx.Transaction, ackn *zmc.Acknowledgment, sci chain.StateContextI) (*zmc.TokenPoolTransfer, error) {
	terms := ackn.Provider.Terms[ackn.AccessPointID]
	m.Balance = terms.GetAmount()
	if m.Balance < 0 {
		return nil, errors.Wrap(errCodeTokenPoolCreate, errTextUnexpected, errNegativeValue)
	}

	poolBalance := state.Balance(m.Balance)
	clientBalance, err := sci.GetClientBalance(ackn.Consumer.ID)
	if err != nil && !errors.Is(err, util.ErrValueNotPresent) {
		return nil, errors.Wrap(errCodeTokenPoolCreate, "fetch client balance failed", err)
	}
	if clientBalance < poolBalance {
		return nil, errors.Wrap(errCodeTokenPoolCreate, errTextUnexpected, errInsufficientFunds)
	}

	m.ID = ackn.SessionID
	m.PayerID = ackn.Consumer.ID
	m.PayeeID = ackn.Provider.ID

	transfer := state.NewTransfer(m.PayerID, txn.ToClientID, poolBalance)
	if err = sci.AddTransfer(transfer); err != nil {
		return nil, errors.Wrap(errCodeTokenPoolCreate, "transfer token pool failed", err)
	}

	resp := zmc.TokenPoolTransfer{
		TxnHash:    txn.Hash,
		ToPool:     m.ID,
		Value:      m.Balance,
		FromClient: m.PayerID,
		ToClient:   txn.ToClientID, // delegate transfer to smart contract address
	}

	return &resp, nil
}

// spend tries to spend the token pool by given amount.
func (m *tokenPool) spend(txn *tx.Transaction, bill *zmc.Billing, sci chain.StateContextI) (*zmc.TokenPoolTransfer, error) {
	if bill.Amount < 0 {
		return nil, errors.Wrap(errCodeTokenPoolSpend, "billing amount is negative", errNegativeValue)
	}

	payee, amount, poolBalance := m.PayeeID, state.Balance(bill.Amount), state.Balance(m.Balance)
	switch {
	case amount == 0: // refund token pool to payer
		payee = m.PayerID

	case amount < poolBalance: // spend part of token pool to payee
		if err := sci.AddTransfer(state.NewTransfer(txn.ToClientID, payee, amount)); err != nil {
			return nil, errors.Wrap(errCodeTokenPoolSpend, "transfer token pool failed", err)
		}
		poolBalance -= amount
		payee = m.PayerID // refund remaining token pool balance to payer
	}

	// spend token pool by balance
	if err := sci.AddTransfer(state.NewTransfer(txn.ToClientID, payee, poolBalance)); err != nil {
		return nil, errors.Wrap(errCodeTokenPoolSpend, "spend token pool failed", err)
	}

	m.Balance = 0
	resp := zmc.TokenPoolTransfer{
		TxnHash:    txn.Hash,
		FromPool:   m.ID,
		Value:      bill.Amount,
		FromClient: m.PayerID,
		ToClient:   m.PayeeID,
	}

	return &resp, nil
}

// uid returns uniq id used to saving token pool into chain state.
func (m *tokenPool) uid(scID string) string {
	return "sc:" + scID + ":tokenpool:" + m.ID
}
