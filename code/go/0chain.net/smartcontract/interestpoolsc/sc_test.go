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

type mockScYml struct {
	minLock       float64
	apr           float64
	minLockPeriod time.Duration
	maxMint       float64
}

const (
	clientId       = "fred"
	startMinted    = 10
	clientStartZCN = 777
	txHash         = "tx hash"
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
	var resp lockResponse
	var userNode UserNode
	var err error

	t.Run("lock", func(t *testing.T) {
		var flags = lockFlags{
			tokens:   1.0,
			duration: 1 * time.Hour,
		}
		resp, userNode, err = testLock(t, flags.tokens, flags.duration)
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
	})

}

func testLock(t *testing.T, tokens float64, duration time.Duration) (lockResponse, UserNode, error) {
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
		CreationDate: common.Timestamp(100),
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
	}
	var globalNode = &GlobalNode{
		SimpleGlobalNode: &SimpleGlobalNode{
			MaxMint:     zcnToBalance(scYml.maxMint),
			TotalMinted: startMinted,
			MinLock:     state.Balance(scYml.minLock),
			APR:         scYml.apr,
		},
		MinLockPeriod: scYml.minLockPeriod,
	}

	output, err := isc.lock(txn, userNode, globalNode, input, ctx)
	var response = lockResponse{}
	require.NoError(t, json.Unmarshal([]byte(output), &response))

	return response, *userNode, err
}

func TestUnlock(t *testing.T) {

}

func TestLockConfig(t *testing.T) {

}
