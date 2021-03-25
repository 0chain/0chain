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
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

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
	clientId    = "fred"
	startMinted = 10
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

const clientStartZCN = 777

func TestLock(t *testing.T) {
	var resp lockResponse
	var userNode UserNode
	var err error

	t.Run("lock", func(T *testing.T) {
		var tokens = 1.0
		var duration = 1 * time.Hour
		resp, userNode, err = testLock(t, tokens, duration)
		require.NoError(t, err)

	})

	fmt.Println(resp, userNode, err)

}

func testLock(t *testing.T, tokens float64, duration time.Duration) (lockResponse, UserNode, error) {

	var input = lockInput(t, duration)

	var isc = &InterestPoolSmartContract{
		SmartContract: &smartcontractinterface.SmartContract{
			ID: storageScId,
		},
	}
	var txn = &transaction.Transaction{
		HashIDField:  datastore.HashIDField{Hash: "tx hash"},
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
	var userNode = newUserNode(clientId)
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
