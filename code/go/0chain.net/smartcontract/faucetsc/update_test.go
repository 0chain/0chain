package faucetsc

import (
	"testing"
	"time"

	configpkg "0chain.net/chaincore/config"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"

	"github.com/stretchr/testify/require"
)

func getGlobalNodeTest() (gn *GlobalNode) {
	const pfx = "smart_contracts.faucetsc."

	configpkg.SmartContractConfig.Set(pfx+"pour_amount", 100)
	configpkg.SmartContractConfig.Set(pfx+"max_pour_amount", 1000)
	configpkg.SmartContractConfig.Set(pfx+"periodic_limit", 5000)
	configpkg.SmartContractConfig.Set(pfx+"global_limit", 1000000)
	configpkg.SmartContractConfig.Set(pfx+"individual_reset", 24*time.Hour)
	configpkg.SmartContractConfig.Set(pfx+"global_reset", 240*time.Hour)

	return &GlobalNode{
		ADDRESS,
		100, 1000, 5000, 1000000,
		24 * time.Hour, 240 * time.Hour,
		0, common.ToTime(common.Timestamp(0)),
	}
}

func Test_getConfig(t *testing.T) {
	var (
		fsc        = newTestFaucetSC()
		balances   = newTestBalances()
		tp         = common.Timestamp(0)
		ownerTxn   = newTransaction(owner, fsc.ID, 0, tp)
		configured = getGlobalNodeTest()
		gn         = fsc.getGlobalVariables(ownerTxn, balances)
	)
	require.Equal(t, configured, gn)
}

