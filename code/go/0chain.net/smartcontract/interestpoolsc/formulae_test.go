package interestpoolsc

import (
	"0chain.net/chaincore/state"
)

// Calculates important 0chain values defined from config
// logs and cli input parameters.
// sc = sc.yaml
// lockFlags input to ./zwallet lock
//
type formulae struct {
	sc        mockScYml
	lockFlags lockFlags
}

// interest earned from a waller lock cli command
func (f formulae) tokensEarned() state.Balance {
	var amount = float64(zcnToBalance(f.lockFlags.tokens))
	var apr = f.sc.apr
	var duration = float64(f.lockFlags.duration)
	var year = float64(YEAR)

	return state.Balance(amount * apr * duration / year)
}
