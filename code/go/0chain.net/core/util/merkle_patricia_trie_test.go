package util

import (
	"0chain.net/core/logging"
	"context"
	"encoding/hex"
	"fmt"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func init() {
	logging.Logger = zap.NewNop()
}

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
		if err := os.RemoveAll(dirname); err != nil {
			t.Fatal(err)
		}
		t.Fatal(err) //
	}

	cleanup = func() {
		pndb.db.Close()
		if err := os.RemoveAll(dirname); err != nil {
			t.Fatal(err)
		}
	}

	return
}

func TestMerkleTreeSaveToDB(t *testing.T) {
	pndb, cleanup := newPNodeDB(t)
	defer cleanup()

	mpt := NewMerklePatriciaTrie(pndb, Sequence(2016))
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
	mpt2 := NewMerklePatriciaTrie(db, Sequence(2016))

	doStateValInsert(t, mpt2, "0123456", 100)
	doStateValInsert(t, mpt2, "0123457", 1000)
	doStateValInsert(t, mpt2, "0123458", 1000000)
	doStateValInsert(t, mpt2, "0133458", 1000000000)

	var err = mpt2.SaveChanges(context.TODO(), pndb, false)
	if err != nil {
		t.Error(err)
	}

	err = mpt2.Iterate(context.TODO(), iterHandler(),
		NodeTypeValueNode|NodeTypeLeafNode|NodeTypeFullNode|NodeTypeExtensionNode)
	if err != nil {
		t.Fatal(err)
	}

	mpt.SetRoot(mpt2.GetRoot())

	err = mpt.Iterate(context.TODO(), iterHandler(),
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
			doStateValInsert(t, mpt2, "0123456", 100+i)
		}
		if i%3 == 0 {
			doStateValInsert(t, mpt2, "0123457", 1000+i)
		}
		if i%5 == 0 {
			doStateValInsert(t, mpt2, "0123458", 1000000+i)
		}
		if i%7 == 0 {
			doStateValInsert(t, mpt2, "0133458", 1000000000+i)
		}
		roots = append(roots, mpt2.GetRoot())
		var err = mpt2.SaveChanges(context.TODO(), pndb, false)
		if err != nil {
			t.Error(err)
		}
		mpt.SetRoot(mpt2.GetRoot())
		origin++
	}

	numStates := 200
	newOrigin := Sequence(origin - numStates)
	root := roots[len(roots)-numStates]
	mpt.SetRoot(root)

	var err = mpt.Iterate(context.TODO(), iterHandler(),
		NodeTypeValueNode|NodeTypeFullNode|NodeTypeExtensionNode)
	if err != nil {
		t.Errorf("iterate error: %v", err)
	}

	if err := pndb.Iterate(context.TODO(), dbIteratorHandler()); err != nil {
		t.Fatal(err)
	}

	missingNodeHandler := func(ctx context.Context, path Path, key Key) error {
		return nil
	}
	err = mpt.UpdateVersion(context.TODO(), newOrigin, missingNodeHandler)
	if err != nil {
		t.Error("error updating origin:", err)
	}

	mpt.SetRoot(mpt2.GetRoot())
	err = mpt.Iterate(context.TODO(), iterHandler(),
		NodeTypeValueNode|NodeTypeFullNode|NodeTypeExtensionNode)
	if err != nil {
		t.Error("iterate error:", err)
	}
	err = pndb.PruneBelowVersion(context.TODO(), newOrigin)
	if err := pndb.Iterate(context.TODO(), dbIteratorHandler()); err != nil {
		t.Fatal(err)
	}

	if err != nil {
		t.Error("error pruning origin:", err)
	}

	mpt.SetRoot(mpt2.GetRoot())
	err = mpt.Iterate(context.TODO(), iterHandler(),
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

	for i := int64(0); i < 10; i++ {
		mpt2.ResetChangeCollector(mpt2.GetRoot())
		mpt2.SetVersion(Sequence(origin))

		doStateValInsert(t, mpt2, "0123456", 100+i)
		doStateValInsert(t, mpt2, "0123457", 1000+i)
		doStateValInsert(t, mpt2, "0123458", 1000000+i)
		doStateValInsert(t, mpt2, "0133458", 1000000000+i)

		if err := mpt2.SaveChanges(context.TODO(), pndb, false); err != nil {
			panic(err)
		}

		origin++
	}

	mpts, err := GetChanges(context.TODO(), mndb, Sequence(origin-3),
		Sequence(origin))
	if err != nil {
		t.Error(err)
	}

	for _, mpt := range mpts {
		if err := mpt.Iterate(context.TODO(), iterHandler(), NodeTypeValueNode); err != nil {
			t.Fatal(err)
		}
	}

}

