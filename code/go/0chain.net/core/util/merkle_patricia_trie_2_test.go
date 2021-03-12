package util

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

type AState struct {
	balance int64
}

func (as *AState) Encode() []byte {
	return []byte(fmt.Sprintf("%v", as.balance))
}

func (as *AState) Decode(buf []byte) error {
	n, err := strconv.ParseInt(string(buf), 10, 63)
	if err != nil {
		return err
	}
	as.balance = n
	return nil
}

func newPNodeDB(t *testing.T) (pndb *PNodeDB, cleanup func()) {
	t.Helper()

	var dirname, err = ioutil.TempDir("", "mpt-pndb")
	require.NoError(t, err)

	pndb, err = NewPNodeDB(filepath.Join(dirname, "mpt"),
		filepath.Join(dirname, "log"))
	if err != nil {
		os.RemoveAll(dirname) //
		t.Fatal(err)          //
	}

	cleanup = func() {
		pndb.db.Close()
		os.RemoveAll(dirname)
	}

	return
}

func TestMerkleTreeSaveToDB(t *testing.T) {
	pndb, cleanup := newPNodeDB(t)
	defer cleanup()

	mpt := NewMerklePatriciaTrie(pndb, Sequence(2016))
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
	mpt2 := NewMerklePatriciaTrie(db, Sequence(2016))

	doStateValInsert(t, "add 100 to c1", mpt2, "0123456", 100)
	doStateValInsert(t, "add 1000 to c2", mpt2, "0123457", 1000)
	doStateValInsert(t, "add 1000 to c3", mpt2, "0123458", 1000000)
	doStateValInsert(t, "add 1000 to c4", mpt2, "0133458", 1000000000)

	printChanges(t, mpt2.GetChangeCollector())

	var err = mpt2.SaveChanges(pndb, false)
	if err != nil {
		t.Error(err)
	}

	err = mpt2.Iterate(context.TODO(), iterHandler(t),
		NodeTypeValueNode|NodeTypeLeafNode|NodeTypeFullNode|NodeTypeExtensionNode)

	mpt.SetRoot(mpt2.GetRoot())

	err = mpt.Iterate(context.TODO(), iterHandler(t),
		NodeTypeValueNode|NodeTypeFullNode|NodeTypeExtensionNode)
	if err != nil {
		t.Errorf("iterate error: %v", err)
	}
}

func TestMerkeTreePruning(t *testing.T) {
	pndb, cleanup := newPNodeDB(t)
	defer cleanup()

	mpt := NewMerklePatriciaTrie(pndb, Sequence(0))
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
	mpt2 := NewMerklePatriciaTrie(db, Sequence(0))
	origin := 2016
	roots := make([]Key, 0, 10)

	for i := int64(0); i < 1000; i++ {
		mpt2.ResetChangeCollector(mpt2.GetRoot())
		mpt2.SetVersion(Sequence(origin))
		if i%2 == 0 {
			doStateValInsert(t, "add 100 to c1", mpt2, "0123456", 100+i)
		}
		if i%3 == 0 {
			doStateValInsert(t, "add 1000 to c2", mpt2, "0123457", 1000+i)
		}
		if i%5 == 0 {
			doStateValInsert(t, "add 1000 to c3", mpt2, "0123458", 1000000+i)
		}
		if i%7 == 0 {
			doStateValInsert(t, "add 1000 to c4", mpt2, "0133458", 1000000000+i)
		}
		roots = append(roots, mpt2.GetRoot())
		var err = mpt2.SaveChanges(pndb, false)
		if err != nil {
			t.Error(err)
		}
		mpt.SetRoot(mpt2.GetRoot())
		prettyPrint(t, mpt)
		origin++
	}

	numStates := 200
	newOrigin := Sequence(origin - numStates)
	root := roots[len(roots)-numStates]
	mpt.SetRoot(root)
	prettyPrint(t, mpt)

	var err = mpt.Iterate(context.TODO(), iterHandler(t),
		NodeTypeValueNode|NodeTypeFullNode|NodeTypeExtensionNode)
	if err != nil {
		t.Errorf("iterate error: %v", err)
	}

	pndb.Iterate(context.TODO(), dbIteratorHandler(t))

	missingNodeHandler := func(ctx context.Context, path Path, key Key) error {
		return nil
	}
	err = mpt.UpdateVersion(context.TODO(), newOrigin, missingNodeHandler)
	if err != nil {
		t.Error("error updating origin:", err)
	}

	mpt.SetRoot(mpt2.GetRoot())
	err = mpt.Iterate(context.TODO(), iterHandler(t),
		NodeTypeValueNode|NodeTypeFullNode|NodeTypeExtensionNode)
	if err != nil {
		t.Error("iterate error:", err)
	}
	err = pndb.PruneBelowVersion(context.TODO(), newOrigin)
	pndb.Iterate(context.TODO(), dbIteratorHandler(t))

	if err != nil {
		t.Error("error pruning origin:", err)
	}

	mpt.SetRoot(mpt2.GetRoot())
	err = mpt.Iterate(context.TODO(), iterHandler(t),
		NodeTypeValueNode|NodeTypeFullNode|NodeTypeExtensionNode)
	if err != nil {
		t.Error("iterate error:", err)
	}
}

