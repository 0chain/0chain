package miner

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/user"
	"strconv"
	"testing"
	"time"

	"0chain.net/chaincore/state"
	"0chain.net/smartcontract/setupsc"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/client"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/memorystore"
	"0chain.net/core/viper"
	"0chain.net/sharder/blockstore"
	"github.com/0chain/common/core/logging"
	"github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/require"

	"github.com/alicebob/miniredis/v2"
)

var numOfTransactions int

func init() {
	flag.IntVar(&numOfTransactions, "num_txns", 4000, "number of transactions per block")

	logging.InitLogging("testing", "")
}

func getContext() (context.Context, func()) {
	ctx := common.GetRootContext()
	ctx = memorystore.WithConnection(ctx)
	return ctx, func() {
		memorystore.Close(ctx)
	}
}

func generateSingleBlock(ctx context.Context, mc *Chain, prevBlock *block.Block, r round.RoundI) (*block.Block, error) {
	b := block.Provider().(*block.Block)
	if prevBlock == nil {
		gb := SetupGenesisBlock()
		prevBlock = gb
		mc.AddGenesisBlock(gb)
	}
	b.ChainID = prevBlock.ChainID
	data := &chain.ConfigData{BlockSize: int32(numOfTransactions)}
	if mc.ChainConfig != nil {
		chain.UpdateConfigImpl(mc.ChainConfig.(*chain.ConfigImpl), data)
	} else {
		mc.ChainConfig = chain.NewConfigImpl(data)
	}

	usr, err := user.Current()
	if err != nil {
		panic(err)
	}

	mClient, err := makeTestMinioClient()
	if err != nil {
		return nil, err
	}
	blockstore.SetupStore(blockstore.NewFSBlockStore(fmt.Sprintf("%v%s.0chain.net",
		usr.HomeDir, string(os.PathSeparator)), mClient))

	var rd *Round
	switch rr := r.(type) {
	case *MockRound:
		rd = rr.Round
	case *Round:
		rd = rr
	default:
		log.Fatalf("unknow round type:%v", rr)
	}

	b, err = mc.GenerateRoundBlock(ctx, rd)
	if err != nil {
		return nil, err
	}
	b.ComputeProperties()
	blockstore.GetStore().Write(b)
	if mkr, ok := r.(*MockRound); ok {
		mkr.HeaviestNotarizedBlock = b
	}
	return b, nil
}

type MockRound struct {
	*Round
	HeaviestNotarizedBlock *block.Block
}

func (mr *MockRound) GetHeaviestNotarizedBlock() *block.Block {
	return mr.HeaviestNotarizedBlock
}

func CreateRound(number int64) *Round {
	mc := GetMinerChain()
	r := round.NewRound(number)
	mr := mc.CreateRound(r)
	mc.AddRound(mr)
	mc.SetCurrentRound(mr.Number)
	return mr
}

func CreateMockRound(number int64) *MockRound {
	mc := GetMinerChain()
	r := round.NewRound(number)
	mr := &MockRound{
		Round: mc.CreateRound(r),
	}
	mc.AddRound(mr)
	mc.SetCurrentRound(mr.Number)
	return mr
}

func makeTestMinioClient() (blockstore.MinioClient, error) {
	//todo: replace play.min.io with local service
	mConf := blockstore.MinioConfiguration{
		StorageServiceURL: "play.min.io",
		AccessKeyID:       "Q3AM3UQ867SPQQA43P2F",
		SecretAccessKey:   "zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG",
		BucketName:        "mytestbucket",
		BucketLocation:    "us-east-1",
		DeleteLocal:       false,
		Secure:            false,
	}

	return blockstore.CreateMinioClientFromConfig(mConf)
}

func setupMinerChain() (*Chain, func()) {
	mc := GetMinerChain()
	if mc.Chain == nil {
		mc.Chain = chain.Provider().(*chain.Chain)
	}

	mc.ChainConfig = chain.NewConfigImpl(&chain.ConfigData{GeneratorsPercent: 33, MinGenerators: 1})
	doneC := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		mc.StartLFMBWorker(ctx)
		close(doneC)
	}()
	return mc, func() {
		cancel()
		<-doneC
	}
}

