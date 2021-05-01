package interestpoolsc

import (
	"testing"
	"time"

	configpkg "0chain.net/chaincore/config"
	"0chain.net/core/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertErrMsg(t *testing.T, err error, msg string) {
	t.Helper()

	if msg == "" {
		assert.Nil(t, err)
		return
	}

	if assert.NotNil(t, err) {
		assert.Equal(t, msg, err.Error())
	}
}

func getGlobalNodeTest() (gn *GlobalNode) {
	const pfx = "smart_contracts.interestpoolsc."

	configpkg.SmartContractConfig.Set(pfx+"max_mint", 100)
	configpkg.SmartContractConfig.Set(pfx+"min_lock", 1000)
	configpkg.SmartContractConfig.Set(pfx+"apr", 0.1)
	configpkg.SmartContractConfig.Set(pfx+"min_lock_period", 24*time.Hour)

	return &GlobalNode{
		ADDRESS, 100 * 1e10, 0, 1000, 0.1, 24 * time.Hour,
	}
}

func Test_getConfig(t *testing.T) {
	var (
		ipsc       = newTestInterestPoolSC()
		balances   = newTestBalances()
		configured = getGlobalNodeTest()
		gn         = ipsc.getGlobalNode(balances, "")
	)
	assert.EqualValues(t, configured, gn)
}

func TestInterestPoolSmartContractUpdate(t *testing.T) {
	var (
		ipsc       = newTestInterestPoolSC()
		balances   = newTestBalances()
		tp         = common.Timestamp(0)
		tx         = newTransaction(owner, ipsc.ID, 0, tp)
		originalGn = getGlobalNodeTest()
		gn         = ipsc.getGlobalNode(balances, "updateVariables")
		err        error
		update     = &GlobalNode{}
	)

	balances.txn = tx

	// 1. Malformed update
	t.Run("malformed update", func(t *testing.T) {
		_, err = ipsc.updateVariables(tx, gn, []byte("} malformed {"), balances)
		assertErrMsg(t, err, "failed to update variables: request not formatted correctly")
	})

	// 2. Non owner account tries to update
	t.Run("non owner account", func(t *testing.T) {
		tx.ClientID = randString(32)
		_, err = ipsc.updateVariables(tx, gn, []byte("} malformed {"), balances)
		assertErrMsg(t, err, "failed to update variables: unauthorized access - only the owner can update the variables")
	})

	// 3. All variables requested shall be denied
	t.Run("all variables denied", func(t *testing.T) {
		update.MaxMint = 0
		update.MinLock = 0
		update.APR = 0.0
		update.MinLockPeriod = 0

		tx.ClientID = owner
		_, err = ipsc.updateVariables(tx, gn, mustEncode(t, update), balances)
		require.NoError(t, err)
		gn = ipsc.getGlobalNode(balances, "")
		assert.EqualValues(t, gn, originalGn)
	})

	// 4. Max mint will updated
	t.Run("max mint update", func(t *testing.T) {
		update.MaxMint = 13
		_, err = ipsc.updateVariables(tx, gn, mustEncode(t, update), balances)
		require.NoError(t, err)
		gn = ipsc.getGlobalNode(balances, "")
		assert.EqualValues(t, gn.MaxMint, update.MaxMint)
	})

	// 5. Min lock will updated
	t.Run("min lock update", func(t *testing.T) {
		update.MinLock = 12345
		_, err = ipsc.updateVariables(tx, gn, mustEncode(t, update), balances)
		require.NoError(t, err)
		gn = ipsc.getGlobalNode(balances, "")
		assert.EqualValues(t, gn.MinLock, update.MinLock)
	})

	// 6. APR will updated
	t.Run("apr update", func(t *testing.T) {
		update.APR = 0.33
		_, err = ipsc.updateVariables(tx, gn, mustEncode(t, update), balances)
		require.NoError(t, err)
		gn = ipsc.getGlobalNode(balances, "")
		assert.EqualValues(t, gn.APR, update.APR)
	})

	// 7. Min lock period will updated
	t.Run("min lock period update", func(t *testing.T) {
		update.MinLockPeriod = 7 * time.Hour
		_, err = ipsc.updateVariables(tx, gn, mustEncode(t, update), balances)
		require.NoError(t, err)
		gn = ipsc.getGlobalNode(balances, "")
		assert.EqualValues(t, gn.MinLockPeriod, update.MinLockPeriod)
	})
}

func TestInterestPoolSmartContractValidateGlobalNode(t *testing.T) {
	for i, tt := range []struct {
		node GlobalNode
		err  string
	}{
		// apr too low
		{GlobalNode{"", 0, 0, 0, -0.1, 0}, "failed to validate global node: apr(-0.1) is too low"},
		// min lock period too low
		{GlobalNode{"", 0, 0, 0, 0.1, -1000000000}, "failed to validate global node: min lock period(-1s) is too short"},
		// min lock too low
		{GlobalNode{"", 0, 0, -1, 0.1, 1}, "failed to validate global node: min lock(-1) is too low"},
		// max mint too low
		{GlobalNode{"", -1, 0, 1, 0.1, 1}, "failed to validate global node: max mint(-1) is too low"},
	} {
		t.Log(i)
		assertErrMsg(t, tt.node.validate(), tt.err)
	}
}
