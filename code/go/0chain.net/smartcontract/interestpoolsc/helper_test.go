package interestpoolsc

import (
	"0chain.net/chaincore/block"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

const x10 = 10 * 1000 * 1000 * 1000

type mockStateContext struct {
	ctx                cstate.StateContext
	clientStartBalance state.Balance
	store              map[datastore.Key]util.Serializable
}

func (sc *mockStateContext) GetLastestFinalizedMagicBlock() *block.Block           { return nil }
func (sc *mockStateContext) GetBlock() *block.Block                                { return nil }
func (sc *mockStateContext) SetMagicBlock(_ *block.MagicBlock)                     { return }
func (sc *mockStateContext) GetState() util.MerklePatriciaTrieI                    { return nil }
func (sc *mockStateContext) GetTransaction() *transaction.Transaction              { return nil }
func (sc *mockStateContext) GetTransfers() []*state.Transfer                       { return nil }
func (sc *mockStateContext) GetSignedTransfers() []*state.SignedTransfer           { return nil }
func (sc *mockStateContext) GetMints() []*state.Mint                               { return nil }
func (sc *mockStateContext) Validate() error                                       { return nil }
func (sc *mockStateContext) GetBlockSharders(_ *block.Block) []string              { return nil }
func (sc *mockStateContext) GetSignatureScheme() encryption.SignatureScheme        { return nil }
func (sc *mockStateContext) AddSignedTransfer(_ *state.SignedTransfer)             { return }
func (sc *mockStateContext) DeleteTrieNode(_ datastore.Key) (datastore.Key, error) { return "", nil }

func (sc *mockStateContext) GetClientBalance(_ datastore.Key) (state.Balance, error) {
	if sc.clientStartBalance == 0 {
		return 0, util.ErrValueNotPresent
	}
	return sc.clientStartBalance, nil
}
func (sc *mockStateContext) SetStateContext(_ *state.State) error { return nil }

func (sc *mockStateContext) GetTrieNode(key datastore.Key) (util.Serializable, error) {
	return sc.store[key], nil
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

func zcnToBalance(token float64) state.Balance {
	return state.Balance(token * float64(x10))
}

//	const txnData = "{\"name\":\"lock\",\"input\":{\"duration\":\"10h0m\"}}"
func lockInput(t *testing.T, duration time.Duration) []byte {
	var txnData = "{\"name\":\"lock\",\"input\":{\"duration\":\""
	txnData += duration.String()
	txnData += "\"}}"

	dataBytes := []byte(txnData)
	var smartContractData smartcontractinterface.SmartContractTransactionData
	var err = json.Unmarshal(dataBytes, &smartContractData)
	require.NoError(t, err)
	return []byte(smartContractData.InputData)

}
