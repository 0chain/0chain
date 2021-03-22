package storagesc

import (
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
)

// Calculates important 0chain values defined from config
// logs and cli input parameters.
// blobber = 0chain_blobber.yaml
// sc = sc.yaml
// ar input to ./zbox newallocation
// readMarker internal parameter object for reads
// writeMarker internal parameter object for writes
//
type formulae struct {
	blobber     mockBlobberYml
	sc          scConfig
	ar          newAllocationRequest
	readMarker  ReadMarker
	writeMarker WriteMarker
}

// amount to charge a write for each unit of time
func (f formulae) writeChargeRate() state.Balance {
	var writeSizePerGB = sizeInGB(f.writeMarker.Size)
	var writePriceGB = float64(zcnToBalance(f.blobber.WritePrice))

	return state.Balance(writeSizePerGB * writePriceGB)
}

// amount to charge for a write lock
func (f formulae) lockCostForWrite() state.Balance {
	var writeChargeRate = float64(f.writeChargeRate())
	var timeLeft = f.lockTimeLeftTU()

	return state.Balance(writeChargeRate * timeLeft)
}

// service charge for a read
func (f formulae) readServiceCharge() state.Balance {
	var serviceChargeFraction = f.blobber.ServiceCharge
	var readCost = float64(f.readCost())

	return state.Balance(serviceChargeFraction * readCost)
}

// blobber reward for a read
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

// cost of a read as defined in read marker
//
func (f formulae) readCost() (value state.Balance) {
	var readPricePerGB = float64(zcnToBalance(f.blobber.ReadPrice))
	var readSizeGB = sizeInGB(f.readMarker.ReadCounter * CHUNK_SIZE)

	return state.Balance(readSizeGB * readPricePerGB)
}

// Utility functions
//

// time remaining in an allocation lock
func (f formulae) remainingTimeTUs(now common.Timestamp) float64 {
	var expiration = f.ar.Expiration

	return f.toTimeUnits(expiration - now)
}

// convert to time units, must be defined in sc.yaml
func (f formulae) toTimeUnits(duration common.Timestamp) float64 {
	if f.sc.TimeUnit == 0 {
		panic("must be > 0, make sure you are setting f.sc")
	}
	return float64(duration.Duration()) / float64(f.sc.TimeUnit)
}

// time remaining in allocation lock at the moment of a write
func (f formulae) lockTimeLeftTU() float64 {
	var now = f.writeMarker.Timestamp

	return f.remainingTimeTUs(now)
}
