package util

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/0chain/gorocksdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/sha3"

	"0chain.net/core/encryption"
	"0chain.net/core/logging"
)

func init() {
	logging.Logger = zap.NewNop()
}

type AState struct {
	balance int64
}

func (as *AState) MarshalMsg([]byte) ([]byte, error) {
	return []byte(fmt.Sprintf("%v", as.balance)), nil
}

func (as *AState) UnmarshalMsg(buf []byte) ([]byte, error) {
	n, err := strconv.ParseInt(string(buf), 10, 63)
	if err != nil {
		return nil, err
	}
	as.balance = n
	return nil, nil
}

// receives a list of values
type valuesSponge struct {
	values []string
}

// receives a list of paths
type pathNodesSponge struct {
	paths []string
}

func newPNodeDB(t *testing.T) (pndb *PNodeDB, cleanup func()) {
	t.Helper()

	var dirname, err = ioutil.TempDir("", "mpt-pndb")
	require.NoError(t, err)

	pndb, err = NewPNodeDB(filepath.Join(dirname, "mpt"),
		filepath.Join(dirname, "deadnodes"),
		filepath.Join(dirname, "log"))
	if err != nil {
		if err := os.RemoveAll(dirname); err != nil {
			t.Fatal(err)
		}
		t.Fatal(err) //
	}

	cleanup = func() {
		// there's a bug on closing the pndb.db here, which would hang the tests,
		// removing the pndb.db.close() does not work, while run pndb.Flush() before
		// deleting the dir could help workaround.
		pndb.Flush()
		if err := os.RemoveAll(dirname); err != nil {
			t.Fatal(err)
		}
	}
	return
}

func TestMerkleTreeSaveToDB(t *testing.T) {
	pndb, cleanup := newPNodeDB(t)
	defer cleanup()

	mpt := NewMerklePatriciaTrie(pndb, Sequence(2016), nil)
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
	mpt2 := NewMerklePatriciaTrie(db, Sequence(2016), mpt.GetRoot())

	doStateValInsert(t, mpt2, "123456", 100)
	doStateValInsert(t, mpt2, "123457", 1000)
	doStateValInsert(t, mpt2, "123458", 1000000)
	doStateValInsert(t, mpt2, "133458", 1000000000)

	var err = mpt2.SaveChanges(context.TODO(), pndb, false)
	if err != nil {
		t.Error(err)
	}

	sponge := sha3.New256()
	err = mpt2.Iterate(context.TODO(), iterSpongeHandler(sponge),
		NodeTypeValueNode|NodeTypeLeafNode|NodeTypeFullNode|NodeTypeExtensionNode)
	if err != nil {
		t.Fatal(err)
	}
	iteratedHash := hex.EncodeToString(sponge.Sum(nil))
	exp := "43f393ebef7ac274d78a30f827dddbb65a5ae480c03592007eb4fbb20812e15c"
	if iteratedHash != exp {
		t.Fatalf("calculated sequence mismatch: %v, %v",
			iteratedHash, exp)
	}

	mpt3 := NewMerklePatriciaTrie(pndb, Sequence(2016), mpt2.GetRoot())

	sponge = sha3.New256()
	err = mpt3.Iterate(context.TODO(), iterSpongeHandler(sponge),
		NodeTypeValueNode|NodeTypeLeafNode|NodeTypeFullNode|NodeTypeExtensionNode)
	if err != nil {
		t.Errorf("iterate error: %v", err)
	}
	iteratedHash = hex.EncodeToString(sponge.Sum(nil))
	if iteratedHash != exp {
		t.Fatalf("calculated sequence mismatch: %v, %v",
			iteratedHash, exp)
	}
}

func TestMerkeTreePruning(t *testing.T) {
	pndb, cleanup := newPNodeDB(t)
	defer cleanup()

	mpt := NewMerklePatriciaTrie(NewLevelNodeDB(NewMemoryNodeDB(), pndb, false), Sequence(0), nil)
	origin := 2016
	roots := make([]Key, 0, 10)

	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
	mpt2 := NewMerklePatriciaTrie(db, Sequence(0), mpt.GetRoot())
	totalDelete := 0
	numStates := 100

	for i := int64(0); i < 200; i++ {
		mpt2.SetVersion(Sequence(origin))
		if i%2 == 0 {
			doStateValInsert(t, mpt2, "123456", 100+i)
		}
		if i%3 == 0 {
			doStateValInsert(t, mpt2, "123457", 1000+i)
		}
		if i%5 == 0 {
			doStateValInsert(t, mpt2, "123458", 1000000+i)
		}
		if i%7 == 0 {
			doStateValInsert(t, mpt2, "133458", 1000000000+i)
		}
		roots = append(roots, mpt2.GetRoot())
		deletedNodes := mpt2.GetDeletes()
		if origin < 2016+200-numStates {
			totalDelete += len(deletedNodes)
		}

		err := pndb.RecordDeadNodesWithVersion(deletedNodes, int64(mpt2.GetVersion()))
		require.NoError(t, err)

		require.NoError(t, mpt2.SaveChanges(context.TODO(), pndb, false))
		origin++
	}

	newOrigin := Sequence(origin - numStates)
	root := roots[len(roots)-numStates]
	mpt = NewMerklePatriciaTrie(mpt.GetNodeDB(), mpt.GetVersion(), root)

	checkIterationHash(t, mpt, "7678d38296cab5f5eb34000e5c0d9718cf79ec82949a1cbd65ce46e676199127")

	assert.NoError(t, pndb.Iterate(context.TODO(), dbIteratorHandler()))

	checkIterationHash(t, mpt, "7678d38296cab5f5eb34000e5c0d9718cf79ec82949a1cbd65ce46e676199127")
	ctx := WithPruneStats(context.Background())
	err := pndb.PruneBelowVersionV(ctx, newOrigin, 0)
	if err != nil {
		t.Error("error pruning origin:", err)
	}
	ps := GetPruneStats(ctx)
	require.NotNil(t, ps)
	require.Equal(t, int64(totalDelete), ps.Deleted)

	assert.NoError(t, pndb.Iterate(context.TODO(), dbIteratorHandler()))

	checkIterationHash(t, mpt, "7678d38296cab5f5eb34000e5c0d9718cf79ec82949a1cbd65ce46e676199127")
}

