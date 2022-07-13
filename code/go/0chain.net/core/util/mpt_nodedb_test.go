package util

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/0chain/gorocksdb"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

type keyNode struct {
	key  Key
	node Node
}

type testValue string

func (tv testValue) MarshalMsg([]byte) ([]byte, error) {
	return []byte(tv), nil
}

func (tv *testValue) UnmarshalMsg(p []byte) ([]byte, error) {
	*tv = testValue(p)
	return nil, nil
}

func getTestKeyValues(n int) (kns []keyNode) {

	kns = make([]keyNode, 0, n)

	for i := 0; i < n; i++ {
		var (
			tv   = testValue(fmt.Sprintf("test_value<%d>", i))
			node = NewValueNode()

			key Key
		)

		node.SetValue(&tv)
		key = node.GetHashBytes()
		kns = append(kns, keyNode{key, node})
	}

	return
}

func getTestKeysAndValues(kvs []keyNode) (keys []Key, nodes []Node) {
	keys = make([]Key, 0, len(kvs))
	nodes = make([]Node, 0, len(kvs))
	for _, kv := range kvs {
		keys = append(keys, kv.key)
		nodes = append(nodes, kv.node)
	}
	return
}

func TestMemoryNodeDB_Full(t *testing.T) {

	const N = 100

	var (
		mndb = NewMemoryNodeDB()
		kvs  = getTestKeyValues(N)
		back = context.Background()

		node Node
		err  error
	)

	require.NotNil(t, mndb)
	require.NotNil(t, mndb.mutex)
	require.NotNil(t, mndb.Nodes)

	//
	// get / put / delete
	//

	t.Run("get_put_delete", func(t *testing.T) {
		require.Zero(t, mndb.Size(back))

		// node not found
		for _, kv := range kvs {
			node, err = mndb.GetNode(kv.key)
			require.Nil(t, node)
			require.Equal(t, ErrNodeNotFound, err)

			require.NoError(t, mndb.DeleteNode(kv.key))
		}

		// insert
		for _, kv := range kvs {
			require.NoError(t, mndb.PutNode(kv.key, kv.node))
		}
		require.EqualValues(t, N, mndb.Size(back))

		// double insert
		for _, kv := range kvs {
			require.NoError(t, mndb.PutNode(kv.key, kv.node))
		}
		require.EqualValues(t, N, mndb.Size(back))

		// found
		for i, kv := range kvs {
			node, err = mndb.GetNode(kv.key)
			require.NoError(t, err)
			require.Equalf(t, kv.node, node, "wrong value: %d", i)
		}

		// delete
		for _, kv := range kvs {
			require.NoError(t, mndb.DeleteNode(kv.key))
		}
		require.Zero(t, mndb.Size(back))
	})

	//
	// multi get / put / delete
	//

	var keys, _ = getTestKeysAndValues(kvs)
	nodes := make([]Node, 0) // reset the list for the next tests

	t.Run("multi_get_put_delete", func(t *testing.T) {
		// node not found
		nodes, err = mndb.MultiGetNode(keys)
		require.Nil(t, nodes)
		require.Equal(t, ErrNodeNotFound, err)

		require.NoError(t, mndb.MultiDeleteNode(keys))
		require.Zero(t, mndb.Size(back))

		// insert
		for _, kv := range kvs {
			nodes = append(nodes, kv.node)
		}

		err = mndb.MultiPutNode(keys, nodes)
		require.NoError(t, err)
		require.EqualValues(t, N, mndb.Size(back))

		// double insert
		err = mndb.MultiPutNode(keys, nodes)
		require.NoError(t, err)
		require.EqualValues(t, N, mndb.Size(back))

		// found
		nodes, err = mndb.MultiGetNode(keys)
		require.NoError(t, err)
		require.Len(t, nodes, len(kvs))
		for i, kv := range kvs {
			require.Equalf(t, kv.node, nodes[i], "wrong node: %d", i)
		}

		// delete
		require.NoError(t, mndb.MultiDeleteNode(keys))
		require.Zero(t, mndb.Size(back))
	})

	t.Run("iterate", func(t *testing.T) {
		keys, nodes = getTestKeysAndValues(kvs)

		require.NoError(t, mndb.MultiPutNode(keys, nodes))
		require.EqualValues(t, N, mndb.Size(back))

		var kvm = make(map[string]Node)

		var i int
		err = mndb.Iterate(back,
			func(ctx context.Context, key Key, node Node) (err error) {
				kvm[string(key)] = node
				i++
				return
			})
		require.NoError(t, err)       // no error
		require.Len(t, kvm, len(kvs)) // all keys
		require.Equal(t, len(kvm), i) // no double handling

		for _, kv := range kvs {
			require.Equal(t, kv.node, kvm[string(kv.key)])
		}

		// iterate handler returns error
		var testError = errors.New("test error")
		i = 0
		err = mndb.Iterate(back,
			func(ctx context.Context, key Key, node Node) (err error) {
				i++
				return testError
			})
		require.Equal(t, testError, err)
		require.Equal(t, 1, i)
	})

	t.Run("prune_below_version", func(t *testing.T) {
		require.NoError(t, mndb.PruneBelowVersion(back, Sequence(100)))
		require.Zero(t, mndb.Size(back))
		for _, kv := range kvs {
			kv.node.SetVersion(300)
		}
		keys, nodes = getTestKeysAndValues(kvs)
		require.NoError(t, mndb.MultiPutNode(keys, nodes))
		if err := mndb.PruneBelowVersion(back, Sequence(200)); err != nil {
			t.Fatal(err)
		}
		require.EqualValues(t, N, mndb.Size(back))
	})

	// TODO (sfdx): additional test for the MemoryDB.Validate
	//
	// additional Validate test for the Memory DB
	//
	// var mx, ok = mndb.(*MemoryNodeDB)
	// if !ok {
	// 	return // done
	// }
	//
	// t.Run("validate", func(t *testing.T) {
	// 	var root = mx.ComputeRoot()
	// 	require.NotNil(t, root)
	// 	require.NoError(t, mx.Validate(root))
	// })

}

