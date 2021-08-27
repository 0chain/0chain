package storagesc

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"0chain.net/core/datastore"

	"github.com/stretchr/testify/require"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/state"
	"0chain.net/core/encryption"
	"0chain.net/core/util"

	"0chain.net/core/common"

	sci "0chain.net/chaincore/smartcontractinterface"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
)

const (
	numBlobbers            = 1000
	numWallets             = 100
	numMiners              = 10
	numSharders            = 5
	numAllocations         = 1000
	numMagnaProviders      = 100
	numZcnscAuthorizers    = 10
	numBlobberStakeHolders = 10
	now                    = common.Timestamp(100000)
	mockClientId           = "31810bd1258ae95955fb40c7ef72498a556d3587121376d9059119d280f34929"
	mockClientPublicKey    = "mockPublicKey"
	mockTransactionHash    = "1234567890"
)

func main() {
	res := testing.Benchmark(BenchmarkExecute)
	fmt.Printf("Memory allocations : %d \n", res.MemAllocs)
	fmt.Printf("Number of bytes allocated: %d \n", res.Bytes)
	fmt.Printf("Number of run: %d \n", res.N)
	fmt.Printf("Time taken: %s \n", res.T)
}

func BenchmarkExecute(b *testing.B) {
	var ssc = StorageSmartContract{
		SmartContract: sci.NewSC(ADDRESS),
	}
	mpt, root, config := setUpMpt(b, mockClientId, ssc.ID)
	config = config
	benchmarks := []struct {
		name     string
		endpoint func(
			*transaction.Transaction,
			[]byte,
			cstate.StateContextI,
		) (string, error)
		txn   transaction.Transaction
		input []byte
	}{
		{
			name:     "new_allocation_request",
			endpoint: ssc.newAllocationRequest,
			txn: transaction.Transaction{
				HashIDField: datastore.HashIDField{
					Hash: mockTransactionHash,
				},
				ClientID:     mockClientId,
				CreationDate: now,
				Value:        config.MinAllocSize,
			},
			input: func() []byte {
				bytes, _ := (&newAllocationRequest{
					DataShards:                 4,
					ParityShards:               4,
					Size:                       config.MinAllocSize,
					Expiration:                 common.Timestamp(config.MinAllocDuration.Seconds()) + now,
					Owner:                      mockClientId,
					OwnerPublicKey:             mockClientPublicKey,
					PreferredBlobbers:          []string{},
					ReadPriceRange:             PriceRange{0, config.MaxReadPrice},
					WritePriceRange:            PriceRange{0, config.MaxWritePrice},
					MaxChallengeCompletionTime: config.MaxChallengeCompletionTime,
					DiversifyBlobbers:          true,
				}).encode()
				return bytes
			}(),
		},
		{
			name:     "new_read_pool",
			endpoint: ssc.newReadPool,
			txn:      transaction.Transaction{},
			input:    []byte{},
		},
		{
			name:     "stake_pool_pay_interests",
			endpoint: ssc.stakePoolPayInterests,
			txn:      transaction.Transaction{},
			input: func() []byte {
				bytes, _ := json.Marshal(&stakePoolRequest{
					BlobberID: "mockBlobber_" + strconv.Itoa(1),
					PoolID:    "mockBlobber_" + strconv.Itoa(1) + "Pool" + strconv.Itoa(1),
				})
				return bytes
			}(),
		},
	}
	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				_, balances := getBalances(b, bm.name, &bm.txn, root, mpt)
				b.StartTimer()
				_, err := bm.endpoint(&bm.txn, bm.input, balances)
				require.NoError(b, err)
			}
		})
	}
}

func setUpMpt(
	b *testing.B,
	client string,
	sscId string,
) (*util.MerklePatriciaTrie, util.Key, scConfig) {
	pMpt, balances, pNode := getNewEmptyMpt(b)

	addMockkClient2(b, client, pMpt, balances, pNode)

	bk := &block.Block{}
	magicBlock := &block.MagicBlock{}
	signatureScheme := &encryption.BLS0ChainScheme{}
	balances = cstate.NewStateContext(
		bk,
		pMpt,
		&state.Deserializer{},
		&transaction.Transaction{
			HashIDField: datastore.HashIDField{
				Hash: mockTransactionHash,
			},
		},
		func(*block.Block) []string { return []string{} },
		func() *block.Block { return bk },
		func() *block.MagicBlock { return magicBlock },
		func() encryption.SignatureScheme { return signatureScheme },
	)

	conf := setConfig(b, balances)
	addMockBlobbers(b, sscId, *conf, balances)

	//_, err := balances.GetClientBalance("31810bd1258ae95955fb40c7ef72498a556d3587121376d9059119d280f34929")
	//require.NoError(b, err)
	//fmt.Println("balance", balance, "id", "31810bd1258ae95955fb40c7ef72498a556d3587121376d9059119d280f34929")

	return pMpt, balances.GetState().GetRoot(), *conf
}

func addMockkClient2(
	b *testing.B,
	client string,
	pMpt *util.MerklePatriciaTrie,
	balances cstate.StateContextI,
	stateDB util.NodeDB,
) {
	initStates := state.NewInitStates()
	err := initStates.Read("testdata/initial_state.yaml")
	require.NoError(b, err)
	for _, v := range initStates.States {
		is := &state.State{}
		is.SetTxnHash("0000000000000000000000000000000000000000000000000000000000000000")
		is.Balance = state.Balance(v.Tokens)
		pMpt.Insert(util.Path(v.ID), is)

		//s, err := pMpt.GetNodeValue(util.Path(v.ID))
		//news := &state.State{}
		//news.Decode(s.Encode())
		//fmt.Println("first loop s", s, "err", err)

		//balance, err := balances.GetClientBalance(v.ID)
		//fmt.Println("balance", balance, "id", v.ID)
	}

}

