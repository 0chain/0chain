package storagesc

import (
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
)

type formulae struct {
	blobber     mock0ChainBlobberYml
	sc          scConfig
	ar          newAllocationRequest
	readMarker  ReadMarker
	writeMarker WriteMarker
}

// write formulae
//

func (f formulae) ChallangePoolBalance() (cpBalance state.Balance) {
	var size = sizeInGB(f.writeMarker.Size)
	var price = float64(convertZcnToValue(f.blobber.WritePrice))
	var duration = f.allocRestOfDurationInTUs(f.writeMarker.Timestamp)
	return state.Balance(size * price * duration)

}

// read formulae
//

func (f formulae) RmRewardsCharge() (blobberCharge state.Balance) {
	return state.Balance(f.blobber.ServiceCharge * float64(f.RmValue()))
}

func (f formulae) RmRewardsBlobber() (blobberCharge state.Balance) {
	return state.Balance((1 - f.blobber.ServiceCharge) * float64(f.RmValue()))
}

func (f formulae) RmRewardsValidator() (validatorCharge state.Balance) {
	return 0 // todo implement validators
}

func (f formulae) RmValue() (value state.Balance) {
	rp := float64(convertZcnToValue(f.blobber.ReadPrice))
	return state.Balance(sizeInGB(f.readMarker.ReadCounter*CHUNK_SIZE) * rp)
}

// Allocation formulae
//

func (f formulae) AllocRestMinLockDemandTotal2(value state.Balance, now common.Timestamp) state.Balance {
	return f.AllocRestMinLockDemandTotal(now) - value
}

func (f formulae) AllocRestMinLockDemandTotal(now common.Timestamp) state.Balance {
	var lockPerBlobber = f.allocLockDemandPerBlobber(f.allocPerBlobber(), now)
	return state.Balance(f.ar.DataShards+f.ar.ParityShards) * lockPerBlobber
}

func (f formulae) allocPerBlobber() (restGB float64) {
	var shards = int64(f.ar.DataShards + f.ar.ParityShards)
	var bsize = (f.ar.Size + shards - 1) / shards
	return sizeInGB(bsize)
}

func (f formulae) allocLockDemandPerBlobber(gbSize float64, now common.Timestamp) (mdl state.Balance) {
	var writePrice = float64(convertZcnToValue(f.blobber.WritePrice))
	var remaining = f.allocRestOfDurationInTUs(now)
	return state.Balance(writePrice * gbSize * remaining * f.blobber.MinLockDemand)
}

func (f *formulae) allocRestOfDurationInTUs(now common.Timestamp) (rdtu float64) {
	return f.toTimeUnits(f.ar.Expiration - now)
}

func (f *formulae) toTimeUnits(duration common.Timestamp) (dtu float64) {
	return float64(duration.Duration()) / float64(f.sc.TimeUnit)
}