func TestLevelNodeDB_Full(t *testing.T) {

	const N = 100

	var (
		prev, curr = NewMemoryNodeDB(), NewMemoryNodeDB()
		lndb       = NewLevelNodeDB(curr, prev, true)
		kvs        = getTestKeyValues(N)
		back       = context.Background()

		node Node
		err  error
	)

	//
	// get / put / delete
	//

	t.Run("get_put_delete", func(t *testing.T) {
		require.Zero(t, lndb.Size(back))

		// node not found
		for _, kv := range kvs {
			node, err = lndb.GetNode(kv.key)
			require.Nil(t, node)
			require.Equal(t, ErrNodeNotFound, err)

			require.NoError(t, lndb.DeleteNode(kv.key))
		}
		require.Zero(t, lndb.Size(back))
		require.Len(t, lndb.DeletedNodes, 0)

		// insert
		for _, kv := range kvs {
			require.NoError(t, lndb.PutNode(kv.key, kv.node))
		}
		require.EqualValues(t, N, lndb.Size(back))
		require.Len(t, lndb.DeletedNodes, 0)

		// double insert
		for _, kv := range kvs {
			require.NoError(t, lndb.PutNode(kv.key, kv.node))
		}
		require.EqualValues(t, N, lndb.Size(back))
		require.Len(t, lndb.DeletedNodes, 0)

		// found
		for i, kv := range kvs {
			node, err = lndb.GetNode(kv.key)
			require.NoError(t, err)
			require.Equalf(t, kv.node, node, "wrong value: %d", i)
		}

		// delete
		for _, kv := range kvs {
			require.NoError(t, lndb.DeleteNode(kv.key))
		}
		require.Zero(t, lndb.Size(back))
		require.Len(t, lndb.DeletedNodes, 0)

		// insert in previous
		for _, kv := range kvs {
			require.NoError(t, prev.PutNode(kv.key, kv.node))
		}
		require.EqualValues(t, N, lndb.Size(back))
		require.Len(t, lndb.DeletedNodes, 0)

		// insert over existing
		for _, kv := range kvs {
			require.NoError(t, lndb.PutNode(kv.key, kv.node))
		}
		require.EqualValues(t, N*2, lndb.Size(back)) // x2 (?!)
		require.Len(t, lndb.DeletedNodes, 0)

		// delete in current
		for _, kv := range kvs {
			require.NoError(t, lndb.DeleteNode(kv.key))
		}
		require.EqualValues(t, N, lndb.Size(back))
		require.Len(t, lndb.DeletedNodes, 0)

		// delete propagated
		for _, kv := range kvs {
			require.NoError(t, lndb.DeleteNode(kv.key))
		}
		require.Zero(t, lndb.Size(back))
		require.Len(t, lndb.DeletedNodes, 0)

		// mark as deleted
		for _, kv := range kvs {
			require.NoError(t, lndb.DeleteNode(kv.key))
		}
		require.Zero(t, lndb.Size(back))
		// should be 100 if deletes is not propagated
		require.Len(t, lndb.DeletedNodes, 0)
	})

	//
	// multi get / put / delete
	//

	var keys, _ = getTestKeysAndValues(kvs)
	nodes := make([]Node, 0) // reset the list for the next tests

	t.Run("multi_get_put_delete", func(t *testing.T) {
		// node not found
		nodes, err = lndb.MultiGetNode(keys)
		require.Nil(t, nodes)
		require.Equal(t, ErrNodeNotFound, err)

		require.NoError(t, lndb.MultiDeleteNode(keys))
		require.Zero(t, lndb.Size(back))

		// insert
		for _, kv := range kvs {
			nodes = append(nodes, kv.node)
		}

		err = lndb.MultiPutNode(keys, nodes)
		require.NoError(t, err)
		require.EqualValues(t, N, lndb.Size(back))

		// double insert
		err = lndb.MultiPutNode(keys, nodes)
		require.NoError(t, err)
		require.EqualValues(t, N, lndb.Size(back))

		// found
		nodes, err = lndb.MultiGetNode(keys)
		require.NoError(t, err)
		require.Len(t, nodes, len(kvs))
		for i, kv := range kvs {
			require.Equalf(t, kv.node, nodes[i], "wrong node: %d", i)
		}

		// delete
		require.NoError(t, lndb.MultiDeleteNode(keys))
		require.Zero(t, lndb.Size(back))
	})

	t.Run("iterate", func(t *testing.T) {
		keys, nodes = getTestKeysAndValues(kvs)

		require.NoError(t, lndb.MultiPutNode(keys, nodes))
		require.EqualValues(t, N, lndb.Size(back))

		var kvm = make(map[string]Node)

		var i int
		err = lndb.Iterate(back,
			func(ctx context.Context, key Key, node Node) (err error) {
				kvm[string(key)] = node
				i++
				return
			})
		require.NoError(t, err)       // no error
		require.Len(t, kvm, len(kvs)) // all keys
		require.Equal(t, len(kvm), i) // no double handling

		for _, kv := range kvs {
			require.Equal(t, kv.node, kvm[string(kv.key)])
		}

		// iterate handler returns error
		var testError = errors.New("test error")
		i = 0
		err = lndb.Iterate(back,
			func(ctx context.Context, key Key, node Node) (err error) {
				i++
				return testError
			})
		require.Equal(t, testError, err)
		require.Equal(t, 1, i)
	})

	// For the LevelDB PruneBelowVersion is not implemented and does nothing.
	t.Run("prune_below_version", func(t *testing.T) {
		require.NoError(t, lndb.PruneBelowVersion(back, Sequence(100)))
		require.EqualValues(t, N, lndb.Size(back))
		for _, kv := range kvs {
			kv.node.SetVersion(300)
		}
		keys, nodes = getTestKeysAndValues(kvs)
		require.NoError(t, lndb.MultiPutNode(keys, nodes))
		if err := lndb.PruneBelowVersion(back, Sequence(200)); err != nil {
			t.Fatal(err)
		}
		require.EqualValues(t, N, lndb.Size(back))
	})

}

