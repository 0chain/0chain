package util

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"testing"
)

func TestMPTHexachars(t *testing.T) {
	cc := NewChangeCollector()
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(2018))
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
	var mpt2 MerklePatriciaTrieI = NewMerklePatriciaTrie(db, Sequence(2018))

	doStrValInsert(t, "insert a leaf node as root", mpt2, "1", "1")
	doStrValInsert(t, "insert a leaf to create full node as root", mpt2, "2", "2")
	doStrValInsert(t, "insert a leaf node with hexa char", mpt2, "a", "a")

	t.Logf("mpt root: %v", string(mpt2.GetRoot()))
	printChanges(t, cc)
}

func TestMPTInsertLeafNode(t *testing.T) {
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(0))
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
	mpt2 := NewMerklePatriciaTrie(db, Sequence(0))

	doStrValInsert(t, "insert a leaf node as root", mpt2, "1234", "1")
	doStrValInsert(t, "split the leaf node into an extension node", mpt2, "12356", "2")
	doStrValInsert(t, "setup for later test case", mpt2, "123671", "3")
	doStrValInsert(t, "setup for later test case", mpt2, "1237123", "4")
	doStrValInsert(t, "setup for later test case", mpt2, "12381234", "5")
	doStrValInsert(t, "setup for later test case", mpt2, "12391234", "6")

	nodes, err := mpt2.GetPathNodes(Path("12391234"))
	if err != nil {
		panic(err)
	}
	for idx, nd := range nodes {
		t.Logf("n:%v:%v", idx, string(nd.GetHash()))
	}

	doStrValInsert(t, "update leaf node with no path", mpt2, "1234", "1.1")
	doStrValInsert(t, "extend leaf node with no path", mpt2, "12345", "1.1.1")
	doStrValInsert(t, "update leaf node with single path element", mpt2, "12356", "2.1")
	doStrValInsert(t, "extend leaf node with single path element", mpt2, "123567", "2.1.1")
	doStrValInsert(t, "update leaf node with multiple path elements", mpt2, "123671", "3.1")
	doStrValInsert(t, "extend leaf node with multiple path elements", mpt2, "1236711", "3.1.1")
	doStrValInsert(t, "break leaf node with multiple path elements creating an extension node and one leafs", mpt2, "123712", "4.1")
	doStrValInsert(t, "break leaf node with multiple path elements creating an extension node and two leafs", mpt2, "1238124", "5.1")
	doStrValInsert(t, "break leaf node with multiple path elements creating a full node", mpt2, "1239", "6.1")
}

func TestMPTInsertFullNode(t *testing.T) {
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(0))
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
	mpt2 := NewMerklePatriciaTrie(db, Sequence(0))

	doStrValInsert(t, "insert a leaf node as root", mpt2, "1", "1")
	doStrValInsert(t, "insert a leaf node to create a full node as root node", mpt2, "2", "2")
	doStrValInsert(t, "setup data", mpt2, "11", "11")
	doStrValInsert(t, "convert leaf to full node", mpt2, "12", "12")
	doStrValInsert(t, "setup data", mpt2, "211", "211")
	doStrValInsert(t, "convert leaf with path to full node with two leaves", mpt2, "212", "212")
	doStrValInsert(t, "setup data", mpt2, "3", "3")
	doStrValInsert(t, "setup data", mpt2, "3112", "3112")
	doStrValInsert(t, "convert leaf with path to extension node with two leaves", mpt2, "3113", "3113")
}

func TestMPTInsertExtensionNode(t *testing.T) {
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(0))
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
	mpt2 := NewMerklePatriciaTrie(db, Sequence(0))

	doStrValInsert(t, "insert a leaf node as root", mpt2, "12345", "12345")
	doStrValInsert(t, "insert a leaf to create an extension node as root node", mpt2, "12346", "12346")
	doStrValInsert(t, "break extension into full node at the beginning", mpt2, "2", "2")
	mpt2.Iterate(context.TODO(), iterStrPathHandler(t), NodeTypeLeafNode|NodeTypeFullNode|NodeTypeExtensionNode)
	doStrValInsert(t, "break extension into full node at the middle", mpt2, "123", "123")
	doStrValInsert(t, "setup data", mpt2, "22345", "22345")
	doStrValInsert(t, "setup data", mpt2, "22346", "22346")
	doStrValInsert(t, "extend extension", mpt2, "22347", "22347")
	doStrValInsert(t, "sibling to extension", mpt2, "23", "23")
	doStrValInsert(t, "update value along an extension path", mpt2, "12345", "12345.1")
	doStrValInsert(t, "add value at the path of an extension node", mpt2, "2234", "2234")
	doStrValInsert(t, "add value at the extension node", mpt2, "22", "22")
}

