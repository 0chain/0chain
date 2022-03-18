package cmd

import (
	"encoding/hex"
	"os"
	"path"
	"sync"
	"time"

	"0chain.net/smartcontract/benchmark/main/cmd/control"
	"0chain.net/smartcontract/interestpoolsc"
	"0chain.net/smartcontract/multisigsc"
	"0chain.net/smartcontract/vestingsc"
	"0chain.net/smartcontract/zcnsc"

	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/dbs/event"

	"0chain.net/smartcontract/benchmark/main/cmd/log"

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

var BenchDataKey = encryption.Hash("benchData")

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
		data.EventDb,
	)
}

func getMpt(loadPath, configPath string) (*util.MerklePatriciaTrie, util.Key, benchmark.BenchData) {
	var mptDir string
	savePath := viper.GetString(benchmark.OptionSavePath)

	if len(savePath) > 0 {
		if loadPath != savePath {
			if err := os.MkdirAll(savePath, os.ModePerm); err != nil {
				log.Fatal("making save directory", savePath)
			}
			if err := viper.WriteConfigAs(path.Join(savePath, "benchmark.yaml")); err != nil {
				log.Fatal("cannot copy config file to", savePath)
			}
		}
		mptDir = path.Join(savePath, "mpt_db")
	} else {
		mptDir = "./mpt_db"
	}

	if len(loadPath) == 0 {
		return setUpMpt(mptDir)
	}

	return openMpt(mptDir)
}

func openMpt(loadPath string) (*util.MerklePatriciaTrie, util.Key, benchmark.BenchData) {
	pNode, err := util.NewPNodeDB(
		loadPath,
		loadPath+"log",
	)
	if err != nil {
		log.Fatal(err)
	}
	pMpt := util.NewMerklePatriciaTrie(pNode, 1, nil)

	root := viper.GetString(benchmark.MptRoot)
	_, balances := getBalances(
		&transaction.Transaction{},
		extractMpt(pMpt, util.Key(root)),
		benchmark.BenchData{},
	)

	var benchData benchmark.BenchData
	val, err := balances.GetTrieNode(BenchDataKey)
	if err != nil {
		log.Fatal(err)
	}
	err = benchData.Decode(val.Encode())
	if err != nil {
		log.Fatal(err)
	}

	return pMpt, util.Key(root), benchData
}

