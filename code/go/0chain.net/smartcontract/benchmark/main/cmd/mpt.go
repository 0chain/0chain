package cmd

import (
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	"golang.org/x/net/context"

	"0chain.net/chaincore/config"
	"0chain.net/chaincore/currency"

	"0chain.net/core/common"
	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/smartcontract/zcnsc"

	"0chain.net/core/datastore"
	"0chain.net/smartcontract/benchmark/main/cmd/control"
	ebk "0chain.net/smartcontract/dbs/benchmark"
	"0chain.net/smartcontract/multisigsc"
	"0chain.net/smartcontract/vestingsc"

	"0chain.net/smartcontract/dbs/event"

	"0chain.net/smartcontract/benchmark/main/cmd/log"

	"0chain.net/smartcontract/faucetsc"

	"0chain.net/chaincore/node"

	"0chain.net/smartcontract/benchmark"

	"0chain.net/chaincore/block"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
	"0chain.net/smartcontract/minersc"
	"0chain.net/smartcontract/storagesc"
	"github.com/spf13/viper"
)

var BenchDataKey = encryption.Hash("benchData")
var executor = common.NewWithContextFunc(4)

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
	bk.CreationDate = common.Timestamp(viper.GetInt64(benchmark.MptCreationTime))
	bk.MinerID = minersc.GetMockNodeId(0, spenum.Miner)
	node.Self.Underlying().SetKey(minersc.GetMockNodeId(0, spenum.Miner))
	magicBlock := &block.MagicBlock{}
	signatureScheme := &encryption.BLS0ChainScheme{}
	return mpt, cstate.NewStateContext(
		bk,
		mpt,
		txn,
		func(*block.Block) []string { return data.Sharders },
		func() *block.Block { return bk },
		func() *block.MagicBlock { return magicBlock },
		func() encryption.SignatureScheme { return signatureScheme },
		func() *block.Block { return bk },
		data.EventDb,
	)
}

func getMpt(loadPath, configPath string, exec *common.WithContextFunc) (*util.MerklePatriciaTrie, util.Key, benchmark.BenchData) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in getMpt", r)
		}
	}()
	var mptDir string
	savePath := viper.GetString(benchmark.OptionSavePath)
	executor = exec

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
	rootBytes, err := hex.DecodeString(root)
	var eventDb *event.EventDb
	if viper.GetBool(benchmark.EventDbEnabled) {
		eventDb = openEventsDb()
	}
	if err != nil {
		panic(err)
	}

	creationDate := common.Timestamp(viper.GetInt64(benchmark.MptCreationTime))
	benchData := benchmark.BenchData{EventDb: eventDb}
	_, balances := getBalances(
		&transaction.Transaction{CreationDate: creationDate},
		extractMpt(pMpt, rootBytes),
		benchData,
	)
	benchData.Now = creationDate

	err = balances.GetTrieNode(BenchDataKey, &benchData)
	if err != nil {
		log.Fatal(err)
	}

	return pMpt, rootBytes, benchData
}

