package util

import (
	"context"
	"encoding/hex"
	"fmt"
	"hash"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"
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
	exp := "ae6a645401f35411371b9d498fa13c663909a7f6463b42a7f2a060db3ef0196b"
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

	for i := int64(0); i < 1000; i++ {
		db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
		mpt2 := NewMerklePatriciaTrie(db, Sequence(0), mpt.GetRoot())
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
		require.NoError(t, mpt2.SaveChanges(context.TODO(), mpt.GetNodeDB(), false))
		origin++
	}

	numStates := 200
	newOrigin := Sequence(origin - numStates)
	root := roots[len(roots)-numStates]
	mpt = NewMerklePatriciaTrie(mpt.GetNodeDB(), mpt.GetVersion(), root)

	checkIterationHash(t, mpt, "84b88172c473aa0350b2841269ff8e58c81fb72f5c8ce877e8c130b7092fddc5")

	assert.NoError(t, pndb.Iterate(context.TODO(), dbIteratorHandler()))

	missingNodeHandler := func(ctx context.Context, path Path, key Key) error {
		return nil
	}
	err := mpt.UpdateVersion(context.TODO(), newOrigin, missingNodeHandler)
	if err != nil {
		t.Error("error updating origin:", err)
	}

	// mpt = NewMerklePatriciaTrie(mpt.GetNodeDB(), mpt.GetVersion(), mpt.GetRoot())
	checkIterationHash(t, mpt, "84b88172c473aa0350b2841269ff8e58c81fb72f5c8ce877e8c130b7092fddc5")
	err = pndb.PruneBelowVersion(context.TODO(), newOrigin)
	if err != nil {
		t.Error("error pruning origin:", err)
	}

	assert.NoError(t, pndb.Iterate(context.TODO(), dbIteratorHandler()))

	checkIterationHash(t, mpt, "84b88172c473aa0350b2841269ff8e58c81fb72f5c8ce877e8c130b7092fddc5")
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

	val, err := mpt.GetNodeValue([]byte(key))
	if err != nil {
		t.Fatalf("getting inserted value: %v %v, err: %v", key, value, err)
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

// calculates hash of all sorted keys in the NodeDB
func calculateKeysHash(t *testing.T, ndb NodeDB) string {
	sponge := valuesSponge{}
	err := ndb.Iterate(context.TODO(), dbKeysSpongeHandler(&sponge))
	require.NoError(t, err)
	sort.Strings(sponge.values)
	hash := sha3.New256()
	for _, key := range sponge.values {
		b, err := hex.DecodeString(key)
		require.NoError(t, err)
		hash.Write(b)
	}
	return hex.EncodeToString(hash.Sum(nil))
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

	var mpt = NewMerklePatriciaTrie(stateDB, 0, nil)

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
	exp := "7ea96443c31290349e030f572c55c73153dc6822d4b1419391df530db0360ac5"
	if iteratedHash != exp {
		t.Fatalf("calculated sequence mismatch: %v, %v",
			iteratedHash, exp)
	}
	rootHash := ToHex(mpt2.root)
	exp = "1d113cf8005c4ab38a7ca31d8cc345fe3875c259eb54ed1bd9b031f2565e8015"
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
	exp = "3624e73be093af74c884eea162070ff5eabcbad4a0fb605d8208cada970117a9"
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
	assert.Equal(t, "b4297fd80bb162a0f766f71197a07690bcb6c2ec198fa02678cb057af0c04276", ToHex(mpt2.root))
	checkValues(t, mpt2, []string{"22345", "22346"})
	mpt2.ChangeCollector.GetChanges()

	doStrValInsert(t, mpt2, "223467", "22347")
	checkValues(t, mpt2, []string{"22345", "22347"})
	doStrValInsert(t, mpt2, "223467", "22346")
	checkValues(t, mpt2, []string{"22345", "22346"})
	assert.Equal(t, "b4297fd80bb162a0f766f71197a07690bcb6c2ec198fa02678cb057af0c04276", ToHex(mpt2.root))
}

func TestMPT_MultipleConcurrentInserts(t *testing.T) {
	t.Parallel()
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
	checkIterationHash(t, mpt, "49989099964c9dff77435c4bee926c76c64006724af5f1efc0deb95488dbff9e")
	mpt2 := NewMerklePatriciaTrie(ldb, Sequence(0), mpt.GetRoot())
	checkIterationHash(t, mpt2, "49989099964c9dff77435c4bee926c76c64006724af5f1efc0deb95488dbff9e")
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
	checkIterationHash(t, mpt2, "3f056cecd45427bc466681a2fe01594a70a50161c66708aec400970f799ef935")
	checkIterationHash(t, mpt, "49989099964c9dff77435c4bee926c76c64006724af5f1efc0deb95488dbff9e")
	require.NoError(t, mpt.MergeMPTChanges(mpt2))
	checkIterationHash(t, mpt, "3f056cecd45427bc466681a2fe01594a70a50161c66708aec400970f799ef935")
}

func TestMPT_ConcurrentMerges(t *testing.T) {
	t.Parallel()
	db := NewLevelNodeDB(NewMemoryNodeDB(), NewMemoryNodeDB(), false)
	mpt := NewMerklePatriciaTrie(db, Sequence(0), nil)
	ldb := NewLevelNodeDB(NewMemoryNodeDB(), db, false)
	numGoRoutines := 10
	numTxns := 10
	txns := make([]*Txn, numGoRoutines*numTxns)
	for i := 0; i < len(txns); i++ {
		txns[i] = &Txn{fmt.Sprintf("%v", len(txns)-i)}
	}
	// insert some of the nodes to the original mpt
	for i := 0; i < numGoRoutines; i++ {
		_, err := mpt.Insert(Path(encryption.Hash(txns[i*numTxns].Data)), txns[i*numTxns])
		require.NoError(t, err)
	}
	mpt2 := NewMerklePatriciaTrie(ldb, Sequence(0), mpt.GetRoot())
	wg := &sync.WaitGroup{}
	for i := 0; i < numGoRoutines; i++ {
		wg.Add(1)
		go func(mpt2 MerklePatriciaTrieI, i int) {
			defer wg.Done()
			for j := 1; j < numTxns; j++ {
				_, err := mpt2.Insert(Path(encryption.Hash(txns[i*numTxns+j].Data)), txns[i*numTxns+j])
				require.NoError(t, err)
				mpt.MergeMPTChanges(mpt2) // may produce error because of optimistic lock failure
				// the transient mpt state contains no missing nodes
				require.NoError(t, mpt.Iterate(context.TODO(), iterNopHandler(), NodeTypeLeafNode|NodeTypeFullNode|NodeTypeExtensionNode))
			}
		}(mpt2, i)
	}
	wg.Wait()
	checkIterationHash(t, mpt2, "e746a622dca7212732dd74521edf4f336b5134321513343819efb74f981f1925")
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
	exp := "2f2ad6f1c18ee4808abde751e08dd2129109338a7866e522b9c5b7796f62f5fc"
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
	exp = "d76edb5b0e5cda8625c81593fd2bccaede906f35610a3e6de2809a862514f30b"
	if iteratedHash != exp {
		t.Fatalf("calculated sequence mismatch: %v, %v",
			iteratedHash, exp)
	}

	key, err := hex.DecodeString("14e6f2fd08c3ba3bc816d16d6af63965e5d82eb7db22761d67b8d63a4e21f1f4")
	if err != nil {
		t.Fatal(err)
	}
	sponge = sha3.New256()
	err = mpt2.IterateFrom(context.TODO(), key, iterSpongeHandler(sponge),
		NodeTypeValueNode|NodeTypeLeafNode|NodeTypeFullNode|NodeTypeExtensionNode)
	if err != nil {
		t.Fatal(err)
	}
	iteratedHash = hex.EncodeToString(sponge.Sum(nil))
	exp = "74869fa61802795b687cdfc2f4a34c71d522444022fad9250d6f19e030ce3fce"
	if iteratedHash != exp {
		t.Fatalf("calculated sequence mismatch: %v, %v",
			iteratedHash, exp)
	}
	// collect values
	valuesSponge := valuesSponge{make([]string, 0, 16)}
	err = mpt2.IterateFrom(context.TODO(), key, iterValuesSpongeHandler(&valuesSponge), NodeTypeValueNode)
	if err != nil {
		t.Fatal(err)
	}
	values := strings.Join(valuesSponge.values, ",")
	// starting with "12345", should miss "hello" and "world"
	expValues := "sun,mercury,green earth and ham,moon,mars,phobos,venus,jupiter,europa,saturn,uranus,neptune,dwarf planet,proxima centauri"
	if values != expValues {
		t.Fatalf("Actual values %v differ from expected %v", values, expValues)
	}
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
	exp := "720a6fff8f2b30647b94a2d801cd1baedcb7e8648a293697550720dcb42405be"
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
	exp = "b1d2d3eae3fb008eb00a456ad63a6e446355f6ebf279fcd261d1ab119c1aa325"
	if iteratedHash != exp {
		t.Fatalf("calculated sequence mismatch: %v, %v",
			iteratedHash, exp)
	}
	doDelete(t, mpt2, "686f727365", nil)
	rootHash = ToHex(mpt2.root)
	exp = "720a6fff8f2b30647b94a2d801cd1baedcb7e8648a293697550720dcb42405be"
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
	exp = "21ccd3041d44d826c06332204d1a3e2c56114e4b831ac8521c1762695c545239"
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
	readValue := string(val.Encode())
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
			sponge.Write(vn.Value.Encode())
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
			sponge.values = append(sponge.values, string(vn.Value.Encode()))
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
	expWithVenus := "4af37dce8b6a8cd3e11b134231963b30eee6b95842f56ca2eab49a5cb0aa52bf"
	if rootHash != expWithVenus {
		t.Fatalf("root hash mismatch: %v, %v",
			rootHash, expWithVenus)
	}
	doDelete(t, mpt2, "1235", nil)
	rootHash = hex.EncodeToString(mpt2.root)
	expWithoutVenus := "500096406b887e6f1c7d13dd4ee9522b44da0a7581e120a9c845211586b70b2b"
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
	exp := "107357c93cf035864ca972b38d4992d0f7529113bfdd7e15bb3d3db1843237cd"
	if rootHash != exp {
		t.Fatalf("root hash mismatch: %v, %v", rootHash, exp)
	}
	checkIterationHash(t, mpt2, "6a15c5ff1772339a49e4bca1cff7d3b38accb6ac15af37a6907024a4e2861391")
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
	v, err := mpt2.GetNodeValue(Path("1234589701"))
	if err != nil {
		t.Error(err)
	}
	value := string(v.Encode())
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
	//doStrValInsert(t,"setup data", mpt2, "123556782", "z")

	doStrValInsert(t, mpt2, "2234567812", "x")
	doStrValInsert(t, mpt2, "2234567822", "y")

	doStrValInsert(t, mpt2, "2234567822", "a")
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
				root:            tt.fields.Root,
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
			t.Parallel()

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
		want    Serializable
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
				root:            tt.fields.Root,
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
			t.Parallel()

			mpt := &MerklePatriciaTrie{
				mutex:           tt.fields.mutex,
				root:            tt.fields.Root,
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
		value  Serializable
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
		value  Serializable
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
			t.Parallel()

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
			t.Parallel()

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
			t.Parallel()

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

	if err := cleanUp(); err != nil {
		t.Fatal(err)
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

func TestMerklePatriciaTrie_UpdateVersion(t *testing.T) {
	t.Parallel()

	mpt := NewMerklePatriciaTrie(nil, 0, nil)
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
				root:            tt.fields.Root,
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
