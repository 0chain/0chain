package tokenpool

import (
	"encoding/json"

	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/util"
)

//go:generate msgp -io=false -tests=false -v

type TokenPoolTransferResponse struct {
	TxnHash    string `json:"txn_hash,omitempty"`
	FromPool   string `json:"from_pool,omitempty"`
	ToPool     string `json:"to_pool,omitempty"`
	Value      int64  `json:"value,omitempty"`
	FromClient string `json:"from_client,omitempty"`
	ToClient   string `json:"to_client,omitempty"`
}

func (p *TokenPoolTransferResponse) Encode() []byte {
	buff, _ := json.Marshal(p)
	return buff
}

func (p *TokenPoolTransferResponse) Decode(input []byte) error {
	err := json.Unmarshal(input, p)
	return err
}

type TokenPoolI interface {
	GetBalance() int64
	SetBalance(value int64)
	GetID() string
	DigPool(id string, txn *transaction.Transaction) (*state.Transfer, string, error)
	FillPool(txn *transaction.Transaction) (*state.Transfer, string, error)
	TransferTo(op TokenPoolI, value int64, entity interface{}) (*state.Transfer, string, error)
	DrainPool(fromClientID, toClientID string, value int64, entity interface{}) (*state.Transfer, string, error)
	EmptyPool(fromClientID, toClientID string, entity interface{}) (*state.Transfer, string, error)
}

type TokenPool struct {
	ID      string `json:"id"`
	Balance int64  `json:"balance"`
}

//go:generate mockery --case underscore --name TokenLockInterface --inpackage --testonly
type TokenLockInterface interface {
	util.MPTSerializableSize
	IsLocked(entity interface{}) bool
	LockStats(entity interface{}) []byte
}