func requireOneOf(t *testing.T, db NodeDB, list ...NodeDB) {
	for _, dx := range list {
		if db == dx {
			return
		}
	}
	require.Fail(t, "unexpected or missing underlying DB")
}

// launch test with -race or make the case sense
func TestLevelNodeDB_Current_Prev_Rebase(t *testing.T) {

	if testing.Short() {
		t.Skip("skip the parallel tests due to -short flag")
	}

	const (
		parallel = 100
		n        = 100
	)

	var (
		prev, curr = NewMemoryNodeDB(), NewMemoryNodeDB()
		lndb       = NewLevelNodeDB(curr, prev, true)
		kvs        = getTestKeyValues(n)
		// back       = context.Background()

		fprev, fcurr = NewMemoryNodeDB(), NewMemoryNodeDB()
	)

	for _, kv := range kvs {
		require.NoError(t, fprev.PutNode(kv.key, kv.node))
		require.NoError(t, fcurr.PutNode(kv.key, kv.node))
	}

	for i := 0; i < parallel; i++ {
		t.Run("parallel access", func(t *testing.T) {
			t.Parallel()

			lndb.RebaseCurrentDB(fcurr)

			if lndb.GetPrev() == prev {
				lndb.SetPrev(fprev)
			} else {
				lndb.SetPrev(prev)
			}

			for j := 0; j < len(kvs); j++ {
				switch j % 3 {
				case 0:
					if _, err := lndb.GetNode(kvs[j].key); err != nil {
						t.Fatal(err)
					}
				case 1:
					if err := lndb.PutNode(kvs[j].key, kvs[j].node); err != nil {
						t.Fatal(err)
					}
				case 2:
					if err := lndb.DeleteNode(kvs[j].key); err != nil {
						t.Fatal(err)
					}
				}
			}

			requireOneOf(t, lndb.GetCurrent(), curr, fcurr, prev, fprev)
			requireOneOf(t, lndb.GetPrev(), curr, fcurr, prev, fprev)
		})
	}

}