func doStateValInsert(t *testing.T, mpt MerklePatriciaTrieI, key string, value int64) {

	state := &AState{}
	state.balance = value
	newRoot, err := mpt.Insert([]byte(key), state)
	if err != nil {
		t.Error(err)
	}
	mpt.SetRoot(newRoot)

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

func dbIteratorHandler() func(ctx context.Context, key Key, node Node) error {
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
	return root.GetHashBytes() // root key
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

		require.NoError(t, blockState.SaveChanges(context.TODO(), stateDB, false))
		mpt.SetRoot(priorHash)

		// //  5. prune state
		// var wps = WithPruneStats(back)
		// err = stateDB.PruneBelowVersion(wps, Sequence(round-1))
		// require.NoError(t, err)
	}
}

func TestMPTHexachars(t *testing.T) {
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(2018))
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
	var mpt2 MerklePatriciaTrieI = NewMerklePatriciaTrie(db, Sequence(2018))

	doStrValInsert(t, mpt2, "1", "1")
	doStrValInsert(t, mpt2, "2", "2")
	doStrValInsert(t, mpt2, "a", "a")
}

func TestMPTInsertLeafNode(t *testing.T) {
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(0))
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
	mpt2 := NewMerklePatriciaTrie(db, Sequence(0))

	doStrValInsert(t, mpt2, "1234", "1")
	doStrValInsert(t, mpt2, "12356", "2")
	doStrValInsert(t, mpt2, "123671", "3")
	doStrValInsert(t, mpt2, "1237123", "4")
	doStrValInsert(t, mpt2, "12381234", "5")
	doStrValInsert(t, mpt2, "12391234", "6")

	_, err := mpt2.GetPathNodes(Path("12391234"))
	if err != nil {
		t.Fatal(err)
	}

	doStrValInsert(t, mpt2, "1234", "1.1")
	doStrValInsert(t, mpt2, "12345", "1.1.1")
	doStrValInsert(t, mpt2, "12356", "2.1")
	doStrValInsert(t, mpt2, "123567", "2.1.1")
	doStrValInsert(t, mpt2, "123671", "3.1")
	doStrValInsert(t, mpt2, "1236711", "3.1.1")
	doStrValInsert(t, mpt2, "123712", "4.1")
	doStrValInsert(t, mpt2, "1238124", "5.1")
	doStrValInsert(t, mpt2, "1239", "6.1")
}

func TestMPTInsertFullNode(t *testing.T) {
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(0))
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
	mpt2 := NewMerklePatriciaTrie(db, Sequence(0))

	doStrValInsert(t, mpt2, "1", "1")
	doStrValInsert(t, mpt2, "2", "2")
	doStrValInsert(t, mpt2, "11", "11")
	doStrValInsert(t, mpt2, "12", "12")
	doStrValInsert(t, mpt2, "211", "211")
	doStrValInsert(t, mpt2, "212", "212")
	doStrValInsert(t, mpt2, "3", "3")
	doStrValInsert(t, mpt2, "3112", "3112")
	doStrValInsert(t, mpt2, "3113", "3113")
}

func TestMPTInsertExtensionNode(t *testing.T) {
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(0))
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
	mpt2 := NewMerklePatriciaTrie(db, Sequence(0))

	doStrValInsert(t, mpt2, "12345", "12345")
	doStrValInsert(t, mpt2, "12346", "12346")
	doStrValInsert(t, mpt2, "2", "2")
	err := mpt2.Iterate(context.TODO(), iterStrPathHandler(), NodeTypeLeafNode|NodeTypeFullNode|NodeTypeExtensionNode)
	if err != nil {
		t.Fatal(err)
	}
	doStrValInsert(t, mpt2, "123", "123")
	doStrValInsert(t, mpt2, "22345", "22345")
	doStrValInsert(t, mpt2, "22346", "22346")
	doStrValInsert(t, mpt2, "22347", "22347")
	doStrValInsert(t, mpt2, "23", "23")
	doStrValInsert(t, mpt2, "12345", "12345.1")
	doStrValInsert(t, mpt2, "2234", "2234")
	doStrValInsert(t, mpt2, "22", "22")
}

