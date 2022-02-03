package multisigsc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/viper"
	"0chain.net/smartcontract/benchmark"
)

func AddMockWallets(
	clients, publicKeys []string,
	balances cstate.StateContextI,
) {
	for i := 0; i < len(clients)-1; i++ {
		wallet := Wallet{
			ClientID:           clients[i],
			SignatureScheme:    viper.GetString(benchmark.InternalSignatureScheme),
			PublicKey:          publicKeys[i],
			SignerPublicKeys:   publicKeys[:MaxSigners],
			SignerThresholdIDs: clients[:MaxSigners],
			NumRequired:        MaxSigners,
		}
		err := balances.InsertTrieNode(getWalletKey(clients[i]), &wallet)
		if err != nil {
			panic(err)
		}
	}
}