func TestPNodeDB_Full(t *testing.T) {

	tm := time.Now()
	const N = 100

	var (
		mndb, cleanup = newPNodeDB(t)

		kvs  = getTestKeyValues(N)
		back = context.Background()

		node Node
		err  error
	)
	defer cleanup()

	fmt.Println(time.Since(tm))
	//
	// get / put / delete
	//

	t.Run("get_put_delete", func(t *testing.T) {
		require.Zero(t, mndb.Size(back))

		// node not found
		for _, kv := range kvs {
			node, err = mndb.GetNode(kv.key)
			require.Nil(t, node)
			require.Equal(t, ErrNodeNotFound, err)

			require.NoError(t, mndb.DeleteNode(kv.key))
		}

		// insert
		for _, kv := range kvs {
			require.NoError(t, mndb.PutNode(kv.key, kv.node))
		}
		require.EqualValues(t, N, mndb.Size(back))

		// double insert
		for _, kv := range kvs {
			require.NoError(t, mndb.PutNode(kv.key, kv.node))
		}
		require.EqualValues(t, N, mndb.Size(back))

		// found
		for i, kv := range kvs {
			node, err = mndb.GetNode(kv.key)
			require.NoError(t, err)
			require.EqualValuesf(t, kv.node.Encode(), node.Encode(),
				"wrong value: %d", i)
		}

		// delete
		for _, kv := range kvs {
			require.NoError(t, mndb.DeleteNode(kv.key))
		}
		require.Zero(t, mndb.Size(back))
	})

	//
	// multi get / put / delete
	//

	var keys, _ = getTestKeysAndValues(kvs)
	nodes := make([]Node, 0) // reset the list for the next tests

	t.Run("multi_get_put_delete", func(t *testing.T) {
		// node not found
		nodes, err = mndb.MultiGetNode(keys)
		require.Nil(t, nodes)
		require.Equal(t, ErrNodeNotFound, err)

		require.NoError(t, mndb.MultiDeleteNode(keys))
		require.Zero(t, mndb.Size(back))

		// insert
		for _, kv := range kvs {
			nodes = append(nodes, kv.node)
		}

		err = mndb.MultiPutNode(keys, nodes)
		require.NoError(t, err)
		require.EqualValues(t, N, mndb.Size(back))

		// double insert
		err = mndb.MultiPutNode(keys, nodes)
		require.NoError(t, err)
		require.EqualValues(t, N, mndb.Size(back))

		// found
		nodes, err = mndb.MultiGetNode(keys)
		require.NoError(t, err)
		require.Len(t, nodes, len(kvs))
		for i, kv := range kvs {
			require.EqualValuesf(t, kv.node.Encode(), nodes[i].Encode(),
				"wrong node: %d", i)
		}

		// delete
		require.NoError(t, mndb.MultiDeleteNode(keys))
		require.Zero(t, mndb.Size(back))
	})

	t.Run("iterate", func(t *testing.T) {
		keys, nodes = getTestKeysAndValues(kvs)

		require.NoError(t, mndb.MultiPutNode(keys, nodes))
		require.EqualValues(t, N, mndb.Size(back))

		var kvm = make(map[string]Node)

		var i int
		err = mndb.Iterate(back,
			func(ctx context.Context, key Key, node Node) (err error) {
				kvm[string(key)] = node
				i++
				return
			})
		require.NoError(t, err)       // no error
		require.Len(t, kvm, len(kvs)) // all keys
		require.Equal(t, len(kvm), i) // no double handling

		for _, kv := range kvs {
			require.NotZero(t, kvm[string(kv.key)])
			require.Equal(t, kv.node.Encode(), kvm[string(kv.key)].Encode())
		}

		// iterate handler returns error
		var testError = errors.New("test error")
		i = 0
		err = mndb.Iterate(back,
			func(ctx context.Context, key Key, node Node) (err error) {
				i++
				return testError
			})
		require.Equal(t, testError, err)
		require.Equal(t, 1, i)

		err = mndb.MultiDeleteNode(keys)
		require.NoError(t, err)
	})
}

