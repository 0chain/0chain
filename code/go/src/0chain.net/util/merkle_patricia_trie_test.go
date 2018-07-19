package util

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"testing"
)

func TestMPTHexachars(t *testing.T) {
	cc := NewChangeCollector()
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb)
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.DB, false)
	var mpt2 MerklePatriciaTrieI = NewMerklePatriciaTrie(db)

	doStrValInsert("insert a leaf node as root", cc, mpt2, "1", "1", true)
	doStrValInsert("insert a leaf to create full node as root", cc, mpt2, "2", "2", true)
	doStrValInsert("insert a leaf node with hexa char", cc, mpt2, "a", "a", true)

	t.Logf("mpt root: %v\n", string(mpt2.GetRoot()))
	printChanges(cc)
}

func TestMPTInsertLeafNode(t *testing.T) {
	cc := NewChangeCollector()
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb)
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.DB, false)
	mpt2 := NewMerklePatriciaTrie(db)

	doStrValInsert("insert a leaf node as root", cc, mpt2, "1234", "1", true)

	doStrValInsert("split the leaf node into an extension node", cc, mpt2, "12356", "2", true)

	doStrValInsert("setup for later test case", cc, mpt2, "123671", "3", false)

	doStrValInsert("setup for later test case", cc, mpt2, "1237123", "4", false)

	doStrValInsert("setup for later test case", cc, mpt2, "12381234", "5", false)

	doStrValInsert("setup for later test case", cc, mpt2, "12391234", "6", false)

	doStrValInsert("update leaf node with no path", cc, mpt2, "1234", "1.1", true)

	doStrValInsert("extend leaf node with no path", cc, mpt2, "12345", "1.1.1", true)

	doStrValInsert("update leaf node with single path element", cc, mpt2, "12356", "2.1", true)

	doStrValInsert("extend leaf node with single path element", cc, mpt2, "123567", "2.1.1", true)

	doStrValInsert("update leaf node with multiple path elements", cc, mpt2, "123671", "3.1", true)

	doStrValInsert("extend leaf node with multiple path elements", cc, mpt2, "1236711", "3.1.1", true)

	doStrValInsert("break leaf node with multiple path elements creating an extension node and one leafs", cc, mpt2, "123712", "4.1", true)

	doStrValInsert("break leaf node with multiple path elements creating an extension node and two leafs", cc, mpt2, "1238124", "5.1", true)

	doStrValInsert("break leaf node with multiple path elements creating a full node", cc, mpt2, "1239", "6.1", true)
}

func TestMPTInsertFullNode(t *testing.T) {
	cc := NewChangeCollector()
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb)
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.DB, false)
	mpt2 := NewMerklePatriciaTrie(db)

	doStrValInsert("insert a leaf node as root", cc, mpt2, "1", "1", true)
	doStrValInsert("insert a leaf node to create a full node as root node", cc, mpt2, "2", "2", true)

	doStrValInsert("setup data", cc, mpt2, "11", "11", true)
	doStrValInsert("convert leaf to full node", cc, mpt2, "12", "12", true)

	doStrValInsert("setup data", cc, mpt2, "211", "211", true)

	doStrValInsert("convert leaf with path to full node with two leaves", cc, mpt2, "212", "212", true)

	doStrValInsert("setup data", cc, mpt2, "3", "3", true)
	doStrValInsert("setup data", cc, mpt2, "3112", "3112", true)
	doStrValInsert("convert leaf with path to extension node with two leaves", cc, mpt2, "3113", "3113", true)

}

func TestMPTInsertExtensionNode(t *testing.T) {
	cc := NewChangeCollector()
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb)
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.DB, false)
	mpt2 := NewMerklePatriciaTrie(db)

	doStrValInsert("insert a leaf node as root", cc, mpt2, "12345", "12345", true)
	doStrValInsert("insert a leaf to create an extension node as root node", cc, mpt2, "12346", "12346", true)

	doStrValInsert("break extension into full node at the beginning", cc, mpt2, "2", "2", true)
	doStrValInsert("break extension into full node at the middle", cc, mpt2, "123", "123", true)

	doStrValInsert("setup data", cc, mpt2, "22345", "22345", false)
	doStrValInsert("setup data", cc, mpt2, "22346", "22346", true)

	doStrValInsert("extend extension", cc, mpt2, "22347", "22347", true)

	doStrValInsert("sibling to extension", cc, mpt2, "23", "23", true)

	doStrValInsert("update value along an extension path", cc, mpt2, "12345", "12345.1", true)

	doStrValInsert("add value at the path of an extension node", cc, mpt2, "2234", "2234", true)

	doStrValInsert("add value at the extension node", cc, mpt2, "22", "22", true)

}

