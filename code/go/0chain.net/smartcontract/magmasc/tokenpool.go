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
		return zmc.ErrDecodeData.Wrap(err)
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
		return errors.Wrap(zmc.ErrCodeTokenPoolCreate, zmc.ErrTextUnexpected, zmc.ErrNegativeValue)
	}

	m.PayerID = cfg.PoolPayerID()
	clientBalance, err := sci.GetClientBalance(m.PayerID)
	if err != nil && !errors.Is(err, util.ErrValueNotPresent) {
		return errors.Wrap(zmc.ErrCodeTokenPoolCreate, "fetch client balance failed", err)
	}

	poolBalance := state.Balance(m.Balance)
	if clientBalance < poolBalance {
		return errors.Wrap(zmc.ErrCodeTokenPoolCreate, zmc.ErrTextUnexpected, zmc.ErrInsufficientFunds)
	}

	m.HolderID = cfg.PoolHolderID()
	transfer := state.NewTransfer(m.PayerID, m.HolderID, poolBalance)
	if err = sci.AddTransfer(transfer); err != nil {
		return errors.Wrap(zmc.ErrCodeTokenPoolCreate, "transfer token pool failed", err)
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

// expendWithFees tries to expend the token pool by given amount with service fees options.
// This method returns remaining amount after expended token pool balance.
func (m *tokenPool) expendWithFees(
	txn *tx.Transaction,
	amount state.Balance,
	sci chain.StateContextI,
	feeRate float64,
	feeToID string,
) (state.Balance, error) {
	remains := amount // set remains equal to amount in the beginning of expend tokens
	switch {
	case amount < 0: // negative amount
		remains = 0
		return remains, errors.Wrap(zmc.ErrCodeTokenPoolSpend, "expend amount is negative", zmc.ErrNegativeValue)

	case feeRate <= 0 || feeRate > 1: // negative amount
		return remains, errors.New(zmc.ErrCodeTokenPoolSpend, "rate must be a percentage value")

	case m.Balance == 0: // nothing to expend
		return remains, nil

	case amount > 0: // expend token pool to payee
		poolBalance := state.Balance(m.Balance)
		if amount > poolBalance {
			amount = poolBalance // expend whole token pool balance to payee
		}

		fees := state.Balance(float64(amount) * feeRate)
		if err := sci.AddTransfer(state.NewTransfer(m.HolderID, feeToID, fees)); err != nil {
			return remains, errors.Wrap(zmc.ErrCodeTokenPoolSpend, "transfer fee payment failed", err)
		}
		m.Transfers = append(m.Transfers, zmc.TokenPoolTransfer{
			TxnHash:    txn.Hash,
			FromPool:   m.ID,
			Value:      int64(fees),
			FromClient: m.PayerID,
			ToClient:   feeToID,
		})

		payment := amount - fees
		if err := sci.AddTransfer(state.NewTransfer(txn.ToClientID, m.PayeeID, payment)); err != nil {
			return remains, errors.Wrap(zmc.ErrCodeTokenPoolSpend, "transfer payment failed", err)
		}
		m.Transfers = append(m.Transfers, zmc.TokenPoolTransfer{
			TxnHash:    txn.Hash,
			FromPool:   m.ID,
			Value:      int64(payment),
			FromClient: m.PayerID,
			ToClient:   m.PayeeID,
		})

		remains -= amount
		m.Balance -= int64(amount)
	}

	return remains, nil
}

// spend tries to spend the token pool by given amount.
func (m *tokenPool) spend(txn *tx.Transaction, amount state.Balance, sci chain.StateContextI) error {
	poolBalance := state.Balance(m.Balance)
	switch {
	case amount < 0: // negative amount
		return errors.Wrap(zmc.ErrCodeTokenPoolSpend, "spend amount is negative", zmc.ErrNegativeValue)

	case amount > poolBalance: // wrong amount
		return errors.New(zmc.ErrCodeTokenPoolSpend, "amount greater then pool balance")

	case poolBalance == 0: // nothing to spend
		return nil

	case amount > 0: // spend part of token pool to payee
		if err := sci.AddTransfer(state.NewTransfer(txn.ToClientID, m.PayeeID, amount)); err != nil {
			return errors.Wrap(zmc.ErrCodeTokenPoolSpend, "transfer payment failed", err)
		}
		m.Transfers = append(m.Transfers, zmc.TokenPoolTransfer{
			TxnHash:    txn.Hash,
			FromPool:   m.ID,
			Value:      int64(amount),
			FromClient: m.PayerID,
			ToClient:   m.PayeeID,
		})
		poolBalance -= amount
	}
	if poolBalance > 0 { // refund remaining token pool balance to payer
		if err := sci.AddTransfer(state.NewTransfer(txn.ToClientID, m.PayerID, poolBalance)); err != nil {
			return errors.Wrap(zmc.ErrCodeTokenPoolSpend, "refund remaining tokens failed", err)
		}
		m.Transfers = append(m.Transfers, zmc.TokenPoolTransfer{
			TxnHash:  txn.Hash,
			FromPool: m.ID,
			Value:    int64(poolBalance),
			ToClient: m.PayerID,
		})
	}
	// make the pool balance zeroed
	m.Balance = 0

	return nil
}

// spendWithFees tries to spend the token pool by given amount with service fees options.
func (m *tokenPool) spendWithFees(
	txn *tx.Transaction,
	amount state.Balance,
	sci chain.StateContextI,
	feeRate float64,
	feeToID string,
) error {
	poolBalance := state.Balance(m.Balance)
	switch {
	case amount < 0: // negative amount
		return errors.Wrap(zmc.ErrCodeTokenPoolSpend, "spend amount is negative", zmc.ErrNegativeValue)

	case feeRate <= 0 || feeRate > 1: // negative amount
		return errors.New(zmc.ErrCodeTokenPoolSpend, "rate must be a percentage value")

	case amount > poolBalance: // wrong amount
		return errors.New(zmc.ErrCodeTokenPoolSpend, "amount greater then pool balance")

	case poolBalance == 0: // nothing to spend
		return nil

	case amount > 0: // spend part of token pool to payee
		fees := state.Balance(float64(amount) * feeRate)
		if err := sci.AddTransfer(state.NewTransfer(txn.ToClientID, feeToID, fees)); err != nil {
			return errors.Wrap(zmc.ErrCodeTokenPoolSpend, "transfer fee payment failed", err)
		}
		m.Transfers = append(m.Transfers, zmc.TokenPoolTransfer{
			TxnHash:    txn.Hash,
			FromPool:   m.ID,
			Value:      int64(fees),
			FromClient: m.PayerID,
			ToClient:   feeToID,
		})

		payment := amount - fees
		if err := sci.AddTransfer(state.NewTransfer(txn.ToClientID, m.PayeeID, payment)); err != nil {
			return errors.Wrap(zmc.ErrCodeTokenPoolSpend, "transfer payment failed", err)
		}
		m.Transfers = append(m.Transfers, zmc.TokenPoolTransfer{
			TxnHash:    txn.Hash,
			FromPool:   m.ID,
			Value:      int64(payment),
			FromClient: m.PayerID,
			ToClient:   m.PayeeID,
		})
		poolBalance -= amount
	}
	// spend token pool by balance
	if poolBalance > 0 {
		if err := sci.AddTransfer(state.NewTransfer(txn.ToClientID, m.PayerID, poolBalance)); err != nil {
			return errors.Wrap(zmc.ErrCodeTokenPoolSpend, "refund remaining tokens failed", err)
		}
		m.Transfers = append(m.Transfers, zmc.TokenPoolTransfer{
			TxnHash:  txn.Hash,
			FromPool: m.ID,
			Value:    int64(poolBalance),
			ToClient: m.PayerID,
		})
	}
	// make the pool balance zeroed
	m.Balance = 0

	return nil
}