func doStateValInsert(t *testing.T, mpt MerklePatriciaTrieI, key string, value int64) {

	state := &AState{}
	state.balance = value
	_, err := mpt.Insert([]byte(key), state)
	if err != nil {
		t.Error(err)
	}

	doGetStateValue(t, mpt, key, value)
}

func doGetStateValue(t *testing.T, mpt MerklePatriciaTrieI,
	key string, value int64) {

	astate := &AState{}
	err := mpt.GetNodeValue([]byte(key), astate)
	assert.NoError(t, err)
	if astate.balance != value {
		t.Fatalf("%s: wrong state value: %d, expected: %d", key, astate.balance,
			value)
	}
}

func dbIteratorHandler() NodeDBIteratorHandler {
	return func(ctx context.Context, key Key, node Node) error {
		return nil
	}
}

// collect db keys
func dbKeysSpongeHandler(sponge *valuesSponge) NodeDBIteratorHandler {
	return func(ctx context.Context, key Key, node Node) error {
		if node == nil || key == nil {
			return fmt.Errorf("stop")
		}
		sponge.values = append(sponge.values, string(hex.EncodeToString(key)))
		return nil
	}
}

func TestMPT_blockGenerationFlow(t *testing.T) {

	// persistent node DB represents chain state DB
	var stateDB, cleanup = newPNodeDB(t)
	defer cleanup()

	var mpt = NewMerklePatriciaTrie(stateDB, 0, nil)

	// prior block DB and hash
	var (
		priorDB   NodeDB = stateDB
		priorHash        = mpt.GetRoot()
	)

	// in loop:
	//  1. create block client state
	//  2. create transaction
	//  3. add/remove/change values
	//  4. merge transaction changes
	//  6. prune sate (not implemented)

	expectedValueSets := [][]string{
		{"test-value-0-one", "test-value-0-two"},
		{"test-value-0-one", "test-value-0-changed", "test-value-1-one", "test-value-1-two"},
		{"test-value-0-changed", "test-value-1-one", "test-value-1-changed", "test-value-2-one", "test-value-2-two"},
		{"test-value-0-changed", "test-value-1-changed", "test-value-2-one", "test-value-2-changed", "test-value-3-one", "test-value-3-two"},
		{"test-value-0-changed", "test-value-1-changed", "test-value-2-changed", "test-value-3-one", "test-value-3-changed",
			"test-value-4-one", "test-value-4-two"},
		{"test-value-0-changed", "test-value-1-changed", "test-value-2-changed", "test-value-3-changed",
			"test-value-4-one", "test-value-4-changed", "test-value-5-one", "test-value-5-two"},
		{"test-value-0-changed", "test-value-1-changed", "test-value-2-changed", "test-value-3-changed", "test-value-4-changed",
			"test-value-5-one", "test-value-5-changed", "test-value-6-one", "test-value-6-two"},
		{"test-value-0-changed", "test-value-1-changed", "test-value-2-changed", "test-value-3-changed", "test-value-4-changed",
			"test-value-5-changed", "test-value-6-one", "test-value-6-changed", "test-value-7-one", "test-value-7-two"},
		{"test-value-0-changed", "test-value-1-changed", "test-value-2-changed", "test-value-3-changed", "test-value-4-changed",
			"test-value-5-changed", "test-value-6-changed", "test-value-7-one", "test-value-7-changed", "test-value-8-one", "test-value-8-two"},
		{"test-value-0-changed", "test-value-1-changed", "test-value-2-changed", "test-value-3-changed", "test-value-4-changed",
			"test-value-5-changed", "test-value-6-changed", "test-value-7-changed", "test-value-8-one", "test-value-8-changed",
			"test-value-9-one", "test-value-9-two"},
	}
	// var back = context.Background()

	//
	for round := int64(0); round < int64(len(expectedValueSets)); round++ {

		//
		// 1. create block client state
		//
		var (
			ndb        = NewLevelNodeDB(NewMemoryNodeDB(), priorDB, false)
			blockState = NewMerklePatriciaTrie(ndb, Sequence(round), priorHash)
			err        error
		)

		//
		// 2. create transaction
		//
		var (
			tdb  = NewLevelNodeDB(NewMemoryNodeDB(), blockState.GetNodeDB(), false)
			tmpt = NewMerklePatriciaTrie(tdb, blockState.GetVersion(), blockState.GetRoot())
		)

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

		checkValues(t, blockState, expectedValueSets[round])
		require.NoError(t, blockState.SaveChanges(context.TODO(), stateDB, false))
		mpt = NewMerklePatriciaTrie(mpt.GetNodeDB(), mpt.GetVersion(), priorHash)
		checkValues(t, mpt, expectedValueSets[round])

		// //  5. prune state
		// var wps = WithPruneStats(back)
		// err = stateDB.PruneBelowVersion(wps, Sequence(round-1))
		// require.NoError(t, err)
	}
}

func TestMPTHexachars(t *testing.T) {
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(2018), nil)
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
	var mpt2 MerklePatriciaTrieI = NewMerklePatriciaTrie(db, Sequence(2018), mpt.GetRoot())

	doStrValInsert(t, mpt2, "01", "1")
	doStrValInsert(t, mpt2, "02", "2")
	doStrValInsert(t, mpt2, "0a", "a")
}

func TestMPTInsertLeafNode(t *testing.T) {
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(0), nil)
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
	mpt2 := NewMerklePatriciaTrie(db, Sequence(0), mpt.GetRoot())

	doStrValInsert(t, mpt2, "1234", "1")
	doStrValInsert(t, mpt2, "123567", "2")
	doStrValInsert(t, mpt2, "123671", "3")
	doStrValInsert(t, mpt2, "12371234", "4")
	doStrValInsert(t, mpt2, "12381234", "5")
	doStrValInsert(t, mpt2, "12391234", "6")

	_, err := mpt2.GetPathNodes(Path("12391234"))
	if err != nil {
		t.Fatal(err)
	}

	doStrValInsert(t, mpt2, "1234", "1.1")
	doStrValInsert(t, mpt2, "123456", "1.1.1")
	doStrValInsert(t, mpt2, "123567", "2.1")
	doStrValInsert(t, mpt2, "12356789", "2.1.1")
	doStrValInsert(t, mpt2, "123671", "3.1")
	doStrValInsert(t, mpt2, "12367112", "3.1.1")
	doStrValInsert(t, mpt2, "123712", "4.1")
	doStrValInsert(t, mpt2, "12381245", "5.1")
	doStrValInsert(t, mpt2, "1239", "6.1")
}

