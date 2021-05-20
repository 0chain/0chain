package scfactory_test

import (
	"0chain.net/chaincore/config"
	. "0chain.net/chaincore/scfactory"
	"0chain.net/chaincore/smartcontract"
	"0chain.net/smartcontract/faucetsc"
	"0chain.net/smartcontract/interestpoolsc"
	"0chain.net/smartcontract/minersc"
	"0chain.net/smartcontract/multisigsc"
	"0chain.net/smartcontract/setupsc"
	"0chain.net/smartcontract/storagesc"
	"0chain.net/smartcontract/vestingsc"
	"0chain.net/smartcontract/zrc20sc"
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
}

func TestGetSmartContract(t *testing.T) {
	t.Parallel()

	SetUpSmartContractFactory()

	tests := []struct {
		name       setupsc.SCName
		address    string
		restpoints int
		null       bool
	}{
		{
			name:       setupsc.Faucet,
			address:    faucetsc.ADDRESS,
			restpoints: 4,
		},
		{
			name:       setupsc.Storage,
			address:    storagesc.ADDRESS,
			restpoints: 16,
		},
		{
			name:       setupsc.Zrc20,
			address:    zrc20sc.ADDRESS,
			restpoints: 0,
		},
		{
			name:       setupsc.Interest,
			address:    interestpoolsc.ADDRESS,
			restpoints: 2,
		},
		{
			name:       setupsc.Multisig,
			address:    multisigsc.Address,
			restpoints: 0,
		},
		{
			name:       setupsc.Miner,
			address:    minersc.ADDRESS,
			restpoints: 13,
		},
		{
			name:       setupsc.Vesting,
			address:    vestingsc.ADDRESS,
			restpoints: 3,
		},
		{
			name:    setupsc.SCName("Nil_OK"),
			address: "not an address",
			null:    true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(string(tt.name), func(t *testing.T) {
			t.Parallel()
			sci, sc := smartcontract.SmartContractFactory.NewSmartContract(string(tt.name))
			if !tt.null == (sci == nil) {
				require.True(t, true)
			}
			require.True(t, tt.null == (sci == nil) && tt.null == (sc == nil))
			if sci == nil || sc == nil {
				return
			}
			require.EqualValues(t, tt.name, sci.GetName())
			require.EqualValues(t, tt.address, sci.GetAddress())
			require.EqualValues(t, tt.address, sc.ID)
			require.EqualValues(t, tt.restpoints, len(sci.GetRestPoints()))
		})
	}
}
