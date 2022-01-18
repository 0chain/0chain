package interestpoolsc

import (
	"encoding/json"
	"fmt"
	"time"

	"0chain.net/chaincore/config"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
)

const (
	clientID1 = "client_1"
	clientID2 = "client_2"

	globalNode1Ok = "global_node1"
)

var (
	timeNow = common.Now()
)

// TEST FUNCTION
// testPoolRequest creates a json of encoded new pool request instance
func testPoolRequest(d time.Duration) []byte {
	dur := time.Duration(d)
	durJson, _ := json.Marshal(dur.String())
	durRawMsg := json.RawMessage(durJson)
	jm, _ := json.Marshal(map[string]*json.RawMessage{
		"duration": &durRawMsg,
	})
	return jm
}

// TEST FUNCTION
// testGlobalNode function creates global node instance using incoming parameters
func testGlobalNode(id string, maxMint, totalMint, minLock state.Balance, apr float64, minLockP time.Duration, ownerId datastore.Key) *GlobalNode {
	var gn = &GlobalNode{ID: id}
	gn.SimpleGlobalNode = &SimpleGlobalNode{
		MaxMint:     maxMint,
		TotalMinted: totalMint,
		MinLock:     minLock,
		APR:         apr,
		OwnerId:     ownerId,
	}
	if minLockP != 0 {
		gn.MinLockPeriod = minLockP
	}
	return gn
}

func testGlobalNodeStringTime(id string, maxMint, totalMint, minLock, apr float64, minLockP string, ownerId string) *GlobalNode {
	var gn = &GlobalNode{ID: id}
	gn.SimpleGlobalNode = &SimpleGlobalNode{
		MaxMint:     state.Balance(maxMint * 1e10),
		TotalMinted: state.Balance(totalMint * 1e10),
		MinLock:     state.Balance(minLock * 1e10),
		APR:         apr,
		OwnerId:     ownerId,
	}
	var err error
	gn.MinLockPeriod, err = time.ParseDuration(minLockP)
	if err != nil {
		panic(err)
	}
	return gn
}

// TEST FUNCTION
// testTxn function creates transaction instance using incoming parameters
func testTxn(owner string, value int64) *transaction.Transaction {
	t := &transaction.Transaction{
		ClientID:          datastore.Key(owner),
		ToClientID:        datastore.Key(clientID2),
		ChainID:           config.GetMainChainID(),
		TransactionData:   "testTxnDataOK",
		TransactionOutput: fmt.Sprintf(`{"name":"payFees","input":{"round":%v}}`, 1),
		Value:             value,
		TransactionType:   transaction.TxnTypeSmartContract,
		CreationDate:      common.Now(),
	}
	t.ComputeOutputHash()
	var scheme = encryption.NewBLS0ChainScheme()
	scheme.GenerateKeys()
	t.PublicKey = scheme.GetPublicKey()
	t.Sign(scheme)
	return t
}

// TEST FUNCTION
// testBalance function creates a new instance of testBalances using incoming parameters
func testBalance(client string, value int64) *testBalances {
	t := &testBalances{
		balances: make(map[datastore.Key]state.Balance),
		tree:     make(map[datastore.Key]util.Serializable),
		txn:      testTxn(clientID1, 10),
	}
	if client != "" {
		t.txn = testTxn(client, value)
		t.setBalance(client, state.Balance(value))
	}

	return t
}

// TEST FUNCTION
// testPoolState creates a new instance of poolState
func testPoolState() *poolStat {
	return &poolStat{
		ID:           "new_test_pool_state",
		StartTime:    timeNow,
		Duartion:     time.Duration(20 * time.Second),
		TimeLeft:     0,
		Locked:       true,
		APR:          10,
		TokensEarned: 10,
		Balance:      10000,
	}
}

// TEST FUNCTION
// testInterestPool creates a new instance of interestPool using incoming parameters
func testInterestPool(sec time.Duration, balance int) *interestPool {
	return &interestPool{ZcnLockingPool: &tokenpool.ZcnLockingPool{
		ZcnPool: tokenpool.ZcnPool{
			TokenPool: tokenpool.TokenPool{
				ID:      "new_test_pool_state",
				Balance: state.Balance(balance),
			},
		},
		TokenLockInterface: tokenLock{
			StartTime: timeNow,
			Duration:  time.Duration(sec * time.Second),
			Owner:     clientID1,
		},
	}}
}

// TEST FUNCTION
// testUserNode creates a new instance of UserNode using incoming parameters
func testUserNode(client string, ip *interestPool) *UserNode {
	un := &UserNode{
		ClientID: client,
		Pools:    make(map[datastore.Key]*interestPool),
	}
	if ip != nil {
		un.addPool(ip)
	}
	return un
}

// TEST FUNCTION
// testTokenPoolTransferResponse creates a new instance of TokenPoolTransferResponse
// and returns encoded string of it
func testTokenPoolTransferResponse(txn *transaction.Transaction) string {
	tpr := &tokenpool.TokenPoolTransferResponse{
		TxnHash:    txn.Hash,
		ToPool:     txn.Hash,
		Value:      state.Balance(txn.Value),
		FromClient: txn.ClientID,
		ToClient:   txn.ToClientID,
	}
	return string(tpr.Encode())
}

// TEST FUNCTION
// testConfiguredGlobalNode function returns an instance of GlobalNode based on
//  config.SmartContractConfig configuration structure
func testConfiguredGlobalNode() *GlobalNode {
	var gn = newGlobalNode()
	const pfx = "smart_contracts.interestpoolsc."
	var conf = config.SmartContractConfig
	gn.MinLockPeriod = conf.GetDuration(pfx + "min_lock_period")
	gn.APR = conf.GetFloat64(pfx + "apr")
	gn.MinLock = state.Balance(conf.GetInt64(pfx + "min_lock"))
	gn.MaxMint = state.Balance(conf.GetFloat64(pfx+"max_mint") * 1e10)
	return gn
}