func TestMPTInsertFullNode(t *testing.T) {
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(0), nil)
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
	mpt2 := NewMerklePatriciaTrie(db, Sequence(0), mpt.GetRoot())

	doStrValInsert(t, mpt2, "01", "1")
	doStrValInsert(t, mpt2, "02", "2")
	doStrValInsert(t, mpt2, "0112", "11")
	doStrValInsert(t, mpt2, "0121", "12")
	doStrValInsert(t, mpt2, "0211", "211")
	doStrValInsert(t, mpt2, "0212", "212")
	doStrValInsert(t, mpt2, "03", "3")
	doStrValInsert(t, mpt2, "0312", "3112")
	doStrValInsert(t, mpt2, "0313", "3113")
}

func TestMPTInsertExtensionNode(t *testing.T) {
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(0), nil)
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
	mpt2 := NewMerklePatriciaTrie(db, Sequence(0), mpt.GetRoot())

	doStrValInsert(t, mpt2, "123456", "12345")
	doStrValInsert(t, mpt2, "123467", "12346")
	doStrValInsert(t, mpt2, "02", "2")

	checkNodePaths(t, mpt2, NodeTypeExtensionNode, []string{"1"}) // pointing to "1234"
	checkNodePaths(t, mpt2, NodeTypeFullNode, []string{"", "1234"})
	checkNodePaths(t, mpt2, NodeTypeLeafNode, []string{"0", "12345", "12346"})

	sponge := sha3.New256()
	err := mpt2.Iterate(context.TODO(), iterSpongeHandler(sponge), NodeTypeLeafNode|NodeTypeFullNode|NodeTypeExtensionNode)
	if err != nil {
		t.Fatal(err)
	}
	iteratedHash := hex.EncodeToString(sponge.Sum(nil))
	exp := "65e7006c095e39e614065d80fd5f58f91c809a0324062c71233930e2650bcc28"
	if iteratedHash != exp {
		t.Fatalf("calculated sequence mismatch: %v, %v",
			iteratedHash, exp)
	}
	rootHash := ToHex(mpt2.root)
	exp = "ab0d18d651160ef469878958ee9b909e4a47c68342aaf2b7863a5681e7b249a5"
	if rootHash != exp {
		t.Fatalf("root hash mismatch: %v, %v",
			rootHash, exp)
	}
	doStrValInsert(t, mpt2, "1234", "123")
	// node paths are same changed, value was added to full node at "1234"
	checkNodePaths(t, mpt2, NodeTypeExtensionNode, []string{"1"})
	checkNodePaths(t, mpt2, NodeTypeFullNode, []string{"", "1234"})
	checkNodePaths(t, mpt2, NodeTypeLeafNode, []string{"0", "12345", "12346"})

	doStrValInsert(t, mpt2, "223456", "22345")
	doStrValInsert(t, mpt2, "223467", "22346")
	doStrValInsert(t, mpt2, "223478", "22347")
	checkNodePaths(t, mpt2, NodeTypeExtensionNode, []string{"1", "2"})
	checkNodePaths(t, mpt2, NodeTypeFullNode, []string{"", "1234", "2234"})
	checkNodePaths(t, mpt2, NodeTypeLeafNode, []string{"0", "12345", "12346", "22345", "22346", "22347"})
	doStrValInsert(t, mpt2, "23", "23")
	doStrValInsert(t, mpt2, "123456", "12345.1")
	doStrValInsert(t, mpt2, "2234", "2234")
	doStrValInsert(t, mpt2, "22", "22")
	checkNodePaths(t, mpt2, NodeTypeExtensionNode, []string{"1", "223"})
	checkNodePaths(t, mpt2, NodeTypeFullNode, []string{"", "1234", "2", "22", "2234"})
	checkNodePaths(t, mpt2, NodeTypeLeafNode, []string{"0", "12345", "12346", "22345", "22346", "22347", "23"})
	rootHash = ToHex(mpt2.root)
	exp = "ae732ce5605e0d240c2989c0f41a9c586c051975aa0b013422fd9cdc10cc2dab"
	if rootHash != exp {
		t.Fatalf("root hash mismatch: %v, %v",
			rootHash, exp)
	}
}

func TestMPTRepetitiveInsert(t *testing.T) {
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(0), nil)
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
	mpt2 := NewMerklePatriciaTrie(db, Sequence(0), mpt.GetRoot())

	doStrValInsert(t, mpt2, "223456", "22345")
	doStrValInsert(t, mpt2, "223467", "22346")
	assert.Equal(t, "eeb5fd1ccfafbe5b7ebf9368370fd2a6b1d2e076f686be78bba3f2b7144d8e07", ToHex(mpt2.root))
	checkValues(t, mpt2, []string{"22345", "22346"})
	mpt2.ChangeCollector.GetChanges()

	doStrValInsert(t, mpt2, "223467", "22347")
	checkValues(t, mpt2, []string{"22345", "22347"})
	doStrValInsert(t, mpt2, "223467", "22346")
	checkValues(t, mpt2, []string{"22345", "22346"})
	assert.Equal(t, "eeb5fd1ccfafbe5b7ebf9368370fd2a6b1d2e076f686be78bba3f2b7144d8e07", ToHex(mpt2.root))
}