func TestMPTDelete(t *testing.T) {
	cc := NewChangeCollector()
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb)
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.DB, false)
	mpt2 := NewMerklePatriciaTrie(db)
	doStrValInsert("insert a leaf node as root", cc, mpt2, "12345", "12345", false)
	doStrValInsert("insert a leaf to create a full root node", cc, mpt2, "22345", "22345", false)

	doStrValInsert("insert a leaf to create a full root node", cc, mpt2, "123", "123", false)
	doStrValInsert("insert a leaf to create a full root node", cc, mpt2, "124", "124", false)

	doStrValInsert("insert a value at full node", cc, mpt2, "12", "12", false)
	doStrValInsert("create an extension path", cc, mpt2, "34567", "34567", false)
	doStrValInsert("create an extension path", cc, mpt2, "34577", "34577", true)

	doStrValInsert("insert a leaf node", cc, mpt2, "412345", "412345", false)
	doStrValInsert("insert a leaf node to create a full node", cc, mpt2, "42234", "42234", false)
	doStrValInsert("insert a leaf node to create a second node which is an extension", cc, mpt2, "412346", "412346", false)
	doStrValInsert("insert a leaf node to convert extension to full node", cc, mpt2, "513346", "513346", false)

	doStrValInsert("insert a leaf node", cc, mpt2, "512345", "512345", false)
	doStrValInsert("insert a leaf node to create a full node", cc, mpt2, "52234", "52234", false)
	doStrValInsert("insert a leaf node to create a second node which is an extension", cc, mpt2, "512346", "512346", false)

	doStrValInsert("insert a leaf node", cc, mpt2, "612345", "612345", false)
	doStrValInsert("insert a leaf node", cc, mpt2, "612512", "612512", false)
	doStrValInsert("insert a leaf node to create a full node under the child of the extension node", cc, mpt2, "612522", "612522", true)

	doDelete("delete a leaf node as root", cc, mpt2, "12345", true)
	doDelete("delete value of a full node", cc, mpt2, "12", true)
	doDelete("delete a leaf from a full node with two children and no value", cc, mpt2, "34577", true)
	doDelete("delete a single leaf of a full node with value", cc, mpt2, "124", true)

	// lift up
	doDelete("delete a leaf node and lift up extension node", cc, mpt2, "42234", true)
	doDelete("delete a leaf node and lift up full node", cc, mpt2, "52234", true)
	doStrValInsert("delete a leaf node so the only other full node is lifted up", cc, mpt2, "612345", "", true)

	// delete not existent node
	doDelete("delete non existent node", cc, mpt2, "abcdef123", true)
	doDelete("delete non existent node detected at leaf", cc, mpt2, "6125123", true)
	doDelete("delete non existent node detected at extension", cc, mpt2, "613512", true)

}

func TestMPTUniverse(t *testing.T) {
	cc := NewChangeCollector()
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb)
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.DB, false)
	mpt2 := NewMerklePatriciaTrie(db)

	doStrValInsert("root node with a single leaf", cc, mpt2, "1234513", "earth", true)

	doStrValInsert("break the leaf node into an extension node and a full node containing child nodes", cc, mpt2, "123451478", "mars", true)

	doStrValInsert("add a value to the full node pointed to by the extension", cc, mpt2, "123451", "mercury", true)

	doStrValInsert("break the extension so it creates a new full node that contains the two children", cc, mpt2, "123455", "jupiter", true)

	doStrValInsert("add a value at the path pointed by the extension (same use case as adding mercury above)", cc, mpt2, "12345", "sun", true)

	doStrValInsert("extend an existing leaf node (leaf becomes full node with a leaf node child)", cc, mpt2, "12345131131", "moon", true)

	// Add a bunch of child nodes to existing full node
	doStrValInsert("", cc, mpt2, "123456", "saturn", false)
	doStrValInsert("", cc, mpt2, "123457", "uranus", false)
	doStrValInsert("", cc, mpt2, "123458", "neptune", false)
	doStrValInsert("more data", cc, mpt2, "123459", "pluto", true)

	doStrValInsert("update value at a leaf node", cc, mpt2, "123459", "dwarf planet", true)

	doStrValInsert("update value at a full node", cc, mpt2, "1234513", "green earth and ham", true)

	doStrValInsert("break the leaf node into an extension node and a full node with value and a child", cc, mpt2, "1234514781", "phobos", true)

	doStrValInsert("break the leaf node into full node with the value & a child leaf node with the added value", cc, mpt2, "1234556", "europa", true)

	doStrValInsert("", cc, mpt2, "123452", "venus", false)

	doStrValInsert("", cc, mpt2, "123", "world", true)

	cc = NewChangeCollector() // adding a new change collector so there are changes with old nodes that are not nil

	doStrValInsert("", cc, mpt2, "12346", "proxima centauri", true)
	doStrValInsert("", cc, mpt2, "1", "hello", true)

	mpt2.Iterate(context.TODO(), iterHandler, NodeTypeValueNode|NodeTypeLeafNode|NodeTypeFullNode|NodeTypeExtensionNode)
}

