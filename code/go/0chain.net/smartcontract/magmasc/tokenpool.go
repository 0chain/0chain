package magmasc

import (
	"github.com/0chain/gosdk/zmagmacore/errors"
	zmc "github.com/0chain/gosdk/zmagmacore/magmasc"
	"github.com/0chain/gosdk/zmagmacore/magmasc/pb"

	chain "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	tx "0chain.net/chaincore/transaction"
	"0chain.net/core/util"
)

type (
	// tokenPool represents token pool wrapper implementation.
	tokenPool struct {
		*zmc.TokenPool
	}
)

var (
	// Make sure tokenPool implements Serializable interface.
	_ util.Serializable = (*tokenPool)(nil)
)

// newTokenPool returns a new constructed token pool.
func newTokenPool() *tokenPool {
	return &tokenPool{TokenPool: zmc.NewTokenPool()}
}

// Decode implements util.Serializable interface.
func (m *tokenPool) Decode(blob []byte) error {
	pool := zmc.NewTokenPool()
	if err := pool.Decode(blob); err != nil {
		return zmc.ErrDecodeData.Wrap(err)
	}

	m.TokenPool = pool

	return nil
}

// create tries to create a new token poll by given session.
func (m *tokenPool) create(txn *tx.Transaction, cfg zmc.PoolConfigurator, sci chain.StateContextI) error {
	m.Balance = cfg.PoolBalance()
	if m.Balance < 0 {
		return errors.Wrap(zmc.ErrCodeTokenPoolCreate, zmc.ErrTextUnexpected, zmc.ErrNegativeValue)
	}

	m.PayerId = cfg.PoolPayerID()
	clientBalance, err := sci.GetClientBalance(m.PayerId)
	if err != nil && !errors.Is(err, util.ErrValueNotPresent) {
		return errors.Wrap(zmc.ErrCodeTokenPoolCreate, "fetch client balance failed", err)
	}

	poolBalance := state.Balance(m.Balance)
	if clientBalance < poolBalance {
		return errors.Wrap(zmc.ErrCodeTokenPoolCreate, zmc.ErrTextUnexpected, zmc.ErrInsufficientFunds)
	}

	m.HolderId = cfg.PoolHolderID()
	transfer := state.NewTransfer(m.PayerId, m.HolderId, poolBalance)
	if err = sci.AddTransfer(transfer); err != nil {
		return errors.Wrap(zmc.ErrCodeTokenPoolCreate, "transfer token pool failed", err)
	}

	m.Id = cfg.PoolID()
	m.PayeeId = cfg.PoolPayeeID()
	m.Transfers = append(m.Transfers, &pb.TokenPoolTransfer{
		TxnHash:    txn.Hash,
		ToPool:     m.Id,
		Value:      m.Balance,
		FromClient: m.PayerId,
		ToClient:   m.HolderId, // make transfer to the holder address
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
		if err := sci.AddTransfer(state.NewTransfer(m.HolderId, feeToID, fees)); err != nil {
			return remains, errors.Wrap(zmc.ErrCodeTokenPoolSpend, "transfer fee payment failed", err)
		}
		m.Transfers = append(m.Transfers, &pb.TokenPoolTransfer{
			TxnHash:    txn.Hash,
			FromPool:   m.Id,
			Value:      int64(fees),
			FromClient: m.PayerId,
			ToClient:   feeToID,
		})

		payment := amount - fees
		if err := sci.AddTransfer(state.NewTransfer(txn.ToClientID, m.PayeeId, payment)); err != nil {
			return remains, errors.Wrap(zmc.ErrCodeTokenPoolSpend, "transfer payment failed", err)
		}
		m.Transfers = append(m.Transfers, &pb.TokenPoolTransfer{
			TxnHash:    txn.Hash,
			FromPool:   m.Id,
			Value:      int64(payment),
			FromClient: m.PayerId,
			ToClient:   m.PayeeId,
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
		if err := sci.AddTransfer(state.NewTransfer(txn.ToClientID, m.PayeeId, amount)); err != nil {
			return errors.Wrap(zmc.ErrCodeTokenPoolSpend, "transfer payment failed", err)
		}
		m.Transfers = append(m.Transfers, &pb.TokenPoolTransfer{
			TxnHash:    txn.Hash,
			FromPool:   m.Id,
			Value:      int64(amount),
			FromClient: m.PayerId,
			ToClient:   m.PayeeId,
		})
		poolBalance -= amount
	}
	if poolBalance > 0 { // refund remaining token pool balance to payer
		if err := sci.AddTransfer(state.NewTransfer(txn.ToClientID, m.PayerId, poolBalance)); err != nil {
			return errors.Wrap(zmc.ErrCodeTokenPoolSpend, "refund remaining tokens failed", err)
		}
		m.Transfers = append(m.Transfers, &pb.TokenPoolTransfer{
			TxnHash:  txn.Hash,
			FromPool: m.Id,
			Value:    int64(poolBalance),
			ToClient: m.PayerId,
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
		m.Transfers = append(m.Transfers, &pb.TokenPoolTransfer{
			TxnHash:    txn.Hash,
			FromPool:   m.Id,
			Value:      int64(fees),
			FromClient: m.PayerId,
			ToClient:   feeToID,
		})

		payment := amount - fees
		if err := sci.AddTransfer(state.NewTransfer(txn.ToClientID, m.PayeeId, payment)); err != nil {
			return errors.Wrap(zmc.ErrCodeTokenPoolSpend, "transfer payment failed", err)
		}
		m.Transfers = append(m.Transfers, &pb.TokenPoolTransfer{
			TxnHash:    txn.Hash,
			FromPool:   m.Id,
			Value:      int64(payment),
			FromClient: m.PayerId,
			ToClient:   m.PayeeId,
		})
		poolBalance -= amount
	}
	// spend token pool by balance
	if poolBalance > 0 {
		if err := sci.AddTransfer(state.NewTransfer(txn.ToClientID, m.PayerId, poolBalance)); err != nil {
			return errors.Wrap(zmc.ErrCodeTokenPoolSpend, "refund remaining tokens failed", err)
		}
		m.Transfers = append(m.Transfers, &pb.TokenPoolTransfer{
			TxnHash:  txn.Hash,
			FromPool: m.Id,
			Value:    int64(poolBalance),
			ToClient: m.PayerId,
		})
	}
	// make the pool balance zeroed
	m.Balance = 0

	return nil
}
