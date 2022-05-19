package storagesc

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/encryption"
	"0chain.net/core/util"

	"github.com/stretchr/testify/require"
)

type mptStore struct {
	mpt  util.MerklePatriciaTrieI
	mndb *util.MemoryNodeDB
	lndb *util.LevelNodeDB
	pndb *util.PNodeDB
	dir  string
}

func newMptStore(tb testing.TB) (mpts *mptStore) {
	mpts = new(mptStore)

	var dir, err = ioutil.TempDir("", "storage-mpt")
	require.NoError(tb, err)

	mpts.pndb, err = util.NewPNodeDB(filepath.Join(dir, "data"),
		filepath.Join(dir, "log"))
	require.NoError(tb, err)

	mpts.merge(tb)

	mpts.dir = dir
	return
}

func (mpts *mptStore) Close() (err error) {
	if mpts == nil {
		return
	}
	if mpts.pndb != nil {
		mpts.pndb.Flush()
	}
	if mpts.dir != "" {
		err = os.RemoveAll(mpts.dir)
	}
	return
}

func (mpts *mptStore) merge(tb testing.TB) {
	if mpts == nil {
		return
	}

	var root util.Key

	if mpts.mndb != nil {
		root = mpts.mpt.GetRoot()
		require.NoError(tb, util.MergeState(
			context.Background(), mpts.mndb, mpts.pndb,
		))
		// mpts.pndb.Flush()
	}

	// for a worst case, no cached data, and we have to get everything from
	// the persistent store, from rocksdb

	mpts.mndb = util.NewMemoryNodeDB()                           //
	mpts.lndb = util.NewLevelNodeDB(mpts.mndb, mpts.pndb, false) // transaction
	mpts.mpt = util.NewMerklePatriciaTrie(mpts.lndb, 1, root)    //
}

//
// 2) Also need to check how fast are the allocations created in storageSC
//    if there are 1000 blobbers.
//

//
// go test -v -timeout 1h -bench newAllocationRequest | prettybench
//

func Benchmark_newAllocationRequest(b *testing.B) {

	for _, n := range []int{
		20,
		100,
		500,
		1000,
		2000,
	} {
		b.Run(fmt.Sprintf("%d blobbers", n), func(b *testing.B) {

			var (
				ssc            = newTestStorageSC()
				balances       = newTestBalances(b, true)
				client         = newClient(100000*x10, balances)
				tp, exp  int64 = 0, int64(toSeconds(time.Hour))

				conf *Config
				err  error
			)

			defer balances.mpts.Close()

			balances.skipMerge = true
			conf = setConfig(b, balances)

			// call the addAllocation to create and stake n blobbers, the resulting
			// allocation will not be used
			tp += 1
			addAllocation(b, ssc, client, tp, exp, n, balances)

			conf.MinAllocSize = 1 * KB
			mustSave(b, scConfigKey(ADDRESS), conf, balances)

			balances.skipMerge = false
			balances.mpts.merge(b)

			b.ResetTimer()

			var (
				input []byte
				tx    *transaction.Transaction
			)

			// create an allocation
			for i := 0; i < b.N; i++ {

				b.StopTimer()
				{
					tp += 1

					var nar = new(newAllocationRequest)
					nar.DataShards = 10
					nar.ParityShards = 10
					nar.Expiration = common.Timestamp(exp)
					nar.Owner = client.id
					nar.OwnerPublicKey = client.pk
					nar.ReadPriceRange = PriceRange{1e10, 10e10}
					nar.WritePriceRange = PriceRange{2e10, 20e10}
					nar.Size = 1 * KB // 2 GB
					nar.MaxChallengeCompletionTime = 200 * time.Hour

					input = mustEncode(b, nar)                        //
					tx = newTransaction(client.id, ADDRESS, 1e10, tp) //
					balances.setTransaction(b, tx)
				}
				b.StartTimer()

				_, err = ssc.newAllocationRequest(tx, input, balances, nil)
				require.NoError(b, err)
			}
			b.ReportAllocs()

		})
	}
}

//
// 3) And how fast the challenges are created if there are 1000 blobbers,
//    1000 allocations, 10000 files.
//

//
// go test -v -timeout 1h -bench generateChallenges | prettybench
//

