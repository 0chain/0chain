package storagesc

import (
	"0chain.net/chaincore/block"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
	"time"
)

type mockStateContext struct {
	ctx           cstate.StateContext
	clientBalance state.Balance
	store         map[datastore.Key]util.Serializable
}

type mockBlobberYaml struct {
	serviceCharge           float64
	readPrice               float64
	writePrice              float64
	challengeCompletionTime time.Duration
	MaxOfferDuration        time.Duration
}

var (
	scYaml          = &scConfig{}
	creationDate    = common.Timestamp(100)
	approvedMinters = []string{
		"6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9", // miner SC
		"cf8d0df9bd8cc637a4ff4e792ffe3686da6220c45f0e1103baa609f3f1751ef4", // interest SC
		"6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7", // storage SC
	}
	storageScId = approvedMinters[2]
)

func (sc *mockStateContext) SetMagicBlock(_ *block.MagicBlock)                     { return }
func (sc *mockStateContext) GetState() util.MerklePatriciaTrieI                    { return nil }
func (sc *mockStateContext) GetTransaction() *transaction.Transaction              { return nil }
func (sc *mockStateContext) GetSignedTransfers() []*state.SignedTransfer           { return nil }
func (sc *mockStateContext) Validate() error                                       { return nil }
func (sc *mockStateContext) GetSignatureScheme() encryption.SignatureScheme        { return nil }
func (sc *mockStateContext) AddSignedTransfer(_ *state.SignedTransfer)             { return }
func (sc *mockStateContext) DeleteTrieNode(_ datastore.Key) (datastore.Key, error) { return "", nil }
func (sc *mockStateContext) GetChainCurrentMagicBlock() *block.MagicBlock          { return nil }
func (sc *mockStateContext) GetClientBalance(_ datastore.Key) (state.Balance, error) {
	return sc.clientBalance, nil
}

func (sc *mockStateContext) GetTransfers() []*state.Transfer {
	return sc.ctx.GetTransfers()
}

func (sc *mockStateContext) GetMints() []*state.Mint {
	return sc.ctx.GetMints()
}

func (sc *mockStateContext) GetLastestFinalizedMagicBlock() *block.Block {
	return nil
}

func (sc *mockStateContext) GetBlockSharders(_ *block.Block) []string {
	return nil
}

func (sc *mockStateContext) GetBlock() *block.Block {
	return nil
}

func (sc *mockStateContext) SetStateContext(_ *state.State) error { return nil }

func (sc *mockStateContext) GetTrieNode(key datastore.Key) (util.Serializable, error) {
	var val, ok = sc.store[key]
	if !ok {
		return nil, util.ErrValueNotPresent
	}
	return val, nil
}

func (sc *mockStateContext) InsertTrieNode(key datastore.Key, node util.Serializable) (datastore.Key, error) {
	sc.store[key] = node
	return key, nil
}

func (sc *mockStateContext) AddTransfer(t *state.Transfer) error {
	return sc.ctx.AddTransfer(t)
}

func (sc *mockStateContext) AddMint(m *state.Mint) error {
	return sc.ctx.AddMint(m)
}

func zcnToInt64(token float64) int64 {
	return int64(token * float64(x10))
}

func zcnToBalance(token float64) state.Balance {
	return state.Balance(token * float64(x10))
}
