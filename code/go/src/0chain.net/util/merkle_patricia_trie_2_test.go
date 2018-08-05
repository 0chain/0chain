package util

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
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
	pndb, err := NewPNodeDB("/tmp/mpt")
	if err != nil {
		panic(err)
	}
	defer pndb.db.Close()

	mpt := NewMerklePatriciaTrie(pndb)
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.DB, false)
	mpt2 := NewMerklePatriciaTrie(db)

	doStateValInsert("add 100 to c1", mpt2, "0123456", 100, false)
	doStateValInsert("add 1000 to c2", mpt2, "0123457", 1000, false)
	doStateValInsert("add 1000 to c3", mpt2, "0123458", 1000000, false)
	doStateValInsert("add 1000 to c4", mpt2, "0133458", 1000000000, true)

	printChanges(mpt2.GetChangeCollector())

	err = mpt2.SaveChanges(pndb, Origin(2016), false)
	if err != nil {
		panic(err)
	}

	err = mpt2.Iterate(context.TODO(), iterHandler, NodeTypeValueNode|NodeTypeLeafNode|NodeTypeFullNode|NodeTypeExtensionNode)

	mpt.SetRoot(mpt2.GetRoot())

	fmt.Printf("\nReading from persistent db\n")
	err = mpt.Iterate(context.TODO(), iterHandler, NodeTypeValueNode|NodeTypeFullNode|NodeTypeExtensionNode)
	if err != nil {
		fmt.Printf("iterate error: %v\n", err)
	}
}

func TestMerkeTreePruning(t *testing.T) {
	pndb, err := NewPNodeDB("/tmp/mpt")
	if err != nil {
		panic(err)
	}
	defer pndb.db.Close()
	if err != nil {
		panic(err)
	}

	mpt := NewMerklePatriciaTrie(pndb)
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.DB, false)
	mpt2 := NewMerklePatriciaTrie(db)
	origin := 2016
	roots := make([]Key, 0, 10)
	for i := int64(0); i < 1000; i++ {
		mpt2.ResetChangeCollector(mpt2.GetRoot())
		doStateValInsert("add 100 to c1", mpt2, "0123456", 100+i, false)
		doStateValInsert("add 1000 to c2", mpt2, "0123457", 1000+i, false)
		doStateValInsert("add 1000 to c3", mpt2, "0123458", 1000000+i, false)
		doStateValInsert("add 1000 to c4", mpt2, "0133458", 1000000000+i, true)
		roots = append(roots, mpt2.GetRoot())
		fmt.Printf("root(%v) = %v: changes: %v\n", origin, ToHex(mpt2.GetRoot()), len(mpt.GetChangeCollector().GetChanges()))
		err = mpt2.SaveChanges(pndb, Origin(origin), false)
		if err != nil {
			panic(err)
		}
		mpt.SetRoot(mpt2.GetRoot())
		mpt.PrettyPrint(os.Stdout)
		fmt.Printf("\n")
		origin++
	}
	numStates := 200
	newOrigin := Origin(origin - numStates)
	root := roots[len(roots)-numStates]
	fmt.Printf("pruning to origin: %v %v\n", newOrigin, ToHex(root))
	mpt.SetRoot(root)
	mpt.PrettyPrint(os.Stdout)
	fmt.Printf("\n")

	err = mpt.Iterate(context.TODO(), iterHandler, NodeTypeValueNode|NodeTypeFullNode|NodeTypeExtensionNode)
	if err != nil {
		fmt.Printf("iterate error: %v\n", err)
	}

	err = mpt.UpdateOrigin(context.TODO(), newOrigin)
	if err != nil {
		fmt.Printf("error updating origin: %v\n", err)
	}

	mpt.SetRoot(mpt2.GetRoot())
	err = mpt.Iterate(context.TODO(), iterHandler, NodeTypeValueNode|NodeTypeFullNode|NodeTypeExtensionNode)
	if err != nil {
		fmt.Printf("iterate error: %v\n", err)
	}

	err = mpt.PruneBelowOrigin(context.TODO(), newOrigin)
	if err != nil {
		fmt.Printf("error pruning origin: %v\n", err)
	}

	mpt.SetRoot(mpt2.GetRoot())
	err = mpt.Iterate(context.TODO(), iterHandler, NodeTypeValueNode|NodeTypeFullNode|NodeTypeExtensionNode)
	if err != nil {
		fmt.Printf("iterate error: %v\n", err)
	}
}