func TestMPT_MultipleConcurrentInserts(t *testing.T) {
	//t.Parallel()
	db := NewLevelNodeDB(NewMemoryNodeDB(), NewMemoryNodeDB(), false)
	mpt := NewMerklePatriciaTrie(db, Sequence(0), nil)
	ldb := NewLevelNodeDB(NewMemoryNodeDB(), db, false)
	numGoRoutines := 10
	numTxns := 100
	txns := make([]*Txn, numGoRoutines*numTxns)
	for i := 0; i < len(txns); i++ {
		txns[i] = &Txn{fmt.Sprintf("%v", len(txns)-i)}
	}
	// insert some of the nodes to the original mpt
	for i := 0; i < numGoRoutines; i++ {
		_, err := mpt.Insert(Path(encryption.Hash(txns[i*numTxns].Data)), txns[i*numTxns])
		require.NoError(t, err)
	}
	checkIterationHash(t, mpt, "6e7eabd0548a424b78bf2c4393a15c42e42fac1287be0e3723dcb31e720b53ec")
	mpt2 := NewMerklePatriciaTrie(ldb, Sequence(0), mpt.GetRoot())
	checkIterationHash(t, mpt2, "6e7eabd0548a424b78bf2c4393a15c42e42fac1287be0e3723dcb31e720b53ec")
	wg := &sync.WaitGroup{}
	for i := 0; i < numGoRoutines; i++ {
		wg.Add(1)
		go func(mpt2 MerklePatriciaTrieI, i int) {
			defer wg.Done()
			for j := 1; j < numTxns; j++ {
				_, err := mpt2.Insert(Path(encryption.Hash(txns[i*numTxns+j].Data)), txns[i*numTxns+j])
				require.NoError(t, err)
			}
		}(mpt2, i)
	}
	wg.Wait()
	checkIterationHash(t, mpt2, "54877a4aac07cc1afd1c544ec8a5d3e3d79403c1b66297d9801635639fb96c26")
	checkIterationHash(t, mpt, "6e7eabd0548a424b78bf2c4393a15c42e42fac1287be0e3723dcb31e720b53ec")
	require.NoError(t, mpt.MergeMPTChanges(mpt2))
	checkIterationHash(t, mpt, "54877a4aac07cc1afd1c544ec8a5d3e3d79403c1b66297d9801635639fb96c26")
}

func TestMPTDelete(t *testing.T) {
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(0), nil)
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
	mpt2 := NewMerklePatriciaTrie(db, Sequence(0), mpt.GetRoot())

	doStrValInsert(t, mpt2, "123456", "12345")
	doStrValInsert(t, mpt2, "223456", "22345")

	doStrValInsert(t, mpt2, "1234", "123")
	doStrValInsert(t, mpt2, "1245", "124")

	doStrValInsert(t, mpt2, "12", "12")
	doStrValInsert(t, mpt2, "345678", "34567")
	doStrValInsert(t, mpt2, "345778", "34577")

	doStrValInsert(t, mpt2, "412345", "412345")
	doStrValInsert(t, mpt2, "4223", "42234")
	doStrValInsert(t, mpt2, "412346", "412346")
	doStrValInsert(t, mpt2, "513346", "513346")

	doStrValInsert(t, mpt2, "512345", "512345")
	doStrValInsert(t, mpt2, "5223", "52234")
	doStrValInsert(t, mpt2, "512346", "512346")

	doStrValInsert(t, mpt2, "612345", "612345")
	doStrValInsert(t, mpt2, "612512", "612512")
	doStrValInsert(t, mpt2, "612522", "612522")

	doDelete(t, mpt2, "123456", nil)
	doDelete(t, mpt2, "12", nil)
	doDelete(t, mpt2, "345778", nil)
	doDelete(t, mpt2, "1245", nil)

	// lift up
	doDelete(t, mpt2, "4223", nil)
	doDelete(t, mpt2, "5223", nil)
	doStrValInsert(t, mpt2, "612345", "")

	// delete not existent node
	doDelete(t, mpt2, "abcdef12", ErrValueNotPresent)
	doDelete(t, mpt2, "61251234", ErrValueNotPresent)
	doDelete(t, mpt2, "613512", ErrValueNotPresent)
}

func TestMPTUniverse(t *testing.T) {
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(0), nil)
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
	mpt2 := NewMerklePatriciaTrie(db, Sequence(0), mpt.GetRoot())

	doStrValInsert(t, mpt2, "01234513", "earth")
	doStrValInsert(t, mpt2, "0123451478", "mars")
	doStrValInsert(t, mpt2, "01234512", "mercury")
	doStrValInsert(t, mpt2, "01234551", "jupiter")
	doStrValInsert(t, mpt2, "012345", "sun")
	doStrValInsert(t, mpt2, "012345131131", "moon")

	// Add a bunch of child nodes to existing full node
	doStrValInsert(t, mpt2, "01234567", "saturn")
	doStrValInsert(t, mpt2, "01234578", "uranus")
	doStrValInsert(t, mpt2, "01234589", "neptune")
	doStrValInsert(t, mpt2, "01234590", "pluto")

	doStrValInsert(t, mpt2, "01234590", "dwarf planet")
	doStrValInsert(t, mpt2, "01234513", "green earth and ham")
	doStrValInsert(t, mpt2, "012345147812", "phobos")
	doStrValInsert(t, mpt2, "0123455167", "europa")
	doStrValInsert(t, mpt2, "01234523", "venus")
	doStrValInsert(t, mpt2, "0123", "world")

	doStrValInsert(t, mpt2, "012346", "proxima centauri")
	doStrValInsert(t, mpt2, "01", "hello")

	rootHash := hex.EncodeToString(mpt2.root)
	exp := "b4cf23e66a77f362b7753435cb67e30e37f8cdf15e193bf7c84ec48c27e232c3"
	if rootHash != exp {
		t.Fatalf("root hash mismatch: %v, %v",
			rootHash, exp)
	}

	sponge := sha3.New256()
	err := mpt2.Iterate(context.TODO(), iterSpongeHandler(sponge), NodeTypeValueNode|NodeTypeLeafNode|NodeTypeFullNode|NodeTypeExtensionNode)
	if err != nil {
		t.Fatal(err)
	}
	iteratedHash := hex.EncodeToString(sponge.Sum(nil))
	exp = "c7880b96f968c6d5b423de5e294a8a045d4173b9bfe3446ab2fffc156ea06533"
	if iteratedHash != exp {
		t.Fatalf("calculated sequence mismatch: %v, %v",
			iteratedHash, exp)
	}
	//fmt.Println(rootHash)
	//
	key, err := hex.DecodeString(rootHash)
	if err != nil {
		t.Fatal(err)
	}
	// collect values
	valuesSponge := valuesSponge{make([]string, 0, 16)}
	err = mpt2.IterateFrom(context.TODO(), key, iterValuesSpongeHandler(&valuesSponge), NodeTypeValueNode)
	if err != nil {
		t.Fatal(err)
	}
	values := strings.Join(valuesSponge.values, ",")
	expValues := "hello,world,sun,mercury,green earth and ham,moon,mars,phobos,venus,jupiter,europa,saturn,uranus,neptune,dwarf planet,proxima centauri"
	require.Equal(t, expValues, values)
}

