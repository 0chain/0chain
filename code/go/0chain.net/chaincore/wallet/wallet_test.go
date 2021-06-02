package wallet

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/0chain/0chain/code/go/0chain.net/core/logging"

	"github.com/0chain/0chain/code/go/0chain.net/chaincore/node"
	"github.com/0chain/0chain/code/go/0chain.net/chaincore/state"
	"github.com/0chain/0chain/code/go/0chain.net/chaincore/transaction"
	"github.com/0chain/0chain/code/go/0chain.net/core/common"
	"github.com/0chain/0chain/code/go/0chain.net/core/encryption"
	"github.com/0chain/0chain/code/go/0chain.net/core/memorystore"

	"github.com/0chain/0chain/code/go/0chain.net/core/util"
)

var randTime = time.Now().UnixNano()

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
		t.Fatal(err)
	}

	_, err = hex.DecodeString(sigScheme.GetPublicKey())
	if err != nil {
		t.Fatal(err)
	}
}

func TestMPTWithWalletTxns(t *testing.T) {
	var rs = rand.NewSource(randTime)
	transactions := 10
	var wallets []*Wallet
	start := 10
	end := 10

	for clients := start; clients <= end; clients *= 10 {
		prng = rand.New(rs)
		wallets = createWallets(clients)

		prng = rand.New(rs)
		lmpt := GetMPT(LEVEL, util.Sequence(2010))
		saveWallets(lmpt, wallets)
		verifyBalance(lmpt, wallets)

		lmpt.ResetChangeCollector(nil)
		generateTransactions(lmpt, wallets, transactions)
		verifyBalance(lmpt, wallets)
	}
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
		for j := 1; j < 10; j++ {
			cmpt := GetMPT(LEVEL, util.Sequence(2010+j))
			lndb := cmpt.GetNodeDB().(*util.LevelNodeDB)
			lndb.SetPrev(lmpt.GetNodeDB())
			cmpt.SetRoot(lmpt.GetRoot())
			mndb := lndb.GetCurrent().(*util.MemoryNodeDB)
			mpt = lmpt
			lmpt = cmpt
			generateTransactions(lmpt, wallets, transactions)

			rootKey := lmpt.GetRoot()
			root, err := mndb.GetNode(rootKey)
			if err != nil {
				t.Fatal(err)
			}
			cmndb := util.NewMemoryNodeDB()
			changes := lmpt.GetChangeCollector().GetChanges()
			for _, change := range changes {
				if err := cmndb.PutNode(change.New.GetHashBytes(), change.New); err != nil {
					t.Fatal(err)
				}
			}
			err = cmndb.Validate(root)
			if err != nil {
				t.Fatal(err)
			}
			err = lmpt.Validate()
			if err != nil {
				t.Fatal(err)
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
	if mpt != nil {
		for _, w := range wallets {
			balance := state.Balance(w.Balance)
			if _, err := mpt.Insert(util.Path(w.ClientID), &state.State{Balance: balance}); err != nil {
				panic(err)
			}
			_, err := getState(mpt, w.ClientID)
			if err != nil {
				panic(err)
			}
		}
	}
}

func generateTransactions(mpt util.MerklePatriciaTrieI, wallets []*Wallet, transactions int) {
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

		if wf == nil {
			panic("expected non nil wallet")
		}
		value := prng.Int63n(wf.Balance)
		if wf.Balance == 0 {
			if mpt != nil {
				if _, err := mpt.Delete(util.Path(wf.ClientID)); err != nil {
					panic(err)
				}
			}
		} else {
			if mpt != nil {
				s, err := getState(mpt, wf.ClientID)
				if err != nil {
					panic(err)
				}
				s.Balance -= state.Balance(value)
				if _, err := mpt.Insert(util.Path(wf.ClientID), s); err != nil {
					panic(err)
				}
				wf.Balance = int64(s.Balance)
			}
		}
		if mpt != nil {
			if wt == nil {
				panic("expected non nil wallet")
			}
			s, err := getState(mpt, wt.ClientID)
			if err != nil {
				panic(err)
			}
			s.Balance += state.Balance(value)
			if _, err := mpt.Insert(util.Path(wt.ClientID), s); err != nil {
				panic(err)
			}
			wt.Balance = int64(s.Balance)
		}
	}
}

func verifyBalance(mpt util.MerklePatriciaTrieI, wallets []*Wallet) {
	zbcount := 0
	for index := 0; index < len(wallets); index++ {
		w := wallets[index]
		if w.Balance == 0 {
			zbcount++
		}
		s, err := getState(mpt, w.ClientID)
		if err != nil {
			panic(err)
		} else {
			if s.Balance != state.Balance(w.Balance) {
				panic(fmt.Sprintf("balance mismatch (%v): %d; Found : %d\n", w.ClientID, w.Balance, s.Balance))
			}
		}
	}
}

func createWallets(num int) []*Wallet {
	wallets := make([]*Wallet, num)
	for i := 0; i < len(wallets); i++ {
		balance := prng.Int63n(1000)
		wallets[i] = &Wallet{Balance: balance}
		if err := wallets[i].Initialize(clientSignatureScheme); err != nil {
			panic(err)
		}
	}
	return wallets
}

func getState(mpt util.MerklePatriciaTrieI, clientID string) (*state.State, error) {
	ss, err := mpt.GetNodeValue(util.Path(clientID))
	if err != nil {
		return nil, err
	}

	s, ok := ss.(*state.State)
	if !ok {
		ssv, ok := ss.(*util.SecureSerializableValue)
		if !ok {
			return nil, errors.New("unexpected type")
		}
		s := &state.State{}
		if err := s.Decode(ssv.Encode()); err != nil {
			return nil, err
		}
		return s, nil
	}

	return s, nil
}

//TestGenerateCompressionTrainingData - generate the training data for compression
func TestGenerateCompressionTrainingData(t *testing.T) {
	if err := os.MkdirAll("/tmp/txn/data/", 0700); err != nil {
		t.Fatal(err)
	}

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
		if wf == nil || wt == nil {
			panic("expected non nil wallets")
		}
		value := prng.Int63n(wf.Balance) + 1
		txn := wf.CreateSendTransaction(wt.ClientID, value, "", 0)
		data := common.ToMsgpack(txn)
		err := ioutil.WriteFile(fmt.Sprintf("/tmp/txn/data/%v.json", txn.Hash), data.Bytes(), 0644)
		if err != nil {
			panic(err)
		}
	}

	if err := os.RemoveAll("/tmp/txn/data/"); err != nil {
		t.Fatal(err)
	}
}
