package storagesc

import (
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
)

type formulae struct {
	blobber mock0ChainBlobberYml
	sc      scConfig
	ar      newAllocationRequest
}

func (f *formulae) ResMinLockDemandTotal(now common.Timestamp) state.Balance {
	var gbSize = f.allocationPerBlobber()
	var mlockShard = f.minLockDemandPerBlobber(gbSize, now)
	return state.Balance(f.ar.DataShards+f.ar.ParityShards) * mlockShard
}

func (f *formulae) allocationPerBlobber() (restGB float64) {
	var shards = int64(f.ar.DataShards + f.ar.ParityShards)
	var bsize = (f.ar.Size + shards - 1) / shards
	return sizeInGB(bsize)
}

func (f *formulae) minLockDemandPerBlobber(gbSize float64, now common.Timestamp) (mdl state.Balance) {
	var writePrice = float64(convertZcnToValue(f.blobber.WritePrice))
	var remaining = f.restDurationInTimeUnits(now)
	return state.Balance(writePrice * gbSize * remaining * f.blobber.MinLockDemand)
}

// time left before expiration in fractions of a time unit.
func (f *formulae) restDurationInTimeUnits(now common.Timestamp) (rdtu float64) {
	rdtu = float64((f.ar.Expiration - now).Duration()) / float64(f.sc.TimeUnit)
	return
}
