package magmasc

import (
	"encoding/json"

	"github.com/0chain/bandwidth_marketplace/code/core/errors"
	bmp "github.com/0chain/bandwidth_marketplace/code/core/magmasc"

	chain "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	tp "0chain.net/chaincore/tokenpool"
	tx "0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

type (
	// tokenPool represents token pool wrapper implementation.
	tokenPool struct {
		tp.ZcnPool // embedded token pool

		PayerID datastore.Key `json:"payer_id"`
		PayeeID datastore.Key `json:"payee_id"`
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
func (m *tokenPool) create(txn *tx.Transaction, ackn *bmp.Acknowledgment, sci chain.StateContextI) (*tp.TokenPoolTransferResponse, error) {
	m.Balance = state.Balance(ackn.Provider.Terms.GetAmount())
	if m.Balance < 0 {
		return nil, errors.Wrap(errCodeTokenPoolCreate, errTextUnexpected, errNegativeValue)
	}

	clientBalance, err := sci.GetClientBalance(ackn.Consumer.ID)
	if err != nil && !errors.Is(err, util.ErrValueNotPresent) {
		return nil, errors.Wrap(errCodeTokenPoolCreate, "fetch client balance failed", err)
	}
	if clientBalance < m.Balance {
		return nil, errors.Wrap(errCodeTokenPoolCreate, errTextUnexpected, errInsufficientFunds)
	}

	m.ID = ackn.SessionID
	m.PayerID = ackn.Consumer.ID
	m.PayeeID = ackn.Provider.ID

	transfer := state.NewTransfer(m.PayerID, txn.ToClientID, m.Balance)
	if err = sci.AddTransfer(transfer); err != nil {
		return nil, errors.Wrap(errCodeTokenPoolCreate, "transfer token pool failed", err)
	}
	if _, err = sci.InsertTrieNode(m.uid(txn.ToClientID), m); err != nil {
		return nil, errors.Wrap(errCodeAcceptTerms, "insert token pool failed", err)
	}

	resp := tp.TokenPoolTransferResponse{
		TxnHash:    txn.Hash,
		ToPool:     m.ID,
		Value:      m.Balance,
		FromClient: m.PayerID,
		ToClient:   txn.ToClientID, // delegate transfer to smart contract address
	}

	return &resp, nil
}

// spend tries to spend the token pool by given amount.
func (m *tokenPool) spend(txn *tx.Transaction, bill *bmp.Billing, sci chain.StateContextI) (*tp.TokenPoolTransferResponse, error) {
	if bill.Amount < 0 {
		return nil, errors.Wrap(errCodeTokenPoolSpend, "billing amount is negative", errNegativeValue)
	}

	payee, amount := m.PayeeID, state.Balance(bill.Amount)
	switch {
	case amount == 0: // refund token pool to payer
		payee = m.PayerID

	case amount < m.Balance: // spend part of token pool to payee
		if err := sci.AddTransfer(state.NewTransfer(txn.ToClientID, payee, amount)); err != nil {
			return nil, errors.Wrap(errCodeTokenPoolSpend, "transfer token pool failed", err)
		}
		m.Balance -= amount
		payee = m.PayerID // refund remaining token pool balance to payer
	}

	// spend token pool by balance
	if err := sci.AddTransfer(state.NewTransfer(txn.ToClientID, payee, m.Balance)); err != nil {
		return nil, errors.Wrap(errCodeTokenPoolSpend, "spend token pool failed", err)
	}

	m.Balance = 0
	if _, err := sci.InsertTrieNode(m.uid(txn.ToClientID), m); err != nil {
		return nil, errors.Wrap(errCodeTokenPoolSpend, "delete token pool failed", err)
	}

	resp := tp.TokenPoolTransferResponse{
		TxnHash:    txn.Hash,
		FromPool:   m.ID,
		Value:      amount,
		FromClient: m.PayerID,
		ToClient:   m.PayeeID,
	}

	return &resp, nil
}

// uid returns uniq id used to saving token pool into chain state.
func (m *tokenPool) uid(scID datastore.Key) datastore.Key {
	return "sc:" + scID + ":tokenpool:" + m.ID
}
