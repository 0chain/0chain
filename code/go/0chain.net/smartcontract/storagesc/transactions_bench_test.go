package storagesc

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
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
	if mpts.pndb != nil {
		mpts.pndb.Flush()
	}
	if mpts.dir != "" {
		err = os.RemoveAll(mpts.dir)
	}
	return
}

func (mpts *mptStore) merge(tb testing.TB) {

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

				blobs []*Client
				conf  *scConfig
				err   error
			)

			defer balances.mpts.Close()

			conf = setConfig(b, balances)

			// call the addAllocation to create and stake n blobbers, the resulting
			// allocation will not be used
			tp += 100
			_, blobs = addAllocation(b, ssc, client, tp, exp, n, balances)
			_ = blobs

			conf.MinAllocSize = 1 * KB
			mustSave(b, scConfigKey(ADDRESS), conf, balances)

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