func TestBlockGeneration(t *testing.T) {
	clean := SetUpSingleSelf()
	defer clean()
	ctx := common.GetRootContext()
	ctx = memorystore.WithConnection(ctx)
	defer memorystore.Close(ctx)

	mc, stopAndClean := setupMinerChain()
	defer stopAndClean()

	config.SetupSmartContractConfig("")

	gb := SetupGenesisBlock()
	mc.AddGenesisBlock(gb)

	b := block.Provider().(*block.Block)
	b.ChainID = datastore.ToKey(config.GetServerChainID())
	usr, err := user.Current()
	if err != nil {
		panic(err)
	}

	mClient, err := makeTestMinioClient()
	if err != nil {
		t.Fatal(err)
	}
	blockstore.SetupStore(blockstore.NewFSBlockStore(fmt.Sprintf("%v%s.0chain.net",
		usr.HomeDir, string(os.PathSeparator)), mClient))

	r := CreateRound(1)
	r.RandomSeed = time.Now().UnixNano()

	b, err = mc.GenerateRoundBlock(ctx, r)
	if err != nil {
		t.Errorf("Error generating block: %v\n", err)
		return
	}

	err = blockstore.Store.Write(b)
	require.NoError(t, err)

	_, err = blockstore.Store.Read(b.Hash, r.Number)
	require.NoError(t, err)

	common.Done()
}

func TestBlockVerification(t *testing.T) {
	clean := SetUpSingleSelf()
	defer clean()
	mc, stopAndClean := setupMinerChain()

	defer stopAndClean()
	ctx, clean := getContext()
	defer clean()
	mb := mc.GetMagicBlock(0)
	mr := CreateRound(1)
	nano := int64(16408760407010)
	mr.SetRandomSeed(nano, len(mb.Miners.Nodes))

	b, err := generateSingleBlock(ctx, mc, nil, mr)
	if err != nil {
		t.Errorf("Block generation failed")
	}

	if b != nil {
		_, err = mc.VerifyRoundBlock(ctx, mr, b)
	}
	if err != nil {
		t.Errorf("Block failed verification because %v", err.Error())
	}
	common.Done()
}

func TestTwoCorrectBlocks(t *testing.T) {
	viper.Set("server_chain.smart_contract.faucet", true)
	viper.Set("server_chain.smart_contract.storage", true)
	viper.Set("server_chain.smart_contract.zcn", true)
	viper.Set("server_chain.smart_contract.multisig", true)
	viper.Set("server_chain.smart_contract.miner", true)
	viper.Set("server_chain.smart_contract.vesting", true)
	setupsc.SetupSmartContracts()

	cleanSS := SetUpSingleSelf()
	defer cleanSS()
	ctx := context.Background()
	mr := CreateMockRound(1)
	mr.RandomSeed = time.Now().UnixNano()
	mc, stopAndClean := setupMinerChain()
	defer stopAndClean()
	b0, err := generateSingleBlock(ctx, mc, nil, mr)
	require.NoError(t, err)

	rd := mc.GetRound(0)
	require.NotNil(t, rd)
	if b0 != nil {
		var b1 *block.Block
		mb := mc.GetMagicBlock(0)
		mr2 := CreateMockRound(1)
		mr2.SetRandomSeed(int64(16408760407010), len(mb.Miners.Nodes))
		b1, err = generateSingleBlock(ctx, mc, b0, mr2.Round)
		require.NoError(t, err)
		_, err = mc.VerifyRoundBlock(ctx, mr2, b1)
	}
	if err != nil {
		t.Errorf("Block failed verification because %v", err.Error())
	}
	common.Done()
}

func TestTwoBlocksWrongRound(t *testing.T) {
	cleanSS := SetUpSingleSelf()
	defer cleanSS()
	ctx, clean := getContext()
	defer clean()
	mr := CreateRound(1)
	mr.RandomSeed = time.Now().UnixNano()
	mc, stopAndClean := setupMinerChain()
	defer stopAndClean()
	b0, err := generateSingleBlock(ctx, mc, nil, mr)
	//mc := GetMinerChain()
	if b0 != nil {
		//var b1 *block.Block
		mr3 := CreateRound(3)
		_, err = generateSingleBlock(ctx, mc, b0, mr3)
		//_, err = mc.VerifyRoundBlock(ctx, b1)
	}
	if err == nil {
		t.Error("Second block generated")
	}
	common.Done()
}

func TestBlockVerificationBadHash(t *testing.T) {
	cleanSS := SetUpSingleSelf()
	defer cleanSS()
	ctx, clean := getContext()
	defer clean()
	mr := CreateRound(1)
	mr.RandomSeed = time.Now().UnixNano()
	mc, stopAndClean := setupMinerChain()
	defer stopAndClean()

	b, err := generateSingleBlock(ctx, mc, nil, mr)
	if b != nil {
		b.Hash = "bad hash"
		_, err = mc.VerifyRoundBlock(ctx, mr, b)
	}
	if err == nil {
		t.Error("FAIL: Block with bad hash passed verification")
	}
	common.Done()
}

