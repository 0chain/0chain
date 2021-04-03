package interestpoolsc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
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
	errInsufficientFunds = "insufficient amount to dig an interest pool"
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
		require.EqualValues(t, storageScId, transfer.Sender)
		require.EqualValues(t, clientId, transfer.Receiver)
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
		Value:        zcnToBalance(tokens),
	}
	var ctx = &mockStateContext{
		ctx: *cstate.NewStateContext(
			nil,
			&util.MerklePatriciaTrie{},
			&state.Deserializer{},
			txn,
			nil,
			nil,
			nil,
		),
		clientStartBalance: zcnToBalance(startBalance),
		store:              make(map[datastore.Key]util.Serializable),
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

	var newUserNode = isc.getUserNode(userNode.ClientID, ctx)

	return response, newUserNode, globalNode, err
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
			&state.Deserializer{},
			txn,
			nil,
			nil,
			nil,
		),
		store: make(map[datastore.Key]util.Serializable),
	}

	output, err := isc.unlock(txn, userNode, globalNode, input, ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	var response = &unlockResponse{}
	require.NoError(t, json.Unmarshal([]byte(output), response))
	var newUserNode = isc.getUserNode(userNode.ClientID, ctx)

	var transfers = ctx.ctx.GetTransfers()
	require.Len(t, transfers, 1)

	return response, newUserNode, transfers[0], nil
}
