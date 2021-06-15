package zcnsc

import (
	//cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// TODO: Mock Transaction.TransactionData with SmartContractTransactionData
// TODO: Mock SmartContractTransactionData

const (
	LOCKUPTIME90DAYS = time.Duration(time.Second * 10)
	C0               = "client_0"
	C1               = "client_1"
)

type tokenStat struct {
	Locked bool `json:"is_locked"`
}

func (ts *tokenStat) Encode() []byte {
	buff, _ := json.Marshal(ts)
	return buff
}

func (ts *tokenStat) Decode(input []byte) error {
	err := json.Unmarshal(input, ts)
	return err
}

type tokenLock struct {
	StartTime common.Timestamp `json:"start_time"`
	Duration  time.Duration    `json:"duration"`
}

func (tl tokenLock) IsLocked(entity interface{}) bool {
	txn, ok := entity.(*transaction.Transaction)
	if ok {
		return common.ToTime(txn.CreationDate).Sub(common.ToTime(tl.StartTime)) < tl.Duration
	}
	return true
}

func (tl tokenLock) LockStats(entity interface{}) []byte {
	txn, ok := entity.(*transaction.Transaction)
	if ok {
		ts := &tokenStat{Locked: tl.IsLocked(txn)}
		return ts.Encode()
	}
	return nil
}

func TestTransferToLockPool(t *testing.T) {

	txn := &transaction.Transaction{}
	txn.ClientID = "Client 0"
	txn.Value = 10
	txn.CreationDate = common.Now()

	p0 := &tokenpool.ZcnLockingPool{}
	p0.TokenLockInterface = &tokenLock{Duration: LOCKUPTIME90DAYS, StartTime: common.Now()}
	if _, _, err := p0.DigPool(C0, txn); err != nil {
		t.Error(err)
	}

	p1 := &tokenpool.ZcnPool{}
	txn.Value = 2
	txn.ClientID = "Client 1"
	txn.CreationDate = common.Now()
	if _, _, err := p1.DigPool("Client 1", txn); err != nil {
		t.Error(err)
	}

	_, _, err := p0.TransferTo(p1, 9, txn)
	if err == nil {
		t.Errorf("transfer happened before lock expired\n\tstart time: %v\n\ttxn time: %v\n", p0.IsLocked(txn), txn.CreationDate)
	}

	time.Sleep(LOCKUPTIME90DAYS)
	txn.CreationDate = common.Now()
	_, _, err = p0.TransferTo(p1, 9, txn)
	if err != nil {
		t.Errorf("an error occoured %v\n", err.Error())
	} else if p1.Balance != 11 {
		t.Errorf("pool 1 has wrong balance: %v\ntransaction time: %v\n", p1, common.ToTime(txn.CreationDate))
	}
}

func TestAuthorizerNodeShouldBeAbleToAddTransfer(t *testing.T) {
	sc := CreateStateContext()
	an := getNewAuthorizer("public key")
	tr := CreateDefaultTransaction()

	var transfer *state.Transfer
	transfer, resp, err := an.Staking.DigPool(tr.Hash, tr)

	require.NoError(t, err, "must be able to dig pool")
	require.NotNil(t, transfer)
	require.NotNil(t, resp)
	require.NoError(t, err)

	err = sc.AddTransfer(transfer)
	require.NoError(t, err, "must be able to add transfer")
}

func TestAuthorizerNodeShouldBeAbleToDigPool(t *testing.T) {
	an := getNewAuthorizer("public key")
	tr := CreateDefaultTransaction()

	var transfer *state.Transfer
	transfer, resp, err := an.Staking.DigPool(tr.Hash, tr)

	require.NoError(t, err, "must be able to dig pool")
	require.NotNil(t, transfer)
	require.NotNil(t, resp)
	require.NoError(t, err)
}

func TestShouldAddAuthorizer(t *testing.T) {
	var data []byte
	sc := CreateZCNSmartContract()
	balances := CreateMockStateContext()
	tr := CreateDefaultTransaction()

	address, err := sc.addAuthorizer(tr, data, balances)

	require.NoError(t, err, "must be able to add authorizer")
	require.NotEmpty(t, address)
}

func TestShouldAddOnlyOneAuthorizer(t *testing.T) {
	var data []byte
	sc := CreateZCNSmartContract()
	balances := CreateMockStateContext()
	tr := CreateDefaultTransaction()

	address, err := sc.addAuthorizer(tr, data, balances)
	address, err = sc.addAuthorizer(tr, data, balances)

	require.Contains(t, err.Error(), "failed to add authorizer")
	require.Error(t, err, "must be able to add only one authorizer")
	require.NotEmpty(t, address)
}

func TestShouldDeleteAuthorizer(t *testing.T) {
	var sc = ZCNSmartContract{}
	require.NotNil(t, sc)
}

func TestShouldFailIfAuthorizerExists(t *testing.T) {
	var sc = ZCNSmartContract{}
	require.NotNil(t, sc)
}

func TestShould_Fail_If_TransactionValue_Less_Then_GlobalNode_MinStake(t *testing.T) {
	var data []byte
	sc := CreateZCNSmartContract()
	balances := CreateMockStateContext()
	tr := CreateDefaultTransaction()
	tr.Value = 99

	node := CreateSmartContractGlobalNode()
	node.MinStakeAmount = 100
	err := node.save(balances)
	require.NoError(t, err)

	resp, err := sc.addAuthorizer(tr, data, balances)
	require.Empty(t, resp)
	require.Error(t, err)
	require.Contains(t, err.Error(), "is lower than min amount")
}

func Test_Should_FailWithoutPublicKey(t *testing.T) {
	var data []byte
	tr := CreateDefaultTransaction()
	tr.PublicKey = ""
	balances := CreateMockStateContext()
	sc := CreateZCNSmartContract()

	resp, err := sc.addAuthorizer(tr, data, balances)
	require.Empty(t, resp)
	require.Error(t, err)
	require.Contains(t, err.Error(), "public key was not included with transaction")
}

func Test_Transaction_Or_InputData_MustBe_A_Key_InputData(t *testing.T) {
	pk := &PublicKey{Key: "public Key"}
	data, _ := json.Marshal(pk)
	tr := CreateDefaultTransaction()
	tr.PublicKey = ""
	balances := CreateMockStateContext()
	sc := CreateZCNSmartContract()

	resp, err := sc.addAuthorizer(tr, data, balances)
	require.NotEmpty(t, resp)
	require.NotNil(t, resp)
	require.NoError(t, err)
}

func Test_Transaction_Or_InputData_MustBe_A_Key_Transaction(t *testing.T) {
	var data []byte
	tr := CreateDefaultTransaction()
	tr.PublicKey = "public Key"
	balances := CreateMockStateContext()
	sc := CreateZCNSmartContract()

	resp, err := sc.addAuthorizer(tr, data, balances)
	require.NotEmpty(t, resp)
	require.NotNil(t, resp)
	require.NoError(t, err)
}

func Test_Cannot_Delete_AuthorizerFromAnotherClient(t *testing.T) {
	var data []byte
	tr := CreateDefaultTransaction()
	balances := CreateMockStateContext()
	sc := CreateZCNSmartContract()

	resp, err := sc.addAuthorizer(tr, data, balances)
	require.NotEmpty(t, resp)
	require.NotNil(t, resp)
	require.NoError(t, err)

	tr = CreateTransaction("another client", 10)

	authorizer, err := sc.deleteAuthorizer(tr, data, balances)
	require.Empty(t, authorizer)
	require.Error(t, err)
	require.Contains(t, err.Error(), "doesn't exist")
}

func Test_Can_Delete_Authorizer(t *testing.T) {
	var data []byte
	tr := CreateDefaultTransaction()
	balances := CreateMockStateContext()
	sc := CreateZCNSmartContract()

	resp, err := sc.addAuthorizer(tr, data, balances)
	require.NotEmpty(t, resp)
	require.NotNil(t, resp)
	require.NoError(t, err)

	tr = CreateDefaultTransaction()

	_, err = sc.deleteAuthorizer(tr, data, balances)
	//require.NotEmpty(t, authorizer)
	require.NoError(t, err)
}

func Test_Authorizer_With_EmptyPool_Cannot_Be_Deleted(t *testing.T) {
	var data []byte
	tr := CreateDefaultTransaction()
	balances := CreateMockStateContext()
	sc := CreateZCNSmartContract()

	resp, err := sc.addAuthorizer(tr, data, balances)
	require.NotEmpty(t, resp)
	require.NotNil(t, resp)
	require.NoError(t, err)

	ans := getAuthorizerNodes(balances)
	gn := getGlobalNode(balances)

	_, err = sc.deleteAuthorizer(tr, data, balances)
	_, _, err = ans.NodeMap[tr.ClientID].Staking.EmptyPool(gn.ID, tr.ClientID, nil)
	require.NoError(t, err)

	//require.NotEmpty(t, authorizer)
	require.NoError(t, err)
}