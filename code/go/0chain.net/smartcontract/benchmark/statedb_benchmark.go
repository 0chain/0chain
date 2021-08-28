package benchmark

import (
	"encoding/hex"
	"testing"

	"0chain.net/smartcontract"

	"0chain.net/chaincore/block"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
	"0chain.net/smartcontract/minersc"
	"0chain.net/smartcontract/storagesc"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func getBalances(
	b *testing.B,
	name string,
	txn *transaction.Transaction,
	root util.Key,
	pMpt *util.MerklePatriciaTrie,
) (*util.MerklePatriciaTrie, cstate.StateContextI) {
	pNode := pMpt.GetNodeDB()
	memNode := util.NewMemoryNodeDB()
	levelNode := util.NewLevelNodeDB(
		memNode,
		pNode,
		false,
	)
	mpt := util.NewMerklePatriciaTrie(
		levelNode,
		1,
		root,
	)
	bk := &block.Block{}
	magicBlock := &block.MagicBlock{}
	signatureScheme := &encryption.BLS0ChainScheme{}
	return mpt, cstate.NewStateContext(
		bk,
		mpt,
		&state.Deserializer{},
		txn,
		func(*block.Block) []string { return []string{} },
		func() *block.Block { return bk },
		func() *block.MagicBlock { return magicBlock },
		func() encryption.SignatureScheme { return signatureScheme },
	)
}

func setUpMpt(
	b *testing.B, vi *viper.Viper,
) (*util.MerklePatriciaTrie, util.Key, []string, []string, []string, []string) {
	pNode, err := util.NewPNodeDB(
		"testdata/name_dataDir",
		"testdata/name_logDir",
	)
	require.NoError(b, err)
	pMpt := util.NewMerklePatriciaTrie(
		pNode,
		1,
		nil,
	)

	clients, keys := AddMockkClients(b, pMpt, vi)

	pMpt.GetNodeDB().(*util.PNodeDB).TrackDBVersion(1)

	bk := &block.Block{}
	magicBlock := &block.MagicBlock{}
	signatureScheme := &encryption.BLS0ChainScheme{}
	balances := cstate.NewStateContext(
		bk,
		pMpt,
		&state.Deserializer{},
		&transaction.Transaction{
			HashIDField: datastore.HashIDField{
				Hash: encryption.Hash("mock transaction hash"),
			},
		},
		func(*block.Block) []string { return []string{} },
		func() *block.Block { return bk },
		func() *block.MagicBlock { return magicBlock },
		func() encryption.SignatureScheme { return signatureScheme },
	)

	_ = storagesc.SetConfig(b, balances)
	blobbers := storagesc.AddMockBlobbers(b, vi, balances)
	allocations := storagesc.AddMockAllocations(b, vi, balances, clients[1:], keys[1:])
	_ = minersc.AddMockMiners(b, vi, balances)

	return pMpt,
		balances.GetState().GetRoot(),
		clients[:vi.GetInt(smartcontract.AvailableKeys)],
		keys[:vi.GetInt(smartcontract.AvailableKeys)],
		blobbers,
		allocations
}

func AddMockkClients(
	b *testing.B,
	pMpt *util.MerklePatriciaTrie,
	vi *viper.Viper,
) ([]string, []string) {
	var sigScheme encryption.SignatureScheme = encryption.GetSignatureScheme(vi.GetString(smartcontract.SignatureScheme))
	var clientIds, publicKeys []string
	for i := 0; i < vi.GetInt(smartcontract.NumClients); i++ {
		err := sigScheme.GenerateKeys()
		require.NoError(b, err)
		publicKeyBytes, err := hex.DecodeString(sigScheme.GetPublicKey())
		require.NoError(b, err)
		clientID := encryption.Hash(publicKeyBytes)
		publicKey := sigScheme.GetPublicKey()
		clientIds = append(clientIds, clientID)
		publicKeys = append(publicKeys, publicKey)
		is := &state.State{}
		is.SetTxnHash("0000000000000000000000000000000000000000000000000000000000000000")
		is.Balance = state.Balance(vi.GetInt64(smartcontract.StartTokens))
		pMpt.Insert(util.Path(clientID), is)
	}

	return clientIds, publicKeys
}
