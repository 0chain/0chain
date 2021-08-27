package storagesc

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

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
	numBlobbers            = 10
	numWallets             = 100
	numMiners              = 10
	numSharders            = 5
	numAllocations         = 1000
	numMagnaProviders      = 100
	numZcnscAuthorizers    = 10
	numBlobberStakeHolders = 10
	now                    = common.Timestamp(100000)
)

func main() {
	res := testing.Benchmark(BenchmarkExecute)
	fmt.Printf("Memory allocations : %d \n", res.MemAllocs)
	fmt.Printf("Number of bytes allocated: %d \n", res.Bytes)
	fmt.Printf("Number of run: %d \n", res.N)
	fmt.Printf("Time taken: %s \n", res.T)
}

func BenchmarkExecute(b *testing.B) {
	const (
		mockClientId        = "1234567890123456"
		mockClientPublicKey = "mockPublicKey"
		mockTransactionHash = "1234567890"
	)
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
	}{ /*
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
			},*/
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
				balances := getBalances(b, bm.name, &bm.txn, root, mpt)
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
) (util.MerklePatriciaTrie, util.Key, scConfig) {
	pMpt, balances := getNewEmptyMpt(b)
	pNode := pMpt.GetNodeDB()

	memNode := util.NewMemoryNodeDB()
	levelNode := util.NewLevelNodeDB(
		memNode,
		pNode,
		false,
	)
	levelMpt := util.NewMerklePatriciaTrie(
		levelNode,
		1,
		pMpt.GetRoot(),
	)

	conf := setConfig(b, balances)
	addMockkClient(b, client, *levelMpt, balances)
	addMockBlobbers(b, sscId, *conf, balances)
	err := pMpt.MergeMPTChanges(levelMpt)
	err = err

	return pMpt, balances.GetState().GetRoot(), *conf
}

func addMockkClient(
	b *testing.B,
	client string,
	pMpt util.MerklePatriciaTrie,
	balances cstate.StateContextI,
) {
	cState := state.State{
		TxnHash: "12",
		//TxnHashBytes: []uint8{49, 50},
		Round:   7,
		Balance: state.Balance(1e10 * 777),
	}
	cState = cState
	key, err := balances.GetState().Insert(util.Path(client), &cState)
	require.NoError(b, err)
	return
	fmt.Println("key", key, err)

	var blobbers StorageNodes
	blobbers.Nodes.add(&StorageNode{ID: "fred"})
	//_, err = balances.InsertTrieNode(ALL_BLOBBERS_KEY, &blobbers)
	key_hash := encryption.Hash(ALL_BLOBBERS_KEY)
	key, err = balances.GetState().Insert(util.Path(key_hash), &cState)

	//err = pMpt.MergeMPTChanges(balances.GetState())

	allBlobbersBytes, err := balances.GetTrieNode(ALL_BLOBBERS_KEY)
	allBlobbersList := &StorageNodes{}
	err = json.Unmarshal(allBlobbersBytes.Encode(), allBlobbersList)

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

func getNewEmptyMpt(b *testing.B) (util.MerklePatriciaTrie, cstate.StateContextI) {
	pNode, err := util.NewPNodeDB(
		"testdata/name_dataDir",
		"testdata/name_logDir",
	)
	require.NoError(b, err)
	mpt := util.NewMerklePatriciaTrie(
		pNode,
		1,
		nil,
	)
	bk := &block.Block{}
	magicBlock := &block.MagicBlock{}
	signatureScheme := &encryption.BLS0ChainScheme{}
	return *mpt, cstate.NewStateContext(
		bk,
		mpt,
		&state.Deserializer{},
		&transaction.Transaction{},
		func(*block.Block) []string { return []string{} },
		func() *block.Block { return bk },
		func() *block.MagicBlock { return magicBlock },
		func() encryption.SignatureScheme { return signatureScheme },
	)
}

func getBalances(
	b *testing.B,
	name string,
	txn *transaction.Transaction,
	root util.Key,
	pMpt util.MerklePatriciaTrie,
) cstate.StateContextI {
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
	return cstate.NewStateContext(
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
func NewStateContext(
	b *block.Block,
	s util.MerklePatriciaTrieI,
	csd state.DeserializerI, t *transaction.Transaction,
	getSharderFunc func(*block.Block) []string,
	getLastestFinalizedMagicBlock func() *block.Block,
	getChainCurrentMagicBlock func() *block.MagicBlock,
	getChainSignature func() encryption.SignatureScheme,
) (
	balances *StateContext,
) {
	return &StateContext{
		block:                         b,
		state:                         s,
		clientStateDeserializer:       csd,
		txn:                           t,
		getSharders:                   getSharderFunc,
		getLastestFinalizedMagicBlock: getLastestFinalizedMagicBlock,
		getChainCurrentMagicBlock:     getChainCurrentMagicBlock,
		getSignature:                  getChainSignature,
	}
}

// NewStateContext creation helper.
func (c *Chain) NewStateContext(b *block.Block, s util.MerklePatriciaTrieI,
	txn *transaction.Transaction) (balances *bcstate.StateContext) {

	return bcstate.NewStateContext(b, s, c.clientStateDeserializer,
		txn,
		c.GetBlockSharders,
		c.GetLatestFinalizedMagicBlock,
		c.GetCurrentMagicBlock,
		c.GetSignatureScheme)
}


























































*/