func TestFaucetSmartContractUpdate(t *testing.T) {
	var (
		fsc         = newTestFaucetSC()
		balances    = newTestBalances()
		tp          = common.Timestamp(0)
		ownerTxn    = newTransaction(owner, fsc.ID, 0, tp)
		nonOwnerTxn = newTransaction(randString(32), fsc.ID, 0, tp)
		originalGn  = getGlobalNodeTest()
		gn          = fsc.getGlobalVariables(ownerTxn, balances)
		err         error
	)

	//test cases that produce errors
	errorTestCases := []struct {
		title string
		txn   *transaction.Transaction
		bytes []byte
		err   string
	}{
		{"malformed request", ownerTxn, []byte("} malformed {"), "bad_request: limit request not formated correctly"},
		{"non owner account", nonOwnerTxn, []byte("} malformed {"), "unauthorized_access: only the owner can update the limits"},
	}
	for _, tc := range errorTestCases {
		t.Run(tc.title, func(t *testing.T) {
			balances.txn = tc.txn
			_, err = fsc.updateLimits(tc.txn, tc.bytes, balances, gn)
			require.Error(t, err)
			require.EqualError(t, err, tc.err)
		})
	}

	//test cases that fail to update
	deniedTestCases := []struct {
		title       string
		request     *limitRequest
		requireFunc func(gn, originalGn *GlobalNode)
	}{
		{
			"all variables denied",
			&limitRequest{
				PourAmount:      -1,
				MaxPourAmount:   -1,
				PeriodicLimit:   -1,
				GlobalLimit:     -1,
				IndividualReset: 1000000000,
				GlobalReset:     1000000000,
			},
			func(gn, originalGn *GlobalNode) {
				require.Equal(t, gn.PourAmount, originalGn.PourAmount)
				require.Equal(t, gn.MaxPourAmount, originalGn.MaxPourAmount)
				require.Equal(t, gn.PeriodicLimit, originalGn.PeriodicLimit)
				require.Equal(t, gn.GlobalLimit, originalGn.GlobalLimit)
				require.Equal(t, gn.IndividualReset, originalGn.IndividualReset)
				require.Equal(t, gn.GlobalReset, originalGn.GlobalReset)
			},
		},
		{
			"max pour amount fail",
			&limitRequest{
				PourAmount:    5,
				MaxPourAmount: 4,
			},
			func(gn, originalGn *GlobalNode) {
				require.Equal(t, gn.MaxPourAmount, originalGn.MaxPourAmount)
			},
		},
		{
			"periodic limit fail",
			&limitRequest{
				PeriodicLimit: 5,
			},
			func(gn, originalGn *GlobalNode) {
				require.Equal(t, gn.PeriodicLimit, originalGn.PeriodicLimit)
			},
		},
		{
			"global limit fail",
			&limitRequest{
				GlobalLimit: 7,
			},
			func(gn, originalGn *GlobalNode) {
				require.Equal(t, gn.GlobalLimit, originalGn.GlobalLimit)
			},
		},
		{
			"individual reset fail",
			&limitRequest{
				IndividualReset: 0,
			},
			func(gn, originalGn *GlobalNode) {
				require.Equal(t, gn.IndividualReset, originalGn.IndividualReset)
			},
		},
		{
			"global reset fail",
			&limitRequest{
				GlobalReset: 0,
			},
			func(gn, originalGn *GlobalNode) {
				require.Equal(t, gn.GlobalReset, originalGn.GlobalReset)
			},
		},
	}
	for _, tc := range deniedTestCases {
		t.Run(tc.title, func(t *testing.T) {
			_, err = fsc.updateLimits(ownerTxn, mustEncode(t, tc.request), balances, gn)
			require.NoError(t, err)
			gn = fsc.getGlobalVariables(ownerTxn, balances)
			tc.requireFunc(gn, originalGn)
		})
	}

	updateTestCases := []struct {
		title       string
		request     *limitRequest
		requireFunc func(gn *GlobalNode, request *limitRequest)
	}{
		{
			"pour amount update",
			&limitRequest{
				PourAmount: 1,
			},
			func(gn *GlobalNode, request *limitRequest) {
				require.Equal(t, gn.PourAmount, request.PourAmount)
			},
		},
		{
			"max pour amount update",
			&limitRequest{
				MaxPourAmount: 6,
			},
			func(gn *GlobalNode, request *limitRequest) {
				require.Equal(t, gn.MaxPourAmount, request.MaxPourAmount)
			},
		},
		{
			"period limit update",
			&limitRequest{
				PeriodicLimit: 7,
			},
			func(gn *GlobalNode, request *limitRequest) {
				require.Equal(t, gn.PeriodicLimit, request.PeriodicLimit)
			},
		},
		{
			"global limit update",
			&limitRequest{
				GlobalLimit: 8,
			},
			func(gn *GlobalNode, request *limitRequest) {
				require.Equal(t, gn.GlobalLimit, request.GlobalLimit)
			},
		},
		{
			"individual reset update",
			&limitRequest{
				IndividualReset: 2000000000,
			},
			func(gn *GlobalNode, request *limitRequest) {
				require.Equal(t, gn.IndividualReset, request.IndividualReset)
			},
		},
		{
			"global reset update",
			&limitRequest{
				GlobalReset: 3000000000,
			},
			func(gn *GlobalNode, request *limitRequest) {
				require.Equal(t, gn.GlobalReset, request.GlobalReset)
			},
		},
	}
	for _, tc := range updateTestCases {
		t.Run(tc.title, func(t *testing.T) {
			_, err = fsc.updateLimits(ownerTxn, mustEncode(t, tc.request), balances, gn)
			require.NoError(t, err)
			gn = fsc.getGlobalVariables(ownerTxn, balances)
			tc.requireFunc(gn, tc.request)
		})
	}
}

func TestFaucetSmartContractValidateGlobalNode(t *testing.T) {
	now := time.Now()
	for i, tt := range []struct {
		node GlobalNode
		err  string
	}{
		// pour amount too low
		{GlobalNode{"", -1, 0, 0, 0, 0, 0, 0, now}, "failed to validate global node: pour amount(-1) is less than 1"},
		// max pour amount too low
		{GlobalNode{"", 2, 1, 0, 0, 0, 0, 0, now}, "failed to validate global node: max pour amount(1) is less than pour amount(2)"},
		// periodic limit too low
		{GlobalNode{"", 2, 3, 2, 0, 0, 0, 0, now}, "failed to validate global node: periodic limit(2) is less than max pour amount(3)"},
		// global periodic limit too low
		{GlobalNode{"", 2, 3, 4, 3, 0, 0, 0, now}, "failed to validate global node: global periodic limit(3) is less than periodic limit(4)"},
		// global periodic limit too low
		{GlobalNode{"", 2, 3, 4, 5, 0, 0, 0, now}, "failed to validate global node: individual reset(0s) is too short"},
		// global periodic limit too low
		{GlobalNode{"", 2, 3, 4, 5, 2000000000, 1000000000, 0, now}, "failed to validate global node: global reset(1s) is less than individual reset(2s)"},
	} {
		t.Log(i)
		err := tt.node.validate()
		require.Error(t, err)
		require.EqualError(t, err, tt.err)
	}
}
