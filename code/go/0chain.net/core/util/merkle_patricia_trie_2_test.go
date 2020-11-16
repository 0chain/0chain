package util

import (
	"context"
	"encoding/hex"
	"fmt"
	"strconv"
	"testing"
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

func TestMerkleTreeSaveToDB(t *testing.T) {
	pndb, err := NewPNodeDB("/tmp/mpt", "/tmp/mpt/log")
	if err != nil {
		t.Fatal(err)
	}
	defer pndb.db.Close()

	mpt := NewMerklePatriciaTrie(pndb, Sequence(2016))
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
	mpt2 := NewMerklePatriciaTrie(db, Sequence(2016))

	doStateValInsert(t, "add 100 to c1", mpt2, "0123456", 100)
	doStateValInsert(t, "add 1000 to c2", mpt2, "0123457", 1000)
	doStateValInsert(t, "add 1000 to c3", mpt2, "0123458", 1000000)
	doStateValInsert(t, "add 1000 to c4", mpt2, "0133458", 1000000000)

	printChanges(t, mpt2.GetChangeCollector())

	err = mpt2.SaveChanges(pndb, false)
	if err != nil {
		t.Error(err)
	}

	err = mpt2.Iterate(context.TODO(), iterHandler(t),
		NodeTypeValueNode|NodeTypeLeafNode|NodeTypeFullNode|NodeTypeExtensionNode)

	mpt.SetRoot(mpt2.GetRoot())

	t.Logf("Reading from persistent db")
	err = mpt.Iterate(context.TODO(), iterHandler(t),
		NodeTypeValueNode|NodeTypeFullNode|NodeTypeExtensionNode)
	if err != nil {
		t.Errorf("iterate error: %v", err)
	}
}

func TestMerkeTreePruning(t *testing.T) {
	pndb, err := NewPNodeDB("/tmp/mpt", "/tmp/mpt/log")
	if err != nil {
		t.Fatal(err)
	}
	defer pndb.db.Close()

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
		t.Logf("root(%v) = %v: changes: %v\n", origin, ToHex(mpt2.GetRoot()),
			len(mpt2.GetChangeCollector().GetChanges()))
		err = mpt2.SaveChanges(pndb, false)
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
	t.Logf("pruning to origin: %v %v", newOrigin, ToHex(root))
	mpt.SetRoot(root)
	prettyPrint(t, mpt)

	err = mpt.Iterate(context.TODO(), iterHandler(t),
		NodeTypeValueNode|NodeTypeFullNode|NodeTypeExtensionNode)
	if err != nil {
		t.Errorf("iterate error: %v", err)
	}

	pndb.Iterate(context.TODO(), dbIteratorHandler(t))

	missingNodeHandler := func(ctx context.Context, path Path, key Key) error {
		t.Logf("missing node: %v %v", path, key)
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
	t.Log("pruning db")
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
	pndb, err := NewPNodeDB("/tmp/mpt", "/tmp/mpt/log")
	if err != nil {
		t.Fatal(err)
	}
	defer pndb.db.Close()

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

		t.Logf("root(%v) = %v: changes: %v ndb size: %v", origin,
			ToHex(mpt2.GetRoot()), len(mpt2.GetChangeCollector().GetChanges()),
			len(mndb.Nodes))

		if err = mpt2.SaveChanges(pndb, false); err != nil {
			panic(err)
		}

		prettyPrint(t, mpt2)
		origin++
	}

	t.Log("get changes")
	mpts, err := GetChanges(context.TODO(), mndb, Sequence(origin-3),
		Sequence(origin))
	if err != nil {
		t.Error(err)
	}

	for origin, mpt := range mpts {
		t.Logf("origin: %v: root: %v", origin, ToHex(mpt.GetRoot()))
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

	t.Logf("test: %v [%v,%v]", testcase, key, value)
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
		t.Logf("iterate:%20s:  p=%v k=%v v=%v", fmt.Sprintf("%T", node),
			hex.EncodeToString(path), hex.EncodeToString(key), astate.balance)
	} else {
		t.Logf("iterate:%20s: orig=%v ver=%v p=%v k=%v",
			fmt.Sprintf("%T", node), node.GetOrigin(), node.GetVersion(),
			hex.EncodeToString(path), hex.EncodeToString(key))
	}

	return nil
}

func dbIteratorHandler(t *testing.T) func(ctx context.Context, key Key, node Node) error {
	return func(ctx context.Context, key Key, node Node) error {
		t.Logf("iteratedb: %v %v %v", ToHex(key), node.GetOrigin(),
			string(node.Encode()))
		return nil
	}
}
