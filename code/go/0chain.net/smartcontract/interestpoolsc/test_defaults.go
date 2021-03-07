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
)

const (
	client1PubKey = "74f8a3642b07b5a13636909531619246e24bdd2697e9d25e59a4f7e001f65b0ebc09c356728216ef0f2b12d80ed29ab536fe8af4b4a3e22f68a7aff2103ff610"
	client2PubKey = "56cb37686ed110ad2e5e8a3bb2baefb793e553192da0cefb6999e335a71dfc2383f3ceef8640597c948bc3568b0edb1c6c26b2ee2a3c01a806d9bf5cab832d09"
)

const (
	globalNode1Ok = "global_node1"
	globalNode2Ok = "global_node2"
)

const poolStateId = "new_test_pool_state"

const (
	okInterestPoolValue    = 5
	wrongInterestPoolValue = 0

	okInterestPoolBalance    = 100
	wrongInterestPoolBalance = 100
)

var (
	txnOutOk = fmt.Sprintf(`{"name":"payFees","input":{"round":%v}}`, 1)

	testPool = newTestPoolState()

	testInterestPool5            = newTestInterestPool(okInterestPoolValue, wrongInterestPoolBalance)
	testInterestPool0            = newTestInterestPool(wrongInterestPoolValue, wrongInterestPoolBalance)
	testInterestPool0WithBalance = newTestInterestPool(wrongInterestPoolValue, okInterestPoolBalance)

	timeNow = common.Now()
)

const (
	testTxnDataOK = "Txn: Pay 42 from 74f8a3642b07b5a13636909531619246e24bdd2697e9d25e59a4f7e001f65b0ebc09c356728216ef0f2b12d80ed29ab536fe8af4b4a3e22f68a7aff2103ff610\n"
)

func makeTestTx1Ok(value int64) *transaction.Transaction {
	t := &transaction.Transaction{
		ClientID:          datastore.Key(clientID1),
		ToClientID:        datastore.Key(clientID2),
		ChainID:           config.GetMainChainID(),
		TransactionData:   testTxnDataOK,
		TransactionOutput: txnOutOk,
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

func makeTestTxAdress1Ok(value int64) *transaction.Transaction {
	t := &transaction.Transaction{
		ClientID:          datastore.Key(clientID1),
		ToClientID:        datastore.Key(clientID2),
		ChainID:           config.GetMainChainID(),
		TransactionData:   testTxnDataOK,
		TransactionOutput: txnOutOk,
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

func newTestInterestPool(sec time.Duration, balance int) *interestPool {
	return &interestPool{ZcnLockingPool: &tokenpool.ZcnLockingPool{
		ZcnPool: tokenpool.ZcnPool{
			TokenPool: tokenpool.TokenPool{
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

func newEmptyUserNode() *UserNode {
	//intp := newInterestPool()
	return &UserNode{
		ClientID: clientID1,
		Pools:    map[datastore.Key]*interestPool{},
	}
}

func newUserNodeWithPool5() *UserNode {
	un := &UserNode{
		ClientID: clientID1,
		Pools: map[datastore.Key]*interestPool{

		},
	}
	testInterestPool5.ID = poolStateId
	un.addPool(testInterestPool5)
	return un
}

func newUserNodeWithPool0() *UserNode {
	un := &UserNode{
		ClientID: clientID1,
		Pools: map[datastore.Key]*interestPool{

		},
	}
	testInterestPool0.ID = poolStateId
	un.addPool(testInterestPool0)
	return un
}

func newUserNodeWithPool0WithBalance() *UserNode {
	un := &UserNode{
		ClientID: clientID1,
		Pools: map[datastore.Key]*interestPool{

		},
	}
	testInterestPool0WithBalance.ID = poolStateId
	un.addPool(testInterestPool0WithBalance)
	return un
}

func newTestPoolRequest2YearOK() []byte {
	dur := time.Duration(2 * YEAR)
	durJson, _ := json.Marshal(dur.String())
	durRawMsg := json.RawMessage(durJson)
	jm, _ := json.Marshal(map[string]*json.RawMessage{
		"duration": &durRawMsg,
	})
	return jm
}

func newTestPoolRequestOK(d time.Duration) []byte {
	dur := time.Duration(d)
	durJson, _ := json.Marshal(dur.String())
	durRawMsg := json.RawMessage(durJson)
	jm, _ := json.Marshal(map[string]*json.RawMessage{
		"duration": &durRawMsg,
	})
	return jm
}

func newWrongInput() []byte {
	return []byte("{test}")
}

func newTestPoolState() *poolStat {
	return &poolStat{
		ID:           poolStateId,
		StartTime:    timeNow,
		Duartion:     time.Duration(20 * time.Second),
		TimeLeft:     0,
		Locked:       true,
		APR:          10,
		TokensEarned: 10,
		Balance:      10000,
	}
}

func newTestSimpleNode(maxInt, totalMinted, minLock state.Balance) *SimpleGlobalNode {
	return &SimpleGlobalNode{
		MaxMint:     10,
		TotalMinted: 10,
		MinLock:     minLock,
		APR:         10,
	}
}

func newTestGlobalNode(lockPeriod time.Duration, balance int) *GlobalNode {
	return &GlobalNode{
		ID: globalNode1Ok,
		SimpleGlobalNode: &SimpleGlobalNode{
			MaxMint:     10,
			TotalMinted: 10,
			MinLock:     state.Balance(balance),
			APR:         10,
		},
		MinLockPeriod: lockPeriod,
	}
}

func newTestGlobalNodeWithMint(lockPeriod time.Duration, balance int) *GlobalNode {
	return &GlobalNode{
		ID: globalNode1Ok,
		SimpleGlobalNode: &SimpleGlobalNode{
			MaxMint:     100,
			TotalMinted: 1,
			MinLock:     state.Balance(balance),
			APR:         10,
		},
		MinLockPeriod: lockPeriod,
	}
}

func newTestEmptyBalances() *testBalances {
	t := &testBalances{
		balances: make(map[datastore.Key]state.Balance),
		tree:     make(map[datastore.Key]util.Serializable),
	}
	return t
}

func newTestBalanceForClient1Ok(value int) *testBalances {
	t := &testBalances{
		balances: make(map[datastore.Key]state.Balance),
		tree:     make(map[datastore.Key]util.Serializable),
		txn:      makeTestTx1Ok(10),
	}
	t.setBalance(clientID1, state.Balance(value))
	return t
}

func newTokenPoolTransferResponse(txn *transaction.Transaction) string {
	p := newInterestPool()
	tpr := &tokenpool.TokenPoolTransferResponse{
		TxnHash:    txn.Hash,
		FromPool:   p.ID,
		ToPool:     txn.Hash,
		Value:      state.Balance(txn.Value),
		FromClient: txn.ClientID,
		ToClient:   txn.ToClientID,
	}
	return string(tpr.Encode())
}