func Benchmark_generateChallenges(b *testing.B) {

	var (
		ssc            = newTestStorageSC()
		balances       = newTestBalances(b, true)
		client         = newClient(100000*x10, balances)
		tp, exp  int64 = 0, int64(toSeconds(time.Hour))

		tx    *transaction.Transaction
		blobs []*Client
		conf  *Config
		err   error
	)

	defer balances.mpts.Close()

	balances.skipMerge = true
	conf = setConfig(b, balances)

	// 1. just create 1000 blobbers
	b.Log("add 1k blobbers")
	tp += 1
	balances.skipMerge = true // don't merge transactions for now
	_, blobs = addAllocation(b, ssc, client, tp, exp, 1000, balances)

	// 2. and 1000 corresponding validators
	b.Log("add 1k corresponding validators")
	for _, bl := range blobs {
		tp += 1
		tx = newTransaction(bl.id, ssc.ID, 0, tp)
		_, err = ssc.addValidator(tx, bl.addValidatorRequest(b), balances)
		require.NoError(b, err)
	}

	conf.MinAllocSize = 1 * KB
	mustSave(b, scConfigKey(ADDRESS), conf, balances)

	// 3. create 1000 allocations
	b.Log("add 1k allocations")
	var allocs []string
	for i := 0; i < 1000; i++ {

		var nar = new(newAllocationRequest)
		nar.DataShards = 10
		nar.ParityShards = 10
		nar.Expiration = common.Timestamp(exp)
		nar.Owner = client.id
		nar.OwnerPublicKey = client.pk
		nar.ReadPriceRange = PriceRange{1 * x10, 10 * x10}
		nar.WritePriceRange = PriceRange{2 * x10, 20 * x10}
		nar.Size = 1 * KB
		nar.MaxChallengeCompletionTime = 200 * time.Hour

		var resp, err = nar.callNewAllocReq(b, client.id, 15*x10, ssc, tp,
			balances)
		require.NoError(b, err)

		var deco StorageAllocation
		require.NoError(b, deco.Decode([]byte(resp)))

		allocs = append(allocs, deco.ID)
	}

	// 4. "write" 10 files for every one of the allocations
	b.Log("write 10k files")
	for _, allocID := range allocs {
		var alloc *StorageAllocation
		alloc, err = ssc.getAllocation(allocID, balances)
		require.NoError(b, err)
		alloc.Stats = new(StorageAllocationStats)
		alloc.Stats.NumWrites += 10 // 10 files
		for _, d := range alloc.BlobberAllocs {
			d.AllocationRoot = "allocation-root"
		}
		_, err = balances.InsertTrieNode(alloc.GetKey(ssc.ID), alloc)
		require.NoError(b, err)
	}

	// 5. merge all transactions into p node db
	b.Log("merge all into p node db")
	balances.skipMerge = false
	balances.mpts.merge(b)

	b.ResetTimer()
	b.Log("start benchmarks")

	var blk = new(block.Block)

	for _, mcpg := range []int{
		5, 10, 15, 20, 30, 100,
	} {

		conf.MaxChallengesPerGeneration = mcpg
		mustSave(b, scConfigKey(ssc.ID), conf, balances)

		b.Run(fmt.Sprintf("max chall per gen %d", mcpg), func(b *testing.B) {

			// 6. generate challenges
			for i := 0; i < b.N; i++ {

				b.StopTimer()
				{
					// revert the stats to allow generation
					tp += 1

					tp += 1
					blk.PrevHash = encryption.Hash(fmt.Sprintf("block-%d", i))
					tx = newTransaction(client.id, ssc.ID, 0, tp)
					balances.setTransaction(b, tx) // merge into p node db
				}
				b.StartTimer()

				err = ssc.generateChallenge(tx, blk, nil, balances)
				require.NoError(b, err)
			}
			b.ReportAllocs()
		})
	}

}

//
// benchmark for a challenge response
//

//
// go test -v -timeout 1h -benchtime=5s -bench verifyChallenge | prettybench
//

