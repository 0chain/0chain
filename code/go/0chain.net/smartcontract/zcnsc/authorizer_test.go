package zcnsc

import (
	//cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// TODO: Mock Transaction.TransactionData with SmartContractTransactionData
// TODO: Mock SmartContractTransactionData

const (
	LOCKUPTIME90DAYS = time.Second * 10
	C0               = "client_0"
	C1               = "client_1"
)

func TestAuthorizerNodeShouldBeAbleToAddTransfer(t *testing.T) {
	sc := CreateStateContext()
	an := getNewAuthorizer("public key", "id")
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
	an := getNewAuthorizer("public key", "id")
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
	require.NotEmpty(t, address)
	require.NoError(t, err, "must be able to add authorizer")

	address, err = sc.addAuthorizer(tr, data, balances)
	require.Empty(t, address)
	require.Contains(t, err.Error(), "failed to add authorizer")
	require.Error(t, err, "must be able to add only one authorizer")
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

	ans := getAuthorizerNodes(balances)
	require.Nil(t, ans.NodeMap[tr.ClientID])

	resp, err := sc.addAuthorizer(tr, data, balances)
	require.NotEmpty(t, resp)
	require.NotNil(t, resp)
	require.NoError(t, err)

	ans = getAuthorizerNodes(balances)
	require.NotNil(t, ans.NodeMap[tr.ClientID])

	tr = CreateTransaction("another client", 10)

	authorizer, err := sc.deleteAuthorizer(tr, data, balances)
	require.Empty(t, authorizer)
	require.Error(t, err)
	require.Contains(t, err.Error(), "doesn't exist")
}

func Test_LockingBasicLogicTest (t *testing.T) {
	tr := CreateDefaultTransaction()
	z := &tokenpool.ZcnLockingPool{
		ZcnPool:            tokenpool.ZcnPool{
			TokenPool: tokenpool.TokenPool{
				ID:      "0",
				Balance: 0,
			},
		},
		TokenLockInterface: tokenLock{
			StartTime: 0,
			Duration:  0,
		},
	}

	locked := z.IsLocked(tr)
	require.Equal(t, locked, true)
}

func Test_Can_DigPool(t *testing.T) {
	tr := CreateDefaultTransaction()
	an := getNewAuthorizer("key", tr.ClientID)

	_, _, err := an.Staking.DigPool(tr.Hash, tr)
	require.NoError(t, err)
}

func Test_Can_EmptyPool(t *testing.T) {
	balances := CreateMockStateContext()
	tr := CreateDefaultTransaction()
	gn := getGlobalNode(balances)

	an := getNewAuthorizer("key", tr.ClientID)

	_, _, _ = an.Staking.DigPool(tr.Hash, tr)
	_, _, err := an.Staking.EmptyPool(gn.ID, tr.ClientID, tr)

	require.NoError(t, err)
}

func Test_NewAuthorizer_MustHave_LockPool_Initialized(t *testing.T) {
	trId := "id"
	node := getNewAuthorizer("pk", trId)
	require.NotNil(t, node.Staking.TokenLockInterface)

	balances := CreateMockStateContext()
	ans := getAuthorizerNodes(balances)
	_ = ans.addAuthorizer(node)

	node = ans.NodeMap[trId]
	require.NotNil(t, node.Staking.TokenLockInterface)
}

func Test_AddedAuthorizer_MustHave_LockPool_Initialized(t *testing.T) {
	var data []byte
	tr := CreateDefaultTransaction()
	balances := CreateMockStateContext()
	sc := CreateZCNSmartContract()
	_, _ = sc.addAuthorizer(tr, data, balances)

	ans := getAuthorizerNodes(balances)
	node := ans.NodeMap[tr.ClientID]

	require.NotNil(t, node.Staking.TokenLockInterface)
}

func Test_Can_Delete_Authorizer(t *testing.T) {
	var data []byte
	tr := CreateDefaultTransaction()
	balances := CreateMockStateContext()
	sc := CreateZCNSmartContract()

	resp, err := sc.addAuthorizer(tr, data, balances)
	require.NotNil(t, resp)
	require.NotEmpty(t, resp)
	require.NoError(t, err)

	ans := getAuthorizerNodes(balances)
	require.NotNil(t, ans.NodeMap[tr.ClientID])
	require.NotNil(t, ans.NodeMap[tr.ClientID].Staking)

	tr = CreateDefaultTransaction()

	//_, err = sc.deleteAuthorizer(tr, data, balances)
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

	//ans := getAuthorizerNodes(balances)
	//gn := getGlobalNode(balances)
	//
	//_, err = sc.deleteAuthorizer(tr, data, balances)
	//_, _, err = ans.NodeMap[tr.ClientID].Staking.EmptyPool(gn.ID, tr.ClientID, nil)
	//require.NoError(t, err)
	//
	////require.NotEmpty(t, authorizer)
	//require.NoError(t, err)
}