func setUpMpt(
	dbPath string,
) (*util.MerklePatriciaTrie, util.Key, benchmark.BenchData) {
	log.Println("starting building blockchain")
	mptGenTime := time.Now()

	pNode, err := util.NewPNodeDB(
		dbPath,
		dbPath+"log",
	)
	if err != nil {
		panic(err)
	}
	pMpt := util.NewMerklePatriciaTrie(pNode, 1, nil)
	log.Println("made empty blockchain")

	timer := time.Now()
	clients, publicKeys, privateKeys := addMockClients(pMpt)
	log.Println("added clients\t", time.Since(timer))

	timer = time.Now()
	faucetsc.FundMockFaucetSmartContract(pMpt)
	log.Println("funded faucet\t", time.Since(timer))

	timer = time.Now()
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
		nil,
	)
	log.Println("created balances\t", time.Since(timer))

	var eventDb *event.EventDb
	if viper.GetBool(benchmark.EventDbEnabled) {
		timer = time.Now()
		eventDb, err := event.NewEventDb(dbs.DbAccess{
			Enabled:         viper.GetBool(benchmark.EventDbEnabled),
			Name:            viper.GetString(benchmark.EventDbName),
			User:            viper.GetString(benchmark.EventDbUser),
			Password:        viper.GetString(benchmark.EventDbPassword),
			Host:            viper.GetString(benchmark.EventDbHost),
			Port:            viper.GetString(benchmark.EventDbPort),
			MaxIdleConns:    viper.GetInt(benchmark.EventDbMaxIdleConns),
			MaxOpenConns:    viper.GetInt(benchmark.EventDbOpenConns),
			ConnMaxLifetime: viper.GetDuration(benchmark.EventDbConnMaxLifetime),
		})
		if err != nil {
			panic(err)
		}
		if err := eventDb.AutoMigrate(); err != nil {
			panic(err)
		}
		log.Println("created event database\t", time.Since(timer))
	}

	var wg sync.WaitGroup

	var blobbers []*storagesc.StorageNode
	var validators []*storagesc.ValidationNode
	var miners, sharders []string

	wg.Add(1)
	go func() {
		defer wg.Done()
		timer = time.Now()
		_ = storagesc.SetMockConfig(balances)
		log.Println("created storage config\t", time.Since(timer))
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		timer = time.Now()
		validators = storagesc.AddMockValidators(publicKeys, balances)
		log.Println("added validators\t", time.Since(timer))
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		timer = time.Now()
		blobbers = storagesc.AddMockBlobbers(eventDb, balances)
		log.Println("added blobbers\t", time.Since(timer))
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		timer = time.Now()
		miners = minersc.AddMockNodes(clients, minersc.NodeTypeMiner, balances)
		log.Println("added miners\t", time.Since(timer))
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		timer = time.Now()
		sharders = minersc.AddMockNodes(clients, minersc.NodeTypeSharder, balances)
		log.Println("added sharders\t", time.Since(timer))
	}()

	timer = time.Now()
	stakePools := storagesc.GetMockBlobberStakePools(clients, balances)
	log.Println("created blobber stake pools\t", time.Since(timer))

	wg.Wait()
	wg.Add(1)
	go func() {
		defer wg.Done()
		timer = time.Now()
		storagesc.GetMockValidatorStakePools(clients, balances)
		log.Println("added validator stake pools\t", time.Since(timer))
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		timer = time.Now()
		storagesc.AddMockAllocations(
			clients, publicKeys, stakePools, blobbers, validators, balances,
		)
		log.Println("added allocations\t", time.Since(timer))
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		timer = time.Now()
		storagesc.SaveMockStakePools(stakePools, balances)
		log.Println("saved blobber stake pools\t", time.Since(timer))
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		timer = time.Now()
		minersc.AddNodeDelegates(clients, miners, sharders, balances)
		log.Println("adding miners and sharders delegates\t", time.Since(timer))
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		timer = time.Now()
		minersc.AddMagicBlock(miners, sharders, balances)
		log.Println("add magic block\t", time.Since(timer))
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		timer = time.Now()
		minersc.SetUpNodes(miners, sharders)
		log.Println("registering miners and sharders\t", time.Since(timer))
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		timer = time.Now()
		minersc.AddPhaseNode(balances)
		log.Println("added miners phase node\t", time.Since(timer))
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		timer = time.Now()
		storagesc.AddMockFreeStorageAssigners(clients, publicKeys, balances)
		log.Println("added free storage assigners\t", time.Since(timer))
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		timer = time.Now()
		storagesc.AddMockStats(balances)
		log.Println("added storage stats\t", time.Since(timer))
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		timer = time.Now()
		storagesc.AddMockWriteRedeems(clients, publicKeys, balances)
		log.Println("added read redeems\t", time.Since(timer))
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		timer = time.Now()
		faucetsc.AddMockGlobalNode(balances)
		log.Println("added faucet global node\t", time.Since(timer))
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		timer = time.Now()
		faucetsc.AddMockUserNodes(clients, balances)
		log.Println("added faucet user nodes\t", time.Since(timer))
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		timer = time.Now()
		interestpoolsc.AddMockNodes(clients, balances)
		log.Println("added user nodes\t", time.Since(timer))
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		timer = time.Now()
		multisigsc.AddMockWallets(clients, publicKeys, balances)
		log.Println("added client wallets\t", time.Since(timer))
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		timer = time.Now()
		vestingsc.AddVestingPools(clients, balances)
		log.Println("added vesting pools\t", time.Since(timer))
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		timer = time.Now()
		zcnsc.Setup(clients, publicKeys, balances)
		log.Println("added zcnsc\t", time.Since(timer))
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		timer = time.Now()
		control.AddControlObjects(balances)
		log.Println("added control objects\t", time.Since(timer))
	}()

	wg.Wait()

	benchData := benchmark.BenchData{
		Clients:     clients,
		PublicKeys:  publicKeys,
		PrivateKeys: privateKeys,
		Sharders:    sharders,
		EventDb:     eventDb,
	}

	if _, err := balances.InsertTrieNode(BenchDataKey, &benchData); err != nil {
		log.Fatal(err)
	}

	var bd benchmark.BenchData
	val, err := balances.GetTrieNode(BenchDataKey)
	if err != nil {
		log.Fatal(err)
	}
	err = bd.Decode(val.Encode())
	if err != nil {
		log.Fatal(err)
	}

	root := balances.GetState().GetRoot()
	viper.Set(benchmark.MptRoot, string((root)))

	log.Println("mpt generation took:", time.Since(mptGenTime), "\n")

	return pMpt, balances.GetState().GetRoot(), benchData
}

func addMockClients(
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