func setUpMpt(
	dbPath string,
) (*util.MerklePatriciaTrie, util.Key, benchmark.BenchData) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in setUpMpt", r)
		}
	}()

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
	clients, publicKeys, privateKeys := addMockClients(context.Background(), pMpt)
	log.Println("added clients\t", time.Since(timer))

	timer = time.Now()
	faucetsc.FundMockFaucetSmartContract(pMpt)
	log.Println("funded faucet\t", time.Since(timer))

	timer = time.Now()

	bk := &block.Block{}
	magicBlock := &block.MagicBlock{}
	signatureScheme := &encryption.BLS0ChainScheme{}

	balances := cstate.NewStateContext(
		bk,
		pMpt,
		&transaction.Transaction{
			HashIDField: datastore.HashIDField{
				Hash: encryption.Hash("mock transaction hash"),
			},
			CreationDate: common.Now(),
		},
		func(*block.Block) []string { return []string{} },
		func() *block.Block { return bk },
		func() *block.MagicBlock { return magicBlock },
		func() encryption.SignatureScheme { return signatureScheme },
		nil,
		nil,
	)

	log.Println("created balances\t", time.Since(timer))

	var eventDb *event.EventDb
	if viper.GetBool(benchmark.EventDbEnabled) {
		eventDb = createEventsDb()
	}

	var wg sync.WaitGroup
	var (
		blobbers         []*storagesc.StorageNode
		miners, sharders []string
	)

	wg.Add(1)
	go func() {
		defer wg.Done()
		timer := time.Now()
		_ = storagesc.SetMockConfig(balances)
		viper.Set(benchmark.MptCreationTime, timer.Unix())
		log.Println("created storage config\t", time.Since(timer))
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		timer := time.Now()
		blobbers = storagesc.AddMockBlobbers(eventDb, balances)
		log.Println("added blobbers\t", time.Since(timer))
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		timer := time.Now()
		_ = storagesc.AddMockValidators(publicKeys, eventDb, balances)
		log.Println("added validators\t", time.Since(timer))
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		timer := time.Now()
		miners = minersc.AddMockNodes(clients, spenum.Miner, eventDb, balances)
		log.Println("added miners\t", time.Since(timer))
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		timer := time.Now()
		sharders = minersc.AddMockNodes(clients, spenum.Sharder, eventDb, balances)
		log.Println("added sharders\t", time.Since(timer))
	}()

	wg.Wait()

	// used as foreign key
	timer = time.Now()
	ebk.AddMockUsers(clients, eventDb)
	log.Println("added mock users\t", time.Since(timer))

	// used as foreign key
	timer = time.Now()
	ebk.AddMockBlocks(miners, eventDb)
	log.Println("added mock blocks\t", time.Since(timer))

	// used as foreign key
	timer = time.Now()
	ebk.AddMockTransactions(clients, eventDb)
	log.Println("added mock transaction\t", time.Since(timer))

	// used as foreign key in readmarkers
	timer = time.Now()
	storagesc.AddMockAllocations(clients, publicKeys, eventDb, balances)
	log.Println("added allocations\t", time.Since(timer))

	timer = time.Now()
	stakePools := storagesc.GetMockBlobberStakePools(clients, eventDb, balances)
	log.Println("created blobber stake pools\t", time.Since(timer))

	wg.Add(1)
	go func() {
		defer wg.Done()
		timer := time.Now()
		storagesc.GetMockValidatorStakePools(clients, balances)
		log.Println("added validator stake pools\t", time.Since(timer))
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		timer := time.Now()
		storagesc.AddMockReadPools(clients, balances)
		log.Println("added allocation read pools\t", time.Since(timer))
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		timer := time.Now()
		storagesc.AddMockChallengePools(balances)
		log.Println("added challenge pools\t", time.Since(timer))
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		timer := time.Now()
		storagesc.AddMockChallenges(blobbers, eventDb, balances)
		log.Println("added challenges\t", time.Since(timer))
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		timer := time.Now()
		storagesc.SaveMockStakePools(stakePools, balances)
		log.Println("saved blobber stake pools\t", time.Since(timer))
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		timer := time.Now()
		minersc.AddNodeDelegates(clients, miners, sharders, balances)
		log.Println("adding miners and sharders delegates\t", time.Since(timer))
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		timer := time.Now()
		minersc.AddMagicBlock(miners, sharders, balances)
		log.Println("add magic block\t", time.Since(timer))
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		timer := time.Now()
		minersc.SetUpNodes(miners, sharders)
		log.Println("registering miners and sharders\t", time.Since(timer))
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		timer := time.Now()
		minersc.AddPhaseNode(balances)
		log.Println("added miners phase node\t", time.Since(timer))
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		timer := time.Now()
		storagesc.AddMockFreeStorageAssigners(clients, publicKeys, balances)
		log.Println("added free storage assigners\t", time.Since(timer))
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		timer := time.Now()
		storagesc.AddMockWriteRedeems(clients, publicKeys, eventDb, balances)
		log.Println("added read redeems\t", time.Since(timer))
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		timer := time.Now()
		faucetsc.AddMockGlobalNode(balances)
		log.Println("added faucet global node\t", time.Since(timer))
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		timer := time.Now()
		faucetsc.AddMockUserNodes(clients, balances)
		log.Println("added faucet user nodes\t", time.Since(timer))
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		timer := time.Now()
		multisigsc.AddMockWallets(clients, publicKeys, balances)
		log.Println("added client wallets\t", time.Since(timer))
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		timer := time.Now()
		vestingsc.AddMockClientPools(clients, balances)
		log.Println("added vesting client pools\t", time.Since(timer))
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		timer := time.Now()
		vestingsc.AddMockVestingPools(clients, balances)
		vestingsc.AddMockConfig(balances)
		log.Println("added vesting pools\t", time.Since(timer))
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		timer := time.Now()
		zcnsc.Setup(eventDb, clients, publicKeys, balances)
		log.Println("added zcnsc\t", time.Since(timer))
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		timer := time.Now()
		control.AddControlObjects(balances)
		log.Println("added control objects\t", time.Since(timer))
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		timer = time.Now()
		ebk.AddMockEvents(eventDb)
		log.Println("added mock events\t", time.Since(timer))
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		timer = time.Now()
		ebk.AddMockErrors(eventDb)
		log.Println("added mock errors\t", time.Since(timer))
	}()

	var benchData benchmark.BenchData
	wg.Add(1)
	go func() {
		defer wg.Done()
		benchData.EventDb = eventDb
		benchData.Clients = clients
		benchData.PublicKeys = publicKeys
		benchData.PrivateKeys = privateKeys
		benchData.Sharders = sharders
		benchData.Now = common.Now()

		if _, err := balances.InsertTrieNode(BenchDataKey, &benchData); err != nil {
			log.Fatal(err)
		}
	}()

	wg.Wait()

	timer = time.Now()
	root := balances.GetState().GetRoot()
	hexBytes := make([]byte, hex.EncodedLen(len(root)))
	hex.Encode(hexBytes, root)
	viper.Set(benchmark.MptRoot, string(hexBytes))
	log.Println("saved simulation parameters\t", time.Since(timer))
	log.Println("mpt generation took:", time.Since(mptGenTime))

	return pMpt, balances.GetState().GetRoot(), benchData
}