func Benchmark_verifyChallenge(b *testing.B) {

	var (
		ssc            = newTestStorageSC()
		balances       = newTestBalances(b, true)
		client         = newClient(100000*x10, balances)
		tp, exp  int64 = 0, int64(toSeconds(time.Hour))

		tx    *transaction.Transaction
		blobs []*Client
		conf  *Config
		err   error
	)

	defer balances.mpts.Close()

	balances.skipMerge = true
	conf = setConfig(b, balances)

	// 1. just create 1000 blobbers
	b.Log("add 1k blobbers")
	tp += 1
	balances.skipMerge = true // don't merge transactions for now
	_, blobs = addAllocation(b, ssc, client, tp, exp, 1000, balances)

	// 2. and 1000 corresponding validators
	b.Log("add 1k corresponding validators")
	for _, bl := range blobs {
		tp += 1
		tx = newTransaction(bl.id, ssc.ID, 0, tp)
		_, err = ssc.addValidator(tx, bl.addValidatorRequest(b), balances)
		require.NoError(b, err)
	}

	// build blobbers/validators mapping id -> instance
	var blobsMap = make(map[string]*Client, len(blobs))
	for _, b := range blobs {
		blobsMap[b.id] = b
	}

	conf.MinAllocSize = 1 * KB
	mustSave(b, scConfigKey(ADDRESS), conf, balances)

	// 3. create 1000 allocations
	b.Log("add 1k allocations")
	var allocs []string
	for i := 0; i < 1000; i++ {

		var nar = new(newAllocationRequest)
		nar.DataShards = 10
		nar.ParityShards = 10
		nar.Expiration = common.Timestamp(exp)
		nar.Owner = client.id
		nar.OwnerPublicKey = client.pk
		nar.ReadPriceRange = PriceRange{1 * x10, 10 * x10}
		nar.WritePriceRange = PriceRange{2 * x10, 20 * x10}
		nar.Size = 1 * KB
		nar.MaxChallengeCompletionTime = 200 * time.Hour

		var resp, err = nar.callNewAllocReq(b, client.id, 15*x10, ssc, tp,
			balances)
		require.NoError(b, err)

		var deco StorageAllocation
		require.NoError(b, deco.Decode([]byte(resp)))

		allocs = append(allocs, deco.ID)
	}

	// 4. "write" 10 files for every one of the allocations
	b.Log("write 10k files")
	for _, allocID := range allocs {
		var alloc *StorageAllocation
		alloc, err = ssc.getAllocation(allocID, balances)
		require.NoError(b, err)
		alloc.Stats = new(StorageAllocationStats)
		alloc.Stats.NumWrites += 10 // 10 files
		for _, d := range alloc.BlobberAllocs {
			d.AllocationRoot = "allocation-root"
		}
		_, err = balances.InsertTrieNode(alloc.GetKey(ssc.ID), alloc)
		require.NoError(b, err)
	}

	// 5. merge all transactions into p node db
	b.Log("merge all into p node db")
	balances.skipMerge = false
	balances.mpts.merge(b)

	b.ResetTimer()
	b.Log("start benchmark")

	valids, err := getValidatorsList(balances)
	require.NoError(b, err)

	// 6. add challenge for an allocation and verify it (successive case)
	b.Run("verify challenge", func(b *testing.B) {

		for i := 0; i < b.N; i++ {

			var (
				allocID   string
				blobberID string
				input     []byte
				tx        *transaction.Transaction
			)

			b.StopTimer()
			{
				// 6.1 generate challenge
				tp += 1
				allocID = allocs[i%len(allocs)]

				var (
					r     = rand.New(rand.NewSource(tp))
					alloc *StorageAllocation
				)
				alloc, err = ssc.getAllocation(allocID, balances)
				require.NoError(b, err)

				// 6.3 keep for the benchmark
				blobberID = alloc.BlobberAllocs[rand.Intn(len(alloc.BlobberAllocs))].BlobberID

				var (
					challID    = encryption.Hash(fmt.Sprintf("chall-%d", tp))
					challBytes string
				)

				storageChall, err := ssc.getStorageChallenge(challID, balances)
				require.NoError(b, err)
				allocChall, err := ssc.getAllocationChallenges(allocID, balances)
				require.NoError(b, err)
				blobberChall, err := ssc.getBlobberChallenges(blobberID, balances)
				require.NoError(b, err)
				challInfo := &StorageChallengeResponse{
					StorageChallenge: storageChall,
				}

				err = ssc.addChallenge(
					alloc,
					storageChall,
					allocChall,
					blobberChall,
					challInfo,
					balances)

				require.NoError(b, err)

				var chall StorageChallenge
				mustDecode(b, []byte(challBytes), &chall)

				// 6.2 create challenge response (with tickets)
				tp += 1

				var challResp ChallengeResponse
				challResp.ID = chall.ID

				var validators []ValidationPartitionNode
				err = valids.GetRandomItems(balances, r, &validators)
				require.NoError(b, err)
				for _, v := range validators {
					var vx = blobsMap[v.Id]
					challResp.ValidationTickets = append(
						challResp.ValidationTickets,
						vx.validTicket(b, chall.ID, chall.BlobberID, true, tp),
					)
				}

				// 6.3 keep for the benchmark
				//blobberID = chall.BlobberID

				// 6.4 prepare transaction
				tp += 1
				tx = newTransaction(blobberID, ssc.ID, 0, tp)
				input = mustEncode(b, challResp)
				balances.setTransaction(b, tx)
			}
			b.StartTimer()

			var resp string
			resp, err = ssc.verifyChallenge(tx, input, balances)
			require.NoError(b, err)
			require.Equal(b, resp, "challenge passed by blobber")
		}
		b.ReportAllocs()
	})

}
