package wallet

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"0chain.net/state"

	"0chain.net/util"
)

var randTime = time.Now().UnixNano()
var rs = rand.NewSource(1534554098076517276)
var prng = rand.New(rs)

func TestMPTWithWalletTxns(t *testing.T) {
	mndb := util.NewMemoryNodeDB()
	mpt := util.NewMerklePatriciaTrie(mndb)

	wallets := createWallets(20)

	for index := 0; index < len(wallets); index++ {
		w := wallets[index]
		balance := state.Balance(w.Balance)
		mpt.Insert(util.Path(w.ClientID), &state.State{Balance: balance})
	}

	for count := 1; count <= 10; count++ {
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

		value := prng.Int63n(wf.Balance) + 1
		wf.Balance -= value
		wt.Balance += value

		if wf.Balance == 0 {
			mpt.Delete(util.Path(wf.ClientID))
		} else {
			s, _ := getState(mpt, wf.ClientID)
			s.Balance -= state.Balance(value)
			mpt.Insert(util.Path(wf.ClientID), s)
			s, _ = getState(mpt, wt.ClientID)
			s.Balance += state.Balance(value)
			mpt.Insert(util.Path(wt.ClientID), s)
		}
	}

	for index := 0; index < len(wallets); index++ {
		w := wallets[index]
		s, err := getState(mpt, w.ClientID)
		if err != nil {
			if err == util.ErrNodeNotFound {
				fmt.Printf("Node not found; client - %s\n", w.ClientID)
			} else if err == util.ErrValueNotPresent {
				fmt.Printf("Client %s - deleted ; (Balance - %d)\n", w.ClientID, w.Balance)
			}
		} else {
			if s.Balance != state.Balance(w.Balance) {
				fmt.Printf("Balance mismatch ; ")
				fmt.Printf("Expected : %d; Found : %d\n", w.Balance, s.Balance)
				fmt.Printf("random source seed %d\n", randTime)
			}
		}
	}
}

func createWallets(num int64) []*Wallet {
	wallets := make([]*Wallet, num)

	for i := 0; i < len(wallets); i++ {
		balance := prng.Int63n(1000)
		wallets[i] = &Wallet{Balance: balance}
		wallets[i].Initialize()
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
	} else {
		deserializer := &state.Deserializer{}
		s = deserializer.Deserialize(ss).(*state.State)
	}
	return s, nil
}
