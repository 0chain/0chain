package storagesc

import (
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
)

type formulae struct {
	blobber    mock0ChainBlobberYml
	sc         scConfig
	ar         newAllocationRequest
	readMarker ReadMarker
}

// read formulae
//

func (f *formulae) RmRewardsCharge() (blobberCharge state.Balance) {
	return state.Balance(f.blobber.ServiceCharge * float64(f.RmValue()))
}

func (f *formulae) RmRewardsBlobber() (blobberCharge state.Balance) {
	return state.Balance((1 - f.blobber.ServiceCharge) * float64(f.RmValue()))
}

func (f *formulae) RmRewardsValidator() (validatorCharge state.Balance) {
	return 0 // todo implement validators
}

func (f *formulae) RmValue() (value state.Balance) {
	rp := float64(convertZcnToValue(f.blobber.ReadPrice))
	return state.Balance(sizeInGB(f.readMarker.ReadCounter*CHUNK_SIZE) * rp)
}

// Allocation formulae
//

func (f *formulae) AllocRestMinLockDemandTotal2(value state.Balance, now common.Timestamp) state.Balance {
	return f.AllocRestMinLockDemandTotal(now) - value
}

func (f *formulae) AllocRestMinLockDemandTotal(now common.Timestamp) state.Balance {
	var gbSize = f.allocPerBlobber()
	var mlockShard = f.allocLockDemandPerBlobber(gbSize, now)
	a := f.ar.DataShards
	a = a
	b := f.ar.ParityShards
	b = b
	c := state.Balance(f.ar.DataShards+f.ar.ParityShards) * mlockShard
	c = c
	return state.Balance(f.ar.DataShards+f.ar.ParityShards) * mlockShard
}

func (f *formulae) allocPerBlobber() (restGB float64) {
	var shards = int64(f.ar.DataShards + f.ar.ParityShards)
	var bsize = (f.ar.Size + shards - 1) / shards
	return sizeInGB(bsize)
}

func (f *formulae) allocLockDemandPerBlobber(gbSize float64, now common.Timestamp) (mdl state.Balance) {
	var writePrice = float64(convertZcnToValue(f.blobber.WritePrice))
	var remaining = f.allocDurationInTimeUnits(now)
	return state.Balance(writePrice * gbSize * remaining * f.blobber.MinLockDemand)
}

// time left before expiration in fractions of a time unit.
func (f *formulae) allocDurationInTimeUnits(now common.Timestamp) (rdtu float64) {
	rdtu = float64((f.ar.Expiration - now).Duration()) / float64(f.sc.TimeUnit)
	return
}
