package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"0chain.net/chaincore/threshold/bls"
	"0chain.net/miner"

	_ "net/http/pprof"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/client"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/diagnostics"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/build"
	"0chain.net/core/common"
	"0chain.net/core/ememorystore"
	"0chain.net/core/logging"
	. "0chain.net/core/logging"
	"0chain.net/core/memorystore"
	"0chain.net/smartcontract/setupsc"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"0chain.net/conductor/conductrpc"
)

// start lock, where the sharder is ready to connect to blockchain (BC)
func integrationsTestsLock(id string) {
	if !viper.GetBool("testing.enabled") {
		return // regular start
	}
	println("TESTING ENABLED")
	var (
		client   = conductrpc.NewClient(viper.GetString("testing.address"))
		interval = viper.GetDuration("testing.lock_interval")
		join     bool
		err      error
	)
	for {
		join, err = client.NodeReady(conductrpc.NodeID(id))
		if err != nil {
			log.Fatal(err)
		}
		if join {
			return // can join blockchain
		}
		// otherwise, have to wait, retry after the interval
		time.Sleep(interval)
	}
}

var mpks map[bls.PartyID][]bls.PublicKey

func main() {
	deploymentMode := flag.Int("deployment_mode", 2, "deployment_mode")
	keysFile := flag.String("keys_file", "", "keys_file")
	delayFile := flag.String("delay_file", "", "delay_file")
	magicBlockFile := flag.String("magic_block_file", "", "magic_block_file")
	flag.Parse()
	config.Configuration.DeploymentMode = byte(*deploymentMode)
	config.SetupDefaultConfig()
	config.SetupConfig()
	config.SetupSmartContractConfig()

	if config.Development() {
		logging.InitLogging("development")
	} else {
		logging.InitLogging("production")
	}

	config.Configuration.ChainID = viper.GetString("server_chain.id")
	transaction.SetTxnTimeout(int64(viper.GetInt("server_chain.transaction.timeout")))

	config.SetServerChainID(config.Configuration.ChainID)

	common.SetupRootContext(node.GetNodeContext())
	ctx := common.GetRootContext()
	initEntities()
	serverChain := chain.NewChainFromConfig()
	signatureScheme := serverChain.GetSignatureScheme()

	Logger.Info("Owner keys file", zap.String("filename", *keysFile))
	reader, err := os.Open(*keysFile)
	if err != nil {
		panic(err)
	}
	err = signatureScheme.ReadKeys(reader)
	if err != nil {
		Logger.Panic("Error reading keys file")
	}
	reader.Close()

	node.Self.SetSignatureScheme(signatureScheme)

	miner.SetupMinerChain(serverChain)
	mc := miner.GetMinerChain()
	mc.SetDiscoverClients(viper.GetBool("server_chain.client.discover"))
	mc.SetGenerationTimeout(viper.GetInt("server_chain.block.generation.timeout"))
	mc.SetRetryWaitTime(viper.GetInt("server_chain.block.generation.retry_wait_time"))
	mc.SetupConfigInfoDB()
	chain.SetServerChain(serverChain)

	miner.SetNetworkRelayTime(viper.GetDuration("network.relay_time") * time.Millisecond)
	node.ReadConfig()

	var magicBlock *block.MagicBlock

	magicBlock = readMagicBlockFile(magicBlockFile, mc, serverChain)

	if state.Debug() {
		chain.SetupStateLogger("/tmp/state.txt")
	}
	gb := mc.SetupGenesisBlock(viper.GetString("server_chain.genesis_block.id"), magicBlock)
	mb := mc.GetLatestMagicBlock()
	Logger.Info("Miners in main", zap.Int("size", mb.Miners.Size()))

	if !mb.IsActiveNode(node.Self.Underlying().GetKey(), 0) {
		hostName, n2nHostName, portNum, err := readNonGenesisHostAndPort(keysFile)
		if err != nil {
			Logger.Panic("Error reading keys file. Non-genesis miner has no host or port number", zap.Error(err))
		}
		Logger.Info("Inside nonGenesis", zap.String("host_name", hostName), zap.Any("n2n_host_name", n2nHostName), zap.Int("port_num", portNum))
		node.Self.Underlying().Host = hostName
		node.Self.Underlying().N2NHost = n2nHostName
		node.Self.Underlying().Port = portNum
	}

	if node.Self.Underlying().GetKey() == "" {
		Logger.Panic("node definition for self node doesn't exist")
	}
	if node.Self.Underlying().Type != node.NodeTypeMiner {
		Logger.Panic("node not configured as miner")
	}
	err = common.NewError("saving self as client", "client save")
	for err != nil {
		_, err = client.PutClient(ctx, &node.Self.Underlying().Client)
	}
	if config.Development() {
		if *delayFile != "" {
			node.ReadNetworkDelays(*delayFile)
		}
	}

	mode := "main net"
	if config.Development() {
		mode = "development"
	} else if config.TestNet() {
		mode = "test net"
	}

	address := fmt.Sprintf(":%v", node.Self.Underlying().Port)

	Logger.Info("Starting miner", zap.String("build_tag", build.BuildTag), zap.String("go_version", runtime.Version()), zap.Int("available_cpus", runtime.NumCPU()), zap.String("port", address))
	Logger.Info("Chain info", zap.String("chain_id", config.GetServerChainID()), zap.String("mode", mode))
	Logger.Info("Self identity", zap.Any("set_index", node.Self.Underlying().SetIndex), zap.Any("id", node.Self.Underlying().GetKey()))

	integrationsTestsLock(node.Self.Underlying().GetKey())

	var server *http.Server
	if config.Development() {
		// No WriteTimeout setup to enable pprof
		server = &http.Server{
			Addr:           address,
			ReadTimeout:    30 * time.Second,
			MaxHeaderBytes: 1 << 20,
		}
	} else {
		server = &http.Server{
			Addr:           address,
			ReadTimeout:    30 * time.Second,
			WriteTimeout:   30 * time.Second,
			MaxHeaderBytes: 1 << 20,
		}
	}
	common.HandleShutdown(server)
	memorystore.GetInfo()
	initWorkers(ctx)
	common.ConfigRateLimits()
	initN2NHandlers()

	if err := mc.WaitForActiveSharders(ctx); err != nil {
		Logger.Error("failed to wait sharders", zap.Error(err))
	}
	if err := getCurrentMagicBlockFromSharders(mc); err != nil {
		Logger.Panic(err.Error())
	}
	mb = mc.GetLatestMagicBlock()
	if mb.StartingRound == 0 && mb.IsActiveNode(node.Self.Underlying().GetKey(), mb.StartingRound) {
		dkgShare := &bls.DKGSummary{
			SecretShares: make(map[string]string),
		}
		dkgShare.ID = strconv.FormatInt(mb.MagicBlockNumber, 10)
		for k, v := range mb.GetShareOrSigns().GetShares() {
			dkgShare.SecretShares[miner.ComputeBlsID(k)] = v.ShareOrSigns[node.Self.Underlying().GetKey()].Share
		}
		err = miner.StoreDKGSummary(ctx, dkgShare)
		if err != nil {
			panic(err)
		}
	}

	initHandlers()

	go func() {
		Logger.Info("Ready to listen to the requests")
		log.Fatal(server.ListenAndServe())
	}()

	mc.RegisterClient()
	chain.StartTime = time.Now().UTC()
	activeMiner := mb.Miners.HasNode(node.Self.Underlying().GetKey())
	if activeMiner {
		mb = mc.GetLatestMagicBlock()
		if err := miner.SetDKGFromMagicBlocksChainPrev(ctx, mb); err != nil {
			Logger.Error("failed to set DKG", zap.Error(err))
		} else {
			miner.StartProtocol(ctx, gb)
		}
	}
	mc.SetStarted()
	miner.SetupWorkers(ctx)

	if config.Development() {
		go TransactionGenerator(mc.Chain)
	}

	go mc.InitSetupSC()

	if config.DevConfiguration.ViewChange {
		go mc.DKGProcess(ctx)
	}

	defer done(ctx)
	<-ctx.Done()
	time.Sleep(time.Second * 5)
}

