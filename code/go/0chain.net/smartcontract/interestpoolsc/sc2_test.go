package interestpoolsc

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"0chain.net/chaincore/block"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
	"0chain.net/smartcontract/dbs/event"
	"github.com/stretchr/testify/require"
)

type lockFlags struct {
	tokens   float64
	duration time.Duration
}

type lockResponse struct {
	Txn_hash    string
	To_pool     string
	Value       float64
	From_client string
	To_client   string
}

type unlockResponse struct {
	From_Pool   string
	Value       float64
	From_Client string
	To_Client   string
}

type mockScYml struct {
	minLock       float64
	apr           float64
	minLockPeriod time.Duration
	maxMint       float64
}

const (
	clientId             = "fred"
	startMinted          = 10
	clientStartZCN       = 777
	txHash               = "tx hash"
	errLock              = "failed locking tokens: "
	errInsufficientFunds = "insufficent amount to dig an interest pool"
	errNoTokens          = "you have no tokens to your name"
	errLockGtBalance     = "lock amount is greater than balance"
	errDurationToLong    = "is longer than max lock period"
	errDurationToShort   = "is shorter than min lock period"
	errMaxMint           = "can't mint anymore"
	errUnlock            = "failed to unlock tokens"
	errEmptyingPool      = "error emptying pool"
	errPoolLocked        = "pool is still locked"
	errPoolNotExist      = "doesn't exist"
	startTime            = common.Timestamp(100)
)

var (
	scYml = mockScYml{
		minLock:       10.0,
		apr:           0.1,
		minLockPeriod: 1 * time.Minute,
		maxMint:       4000000.0,
	}
	approvedMinters = []string{
		"6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9", // miner SC
		"cf8d0df9bd8cc637a4ff4e792ffe3686da6220c45f0e1103baa609f3f1751ef4", // interest SC
		"6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d7", // storage SC
	}
	storageScId = approvedMinters[1]
)

func TestLock(t *testing.T) {
	var resp *lockResponse
	var userNode *UserNode
	var globalNode *GlobalNode
	var err error

	t.Run("lock", func(t *testing.T) {
		var flags = lockFlags{
			tokens:   1.0,
			duration: 1 * time.Hour,
		}
		resp, userNode, globalNode, err = testLock(t, flags.tokens, flags.duration, clientStartZCN, startMinted)
		require.NoError(t, err)
		require.EqualValues(t, resp.Txn_hash, txHash)
		require.EqualValues(t, resp.To_pool, txHash)
		require.EqualValues(t, resp.Value, zcnToBalance(flags.tokens))
		require.EqualValues(t, resp.From_client, clientId)
		require.EqualValues(t, resp.To_client, storageScId)
		require.Len(t, userNode.Pools, 1)
		var userPool = userNode.Pools[txHash]
		var f = formulae{
			sc:        scYml,
			lockFlags: flags,
		}
		require.EqualValues(t, userPool.TokensEarned, f.tokensEarned())
		require.EqualValues(t, globalNode.SimpleGlobalNode.TotalMinted, f.tokensEarned()+zcnToBalance(startMinted))
	})

	t.Run(errInsufficientFunds, func(t *testing.T) {
		var flags = lockFlags{
			tokens:   0,
			duration: 1 * time.Hour,
		}
		_, _, globalNode, err = testLock(t, flags.tokens, flags.duration, clientStartZCN, startMinted)
		require.Error(t, err)
		require.EqualValues(t, err.Error(), errLock+errInsufficientFunds)
		require.EqualValues(t, globalNode.SimpleGlobalNode.TotalMinted, zcnToBalance(startMinted))
	})

	t.Run(errNoTokens, func(t *testing.T) {
		var flags = lockFlags{
			tokens:   1.0,
			duration: 1 * time.Hour,
		}
		_, _, _, err = testLock(t, flags.tokens, flags.duration, 0, startMinted)
		require.Error(t, err)
		require.EqualValues(t, err.Error(), errLock+errNoTokens)
	})

	t.Run(errLockGtBalance, func(t *testing.T) {
		var flags = lockFlags{
			tokens:   1.0,
			duration: 1 * time.Hour,
		}
		_, _, _, err = testLock(t, flags.tokens, flags.duration, flags.tokens-0.001, startMinted)
		require.Error(t, err)
		require.EqualValues(t, err.Error(), errLock+errLockGtBalance)
	})

	t.Run(errDurationToLong, func(t *testing.T) {
		var flags = lockFlags{
			tokens:   1.0,
			duration: YEAR + 1*time.Nanosecond,
		}
		resp, userNode, globalNode, err = testLock(t, flags.tokens, flags.duration, clientStartZCN, startMinted)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), errDurationToLong))
	})

	t.Run(errDurationToShort, func(t *testing.T) {
		var flags = lockFlags{
			tokens:   1.0,
			duration: scYml.minLockPeriod - 1*time.Nanosecond,
		}
		resp, userNode, globalNode, err = testLock(t, flags.tokens, flags.duration, clientStartZCN, startMinted)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), errDurationToShort))
	})

	t.Run(errMaxMint, func(t *testing.T) {
		var flags = lockFlags{
			tokens:   1.0,
			duration: 1 * time.Hour,
		}
		resp, userNode, globalNode, err = testLock(t, flags.tokens, flags.duration, clientStartZCN, scYml.maxMint+1)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), errMaxMint))
	})
}

