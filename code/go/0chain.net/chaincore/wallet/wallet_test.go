package wallet

import (
	"0chain.net/core/logging"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"

	"0chain.net/chaincore/node"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/encryption"
	"0chain.net/core/memorystore"

	"0chain.net/core/util"
)

var debug = false
var randTime = time.Now().UnixNano()
var deletePercent = 0

var prng *rand.Rand

const (
	PERSIST = 1
	MEMORY  = 2
	LEVEL   = 3
)

func init() {
	logging.InitLogging("development")
	var rs = rand.NewSource(randTime)
	prng = rand.New(rs)
}

var clientSignatureScheme = "bls0chain"

func TestWalletSetup(t *testing.T) {
	sigScheme := encryption.GetSignatureScheme(clientSignatureScheme)
	err := sigScheme.GenerateKeys()
	if err != nil {
		panic(err)
	}
	sigScheme.WriteKeys(os.Stdout)
	publicKeyBytes, err := hex.DecodeString(sigScheme.GetPublicKey())
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(os.Stdout, "%v\n", encryption.Hash(publicKeyBytes))
}
func TestMPTWithWalletTxns(t *testing.T) {
	var rs = rand.NewSource(randTime)
	transactions := 100
	var wallets []*Wallet
	pmpt := GetMPT(PERSIST, util.Sequence(2010))
	start := 10
	end := 10

	for true {
		for clients := start; clients <= end; clients *= 10 {
			prng = rand.New(rs)
			wallets = createWallets(clients)
			/*
				prng = rand.New(rs)
				fmt.Printf("using in no db\n")
				generateTransactions(nil, wallets, transactions)
			*/
			/*
				prng = rand.New(rs)
				fmt.Printf("using in memory db\n")
				generateTransactions(GetMPT(MEMORY), wallets,transactions)
			*/
			prng = rand.New(rs)
			fmt.Printf("using level db\n")
			lmpt := GetMPT(LEVEL, util.Sequence(2010))
			saveWallets(lmpt, wallets)
			verifyBalance(lmpt, wallets)
			lmpt.SaveChanges(pmpt.GetNodeDB(), false)
			(lmpt.GetNodeDB().(*util.LevelNodeDB)).RebaseCurrentDB(pmpt.GetNodeDB())

			lmpt.ResetChangeCollector(nil)
			generateTransactions(lmpt, wallets, transactions)
			verifyBalance(lmpt, wallets)
			ts := time.Now()
			lmpt.SaveChanges(pmpt.GetNodeDB(), false)
			fmt.Printf("time taken to persist: %v\n", time.Since(ts))
		}
	}
	/*
		prng = rand.New(rs)
		fmt.Printf("using persist db\n")
		testWithMPT(pmpt, wallets, transactions,false)
	*/
}