func TestMerkeTreeGetChanges(t *testing.T) {
	pndb, err := NewPNodeDB("/tmp/mpt")
	if err != nil {
		panic(err)
	}
	defer pndb.db.Close()
	if err != nil {
		panic(err)
	}

	mpt := NewMerklePatriciaTrie(pndb)
	var mndb = NewMemoryNodeDB()
	db := NewLevelNodeDB(mndb, mpt.DB, false)
	mpt2 := NewMerklePatriciaTrie(db)
	origin := 2016
	roots := make([]Key, 0, 10)
	for i := int64(0); i < 10; i++ {
		mpt2.ResetChangeCollector(mpt2.GetRoot())
		doStateValInsert("add 100 to c1", mpt2, "0123456", 100+i, false)
		doStateValInsert("add 1000 to c2", mpt2, "0123457", 1000+i, false)
		doStateValInsert("add 1000 to c3", mpt2, "0123458", 1000000+i, false)
		doStateValInsert("add 1000 to c4", mpt2, "0133458", 1000000000+i, false)
		roots = append(roots, mpt2.GetRoot())
		fmt.Printf("root(%v) = %v: changes: %v ndb size: %v\n", origin, ToHex(mpt2.GetRoot()), len(mpt2.GetChangeCollector().GetChanges()), len(mndb.Nodes))
		err = mpt2.SaveChanges(pndb, Origin(origin), false)
		if err != nil {
			panic(err)
		}
		//mpt2.PrettyPrint(os.Stdout)
		origin++
	}
	fmt.Printf("get changes\n")
	mpts, err := GetChanges(context.TODO(), mndb, Origin(origin-3), Origin(origin))
	if err != nil {
		panic(err)
	}
	for origin, mpt := range mpts {
		fmt.Printf("origin:%v: root:%v\n", origin, ToHex(mpt.GetRoot()))
		mpt.PrettyPrint(os.Stdout)
		mpt.Iterate(context.TODO(), iterHandler, NodeTypeValueNode)
	}
}

func doStateValInsert(testcase string, mpt MerklePatriciaTrieI, key string, value int64, print bool) {
	state := &AState{}
	state.balance = value
	newRoot, err := mpt.Insert([]byte(key), state)
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
	mpt.SetRoot(newRoot)
	if print {
		fmt.Printf("test: %v [%v,%v]\n", testcase, key, value)
		mpt.PrettyPrint(os.Stdout)
		fmt.Printf("\n")
	}
	doGetStateValue(mpt, key, value)
}

func doGetStateValue(mpt MerklePatriciaTrieI, key string, value int64) {
	val, err := mpt.GetNodeValue([]byte(key))
	if err != nil {
		fmt.Printf("error: getting inserted value: %v %v", key, value)
		panic("doGetStrValueError")
	}
	if val == nil {
		fmt.Printf("error: inserted value not found: %v %v", key, value)
		panic("doGetStrValueError")
	}
}

func stateIterHandler(ctx context.Context, path Path, key Key, node Node) error {
	vn, ok := node.(*ValueNode)
	if ok {
		astate := &AState{}
		astate.Decode(vn.GetValue().Encode())
		fmt.Printf("iterate:%20s: p=%v k=%v v=%v\n", fmt.Sprintf("%T", node), hex.EncodeToString(path), hex.EncodeToString(key), astate.balance)
	} else {
		fmt.Printf("iterate:%20s: p=%v k=%v\n", fmt.Sprintf("%T", node), hex.EncodeToString(path), hex.EncodeToString(key))
	}
	return nil
}