func TestMPTInsertEthereumExample(t *testing.T) {
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(0), nil)
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
	mpt2 := NewMerklePatriciaTrie(db, Sequence(0), mpt.GetRoot())

	doStrValInsert(t, mpt2, "646f", "verb")
	doStrValInsert(t, mpt2, "646f67", "puppy")
	doStrValInsert(t, mpt2, "646f6765", "coin")
	rootHash := ToHex(mpt2.root)
	exp := "08a9172ec78ee7405e5fd8fb7be7898c6cfe57c5f0cdbe6180109dc9b9afe459"
	if rootHash != exp {
		t.Fatalf("root hash mismatch: %v, %v",
			rootHash, exp)
	}
	doStrValInsert(t, mpt2, "686f727365", "stallion")

	sponge := sha3.New256()
	err := mpt2.Iterate(context.TODO(), iterSpongeHandler(sponge), NodeTypeLeafNode|NodeTypeFullNode|NodeTypeExtensionNode)
	if err != nil {
		t.Fatal(err)
	}
	iteratedHash := hex.EncodeToString(sponge.Sum(nil))
	exp = "864ec1ef8fdbc2a0385b31c3910f0d63561d5a7f90b1595b1246f5998f54fb3b"
	if iteratedHash != exp {
		t.Fatalf("calculated sequence mismatch: %v, %v",
			iteratedHash, exp)
	}
	doDelete(t, mpt2, "686f727365", nil)
	rootHash = ToHex(mpt2.root)
	exp = "08a9172ec78ee7405e5fd8fb7be7898c6cfe57c5f0cdbe6180109dc9b9afe459"
	if rootHash != exp {
		t.Fatalf("root hash mismatch: %v, %v",
			rootHash, exp)
	}
	sponge = sha3.New256()
	err = mpt2.Iterate(context.TODO(), iterSpongeHandler(sponge), NodeTypeLeafNode|NodeTypeFullNode|NodeTypeExtensionNode)
	if err != nil {
		t.Fatal(err)
	}
	iteratedHash = hex.EncodeToString(sponge.Sum(nil))
	exp = "f491725f0fb091592cddea28f33ba2ca2ca465b8b437b4d535bbe9114e906769"
	if iteratedHash != exp {
		t.Fatalf("calculated sequence mismatch: %v, %v",
			iteratedHash, exp)
	}
}

func checkNodePaths(t *testing.T, mpt MerklePatriciaTrieI, visitNodeTypes byte, values []string) {
	t.Helper()

	pathsSponge := pathNodesSponge{}
	err := mpt.Iterate(context.TODO(), iterPathNodesSpongeHandler(&pathsSponge), visitNodeTypes)
	require.NoError(t, err)
	require.Equal(t, values, pathsSponge.paths)
}

func checkValues(t *testing.T, mpt MerklePatriciaTrieI, values []string) {
	t.Helper()

	sponge := valuesSponge{make([]string, 0, 16)}
	require.NoError(t, mpt.Iterate(context.TODO(), iterValuesSpongeHandler(&sponge), NodeTypeValueNode))
	assert.ElementsMatch(t, values, sponge.values)
}

func checkIterationHash(t *testing.T, mpt MerklePatriciaTrieI, expectedHash string) {
	t.Helper()

	sponge := sha3.New256()
	require.NoError(t, mpt.Iterate(context.TODO(), iterSpongeHandler(sponge), NodeTypeLeafNode|NodeTypeFullNode|NodeTypeExtensionNode))
	assert.Equal(t, expectedHash, ToHex(sponge.Sum(nil)))
}

func doStrValInsert(t *testing.T, mpt MerklePatriciaTrieI, key, value string) {

	t.Helper()

	_, err := mpt.Insert(Path(key), &Txn{value})
	if err != nil {
		t.Error(err)
	}

	doGetStrValue(t, mpt, key, value)
}

func doGetStrValue(t *testing.T, mpt MerklePatriciaTrieI, key, value string) {
	val := &Txn{}
	err := mpt.GetNodeValue(Path(key), val)

	if value == "" {
		if err != ErrValueNotPresent {
			t.Fatalf("setting value to blank didn't return nil value: %v, %v",
				val, err)
		}
		return
	}

	if err != nil {
		t.Fatalf("getting inserted value: %v %v", key, err)
	}

	readValue := val.Data
	if readValue != value {
		t.Fatalf("Read value doesn't match: %v %v", readValue, value)
	}
}

// aggregate into hash
func iterSpongeHandler(sponge hash.Hash) MPTIteratorHandler {
	return func(ctx context.Context, path Path, key Key, node Node) error {
		if node == nil {
			return fmt.Errorf("stop")
		}
		if key == nil {
			// value node
			vn, ok := node.(*ValueNode)
			if !ok {
				return fmt.Errorf("value node expected")
			}
			v, err := vn.Value.MarshalMsg(nil)
			if err != nil {
				panic(err)
			}
			sponge.Write(v)
		} else {
			sponge.Write(key)
		}
		return nil
	}
}

// aggregate into a list of values
func iterValuesSpongeHandler(sponge *valuesSponge) MPTIteratorHandler {
	return func(ctx context.Context, path Path, key Key, node Node) error {
		if node == nil {
			return fmt.Errorf("stop")
		}
		if key == nil {
			// value node
			vn, ok := node.(*ValueNode)
			if !ok {
				return fmt.Errorf("value node expected")
			}

			v, err := vn.Value.MarshalMsg(nil)
			if err != nil {
				panic(err)
			}
			sponge.values = append(sponge.values, string(v))
		}
		return nil
	}
}

func iterPathNodesSpongeHandler(sponge *pathNodesSponge) MPTIteratorHandler {
	return func(ctx context.Context, path Path, key Key, node Node) error {
		if node == nil {
			return fmt.Errorf("stop")
		}
		if key != nil {
			sponge.paths = append(sponge.paths, string(path))
		}
		return nil
	}
}

func iterNopHandler() MPTIteratorHandler {
	return func(ctx context.Context, path Path, key Key, node Node) error {
		return nil
	}
}