func TestMerkeTreeGetChanges(t *testing.T) {
	pndb, cleanup := newPNodeDB(t)
	defer cleanup()

	mpt := NewMerklePatriciaTrie(pndb, Sequence(0))
	var mndb = NewMemoryNodeDB()
	db := NewLevelNodeDB(mndb, mpt.db, false)
	mpt2 := NewMerklePatriciaTrie(db, Sequence(0))
	origin := 2016
	roots := make([]Key, 0, 10)

	for i := int64(0); i < 10; i++ {
		mpt2.ResetChangeCollector(mpt2.GetRoot())
		mpt2.SetVersion(Sequence(origin))

		doStateValInsert(t, "add 100 to c1", mpt2, "0123456", 100+i)
		doStateValInsert(t, "add 1000 to c2", mpt2, "0123457", 1000+i)
		doStateValInsert(t, "add 1000 to c3", mpt2, "0123458", 1000000+i)
		doStateValInsert(t, "add 1000 to c4", mpt2, "0133458", 1000000000+i)

		roots = append(roots, mpt2.GetRoot())

		if err := mpt2.SaveChanges(pndb, false); err != nil {
			panic(err)
		}

		prettyPrint(t, mpt2)
		origin++
	}

	mpts, err := GetChanges(context.TODO(), mndb, Sequence(origin-3),
		Sequence(origin))
	if err != nil {
		t.Error(err)
	}

	for _, mpt := range mpts {
		prettyPrint(t, mpt)
		mpt.Iterate(context.TODO(), iterHandler(t), NodeTypeValueNode)
	}

}

func doStateValInsert(t *testing.T, testcase string, mpt MerklePatriciaTrieI,
	key string, value int64) {

	state := &AState{}
	state.balance = value
	newRoot, err := mpt.Insert([]byte(key), state)
	if err != nil {
		t.Error(err)
	}
	mpt.SetRoot(newRoot)

	prettyPrint(t, mpt)

	doGetStateValue(t, mpt, key, value)
}

func doGetStateValue(t *testing.T, mpt MerklePatriciaTrieI,
	key string, value int64) {

	val, err := mpt.GetNodeValue([]byte(key))
	if err != nil {
		t.Fatalf("getting inserted value: %v %v", key, value)
	}
	if val == nil {
		t.Fatalf("inserted value not found: %v %v", key, value)
	}
	var astate, ok = val.(*AState)
	if !ok {
		t.Fatalf("wrong state type: %T", val)
	}
	if astate.balance != value {
		t.Fatalf("%s: wrong state value: %d, expected: %d", key, astate.balance,
			value)
	}
}

func stateIterHandler(t *testing.T, ctx context.Context, path Path, key Key,
	node Node) error {

	vn, ok := node.(*ValueNode)
	if ok {
		astate := &AState{}
		astate.Decode(vn.GetValue().Encode())
	}

	return nil
}

func dbIteratorHandler(t *testing.T) func(ctx context.Context, key Key, node Node) error {
	return func(ctx context.Context, key Key, node Node) error {
		return nil
	}
}

func computeMPTRoot(t *testing.T, mpt MerklePatriciaTrieI) (rk Key) {
	var (
		ndb  = mpt.GetNodeDB()
		mndb = NewMemoryNodeDB()
		back = context.Background()
	)
	require.NoError(t, MergeState(back, ndb, mndb))
	var root = mndb.ComputeRoot()
	if root == nil {
		return // nil
	}
	return Key(root.GetHashBytes()) // root key
}

func TestMPT_blockGenerationFlow(t *testing.T) {

	// persistent node DB represents chain state DB
	var stateDB, cleanup = newPNodeDB(t)
	defer cleanup()

	var mpt = NewMerklePatriciaTrie(stateDB, 0)

	// prior block DB and hash
	var (
		priorDB   NodeDB = stateDB
		priorHash        = computeMPTRoot(t, mpt)
	)

	// in loop:
	//  1. create block client state
	//  2. create transaction
	//  3. add/remove/change values
	//  4. merge transaction changes
	//  6. prune sate (not implemented)

	const n = 20

	// var back = context.Background()

	//
	for round := int64(0); round < n; round++ {

		//
		// 1. create block client state
		//
		var (
			ndb        = NewLevelNodeDB(NewMemoryNodeDB(), priorDB, false)
			blockState = NewMerklePatriciaTrie(ndb, Sequence(round))
			err        error
		)

		blockState.SetRoot(priorHash)

		//
		// 2. create transaction
		//
		var (
			tdb  = NewLevelNodeDB(NewMemoryNodeDB(), blockState.GetNodeDB(), false)
			tmpt = NewMerklePatriciaTrie(tdb, blockState.GetVersion())
		)
		tmpt.SetRoot(blockState.GetRoot())

		//
		//  3. add/remove/change values
		//

		// add
		var (
			v1 = testValue(fmt.Sprintf("test-value-%d-one", round))
			v2 = testValue(fmt.Sprintf("test-value-%d-two", round))
			p1 = Path(fmt.Sprintf("cafe%d", round))
			p2 = Path(fmt.Sprintf("face%d", round))
		)
		_, err = tmpt.Insert(p1, &v1)
		require.NoError(t, err)
		_, err = tmpt.Insert(p2, &v2)
		require.NoError(t, err)

		// remove
		if round-2 >= 0 {
			_, err = tmpt.Delete(Path(fmt.Sprintf("cafe%d", round-2)))
			require.NoError(t, err)
		}

		// change
		if round-1 >= 0 {
			var cval = testValue(fmt.Sprintf("test-value-%d-changed", round-1))
			_, err = tmpt.Insert(Path(fmt.Sprintf("face%d", round-1)),
				&cval)
			require.NoError(t, err)
		}

		//
		//  4. merge transaction changes
		//
		require.NoError(t, blockState.MergeMPTChanges(tmpt))

		priorDB = blockState.GetNodeDB()
		priorHash = blockState.GetRoot()

		require.NoError(t, blockState.SaveChanges(stateDB, false))
		mpt.SetRoot(priorHash)

		// //  5. prune state
		// var wps = WithPruneStats(back)
		// err = stateDB.PruneBelowVersion(wps, Sequence(round-1))
		// require.NoError(t, err)

		prettyPrint(t, mpt)
	}

}
