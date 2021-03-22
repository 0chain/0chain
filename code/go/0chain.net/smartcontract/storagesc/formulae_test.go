package storagesc

import (
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
)

type formulae struct {
	blobber     mockBlobberYml
	sc          scConfig
	ar          newAllocationRequest
	readMarker  ReadMarker
	writeMarker WriteMarker
}

func (f formulae) writeChargeRate() state.Balance {
	var writeSizePerGB = sizeInGB(f.writeMarker.Size)
	var writePriceGB = float64(zcnToBalance(f.blobber.WritePrice))

	return state.Balance(writeSizePerGB * writePriceGB)
}

func (f formulae) lockCostForWrite() state.Balance {
	var writeChargeRate = float64(f.writeChargeRate())

	return state.Balance(writeChargeRate * f.lockTimeLeftTU())
}

func (f formulae) readCharge() state.Balance {
	var serviceChargeFraction = f.blobber.ServiceCharge
	var readCost = float64(f.readCost())

	return state.Balance(serviceChargeFraction * readCost)
}

func (f formulae) readRewardsBlobber() (blobberCharge state.Balance) {
	var blobberRewardFraction = 1 - f.blobber.ServiceCharge
	var readCost = float64(f.readCost())

	return state.Balance(blobberRewardFraction * readCost)
}

// todo add validators
func (f formulae) readRewardsValidator() state.Balance {
	panic("validators not implemented")
	return 0
}

// In blobber.go StorageSmartContract.commitBlobberRead blobber.go
// https://github.com/0chain/0chain/blob/master/code/go/0chain.net/smartcontract/storagesc/blobber.go#L462
//
func (f formulae) readCost() (value state.Balance) {
	var readPricePerGB = float64(zcnToBalance(f.blobber.ReadPrice))
	var readSizeGB = sizeInGB(f.readMarker.ReadCounter * CHUNK_SIZE)

	return state.Balance(readSizeGB * readPricePerGB)
}

// Allocation formulae
//
func (f formulae) allocRestMinLockDemandTotal(now common.Timestamp) state.Balance {
	var lockPerBlobber = f.allocLockDemandPerBlobber(f.allocPerBlobber(), now)
	return state.Balance(f.ar.DataShards+f.ar.ParityShards) * lockPerBlobber
}

func (f formulae) allocPerBlobber() float64 {
	var shards = int64(f.ar.DataShards + f.ar.ParityShards)
	var bSize = (f.ar.Size + shards - 1) / shards
	return sizeInGB(bSize)
}

func (f formulae) allocLockDemandPerBlobber(gbSize float64, now common.Timestamp) state.Balance {
	var writePrice = float64(zcnToBalance(f.blobber.WritePrice))
	var remaining = f.remainingTimeTUs(now)
	return state.Balance(writePrice * gbSize * remaining * f.blobber.MinLockDemand)
}

// Utility functions
//
func (f formulae) remainingTimeTUs(now common.Timestamp) float64 {
	return f.toTimeUnits(f.ar.Expiration - now)
}

func (f formulae) toTimeUnits(duration common.Timestamp) float64 {
	return float64(duration.Duration()) / float64(f.sc.TimeUnit)
}

func (f formulae) lockTimeLeftTU() float64 {
	return f.remainingTimeTUs(f.writeMarker.Timestamp)
}