func doDelete(t *testing.T, mpt MerklePatriciaTrieI, key string, expErr error) {
	t.Helper()

	_, err := mpt.Delete([]byte(key))
	if err != expErr {
		t.Fatalf("expect err: %v, got err: %v", expErr, err)
		return
	}
	doGetStrValue(t, mpt, key, "")
}

/*
  merge extensions : delete L from P(E(F(L,E))) and ensure P(E(F(E))) becomes P(E)
*/
func TestCasePEFLEdeleteL(t *testing.T) {
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(0), nil)
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
	mpt2 := NewMerklePatriciaTrie(db, Sequence(0), nil)

	doStrValInsert(t, mpt2, "22345678", "mercury")
	doStrValInsert(t, mpt2, "1235", "venus")
	doStrValInsert(t, mpt2, "1234589701", "earth")
	doStrValInsert(t, mpt2, "1234590121", "mars")
	doStrValInsert(t, mpt2, "1234590131", "jupiter")
	doStrValInsert(t, mpt2, "1234590231", "saturn")
	doStrValInsert(t, mpt2, "1234590241", "uranus")
	rootHash := hex.EncodeToString(mpt2.root)
	expWithVenus := "9dce15353bd2b14ca9ac4845edcb4cdb874280d23725cec2426cec507e50034d"
	if rootHash != expWithVenus {
		t.Fatalf("root hash mismatch: %v, %v",
			rootHash, expWithVenus)
	}
	doDelete(t, mpt2, "1235", nil)
	rootHash = hex.EncodeToString(mpt2.root)
	expWithoutVenus := "2d76ab31e7ed52310ed734eaa9867be8a9173c98807a51f0d24cdd5c478c0f23"
	if rootHash != expWithoutVenus {
		t.Fatalf("root hash mismatch: %v, %v",
			rootHash, expWithoutVenus)
	}
	doStrValInsert(t, mpt2, "1235", "venus")
	rootHash = hex.EncodeToString(mpt2.root)
	if rootHash != expWithVenus {
		t.Fatalf("root hash mismatch: %v, %v",
			rootHash, expWithVenus)
	}
	doDelete(t, mpt2, "1235", nil)
	rootHash = hex.EncodeToString(mpt2.root)
	if rootHash != expWithoutVenus {
		t.Fatalf("root hash mismatch: %v, %v",
			rootHash, expWithoutVenus)
	}
	doStrValInsert(t, mpt2, "1234590341", "neptune")
	rootHash = hex.EncodeToString(mpt2.root)
	exp := "44e2ec885f3951eb8f31b9ea11d1b906671b792537e33c04a7cc05fe7b82e720"
	if rootHash != exp {
		t.Fatalf("root hash mismatch: %v, %v", rootHash, exp)
	}
	checkIterationHash(t, mpt2, "d46585b7e3366749c27ddf90622006b793e8c6ec21702a4c00dd512dd074895f")
	// collect values
	valuesSponge := valuesSponge{make([]string, 0, 16)}
	err := mpt2.Iterate(context.TODO(), iterValuesSpongeHandler(&valuesSponge), NodeTypeValueNode)
	if err != nil {
		t.Fatal(err)
	}
	values := strings.Join(valuesSponge.values, ",")
	exp = "earth,mars,jupiter,saturn,uranus,neptune,mercury"
	if values != exp {
		t.Fatalf("values mismatch: %v, got %v", values, exp)
	}

	val := &Txn{}
	err = mpt2.GetNodeValue(Path("1234589701"), val)
	if err != nil {
		t.Error(err)
	}
	value := val.Data
	exp = "earth"
	if value != exp {
		t.Fatalf("value mismatch: %v, %v", value, exp)
	}
}

func TestAddTwiceDeleteOnce(t *testing.T) {
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(0), nil)
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
	mpt2 := NewMerklePatriciaTrie(db, Sequence(0), nil)

	doStrValInsert(t, mpt2, "1234567812", "x")
	doStrValInsert(t, mpt2, "1234567822", "y")

	doStrValInsert(t, mpt2, "2234567812", "x")
	doStrValInsert(t, mpt2, "2234567822", "y")

	doStrValInsert(t, mpt2, "2234567822", "a")
}

func TestWithPruneStats(t *testing.T) {

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
			if got := WithPruneStats(tt.args.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WithPruneStats() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetPruneStats(t *testing.T) {
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
			if got := GetPruneStats(tt.args.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetPruneStats() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCloneMPT(t *testing.T) {
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(0), nil)

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

			if got := CloneMPT(tt.args.mpt); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CloneMPT() = %v, want %v", got, tt.want)
			}
		})
	}
}

type User struct {
	Name        string `json:"full_name"`
	Age         int    `json:"age,omitempty"`
	Active      bool   `json:"-"`
	lastLoginAt string
}

func (u *User) MarshalMsg([]byte) ([]byte, error) {
	marshal, _ := json.Marshal(u)
	return marshal, nil
}

func (u *User) UnmarshalMsg(bytes []byte) ([]byte, error) {
	err := json.Unmarshal(bytes, u)
	return nil, err
}

func TestCloneMPT2(t *testing.T) {
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(0), nil)

	if _, err := mpt.Insert([]byte("aaa"), &User{}); err != nil {
		t.Error(err)
	}

	mpt1 := NewMerklePatriciaTrie(mndb, Sequence(1), mpt.root)

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
			args: args{mpt: mpt1},
			want: mpt,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := CloneMPT(tt.args.mpt)
			if _, err := mpt1.Insert([]byte("bbb"), &User{Age: 1}); err != nil {
				t.Error(err)
			}
			mpt2 := NewMerklePatriciaTrie(mndb, Sequence(1), mpt.root)

			if !reflect.DeepEqual(got, mpt2) {
				t.Errorf("CloneMPT() = %v, want %v", got, mpt2)
			}
		})
	}
}

func TestMerklePatriciaTrie_SetNodeDB(t *testing.T) {
	t.Parallel()

	mndb := NewMemoryNodeDB()
	mndb1 := NewLevelNodeDB(NewMemoryNodeDB(), NewMemoryNodeDB(), false)
	mpt := NewMerklePatriciaTrie(mndb, Sequence(0), nil)

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
				Root:            mpt.root,
				db:              mndb1,
				ChangeCollector: mpt.ChangeCollector,
				Version:         mpt.Version,
			},
			args: args{ndb: mndb},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mpt := &MerklePatriciaTrie{
				mutex:           tt.fields.mutex,
				root:            tt.fields.Root,
				db:              tt.fields.db,
				ChangeCollector: tt.fields.ChangeCollector,
				Version:         tt.fields.Version,
			}

			mpt.SetNodeDB(tt.args.ndb)
			lndb := mpt.db.(*LevelNodeDB)
			if !reflect.DeepEqual(tt.args.ndb, lndb.current) {
				t.Errorf("SetNodeDB() setted = %v, want = %v", mpt.db, tt.args.ndb)
			}
		})
	}
}

