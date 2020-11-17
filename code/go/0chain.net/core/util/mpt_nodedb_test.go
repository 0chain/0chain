package util

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	//"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"0chain.net/core/logging"
	"go.uber.org/zap"
)

func init() {
	logging.Logger = zap.NewNop() // suppress all logs for the tests
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

// based on full previous with empty current
func newTestLevelDB(t *testing.T, n int) (mndb NodeDB) {
	var (
		prev, next = NewMemoryNodeDB(), NewMemoryNodeDB()
		kvs        = getTestKeyValues(n)
	)
	for _, kv := range kvs {
		require.NoError(t, prev.PutNode(kv.key, kv.node))
	}
	return NewLevelNodeDB(prev, next, true)
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

	var keys, nodes = getTestKeysAndValues(kvs)
	nodes = nil // reset the list for the next tests

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
		mndb.PruneBelowVersion(back, Sequence(200))
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

//
//
//

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

	var keys, nodes = getTestKeysAndValues(kvs)
	nodes = nil // reset the list for the next tests

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
		lndb.PruneBelowVersion(back, Sequence(200))
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
					lndb.GetNode(kvs[j].key)
				case 1:
					lndb.PutNode(kvs[j].key, kvs[j].node)
				case 2:
					lndb.DeleteNode(kvs[j].key)
				}
			}

			requireOneOf(t, lndb.GetCurrent(), curr, fcurr, prev, fprev)
			requireOneOf(t, lndb.GetPrev(), curr, fcurr, prev, fprev)
		})
	}

}

func TestMergeState(t *testing.T) {

	const (
		parallel = 100
		n        = 100
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

}