func TestMPTDelete(t *testing.T) {
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(0))
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
	mpt2 := NewMerklePatriciaTrie(db, Sequence(0))

	doStrValInsert(t, "insert a leaf node as root", mpt2, "12345", "12345")
	doStrValInsert(t, "insert a leaf to create a full root node", mpt2, "22345", "22345")

	doStrValInsert(t, "insert a leaf to create a full root node", mpt2, "123", "123")
	doStrValInsert(t, "insert a leaf to create a full root node", mpt2, "124", "124")

	doStrValInsert(t, "insert a value at full node", mpt2, "12", "12")
	doStrValInsert(t, "create an extension path", mpt2, "34567", "34567")
	doStrValInsert(t, "create an extension path", mpt2, "34577", "34577")

	doStrValInsert(t, "insert a leaf node", mpt2, "412345", "412345")
	doStrValInsert(t, "insert a leaf node to create a full node", mpt2, "42234", "42234")
	doStrValInsert(t, "insert a leaf node to create a second node which is an extension", mpt2, "412346", "412346")
	doStrValInsert(t, "insert a leaf node to convert extension to full node", mpt2, "513346", "513346")

	doStrValInsert(t, "insert a leaf node", mpt2, "512345", "512345")
	doStrValInsert(t, "insert a leaf node to create a full node", mpt2, "52234", "52234")
	doStrValInsert(t, "insert a leaf node to create a second node which is an extension", mpt2, "512346", "512346")

	doStrValInsert(t, "insert a leaf node", mpt2, "612345", "612345")
	doStrValInsert(t, "insert a leaf node", mpt2, "612512", "612512")
	doStrValInsert(t, "insert a leaf node to create a full node under the child of the extension node", mpt2, "612522", "612522")

	doDelete(t, "delete a leaf node as root", mpt2, "12345")
	doDelete(t, "delete value of a full node", mpt2, "12")
	doDelete(t, "delete a leaf from a full node with two children and no value", mpt2, "34577")
	doDelete(t, "delete a single leaf of a full node with value", mpt2, "124")

	// lift up
	doDelete(t, "delete a leaf node and lift up extension node", mpt2, "42234")
	doDelete(t, "delete a leaf node and lift up full node", mpt2, "52234")
	doStrValInsert(t, "delete a leaf node so the only other full node is lifted up", mpt2, "612345", "")

	// delete not existent node
	doDelete(t, "delete non existent node", mpt2, "abcdef123")
	doDelete(t, "delete non existent node detected at leaf", mpt2, "6125123")
	doDelete(t, "delete non existent node detected at extension", mpt2, "613512")

}