func done(ctx context.Context) {
	mc := miner.GetMinerChain()
	mc.Stop()
}

func readNonGenesisHostAndPort(keysFile *string) (string, string, int, error) {
	reader, err := os.Open(*keysFile)
	if err != nil {
		panic(err)
	}
	defer reader.Close()
	scanner := bufio.NewScanner(reader)
	scanner.Scan() //throw away the publickey
	scanner.Scan() //throw away the secretkey
	result := scanner.Scan()
	if result == false {
		return "", "", 0, errors.New("error reading Host")
	}

	h := scanner.Text()
	Logger.Info("Host inside", zap.String("host", h))

	result = scanner.Scan()
	if result == false {
		return "", "", 0, errors.New("error reading n2n host")
	}

	n2nh := scanner.Text()
	Logger.Info("N2NHost inside", zap.String("n2n_host", n2nh))

	scanner.Scan()
	po, err := strconv.ParseInt(scanner.Text(), 10, 32)
	p := int(po)
	if err != nil {
		return "", "", 0, err
	}
	return h, n2nh, p, nil

}

func readMagicBlockFile(magicBlockFile *string, mc *miner.Chain, serverChain *chain.Chain) *block.MagicBlock {
	magicBlockConfigFile := viper.GetString("network.magic_block_file")
	if magicBlockConfigFile == "" {
		magicBlockConfigFile = *magicBlockFile
	}
	if magicBlockConfigFile == "" {
		panic("Please specify --nodes_file file.txt option with a file.txt containing nodes including self")
	}
	if strings.HasSuffix(magicBlockConfigFile, "json") {
		mBfile, err := ioutil.ReadFile(magicBlockConfigFile)
		if err != nil {
			Logger.Panic(fmt.Sprintf("failed to read magic block file: %v", err))
		}
		mB := block.NewMagicBlock()
		err = mB.Decode([]byte(mBfile))
		if err != nil {
			Logger.Panic(fmt.Sprintf("failed to decode magic block file: %v", err))
		}
		mB.Hash = mB.GetHash()
		mpks = make(map[bls.PartyID][]bls.PublicKey)
		for k, v := range mB.Mpks.Mpks {
			mpks[bls.ComputeIDdkg(k)] = bls.ConvertStringToMpk(v.Mpk)
		}
		return mB
	} else {
		Logger.Panic(fmt.Sprintf("magic block file (%v) is in the wrong format. It should be a json", magicBlockConfigFile))
	}
	return nil
}