func BenchmarkGenerateALotTransactions(b *testing.B) {
	cleanSS := SetUpSingleSelf()
	defer cleanSS()

	ctx, clean := getContext()
	defer clean()

	mr := CreateRound(1)
	mr.RandomSeed = time.Now().UnixNano()
	mc, stopAndClean := setupMinerChain()
	defer stopAndClean()
	block, _ := generateSingleBlock(ctx, mc, nil, mr)
	if block != nil {
		b.Logf("Created block with %v transactions", len(block.Txns))
	} else {
		b.Error("Failed to even generate a block... OUCH!")
	}
}

func BenchmarkGenerateAndVerifyALotTransactions(b *testing.B) {
	cleanSS := SetUpSingleSelf()
	defer cleanSS()
	ctx, clean := getContext()
	defer clean()
	mr := CreateRound(1)
	mc, stopAndClean := setupMinerChain()
	defer stopAndClean()
	block, err := generateSingleBlock(ctx, mc, nil, mr)
	if block != nil && err == nil {
		_, err = mc.VerifyRoundBlock(ctx, mr, block)
		if err != nil {
			b.Errorf("Block with %v transactions failed verication", len(block.Txns))
		} else {
			b.Logf("Created block with %v transactions", len(block.Txns))
		}
	} else {
		b.Error("Failed to even generate a block... OUCH!")
	}
}

func setupTempRocksDBDir() func() {
	if err := os.MkdirAll("data/rocksdb/state", 0766); err != nil {
		panic(err)
	}

	return func() {
		if err := os.RemoveAll("data/rocksdb/state"); err != nil {
			panic(err)
		}

		if err := os.RemoveAll("log"); err != nil {
			panic(err)
		}
	}
}

func setupSelfNodeKeys() { //nolint
	keys := "e065fc02aaf7aaafaebe5d2dedb9c7c1d63517534644434b813cb3bdab0f94a0\naa3e1ae2290987959dc44e43d138c81f15f93b2d56d7a06c51465f345df1a8a6e065fc02aaf7aaafaebe5d2dedb9c7c1d63517534644434b813cb3bdab0f94a0"
	breader := bytes.NewBuffer([]byte(keys))
	sigScheme := encryption.NewED25519Scheme()
	sigScheme.ReadKeys(breader)
	node.Self.SetSignatureScheme(sigScheme)
}

func SetupGenesisBlock() *block.Block {
	mc := GetMinerChain()
	data := &chain.ConfigData{BlockSize: int32(numOfTransactions)}
	if mc.ChainConfig != nil {
		chain.UpdateConfigImpl(mc.ChainConfig.(*chain.ConfigImpl), data)
	} else {
		mc.ChainConfig = chain.NewConfigImpl(data)
	}

	mb := mc.GetMagicBlock(0)
	if mb == nil {
		mb = block.NewMagicBlock()
		mp := node.NewPool(node.NodeTypeMiner)
		mb.Miners = mp
		sp := node.NewPool(node.NodeTypeSharder)
		mb.Sharders = sp
		mc.SetMagicBlock(mb)
	}

	gr, gb := mc.GenerateGenesisBlock("ed79cae70d439c11258236da1dfa6fc550f7cc569768304623e8fbd7d70efae4", mb, state.NewInitStates())
	mr := mc.CreateRound(gr.(*round.Round))
	mc.AddRoundBlock(gr, gb)
	mc.AddRound(mr)
	return gb
}