func TestMPTUniverse(t *testing.T) {
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(0))
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
	mpt2 := NewMerklePatriciaTrie(db, Sequence(0))

	doStrValInsert(t, "root node with a single leaf", mpt2, "1234513", "earth")
	doStrValInsert(t, "break the leaf node into an extension node and a full node containing child nodes", mpt2, "123451478", "mars")
	doStrValInsert(t, "add a value to the full node pointed to by the extension", mpt2, "123451", "mercury")
	doStrValInsert(t, "break the extension so it creates a new full node that contains the two children", mpt2, "123455", "jupiter")
	doStrValInsert(t, "add a value at the path pointed by the extension (same use case as adding mercury above)", mpt2, "12345", "sun")
	doStrValInsert(t, "extend an existing leaf node (leaf becomes full node with a leaf node child)", mpt2, "12345131131", "moon")

	// Add a bunch of child nodes to existing full node
	doStrValInsert(t, "", mpt2, "123456", "saturn")
	doStrValInsert(t, "", mpt2, "123457", "uranus")
	doStrValInsert(t, "", mpt2, "123458", "neptune")
	doStrValInsert(t, "more data", mpt2, "123459", "pluto")

	doStrValInsert(t, "update value at a leaf node", mpt2, "123459", "dwarf planet")
	doStrValInsert(t, "update value at a full node", mpt2, "1234513", "green earth and ham")
	doStrValInsert(t, "break the leaf node into an extension node and a full node with value and a child", mpt2, "1234514781", "phobos")
	doStrValInsert(t, "break the leaf node into full node with the value & a child leaf node with the added value", mpt2, "1234556", "europa")
	doStrValInsert(t, "", mpt2, "123452", "venus")
	doStrValInsert(t, "", mpt2, "123", "world")

	mpt.ResetChangeCollector(mpt.GetRoot()) // adding a new change collector so there are changes with old nodes that are not nil

	doStrValInsert(t, "", mpt2, "12346", "proxima centauri")
	doStrValInsert(t, "", mpt2, "1", "hello")

	mpt2.Iterate(context.TODO(), iterHandler(t), NodeTypeValueNode|NodeTypeLeafNode|NodeTypeFullNode|NodeTypeExtensionNode)

	key, err := hex.DecodeString("aabaed5911cb89fe95680df9f42e07c5bb147fc7a742bde7cb5be62419eb41bf")
	if err != nil {
		t.Fatal(err)
	}
	t.Log("iterating from intermediate node")
	mpt2.IterateFrom(context.TODO(), key, iterHandler(t),
		NodeTypeValueNode|NodeTypeLeafNode|NodeTypeFullNode|NodeTypeExtensionNode)
}

func TestMPTInsertEthereumExample(t *testing.T) {
	cc := NewChangeCollector()
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(0))
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
	mpt2 := NewMerklePatriciaTrie(db, Sequence(0))

	doStrValInsert(t, "setup data", mpt2, "646f", "verb")
	doStrValInsert(t, "setup data", mpt2, "646f67", "puppy")
	doStrValInsert(t, "setup data", mpt2, "646f6765", "coin")
	doStrValInsert(t, "setup data", mpt2, "686f727365", "stallion")

	mpt2.Iterate(context.TODO(), iterHandler(t), NodeTypeLeafNode|NodeTypeFullNode|NodeTypeExtensionNode)
	printChanges(t, cc)
}

func prettyPrint(t *testing.T, mpt MerklePatriciaTrieI) {
	var buf bytes.Buffer
	mpt.PrettyPrint(&buf)
	t.Log(buf.String())
}

