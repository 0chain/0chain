package util

import (
	"context"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"sync"
	"testing"

	"github.com/0chain/gorocksdb"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"0chain.net/core/logging"
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

func TestWithPruneStats(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()

	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		args args
		want context.Context
	}{
		{
			name: "Test_WithPruneStats_OK",
			args: args{ctx: ctx},
			want: context.WithValue(ctx, PruneStatsKey, &PruneStats{Stage: PruneStateStart}),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := WithPruneStats(tt.args.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WithPruneStats() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetPruneStats(t *testing.T) {
	t.Parallel()

	ps := PruneStats{Stage: PruneStateStart}
	ctx := context.WithValue(context.TODO(), PruneStatsKey, &ps)

	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		args args
		want *PruneStats
	}{
		{
			name: "Test_GetPruneStats_OK",
			args: args{ctx: ctx},
			want: &ps,
		},
		{
			name: "Test_GetPruneStats_No_Prune_Stats_OK",
			args: args{ctx: context.TODO()},
			want: nil,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := GetPruneStats(tt.args.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetPruneStats() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCloneMPT(t *testing.T) {
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(0))

	type args struct {
		mpt MerklePatriciaTrieI
	}
	tests := []struct {
		name string
		args args
		want *MerklePatriciaTrie
	}{
		{
			name: "Test_CloneMPT_OK",
			args: args{mpt: mpt},
			want: mpt,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := CloneMPT(tt.args.mpt); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CloneMPT() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMerklePatriciaTrie_SetNodeDB(t *testing.T) {
	t.Parallel()

	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(0))

	type fields struct {
		mutex           *sync.RWMutex
		Root            Key
		db              NodeDB
		ChangeCollector ChangeCollectorI
		Version         Sequence
	}
	type args struct {
		ndb NodeDB
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "Test_MerklePatriciaTrie_SetNodeDB_OK",
			fields: fields{
				mutex:           &sync.RWMutex{},
				Root:            mpt.Root,
				db:              nil,
				ChangeCollector: mpt.ChangeCollector,
				Version:         mpt.Version,
			},
			args: args{ndb: mndb},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mpt := &MerklePatriciaTrie{
				mutex:           tt.fields.mutex,
				Root:            tt.fields.Root,
				db:              tt.fields.db,
				ChangeCollector: tt.fields.ChangeCollector,
				Version:         tt.fields.Version,
			}

			mpt.SetNodeDB(tt.args.ndb)

			if !reflect.DeepEqual(tt.args.ndb, mpt.db) {
				t.Errorf("SetNodeDB() setted = %v, want = %v", mpt.db, tt.args.ndb)
			}
		})
	}
}

func TestMerklePatriciaTrie_getNodeDB(t *testing.T) {
	t.Parallel()

	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(0))

	type fields struct {
		mutex           *sync.RWMutex
		Root            Key
		db              NodeDB
		ChangeCollector ChangeCollectorI
		Version         Sequence
	}
	tests := []struct {
		name   string
		fields fields
		want   NodeDB
	}{
		{
			name: "Test_MerklePatriciaTrie_getNodeDB_OK",
			fields: fields{
				mutex:           &sync.RWMutex{},
				Root:            mpt.Root,
				db:              mpt.db,
				ChangeCollector: mpt.ChangeCollector,
				Version:         mpt.Version,
			},
			want: mndb,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mpt := &MerklePatriciaTrie{
				mutex:           tt.fields.mutex,
				Root:            tt.fields.Root,
				db:              tt.fields.db,
				ChangeCollector: tt.fields.ChangeCollector,
				Version:         tt.fields.Version,
			}
			if got := mpt.getNodeDB(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getNodeDB() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMerklePatriciaTrie_getChangeCollector(t *testing.T) {
	t.Parallel()

	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(0))

	type fields struct {
		mutex           *sync.RWMutex
		Root            Key
		db              NodeDB
		ChangeCollector ChangeCollectorI
		Version         Sequence
	}
	tests := []struct {
		name   string
		fields fields
		want   ChangeCollectorI
	}{
		{
			name: "TestMerklePatriciaTrie_getChangeCollector_OK",
			fields: fields{
				mutex:           &sync.RWMutex{},
				Root:            mpt.Root,
				db:              mpt.db,
				ChangeCollector: mpt.ChangeCollector,
				Version:         mpt.Version,
			},
			want: mpt.ChangeCollector,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mpt := &MerklePatriciaTrie{
				mutex:           tt.fields.mutex,
				Root:            tt.fields.Root,
				db:              tt.fields.db,
				ChangeCollector: tt.fields.ChangeCollector,
				Version:         tt.fields.Version,
			}
			if got := mpt.getChangeCollector(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getChangeCollector() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMerklePatriciaTrie_GetNodeValue(t *testing.T) {
	// case 1
	pdb, cleanup := newPNodeDB(t)
	defer cleanup()

	pmpt := NewMerklePatriciaTrie(pdb, 0)
	pmpt.Root = Key("qwe")

	// case 2

	mdb := NewMemoryNodeDB()
	key := "key"
	mdb.Nodes[StrKey(key)] = nil

	mmpt := NewMerklePatriciaTrie(mdb, 0)
	mmpt.Root = Key("key")

	type fields struct {
		mutex           *sync.RWMutex
		Root            Key
		db              NodeDB
		ChangeCollector ChangeCollectorI
		Version         Sequence
	}
	type args struct {
		path Path
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    Serializable
		wantErr bool
	}{
		{
			name: "Test_MerklePatriciaTrie_GetNodeValue_Not_Found_ERR",
			fields: fields{
				mutex:           &sync.RWMutex{},
				Root:            pmpt.Root,
				db:              pmpt.db,
				ChangeCollector: pmpt.ChangeCollector,
				Version:         pmpt.Version,
			},
			args:    args{path: Path("qwe")},
			wantErr: true,
		},
		{
			name: "Test_MerklePatriciaTrie_GetNodeValue_Nil_Node_ERR",
			fields: fields{
				mutex:           &sync.RWMutex{},
				Root:            mmpt.Root,
				db:              mmpt.db,
				ChangeCollector: mmpt.ChangeCollector,
				Version:         mmpt.Version,
			},
			args:    args{path: Path(key)},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mpt := &MerklePatriciaTrie{
				mutex:           tt.fields.mutex,
				Root:            tt.fields.Root,
				db:              tt.fields.db,
				ChangeCollector: tt.fields.ChangeCollector,
				Version:         tt.fields.Version,
			}
			got, err := mpt.GetNodeValue(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetNodeValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetNodeValue() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMerklePatriciaTrie_Insert(t *testing.T) {
	db, cleanup := newPNodeDB(t)
	defer cleanup()

	db.wo = gorocksdb.NewDefaultWriteOptions()
	db.wo.SetSync(true)
	db.wo.DisableWAL(true)

	type fields struct {
		mutex           *sync.RWMutex
		Root            Key
		db              NodeDB
		ChangeCollector ChangeCollectorI
		Version         Sequence
	}
	type args struct {
		path  Path
		value Serializable
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    Key
		wantErr bool
	}{
		{
			name:    "Test_MerklePatriciaTrie_Insert_Nil_Value_ERR",
			fields:  fields{mutex: &sync.RWMutex{}, db: NewMemoryNodeDB()},
			wantErr: true,
		},
		{
			name:    "Test_MerklePatriciaTrie_Insert_Insert_Node_ERR",
			fields:  fields{mutex: &sync.RWMutex{}, db: db},
			args:    args{value: &SecureSerializableValue{Buffer: []byte("data")}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mpt := &MerklePatriciaTrie{
				mutex:           tt.fields.mutex,
				Root:            tt.fields.Root,
				db:              tt.fields.db,
				ChangeCollector: tt.fields.ChangeCollector,
				Version:         tt.fields.Version,
			}
			got, err := mpt.Insert(tt.args.path, tt.args.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("Insert() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Insert() got = %v, want %v", got, tt.want)
			}
		})
	}

	if err := cleanUp(); err != nil {
		t.Fatal(err)
	}
}

func TestMerklePatriciaTrie_GetPathNodes(t *testing.T) {
	t.Parallel()

	type fields struct {
		mutex           *sync.RWMutex
		Root            Key
		db              NodeDB
		ChangeCollector ChangeCollectorI
		Version         Sequence
	}
	type args struct {
		path Path
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []Node
		wantErr bool
	}{
		{
			name:    "Test_MerklePatriciaTrie_GetPathNodes_OK",
			fields:  fields{mutex: &sync.RWMutex{}, db: NewMemoryNodeDB()},
			args:    args{path: Path("path")},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mpt := &MerklePatriciaTrie{
				mutex:           tt.fields.mutex,
				Root:            tt.fields.Root,
				db:              tt.fields.db,
				ChangeCollector: tt.fields.ChangeCollector,
				Version:         tt.fields.Version,
			}
			got, err := mpt.GetPathNodes(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetPathNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetPathNodes() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMerklePatriciaTrie_getPathNodes(t *testing.T) {
	t.Parallel()

	fn := NewFullNode(&SecureSerializableValue{Buffer: []byte("fn data")})
	keyFn := Key(fn.GetHash())
	fn1 := NewFullNode(&SecureSerializableValue{Buffer: []byte("data")})
	keyFn1 := Key(fn1.GetHash())
	fn1.Children[0] = NewFullNode(&SecureSerializableValue{Buffer: []byte("children data")}).Encode()

	ln := NewLeafNode(Path("path"), 0, &SecureSerializableValue{Buffer: []byte("ln data")})
	keyLn := Key(ln.GetHash())

	keyEn := Key("key")
	en := NewExtensionNode(Path("path"), keyEn)

	db := NewMemoryNodeDB()
	err := db.PutNode(keyFn, fn)
	require.NoError(t, err)
	err = db.PutNode(keyFn1, fn1)
	require.NoError(t, err)
	err = db.PutNode(keyLn, ln)
	require.NoError(t, err)
	err = db.PutNode(keyEn, en)
	require.NoError(t, err)

	type fields struct {
		mutex           *sync.RWMutex
		Root            Key
		db              NodeDB
		ChangeCollector ChangeCollectorI
		Version         Sequence
	}
	type args struct {
		key  Key
		path Path
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []Node
		wantErr bool
	}{
		{
			name:    "Test_MerklePatriciaTrie_getPathNodes_Path_With_Zero_Length_OK",
			want:    nil,
			wantErr: false,
		},
		{
			name:    "Test_MerklePatriciaTrie_getPathNodes_Leaf_Node_Value_Not_Present_ERR",
			fields:  fields{mutex: &sync.RWMutex{}, db: db},
			args:    args{key: keyLn, path: Path("0")},
			wantErr: true,
		},
		{
			name:    "Test_MerklePatriciaTrie_getPathNodes_Full_Node_Value_Not_Present_ERR",
			fields:  fields{mutex: &sync.RWMutex{}, db: db},
			args:    args{key: keyFn, path: Path("0")},
			wantErr: true,
		},
		{
			name:    "Test_MerklePatriciaTrie_getPathNodes_Full_Node_Children_Value_Not_Present_ERR",
			fields:  fields{db: db, mutex: &sync.RWMutex{}},
			args:    args{key: keyFn1, path: Path("0123")},
			wantErr: true,
		},
		{
			name:    "Test_MerklePatriciaTrie_getPathNodes_Extension_Node_Value_Not_Present_ERR",
			fields:  fields{mutex: &sync.RWMutex{}, db: db},
			args:    args{key: keyEn, path: Path("0")},
			wantErr: true,
		},
		{
			name:    "Test_MerklePatriciaTrie_getPathNodes_Extension_Node_ERR",
			fields:  fields{mutex: &sync.RWMutex{}, db: db},
			args:    args{key: keyEn, path: Path("path:123")},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mpt := &MerklePatriciaTrie{
				mutex:           tt.fields.mutex,
				Root:            tt.fields.Root,
				db:              tt.fields.db,
				ChangeCollector: tt.fields.ChangeCollector,
				Version:         tt.fields.Version,
			}
			got, err := mpt.getPathNodes(tt.args.key, tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("getPathNodes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getPathNodes() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMerklePatriciaTrie_Iterate(t *testing.T) {
	t.Parallel()

	type fields struct {
		mutex           *sync.RWMutex
		Root            Key
		db              NodeDB
		ChangeCollector ChangeCollectorI
		Version         Sequence
	}
	type args struct {
		ctx            context.Context
		handler        MPTIteratorHandler
		visitNodeTypes byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "TestMerklePatriciaTrie_Iterate_OK",
			fields:  fields{mutex: &sync.RWMutex{}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mpt := &MerklePatriciaTrie{
				mutex:           tt.fields.mutex,
				Root:            tt.fields.Root,
				db:              tt.fields.db,
				ChangeCollector: tt.fields.ChangeCollector,
				Version:         tt.fields.Version,
			}
			if err := mpt.Iterate(tt.args.ctx, tt.args.handler, tt.args.visitNodeTypes); (err != nil) != tt.wantErr {
				t.Errorf("Iterate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMerklePatriciaTrie_getNodeValue(t *testing.T) {
	t.Parallel()

	fn := NewFullNode(&SecureSerializableValue{Buffer: []byte("fn data")})
	keyFn := Key(fn.GetHash())
	fn1 := NewFullNode(&SecureSerializableValue{Buffer: []byte("data")})
	keyFn1 := Key(fn1.GetHash())
	fn1.Children[0] = NewFullNode(&SecureSerializableValue{Buffer: []byte("children data")}).Encode()

	keyEn := Key("key")
	en := NewExtensionNode(Path("path"), keyEn)

	db := NewMemoryNodeDB()
	err := db.PutNode(keyFn, fn)
	require.NoError(t, err)
	err = db.PutNode(keyFn1, fn1)
	require.NoError(t, err)

	type fields struct {
		mutex           *sync.RWMutex
		Root            Key
		db              NodeDB
		ChangeCollector ChangeCollectorI
		Version         Sequence
	}
	type args struct {
		path Path
		node Node
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    Serializable
		wantErr bool
	}{
		{
			name:    "Test_MerklePatriciaTrie_getNodeValue_Full_Node_Value_Not_Present_ERR",
			fields:  fields{mutex: &sync.RWMutex{}, db: db},
			args:    args{node: fn, path: Path("0")},
			wantErr: true,
		},
		{
			name:    "Test_MerklePatriciaTrie_getNodeValue_Full_Node_Children_Value_Not_Present_ERR",
			fields:  fields{db: db, mutex: &sync.RWMutex{}},
			args:    args{node: fn1, path: Path("0123")},
			wantErr: true,
		},
		{
			name:    "Test_MerklePatriciaTrie_getNodeValues_Extension_Node_ERR",
			fields:  fields{mutex: &sync.RWMutex{}, db: db},
			args:    args{node: en, path: Path("path:123")},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mpt := &MerklePatriciaTrie{
				mutex:           tt.fields.mutex,
				Root:            tt.fields.Root,
				db:              tt.fields.db,
				ChangeCollector: tt.fields.ChangeCollector,
				Version:         tt.fields.Version,
			}
			got, err := mpt.getNodeValue(tt.args.path, tt.args.node)
			if (err != nil) != tt.wantErr {
				t.Errorf("getNodeValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getNodeValue() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMerklePatriciaTrie_insert(t *testing.T) {
	t.Parallel()

	type fields struct {
		mutex           *sync.RWMutex
		Root            Key
		db              NodeDB
		ChangeCollector ChangeCollectorI
		Version         Sequence
	}
	type args struct {
		value Serializable
		key   Key
		path  Path
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    Node
		want1   Key
		wantErr bool
	}{
		{
			name:    "Test_MerklePatriciaTrie_insert_ERR",
			fields:  fields{mutex: &sync.RWMutex{}, db: NewMemoryNodeDB()},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mpt := &MerklePatriciaTrie{
				mutex:           tt.fields.mutex,
				Root:            tt.fields.Root,
				db:              tt.fields.db,
				ChangeCollector: tt.fields.ChangeCollector,
				Version:         tt.fields.Version,
			}
			got, got1, err := mpt.insert(tt.args.value, tt.args.key, tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("insert() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("insert() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("insert() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestMerklePatriciaTrie_insertAtNode(t *testing.T) {
	db, cleanup := newPNodeDB(t)
	defer cleanup()
	db.wo = gorocksdb.NewDefaultWriteOptions()
	db.wo.SetSync(true)
	db.wo.DisableWAL(true)

	path := Path("path")

	type fields struct {
		mutex           *sync.RWMutex
		Root            Key
		db              NodeDB
		ChangeCollector ChangeCollectorI
		Version         Sequence
	}
	type args struct {
		value Serializable
		node  Node
		path  Path
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    Node
		want1   Key
		wantErr bool
	}{
		{
			name:   "Test_MerklePatriciaTrie_insertAtNode_Full_Node_ERR",
			fields: fields{mutex: &sync.RWMutex{}, db: db},
			args: args{
				node: NewFullNode(&SecureSerializableValue{}),
				path: Path("01"),
			},
			wantErr: true,
		},
		{
			name:   "Test_MerklePatriciaTrie_insertAtNode_Leaf_Node_ERR",
			fields: fields{mutex: &sync.RWMutex{}, db: db},
			args: args{
				node: NewLeafNode(Path(""), 0, &SecureSerializableValue{}),
				path: Path("01"),
			},
			wantErr: true,
		},
		{
			name:   "Test_MerklePatriciaTrie_insertAtNode_Leaf_Node_ERR2",
			fields: fields{mutex: &sync.RWMutex{}, db: db},
			args: args{
				node: NewLeafNode(path, 0, &SecureSerializableValue{}),
				path: append(path, []byte("123")...),
			},
			wantErr: true,
		},
		{
			name:   "Test_MerklePatriciaTrie_insertAtNode_Leaf_Node_ERR3",
			fields: fields{mutex: &sync.RWMutex{}, db: db},
			args: args{
				node: NewLeafNode(append(path, []byte("098")...), 0, &SecureSerializableValue{}),
				path: append(path, []byte("123")...),
			},
			wantErr: true,
		},
		{
			name:   "Test_MerklePatriciaTrie_insertAtNode_Leaf_Node_ERR4",
			fields: fields{mutex: &sync.RWMutex{}, db: db},
			args: args{
				node: NewLeafNode(append(path, []byte("098")...), 0, &SecureSerializableValue{}),
				path: path,
			},
			wantErr: true,
		},
		{
			name:   "Test_MerklePatriciaTrie_insertAtNode_Extension_Node_ERR",
			fields: fields{mutex: &sync.RWMutex{}, db: db},
			args: args{
				node: NewExtensionNode(path, Key("Key")),
				path: path,
			},
			wantErr: true,
		},
		{
			name:   "Test_MerklePatriciaTrie_insertAtNode_Extension_Node_ERR2",
			fields: fields{mutex: &sync.RWMutex{}, db: db},
			args: args{
				node: NewExtensionNode(path, Key("Key")),
				path: append(path, []byte("123")...),
			},
			wantErr: true,
		},
		{
			name:   "Test_MerklePatriciaTrie_insertAtNode_Extension_Node_ERR3",
			fields: fields{mutex: &sync.RWMutex{}, db: db},
			args: args{
				node: NewExtensionNode(append(path, []byte("0")...), Key("Key")),
				path: append(path, []byte("123")...),
			},
			wantErr: true,
		},
		{
			name:   "Test_MerklePatriciaTrie_insertAtNode_Extension_Node_ERR4",
			fields: fields{mutex: &sync.RWMutex{}, db: db},
			args: args{
				node: NewExtensionNode(append(path, []byte("0")...), Key("Key")),
				path: path,
			},
			wantErr: true,
		},
		{
			name:   "Test_MerklePatriciaTrie_insertAtNode_Extension_Node_ERR5",
			fields: fields{mutex: &sync.RWMutex{}, db: db},
			args: args{
				node: NewExtensionNode(append(path, []byte("098")...), Key("Key")),
				path: path,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mpt := &MerklePatriciaTrie{
				mutex:           tt.fields.mutex,
				Root:            tt.fields.Root,
				db:              tt.fields.db,
				ChangeCollector: tt.fields.ChangeCollector,
				Version:         tt.fields.Version,
			}
			got, got1, err := mpt.insertAtNode(tt.args.value, tt.args.node, tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("insertAtNode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("insertAtNode() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("insertAtNode() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestMerklePatriciaTrie_MergeDB(t *testing.T) {
	t.Parallel()

	mpt := NewMerklePatriciaTrie(NewMemoryNodeDB(), 0)

	mndb := NewMemoryNodeDB()
	err := mndb.PutNode(Key("key"), NewValueNode())
	require.NoError(t, err)

	type fields struct {
		mutex           *sync.RWMutex
		Root            Key
		db              NodeDB
		ChangeCollector ChangeCollectorI
		Version         Sequence
	}
	type args struct {
		ndb  NodeDB
		root Key
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Test_MerklePatriciaTrie_MergeDB_OK",
			fields: fields{
				mutex:           &sync.RWMutex{},
				Root:            mpt.Root,
				db:              NewMemoryNodeDB(),
				ChangeCollector: mpt.ChangeCollector,
				Version:         mpt.Version,
			},
			args:    args{ndb: mndb},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mpt := &MerklePatriciaTrie{
				mutex:           tt.fields.mutex,
				Root:            tt.fields.Root,
				db:              tt.fields.db,
				ChangeCollector: tt.fields.ChangeCollector,
				Version:         tt.fields.Version,
			}
			if err := mpt.MergeDB(tt.args.ndb, tt.args.root); (err != nil) != tt.wantErr {
				t.Errorf("MergeDB() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMerklePatriciaTrie_MergeMPTChanges(t *testing.T) {
	t.Skip("need protect DebugMPTNode against concurrent access")
	t.Parallel()

	DebugMPTNode = true

	db := NewLevelNodeDB(NewMemoryNodeDB(), NewMemoryNodeDB(), true)
	mpt := NewMerklePatriciaTrie(db, 0)

	type fields struct {
		mutex           *sync.RWMutex
		Root            Key
		db              NodeDB
		ChangeCollector ChangeCollectorI
		Version         Sequence
	}
	type args struct {
		mpt2 MerklePatriciaTrieI
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Test_MerklePatriciaTrie_MergeMPTChanges_OK",
			fields: fields{
				mutex:           &sync.RWMutex{},
				Root:            mpt.Root,
				db:              NewMemoryNodeDB(),
				ChangeCollector: mpt.ChangeCollector,
				Version:         mpt.Version,
			},
			args:    args{mpt2: NewMerklePatriciaTrie(NewMemoryNodeDB(), 0)},
			wantErr: false,
		},
		{
			name: "Test_MerklePatriciaTrie_MergeMPTChanges_Invalid_MPT_DB_ERR",
			fields: fields{
				mutex:           &sync.RWMutex{},
				Root:            mpt.Root,
				db:              NewMemoryNodeDB(),
				ChangeCollector: mpt.ChangeCollector,
				Version:         mpt.Version,
			},
			args: func() args {
				mpt := NewMerklePatriciaTrie(NewMemoryNodeDB(), 0)
				mpt.Root = Key("key")

				cc := &ChangeCollector{
					Changes: make(map[string]*NodeChange),
					Deletes: make(map[string]Node),
				}
				cc.Changes["key"] = nil
				cc.Deletes["key"] = nil
				mpt.ChangeCollector = cc

				return args{mpt2: mpt}
			}(),
			wantErr: true,
		},
		{
			name: "Test_MerklePatriciaTrie_MergeMPTChanges_LevelNDB_ERR",
			fields: fields{
				mutex:           &sync.RWMutex{},
				Root:            mpt.Root,
				db:              NewMemoryNodeDB(),
				ChangeCollector: mpt.ChangeCollector,
				Version:         mpt.Version,
			},
			args: args{
				mpt2: func() *MerklePatriciaTrie {
					mpt := NewMerklePatriciaTrie(
						NewLevelNodeDB(NewMemoryNodeDB(), NewMemoryNodeDB(), false),
						0,
					)
					mpt.Root = Key("key")

					return mpt
				}(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mpt := &MerklePatriciaTrie{
				mutex:           tt.fields.mutex,
				Root:            tt.fields.Root,
				db:              tt.fields.db,
				ChangeCollector: tt.fields.ChangeCollector,
				Version:         tt.fields.Version,
			}
			if err := mpt.MergeMPTChanges(tt.args.mpt2); (err != nil) != tt.wantErr {
				t.Errorf("MergeMPTChanges() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMerklePatriciaTrie_Validate(t *testing.T) {
	t.Parallel()

	pndb, cleanup := newPNodeDB(t)
	defer cleanup()

	mpt := NewMerklePatriciaTrie(nil, 0)

	lndb := NewLevelNodeDB(NewMemoryNodeDB(), NewMemoryNodeDB(), true)
	n := NewFullNode(&SecureSerializableValue{Buffer: []byte("value")})
	err := lndb.PutNode(n.GetHashBytes(), n)
	require.NoError(t, err)

	type fields struct {
		mutex           *sync.RWMutex
		Root            Key
		db              NodeDB
		ChangeCollector ChangeCollectorI
		Version         Sequence
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Test_MerklePatriciaTrie_Validate_PNDB_OK",
			fields: fields{
				mutex:           &sync.RWMutex{},
				Root:            mpt.Root,
				db:              pndb,
				ChangeCollector: mpt.ChangeCollector,
				Version:         mpt.Version,
			},
			wantErr: false,
		},
		{
			name: "Test_MerklePatriciaTrie_Validate_MNDB_OK",
			fields: fields{
				mutex:           &sync.RWMutex{},
				Root:            mpt.Root,
				db:              NewMemoryNodeDB(),
				ChangeCollector: mpt.ChangeCollector,
				Version:         mpt.Version,
			},
			wantErr: false,
		},
		{
			name: "Test_MerklePatriciaTrie_Validate_LNDB_OK",
			fields: fields{
				mutex: &sync.RWMutex{},
				Root:  mpt.Root,
				db:    NewLevelNodeDB(NewMemoryNodeDB(), NewMemoryNodeDB(), true),
				ChangeCollector: func() ChangeCollectorI {
					cc := ChangeCollector{
						Changes: make(map[string]*NodeChange),
						Deletes: make(map[string]Node),
					}
					cc.Changes["key"] = &NodeChange{}

					return &cc
				}(),
				Version: mpt.Version,
			},
			wantErr: false,
		},
		{
			name: "Test_MerklePatriciaTrie_Validate_LNDB_OK2",
			fields: fields{
				mutex: &sync.RWMutex{},
				Root:  mpt.Root,
				db:    lndb,
				ChangeCollector: func() ChangeCollectorI {
					cc := ChangeCollector{
						Changes: make(map[string]*NodeChange),
						Deletes: make(map[string]Node),
					}
					cc.Changes["key"] = &NodeChange{Old: n, New: n}

					return &cc
				}(),
				Version: mpt.Version,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mpt := &MerklePatriciaTrie{
				mutex:           tt.fields.mutex,
				Root:            tt.fields.Root,
				db:              tt.fields.db,
				ChangeCollector: tt.fields.ChangeCollector,
				Version:         tt.fields.Version,
			}
			if err := mpt.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	if err := cleanUp(); err != nil {
		t.Fatal(err)
	}
}

func TestGetChanges(t *testing.T) {
	t.Parallel()

	ndb := NewMemoryNodeDB()
	n := NewFullNode(&SecureSerializableValue{Buffer: []byte("data")})
	err := ndb.PutNode(n.GetHashBytes(), n)
	require.NoError(t, err)

	type args struct {
		ctx   context.Context
		ndb   NodeDB
		start Sequence
		end   Sequence
	}
	tests := []struct {
		name    string
		args    args
		want    map[Sequence]MerklePatriciaTrieI
		wantErr bool
	}{
		{
			name:    "Test_GetChanges_OK",
			args:    args{ndb: ndb, start: 1},
			want:    make(map[Sequence]MerklePatriciaTrieI),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := GetChanges(tt.args.ctx, tt.args.ndb, tt.args.start, tt.args.end)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetChanges() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetChanges() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsMPTValid(t *testing.T) {
	t.Parallel()

	type args struct {
		mpt MerklePatriciaTrieI
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "Test_IsMPTValid_OK",
			args:    args{mpt: NewMerklePatriciaTrie(NewMemoryNodeDB(), 0)},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if err := IsMPTValid(tt.args.mpt); (err != nil) != tt.wantErr {
				t.Errorf("IsMPTValid() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMerklePatriciaTrie_UpdateVersion(t *testing.T) {
	t.Parallel()

	mpt := NewMerklePatriciaTrie(nil, 0)
	mnh := func(ctx context.Context, path Path, key Key) error {
		return nil
	}
	root := []byte("root")

	type fields struct {
		mutex           *sync.RWMutex
		Root            Key
		db              NodeDB
		ChangeCollector ChangeCollectorI
		Version         Sequence
	}
	type args struct {
		ctx               context.Context
		version           Sequence
		missingNodeHander MPTMissingNodeHandler
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Test_MerklePatriciaTrie_UpdateVersion_Nil_Node_ERR",
			fields: fields{
				mutex: &sync.RWMutex{},
				Root:  root,
				db: func() NodeDB {
					db := NewMemoryNodeDB()
					n := NewFullNode(&SecureSerializableValue{Buffer: []byte("data")})
					err := db.PutNode(n.GetHashBytes(), n)
					require.NoError(t, err)

					return db
				}(),
				ChangeCollector: mpt.ChangeCollector,
				Version:         mpt.Version,
			},
			args: args{
				ctx:               context.WithValue(context.TODO(), PruneStatsKey, &PruneStats{}),
				version:           0,
				missingNodeHander: mnh,
			},
			wantErr: true,
		},
		{
			name: "Test_MerklePatriciaTrie_UpdateVersion_ERR",
			fields: fields{
				mutex: &sync.RWMutex{},
				Root:  root,
				db: func() NodeDB {
					db := NewMemoryNodeDB()

					n := NewExtensionNode([]byte("root"), []byte("key"))
					n.NodeKey = []byte(strconv.Itoa(0))
					err := db.PutNode(root, n)
					require.NoError(t, err)

					for i := 0; i < BatchSize+1; i++ {
						n := NewExtensionNode([]byte("root"), []byte("key"))
						n.NodeKey = []byte(strconv.Itoa(i + 1))
						err := db.PutNode([]byte(strconv.Itoa(i)), n)
						require.NoError(t, err)
					}

					return db
				}(),
				ChangeCollector: mpt.ChangeCollector,
				Version:         mpt.Version,
			},
			args: args{
				ctx:               context.WithValue(context.TODO(), PruneStatsKey, &PruneStats{}),
				version:           1,
				missingNodeHander: mnh,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mpt := &MerklePatriciaTrie{
				mutex:           tt.fields.mutex,
				Root:            tt.fields.Root,
				db:              tt.fields.db,
				ChangeCollector: tt.fields.ChangeCollector,
				Version:         tt.fields.Version,
			}
			if err := mpt.UpdateVersion(tt.args.ctx, tt.args.version, tt.args.missingNodeHander); (err != nil) != tt.wantErr {
				t.Errorf("UpdateVersion() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
