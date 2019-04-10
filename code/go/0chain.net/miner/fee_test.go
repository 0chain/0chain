package miner

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/transaction"
	"0chain.net/chaincore/wallet"
	"0chain.net/core/common"
)

const (
	ed  = "ed25519"
	bls = "bls0chain"
)

var wallets []*wallet.Wallet

func initializeWallets(walletSize int) {
	signScheme := ed
	for i := 0; i < walletSize; i++ {
		w := &wallet.Wallet{}
		w.Initialize(signScheme)
		wallets = append(wallets, w)
	}
}

func createTransactions(numTxns int) {
	rs := rand.NewSource(time.Now().UnixNano())
	prng := rand.New(rs)
	for i := 0; i < numTxns; i++ {
		txn := createTransaction(prng)
		ctx := common.GetRootContext()
		transaction.PutTransaction(txn)
	}
}

func createTransaction(prng *rand.Rand) *transaction.Transaction {
	var wf, wt *wallet.Wallet
	csize := len(wallets)
	for true {
		wf = wallets[prng.Intn(csize)]
		wt = wallets[prng.Intn(csize)]
		if wf != wt {
			break
		}
	}
	txn := wf.CreateRandomSendTransaction(wt.ClientID, 10000000)
	return txn
}

func TestBlockGenerate(t *testing.T) {
	initializeWallets(50)
	createTransactions(1000)
	c, ok := chain.Provider().(chain.Chain)
	if !ok {
		t.Error("chain.Chain failed to be miner.Chain")
	}
	r := round.NewRound(1)
	c.Initialize()
	SetupMinerChain(c)
	b, err := c.GenerateRoundBlock(ctx, r)
	// (mc *Chain) GenerateRoundBlock(ctx context.Context, r *Round) (*block.Block, error)
}