func TestMPTChangeCollector(t *testing.T) {
	var rs = rand.NewSource(randTime)
	transactions := 1000
	var wallets []*Wallet
	var clients = 1000
	for i := 0; i < 1; i++ {
		prng = rand.New(rs)
		wallets = createWallets(clients)
		mpt := GetMPT(MEMORY, util.Sequence(2010))
		saveWallets(mpt, wallets)
		verifyBalance(mpt, wallets)
		lmpt := mpt
		for j := 1; j < 10000; j++ {
			cmpt := GetMPT(LEVEL, util.Sequence(2010+j))
			lndb := cmpt.GetNodeDB().(*util.LevelNodeDB)
			lndb.SetPrev(lmpt.GetNodeDB())
			cmpt.SetRoot(lmpt.GetRoot())
			mndb := lndb.GetCurrent().(*util.MemoryNodeDB)
			mpt = lmpt
			lmpt = cmpt
			fmt.Printf("Generating for %v\n", 2010+j)
			generateTransactions(lmpt, wallets, transactions)

			rootKey := lmpt.GetRoot()
			root, err := mndb.GetNode(rootKey)
			if err != nil {
				fmt.Printf("randtime: %v %v\n", i, randTime)
				fmt.Printf("%v\n", err)
				panic(err)
			}
			cmndb := util.NewMemoryNodeDB()
			changes := lmpt.GetChangeCollector().GetChanges()
			for _, change := range changes {
				cmndb.PutNode(change.New.GetHashBytes(), change.New)
			}
			err = cmndb.Validate(root)
			if err != nil {
				mpt.PrettyPrint(os.Stdout)
				fmt.Printf("\n")

				lmpt.PrettyPrint(os.Stdout)
				fmt.Printf("\n")

				fmt.Printf("randtime: %v %v\n", i, randTime)
				fmt.Printf("%v\n", err)
				for _, change := range changes {
					oHash := ""
					if change.Old != nil {
						oHash = change.Old.GetHash()
					}
					fmt.Printf("change: %T %v : %T %v\n", change.Old, oHash, change.New, change.New.GetHash())
				}
				panic(err)
			}
			err = lmpt.Validate()
			if err != nil {
				fmt.Printf("initial mpt\n")
				mpt.PrettyPrint(os.Stdout)
				fmt.Printf("\n")
				fmt.Printf("updated mpt\n")
				lmpt.PrettyPrint(os.Stdout)
				fmt.Printf("\n")

				fmt.Printf("randtime: %v %v\n", i, randTime)
				fmt.Printf("%v\n", err)
				for _, change := range changes {
					oHash := ""
					if change.Old != nil {
						oHash = change.Old.GetHash()
					}
					fmt.Printf("change: %T %v : %T %v\n", change.Old, oHash, change.New, change.New.GetHash())
				}
				panic(err)
			}
		}
	}
}

func GetMPT(dbType int, version util.Sequence) util.MerklePatriciaTrieI {
	var mpt util.MerklePatriciaTrieI

	switch dbType {
	case MEMORY:
		mndb := util.NewMemoryNodeDB()
		mpt = util.NewMerklePatriciaTrie(mndb, version)
	case PERSIST:
		pndb, err := util.NewPNodeDB("/tmp/mpt", "/tmp/mpt/log")
		if err != nil {
			panic(err)
		}
		mpt = util.NewMerklePatriciaTrie(pndb, version)
	case LEVEL:
		mndb := util.NewMemoryNodeDB()
		pndb := util.NewMemoryNodeDB()
		lndb := util.NewLevelNodeDB(mndb, pndb, false)
		mpt = util.NewMerklePatriciaTrie(lndb, version)
	}
	return mpt
}

func saveWallets(mpt util.MerklePatriciaTrieI, wallets []*Wallet) {
	fmt.Printf("number of clients: %v\n", len(wallets))
	if mpt != nil {
		for idx, w := range wallets {
			balance := state.Balance(w.Balance)
			mpt.Insert(util.Path(w.ClientID), &state.State{Balance: balance})
			state, err := getState(mpt, w.ClientID)
			if err != nil {
				panic(err)
			}
			if debug {
				fmt.Printf("INFO:(%v) id:%v balance:%v (%v)\n", idx, w.ClientID, w.Balance, state.Balance)
			}
		}
	}
}