func addMockkClient(
	b *testing.B,
	client string,
	pMpt util.MerklePatriciaTrie,
	balances cstate.StateContextI,
) {
	cState := &state.State{
		TxnHash: "0000000000000000000000000000000000000000000000000000000000000000",
		Round:   7,
		Balance: state.Balance(1e10 * 777),
	}
	cStateTest := state.State{}
	cStateTest.Decode(cState.Encode())
	cState.TxnHashBytes = cState.Encode()
	cStateTest.Decode(cState.Encode())

	balances.SetStateContext(cState)
	//_, err := balances.GetState().Insert(util.Path(client), cState)
	_, err := pMpt.Insert(util.Path(client), cState)
	require.NoError(b, err)

	s, err := pMpt.GetNodeValue(util.Path(client))
	news := &state.State{}
	news.Decode(s.Encode())

	bal, err := balances.GetClientBalance(client)
	bal = bal
	require.NoError(b, err)
}

func addMockBlobbers(
	b *testing.B,
	sscId string,
	conf scConfig,
	balances cstate.StateContextI,
) {
	var blobbers StorageNodes
	for i := 0; i < numBlobbers; i++ {
		spSettings := stakePoolSettings{
			DelegateWallet: "",
			MinStake:       0,
			MaxStake:       1e12,
			NumDelegates:   10,
			ServiceCharge:  0.1,
		}
		blobber := &StorageNode{
			ID:      "mockBlobber_" + strconv.Itoa(i),
			BaseURL: "mockBlobber_" + strconv.Itoa(i) + ".com",
			Geolocation: StorageNodeGeolocation{
				Latitude:  0.0001 * float64(i),
				Longitude: 0.0001 * float64(i),
			},
			Terms: Terms{
				ReadPrice:               conf.MaxReadPrice,
				WritePrice:              conf.MaxWritePrice,
				MinLockDemand:           1,
				MaxOfferDuration:        10000 * conf.MinOfferDuration,
				ChallengeCompletionTime: conf.MaxChallengeCompletionTime,
			},
			Capacity:          conf.MinBlobberCapacity * 2,
			Used:              0,
			LastHealthCheck:   now - common.Timestamp(1),
			PublicKey:         "",
			StakePoolSettings: spSettings,
		}
		blobbers.Nodes.add(blobber)
		_, err := balances.InsertTrieNode(blobber.GetKey(sscId), blobber)
		require.NoError(b, err)
		sp := &stakePool{
			Pools:  make(map[string]*delegatePool),
			Offers: make(map[string]*offerPool),
			Rewards: stakePoolRewards{
				Charge:    0,
				Blobber:   0,
				Validator: 0,
			},
			Settings: spSettings,
		}
		for j := 0; j < numBlobberStakeHolders; j++ {
			id := blobber.ID + "Pool" + strconv.Itoa(i)
			sp.Pools[id] = &delegatePool{}
			sp.Pools[id].ID = id
			sp.Pools[id].Balance = conf.MaxStake
		}
		require.NoError(b, sp.save(sscId, blobber.ID, balances))
	}
	_, err := balances.InsertTrieNode(ALL_BLOBBERS_KEY, &blobbers)

	allBlobbersBytes, err := balances.GetTrieNode(ALL_BLOBBERS_KEY)
	allBlobbersBytes = allBlobbersBytes
	require.NoError(b, err)
}

func getNewEmptyMpt(b *testing.B) (*util.MerklePatriciaTrie, cstate.StateContextI, *util.PNodeDB) {
	pNode, err := util.NewPNodeDB(
		"testdata/name_dataDir",
		"testdata/name_logDir",
	)
	require.NoError(b, err)
	mpt := util.NewMerklePatriciaTrie(
		pNode,
		0,
		nil,
	)
	bk := &block.Block{}
	magicBlock := &block.MagicBlock{}
	signatureScheme := &encryption.BLS0ChainScheme{}
	return mpt, cstate.NewStateContext(
		bk,
		mpt,
		&state.Deserializer{},
		&transaction.Transaction{
			HashIDField: datastore.HashIDField{
				Hash: mockTransactionHash,
			},
		},
		func(*block.Block) []string { return []string{} },
		func() *block.Block { return bk },
		func() *block.MagicBlock { return magicBlock },
		func() encryption.SignatureScheme { return signatureScheme },
	), pNode
}

func getBalances(
	b *testing.B,
	name string,
	txn *transaction.Transaction,
	root util.Key,
	pMpt *util.MerklePatriciaTrie,
) (*util.MerklePatriciaTrie, cstate.StateContextI) {
	pNode := pMpt.GetNodeDB()
	memNode := util.NewMemoryNodeDB()
	levelNode := util.NewLevelNodeDB(
		memNode,
		pNode,
		false,
	)
	mpt := util.NewMerklePatriciaTrie(
		levelNode,
		1,
		root,
	)
	bk := &block.Block{}
	magicBlock := &block.MagicBlock{}
	signatureScheme := &encryption.BLS0ChainScheme{}
	return mpt, cstate.NewStateContext(
		bk,
		mpt,
		&state.Deserializer{},
		txn,
		func(*block.Block) []string { return []string{} },
		func() *block.Block { return bk },
		func() *block.MagicBlock { return magicBlock },
		func() encryption.SignatureScheme { return signatureScheme },
	)
}

/*
















































 */
