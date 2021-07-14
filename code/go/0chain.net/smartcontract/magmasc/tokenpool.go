package magmasc

import (
	"encoding/json"

	chain "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	tx "0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

type (
	// tokenPool represents token pool wrapper implementation.
	tokenPool struct {
		tokenpool.ZcnPool // embedded token pool

		PayerID datastore.Key `json:"payer_id"`
		PayeeID datastore.Key `json:"payee_id"`
	}
)

var (
	// Make sure tokenPool implements Serializable interface.
	_ util.Serializable = (*tokenPool)(nil)
)

// Decode implements util.Serializable interface.
func (m *tokenPool) Decode(blob []byte) error {
	var pool tokenPool
	if err := json.Unmarshal(blob, &pool); err != nil {
		return errDecodeData.WrapErr(err)
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

// create creates token poll by given acknowledgment.
func (m *tokenPool) create(txn *tx.Transaction, ackn *Acknowledgment, sci chain.StateContextI) (string, error) {
	clientBalance, err := sci.GetClientBalance(ackn.Consumer.ID)
	if err != nil {
		return "", errWrap(errCodeTokenPoolCreate, errTextUnexpected, errInsufficientFunds)
	}

	m.Balance = ackn.Provider.Terms.GetAmount()
	if clientBalance < m.Balance {
		return "", errWrap(errCodeTokenPoolCreate, errTextUnexpected, errInsufficientFunds)
	}

	m.ID = ackn.SessionID
	m.PayerID = ackn.Consumer.ID
	m.PayeeID = ackn.Provider.ID

	transfer := state.NewTransfer(m.PayerID, txn.ToClientID, m.Balance)
	if err = sci.AddTransfer(transfer); err != nil {
		return "", errWrap(errCodeTokenPoolCreate, "transfer token pool failed", err)
	}
	if _, err = sci.InsertTrieNode(m.uid(txn.ToClientID), m); err != nil {
		return "", errWrap(errCodeAcceptTerms, "insert token pool failed", err)
	}

	resp := &tokenpool.TokenPoolTransferResponse{
		TxnHash:    txn.Hash,
		ToPool:     m.ID,
		Value:      m.Balance,
		FromClient: m.PayerID,
		ToClient:   txn.ToClientID, // delegate transfer to smart contract address
	}

	return string(resp.Encode()), nil
}

// spend spends token pool by given amount.
func (m *tokenPool) spend(txn *tx.Transaction, bill *Billing, sci chain.StateContextI) error {
	if bill.Amount < 0 {
		return errWrap(errCodeTokenPoolSpend, "billing amount is negative", errNegativeValue)
	}

	payee, amount := m.PayeeID, state.Balance(bill.Amount)
	switch {
	case amount == 0: // refund token pool to payer
		payee = m.PayerID

	case amount < m.Balance: // spend part of token pool to payee
		if err := sci.AddTransfer(state.NewTransfer(txn.ToClientID, payee, amount)); err != nil {
			return errWrap(errCodeTokenPoolSpend, "transfer token pool failed", err)
		}
		m.Balance -= amount
		payee = m.PayerID // refund remaining token pool balance to payer
	}

	// spend token pool by balance
	if err := sci.AddTransfer(state.NewTransfer(txn.ToClientID, payee, m.Balance)); err != nil {
		return errWrap(errCodeTokenPoolSpend, "spend token pool failed", err)
	}
	m.Balance = 0
	if _, err := sci.DeleteTrieNode(m.uid(txn.ToClientID)); err != nil {
		return errWrap(errCodeTokenPoolSpend, "delete token pool failed", err)
	}

	return nil
}

// uid returns uniq id used to saving token pool into chain state.
func (m *tokenPool) uid(scID datastore.Key) datastore.Key {
	return "sc:" + scID + ":tokenpool:" + m.ID
}
