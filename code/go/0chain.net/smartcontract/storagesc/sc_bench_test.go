package storagesc

import (
	"encoding/hex"
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
	numBlobbers              = 30
	numWallets               = 20
	numMiners                = 4
	numSharders              = 2
	numAllocations           = 100
	numMagnaProviders        = 100
	numZcnscAuthorizers      = 10
	numBlobberDelegates      = 4
	numMinerDelegates        = 10
	numSharderDelegates      = 10
	numBlobbersPerAllocation = 4
	numAllocationPayers      = 2
	numPoolsPerDelegate      = 3
	numAllocationPayersPools = 2
	initTokens               = 100000000000
	now                      = common.Timestamp(100000)
	availableKeys            = 10
	signatureScheme          = "bls0chain"
	mockClientId             = "31810bd1258ae95955fb40c7ef72498a556d3587121376d9059119d280f34929"
	mockClientPublicKey      = "mockPublicKey"
	mockTransactionHash      = "1234567890"
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
	mpt, root, config, clients, keys, blobbers, allocatins := setUpMpt(b, mockClientId, ssc.ID)
	allocatins = allocatins
	blobbers = blobbers
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
				ClientID:     clients[0],
				CreationDate: now,
				Value:        config.MinAllocSize,
			},
			input: func() []byte {
				bytes, _ := (&newAllocationRequest{
					DataShards:                 4,
					ParityShards:               4,
					Size:                       config.MinAllocSize,
					Expiration:                 common.Timestamp(config.MinAllocDuration.Seconds()) + now,
					Owner:                      clients[0],
					OwnerPublicKey:             keys[0],
					PreferredBlobbers:          []string{},
					ReadPriceRange:             PriceRange{0, config.MaxReadPrice},
					WritePriceRange:            PriceRange{0, config.MaxWritePrice},
					MaxChallengeCompletionTime: config.MaxChallengeCompletionTime,
					DiversifyBlobbers:          false,
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

func setUpMpt(
	b *testing.B,
	client string,
	sscId string,
) (*util.MerklePatriciaTrie, util.Key, scConfig, []string, []string, []string, []string) {
	pNode, err := util.NewPNodeDB(
		"testdata/name_dataDir",
		"testdata/name_logDir",
	)
	require.NoError(b, err)
	pMpt := util.NewMerklePatriciaTrie(
		pNode,
		0,
		nil,
	)

	clients, keys := addMockkClients(b, pMpt)

	bk := &block.Block{}
	magicBlock := &block.MagicBlock{}
	signatureScheme := &encryption.BLS0ChainScheme{}
	balances := cstate.NewStateContext(
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
	blobbers := addMockBlobbers(b, sscId, *conf, balances)
	allocations := addMockAllocations(b, sscId, *conf, balances, clients[1:], keys[1:])

	return pMpt,
		balances.GetState().GetRoot(),
		*conf,
		clients[:availableKeys],
		keys[:availableKeys],
		blobbers,
		allocations

}

func addMockAllocations(
	b *testing.B,
	sscId string,
	config scConfig,
	balances cstate.StateContextI,
	clients, publicKeys []string,
) []string {
	const mockMinLockDemand = 1
	var allocationIds []string
	txn := transaction.Transaction{
		CreationDate: now,
		ClientID:     datastore.Key(clientId[0]),
		ToClientID:   datastore.Key(clientId[0]),
	}
	var allocations Allocations
	var wps = make([]*writePool, 0, len(clients))
	var rps = make([]*readPool, 0, len(clients))
	for i := 0; i < numAllocations; i++ {
		clientIndex := (i % (len(clients) - 1 - numAllocationPayersPools)) + 1
		client := clients[clientIndex]
		hashdata := fmt.Sprintf("%v:%v:%v:%v:%v", txn.CreationDate, txn.ClientID,
			txn.ToClientID, txn.Value, encryption.Hash(strconv.Itoa(i)))
		txn.Hash = encryption.Hash(hashdata)
		if i < availableKeys {
			allocationIds = append(allocationIds, txn.Hash)
		}
		sa := &StorageAllocation{
			ID:                         txn.Hash,
			DataShards:                 numBlobbersPerAllocation / 2,
			ParityShards:               numBlobbersPerAllocation / 2,
			Size:                       config.MinAllocSize,
			Expiration:                 common.Timestamp(config.MinAllocDuration.Seconds()) + now,
			Owner:                      client,
			OwnerPublicKey:             publicKeys[i%clientIndex],
			ReadPriceRange:             PriceRange{0, config.MaxReadPrice},
			WritePriceRange:            PriceRange{0, config.MaxWritePrice},
			MaxChallengeCompletionTime: config.MaxChallengeCompletionTime,
			DiverseBlobbers:            false,
		}

		numAllocBlobbers := sa.DataShards + sa.ParityShards
		startBlobbers := i % (numBlobbers - numAllocBlobbers)
		for j := 0; j < numAllocBlobbers; j++ {
			sa.BlobberDetails = append(sa.BlobberDetails, &BlobberAllocation{
				BlobberID:     getMockBlobberId(startBlobbers + j),
				AllocationID:  sa.ID,
				Size:          config.MinAllocSize,
				Stats:         &StorageAllocationStats{},
				Terms:         getMockBlobberTerms(config),
				MinLockDemand: mockMinLockDemand,
			})
		}
		_, err := balances.InsertTrieNode(sa.GetKey(sscId), sa)
		require.NoError(b, err)

		cp := newChallengePool()
		cp.TokenPool.ID = challengePoolKey(sscId, sscId)
		_, err = balances.InsertTrieNode(challengePoolKey(sscId, sscId), cp)

		startClients := (i % (len(clients) - numAllocationPayersPools)) + 1
		amountPerBlobber := state.Balance(float64(sa.Size) / float64(numAllocBlobbers))
		for j := 0; j < numAllocationPayers; j++ {
			cIndex := startClients + j
			var wp *writePool
			var rp *readPool
			if len(wps) > cIndex {
				wp = wps[cIndex]
				rp = rps[cIndex]
			} else {
				wp = new(writePool)
				wps = append(wps, wp)
				rp = new(readPool)
				rps = append(rps, rp)
			}
			for k := 0; k < numAllocationPayersPools; k++ {
				wap := allocationPool{
					ExpireAt:     sa.Expiration,
					AllocationID: sa.ID,
				}
				wap.ID = sa.ID + strconv.Itoa(j) + strconv.Itoa(k)
				rap := allocationPool{
					ExpireAt:     sa.Expiration,
					AllocationID: sa.ID,
				}
				rap.ID = sa.ID + strconv.Itoa(j) + strconv.Itoa(k)
				for l := 0; l < numAllocBlobbers; l++ {
					wap.Blobbers.add(&blobberPool{
						BlobberID: getMockBlobberId(startBlobbers + l),
						Balance:   amountPerBlobber,
					})
					rap.Blobbers.add(&blobberPool{
						BlobberID: getMockBlobberId(startBlobbers + l),
						Balance:   amountPerBlobber,
					})
				}
				wp.Pools = append(wp.Pools, &wap)
				rp.Pools = append(rp.Pools, &rap)
			}
		}
	}
	for i := 0; i < len(wps); i++ {
		_, err := balances.InsertTrieNode(readPoolKey(sscId, clients[i]), wps[i])
		require.NoError(b, err)
		_, err = balances.InsertTrieNode(readPoolKey(sscId, clients[i]), rps[i])
		require.NoError(b, err)
	}

	_, err := balances.InsertTrieNode(ALL_ALLOCATIONS_KEY, &allocations)
	require.NoError(b, err)
	return allocationIds
}

func addMockkClients(
	b *testing.B,
	pMpt *util.MerklePatriciaTrie,
) ([]string, []string) {
	var sigScheme encryption.SignatureScheme = encryption.GetSignatureScheme(signatureScheme)
	var clientIds, publicKeys []string
	for i := 0; i < numWallets; i++ {
		err := sigScheme.GenerateKeys()
		require.NoError(b, err)
		publicKeyBytes, err := hex.DecodeString(sigScheme.GetPublicKey())
		require.NoError(b, err)
		clientID := encryption.Hash(publicKeyBytes)
		publicKey := sigScheme.GetPublicKey()
		clientIds = append(clientIds, clientID)
		publicKeys = append(publicKeys, publicKey)
		is := &state.State{}
		is.SetTxnHash("0000000000000000000000000000000000000000000000000000000000000000")
		is.Balance = state.Balance(initTokens)
		pMpt.Insert(util.Path(clientID), is)
	}

	return clientIds, publicKeys
}

func getMockBlobberTerms(conf scConfig) Terms {
	return Terms{
		ReadPrice:               conf.MaxReadPrice,
		WritePrice:              conf.MaxWritePrice,
		MinLockDemand:           1,
		MaxOfferDuration:        10000 * conf.MinOfferDuration,
		ChallengeCompletionTime: conf.MaxChallengeCompletionTime,
	}
}

func addMockBlobbers(
	b *testing.B,
	sscId string,
	conf scConfig,
	balances cstate.StateContextI,
) []string {
	var blobbers StorageNodes
	var blobberIds []string
	for i := 0; i < numBlobbers; i++ {
		spSettings := stakePoolSettings{
			DelegateWallet: "",
			MinStake:       0,
			MaxStake:       1e12,
			NumDelegates:   10,
			ServiceCharge:  0.1,
		}
		blobber := &StorageNode{
			ID:      getMockBlobberId(i),
			BaseURL: getMockBlobberId(i) + ".com",
			Geolocation: StorageNodeGeolocation{
				Latitude:  0.0001 * float64(i),
				Longitude: 0.0001 * float64(i),
			},
			Terms:             getMockBlobberTerms(conf),
			Capacity:          conf.MinBlobberCapacity * 2,
			Used:              0,
			LastHealthCheck:   now - common.Timestamp(1),
			PublicKey:         "",
			StakePoolSettings: spSettings,
		}
		if i < availableKeys {
			blobberIds = append(blobberIds, blobber.ID)
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
		for j := 0; j < numBlobberDelegates; j++ {
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
	return blobberIds
}

func getMockBlobberId(index int) string {
	return "mockBlobber_" + strconv.Itoa(index)
}

/*
















































 */