func doStrValInsert(t *testing.T, testcase string, mpt MerklePatriciaTrieI,
	key, value string) {

	t.Helper()

	newRoot, err := mpt.Insert(Path(key), &Txn{value})
	if err != nil {
		t.Error(err)
	}

	mpt.SetRoot(newRoot)
	t.Logf("test: %v [%v,%v]", testcase, key, value)
	prettyPrint(t, mpt)
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

func iterHandler(t *testing.T) func(ctx context.Context, path Path, key Key, node Node) error {
	return func(ctx context.Context, path Path, key Key, node Node) error {
		if node == nil {
			return fmt.Errorf("stop")
		}
		vn, ok := node.(*ValueNode)
		if ok {
			t.Logf("iterate:%20s: p=%v k=%v v=%v\n", fmt.Sprintf("%T", node),
				hex.EncodeToString(path), hex.EncodeToString(key),
				string(vn.GetValue().Encode()))
		} else {
			t.Logf("iterate:%20s: orig=%v ver=%v p=%v k=%v\n",
				fmt.Sprintf("%T", node), node.GetOrigin(), node.GetVersion(),
				hex.EncodeToString(path), hex.EncodeToString(key))
		}
		return nil
	}
}

func iterStrPathHandler(t *testing.T) func(ctx context.Context, path Path, key Key, node Node) error {
	return func(ctx context.Context, path Path, key Key, node Node) error {
		if node == nil {
			return fmt.Errorf("stop")
		}
		if vn, ok := node.(*ValueNode); ok {
			t.Logf("iterate:%20s: p=%v k=%v v=%v", fmt.Sprintf("%T", node),
				string(path), hex.EncodeToString(key),
				string(vn.GetValue().Encode()))
		} else {
			var val interface{}
			if ln, ok := node.(*LeafNode); ok {
				val = ln.GetValue()
			}
			t.Logf("iterate:%20s: orig=%v ver=%v p=%v k=%v v=%v",
				fmt.Sprintf("%T", node), node.GetOrigin(), node.GetVersion(),
				string(path), hex.EncodeToString(key), val)
		}
		return nil
	}
}

func doDelete(t *testing.T, testcase string, mpt MerklePatriciaTrieI, key string) {
	t.Logf("test: %v [%v]", testcase, key)
	newRoot, err := mpt.Delete([]byte(key))
	if err != nil {
		t.Error(err)
		return
	}
	mpt.SetRoot(newRoot)
	prettyPrint(t, mpt)
	doGetStrValue(t, mpt, key, "")
}

func printChanges(t *testing.T, cc ChangeCollectorI) {
	changes := cc.GetChanges()
	t.Log("number of changes:", len(changes))
	for _, change := range changes {
		if change.Old == nil {
			t.Logf("cc: (nil) %v -> (%T) %v",
				nil, change.New, string(change.New.GetHashBytes()))
		} else {
			t.Logf("cc: (%T) %v -> (%T) %v",
				change.Old, string(change.Old.GetHashBytes()), change.New,
				string(change.New.GetHashBytes()))
		}
	}

	for _, change := range cc.GetDeletes() {
		t.Logf("d: %T %v", change, string(change.GetHashBytes()))
	}
}

/*
  merge extensions : delete L from P(E(F(L,E))) and ensure P(E(F(E))) becomes P(E)
*/
func TestCasePEFLEdeleteL(t *testing.T) {
	cc := NewChangeCollector()
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(0))
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
	mpt2 := NewMerklePatriciaTrie(db, Sequence(0))

	doStrValInsert(t, "setup data", mpt2, "223456789", "mercury")
	doStrValInsert(t, "setup data", mpt2, "1235", "venus")
	doStrValInsert(t, "setup data", mpt2, "123458970", "earth")
	doStrValInsert(t, "setup data", mpt2, "123459012", "mars")
	doStrValInsert(t, "setup data", mpt2, "123459013", "jupiter")
	doStrValInsert(t, "setup data", mpt2, "123459023", "saturn")
	doStrValInsert(t, "setup data", mpt2, "123459024", "uranus")

	doDelete(t, "delete a leaf node and merge the extension node", mpt2, "1235")
	doStrValInsert(t, "reinsert data", mpt2, "1235", "venus")
	doDelete(t, "delete a leaf node and merge the extension node", mpt2, "1235")
	doStrValInsert(t, "update after delete", mpt2, "12345903", "neptune")

	mpt2.Iterate(context.TODO(), iterHandler(t), NodeTypeLeafNode|NodeTypeFullNode|NodeTypeExtensionNode)

	printChanges(t, cc)

	v, err := mpt2.GetNodeValue(Path("123458970"))
	if err != nil {
		t.Error(err)
	} else {
		t.Logf("%+v", v)
	}
}

func TestAddTwiceDeleteOnce(t *testing.T) {
	cc := NewChangeCollector()
	mndb := NewMemoryNodeDB()
	mpt := NewMerklePatriciaTrie(mndb, Sequence(0))
	db := NewLevelNodeDB(NewMemoryNodeDB(), mpt.db, false)
	mpt2 := NewMerklePatriciaTrie(db, Sequence(0))

	doStrValInsert(t, "setup data", mpt2, "123456781", "x")
	doStrValInsert(t, "setup data", mpt2, "123456782", "y")
	//doStrValInsert(t,"setup data", mpt2, "123556782", "z")

	doStrValInsert(t, "setup data", mpt2, "223456781", "x")
	doStrValInsert(t, "setup data", mpt2, "223456782", "y")

	prettyPrint(t, mpt2)

	doStrValInsert(t, "setup data", mpt2, "223456782", "a")
	//doStrValInsert(t,"setup data", mpt2, "223556782", "b")

	//mpt2.Iterate(context.TODO(), iterHandler, NodeTypeLeafNode /*|NodeTypeFullNode|NodeTypeExtensionNode */)
	prettyPrint(t, mpt2)
	printChanges(t, cc)

	//doDelete("delete a leaf node", mpt2, "123456781", true)
	//mpt2.PrettyPrint(os.Stdout)

	//doDelete("delete a leaf node", mpt2, "223556782", true)
	prettyPrint(t, mpt2)
}