func SetUpSingleSelf() func() {
	// create rocksdb state dir
	clean := setupTempRocksDBDir()
	s, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	p, err := strconv.Atoi(s.Port())
	if err != nil {
		panic(err)
	}
	memorystore.InitDefaultPool(s.Host(), p)

	memorystore.AddPool("txndb", &redis.Pool{
		MaxIdle:   80,
		MaxActive: 1000, // max number of connections
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", s.Addr())
			if err != nil {
				panic(err.Error())
			}
			return c, err
		},
	})

	memorystore.AddPool("clientdb", &redis.Pool{
		MaxIdle:   80,
		MaxActive: 1000, // max number of connections
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", s.Addr())
			if err != nil {
				panic(err.Error())
			}
			return c, err
		},
	})

	m := make(map[datastore.Key]encryption.SignatureScheme)
	n1 := &node.Node{Type: node.NodeTypeMiner, Host: "", Port: 7071, Status: node.NodeStatusActive}
	s1 := encryption.NewED25519Scheme()
	s1.GenerateKeys()
	n1.SetSignatureScheme(s1)
	m[n1.Client.ID] = s1
	//n1.ID = "24e23c52e2e40689fdb700180cd68ac083a42ed292d90cc021119adaa4d21509"
	n2 := &node.Node{Type: node.NodeTypeMiner, Host: "", Port: 7072, Status: node.NodeStatusActive}
	n2.ID = "2"
	s2 := encryption.NewED25519Scheme()
	s2.GenerateKeys()
	n2.SetSignatureScheme(s2)
	m[n2.Client.ID] = s2
	//n2.ID = "5fbb6924c222e96df6c491dfc4a542e1bbfc75d821bcca992544899d62121b55"
	n3 := &node.Node{Type: node.NodeTypeMiner, Host: "", Port: 7073, Status: node.NodeStatusActive}
	n3.ID = "3"
	s3 := encryption.NewED25519Scheme()
	s3.GenerateKeys()
	n3.SetSignatureScheme(s3)
	m[n3.Client.ID] = s3
	//n3.ID = "103c274502661e78a2b5c470057e57699e372a4382a4b96b29c1bec993b1d19c"

	np := node.NewPool(node.NodeTypeMiner)
	np.AddNode(n1)
	np.AddNode(n2)
	np.AddNode(n3)

	node.Self = &node.SelfNode{}
	node.Self.Node = np.Nodes[0]
	node.Self.SetSignatureScheme(m[node.Self.Node.ID])

	//setupSelfNodeKeys()

	mb := block.NewMagicBlock()
	mb.Miners = np

	sp := node.NewPool(node.NodeTypeSharder)
	mb.Sharders = sp

	common.SetupRootContext(node.GetNodeContext())
	config.SetServerChainID(config.GetMainChainID())
	transaction.SetupEntity(memorystore.GetStorageProvider())

	block.SetupEntity(memorystore.GetStorageProvider())
	block.SetupBlockSummaryEntity(memorystore.GetStorageProvider())
	client.SetupEntity(memorystore.GetStorageProvider())

	chain.SetupEntity(memorystore.GetStorageProvider(), "")
	round.SetupEntity(memorystore.GetStorageProvider())

	c := chain.Provider().(*chain.Chain)
	c.ID = datastore.ToKey(config.GetServerChainID())
	c.SetMagicBlock(mb)
	data := &chain.ConfigData{BlockSize: 1024}
	c.ChainConfig = chain.NewConfigImpl(data)
	data.BlockSize = int32(numOfTransactions)

	data.MinGenerators = 1
	data.RoundRange = 10000000
	data.MinBlockSize = 1
	data.MaxByteSize = 1638400

	c.SetGenerationTimeout(15)
	chain.SetServerChain(c)
	SetupMinerChain(c)
	mc := GetMinerChain()
	mc.SetMagicBlock(mb)
	SetupM2MSenders()
	return func() {
		chain.CloseStateDB()
		clean()
		s.Close()
	}
}

func setupSelf() func() { //nolint
	clean := setupTempRocksDBDir()
	s, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	p, err := strconv.Atoi(s.Port())
	if err != nil {
		panic(err)
	}
	memorystore.InitDefaultPool(s.Host(), p)

	memorystore.AddPool("txndb", &redis.Pool{
		MaxIdle:   80,
		MaxActive: 1000, // max number of connections
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", s.Addr())
			if err != nil {
				panic(err.Error())
			}
			return c, err
		},
	})

	n1 := &node.Node{Type: node.NodeTypeMiner, Host: "", Port: 7071, Status: node.NodeStatusActive}
	n1.ID = "24e23c52e2e40689fdb700180cd68ac083a42ed292d90cc021119adaa4d21509"
	n2 := &node.Node{Type: node.NodeTypeMiner, Host: "", Port: 7072, Status: node.NodeStatusActive}
	n2.ID = "5fbb6924c222e96df6c491dfc4a542e1bbfc75d821bcca992544899d62121b55"
	n3 := &node.Node{Type: node.NodeTypeMiner, Host: "", Port: 7073, Status: node.NodeStatusActive}
	n3.ID = "103c274502661e78a2b5c470057e57699e372a4382a4b96b29c1bec993b1d19c"

	node.Self = &node.SelfNode{}
	node.Self.Node = n1

	setupSelfNodeKeys()

	np := node.NewPool(node.NodeTypeMiner)
	np.AddNode(n1)
	np.AddNode(n2)
	np.AddNode(n3)

	common.SetupRootContext(node.GetNodeContext())
	config.SetServerChainID(config.GetMainChainID())
	transaction.SetupEntity(memorystore.GetStorageProvider())

	block.SetupEntity(memorystore.GetStorageProvider())
	client.SetupEntity(memorystore.GetStorageProvider())
	chain.SetupEntity(memorystore.GetStorageProvider(), "")
	round.SetupEntity(memorystore.GetStorageProvider())

	mb := block.NewMagicBlock()
	mb.Miners = np

	c := chain.Provider().(*chain.Chain)
	c.ID = datastore.ToKey(config.GetServerChainID())
	c.SetMagicBlock(mb)
	chain.SetServerChain(c)
	SetupMinerChain(c)
	mc := GetMinerChain()
	mc.SetMagicBlock(mb)
	SetupM2MSenders()

	return func() {
		chain.CloseStateDB()
		clean()
		s.Close()
	}
}
