package faucetsc

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
		tx         = newTransaction(owner, fsc.ID, 0, tp)
		configured = getGlobalNodeTest()
		gn         = fsc.getGlobalVariables(tx, balances)
	)
	assert.EqualValues(t, configured, gn)
}

func TestFaucetSmartContractUpdate(t *testing.T) {
	var (
		fsc        = newTestFaucetSC()
		balances   = newTestBalances()
		tp         = common.Timestamp(0)
		tx         = newTransaction(owner, fsc.ID, 0, tp)
		originalGn = getGlobalNodeTest()
		gn         = fsc.getGlobalVariables(tx, balances)
		lr         = &limitRequest{}
		err        error
	)

	balances.txn = tx

	// 1. Malformed limit request
	t.Run("malformed request", func(t *testing.T) {
		_, err = fsc.updateLimits(tx, []byte("} malformed {"), balances, gn)
		assertErrMsg(t, err, "bad_request: limit request not formated correctly")
	})

	// 2. Non owner account tries to update
	t.Run("non owner account", func(t *testing.T) {
		tx.ClientID = randString(32)
		_, err = fsc.updateLimits(tx, []byte("} malformed {"), balances, gn)
		assertErrMsg(t, err, "unauthorized_access: only the owner can update the limits")
	})

	// 3. All variables are request shall be denied
	t.Run("all variables denied", func(t *testing.T) {
		lr.PourAmount = -1
		lr.MaxPourAmount = -1
		lr.PeriodicLimit = -1
		lr.GlobalLimit = -1
		lr.IndividualReset = 1000000000
		lr.GlobalReset = 1000000000
		tx.ClientID = owner
		_, err = fsc.updateLimits(tx, mustEncode(t, lr), balances, gn)
		require.NoError(t, err)
		gn = fsc.getGlobalVariables(tx, balances)
		assert.EqualValues(t, gn.PourAmount, originalGn.PourAmount)
		assert.EqualValues(t, gn.MaxPourAmount, originalGn.MaxPourAmount)
		assert.EqualValues(t, gn.PeriodicLimit, originalGn.PeriodicLimit)
		assert.EqualValues(t, gn.GlobalLimit, originalGn.GlobalLimit)
		assert.EqualValues(t, gn.IndividualReset, originalGn.IndividualReset)
		assert.EqualValues(t, gn.GlobalReset, originalGn.GlobalReset)
	})

	// 4. Pour amount will be the only one updated
	t.Run("pour amount update", func(t *testing.T) {
		lr.PourAmount = 1
		_, err = fsc.updateLimits(tx, mustEncode(t, lr), balances, gn)
		require.NoError(t, err)
		gn = fsc.getGlobalVariables(tx, balances)
		assert.EqualValues(t, gn.PourAmount, lr.PourAmount)
	})

	// 5. Max pour amount too small for update
	t.Run("max pour amount fail", func(t *testing.T) {
		lr.PourAmount = 5
		lr.MaxPourAmount = 4
		_, err = fsc.updateLimits(tx, mustEncode(t, lr), balances, gn)
		require.NoError(t, err)
		gn = fsc.getGlobalVariables(tx, balances)
		assert.EqualValues(t, gn.MaxPourAmount, originalGn.MaxPourAmount)
	})

	// 6. Max pour amount accepted
	t.Run("max pour amount update", func(t *testing.T) {
		lr.MaxPourAmount = 6
		_, err = fsc.updateLimits(tx, mustEncode(t, lr), balances, gn)
		require.NoError(t, err)
		gn = fsc.getGlobalVariables(tx, balances)
		assert.EqualValues(t, gn.MaxPourAmount, lr.MaxPourAmount)
	})

	// 7. Periodic limit too small for update
	t.Run("periodic limit fail", func(t *testing.T) {
		lr.PeriodicLimit = 5
		_, err = fsc.updateLimits(tx, mustEncode(t, lr), balances, gn)
		require.NoError(t, err)
		gn = fsc.getGlobalVariables(tx, balances)
		assert.EqualValues(t, gn.PeriodicLimit, originalGn.PeriodicLimit)
	})

	// 8. Periodic limit accepted
	t.Run("period limit update", func(t *testing.T) {
		lr.PeriodicLimit = 7
		_, err = fsc.updateLimits(tx, mustEncode(t, lr), balances, gn)
		require.NoError(t, err)
		gn = fsc.getGlobalVariables(tx, balances)
		assert.EqualValues(t, gn.PeriodicLimit, lr.PeriodicLimit)
	})

	// 9. Global limit too small for update
	t.Run("global limit fail", func(t *testing.T) {
		lr.GlobalLimit = 7
		_, err = fsc.updateLimits(tx, mustEncode(t, lr), balances, gn)
		require.NoError(t, err)
		gn = fsc.getGlobalVariables(tx, balances)
		assert.EqualValues(t, gn.GlobalLimit, originalGn.GlobalLimit)
	})

	// 10. Global limit accepted
	t.Run("global limit update", func(t *testing.T) {
		lr.GlobalLimit = 8
		_, err = fsc.updateLimits(tx, mustEncode(t, lr), balances, gn)
		require.NoError(t, err)
		gn = fsc.getGlobalVariables(tx, balances)
		assert.EqualValues(t, gn.GlobalLimit, lr.GlobalLimit)
	})

	// 11. Individual reset too small for update
	t.Run("individual reset fail", func(t *testing.T) {
		lr.IndividualReset = 0
		_, err = fsc.updateLimits(tx, mustEncode(t, lr), balances, gn)
		require.NoError(t, err)
		gn = fsc.getGlobalVariables(tx, balances)
		assert.EqualValues(t, gn.IndividualReset, originalGn.IndividualReset)
	})

	// 12. Individual reset accepted
	t.Run("individual reset update", func(t *testing.T) {
		lr.IndividualReset = 2000000000
		_, err = fsc.updateLimits(tx, mustEncode(t, lr), balances, gn)
		require.NoError(t, err)
		gn = fsc.getGlobalVariables(tx, balances)
		assert.EqualValues(t, gn.IndividualReset, lr.IndividualReset)
	})

	// 13. Global reset too small for update
	t.Run("global reset fail", func(t *testing.T) {
		lr.GlobalReset = 0
		_, err = fsc.updateLimits(tx, mustEncode(t, lr), balances, gn)
		require.NoError(t, err)
		gn = fsc.getGlobalVariables(tx, balances)
		assert.EqualValues(t, gn.GlobalReset, originalGn.GlobalReset)
	})

	// 14. Global reset accepted
	t.Run("global reset update", func(t *testing.T) {
		lr.GlobalReset = 3000000000
		_, err = fsc.updateLimits(tx, mustEncode(t, lr), balances, gn)
		require.NoError(t, err)
		gn = fsc.getGlobalVariables(tx, balances)
		assert.EqualValues(t, gn.GlobalReset, lr.GlobalReset)
	})
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
		assertErrMsg(t, tt.node.validate(), tt.err)
	}
}
