package smartcontract_test

import (
	"0chain.net/chaincore/config"
	. "0chain.net/chaincore/smartcontract"
	"0chain.net/smartcontract/faucetsc"
	"0chain.net/smartcontract/interestpoolsc"
	"0chain.net/smartcontract/minersc"
	"0chain.net/smartcontract/multisigsc"
	"0chain.net/smartcontract/setupsc"
	"0chain.net/smartcontract/storagesc"
	"0chain.net/smartcontract/vestingsc"
	"0chain.net/smartcontract/zrc20sc"
	"fmt"
	"github.com/rcrowley/go-metrics"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"testing"
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
			if !tt.null == (got == nil) {
				require.True(t, true)
			}
			require.True(t, tt.null == (got == nil))
			if got == nil {
				return
			}
			fmt.Println("name", got.GetName(), "address", got.GetAddress(), "rest points", len(got.GetRestPoints()))
			require.EqualValues(t, tt.name, got.GetName())
			require.EqualValues(t, tt.address, got.GetAddress())
			require.EqualValues(t, tt.restpoints, len(got.GetRestPoints()))
		})
	}
}