func getCurrentMagicBlockFromSharders(mc *miner.Chain) error {
	const limitAttempts = 10
	attempt := 0
	retryTimeout := time.Second * 5
	var mbs []*block.Block
	for len(mbs) == 0 {
		mbs = mc.GetLatestFinalizedMagicBlockFromSharder(common.GetRootContext())
		if len(mbs) == 0 {
			attempt++
			if attempt >= limitAttempts {
				Logger.DPanic("No finalized magic block from sharder")
			}
			Logger.Warn("get_current_mb_sharder -- retry", zap.Any("attempt", attempt),
				zap.Any("timeout", retryTimeout))
			time.Sleep(retryTimeout)
		}
	}
	if len(mbs) > 1 {
		sort.Slice(mbs, func(i, j int) bool {
			return mbs[i].StartingRound < mbs[j].StartingRound
		})
	}
	magicBlock := mbs[0]
	if err := mc.MustVerifyChainHistory(common.GetRootContext(), magicBlock, nil); err != nil {
		return err
	}
	if err := mc.UpdateMagicBlock(magicBlock.MagicBlock); err != nil {
		return fmt.Errorf("failed to update magic block: %v", err.Error())
	}
	mc.SetLatestFinalizedMagicBlock(magicBlock)
	mc.UpdateNodesFromMagicBlock(magicBlock.MagicBlock)
	return nil
}

func initEntities() {
	memoryStorage := memorystore.GetStorageProvider()

	chain.SetupEntity(memoryStorage)
	round.SetupEntity(memoryStorage)
	round.SetupVRFShareEntity(memoryStorage)
	block.SetupEntity(memoryStorage)
	block.SetupBlockSummaryEntity(memoryStorage)
	block.SetupStateChange(memoryStorage)
	state.SetupPartialState(memoryStorage)
	state.SetupStateNodes(memoryStorage)
	client.SetupEntity(memoryStorage)

	transaction.SetupTransactionDB()
	transaction.SetupEntity(memoryStorage)

	miner.SetupNotarizationEntity()
	miner.SetupStartChainEntity()

	ememoryStorage := ememorystore.GetStorageProvider()
	bls.SetupDKGEntity()
	bls.SetupDKGSummary(ememoryStorage)
	bls.SetupDKGDB()
	setupsc.SetupSmartContracts()

	block.SetupMagicBlockData(ememoryStorage)
	block.SetupMagicBlockDataDB()
}

func initHandlers() {
	SetupHandlers()
	config.SetupHandlers()
	node.SetupHandlers()
	chain.SetupHandlers()
	client.SetupHandlers()
	transaction.SetupHandlers()
	block.SetupHandlers()
	miner.SetupHandlers()
	diagnostics.SetupHandlers()
	chain.SetupStateHandlers()

	serverChain := chain.GetServerChain()
	serverChain.SetupNodeHandlers()
}

func initN2NHandlers() {
	node.SetupN2NHandlers()
	miner.SetupM2MReceivers()
	miner.SetupM2MSenders()
	miner.SetupM2SSenders()
	miner.SetupM2SRequestors()
	miner.SetupM2MRequestors()

	miner.SetupX2MResponders()
	chain.SetupX2XResponders()
	chain.SetupX2MRequestors()
	chain.SetupX2SRequestors()
}

func initWorkers(ctx context.Context) {
	serverChain := chain.GetServerChain()
	serverChain.SetupWorkers(ctx)
	//miner.SetupWorkers(ctx)
	transaction.SetupWorkers(ctx)
}
