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

func extractMpt(mpt *util.MerklePatriciaTrie, root util.Key) *util.MerklePatriciaTrie {
	pNode := mpt.GetNodeDB()
	memNode := util.NewMemoryNodeDB()
	levelNode := util.NewLevelNodeDB(
		memNode,
		pNode,
		false,
	)
	return util.NewMerklePatriciaTrie(levelNode, 1, root)
}

func getBalances(
	txn transaction.Transaction,
	mpt *util.MerklePatriciaTrie,
) (*util.MerklePatriciaTrie, cstate.StateContextI) {
	bk := &block.Block{}
	magicBlock := &block.MagicBlock{}
	signatureScheme := &encryption.BLS0ChainScheme{}
	return mpt, cstate.NewStateContext(
		bk,
		mpt,
		&state.Deserializer{},
		&txn,
		func(*block.Block) []string { return []string{} },
		func() *block.Block { return bk },
		func() *block.MagicBlock { return magicBlock },
		func() encryption.SignatureScheme { return signatureScheme },
	)
}

func setUpMpt(
	dbPath string,
) (*util.MerklePatriciaTrie, util.Key, benchmark.BenchData) {
	pNode, err := util.NewPNodeDB(
		dbPath+"name_dataDir",
		dbPath+"name_logDir",
	)
	if err != nil {
		panic(err)
	}
	pMpt := util.NewMerklePatriciaTrie(pNode, 1, nil)

	clients, publicKeys, privateKeys := AddMockkClients(pMpt)

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

	_ = storagesc.SetConfig(balances)
	blobbers := storagesc.AddMockBlobbers(balances)
	validators := storagesc.AddMockValidators(balances)
	stakePools := storagesc.GetStakePools(clients, balances)
	allocations := storagesc.AddMockAllocations(balances, clients, publicKeys, stakePools)
	storagesc.SaveStakePools(stakePools, balances)
	_ = minersc.AddMockNodes(minersc.NodeTypeMiner, balances)
	_ = minersc.AddMockNodes(minersc.NodeTypeSharder, balances)
	storagesc.AddFreeStorageAssigners(clients, publicKeys, balances)
	storagesc.AddStats(balances)
	return pMpt, balances.GetState().GetRoot(), benchmark.BenchData{
		Clients:     clients[:viper.GetInt(benchmark.AvailableKeys)],
		PublicKeys:  publicKeys[:viper.GetInt(benchmark.AvailableKeys)],
		PrivateKeys: privateKeys[:viper.GetInt(benchmark.AvailableKeys)],
		Blobbers:    blobbers[:viper.GetInt(benchmark.AvailableKeys)],
		Validators:  validators[:viper.GetInt(benchmark.AvailableKeys)],
		Allocations: allocations[:viper.GetInt(benchmark.AvailableKeys)],
	}
}

func AddMockkClients(
	pMpt *util.MerklePatriciaTrie,
) ([]string, []string, []string) {
	//var sigScheme encryption.SignatureScheme = encryption.GetSignatureScheme(viper.GetString(benchmark.SignatureScheme))
	blsScheme := BLS0ChainScheme{}
	var clientIds, publicKeys, privateKeys []string
	for i := 0; i < viper.GetInt(benchmark.NumClients); i++ {
		err := blsScheme.GenerateKeys()
		if err != nil {
			panic(err)
		}
		publicKeyBytes, err := hex.DecodeString(blsScheme.GetPublicKey())
		if err != nil {
			panic(err)
		}
		clientID := encryption.Hash(publicKeyBytes)

		clientIds = append(clientIds, clientID)
		publicKeys = append(publicKeys, blsScheme.GetPublicKey())
		privateKeys = append(privateKeys, blsScheme.GetPrivateKey())
		is := &state.State{}
		is.SetTxnHash("0000000000000000000000000000000000000000000000000000000000000000")
		is.Balance = state.Balance(viper.GetInt64(benchmark.StartTokens))
		pMpt.Insert(util.Path(clientID), is)
	}

	return clientIds, publicKeys, privateKeys
}
