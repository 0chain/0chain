package util

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"github.com/0chain/gorocksdb"
	"reflect"
	"strconv"
	"sync"
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

	_, err := mpt2.GetPathNodes(Path("12391234"))
	if err != nil {
		t.Fatal(err)
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

	doDelete(t, "delete a leaf node as root", mpt2, "12345", nil)
	doDelete(t, "delete value of a full node", mpt2, "12", nil)
	doDelete(t, "delete a leaf from a full node with two children and no value", mpt2, "34577", nil)
	doDelete(t, "delete a single leaf of a full node with value", mpt2, "124", nil)

	// lift up
	doDelete(t, "delete a leaf node and lift up extension node", mpt2, "42234", nil)
	doDelete(t, "delete a leaf node and lift up full node", mpt2, "52234", nil)
	doStrValInsert(t, "delete a leaf node so the only other full node is lifted up", mpt2, "612345", "")

	// delete not existent node
	doDelete(t, "delete non existent node", mpt2, "abcdef123", ErrNodeNotFound)
	doDelete(t, "delete non existent node detected at leaf", mpt2, "6125123", ErrNodeNotFound)
	doDelete(t, "delete non existent node detected at extension", mpt2, "613512", ErrNodeNotFound)

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
}

func doStrValInsert(t *testing.T, testcase string, mpt MerklePatriciaTrieI,
	key, value string) {

	t.Helper()

	newRoot, err := mpt.Insert(Path(key), &Txn{value})
	if err != nil {
		t.Error(err)
	}

	mpt.SetRoot(newRoot)
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
		return nil
	}
}

func iterStrPathHandler(t *testing.T) func(ctx context.Context, path Path, key Key, node Node) error {
	return func(ctx context.Context, path Path, key Key, node Node) error {
		if node == nil {
			return fmt.Errorf("stop")
		}
		return nil
	}
}

func doDelete(t *testing.T, testcase string, mpt MerklePatriciaTrieI,
	key string, expErr error) {

	newRoot, err := mpt.Delete([]byte(key))
	if err != expErr {
		t.Error(err)
		return
	}
	mpt.SetRoot(newRoot)
	prettyPrint(t, mpt)
	doGetStrValue(t, mpt, key, "")
}

func printChanges(t *testing.T, cc ChangeCollectorI) {
	_ = cc.GetChanges()
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

	doDelete(t, "delete a leaf node and merge the extension node", mpt2, "1235", nil)
	doStrValInsert(t, "reinsert data", mpt2, "1235", "venus")
	doDelete(t, "delete a leaf node and merge the extension node", mpt2, "1235", nil)
	doStrValInsert(t, "update after delete", mpt2, "12345903", "neptune")

	mpt2.Iterate(context.TODO(), iterHandler(t), NodeTypeLeafNode|NodeTypeFullNode|NodeTypeExtensionNode)

	printChanges(t, cc)

	_, err := mpt2.GetNodeValue(Path("123458970"))
	if err != nil {
		t.Error(err)
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
		t.Run(tt.name, func(t *testing.T) {
			if got := CloneMPT(tt.args.mpt); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CloneMPT() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMerklePatriciaTrie_SetNodeDB(t *testing.T) {
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
		t.Run(tt.name, func(t *testing.T) {
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
		t.Run(tt.name, func(t *testing.T) {
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
	pdb, err := NewPNodeDB(dataDir, "")
	if err != nil {
		t.Error(err)
	}
	defer pdb.Close()

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

	if err := cleanUp(); err != nil {
		t.Error(err)
	}
}

func TestMerklePatriciaTrie_Insert(t *testing.T) {
	db, err := NewPNodeDB(dataDir, "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
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
		t.Run(tt.name, func(t *testing.T) {
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
	db.PutNode(keyFn, fn)
	db.PutNode(keyFn1, fn1)
	db.PutNode(keyLn, ln)
	db.PutNode(keyEn, en)

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
		t.Run(tt.name, func(t *testing.T) {
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

func TestMerklePatriciaTrie_SaveChanges(t *testing.T) {
	db, err := NewPNodeDB(dataDir, "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
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
		ndb            NodeDB
		includeDeletes bool
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "Test_MerklePatriciaTrie_SaveChanges_ERR",
			fields:  fields{mutex: &sync.RWMutex{}, ChangeCollector: &ChangeCollector{}},
			args:    args{ndb: db},
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
			if err := mpt.SaveChanges(tt.args.ndb, tt.args.includeDeletes); (err != nil) != tt.wantErr {
				t.Errorf("SaveChanges() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	if err := cleanUp(); err != nil {
		t.Fatal(err)
	}
}

func TestMerklePatriciaTrie_Iterate(t *testing.T) {
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
		t.Run(tt.name, func(t *testing.T) {
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
	fn := NewFullNode(&SecureSerializableValue{Buffer: []byte("fn data")})
	keyFn := Key(fn.GetHash())
	fn1 := NewFullNode(&SecureSerializableValue{Buffer: []byte("data")})
	keyFn1 := Key(fn1.GetHash())
	fn1.Children[0] = NewFullNode(&SecureSerializableValue{Buffer: []byte("children data")}).Encode()

	keyEn := Key("key")
	en := NewExtensionNode(Path("path"), keyEn)

	db := NewMemoryNodeDB()
	db.PutNode(keyFn, fn)
	db.PutNode(keyFn1, fn1)

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
		t.Run(tt.name, func(t *testing.T) {
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
		t.Run(tt.name, func(t *testing.T) {
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
	db, err := NewPNodeDB(dataDir, "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
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

	if err := cleanUp(); err != nil {
		t.Fatal(err)
	}
}

func TestMerklePatriciaTrie_MergeDB(t *testing.T) {
	mpt := NewMerklePatriciaTrie(NewMemoryNodeDB(), 0)

	mndb := NewMemoryNodeDB()
	mndb.PutNode(Key("key"), NewValueNode())

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
		t.Run(tt.name, func(t *testing.T) {
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
		t.Run(tt.name, func(t *testing.T) {
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
	pndb, err := NewPNodeDB(dataDir, "")
	if err != nil {
		t.Fatal(err)
	}
	defer pndb.Close()

	mpt := NewMerklePatriciaTrie(nil, 0)

	lndb := NewLevelNodeDB(NewMemoryNodeDB(), NewMemoryNodeDB(), true)
	n := NewFullNode(&SecureSerializableValue{Buffer: []byte("value")})
	lndb.PutNode(n.GetHashBytes(), n)

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
		t.Run(tt.name, func(t *testing.T) {
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
	ndb := NewMemoryNodeDB()
	n := NewFullNode(&SecureSerializableValue{Buffer: []byte("data")})
	ndb.PutNode(n.GetHashBytes(), n)

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
		t.Run(tt.name, func(t *testing.T) {
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
		t.Run(tt.name, func(t *testing.T) {
			if err := IsMPTValid(tt.args.mpt); (err != nil) != tt.wantErr {
				t.Errorf("IsMPTValid() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMerklePatriciaTrie_UpdateVersion(t *testing.T) {
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
					db.PutNode(n.GetHashBytes(), n)

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
					db.PutNode(root, n)

					for i := 0; i < BatchSize+1; i++ {
						n := NewExtensionNode([]byte("root"), []byte("key"))
						n.NodeKey = []byte(strconv.Itoa(i + 1))
						db.PutNode([]byte(strconv.Itoa(i)), n)
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
		t.Run(tt.name, func(t *testing.T) {
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
