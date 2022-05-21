package main

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"

	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/client"
	"0chain.net/chaincore/transaction"
	"0chain.net/chaincore/wallet"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/logging"
	"0chain.net/core/memorystore"
	"0chain.net/core/viper"
	"0chain.net/miner"
	"0chain.net/smartcontract/faucetsc"
)

var (
	wallets  []*wallet.Wallet
	maxFee   int64
	minFee   int64
	maxValue int64
	minValue int64
)

/*TransactionGenerator - generates a steady stream of transactions */
func TransactionGenerator(c *chain.Chain, workdir string) {
	wallet.SetupWallet()

	viper.SetDefault("development.txn_generation.max_txn_fee", 10000)
	maxFee = viper.GetInt64("development.txn_generation.max_txn_fee")
	viper.SetDefault("development.txn_generation.min_txn_fee", 0)
	minFee = viper.GetInt64("development.txn_generation.min_txn_fee")
	viper.SetDefault("development.txn_generation.max_txn_value", 10000000000)
	maxValue = viper.GetInt64("development.txn_generation.max_txn_value")
	viper.SetDefault("development.txn_generation.min_txn_value", 100)
	minValue = viper.GetInt64("development.txn_generation.min_txn_value")

	var (
		numClients = viper.GetInt("development.txn_generation.wallets")
		numTxns    int32
		numWorkers int
	)

	GenerateClients(c, numClients, workdir)

	viper.SetDefault("development.txn_generation.max_transactions", c.BlockSize())
	blockSize := viper.GetInt32("development.txn_generation.max_transactions")
	if blockSize <= 0 {
		return
	}

	// validate the maxFee and minFee, maxFee must > minFee, otherwise, will panic
	if maxFee-minFee <= 0 {
		logging.Logger.Panic(fmt.Sprintf("development.txn_generation.max_txn_fee must be greater than "+
			"development.txn_generation.min_txn_fee, max_fee: %v, min_fee: %v", maxFee, minFee))
	}

	switch {
	case blockSize <= 10:
		numWorkers = 1
	case blockSize <= 100:
		numWorkers = 1
	case blockSize <= 1000:
		numWorkers = 2
		//numTxns = blockSize / 2
	case blockSize <= 10000:
		numWorkers = 4
		//numTxns = blockSize / 2
	case blockSize <= 100000:
		numWorkers = 8
		//numTxns = blockSize / 2
	default:
		numWorkers = 16
	}
	txnMetadataProvider := datastore.GetEntityMetadata("txn")
	ctx := memorystore.WithEntityConnection(common.GetRootContext(), txnMetadataProvider)
	defer memorystore.Close(ctx)
	txn := txnMetadataProvider.Instance().(*transaction.Transaction)
	txn.ChainID = miner.GetMinerChain().ID
	collectionName := txn.GetCollectionName()
	sc := chain.GetServerChain()

	//Ensure the initial set of transactions succeed or become invalid
	txnCount := int32(txnMetadataProvider.GetStore().GetCollectionSize(ctx, txnMetadataProvider, collectionName))
	for txnCount > blockSize {
		time.Sleep(20 * time.Millisecond)
		txnCount = int32(txnMetadataProvider.GetStore().GetCollectionSize(ctx, txnMetadataProvider, collectionName))
	}

	numGenerators := sc.GetGeneratorsNum()
	mb := sc.GetCurrentMagicBlock()
	numMiners := mb.Miners.Size()
	var timerCount int64
	ts := rand.NewSource(time.Now().UnixNano())
	trng := rand.New(ts)
	for {
		numTxns = trng.Int31n(blockSize)
		numWorkerTxns := numTxns / int32(numWorkers)
		if numWorkerTxns*int32(numWorkers) < numTxns {
			numWorkerTxns++
		}
		blockRate := chain.SteadyStateFinalizationTimer.Rate1()
		if chain.SteadyStateFinalizationTimer.Count() < 250 && blockRate < 2 {
			blockRate = 2
		}
		totalBlocks := float64(numGenerators) * blockRate
		blocksPerMiner := totalBlocks / float64(numMiners)
		if blocksPerMiner < 1 {
			blocksPerMiner = 1
		}
		waitTime := time.Millisecond * time.Duration(1000./1.05/blocksPerMiner)
		timer := time.NewTimer(waitTime)
		if sc.GetCurrentRound()%100 == 0 {
			logging.Logger.Info("background transactions generation", zap.Duration("frequency", waitTime), zap.Float64("blocks", blocksPerMiner))
		}
		select {
		case <-ctx.Done():
			logging.Logger.Info("transaction generation", zap.Any("timer_count", timerCount))
			return
		case <-timer.C:
			timerCount++
			txnCount := int32(txnMetadataProvider.GetStore().GetCollectionSize(ctx, txnMetadataProvider, collectionName))
			if timerCount%300 == 0 {
				logging.Logger.Info("transaction generation", zap.Any("txn_count", txnCount), zap.Any("blocks_per_miner", blocksPerMiner), zap.Any("num_txns", numTxns))
			}
			if float64(txnCount) >= blocksPerMiner*float64(8*numTxns) {
				continue
			}
			wg := sync.WaitGroup{}
			for i := 0; i < numWorkers; i++ {
				ctx := datastore.WithAsyncChannel(common.GetRootContext(), transaction.TransactionEntityChannel)
				wg.Add(1)
				go func() {
					ctx = memorystore.WithEntityConnection(ctx, txnMetadataProvider)
					defer memorystore.Close(ctx)
					rs := rand.NewSource(time.Now().UnixNano())
					prng := rand.New(rs)
					var txn *transaction.Transaction
					for t := int32(0); t <= numWorkerTxns; t++ {
						r := prng.Int63n(100)
						if r < 25 {
							txn = createSendTransaction(c, prng)
						} else {
							txn = createDataTransaction(prng)
						}
						_, err := transaction.PutTransactionWithoutVerifySig(ctx, txn)
						if err != nil {
							logging.Logger.Info("transaction generator", zap.Any("error", err))
						}
					}
					wg.Done()
				}()
			}
			wg.Wait()
		}
	}
}

