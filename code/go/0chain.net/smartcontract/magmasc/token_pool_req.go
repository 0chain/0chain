package magmasc

import (
	"github.com/0chain/gosdk/core/util"
	"github.com/0chain/gosdk/zmagmacore/errors"
	zmc "github.com/0chain/gosdk/zmagmacore/magmasc"

	tx "0chain.net/chaincore/transaction"
)

type (
	// tokenPoolReq represents lock pool request implementation.
	tokenPoolReq struct {
		*zmc.TokenPoolReq
		txn *tx.Transaction
	}
)

var (
	// Make sure tokenPoolReq implements Serializable interface.
	_ util.Serializable = (*tokenPoolReq)(nil)

	// Make sure tokenPoolReq implements PoolConfigurator interface.
	_ zmc.PoolConfigurator = (*tokenPoolReq)(nil)
)

// Decode implements util.Serializable interface.
func (m *tokenPoolReq) Decode(blob []byte) error {
	req := tokenPoolReq{TokenPoolReq: zmc.NewTokenPoolReq()}
	if err := req.TokenPoolReq.Decode(blob); err != nil {
		return zmc.ErrDecodeData.Wrap(err)
	}

	req.txn = m.txn
	if err := req.Validate(); err != nil {
		return err
	}

	m.TokenPoolReq = req.TokenPoolReq

	return nil
}

// PoolBalance implements PoolConfigurator interface.
func (m *tokenPoolReq) PoolBalance() int64 {
	return m.txn.Value
}

// PoolID implements PoolConfigurator interface.
func (m *tokenPoolReq) PoolID() string {
	return m.Id
}

// PoolHolderID implements PoolConfigurator interface.
func (m *tokenPoolReq) PoolHolderID() string {
	return zmc.Address
}

// PoolPayerID implements PoolConfigurator interface.
func (m *tokenPoolReq) PoolPayerID() string {
	return m.txn.ClientID
}

// PoolPayeeID implements PoolConfigurator interface.
func (m *tokenPoolReq) PoolPayeeID() string {
	return m.PayeeId
}

// Validate checks tokenPoolReq for correctness.
func (m *tokenPoolReq) Validate() (err error) {
	switch { // is invalid
	case m.txn == nil:
		err = errors.New(zmc.ErrCodeInternal, "transaction data is required")
	}

	return err
}
