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
)

var mpks map[bls.PartyID][]bls.PublicKey

func main() {
	deploymentMode := flag.Int("deployment_mode", 2, "deployment_mode")
	nonGenesis := flag.Bool("non_genesis", false, "non_genesis")
	keysFile := flag.String("keys_file", "", "keys_file")
	mskFile := flag.String("msk_file", "", "msk_file")
	delayFile := flag.String("delay_file", "", "delay_file")
	magicBlockFile := flag.String("magic_block_file", "", "magic_block_file")
	flag.Parse()
	genesis := !*nonGenesis
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
	var msk []string
	reader, err = os.Open(*mskFile)
	if err != nil {
		Logger.Error("start miner -- cant find msk file", zap.Any("file_name", *mskFile))
	} else {
		scanner := bufio.NewScanner(reader)
		result := scanner.Scan()
		if result == false {
			Logger.Panic("Error reading keys file")
		}
		msk = append(msk, scanner.Text())
		result = scanner.Scan()
		if result == false {
			Logger.Panic("Error reading keys file")
		}
		msk = append(msk, scanner.Text())
		reader.Close()
	}

	if !genesis {
		hostName, portNum, err := readNonGenesisHostAndPort(keysFile)
		if err != nil {
			Logger.Panic("Error reading keys file. Non-genesis miner has no host or port number", zap.Error(err))
		}
		Logger.Info("Inside nonGenesis", zap.String("hostname", hostName), zap.Int("port Num", portNum))
		node.Self.Host = "localhost"
		node.Self.N2NHost = hostName
		node.Self.Port = portNum
	}
	miner.SetupMinerChain(serverChain)
	mc := miner.GetMinerChain()
	mc.DiscoverClients = viper.GetBool("server_chain.client.discover")
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
	mc.SetupGenesisBlock(viper.GetString("server_chain.genesis_block.id"), magicBlock)
	Logger.Info("Miners in main", zap.Int("size", mc.Miners.Size()))

	if node.Self.ID == "" {
		Logger.Panic("node definition for self node doesn't exist")
	}
	if node.Self.Type != node.NodeTypeMiner {
		Logger.Panic("node not configured as miner")
	}
	err = common.NewError("saving self as client", "client save")
	for err != nil {
		_, err = client.PutClient(ctx, &node.Self.Client)
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

	address := fmt.Sprintf(":%v", node.Self.Port)

	Logger.Info("Starting miner", zap.String("build_tag", build.BuildTag), zap.String("go_version", runtime.Version()), zap.Int("available_cpus", runtime.NumCPU()), zap.String("port", address))
	Logger.Info("Chain info", zap.String("chain_id", config.GetServerChainID()), zap.String("mode", mode))
	Logger.Info("Self identity", zap.Any("set_index", node.Self.Node.SetIndex), zap.Any("id", node.Self.Node.GetKey()))

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

	getCurrentMagicBlock(mc)

	initServer()
	initHandlers()

	chain.StartTime = time.Now().UTC()
	_, activeMiner := mc.Miners.NodesMap[node.Self.ID]
	if activeMiner && magicBlock.Hash == mc.MagicBlock.Hash {
		kickoffMiner(ctx, mc, msk, mpks)
	}
	if config.Development() {
		go TransactionGenerator(mc.Chain)
	}
	Logger.Info("Ready to listen to the requests")
	log.Fatal(server.ListenAndServe())
}

func initServer() {
	/* TODO: when a new server is brought up, it needs to first download
	all the state before it can start accepting requests
	*/
	time.Sleep(time.Second)
}

func readNonGenesisHostAndPort(keysFile *string) (string, int, error) {
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
		return "", 0, errors.New("error reading Host")
	}

	h := scanner.Text()
	Logger.Info("Host inside", zap.String("host", h))
	scanner.Scan()
	po, err := strconv.ParseInt(scanner.Text(), 10, 32)
	p := int(po)
	if err != nil {
		return "", 0, err
	}
	return h, p, nil

}
func kickoffMiner(ctx context.Context, mc *miner.Chain, msk []string, mpks map[bls.PartyID][]bls.PublicKey) {
	go func() {
		miner.SetDKG(ctx, mc.MagicBlock, msk, mpks)
	}()
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

func getCurrentMagicBlock(mc *miner.Chain) {
	mbs := mc.GetLatestFinalizedMagicBlockFromSharder(common.GetRootContext())
	if len(mbs) == 0 {
		Logger.DPanic("No finalized magic block from sharder")
	}
	if len(mbs) > 1 {
		sort.Slice(mbs, func(i, j int) bool {
			return mbs[i].StartingRound < mbs[j].StartingRound
		})
	}
	magicBlock := mbs[0]
	if mc.MagicBlock.Hash != magicBlock.MagicBlock.Hash {
		havePreviousMagicBlock := false
		var pmbs []*block.Block
		for !havePreviousMagicBlock {
			pmbs = mc.GetBlockFromSharder(common.GetRootContext(), magicBlock.LatestFinalizedMagicBlockHash)
			if len(pmbs) > 0 {
				havePreviousMagicBlock = true
			}
			time.Sleep(time.Millisecond * 100)
		}
		if len(pmbs) > 1 {
			sort.Slice(pmbs, func(i, j int) bool {
				return pmbs[i].StartingRound < pmbs[j].StartingRound
			})
		}
		err := mc.UpdateMagicBlock(pmbs[0].MagicBlock)
		if err != nil {
			Logger.DPanic(fmt.Sprintf("failed to update magic block: %v", err.Error()))
		}
	}
	err := mc.UpdateMagicBlock(magicBlock.MagicBlock)
	if err != nil {
		Logger.DPanic(fmt.Sprintf("failed to update magic block: %v", err.Error()))
	}
	mc.SetLatestFinalizedMagicBlock(magicBlock)
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

	ememoryStorage := ememorystore.GetStorageProvider()
	bls.SetupDKGEntity()
	bls.SetupDKGSummary(ememoryStorage)
	bls.SetupDKGDB()
	setupsc.SetupSmartContracts()
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

	miner.SetupX2MResponders()
	chain.SetupX2XResponders()
	chain.SetupX2MRequestors()
}

func initWorkers(ctx context.Context) {
	serverChain := chain.GetServerChain()
	serverChain.SetupWorkers(ctx)
	miner.SetupWorkers(ctx)
	transaction.SetupWorkers(ctx)
}
