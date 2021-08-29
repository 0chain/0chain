package cmd

import (
	"encoding/hex"

	"0chain.net/smartcontract/benchmark"

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
)

func getBalances(
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
	vi *viper.Viper,
	dbPath string,
) (*util.MerklePatriciaTrie, util.Key, []string, []string, []string, []string) {
	pNode, err := util.NewPNodeDB(
		dbPath+"name_dataDir",
		dbPath+"name_logDir",
	)
	if err != nil {
		panic(err)
	}
	pMpt := util.NewMerklePatriciaTrie(pNode, 1, nil)

	clients, keys := AddMockkClients(pMpt, vi)

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

	_ = storagesc.SetConfig(vi, balances)
	blobbers := storagesc.AddMockBlobbers(vi, balances)
	stakePools := storagesc.GetStakePools(vi, balances)
	allocations := storagesc.AddMockAllocations(vi, balances, clients, keys, stakePools)
	storagesc.SaveStakePools(vi, stakePools, balances)
	_ = minersc.AddMockNodes(minersc.NodeTypeMiner, vi, balances)
	_ = minersc.AddMockNodes(minersc.NodeTypeSharder, vi, balances)
	return pMpt,
		balances.GetState().GetRoot(),
		clients[:vi.GetInt(benchmark.AvailableKeys)],
		keys[:vi.GetInt(benchmark.AvailableKeys)],
		blobbers,
		allocations
}

func AddMockkClients(
	pMpt *util.MerklePatriciaTrie,
	vi *viper.Viper,
) ([]string, []string) {
	var sigScheme encryption.SignatureScheme = encryption.GetSignatureScheme(vi.GetString(benchmark.SignatureScheme))
	var clientIds, publicKeys []string
	for i := 0; i < vi.GetInt(benchmark.NumClients); i++ {
		err := sigScheme.GenerateKeys()
		if err != nil {
			panic(err)
		}
		publicKeyBytes, err := hex.DecodeString(sigScheme.GetPublicKey())
		if err != nil {
			panic(err)
		}
		clientID := encryption.Hash(publicKeyBytes)
		publicKey := sigScheme.GetPublicKey()
		clientIds = append(clientIds, clientID)
		publicKeys = append(publicKeys, publicKey)
		is := &state.State{}
		is.SetTxnHash("0000000000000000000000000000000000000000000000000000000000000000")
		is.Balance = state.Balance(vi.GetInt64(benchmark.StartTokens))
		pMpt.Insert(util.Path(clientID), is)
	}

	return clientIds, publicKeys
}