func TestUnlock(t *testing.T) {
	var resp *unlockResponse
	var userNode *UserNode
	var globalNode *GlobalNode
	var transfer *state.Transfer
	var err error

	t.Run("unlock", func(t *testing.T) {
		var flags = lockFlags{
			tokens:   1.0,
			duration: 1 * time.Hour,
		}
		_, userNode, globalNode, err = testLock(t, flags.tokens, flags.duration, clientStartZCN, startMinted)
		require.NoError(t, err)

		var now = common.Timestamp(common.ToTime(startTime).Add(flags.duration + 1).Unix())

		resp, userNode, transfer, err = testUnlock(t, userNode, globalNode, &poolStat{
			ID:     txHash,
			Locked: false,
		}, now)
		require.NoError(t, err)
		require.Len(t, userNode.Pools, 0)
		require.EqualValues(t, storageScId, transfer.ClientID)
		require.EqualValues(t, clientId, transfer.ToClientID)
		require.EqualValues(t, zcnToBalance(flags.tokens), transfer.Amount)
		require.EqualValues(t, resp.From_Pool, txHash)
		require.EqualValues(t, resp.Value, zcnToBalance(flags.tokens))
		require.EqualValues(t, resp.To_Client, clientId)
		require.EqualValues(t, resp.From_Client, storageScId)
	})

	t.Run(errPoolLocked, func(t *testing.T) {
		var flags = lockFlags{
			tokens:   1.0,
			duration: 1 * time.Hour,
		}
		_, userNode, globalNode, err = testLock(t, flags.tokens, flags.duration, clientStartZCN, startMinted)
		require.NoError(t, err)

		var now = common.Timestamp(common.ToTime(startTime).Add(flags.duration - 1).Unix())

		_, _, _, err = testUnlock(t, userNode, globalNode, &poolStat{
			ID:     txHash,
			Locked: false,
		}, now)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), errUnlock))
		require.True(t, strings.Contains(err.Error(), errEmptyingPool))
		require.True(t, strings.Contains(err.Error(), errPoolLocked))
	})

	t.Run(errPoolNotExist, func(t *testing.T) {
		var flags = lockFlags{
			tokens:   1.0,
			duration: 1 * time.Hour,
		}
		_, userNode, globalNode, err = testLock(t, flags.tokens, flags.duration, clientStartZCN, startMinted)
		require.NoError(t, err)

		var now = common.Timestamp(common.ToTime(startTime).Add(flags.duration + 1).Unix())
		delete(userNode.Pools, txHash)

		_, _, _, err = testUnlock(t, userNode, globalNode, &poolStat{
			ID:     txHash,
			Locked: false,
		}, now)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), errUnlock))
		require.True(t, strings.Contains(err.Error(), errPoolNotExist))
	})

}