func TestMerklePatriciaTrie_getNodeDB(t *testing.T) {
	t.Parallel()

	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(0), nil)

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
				Root:            mpt.root,
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
			mpt := &MerklePatriciaTrie{
				mutex:           tt.fields.mutex,
				root:            tt.fields.Root,
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

func TestMerklePatriciaTrie_GetNodeValue(t *testing.T) {
	// case 1
	pdb, cleanup := newPNodeDB(t)
	defer cleanup()

	pmpt := NewMerklePatriciaTrie(pdb, 0, Key("qwe"))

	// case 2

	mdb := NewMemoryNodeDB()
	key := "key"
	mdb.Nodes[StrKey(key)] = nil

	mmpt := NewMerklePatriciaTrie(mdb, 0, Key("key"))

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
		want    MPTSerializable
		wantErr bool
	}{
		{
			name: "Test_MerklePatriciaTrie_GetNodeValue_Not_Found_ERR",
			fields: fields{
				mutex:           &sync.RWMutex{},
				Root:            pmpt.GetRoot(),
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
				Root:            mmpt.GetRoot(),
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
				root:            tt.fields.Root,
				db:              tt.fields.db,
				ChangeCollector: tt.fields.ChangeCollector,
				Version:         tt.fields.Version,
			}
			mv := MockMPTSerializable{}

			err := mpt.GetNodeValue(tt.args.path, &mv)
			if tt.wantErr {
				require.Error(t, err, "GetNodeValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			require.Equal(t, &mv, tt.want)
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
		value MPTSerializable
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
				root:            tt.fields.Root,
				db:              tt.fields.db,
				ChangeCollector: tt.fields.ChangeCollector,
				Version:         tt.fields.Version,
			}

			got, err := mpt.Insert(tt.args.path, tt.args.value)
			if tt.wantErr {
				require.Error(t, err, fmt.Sprintf("Insert() error = %v, wantErr %v", err, tt.wantErr))
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Insert() got = %v, want %v", got, tt.want)
			}
		})
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
				root:            tt.fields.Root,
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

	ln := NewLeafNode(Path(""), Path("path"), 0, &SecureSerializableValue{Buffer: []byte("ln data")})
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
				root:            tt.fields.Root,
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
		visitNodeTypes byte
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantErr  bool
		wantHash string
	}{
		{
			name:     "TestMerklePatriciaTrie_Iterate_Empty_Tree_OK",
			fields:   fields{mutex: &sync.RWMutex{}},
			wantErr:  false,
			wantHash: "a7ffc6f8bf1ed76651c14756a061d662f580ff4de43b49fa82d80a4b80f8434a",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mpt := &MerklePatriciaTrie{
				mutex:           tt.fields.mutex,
				root:            tt.fields.Root,
				db:              tt.fields.db,
				ChangeCollector: tt.fields.ChangeCollector,
				Version:         tt.fields.Version,
			}
			sponge := sha3.New256()
			if err := mpt.Iterate(tt.args.ctx, iterSpongeHandler(sponge), tt.args.visitNodeTypes); (err != nil) != tt.wantErr {
				t.Errorf("Iterate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantHash != "" {
				iteratedHash := hex.EncodeToString(sponge.Sum(nil))
				if iteratedHash != tt.wantHash {
					t.Errorf("Iterate() hash = %v, want = %v", iteratedHash, tt.wantHash)
				}
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
			mpt := &MerklePatriciaTrie{
				mutex:           tt.fields.mutex,
				root:            tt.fields.Root,
				db:              tt.fields.db,
				ChangeCollector: tt.fields.ChangeCollector,
				Version:         tt.fields.Version,
			}
			v := &MockMPTSerializable{}
			err := mpt.getNodeValue(tt.args.path, tt.args.node, v)
			if tt.wantErr {
				require.Error(t, err, fmt.Errorf("getNodeValue() error = %v, wantErr %v", err, tt.wantErr))
				return
			}

			require.Equal(t, tt.want, v)
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
		value  MPTSerializable
		key    Key
		prefix Path
		path   Path
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
				root:            tt.fields.Root,
				db:              tt.fields.db,
				ChangeCollector: tt.fields.ChangeCollector,
				Version:         tt.fields.Version,
			}
			got, got1, err := mpt.insert(tt.args.value, tt.args.key, tt.args.prefix, tt.args.path)
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
	t.Parallel()
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
		value  MPTSerializable
		node   Node
		prefix Path
		path   Path
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
				node: NewLeafNode(Path(""), Path(""), 0, &SecureSerializableValue{}),
				path: Path("01"),
			},
			wantErr: true,
		},
		{
			name:   "Test_MerklePatriciaTrie_insertAtNode_Leaf_Node_ERR2",
			fields: fields{mutex: &sync.RWMutex{}, db: db},
			args: args{
				node: NewLeafNode(Path(""), path, 0, &SecureSerializableValue{}),
				path: append(path, []byte("123")...),
			},
			wantErr: true,
		},
		{
			name:   "Test_MerklePatriciaTrie_insertAtNode_Leaf_Node_ERR3",
			fields: fields{mutex: &sync.RWMutex{}, db: db},
			args: args{
				node: NewLeafNode(Path(""), append(path, []byte("098")...), 0, &SecureSerializableValue{}),
				path: append(path, []byte("123")...),
			},
			wantErr: true,
		},
		{
			name:   "Test_MerklePatriciaTrie_insertAtNode_Leaf_Node_ERR4",
			fields: fields{mutex: &sync.RWMutex{}, db: db},
			args: args{
				node: NewLeafNode(Path(""), append(path, []byte("098")...), 0, &SecureSerializableValue{}),
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
				root:            tt.fields.Root,
				db:              tt.fields.db,
				ChangeCollector: tt.fields.ChangeCollector,
				Version:         tt.fields.Version,
			}

			got, got1, err := mpt.insertAtNode(tt.args.value, tt.args.node, tt.args.prefix, tt.args.path)
			if tt.wantErr {
				require.Error(t, err, fmt.Errorf("insertAtNode() error = %v, wantErr %v", err, tt.wantErr))
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

func TestMerklePatriciaTrie_MergeChanges(t *testing.T) {
	t.Parallel()

	mpt := NewMerklePatriciaTrie(NewMemoryNodeDB(), 0, nil)

	mndb := NewMemoryNodeDB()
	mpt2 := NewMerklePatriciaTrie(mndb, 0, nil)
	doStrValInsert(t, mpt2, "1234", "test")

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
			name: "Test_MerklePatriciaTrie_MergeChanges_OK",
			fields: fields{
				mutex:           &sync.RWMutex{},
				Root:            mpt.GetRoot(),
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
			mpt := &MerklePatriciaTrie{
				mutex:           tt.fields.mutex,
				root:            tt.fields.Root,
				db:              tt.fields.db,
				ChangeCollector: tt.fields.ChangeCollector,
				Version:         tt.fields.Version,
			}
			if err := mpt.MergeChanges(mpt2.GetChanges()); (err != nil) != tt.wantErr {
				t.Errorf("MergeDB() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMerklePatriciaTrie_MergeMPTChanges(t *testing.T) {
	t.Parallel()

	DebugMPTNode = true

	db := NewLevelNodeDB(NewMemoryNodeDB(), NewMemoryNodeDB(), true)
	mpt := NewMerklePatriciaTrie(db, 0, nil)

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
				Root:            mpt.GetRoot(),
				db:              NewMemoryNodeDB(),
				ChangeCollector: mpt.ChangeCollector,
				Version:         mpt.Version,
			},
			args:    args{mpt2: NewMerklePatriciaTrie(NewMemoryNodeDB(), 0, nil)},
			wantErr: false,
		},
		{
			name: "Test_MerklePatriciaTrie_MergeMPTChanges_Invalid_MPT_DB_ERR",
			fields: fields{
				mutex:           &sync.RWMutex{},
				Root:            mpt.GetRoot(),
				db:              NewMemoryNodeDB(),
				ChangeCollector: mpt.ChangeCollector,
				Version:         mpt.Version,
			},
			args: func() args {
				mpt := NewMerklePatriciaTrie(NewMemoryNodeDB(), 0, Key("key"))

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
				Root:            mpt.GetRoot(),
				db:              NewMemoryNodeDB(),
				ChangeCollector: mpt.ChangeCollector,
				Version:         mpt.Version,
			},
			args: args{
				mpt2: func() *MerklePatriciaTrie {
					mpt := NewMerklePatriciaTrie(
						NewLevelNodeDB(NewMemoryNodeDB(), NewMemoryNodeDB(), false),
						0, Key("key"),
					)

					return mpt
				}(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mpt := &MerklePatriciaTrie{
				mutex:           tt.fields.mutex,
				root:            tt.fields.Root,
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

func TestMerklePatriciaTrie_IntegrityAfterValueUpdate(t *testing.T) {
	t.Parallel()
	db := NewLevelNodeDB(NewMemoryNodeDB(), NewMemoryNodeDB(), false)
	mpt := NewMerklePatriciaTrie(db, Sequence(0), nil)
	txn := &Txn{"1"}

	_, err := mpt.Insert(Path("00"), txn)
	require.NoError(t, err)
	checkIterationHash(t, mpt, "34a278944ef883d7c642a7b69b5675cf9d8cc5c60dd90d00adea1c4164425037")
	_, changes, _, _ := mpt.GetChanges()
	oldEncodedValue := changes[0].New.Encode()
	txn.Data = "2"
	checkIterationHash(t, mpt, "34a278944ef883d7c642a7b69b5675cf9d8cc5c60dd90d00adea1c4164425037")
	_, changes, _, _ = mpt.GetChanges()
	assert.Equal(t, 1, len(changes))
	assert.Equal(t, oldEncodedValue, changes[0].New.Encode())
}

func TestMerklePatriciaTrie_Validate(t *testing.T) {
	t.Parallel()

	pndb, cleanup := newPNodeDB(t)
	defer cleanup()

	mpt := NewMerklePatriciaTrie(nil, 0, nil)

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
				Root:            mpt.GetRoot(),
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
				Root:            mpt.GetRoot(),
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
				Root:  mpt.GetRoot(),
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
				Root:  mpt.GetRoot(),
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
			mpt := &MerklePatriciaTrie{
				mutex:           tt.fields.mutex,
				root:            tt.fields.Root,
				db:              tt.fields.db,
				ChangeCollector: tt.fields.ChangeCollector,
				Version:         tt.fields.Version,
			}
			if err := mpt.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
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
			args:    args{mpt: NewMerklePatriciaTrie(NewMemoryNodeDB(), 0, nil)},
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

func TestMPTInsertABC(t *testing.T) {
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(0), nil)
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
	mpt2 := NewMerklePatriciaTrie(db, Sequence(0), nil)

	doStrValInsert(t, mpt2, "12345897", "earth")
	doStrValInsert(t, mpt2, "1234", "mars")
	doStrValInsert(t, mpt2, "1234", "mars")
}

func TestMPTDeleteSameEndingPathNode(t *testing.T) {
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(0), nil)
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
	mpt2 := NewMerklePatriciaTrie(db, Sequence(0), nil)

	doStrValInsert(t, mpt2, "1245", "1234")
	doStrValInsert(t, mpt2, "12", "12")
	doStrValInsert(t, mpt2, "2345", "1234")
	doStrValInsert(t, mpt2, "23", "23")

	doDelete(t, mpt2, "1245", nil)
	doDelete(t, mpt2, "2345", nil)
}
func TestMPTFullToLeafNodeDelete(t *testing.T) {
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(0), nil)
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
	mpt2 := NewMerklePatriciaTrie(db, Sequence(0), nil)

	doStrValInsert(t, mpt2, "1245", "1234")
	doStrValInsert(t, mpt2, "12", "12")
	doStrValInsert(t, mpt2, "2345", "1234")
	doStrValInsert(t, mpt2, "23", "23")

	doDelete(t, mpt2, "1245", nil)
}