func TestMergeState(t *testing.T) {

	const (
		n = 100
	)

	var (
		fmdb, tmdb = NewMemoryNodeDB(), NewMemoryNodeDB()
		kvs        = getTestKeyValues(n)
		back       = context.Background()
	)

	for _, kv := range kvs {
		require.NoError(t, fmdb.PutNode(kv.key, kv.node))
	}

	t.Run("memory_db", func(t *testing.T) {
		require.NoError(t, MergeState(back, fmdb, tmdb))
		require.EqualValues(t, n, tmdb.Size(back))
	})

	var (
		prev, curr = NewMemoryNodeDB(), NewMemoryNodeDB()
		lndb       = NewLevelNodeDB(curr, prev, true)
	)

	t.Run("level_db", func(t *testing.T) {
		require.NoError(t, MergeState(back, fmdb, lndb))
		require.EqualValues(t, n, lndb.Size(back))
	})

	t.Run("pnode_db", func(t *testing.T) {
		var pndb, cleanup = newPNodeDB(t)
		defer cleanup()

		require.NoError(t, MergeState(back, fmdb, pndb))
		require.EqualValues(t, n, pndb.Size(back))
	})
}

func noNodeNotFound(err error) error {
	if err == ErrNodeNotFound {
		return nil
	}
	return err
}