func testLock(t *testing.T, tokens float64, duration time.Duration, startBalance float64, alredyMinted float64) (
	*lockResponse, *UserNode, *GlobalNode, error) {
	var input = lockInput(t, duration)
	var userNode = newUserNode(clientId)
	var isc = &InterestPoolSmartContract{
		SmartContract: &smartcontractinterface.SmartContract{
			ID: storageScId,
		},
	}
	var txn = &transaction.Transaction{
		HashIDField:  datastore.HashIDField{Hash: txHash},
		ClientID:     clientId,
		ToClientID:   storageScId,
		CreationDate: startTime,
		Value:        int64(zcnToBalance(tokens)),
	}
	var ctx = &mockStateContext{
		ctx: *cstate.NewStateContext(
			nil,
			&util.MerklePatriciaTrie{},
			txn,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
		),
		clientStartBalance: zcnToBalance(startBalance),
		store:              make(map[datastore.Key]util.MPTSerializable),
	}
	var globalNode = &GlobalNode{
		ID: storageScId,
		SimpleGlobalNode: &SimpleGlobalNode{
			MaxMint:     zcnToBalance(scYml.maxMint),
			TotalMinted: zcnToBalance(alredyMinted),
			MinLock:     state.Balance(scYml.minLock),
			APR:         scYml.apr,
		},
		MinLockPeriod: scYml.minLockPeriod,
	}

	output, err := isc.lock(txn, userNode, globalNode, input, ctx)
	if err != nil {
		return nil, nil, globalNode, err
	}

	var response = &lockResponse{}
	require.NoError(t, json.Unmarshal([]byte(output), response))

	newUserNode, err := isc.getUserNode(userNode.ClientID, ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	return response, newUserNode, globalNode, nil
}

func testUnlock(t *testing.T, userNode *UserNode, globalNode *GlobalNode, poolStats *poolStat, now common.Timestamp) (
	*unlockResponse, *UserNode, *state.Transfer, error) {

	input, err := json.Marshal(poolStats)
	require.NoError(t, err)
	var isc = &InterestPoolSmartContract{
		SmartContract: &smartcontractinterface.SmartContract{
			ID: storageScId,
		},
	}
	var txn = &transaction.Transaction{
		HashIDField:  datastore.HashIDField{Hash: txHash},
		ClientID:     clientId,
		ToClientID:   storageScId,
		CreationDate: now,
	}
	var ctx = &mockStateContext{
		ctx: *cstate.NewStateContext(
			nil,
			&util.MerklePatriciaTrie{},
			txn,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
		),
		store: make(map[datastore.Key]util.MPTSerializable),
	}

	output, err := isc.unlock(txn, userNode, globalNode, input, ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	var response = &unlockResponse{}
	require.NoError(t, json.Unmarshal([]byte(output), response))
	newUserNode, err := isc.getUserNode(userNode.ClientID, ctx)
	require.NoError(t, err)

	var transfers = ctx.ctx.GetTransfers()
	require.Len(t, transfers, 1)

	return response, newUserNode, transfers[0], nil
}

const x10 = 10 * 1000 * 1000 * 1000

type mockStateContext struct {
	ctx                cstate.StateContext
	clientStartBalance state.Balance
	store              map[datastore.Key]util.MPTSerializable
}

func (sc *mockStateContext) GetLastestFinalizedMagicBlock() *block.Block               { return nil }
func (sc *mockStateContext) GetLatestFinalizedBlock() *block.Block                     { return nil }
func (sc *mockStateContext) GetBlock() *block.Block                                    { return nil }
func (sc *mockStateContext) SetMagicBlock(_ *block.MagicBlock)                         {}
func (sc *mockStateContext) GetState() util.MerklePatriciaTrieI                        { return nil }
func (sc *mockStateContext) GetTransaction() *transaction.Transaction                  { return nil }
func (sc *mockStateContext) GetTransfers() []*state.Transfer                           { return nil }
func (sc *mockStateContext) GetSignedTransfers() []*state.SignedTransfer               { return nil }
func (sc *mockStateContext) GetMints() []*state.Mint                                   { return nil }
func (sc *mockStateContext) Validate() error                                           { return nil }
func (sc *mockStateContext) GetBlockSharders(_ *block.Block) []string                  { return nil }
func (sc *mockStateContext) GetSignatureScheme() encryption.SignatureScheme            { return nil }
func (sc *mockStateContext) AddSignedTransfer(_ *state.SignedTransfer)                 {}
func (sc *mockStateContext) DeleteTrieNode(_ datastore.Key) (datastore.Key, error)     { return "", nil }
func (sc *mockStateContext) GetChainCurrentMagicBlock() *block.MagicBlock              { return nil }
func (sc *mockStateContext) EmitEvent(event.EventType, event.EventTag, string, string) {}
func (sc *mockStateContext) EmitError(error)                                           {}
func (sc *mockStateContext) GetEvents() []event.Event                                  { return nil }
func (sc *mockStateContext) GetEventDB() *event.EventDb                                { return nil }
func (sc *mockStateContext) GetClientBalance(_ datastore.Key) (state.Balance, error) {
	if sc.clientStartBalance == 0 {
		return 0, util.ErrValueNotPresent
	}
	return sc.clientStartBalance, nil
}
func (sc *mockStateContext) SetStateContext(_ *state.State) error { return nil }

func (sc *mockStateContext) GetTrieNode(key datastore.Key, v util.MPTSerializable) error {
	n, ok := sc.store[key]
	if !ok {
		return util.ErrValueNotPresent
	}

	b, err := n.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}

	_, err = v.UnmarshalMsg(b)
	return err
}

func (sc *mockStateContext) InsertTrieNode(key datastore.Key, node util.MPTSerializable) (datastore.Key, error) {
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

// Calculates important 0chain values defined from config
// logs and cli input parameters.
// sc = sc.yaml
// lockFlags input to ./zwallet lock
//
type formulae struct {
	sc        mockScYml
	lockFlags lockFlags
}

// interest earned from a waller lock cli command
func (f formulae) tokensEarned() state.Balance {
	var amount = float64(zcnToBalance(f.lockFlags.tokens))
	var apr = f.sc.apr
	var duration = float64(f.lockFlags.duration)
	var year = float64(YEAR)

	return state.Balance(amount * apr * duration / year)
}
