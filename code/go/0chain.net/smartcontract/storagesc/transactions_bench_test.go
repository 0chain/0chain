package storagesc

import (
	"context"
	"fmt"
	"io/ioutil"
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
	mpts.mpt = util.NewMerklePatriciaTrie(mpts.lndb, 1)          //
	mpts.mpt.SetRoot(root)
}

//
// 2) Also need to check how fast are the allocations created in storageSC
//    if there are 1000 blobbers.
//

//
// go test -v -timeout 1h -bench Benchmark_newAllocationRequest | prettybench
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

				conf *scConfig
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

				_, err = ssc.newAllocationRequest(tx, input, balances)
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
// go test -v -timeout 1h -bench Benchmark_generateChallenges | prettybench
//

func Benchmark_generateChallenges(b *testing.B) {

	var (
		ssc            = newTestStorageSC()
		balances       = newTestBalances(b, true)
		client         = newClient(100000*x10, balances)
		tp, exp  int64 = 0, int64(toSeconds(time.Hour))

		tx    *transaction.Transaction
		blobs []*Client
		conf  *scConfig
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
	var stats StorageStats
	stats.Stats = new(StorageAllocationStats)
	for _, allocID := range allocs {
		var alloc *StorageAllocation
		alloc, err = ssc.getAllocation(allocID, balances)
		require.NoError(b, err)
		alloc.Stats = new(StorageAllocationStats)
		alloc.Stats.NumWrites += 10 // 10 files
		for _, d := range alloc.BlobberDetails {
			d.AllocationRoot = "allocation-root"
		}
		_, err = balances.InsertTrieNode(alloc.GetKey(ssc.ID), alloc)
		require.NoError(b, err)
		stats.Stats.NumWrites += 10    // total stats
		stats.Stats.UsedSize += 1 * GB // fake size just for the challenges
	}
	_, err = balances.InsertTrieNode(stats.GetKey(ssc.ID), &stats)
	require.NoError(b, err)

	// 5. merge all transactions into p node db
	b.Log("merge all into p node db")
	balances.skipMerge = false
	balances.mpts.merge(b)

	b.ResetTimer()
	b.Log("start benchmark loop")

	var blk = new(block.Block)

	// 6. generate challenges
	for i := 0; i < b.N; i++ {

		b.StopTimer()
		{
			// revert the stats to allow generation
			tp += 1
			var statsb util.Serializable
			statsb, err = balances.GetTrieNode(stats.GetKey(ssc.ID))
			require.NoError(b, err)
			require.NoError(b, stats.Decode(statsb.Encode()))
			stats.LastChallengedSize = 0
			stats.LastChallengedTime = 0
			_, err = balances.InsertTrieNode(stats.GetKey(ssc.ID), &stats)
			require.NoError(b, err)

			tp += 1
			blk.PrevHash = encryption.Hash(fmt.Sprintf("block-%d", i))
			tx = newTransaction(client.id, ssc.ID, 0, tp)
			balances.setTransaction(b, tx)
		}
		b.StartTimer()

		err = ssc.generateChallenges(tx, blk, nil, balances)
		require.NoError(b, err)
	}
	b.ReportAllocs()

}