func TestRaceMemoryNodeDB_Full(t *testing.T) {

	const N = 100

	var (
		mndb = NewMemoryNodeDB()
		kvs  = getTestKeyValues(N)
		back = context.Background()
	)

	require.NotNil(t, mndb)
	require.NotNil(t, mndb.mutex)
	require.NotNil(t, mndb.Nodes)

	//
	// get / put / delete
	//

	var wg sync.WaitGroup

	t.Run("get_put_delete", func(t *testing.T) {
		require.Zero(t, mndb.Size(back))

		// node not found
		for i := range kvs {
			kv := kvs[i]
			wg.Add(1)
			go func() {
				node, err := mndb.GetNode(kv.key)
				require.Nil(t, node)
				require.Equal(t, ErrNodeNotFound, err)

				require.NoError(t, mndb.DeleteNode(kv.key))
				wg.Done()
			}()
		}
		wg.Wait()

		// insert
		for i := range kvs {
			kv := kvs[i]
			wg.Add(1)
			go func() {
				require.NoError(t, mndb.PutNode(kv.key, kv.node))
				wg.Done()
			}()
		}
		wg.Wait()
		require.EqualValues(t, N, mndb.Size(back))

		// double insert
		for i := range kvs {
			kv := kvs[i]
			wg.Add(1)
			go func() {
				require.NoError(t, mndb.PutNode(kv.key, kv.node))
				wg.Done()
			}()
		}
		wg.Wait()
		require.EqualValues(t, N, mndb.Size(back))

		// found
		for i := range kvs {
			kv := kvs[i]
			wg.Add(1)
			go func() {
				node, err := mndb.GetNode(kv.key)
				require.NoError(t, err)
				// require.Equalf(t, kv.node, node, "wrong value: %d", i) // i captured by next iteration tested with const 0 below
				require.Equalf(t, kv.node, node, "wrong value: %d", 0)
				wg.Done()
			}()
		}
		wg.Wait()

		// delete
		for i := range kvs {
			kv := kvs[i]
			wg.Add(1)
			go func() {
				require.NoError(t, mndb.DeleteNode(kv.key))
				wg.Done()
			}()
		}
		wg.Wait()
		require.Zero(t, mndb.Size(back))
	})
}

func TestRaceLevelNodeDB_Full(t *testing.T) {

	const N = 100

	var (
		prev, curr = NewMemoryNodeDB(), NewMemoryNodeDB()
		lndb       = NewLevelNodeDB(curr, prev, true)
		kvs        = getTestKeyValues(N)
		back       = context.Background()
	)

	//
	// get / put / delete
	//
	var wg sync.WaitGroup
	t.Run("get_put_delete", func(t *testing.T) {
		require.Zero(t, lndb.Size(back))

		// node not found
		for i := range kvs {
			kv := kvs[i]
			wg.Add(1)
			go func() {

				node, err := lndb.GetNode(kv.key)
				require.Nil(t, node)
				require.Equal(t, ErrNodeNotFound, err)

				require.NoError(t, lndb.DeleteNode(kv.key))
				wg.Done()
			}()
		}
		wg.Wait()
		require.Zero(t, lndb.Size(back))
		require.Len(t, lndb.DeletedNodes, 0)

		// insert
		for i := range kvs {
			kv := kvs[i]
			wg.Add(1)
			go func() {
				require.NoError(t, lndb.PutNode(kv.key, kv.node))
				wg.Done()
			}()
		}
		wg.Wait()
		require.EqualValues(t, N, lndb.Size(back))
		require.Len(t, lndb.DeletedNodes, 0)

		// double insert
		for i := range kvs {
			kv := kvs[i]
			wg.Add(1)
			go func() {
				require.NoError(t, lndb.PutNode(kv.key, kv.node))
				wg.Done()
			}()
		}
		wg.Wait()
		require.EqualValues(t, N, lndb.Size(back))
		require.Len(t, lndb.DeletedNodes, 0)

		// found
		for i := range kvs {
			kv := kvs[i]
			wg.Add(1)
			go func() {
				node, err := lndb.GetNode(kv.key)
				require.NoError(t, err)
				// require.Equalf(t, kv.node, node, "wrong value: %d", i) // i captured by next iteration using constant
				require.Equalf(t, kv.node, node, "wrong value: %d", 0)
				wg.Done()
			}()
		}
		wg.Wait()
		// delete
		for i := range kvs {
			kv := kvs[i]
			wg.Add(1)
			go func() {
				require.NoError(t, lndb.DeleteNode(kv.key))
				wg.Done()
			}()
		}
		wg.Wait()
		require.Zero(t, lndb.Size(back))
		require.Len(t, lndb.DeletedNodes, 0)

		// insert in previous
		for i := range kvs {
			kv := kvs[i]
			wg.Add(1)
			go func() {
				require.NoError(t, prev.PutNode(kv.key, kv.node))
				wg.Done()
			}()
		}
		wg.Wait()
		require.EqualValues(t, N, lndb.Size(back))
		require.Len(t, lndb.DeletedNodes, 0)

		// insert over existing
		for i := range kvs {
			kv := kvs[i]
			wg.Add(1)
			go func() {
				require.NoError(t, lndb.PutNode(kv.key, kv.node))
				wg.Done()
			}()
		}
		wg.Wait()
		require.EqualValues(t, N*2, lndb.Size(back)) // x2 (?!)
		require.Len(t, lndb.DeletedNodes, 0)

		// delete in current
		for i := range kvs {
			kv := kvs[i]
			wg.Add(1)
			go func() {
				require.NoError(t, lndb.DeleteNode(kv.key))
				wg.Done()
			}()

		}
		wg.Wait()
		require.EqualValues(t, N, lndb.Size(back))
		require.Len(t, lndb.DeletedNodes, 0)

		// delete propagated
		for i := range kvs {
			kv := kvs[i]
			wg.Add(1)
			go func() {
				require.NoError(t, lndb.DeleteNode(kv.key))
				wg.Done()
			}()
		}
		wg.Wait()
		require.Zero(t, lndb.Size(back))
		require.Len(t, lndb.DeletedNodes, 0)

		// mark as deleted
		for i := range kvs {
			kv := kvs[i]
			wg.Add(1)
			go func() {
				require.NoError(t, lndb.DeleteNode(kv.key))
				wg.Done()
			}()
		}
		wg.Wait()
		require.Zero(t, lndb.Size(back))
		// should be 100 if deletes is not propagated
		require.Len(t, lndb.DeletedNodes, 0)
	})
}

