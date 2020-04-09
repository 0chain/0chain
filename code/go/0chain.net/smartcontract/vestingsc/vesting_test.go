package vestingsc

import (
	"testing"
	"time"

	"0chain.net/chaincore/state"
	"0chain.net/core/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_toSeconds(t *testing.T) {
	assert.Equal(t, common.Timestamp(1),
		toSeconds(1*time.Second+500*time.Millisecond))
}

func Test_lockRequest_decode(t *testing.T) {
	var lr lockRequest
	require.NoError(t, lr.decode([]byte(`{"pool_id":"pool_hex"}`)))
	assert.Equal(t, "pool_hex", lr.PoolID)
}

func Test_addRequest_decode(t *testing.T) {
	var are, ard addRequest
	are.Description = "for something"
	are.StartTime = 10
	are.Duration = 2 * time.Second
	are.Friquency = 3 * time.Second
	are.Destinations = []string{"one", "two"}
	are.Amount = 400
	require.NoError(t, ard.decode(mustEncode(t, &are)))
	assert.EqualValues(t, &are, &ard)
}

func Test_addRequest_validate(t *testing.T) {
	var (
		conf = avgConfig()
		ar   addRequest
	)
	ar.Description = "very very very long description"
	assertErrMsg(t, ar.validate(10, conf), "entry description is too long")
	ar.Description = "short desc."

	ar.StartTime = 1
	assertErrMsg(t, ar.validate(10, conf), "vesting starts before now")
	ar.StartTime = 20

	assertErrMsg(t, ar.validate(10, conf), "vesting duration is too short")
	ar.Duration = 2 * time.Hour
	assertErrMsg(t, ar.validate(10, conf), "vesting duration is too long")
	ar.Duration = 1 * time.Minute

	assertErrMsg(t, ar.validate(10, conf), "vesting friquency is too low")
	ar.Friquency = 2 * time.Hour
	assertErrMsg(t, ar.validate(10, conf), "vesting friquency is too high")
	ar.Friquency = 1 * time.Minute

	assertErrMsg(t, ar.validate(10, conf), "no destinations")
	ar.Destinations = []string{"one", "two", "three"}
	assertErrMsg(t, ar.validate(10, conf), "too many destinations")
	ar.Destinations = []string{"one", "two"}

	assert.NoError(t, ar.validate(10, conf))
	ar.StartTime = 0
	assert.NoError(t, ar.validate(10, conf))
}

func Test_vestingPool(t *testing.T) {
	const poolID, clientID = "pool_hex", "client_hex"
	require.NotZero(t, poolKey(ADDRESS, poolID))
	var vp = newVestingPool()
	assert.NotNil(t, vp)
	assert.NotNil(t, vp.TokenPool)
	var ar addRequest
	ar.Description = "for something"
	ar.StartTime = 10
	ar.Duration = 2 * time.Second
	ar.Friquency = 3 * time.Second
	ar.Destinations = []string{"one", "two"}
	ar.Amount = 400
	vp = newVestingPoolFromReqeust(clientID, &ar)
	assert.NotNil(t, vp)
	assert.NotNil(t, vp.TokenPool)

	assert.Zero(t, vp.Last)

	assert.Equal(t, vp.Description, ar.Description)
	assert.Equal(t, vp.StartTime, ar.StartTime)
	assert.Equal(t, vp.ExpireAt, ar.StartTime+toSeconds(ar.Duration))
	assert.Equal(t, vp.Friquency, ar.Friquency)
	assert.Equal(t, vp.Destinations, ar.Destinations)
	assert.Equal(t, vp.Amount, ar.Amount)

	var vpd = new(vestingPool)
	require.NoError(t, vpd.Decode(vp.Encode()))
	assert.Equal(t, vp, vpd)

	var inf = vpd.info()
	assert.Equal(t, vp.Description, inf.Description)
	assert.Equal(t, vp.StartTime, inf.StartTime)
	assert.Equal(t, vp.ExpireAt, inf.ExpireAt)
	assert.Equal(t, vp.Friquency, inf.Friquency)
	assert.Equal(t, vp.Destinations, inf.Destinations)
	assert.Equal(t, vp.Amount, inf.Amount)
	assert.Equal(t, state.Balance(0), inf.Balance)
	assert.Equal(t, common.Timestamp(0), inf.Last)
}

/*

TODO (sfxdx): cover

func (vp *vestingPool) vestTo(vscKey string, toClientID datastore.Key,
func (vp *vestingPool) vest(vscKey string, now common.Timestamp,
func (vp *vestingPool) fill(t *transaction.Transaction,
func (vp *vestingPool) empty(t *transaction.Transaction,
func (vp *vestingPool) save(balances chainstate.StateContextI) (err error) {

func (vsc *VestingSmartContract) getPoolBytes(poolID datastore.Key,
func (vsc *VestingSmartContract) getPool(poolID datastore.Key,
func (vsc *VestingSmartContract) checkFill(t *transaction.Transaction,
func (vsc *VestingSmartContract) add(t *transaction.Transaction,
func (vsc *VestingSmartContract) delete(t *transaction.Transaction,
func (vsc *VestingSmartContract) lock(t *transaction.Transaction, input []byte,
func (vsc *VestingSmartContract) unlock(t *transaction.Transaction,
func (vsc *VestingSmartContract) trigger(t *transaction.Transaction,
func (vsc *VestingSmartContract) getPoolInfoHandler(ctx context.Context,
*/
