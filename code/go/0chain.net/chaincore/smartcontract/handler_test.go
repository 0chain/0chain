package smartcontract_test

import (
	"testing"

	"github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/require"

	"github.com/0chain/0chain/code/go/0chain.net/chaincore/config"
	. "github.com/0chain/0chain/code/go/0chain.net/chaincore/smartcontract"
	"github.com/0chain/0chain/code/go/0chain.net/core/viper"
	"github.com/0chain/0chain/code/go/0chain.net/smartcontract/faucetsc"
	"github.com/0chain/0chain/code/go/0chain.net/smartcontract/interestpoolsc"
	"github.com/0chain/0chain/code/go/0chain.net/smartcontract/minersc"
	"github.com/0chain/0chain/code/go/0chain.net/smartcontract/multisigsc"
	"github.com/0chain/0chain/code/go/0chain.net/smartcontract/setupsc"
	"github.com/0chain/0chain/code/go/0chain.net/smartcontract/storagesc"
	"github.com/0chain/0chain/code/go/0chain.net/smartcontract/vestingsc"
	"github.com/0chain/0chain/code/go/0chain.net/smartcontract/zrc20sc"
)

func init() {
	metrics.DefaultRegistry = metrics.NewRegistry()
	config.SmartContractConfig = viper.New()
	setupsc.SetupSmartContracts()
	viper.Set("development.smart_contract.faucet", true)
	viper.Set("development.smart_contract.storage", true)
	viper.Set("development.smart_contract.zrc20", true)
	viper.Set("development.smart_contract.interest", true)
	viper.Set("development.smart_contract.multisig", true)
	viper.Set("development.smart_contract.miner", true)
	viper.Set("development.smart_contract.vesting", true)
	setupsc.SetupSmartContracts()
}

func TestGetSmartContract(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		address    string
		restpoints int
		null       bool
	}{
		{
			name:       "faucet",
			address:    faucetsc.ADDRESS,
			restpoints: 4,
		},
		{
			name:       "storage",
			address:    storagesc.ADDRESS,
			restpoints: 16,
		},
		{
			name:       "zrc20",
			address:    zrc20sc.ADDRESS,
			restpoints: 0,
		},
		{
			name:       "interest",
			address:    interestpoolsc.ADDRESS,
			restpoints: 2,
		},
		{
			name:       "multisig",
			address:    multisigsc.Address,
			restpoints: 0,
		},
		{
			name:       "miner",
			address:    minersc.ADDRESS,
			restpoints: 13,
		},
		{
			name:       "vesting",
			address:    vestingsc.ADDRESS,
			restpoints: 3,
		},
		{
			name:    "Nil_OK",
			address: "not an address",
			null:    true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := GetSmartContract(tt.address)
			require.True(t, tt.null == (got == nil))
			if got == nil {
				return
			}
			require.EqualValues(t, tt.name, got.GetName())
			require.EqualValues(t, tt.address, got.GetAddress())
			require.EqualValues(t, tt.restpoints, len(got.GetRestPoints()))
		})
	}
}