func openEventsDb() *event.EventDb {
	timer := time.Now()
	eventDb := newEventsDb()
	log.Println("opened event database\t", time.Since(timer))
	return eventDb
}

func createEventsDb() *event.EventDb {
	timer := time.Now()
	eventDb := newEventsDb()
	err := eventDb.Drop()
	if err != nil {
		log.Fatal(err)
	}

	if err := eventDb.AutoMigrate(); err != nil {
		log.Fatal(err)
	}
	log.Println("created event database\t", time.Since(timer))
	return eventDb
}

func newEventsDb() *event.EventDb {
	timer := time.Now()
	var eventDb *event.EventDb
	tick := func() (*event.EventDb, error) {
		return event.NewEventDb(config.DbAccess{
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

	}

	t := time.NewTicker(time.Second)
	var err error
	eventDb, err = tick()
	if err != nil {
		for {
			<-t.C
			eventDb, err = tick()
			if err == nil {
				break
			} else {
				log.Println("no connection to eventDB yet: " + err.Error())
			}
		}

	}
	if err != nil {
		log.Fatal(err)
	}
	log.Println("created event database\t", time.Since(timer))
	return eventDb
}

func addMockClients(ctx context.Context,
	pMpt *util.MerklePatriciaTrie,
) ([]string, []string, []string) {
	var clientIds, publicKeys, privateKeys []string
	activeClients := viper.GetInt(benchmark.NumActiveClients)
	for i := 0; i < viper.GetInt(benchmark.NumClients); i++ {
		err := executor.Run(ctx, func(i int) func() error {
			return func() error {
				blsScheme := BLS0ChainScheme{}
				err := blsScheme.GenerateKeys()
				if err != nil {
					return err
				}
				publicKeyBytes, err := hex.DecodeString(blsScheme.GetPublicKey())
				if err != nil {
					return err
				}
				clientID := encryption.Hash(publicKeyBytes)

				if i < activeClients {
					clientIds = append(clientIds, clientID)
					publicKeys = append(publicKeys, blsScheme.GetPublicKey())
					privateKeys = append(privateKeys, blsScheme.GetPrivateKey())
				}
				is := &state.State{}
				_ = is.SetTxnHash("0000000000000000000000000000000000000000000000000000000000000000")
				is.Balance = currency.Coin(viper.GetInt64(benchmark.StartTokens))
				_, err = pMpt.Insert(util.Path(clientID), is)
				if err != nil {
					return err
				}
				return nil
			}
		}(i))
		if err != nil {
			panic(err)
		}
	}

	return clientIds, publicKeys, privateKeys
}