func TestMPTDelete(t *testing.T) {
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(0))
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
	mpt2 := NewMerklePatriciaTrie(db, Sequence(0))

	doStrValInsert(t, mpt2, "12345", "12345")
	doStrValInsert(t, mpt2, "22345", "22345")

	doStrValInsert(t, mpt2, "123", "123")
	doStrValInsert(t, mpt2, "124", "124")

	doStrValInsert(t, mpt2, "12", "12")
	doStrValInsert(t, mpt2, "34567", "34567")
	doStrValInsert(t, mpt2, "34577", "34577")

	doStrValInsert(t, mpt2, "412345", "412345")
	doStrValInsert(t, mpt2, "42234", "42234")
	doStrValInsert(t, mpt2, "412346", "412346")
	doStrValInsert(t, mpt2, "513346", "513346")

	doStrValInsert(t, mpt2, "512345", "512345")
	doStrValInsert(t, mpt2, "52234", "52234")
	doStrValInsert(t, mpt2, "512346", "512346")

	doStrValInsert(t, mpt2, "612345", "612345")
	doStrValInsert(t, mpt2, "612512", "612512")
	doStrValInsert(t, mpt2, "612522", "612522")

	doDelete(t, mpt2, "12345", nil)
	doDelete(t, mpt2, "12", nil)
	doDelete(t, mpt2, "34577", nil)
	doDelete(t, mpt2, "124", nil)

	// lift up
	doDelete(t, mpt2, "42234", nil)
	doDelete(t, mpt2, "52234", nil)
	doStrValInsert(t, mpt2, "612345", "")

	// delete not existent node
	doDelete(t, mpt2, "abcdef123", ErrNodeNotFound)
	doDelete(t, mpt2, "6125123", ErrNodeNotFound)
	doDelete(t, mpt2, "613512", ErrNodeNotFound)
}

func TestMPTUniverse(t *testing.T) {
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(0))
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
	mpt2 := NewMerklePatriciaTrie(db, Sequence(0))

	doStrValInsert(t, mpt2, "1234513", "earth")
	doStrValInsert(t, mpt2, "123451478", "mars")
	doStrValInsert(t, mpt2, "123451", "mercury")
	doStrValInsert(t, mpt2, "123455", "jupiter")
	doStrValInsert(t, mpt2, "12345", "sun")
	doStrValInsert(t, mpt2, "12345131131", "moon")

	// Add a bunch of child nodes to existing full node
	doStrValInsert(t, mpt2, "123456", "saturn")
	doStrValInsert(t, mpt2, "123457", "uranus")
	doStrValInsert(t, mpt2, "123458", "neptune")
	doStrValInsert(t, mpt2, "123459", "pluto")

	doStrValInsert(t, mpt2, "123459", "dwarf planet")
	doStrValInsert(t, mpt2, "1234513", "green earth and ham")
	doStrValInsert(t, mpt2, "1234514781", "phobos")
	doStrValInsert(t, mpt2, "1234556", "europa")
	doStrValInsert(t, mpt2, "123452", "venus")
	doStrValInsert(t, mpt2, "123", "world")

	mpt.ResetChangeCollector(mpt.GetRoot()) // adding a new change collector so there are changes with old nodes that are not nil

	doStrValInsert(t, mpt2, "12346", "proxima centauri")
	doStrValInsert(t, mpt2, "1", "hello")

	err := mpt2.Iterate(context.TODO(), iterHandler(), NodeTypeValueNode|NodeTypeLeafNode|NodeTypeFullNode|NodeTypeExtensionNode)
	if err != nil {
		t.Fatal(err)
	}

	key, err := hex.DecodeString("aabaed5911cb89fe95680df9f42e07c5bb147fc7a742bde7cb5be62419eb41bf")
	if err != nil {
		t.Fatal(err)
	}
	err = mpt2.IterateFrom(context.TODO(), key, iterHandler(),
		NodeTypeValueNode|NodeTypeLeafNode|NodeTypeFullNode|NodeTypeExtensionNode)
	if err != nil {
		t.Fatal(err)
	}
}

