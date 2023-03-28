package storagesc

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidatorHealthCheck(t *testing.T) {

	var (
		ssc      = newTestStorageSC()
		balances = newTestBalances(t, false)

		tp int64 = 100
	)

	setConfig(t, balances)

	var (
		validator = addValidator(t, ssc, tp, balances)
		v, err    = getValidator(validator.id, balances)
	)

	require.NoError(t, err)

	_, err = healthCheckValidator(t, v, 0, tp, ssc, balances)

	require.NoError(t, err)
}
