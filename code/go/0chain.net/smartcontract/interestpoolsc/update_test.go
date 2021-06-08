package interestpoolsc

import (
	"testing"
	"time"

	configpkg "0chain.net/chaincore/config"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"

	"github.com/stretchr/testify/require"
)

func getGlobalNodeTest() (gn *GlobalNode) {
	const pfx = "smart_contracts.interestpoolsc."

	configpkg.SmartContractConfig.Set(pfx+"max_mint", 100)
	configpkg.SmartContractConfig.Set(pfx+"min_lock", 1000)
	configpkg.SmartContractConfig.Set(pfx+"apr", 0.1)
	configpkg.SmartContractConfig.Set(pfx+"min_lock_period", 24*time.Hour)

	return &GlobalNode{
		ID: ADDRESS,
		SimpleGlobalNode: &SimpleGlobalNode{
			MaxMint: 100 * 1e10,
			MinLock: 1000,
			APR:     0.1,
		},
		MinLockPeriod: 24 * time.Hour,
	}
}

func Test_getConfig(t *testing.T) {
	var (
		ipsc       = newTestInterestPoolSC()
		balances   = newTestBalances()
		configured = getGlobalNodeTest()
		gn         = ipsc.getGlobalNode(balances, "")
	)
	require.Equal(t, configured, gn)
}

func TestInterestPoolSmartContractUpdate(t *testing.T) {
	var (
		ipsc        = newTestInterestPoolSC()
		balances    = newTestBalances()
		tp          = common.Timestamp(0)
		ownerTxn    = newTransaction(owner, ipsc.ID, 0, tp)
		nonOwnerTxn = newTransaction(randString(32), ipsc.ID, 0, tp)
		originalGn  = getGlobalNodeTest()
		gn          = ipsc.getGlobalNode(balances, "updateVariables")
		err         error
	)

	//test cases that produce errors
	errorTestCases := []struct {
		title string
		txn   *transaction.Transaction
		bytes []byte
		err   string
	}{
		{"malformed update", ownerTxn, []byte("} malformed {"), "failed to update variables: request not formatted correctly"},
		{"non owner account", nonOwnerTxn, []byte("} malformed {"), "failed to update variables: unauthorized access - only the owner can update the variables"},
	}
	for _, tc := range errorTestCases {
		t.Run(tc.title, func(t *testing.T) {
			balances.txn = tc.txn
			_, err = ipsc.updateVariables(tc.txn, gn, tc.bytes, balances)
			require.Error(t, err)
			require.EqualError(t, err, tc.err)
		})
	}

	//test cases that will be denied
	deniedTestCases := []struct {
		title       string
		request     *GlobalNode
		requireFunc func(gn, originalGn *GlobalNode)
	}{
		{"all variables denied",
			&GlobalNode{
				SimpleGlobalNode: &SimpleGlobalNode{
					MaxMint: 0,
					MinLock: 0,
					APR:     0.0,
				},
				MinLockPeriod: 0,
			},
			func(gn, originalGn *GlobalNode) {
				require.Equal(t, gn, originalGn)
			},
		},
	}
	for _, tc := range deniedTestCases {
		t.Run(tc.title, func(t *testing.T) {
			balances.txn = ownerTxn
			_, err = ipsc.updateVariables(ownerTxn, gn, tc.request.Encode(), balances)
			require.NoError(t, err)
			tc.requireFunc(gn, originalGn)
		})
	}

	updateTestCases := []struct {
		title       string
		request     *GlobalNode
		requireFunc func(gn *GlobalNode, request *GlobalNode)
	}{
		{
			"max mint update",
			&GlobalNode{
				SimpleGlobalNode: &SimpleGlobalNode{
					MaxMint: 13,
				},
			},
			func(gn, request *GlobalNode) {
				require.Equal(t, gn.MaxMint, request.MaxMint)
			},
		},
		{
			"min lock update",
			&GlobalNode{
				SimpleGlobalNode: &SimpleGlobalNode{
					MinLock: 12345,
				},
			},
			func(gn, request *GlobalNode) {
				require.Equal(t, gn.MinLock, request.MinLock)
			},
		},
		{
			"apr update",
			&GlobalNode{
				SimpleGlobalNode: &SimpleGlobalNode{
					APR: 0.33,
				},
			},
			func(gn, request *GlobalNode) {
				require.Equal(t, gn.APR, request.APR)
			},
		},
		{
			"min lock period update",
			&GlobalNode{
				SimpleGlobalNode: &SimpleGlobalNode{},
				MinLockPeriod:    7 * time.Hour,
			},
			func(gn, request *GlobalNode) {
				require.Equal(t, gn.MinLockPeriod, request.MinLockPeriod)
			},
		},
	}
	for _, tc := range updateTestCases {
		t.Run(tc.title, func(t *testing.T) {
			balances.txn = ownerTxn
			_, err = ipsc.updateVariables(ownerTxn, gn, tc.request.Encode(), balances)
			require.NoError(t, err)
			gn = ipsc.getGlobalNode(balances, "")
			tc.requireFunc(gn, tc.request)
		})
	}

}

func TestInterestPoolSmartContractValidateGlobalNode(t *testing.T) {
	for _, tt := range []struct {
		node GlobalNode
		err  string
	}{
		// apr too low
		{GlobalNode{"", &SimpleGlobalNode{0, 0, 0, -0.1}, 0}, "failed to validate global node: apr(-0.1) is too low"},
		// min lock period too low
		{GlobalNode{"", &SimpleGlobalNode{0, 0, 0, 0.1}, -1000000000}, "failed to validate global node: min lock period(-1s) is too short"},
		// min lock too low
		{GlobalNode{"", &SimpleGlobalNode{0, 0, -1, 0.1}, 1}, "failed to validate global node: min lock(-1) is too low"},
		// max mint too low
		{GlobalNode{"", &SimpleGlobalNode{-1, 0, 1, 0.1}, 1}, "failed to validate global node: max mint(-1) is too low"},
	} {
		err := tt.node.validate()
		require.Error(t, err)
		require.EqualError(t, err, tt.err)
	}
}