func TestMPTInsertEthereumExample(t *testing.T) {
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(0))
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
	mpt2 := NewMerklePatriciaTrie(db, Sequence(0))

	doStrValInsert(t, mpt2, "646f", "verb")
	doStrValInsert(t, mpt2, "646f67", "puppy")
	doStrValInsert(t, mpt2, "646f6765", "coin")
	doStrValInsert(t, mpt2, "686f727365", "stallion")

	err := mpt2.Iterate(context.TODO(), iterHandler(), NodeTypeLeafNode|NodeTypeFullNode|NodeTypeExtensionNode)
	if err != nil {
		t.Fatal(err)
	}
}

func doStrValInsert(t *testing.T, mpt MerklePatriciaTrieI, key, value string) {

	t.Helper()

	newRoot, err := mpt.Insert(Path(key), &Txn{value})
	if err != nil {
		t.Error(err)
	}

	mpt.SetRoot(newRoot)
	doGetStrValue(t, mpt, key, value)
}

func doGetStrValue(t *testing.T, mpt MerklePatriciaTrieI, key, value string) {
	val, err := mpt.GetNodeValue(Path(key))
	if value == "" {
		if !(val == nil || err == ErrValueNotPresent) {
			t.Fatalf("setting value to blank didn't return nil value: %v, %v",
				val, err)
		}
		return
	}
	if err != nil {
		t.Fatalf("getting inserted value: %v %v", key, err)
	}
	if val == nil {
		t.Fatalf("inserted value not found: %v %v", key, value)
	}
}

func iterHandler() func(ctx context.Context, path Path, key Key, node Node) error {
	return func(ctx context.Context, path Path, key Key, node Node) error {
		if node == nil {
			return fmt.Errorf("stop")
		}
		return nil
	}
}

func iterStrPathHandler() func(ctx context.Context, path Path, key Key, node Node) error {
	return func(ctx context.Context, path Path, key Key, node Node) error {
		if node == nil {
			return fmt.Errorf("stop")
		}
		return nil
	}
}

func doDelete(t *testing.T, mpt MerklePatriciaTrieI, key string, expErr error) {

	newRoot, err := mpt.Delete([]byte(key))
	if err != expErr {
		t.Error(err)
		return
	}
	mpt.SetRoot(newRoot)
	doGetStrValue(t, mpt, key, "")
}

/*
  merge extensions : delete L from P(E(F(L,E))) and ensure P(E(F(E))) becomes P(E)
*/
func TestCasePEFLEdeleteL(t *testing.T) {
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(0))
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
	mpt2 := NewMerklePatriciaTrie(db, Sequence(0))

	doStrValInsert(t, mpt2, "223456789", "mercury")
	doStrValInsert(t, mpt2, "1235", "venus")
	doStrValInsert(t, mpt2, "123458970", "earth")
	doStrValInsert(t, mpt2, "123459012", "mars")
	doStrValInsert(t, mpt2, "123459013", "jupiter")
	doStrValInsert(t, mpt2, "123459023", "saturn")
	doStrValInsert(t, mpt2, "123459024", "uranus")

	doDelete(t, mpt2, "1235", nil)
	doStrValInsert(t, mpt2, "1235", "venus")
	doDelete(t, mpt2, "1235", nil)
	doStrValInsert(t, mpt2, "12345903", "neptune")

	err := mpt2.Iterate(context.TODO(), iterHandler(), NodeTypeLeafNode|NodeTypeFullNode|NodeTypeExtensionNode)
	if err != nil {
		t.Fatal(err)
	}

	_, err = mpt2.GetNodeValue(Path("123458970"))
	if err != nil {
		t.Error(err)
	}
}

func TestAddTwiceDeleteOnce(t *testing.T) {
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(0))
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
	mpt2 := NewMerklePatriciaTrie(db, Sequence(0))

	doStrValInsert(t, mpt2, "123456781", "x")
	doStrValInsert(t, mpt2, "123456782", "y")
	//doStrValInsert(t,"setup data", mpt2, "123556782", "z")

	doStrValInsert(t, mpt2, "223456781", "x")
	doStrValInsert(t, mpt2, "223456782", "y")

	doStrValInsert(t, mpt2, "223456782", "a")
	//doStrValInsert(t,"setup data", mpt2, "223556782", "b")

	//mpt2.Iterate(context.TODO(), iterHandler, NodeTypeLeafNode /*|NodeTypeFullNode|NodeTypeExtensionNode */)

	//doDelete("delete a leaf node", mpt2, "123456781", true)
	//mpt2.PrettyPrint(os.Stdout)

	//doDelete("delete a leaf node", mpt2, "223556782", true)
}
