package magmasc

import (
	"encoding/json"

	"github.com/0chain/gosdk/zmagmacore/errors"
	zmc "github.com/0chain/gosdk/zmagmacore/magmasc"
	"github.com/0chain/gosdk/zmagmacore/time"

	chain "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	tx "0chain.net/chaincore/transaction"
	"0chain.net/core/util"
)

type (
	// tokenPool represents token pool wrapper implementation.
	tokenPool struct {
		zmc.TokenPool
		ExpireAt time.Timestamp `json:"expire_at,omitempty"`
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
	m.HolderID = pool.HolderID
	m.PayerID = pool.PayerID
	m.PayeeID = pool.PayeeID
	m.ExpireAt = pool.ExpireAt

	return nil
}

// create tries to create a new token poll by given session.
func (m *tokenPool) create(txn *tx.Transaction, cfg zmc.PoolConfigurator, sci chain.StateContextI) error {
	m.Balance = cfg.PoolBalance()
	if m.Balance < 0 {
		return errors.Wrap(errCodeTokenPoolCreate, errTextUnexpected, errNegativeValue)
	}

	m.PayerID = cfg.PoolPayerID()
	clientBalance, err := sci.GetClientBalance(m.PayerID)
	if err != nil && !errors.Is(err, util.ErrValueNotPresent) {
		return errors.Wrap(errCodeTokenPoolCreate, "fetch client balance failed", err)
	}

	poolBalance := state.Balance(m.Balance)
	if clientBalance < poolBalance {
		return errors.Wrap(errCodeTokenPoolCreate, errTextUnexpected, errInsufficientFunds)
	}

	m.HolderID = cfg.PoolHolderID()
	transfer := state.NewTransfer(m.PayerID, m.HolderID, poolBalance)
	if err = sci.AddTransfer(transfer); err != nil {
		return errors.Wrap(errCodeTokenPoolCreate, "transfer token pool failed", err)
	}

	m.ID = cfg.PoolID()
	m.PayeeID = cfg.PoolPayeeID()
	m.Transfers = append(m.Transfers, zmc.TokenPoolTransfer{
		TxnHash:    txn.Hash,
		ToPool:     m.ID,
		Value:      m.Balance,
		FromClient: m.PayerID,
		ToClient:   m.HolderID, // make transfer to the holder address
	})

	return nil
}

// spend tries to spend the token pool by given amount.
func (m *tokenPool) spend(txn *tx.Transaction, amount state.Balance, sci chain.StateContextI) error {
	if amount < 0 {
		return errors.Wrap(errCodeTokenPoolSpend, "spend amount is negative", errNegativeValue)
	}

	payee, poolBalance := m.PayeeID, state.Balance(m.Balance)
	switch {
	case amount > poolBalance: // wrong amount
		return errors.New(errCodeTokenPoolSpend, "amount greater then pool balance")

	case poolBalance == 0: // nothing to spend
		return nil

	case amount == 0: // refund token pool to payer
		payee = m.PayerID

	case amount < poolBalance: // spend part of token pool to payee
		if err := sci.AddTransfer(state.NewTransfer(txn.ToClientID, payee, amount)); err != nil {
			return errors.Wrap(errCodeTokenPoolSpend, "transfer token pool failed", err)
		}
		poolBalance -= amount
		payee = m.PayerID // refund remaining token pool balance to payer
	}

	// spend token pool by balance
	if err := sci.AddTransfer(state.NewTransfer(txn.ToClientID, payee, poolBalance)); err != nil {
		return errors.Wrap(errCodeTokenPoolSpend, "spend token pool failed", err)
	}

	m.Balance = 0
	m.Transfers = append(m.Transfers, zmc.TokenPoolTransfer{
		TxnHash:    txn.Hash,
		FromPool:   m.ID,
		Value:      int64(amount),
		FromClient: m.PayerID,
		ToClient:   m.PayeeID,
	})

	return nil
}

// spend tries to spend the token pool by given amount with serviceChargeConfigurator.
func (m *tokenPool) spendWithServiceCharge(txn *tx.Transaction, amount state.Balance, sci chain.StateContextI, serviceCharge float64, serviceID string) error {
	if amount < 0 {
		return errors.Wrap(errCodeTokenPoolSpend, "spend amount is negative", errNegativeValue)
	}
	if !(serviceCharge >= 0 || serviceCharge < 1) {
		return errors.New(errCodeTokenPoolSpend, "service charge must be in [0;1) interval")
	}

	payee, poolBalance := m.PayeeID, state.Balance(m.Balance)
	switch {
	case amount > poolBalance: // wrong amount
		return errors.New(errCodeTokenPoolSpend, "amount greater then pool balance")

	case poolBalance == 0: // nothing to spend
		return nil

	case amount == 0: // refund token pool to payer
		payee = m.PayerID

	case amount < poolBalance: // spend part of token pool to payee
		// paying charge
		charge := state.Balance(float64(amount) * serviceCharge)
		if err := sci.AddTransfer(state.NewTransfer(txn.ToClientID, serviceID, charge)); err != nil {
			return errors.Wrap(errCodeTokenPoolSpend, "transfer token pool failed", err)
		}
		poolBalance -= charge

		// paying reward
		servicePay := amount - charge
		if err := sci.AddTransfer(state.NewTransfer(txn.ToClientID, payee, servicePay)); err != nil {
			return errors.Wrap(errCodeTokenPoolSpend, "transfer token pool failed", err)
		}
		poolBalance -= servicePay

		payee = m.PayerID // refund remaining token pool balance to payer
	}

	// spend token pool by balance
	if err := sci.AddTransfer(state.NewTransfer(txn.ToClientID, payee, poolBalance)); err != nil {
		return errors.Wrap(errCodeTokenPoolSpend, "spend token pool failed", err)
	}

	m.Balance = 0
	m.Transfers = append(m.Transfers, zmc.TokenPoolTransfer{
		TxnHash:    txn.Hash,
		FromPool:   m.ID,
		Value:      int64(amount),
		FromClient: m.PayerID,
		ToClient:   m.PayeeID,
	})

	return nil
}