func createSendTransaction(c *chain.Chain, prng *rand.Rand) *transaction.Transaction {
	var wf, wt *wallet.Wallet
	csize := len(wallets)
	for {
		wf = wallets[prng.Intn(csize)]
		wt = wallets[prng.Intn(csize)]
		if wf != wt {
			break
		}
	}
	fee := prng.Int63n(maxFee-minFee) + minFee
	value := prng.Int63n(maxValue-minValue) + minValue
	txn := wf.CreateRandomSendTransaction(wt.ClientID, value, fee)
	return txn
}

func createDataTransaction(prng *rand.Rand) *transaction.Transaction {
	csize := len(wallets)
	wf := wallets[prng.Intn(csize)]
	txn := wf.CreateRandomDataTransaction(0)
	return txn
}

/*GetOwnerWallet - get the owner wallet. Used to get the initial state get going */
func GetOwnerWallet(c *chain.Chain, workdir string) *wallet.Wallet {
	var keysFile string
	if c.ClientSignatureScheme() == "ed25519" {
		keysFile = filepath.Join(workdir, "config/owner_keys.txt")
	} else {
		keysFile = filepath.Join(workdir, "config/b0owner_keys.txt")
	}
	reader, err := os.Open(keysFile)
	if err != nil {
		panic(err)
	}
	sigScheme := c.GetSignatureScheme()
	err = sigScheme.ReadKeys(reader)
	if err != nil {
		panic(err)
	}
	w := &wallet.Wallet{}
	err = w.SetSignatureScheme(sigScheme)
	if err != nil {
		panic(err)
	}
	clientMetadataProvider := datastore.GetEntityMetadata("client")
	ctx := memorystore.WithEntityConnection(common.GetRootContext(), clientMetadataProvider)
	defer memorystore.Close(ctx)
	ctx = datastore.WithAsyncChannel(ctx, client.ClientEntityChannel)
	err = w.Register(ctx)
	if err != nil {
		panic(err)
	}
	return w
}

/*GenerateClients - generate the given number of clients */
func GenerateClients(c *chain.Chain, numClients int, workdir string) {
	ownerWallet := GetOwnerWallet(c, workdir)
	rs := rand.NewSource(time.Now().UnixNano())
	prng := rand.New(rs)

	clientMetadataProvider := datastore.GetEntityMetadata("client")
	ctx := memorystore.WithEntityConnection(common.GetRootContext(), clientMetadataProvider)
	defer memorystore.Close(ctx)
	ctx = datastore.WithAsyncChannel(ctx, client.ClientEntityChannel)

	txnMetadataProvider := datastore.GetEntityMetadata("txn")
	tctx := memorystore.WithEntityConnection(common.GetRootContext(), txnMetadataProvider)
	defer memorystore.Close(tctx)
	tctx = datastore.WithAsyncChannel(ctx, transaction.TransactionEntityChannel)

	for i := 0; i < numClients; i++ {
		//client side code
		w := &wallet.Wallet{}
		if err := w.Initialize(c.ClientSignatureScheme()); err != nil {
			panic(err)
		}
		wallets = append(wallets, w)

		//Server side code bypassing REST for speed
		err := w.Register(ctx)
		if err != nil {
			panic(err)
		}
	}
	time.Sleep(1 * time.Second)
	for _, w := range wallets {
		//generous airdrop in dev/test mode :)
		txn := ownerWallet.CreateSendTransaction(w.ClientID, prng.Int63n(100)*10000000000, "generous air drop! :)", prng.Int63n(10)+1)
		_, err := transaction.PutTransactionWithoutVerifySig(tctx, txn)
		if err != nil {
			logging.Logger.Info("client generator", zap.Any("error", err))
		}
	}
	if c.ChainConfig.Faucet() {
		txn := ownerWallet.CreateSCTransaction(faucetsc.ADDRESS,
			viper.GetInt64("development.faucet.refill_amount"),
			`{"name":"refill","input":{}}`, 0)
		_, err := transaction.PutTransactionWithoutVerifySig(tctx, txn)
		if err != nil {
			logging.Logger.Info("client generator - faucet refill", zap.Any("error", err))
		}
	}
	logging.Logger.Info("generation of wallets complete", zap.Int("wallets", len(wallets)))
}