func TestMemoryNodeDB_reachable(t *testing.T) {
	t.Parallel()

	mndb := NewMemoryNodeDB()

	fn := NewFullNode(nil)
	bi := fn.indexToByte(0)
	fn.Children[fn.index(bi)] = []byte("children")

	type fields struct {
		Nodes map[StrKey]Node
		mutex *sync.RWMutex
	}
	type args struct {
		node  Node
		node2 Node
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "Test_MemoryNodeDB_reachable_Node_Not_Found_OK",
			fields: fields{
				Nodes: mndb.Nodes,
				mutex: &sync.RWMutex{},
			},
			args: args{node: NewExtensionNode([]byte("path"), []byte("key"))},
			want: false,
		},
		{
			name: "Test_MemoryNodeDB_reachable_Node_Not_Found_OK",
			fields: fields{
				Nodes: mndb.Nodes,
				mutex: &sync.RWMutex{},
			},
			args: args{node: fn},
			want: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mndb := &MemoryNodeDB{
				Nodes: tt.fields.Nodes,
				mutex: tt.fields.mutex,
			}
			if got := mndb.reachable(tt.args.node, tt.args.node2); got != tt.want {
				t.Errorf("reachable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLevelNodeDB_GetDBVersion(t *testing.T) {
	t.Parallel()

	type fields struct {
		mu               *sync.RWMutex
		current          NodeDB
		prev             NodeDB
		PropagateDeletes bool
		DeletedNodes     map[StrKey]bool
		version          int64
		versions         []int64
	}
	tests := []struct {
		name   string
		fields fields
		want   int64
	}{
		{
			name:   "Test_LevelNodeDB_GetDBVersion_OK",
			fields: fields{version: 1, mu: &sync.RWMutex{}},
			want:   1,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			lndb := &LevelNodeDB{
				mutex:            &sync.RWMutex{},
				current:          tt.fields.current,
				prev:             tt.fields.prev,
				PropagateDeletes: tt.fields.PropagateDeletes,
				DeletedNodes:     tt.fields.DeletedNodes,
				version:          tt.fields.version,
			}
			if got := lndb.GetDBVersion(); got != tt.want {
				t.Errorf("GetDBVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLevelNodeDB_GetNode(t *testing.T) {
	t.Parallel()

	ndb := NewMemoryNodeDB()

	type fields struct {
		mu               *sync.RWMutex
		current          NodeDB
		prev             NodeDB
		PropagateDeletes bool
		DeletedNodes     map[StrKey]bool
		version          int64
		versions         []int64
	}
	type args struct {
		key Key
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    Node
		wantErr bool
	}{
		{
			name: "Test_LevelNodeDB_GetNode_ERR",
			fields: fields{
				mu:      &sync.RWMutex{},
				prev:    ndb,
				current: ndb,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			lndb := &LevelNodeDB{
				mutex:            &sync.RWMutex{},
				current:          tt.fields.current,
				prev:             tt.fields.prev,
				PropagateDeletes: tt.fields.PropagateDeletes,
				DeletedNodes:     tt.fields.DeletedNodes,
				version:          tt.fields.version,
			}
			got, err := lndb.GetNode(tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetNode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetNode() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLevelNodeDB_MultiPutNode(t *testing.T) {
	current, cleanup := newPNodeDB(t)
	defer cleanup()

	current.wo = gorocksdb.NewDefaultWriteOptions()
	current.wo.DisableWAL(true)
	current.wo.SetSync(true)

	type fields struct {
		mu               *sync.RWMutex
		current          NodeDB
		prev             NodeDB
		PropagateDeletes bool
		DeletedNodes     map[StrKey]bool
		version          int64
		versions         []int64
	}
	type args struct {
		keys  []Key
		nodes []Node
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:   "Test_LevelNodeDB_MultiPutNode_ERR",
			fields: fields{current: current, mu: &sync.RWMutex{}},
			args: args{
				keys:  []Key{Key("key")},
				nodes: []Node{NewFullNode(nil)},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lndb := &LevelNodeDB{
				mutex:            &sync.RWMutex{},
				current:          tt.fields.current,
				prev:             tt.fields.prev,
				PropagateDeletes: tt.fields.PropagateDeletes,
				DeletedNodes:     tt.fields.DeletedNodes,
				version:          tt.fields.version,
			}
			if err := lndb.MultiPutNode(tt.args.keys, tt.args.nodes); (err != nil) != tt.wantErr {
				t.Errorf("MultiPutNode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLevelNodeDB_Iterate(t *testing.T) {
	t.Parallel()

	ndm := NewMemoryNodeDB()

	type fields struct {
		mu               *sync.RWMutex
		current          NodeDB
		prev             NodeDB
		PropagateDeletes bool
		DeletedNodes     map[StrKey]bool
		version          int64
		versions         []int64
	}
	type args struct {
		ctx     context.Context
		handler NodeDBIteratorHandler
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "Test_LevelNodeDB_Iterate_OK",
			fields:  fields{current: ndm, prev: ndm, mu: &sync.RWMutex{}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			lndb := &LevelNodeDB{
				mutex:            &sync.RWMutex{},
				current:          tt.fields.current,
				prev:             tt.fields.prev,
				PropagateDeletes: tt.fields.PropagateDeletes,
				DeletedNodes:     tt.fields.DeletedNodes,
				version:          tt.fields.version,
			}
			if err := lndb.Iterate(tt.args.ctx, tt.args.handler); (err != nil) != tt.wantErr {
				t.Errorf("Iterate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMemoryNodeDB_Validate(t *testing.T) {
	t.Skip("need protect DebugMPTNode against concurrent access")
	t.Parallel()

	mndb := NewMemoryNodeDB()

	n1 := NewFullNode(&AState{balance: 1})

	type fields struct {
		Nodes map[StrKey]Node
		mutex *sync.RWMutex
	}
	type args struct {
		root Node
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "TestMemoryNodeDB_Validate_OK",
			fields: fields{
				Nodes: mndb.Nodes,
				mutex: &sync.RWMutex{},
			},
			args:    args{root: n1},
			wantErr: false,
		},
		{
			name: "TestMemoryNodeDB_Validate_ERR",
			fields: func() fields {
				mndb := NewMemoryNodeDB()
				mpt := NewMerklePatriciaTrie(mndb, 1, nil)

				n := &AState{balance: 2}
				_, err := mpt.Insert(Path("astate_2"), n)
				require.NoError(t, err)

				return fields{
					Nodes: mndb.Nodes,
					mutex: &sync.RWMutex{},
				}
			}(),
			args:    args{root: NewExtensionNode([]byte("path"), []byte("path"))},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mndb := &MemoryNodeDB{
				Nodes: tt.fields.Nodes,
				mutex: tt.fields.mutex,
			}
			if err := mndb.Validate(tt.args.root); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
