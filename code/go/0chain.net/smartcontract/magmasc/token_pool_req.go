package magmasc

import (
	"encoding/json"

	"github.com/0chain/gosdk/core/util"
	"github.com/0chain/gosdk/zmagmacore/errors"
	zmc "github.com/0chain/gosdk/zmagmacore/magmasc"
	"github.com/0chain/gosdk/zmagmacore/time"

	tx "0chain.net/chaincore/transaction"
)

type (
	// tokenPoolReq represents lock pool request implementation.
	tokenPoolReq struct {
		ID       string         `json:"id"`
		PayeeID  string         `json:"payee_id"` // empty val means the pool for all
		ExpireAt time.Timestamp `json:"expire_at"` // empty val means the pool has no time limits
		txn      *tx.Transaction
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
	req := tokenPoolReq{txn: m.txn}
	if err := json.Unmarshal(blob, &req); err != nil {
		return zmc.ErrDecodeData.Wrap(err)
	}
	if err := req.Validate(); err != nil {
		return err
	}

	m.ID = req.ID
	m.PayeeID = req.PayeeID
	m.ExpireAt = req.ExpireAt

	return nil
}

// Encode implements util.Serializable interface.
func (m *tokenPoolReq) Encode() []byte {
	blob, _ := json.Marshal(m)
	return blob
}

// PoolBalance implements PoolConfigurator interface.
func (m *tokenPoolReq) PoolBalance() int64 {
	return m.txn.Value
}

// PoolID implements PoolConfigurator interface.
func (m *tokenPoolReq) PoolID() string {
	return m.ID
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
	return m.PayeeID
}

// Validate checks tokenPoolReq for correctness.
func (m *tokenPoolReq) Validate() (err error) {
	switch { // is invalid
	case m.txn == nil:
		err = errors.New(zmc.ErrCodeInternal, "transaction data is required")

	case m.txn.Value <= 0:
		err = errors.New(zmc.ErrCodeInternal, "transaction value is required")

	case m.ID == "":
		err = errors.New(zmc.ErrCodeBadRequest, "pool id is required")
	}

	return err
}