func TestMPTInsertEthereumExample(t *testing.T) {
	cc := NewChangeCollector()
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb)
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.DB, false)
	mpt2 := NewMerklePatriciaTrie(db)

	doStrValInsert("setup data", cc, mpt2, "646f", "verb", false)
	doStrValInsert("setup data", cc, mpt2, "646f67", "puppy", false)
	doStrValInsert("setup data", cc, mpt2, "646f6765", "coin", false)
	doStrValInsert("setup data", cc, mpt2, "686f727365", "stallion", true)

	mpt2.Iterate(context.TODO(), iterHandler, NodeTypeLeafNode|NodeTypeFullNode|NodeTypeExtensionNode)

	printChanges(cc)
}

func doStrValInsert(testcase string, cc ChangeCollectorI, mpt MerklePatriciaTrieI, key string, value string, print bool) {
	newRoot, err := mpt.Insert([]byte(key), &Txn{value}, cc)
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
	mpt.SetRoot(newRoot)
	if print {
		fmt.Printf("test: %v [%v,%v]\n", testcase, key, value)
		mpt.PrettyPrint(os.Stdout)
		fmt.Printf("\n")
	}
	doGetStrValue(mpt, key, value)
}

func doGetStrValue(mpt MerklePatriciaTrieI, key string, value string) {
	val, err := mpt.GetNodeValue([]byte(key))
	if value == "" {
		if !(val == nil || err == ErrValueNotPresent) {
			fmt.Printf("error: setting value to blank didn't return nil value: %v, %v", val, err)
			panic("doGetStrValueError")
		}
		return
	}
	if err != nil {
		fmt.Printf("error: getting inserted value: %v %v", key, value)
		panic("doGetStrValueError")
	}
	if val == nil {
		fmt.Printf("error: inserted value not found: %v %v", key, value)
		panic("doGetStrValueError")
	}
}

func iterHandler(ctx context.Context, path Path, key Key, node Node) error {
	vn, ok := node.(*ValueNode)
	if ok {
		fmt.Printf("iterate:%20s: p=%v k=%v v=%v\n", fmt.Sprintf("%T", node), hex.EncodeToString(path), hex.EncodeToString(key), string(vn.GetValue().Encode()))
	} else {
		fmt.Printf("iterate:%20s: p=%v k=%v\n", fmt.Sprintf("%T", node), hex.EncodeToString(path), hex.EncodeToString(key))
	}
	return nil
}

func doDelete(testcase string, cc ChangeCollectorI, mpt MerklePatriciaTrieI, key string, print bool) {
	if print {
		fmt.Printf("test: %v [%v]\n", testcase, key)
	}
	newRoot, err := mpt.Delete([]byte(key), cc)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}
	mpt.SetRoot(newRoot)
	if print {
		mpt.PrettyPrint(os.Stdout)
		fmt.Printf("\n")
	}
	doGetStrValue(mpt, key, "")
}

func printChanges(cc ChangeCollectorI) {
	changes := cc.GetChanges()
	fmt.Printf("number of changes: %v\n", len(changes))
	for _, change := range changes {
		if change.Old == nil {
			fmt.Printf("cc: (nil) %v -> (%T) %v\n", nil, change.New, string(change.New.GetHashBytes()))
		} else {
			fmt.Printf("cc: (%T) %v -> (%T) %v\n", change.Old, string(change.Old.GetHashBytes()), change.New, string(change.New.GetHashBytes()))
		}
	}

	for _, change := range cc.GetDeletes() {
		fmt.Printf("d: %T %v\n", change, string(change.GetHashBytes()))
	}
}
