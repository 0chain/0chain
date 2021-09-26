package cmd

import (
	"encoding/hex"

	"0chain.net/smartcontract/benchmark/main/cmd/log"

	"0chain.net/smartcontract/multisigsc"

	"0chain.net/smartcontract/vestingsc"

	"0chain.net/smartcontract/interestpoolsc"

	"0chain.net/smartcontract/faucetsc"

	"0chain.net/chaincore/node"

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
	txn *transaction.Transaction,
	mpt *util.MerklePatriciaTrie,
	data benchmark.BenchData,
) (*util.MerklePatriciaTrie, cstate.StateContextI) {
	bk := &block.Block{
		MagicBlock: &block.MagicBlock{
			StartingRound: 0,
		},
		PrevBlock: &block.Block{},
	}
	bk.Round = 2
	bk.MinerID = minersc.GetMockNodeId(0, minersc.NodeTypeMiner)
	node.Self.Underlying().SetKey(minersc.GetMockNodeId(0, minersc.NodeTypeMiner))
	magicBlock := &block.MagicBlock{}
	signatureScheme := &encryption.BLS0ChainScheme{}
	return mpt, cstate.NewStateContext(
		bk,
		mpt,
		&state.Deserializer{},
		txn,
		func(*block.Block) []string { return data.Sharders },
		func() *block.Block { return bk },
		func() *block.MagicBlock { return magicBlock },
		func() encryption.SignatureScheme { return signatureScheme },
	)
}

func setUpMpt(
	dbPath string,
) (*util.MerklePatriciaTrie, util.Key, benchmark.BenchData) {

	log.Println("starting building blockchain")

	pNode, err := util.NewPNodeDB(
		dbPath+"name_dataDir",
		dbPath+"name_logDir",
	)
	if err != nil {
		panic(err)
	}
	pMpt := util.NewMerklePatriciaTrie(pNode, 1, nil)
	log.Println("made empty blockchain")
	clients, publicKeys, privateKeys := addMockkClients(pMpt)
	log.Println("added clients")
	faucetsc.FundMockFaucetSmartContract(pMpt)
	log.Println("funded faucet")
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

	log.Println("created balances")
	_ = storagesc.SetMockConfig(balances)
	log.Println("created storage config")
	validators := storagesc.AddMockValidators(balances)
	log.Println("added validators")
	blobbers := storagesc.AddMockBlobbers(balances)
	log.Println("added blobbers")
	stakePools := storagesc.GetMockStakePools(clients, balances)
	log.Println("added stake pools")
	storagesc.AddMockAllocations(
		clients, publicKeys, stakePools, blobbers, validators, balances,
	)
	log.Println("added allocations")
	storagesc.SaveMockStakePools(stakePools, balances)
	log.Println("added stake pools")
	miners := minersc.AddMockNodes(clients, minersc.NodeTypeMiner, balances)
	log.Println("added miners")
	sharders := minersc.AddMockNodes(clients, minersc.NodeTypeSharder, balances)
	log.Println("added sharders")
	minersc.AddNodeDelegates(clients, miners, sharders, balances)
	log.Println("adding miners and sharders delegates")
	minersc.AddMagicBlock(miners, sharders, balances)
	log.Println("add magic block")
	minersc.SetUpNodes(miners, sharders)
	log.Println("registering miners and sharders")
	storagesc.AddMockFreeStorageAssigners(clients, publicKeys, balances)
	log.Println("added free storage assigners")
	storagesc.AddMockStats(balances)
	log.Println("added storage stats")
	storagesc.AddMockWriteRedeems(clients, publicKeys, balances)
	log.Println("added read redeems")
	faucetsc.AddMockGlobalNode(balances)
	log.Println("added faucet global node")
	faucetsc.AddMockUserNodes(clients, balances)
	log.Println("added faucet user nodes")
	interestpoolsc.AddMockNodes(clients, balances)
	log.Println("added user nodes")
	multisigsc.AddMockWallets(clients, publicKeys, balances)
	log.Println("added client wallets")
	vestingsc.AddVestingPools(clients, balances)
	log.Println("added vesting pools")
	minersc.AddPhaseNode(balances)

	return pMpt, pMpt.GetRoot(), benchmark.BenchData{
		Clients:     clients,
		PublicKeys:  publicKeys,
		PrivateKeys: privateKeys,
		Sharders:    sharders,
	}
}

func addMockkClients(
	pMpt *util.MerklePatriciaTrie,
) ([]string, []string, []string) {
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
		_ = is.SetTxnHash("0000000000000000000000000000000000000000000000000000000000000000")
		is.Balance = state.Balance(viper.GetInt64(benchmark.StartTokens))
		_, err = pMpt.Insert(util.Path(clientID), is)
		if err != nil {
			panic(err)
		}
	}

	return clientIds, publicKeys, privateKeys
}
