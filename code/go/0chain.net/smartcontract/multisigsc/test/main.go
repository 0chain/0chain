// Test program for multi-sig smart contract.

package main

import (
	"flag"
	"fmt"
	"math/rand"
	"strings"
	"time"

	chainconfig "0chain.net/chaincore/config"
	mptwallet "0chain.net/chaincore/wallet"
	"0chain.net/core/encryption"
	"0chain.net/smartcontract/multisigsc"
	. "github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

type config struct {
	discoveryFile   string
	signatureScheme string
	ownerKeysFile   string
	numWallets      int
	t, n            int
}

var c config

func main() {
	discoveryFile := flag.String("discoveryfile", "config/discover_pool.yml", "Path to pool discovery file")
	signatureScheme := flag.String("signaturescheme", "bls0chain", "Name of encryption scheme")
	ownerKeysFile := flag.String("ownerkeysfile", "config/b0owner_keys.txt", "Path to chain's owner keys")
	numWallets := flag.Int("numwallets", 5, "Number of multi-sig wallets to concurrently exercise")
	t := flag.Int("t", 2, "Number of sub-keys required for token transfer")
	n := flag.Int("n", 3, "Total number of sub-keys")

	flag.Parse()

	if discoveryFile == nil || signatureScheme == nil || numWallets == nil || t == nil || n == nil {
		panic("Missing an argument.")
	}
	if !encryption.IsValidThresholdSignatureScheme(*signatureScheme) {
		panic("Invalid signature scheme.")
	}
	if *t > *n {
		panic("Required: t <= n.")
	}

	c = config{
		discoveryFile:   *discoveryFile,
		signatureScheme: *signatureScheme,
		ownerKeysFile:   *ownerKeysFile,
		numWallets:      *numWallets,
		t:               *t,
		n:               *n,
	}

	// Initialize github.com/0chain/common/core/logging
	chainconfig.Configuration().DeploymentMode = chainconfig.DeploymentDevelopment
	chainconfig.SetupDefaultConfig()
	chainconfig.SetupConfig("")
	chainconfig.SetupSmartContractConfig("")
	InitLogging("development", "")

	// Find our miners and sharders.
	discoverPoolMembers(c.discoveryFile)

	testRegistration()

	Logger.Info("")
	Logger.Info("")
	Logger.Info("")
	time.Sleep(10 * time.Second)

	if multisigsc.ExpirationTime <= 60 {
		// For this to be true, you have to change the compile time constant.
		// TODO: Put this value in the chain's config file so you don't have to
		//       do a recompile.
		testExpiration()

		Logger.Info("")
		Logger.Info("")
		Logger.Info("")
		time.Sleep(10 * time.Second)
	} else {
		Logger.Info("Multi-sig proposal expiration time is large, not testing expiration in this run")

		Logger.Info("")
		Logger.Info("")
		Logger.Info("")
	}

	testFinishProposal()

	Logger.Info("")
	Logger.Info("")
	Logger.Info("")
	time.Sleep(10 * time.Second)

	for i := 0; i < c.numWallets; i++ {
		go testStress(i)
	}

	// Don't quit.
	idle()
}

func idle() {
	for {
		time.Sleep(1 * time.Second)
	}
}

func testRegistration() {
	Logger.Info("Testing multi-sig wallet registration...")

	// Generate a group key and associated sub-keys.
	w := newTestWallet(0, c.signatureScheme, c.t, c.n)

	// Register MPT wallets for everyone in our group and give them some tokens
	// to play with.
	w.registerMPTWallets()

	// Start the real test...
	output := w.registerSCWallet()
	if !strings.HasPrefix(output, "success:") {
		Logger.Fatal("Register failed: TxnOutput should have prefix 'success:'")
	}

	Logger.Info("Finished test")
}

func testExpiration() {
	Logger.Info("Testing multi-sig proposal expiration...")

	// Generate a group key and associated sub-keys.
	w := newTestWallet(0, c.signatureScheme, c.t, c.n)

	// Register MPT wallets for everyone in our group and give them some tokens
	// to play with.
	w.registerMPTWallets()

	output := w.registerSCWallet()
	if !strings.HasPrefix(output, "success:") {
		Logger.Fatal("Register failed: TxnOutput should have prefix 'success:'")
	}

	// Start the real test...
	doExpiredProposal(w)

	Logger.Info("Finished test")
}

func doExpiredProposal(w testWallet) {
	Logger.Info("Testing proposal expiration...")

	anonWallet := newRegisteredMPTWallet()

	p := w.newProposal("expiring", anonWallet, 100)

	signer := w.signerClientIDs[0]
	expectedOutput := fmt.Sprintf("success %d:", w.t-1)

	// Create proposal.
	output := w.registerVote(p, signer)
	if !strings.HasPrefix(output, expectedOutput) {
		Logger.Fatal("Vote before expiration failed: TxnOutput should have prefix '" + expectedOutput + "'")
	}

	// Let the proposal expire.
	Logger.Info("Waiting until proposal expires...", zap.Int("seconds", multisigsc.ExpirationTime))
	time.Sleep(multisigsc.ExpirationTime * time.Second)

	// This should re-create it, which means we'll get the same output as above.
	output2 := w.registerVote(p, signer)
	if output != output2 {
		Logger.Fatal("Vote after expiration failed: Second vote should be identical because first expired", zap.String("first vote", output), zap.String("second vote", output2))
	}

	Logger.Info("Success on proposal expiration")
}

func testFinishProposal() {
	Logger.Info("Testing multi-sig transfer...")

	// Generate a group key and associated sub-keys.
	w := newTestWallet(0, c.signatureScheme, c.t, c.n)

	// Register MPT wallets for everyone in our group and give them some tokens
	// to play with.
	w.registerMPTWallets()

	output := w.registerSCWallet()
	if !strings.HasPrefix(output, "success:") {
		Logger.Fatal("Register failed: TxnOutput should have prefix 'success:'")
	}

	// Start the real test...
	doProposalWithAllN(w)
	printBalance(0, w)

	Logger.Info("Finished test")
}

func doProposalWithAllN(w testWallet) {
	anonWallet := newRegisteredMPTWallet()

	p := w.newProposal("all n of n", anonWallet, 100)

	for i, signer := range w.signerClientIDs {
		output := w.registerVote(p, signer)

		var expectedOutput string

		switch remainingVotes := w.t - (i + 1); {
		case remainingVotes > 0:
			// Still need votes.
			expectedOutput = fmt.Sprintf("success %d:", remainingVotes)
		case remainingVotes == 0:
			// Enough votes.
			expectedOutput = "success 0: transfer executed"
		default:
			// Extra unnecessary votes, i.e. more than t-of-n votes.
			expectedOutput = "success 0: proposal previously executed"
		}

		if !strings.HasPrefix(output, expectedOutput) {
			Logger.Fatal("Vote failed: TxnOutput should have prefix '" + expectedOutput + "'")
		}
	}
}

func testStress(id int) {
	Logger.Info("Stress testing multi-sig transfers...", zap.Int("worker#", id))

	// Generate a group key and associated sub-keys.
	w := newTestWallet(id, c.signatureScheme, c.t, c.n)

	// Register MPT wallets for everyone in our group and give them some tokens
	// to play with.
	w.registerMPTWallets()

	output := w.registerSCWallet()
	if !strings.HasPrefix(output, "success:") {
		Logger.Fatal("Register failed: TxnOutput should have prefix 'success:'")
	}

	// Start the real test...
	for finished := 1; ; finished++ {
		doProposalWithT(id, w, finished)
		printBalance(id, w)
	}
}

func doProposalWithT(id int, w testWallet, finished int) {
	anonWallet := newRegisteredMPTWallet()

	amount := int64(100 + rand.Intn(900))
	p := w.newProposal(fmt.Sprintf("stress%d", finished), anonWallet, amount)

	for cnt, idx := range rand.Perm(w.t) {
		signer := w.signerClientIDs[idx]

		output := w.registerVote(p, signer)

		var expectedOutput string

		switch remainingVotes := w.t - (cnt + 1); {
		case remainingVotes > 0:
			// Still need votes.
			expectedOutput = fmt.Sprintf("success %d:", remainingVotes)
		case remainingVotes == 0:
			// Enough votes.
			expectedOutput = "success 0: transfer executed"
		}

		if !strings.HasPrefix(output, expectedOutput) {
			Logger.Fatal("Vote failed: TxnOutput should have prefix '"+expectedOutput+"'", zap.Int("multi-sig wallet#", id))
		}
	}

	Logger.Info("Successful multi-sig transfer", zap.Int("multi-sig wallet#", id), zap.Int("num finished", finished))
}

func printBalance(id int, w testWallet) {
	balance := getBalance(w.groupClientID)

	Logger.Info("Multi-sig wallet balance", zap.Int("multi-sig wallet#", id), zap.Int64("balance", int64(balance)))
}

func newRegisteredMPTWallet() string {
	scheme := encryption.GetSignatureScheme(c.signatureScheme)

	err := scheme.GenerateKeys()
	if err != nil {
		Logger.Fatal("Couldn't generate key pair", zap.Error(err))
	}

	clientID := clientIDForKey(scheme)

	w := mptwallet.Wallet{
		SignatureScheme: scheme,
		PublicKey:       scheme.GetPublicKey(),
		ClientID:        clientID,
	}

	// registerMPTWallet(w)

	return clientID
}