func generateTransactions(mpt util.MerklePatriciaTrieI, wallets []*Wallet, transactions int) {
	if debug {
		fmt.Printf("INFO: random source seed %d\n", randTime)
	}
	ts := time.Now()
	for count := 1; count <= transactions; count++ {
		var wf, wt *Wallet
		csize := len(wallets)
		for true {
			wf = wallets[prng.Intn(csize)]
			if wf.Balance == 0 {
				continue
			}
			wt = wallets[prng.Intn(csize)]
			if wf != wt {
				break
			}
		}

		value := state.Balance(prng.Int63n(int64(wf.Balance)) + 1)
		if deletePercent > 0 && prng.Intn(100) < int(deletePercent) {
			value = wf.Balance
		}
		wf.Balance -= value
		wt.Balance += value
		if wf.Balance == 0 {
			//if debug {
			fmt.Printf("INFO: deleting wallet of %v as balance is zero\n", wf.ClientID)
			//}
			if mpt != nil {
				mpt.Delete(util.Path(wf.ClientID))
			}
		} else {
			if debug {
				fmt.Printf("INFO: moving balance %v from %v to %v\n", value, wf.ClientID, wt.ClientID)
			}
			if mpt != nil {
				s, err := getState(mpt, wf.ClientID)
				if err != nil {
					panic(err)
				}
				s.Balance -= state.Balance(value)
				mpt.Insert(util.Path(wf.ClientID), s)
			}
		}
		if mpt != nil {
			s, err := getState(mpt, wt.ClientID)
			if err != nil && err != util.ErrValueNotPresent {
				fmt.Printf("wt balance: %v %v\n", wt.ClientID, wt.Balance)
				panic(err)
			}
			s.Balance += state.Balance(value)
			mpt.Insert(util.Path(wt.ClientID), s)
		}
		if debug {
			mpt.PrettyPrint(os.Stdout)
			fmt.Printf("\n")
		}
	}
	if mpt != nil {
		fmt.Printf("transactions - num changes: %v in %v\n", len(mpt.GetChangeCollector().GetChanges()), time.Since(ts))
	} else {
		fmt.Printf("transactions - time taken: %v\n", time.Since(ts))
	}
	if mpt == nil {
		return
	}
}

func verifyBalance(mpt util.MerklePatriciaTrieI, wallets []*Wallet) {
	fmt.Printf("verifying balance\n")
	zbcount := 0
	for index := 0; index < len(wallets); index++ {
		w := wallets[index]
		if w.Balance == 0 {
			zbcount++
		}
		s, err := getState(mpt, w.ClientID)
		if err != nil {
			if err == util.ErrNodeNotFound {
				fmt.Printf("Node not found; client - %s\n", w.ClientID)
			} else if err == util.ErrValueNotPresent {
				fmt.Printf("Client %s - deleted ; (Balance - %d)\n", w.ClientID, w.Balance)
			}
		} else {
			if s.Balance != state.Balance(w.Balance) {
				fmt.Printf("balance mismatch (%v): %d; Found : %d\n", w.ClientID, w.Balance, s.Balance)
			}
		}
	}
	fmt.Printf("zero balance clients %v\n", zbcount)
}

func createWallets(num int) []*Wallet {
	wallets := make([]*Wallet, num)
	for i := 0; i < len(wallets); i++ {
		balance := state.Balance(prng.Int63n(1000))
		wallets[i] = &Wallet{Balance: balance}
		wallets[i].Initialize(clientSignatureScheme)
	}
	return wallets
}

func getState(mpt util.MerklePatriciaTrieI, clientID string) (*state.State, error) {
	s := &state.State{}
	s.Balance = state.Balance(0)
	ss, err := mpt.GetNodeValue(util.Path(clientID))
	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		}
		return s, err
	}
	deserializer := &state.Deserializer{}
	s = deserializer.Deserialize(ss).(*state.State)
	return s, nil
}

//TestGenerateCompressionTrainingData - generate the training data for compression
func TestGenerateCompressionTrainingData(t *testing.T) {
	common.SetupRootContext(node.GetNodeContext())
	transaction.SetupEntity(memorystore.GetStorageProvider())
	SetupWallet()
	numClients := 1000
	numTxns := 1000
	wallets := createWallets(numClients)
	for count := 1; count <= numTxns; count++ {
		var wf, wt *Wallet
		csize := len(wallets)
		for true {
			wf = wallets[prng.Intn(csize)]
			if wf.Balance == 0 {
				continue
			}
			wt = wallets[prng.Intn(csize)]
			if wf != wt {
				break
			}
		}
		value := state.Balance(prng.Int63n(int64(wf.Balance)) + 1)
		wf.Balance -= value
		wt.Balance += value
		txn := wf.CreateSendTransaction(wt.ClientID, value, "", 0)
		data := common.ToMsgpack(txn)
		ioutil.WriteFile(fmt.Sprintf("/tmp/txn/data/%v.json", txn.Hash), data.Bytes(), 0644)
	}
}
