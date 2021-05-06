package util

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"

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

func (tv testValue) Encode() (p []byte) {
	var err error
	if p, err = json.Marshal(tv); err != nil {
		panic(err)
	}
	return
}

func (tv *testValue) Decode(p []byte) error {
	return json.Unmarshal(p, tv)
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
		key = Key(node.GetHashBytes())
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
				switch (i + j) % 3 {
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

	const N = 100

	var (
		mndb, cleanup = newPNodeDB(t)

		kvs  = getTestKeyValues(N)
		back = context.Background()

		node Node
		err  error
	)
	defer cleanup()

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

	var pndb, cleanup = newPNodeDB(t)
	defer cleanup()

	t.Run("pnode_db", func(t *testing.T) {
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

// launch test with -race or make the case sense
func TestNodeDB_parallel(t *testing.T) {

	if testing.Short() {
		t.Skip("skip the parallel tests due to -short flag")
	}

	const (
		parallel = 100
		n        = 100
	)

	var (
		pndb, cleanup = newPNodeDB(t)

		mndb = NewMemoryNodeDB()
		lndb = NewLevelNodeDB(mndb, pndb, false)

		kvs         = getTestKeyValues(n)
		keys, nodes = getTestKeysAndValues(kvs)
	)
	defer cleanup()

	for _, ndb := range []NodeDB{
		pndb,
		lndb,
		mndb,
	} {
		t.Run(fmt.Sprintf("%T", ndb), func(t *testing.T) {
			for i := 0; i < parallel; i++ {
				t.Run("parallel access", func(t *testing.T) {
					t.Parallel()
					for j := 0; j < len(kvs); j++ {
						var err error
						switch (i + j) % 6 {
						case 0:
							_, err = ndb.GetNode(kvs[j].key)
							require.NoError(t, noNodeNotFound(err))
						case 1:
							err = ndb.PutNode(kvs[j].key, kvs[j].node)
							require.NoError(t, err)
						case 2:
							require.NoError(t, ndb.DeleteNode(kvs[j].key))
						case 3:
							_, err = ndb.MultiGetNode(keys)
							require.NoError(t, noNodeNotFound(err))
						case 4:
							require.NoError(t, ndb.MultiPutNode(keys, nodes))
						case 5:
							require.NoError(t, ndb.MultiDeleteNode(keys))
						}
					}
				})
			}
		})
	}

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
