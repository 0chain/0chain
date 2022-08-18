package util

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"0chain.net/core/logging"
	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

/*MerklePatriciaTrie - it's a merkle tree and a patricia trie */
type MerklePatriciaTrie struct {
	mutex           *sync.RWMutex
	root            Key
	db              NodeDB
	ChangeCollector ChangeCollectorI
	Version         Sequence
}

/*NewMerklePatriciaTrie - create a new patricia merkle trie */
func NewMerklePatriciaTrie(db NodeDB, version Sequence, root Key) *MerklePatriciaTrie {
	mpt := &MerklePatriciaTrie{
		mutex: &sync.RWMutex{},
		db:    db,
	}
	mpt.root = root
	mpt.ChangeCollector = NewChangeCollector(root)
	mpt.SetVersion(version)
	return mpt
}

//CloneMPT - clone an existing MPT so it can go off of a different root
func CloneMPT(mpt MerklePatriciaTrieI) *MerklePatriciaTrie {
	clone := NewMerklePatriciaTrie(mpt.GetNodeDB(), mpt.GetVersion(), mpt.GetRoot())
	return clone
}

/*SetNodeDB - implement interface */
func (mpt *MerklePatriciaTrie) SetNodeDB(ndb NodeDB) {
	mpt.mutex.Lock()
	defer mpt.mutex.Unlock()
	if lndb, ok := mpt.db.(*LevelNodeDB); ok {
		lndb.RebaseCurrentDB(ndb)
	}
	//mpt.db = ndb
}

/*GetNodeDB - implement interface */
func (mpt *MerklePatriciaTrie) GetNodeDB() NodeDB {
	mpt.mutex.RLock()
	defer mpt.mutex.RUnlock()
	return mpt.db
}

func (mpt *MerklePatriciaTrie) getNodeDB() NodeDB {
	return mpt.db
}

//SetVersion - implement interface
func (mpt *MerklePatriciaTrie) SetVersion(version Sequence) {
	current := (*int64)(&mpt.Version)
	atomic.StoreInt64(current, int64(version))
}

//GetVersion - implement interface
func (mpt *MerklePatriciaTrie) GetVersion() Sequence {
	current := (*int64)(&mpt.Version)
	return Sequence(atomic.LoadInt64(current))
}

func (mpt *MerklePatriciaTrie) setRoot(root Key) {
	mpt.root = root
}

/*GetRoot - implement interface */
func (mpt *MerklePatriciaTrie) GetRoot() Key {
	mpt.mutex.RLock()
	defer mpt.mutex.RUnlock()
	return mpt.root
}

/*GetNodeValue - get the value for a given path */
func (mpt *MerklePatriciaTrie) GetNodeValue(path Path, v MPTSerializable) error {
	d, err := mpt.GetNodeValueRaw(path)
	if err != nil {
		return err
	}

	_, err = v.UnmarshalMsg(d)
	return err
}

// GetNodeValueRaw gets the raw data slice for a given path without decodding
func (mpt *MerklePatriciaTrie) GetNodeValueRaw(path Path) ([]byte, error) {
	if _, err := hex.DecodeString(string(path)); err != nil {
		return nil, fmt.Errorf("invalid hex path: path=%q, err=%v", string(path), err)
	}

	mpt.mutex.RLock()
	defer mpt.mutex.RUnlock()

	rootKey := []byte(mpt.root)
	if len(rootKey) == 0 {
		return nil, ErrValueNotPresent
	}

	rootNode, err := mpt.db.GetNode(rootKey)
	if err != nil {
		return nil, err
	}
	if rootNode == nil {
		return nil, ErrNodeNotFound
	}

	return mpt.getNodeValueRaw(path, rootNode)
}

/*Insert - inserts (updates) a value into this trie and updates the trie all the way up and produces a new root */
func (mpt *MerklePatriciaTrie) Insert(path Path, value MPTSerializable) (Key, error) {
	if value == nil {
		Logger.Debug("Insert nil value, delete data on path:",
			zap.String("path", string(path)))
		return mpt.Delete(path)
	}
	eval, err := value.MarshalMsg(nil)
	if err != nil {
		return nil, err
	}

	if len(eval) == 0 {
		Logger.Debug("Insert encoded nil value, delete data on path:",
			zap.String("path", string(path)))
		return mpt.Delete(path)
	}

	valueCopy := &SecureSerializableValue{eval}
	mpt.mutex.Lock()
	defer mpt.mutex.Unlock()
	var newRootHash Key
	if mpt.root == nil {
		_, newRootHash, err = mpt.insertLeaf(nil, valueCopy, Path(""), path)
	} else {
		_, newRootHash, err = mpt.insert(valueCopy, mpt.root, Path(""), path)
	}
	if err != nil {
		return nil, err
	}
	mpt.setRoot(newRootHash)
	return newRootHash, nil
}

/*Delete - delete a value from the trie */
func (mpt *MerklePatriciaTrie) Delete(path Path) (Key, error) {
	mpt.mutex.Lock()
	defer mpt.mutex.Unlock()

	_, newRootHash, err := mpt.delete(mpt.root, Path(""), path)
	if err != nil {
		return nil, err
	}
	mpt.setRoot(newRootHash)
	return newRootHash, nil
}

//GetPathNodes - implement interface */
func (mpt *MerklePatriciaTrie) GetPathNodes(path Path) ([]Node, error) {
	mpt.mutex.RLock()
	defer mpt.mutex.RUnlock()

	nodes, err := mpt.getPathNodes(mpt.root, path)
	if err != nil {
		return nil, err
	}
	for i, j := 0, len(nodes)-1; i < j; i, j = i+1, j-1 {
		nodes[i], nodes[j] = nodes[j], nodes[i]
	}
	return nodes, nil
}

func (mpt *MerklePatriciaTrie) getPathNodes(key Key, path Path) ([]Node, error) {
	if len(path) == 0 {
		return nil, nil
	}
	node, err := mpt.db.GetNode(key)
	if err != nil {
		return nil, err
	}
	switch nodeImpl := node.(type) {
	case *LeafNode:
		if bytes.Equal(nodeImpl.Path, path) {
			return []Node{node}, nil
		}
		return nil, ErrValueNotPresent
	case *FullNode:
		ckey := nodeImpl.GetChild(path[0])
		if ckey == nil {
			return nil, ErrValueNotPresent
		}
		npath, err := mpt.getPathNodes(ckey, path[1:])
		if err != nil {
			return nil, err
		}
		npath = append(npath, node)
		return npath, nil
	case *ExtensionNode:
		prefix := mpt.matchingPrefix(path, nodeImpl.Path)
		if len(prefix) == 0 {
			return nil, ErrValueNotPresent
		}
		if bytes.Equal(nodeImpl.Path, prefix) {
			npath, err := mpt.getPathNodes(nodeImpl.NodeKey, path[len(prefix):])
			if err != nil {
				return nil, err
			}
			npath = append(npath, node)
			return npath, nil
		}
		return nil, ErrValueNotPresent
	default:
		panic(fmt.Sprintf("unknown node type: %T %v", node, node))
	}
}

/*GetChanges - implement interface */
func (mpt *MerklePatriciaTrie) GetChanges() (Key, []*NodeChange, []Node, Key) {
	mpt.mutex.RLock()
	defer mpt.mutex.RUnlock()
	return mpt.root, mpt.ChangeCollector.GetChanges(), mpt.ChangeCollector.GetDeletes(), mpt.ChangeCollector.GetStartRoot()
}

func (mpt *MerklePatriciaTrie) GetDeletes() []Node {
	var nodes []Node
	mpt.mutex.RLock()
	nodes = mpt.ChangeCollector.GetDeletes()
	mpt.mutex.RUnlock()
	return nodes
}

/*GetChangeCount - implement interface */
func (mpt *MerklePatriciaTrie) GetChangeCount() int {
	mpt.mutex.RLock()
	defer mpt.mutex.RUnlock()
	return len(mpt.ChangeCollector.GetChanges())
}

/*SaveChanges - implement interface */
func (mpt *MerklePatriciaTrie) SaveChanges(ctx context.Context, ndb NodeDB, includeDeletes bool) error {
	mpt.mutex.RLock()
	defer mpt.mutex.RUnlock()
	cc := mpt.ChangeCollector

	doneC := make(chan struct{})
	errC := make(chan error, 1)
	ts := time.Now()
	go func() {
		defer func() {
			close(doneC)
			logging.Logger.Debug("MPT save changes success", zap.Any("duration", time.Since(ts)))
		}()
		err := cc.UpdateChanges(ndb, mpt.Version, includeDeletes)
		if err != nil {
			logging.Logger.Error("MPT save changes failed",
				zap.Any("version", mpt.Version),
				zap.Error(err))
			errC <- err
		}
	}()

	select {
	case <-ctx.Done():
		Logger.Debug("MPT save changes timeout",
			zap.Any("duration", time.Since(ts)),
			zap.Error(ctx.Err()))
		return ctx.Err()
	case err := <-errC:
		Logger.Debug("MPT save changes failed", zap.Error(err))
		return err
	case <-doneC:
	}
	return nil
}

/*Iterate - iterate the entire trie */
func (mpt *MerklePatriciaTrie) Iterate(ctx context.Context, handler MPTIteratorHandler, visitNodeTypes byte) error {
	mpt.mutex.RLock()
	defer mpt.mutex.RUnlock()

	rootKey := mpt.root
	// it might be nil or empty
	if len(rootKey) == 0 { //nolint
		return nil
	}
	return mpt.iterate(ctx, Path{}, rootKey, handler, visitNodeTypes)
}

/*IterateFrom - iterate the trie from a given node */
func (mpt *MerklePatriciaTrie) IterateFrom(ctx context.Context, node Key, handler MPTIteratorHandler, visitNodeTypes byte) error {
	//NOTE: we don't have the path to this node. So, the handler gets the partial path starting from this node
	return mpt.iterate(ctx, Path{}, node, handler, visitNodeTypes)
}

/*PrettyPrint - print this trie */
func (mpt *MerklePatriciaTrie) PrettyPrint(w io.Writer) error {
	mpt.mutex.RLock()
	defer mpt.mutex.RUnlock()
	return mpt.pp(w, mpt.root, 0, false)
}

func (mpt *MerklePatriciaTrie) getNodeValue(path Path, node Node, v MPTSerializable) error {
	d, err := mpt.getNodeValueRaw(path, node)
	if err != nil {
		return err
	}

	_, err = v.UnmarshalMsg(d)
	return err
}

func (mpt *MerklePatriciaTrie) getNodeValueRaw(path Path, node Node) ([]byte, error) {
	switch nodeImpl := node.(type) {
	case *LeafNode:
		if bytes.Equal(nodeImpl.Path, path) {
			d := nodeImpl.GetValueBytes()
			if len(d) == 0 {
				return nil, ErrValueNotPresent
			}

			return d, nil
		}
		return nil, ErrValueNotPresent
	case *FullNode:
		if len(path) == 0 {
			d := nodeImpl.GetValueBytes()
			if len(d) == 0 {
				return nil, ErrValueNotPresent
			}

			return d, nil
		}
		ckey := nodeImpl.GetChild(path[0])
		if ckey == nil {
			return nil, ErrValueNotPresent
		}

		nnode, err := mpt.db.GetNode(ckey)
		if err != nil || nnode == nil {
			if err != nil {
				Logger.Error("full node get node failed",
					zap.Any("node version", nodeImpl.GetVersion()),
					zap.Any("mpt version", mpt.Version),
					zap.String("key", ToHex(ckey)),
					zap.Error(err))
			}
			return nil, ErrNodeNotFound
		}
		return mpt.getNodeValueRaw(path[1:], nnode)
	case *ExtensionNode:
		prefix := mpt.matchingPrefix(path, nodeImpl.Path)
		if len(prefix) == 0 {
			return nil, ErrValueNotPresent
		}
		if bytes.Equal(nodeImpl.Path, prefix) {
			nnode, err := mpt.db.GetNode(nodeImpl.NodeKey)
			if err != nil || nnode == nil {
				if err != nil {
					Logger.Error("extension node get node failed", zap.Error(err))
				}
				return nil, ErrNodeNotFound
			}
			return mpt.getNodeValueRaw(path[len(prefix):], nnode)
		}
		return nil, ErrValueNotPresent
	default:
		panic(fmt.Sprintf("unknown node type: %T %v", node, node))
	}
}

func (mpt *MerklePatriciaTrie) insert(value MPTSerializable, key Key, prefix, path Path) (Node, Key, error) {
	node, err := mpt.db.GetNode(key)
	if err != nil {
		return nil, nil, err
	}
	if len(path) == 0 {
		return mpt.insertAfterPathTraversal(value, node)
	}
	return mpt.insertAtNode(value, node, prefix, path)

}

func (mpt *MerklePatriciaTrie) insertLeaf(oldNode Node, value MPTSerializable, prefix, path Path) (Node, Key, error) {
	return mpt.insertNode(oldNode, NewLeafNode(prefix, path, mpt.Version, value))
}

func (mpt *MerklePatriciaTrie) insertExtension(oldNode Node, path Path, key Key) (Node, Key, error) {
	return mpt.insertNode(oldNode, NewExtensionNode(path, key))
}

func (mpt *MerklePatriciaTrie) delete(key Key, prefix, path Path) (Node, Key, error) {
	if key == nil {
		return nil, nil, ErrValueNotPresent
	}
	node, err := mpt.db.GetNode(key)
	if err != nil {
		return nil, nil, err
	}
	if len(path) == 0 {
		return mpt.deleteAfterPathTraversal(node)
	}
	return mpt.deleteAtNode(node, prefix, path)
}

func (mpt *MerklePatriciaTrie) insertAtNode(value MPTSerializable, node Node, prefix, path Path) (Node, Key, error) {
	var err error
	switch nodeImpl := node.(type) {
	case *FullNode:
		ckey := nodeImpl.GetChild(path[0])
		if ckey == nil {
			_, ckey, err = mpt.insertLeaf(nil, value,
				concat(prefix, path[:1]...), path[1:])
		} else {
			_, ckey, err = mpt.insert(value, ckey,
				concat(prefix, path[:1]...), path[1:])
		}
		if err != nil {
			return nil, nil, err
		}
		nnode := nodeImpl.Clone().(*FullNode)
		nnode.PutChild(path[0], ckey)
		return mpt.insertNode(node, nnode)
	case *LeafNode:
		if len(nodeImpl.Path) == 0 {
			_, ckey, err := mpt.insertLeaf(nil, value,
				concat(prefix, path[:1]...), path[1:])
			if err != nil {
				return nil, nil, err
			}

			nnode := NewFullNode(nodeImpl.GetValue())
			nnode.PutChild(path[0], ckey)
			return mpt.insertNode(node, nnode)
		}
		// updating an existing leaf
		if bytes.Equal(path, nodeImpl.Path) {
			return mpt.insertLeaf(node, value, concat(prefix), nodeImpl.Path)
		}

		matchPrefix := mpt.matchingPrefix(path, nodeImpl.Path)
		plen := len(matchPrefix)
		cnode := NewFullNode(nil)
		if bytes.Equal(matchPrefix, path) {
			// path is a prefix of the existing leaf (node.Path = "hello world", path = "hello")
			_, gckey2, err := mpt.insertLeaf(nil, nodeImpl.GetValue(),
				concat(prefix, nodeImpl.Path[:plen+1]...), nodeImpl.Path[plen+1:])
			if err != nil {
				return nil, nil, err
			}
			cnode.PutChild(nodeImpl.Path[plen], gckey2)
			cnode.SetValue(value)
		} else if bytes.Equal(matchPrefix, nodeImpl.Path) {
			// existing leaf path is a prefix of the path (node.Path = "hello", path = "hello world")
			_, gckey1, err := mpt.insertLeaf(nil, value, concat(prefix, path[:plen+1]...), path[plen+1:])
			if err != nil {
				return nil, nil, err
			}
			cnode.PutChild(path[plen], gckey1)
			cnode.SetValue(nodeImpl.GetValue())
		} else {
			// existing leaf path and the given path have a prefix (one or more) and separate suffixes (node.Path = "hello world", path = "hello earth")
			// a full node that would contain children with indexes "w" and "e" , an extension node with path "hello "
			//
			// Use concat (which always creates a new slice) instead of append to avoid modifying
			// prefix path since it might be changed by the second insertLeaf below for gckey2).
			_, gckey1, err := mpt.insertLeaf(nil, value, concat(prefix, path[:plen+1]...), path[plen+1:])
			if err != nil {
				return nil, nil, err
			}
			_, gckey2, err := mpt.insertLeaf(nil, nodeImpl.GetValue(),
				concat(prefix, nodeImpl.Path[:plen+1]...), nodeImpl.Path[plen+1:])
			if err != nil {
				return nil, nil, err
			}
			cnode.PutChild(path[plen], gckey1)
			cnode.PutChild(nodeImpl.Path[plen], gckey2)
		}
		if plen == 0 { // node.Path = "world" and path = "earth" => old leaf node is replaced with new branch node
			return mpt.insertNode(node, cnode)
		}
		// if there is a matching prefix, it becomes an extension node
		// node.Path == "hello world", path = "hello earth", enode.Path = "hello " and replaces the old node.
		// enode.NodeKey points to ckey which is a new branch node that contains "world" and "earth" paths
		_, ckey, err := mpt.insertNode(nil, cnode)
		if err != nil {
			return nil, nil, err
		}

		return mpt.insertExtension(node, matchPrefix, ckey)
	case *ExtensionNode:
		// updating an existing extension node with value
		if bytes.Equal(path, nodeImpl.Path) {
			_, ckey, err := mpt.insert(value, nodeImpl.NodeKey,
				concat(prefix, path[:]...), Path{})
			if err != nil {
				return nil, nil, err
			}

			return mpt.insertExtension(node, path, ckey)
		}
		matchPrefix := mpt.matchingPrefix(path, nodeImpl.Path)
		plen := len(matchPrefix)
		// existing branch path is a prefix of the path (node.Path = "hello", path = "hello world")
		if bytes.Equal(matchPrefix, nodeImpl.Path) {
			_, gckey1, err := mpt.insert(value, nodeImpl.NodeKey,
				concat(prefix, matchPrefix...), path[plen:])
			if err != nil {
				return nil, nil, err
			}

			nnode := nodeImpl.Clone().(*ExtensionNode)
			nnode.NodeKey = gckey1
			return mpt.insertNode(node, nnode)
		}

		cnode := NewFullNode(nil)
		if bytes.Equal(matchPrefix, path) {
			// path is a prefix of the existing extension (node.Path = "hello world", path = "hello")
			cnode.SetValue(value)
		} else {
			// existing branch path and the given path have a prefix (one or more) and separate suffixes (node.Path = "hello world", path = "hello earth")
			// a full node that would contain children with indexes "w" and "e" , an extension node with path "hello "
			_, gckey1, err := mpt.insertLeaf(nil, value,
				concat(prefix, path[:plen+1]...), path[plen+1:])
			if err != nil {
				return nil, nil, err
			}

			cnode.PutChild(path[plen], gckey1)
		}
		var gckey2 Key
		if len(nodeImpl.Path) == plen+1 {
			gckey2 = nodeImpl.NodeKey
		} else {
			_, gckey2, err = mpt.insertExtension(nil, nodeImpl.Path[plen+1:], nodeImpl.NodeKey)
			if err != nil {
				return nil, nil, err
			}
		}
		cnode.PutChild(nodeImpl.Path[plen], gckey2)
		if plen == 0 { // node.Path = "world" and path = "earth" => old leaf node is replaced with new branch node
			_, ckey, err := mpt.insertNode(node, cnode)
			if err != nil {
				return nil, nil, err
			}
			return cnode, ckey, nil
		}
		_, ckey, err := mpt.insertNode(nil, cnode)
		if err != nil {
			return nil, nil, err
		}
		// if there is a matching prefix, it becomes an extension node
		// node.Path == "hello world", path = "hello earth", enode.Path = "hello " and replaces the old node.
		// enode.NodeKey points to ckey which is a new branch node that contains "world" and "earth" paths
		return mpt.insertExtension(node, matchPrefix, ckey)
	default:
		panic(fmt.Sprintf("uknown node type: %T %v", node, node))
	}
}

func concat(s1 []byte, s2 ...byte) []byte {
	r := make([]byte, len(s1)+len(s2))
	copy(r, s1)
	copy(r[len(s1):], s2)
	return r
}

func (mpt *MerklePatriciaTrie) deleteAtNode(node Node, prefix, path Path) (Node, Key, error) {
	switch nodeImpl := node.(type) {
	case *FullNode:
		_, ckey, err := mpt.delete(nodeImpl.GetChild(path[0]),
			concat(prefix, path[:1]...), path[1:])
		if err != nil {
			return nil, nil, err
		}
		if ckey == nil {
			numChildren := nodeImpl.GetNumChildren()
			if numChildren == 1 {
				if nodeImpl.HasValue() { // a full node with no children anymore but with a value becomes a leaf node
					return mpt.insertLeaf(node, nodeImpl.GetValue(), concat(prefix), nil)
				}
				// a full node with no children anymore and no value should be removed
				return nil, nil, nil
			}
			if numChildren == 2 {
				if !nodeImpl.HasValue() {
					// a full node with a single child and no value should lift up the child
					tempNode := nodeImpl.Clone().(*FullNode)
					// clear the child being deleted
					tempNode.PutChild(path[0], nil)
					var otherChildKey []byte
					var oidx byte
					for idx, pe := range PathElements {
						child := tempNode.GetChild(pe)
						if child != nil {
							oidx = byte(idx)
							otherChildKey = child
							break
						}
					}
					ochild, err := mpt.db.GetNode(otherChildKey)
					if err != nil {
						return nil, nil, err
					}
					npath := []byte{nodeImpl.indexToByte(oidx)}
					var nnode Node
					switch onodeImpl := ochild.(type) {
					case *FullNode:
						nnode = NewExtensionNode(npath, otherChildKey)
					case *LeafNode:
						if onodeImpl.Path != nil {
							npath = append(npath, onodeImpl.Path...)
						}
						lnode := ochild.Clone().(*LeafNode)
						lnode.SetOrigin(mpt.Version)
						lnode.Path = npath
						lnode.Prefix = concat(prefix)
						nnode = lnode
						if err := mpt.deleteNode(ochild); err != nil {
							return nil, nil, err
						}
					case *ExtensionNode:
						if onodeImpl.Path != nil {
							npath = append(npath, onodeImpl.Path...)
						}
						enode := ochild.Clone().(*ExtensionNode)
						enode.Path = npath
						nnode = enode
						if err := mpt.deleteNode(ochild); err != nil {
							return nil, nil, err
						}
					default:
						panic(fmt.Sprintf("unknown node type: %T %v %T", ochild, ochild, mpt.db))
					}
					return mpt.insertNode(node, nnode)
				}
			}
		}
		nnode := nodeImpl.Clone().(*FullNode)
		nnode.PutChild(path[0], ckey)
		return mpt.insertNode(node, nnode)
	case *LeafNode:
		if bytes.Equal(path, nodeImpl.Path) {
			return mpt.deleteAfterPathTraversal(node)
		}

		return nil, nil, ErrValueNotPresent // There is nothing to delete
	case *ExtensionNode:
		matchPrefix := mpt.matchingPrefix(path, nodeImpl.Path)
		if !bytes.Equal(matchPrefix, nodeImpl.Path) {
			return nil, nil, ErrValueNotPresent // There is nothing to delete
		}

		plen := len(matchPrefix)
		cnode, ckey, err := mpt.delete(nodeImpl.NodeKey, concat(prefix, path[:plen]...), path[plen:])
		if err != nil {
			return nil, nil, err
		}
		switch cnodeImpl := cnode.(type) {
		case *LeafNode:
			// if extension child changes from full node to leaf, convert the extension into a leaf node
			nnode := cnode.Clone().(*LeafNode)
			nnode.SetOrigin(mpt.Version)
			nnode.Prefix = concat(prefix)
			nnode.Path = append(nodeImpl.Path, cnodeImpl.Path...)
			nnode.SetValue(cnodeImpl.GetValue())
			if err := mpt.deleteNode(cnode); err != nil {
				return nil, nil, err
			}
			return mpt.insertNode(node, nnode)
		case *FullNode:
			nnode := nodeImpl.Clone().(*ExtensionNode)
			nnode.NodeKey = ckey
			return mpt.insertNode(node, nnode)
		case *ExtensionNode:
			// if extension child changes from full node to extension node, merge the extensions
			nnode := nodeImpl.Clone().(*ExtensionNode)
			nnode.Path = append(nnode.Path, cnodeImpl.Path...)
			nnode.NodeKey = cnodeImpl.NodeKey
			if err := mpt.deleteNode(cnode); err != nil {
				return nil, nil, err
			}
			return mpt.insertNode(node, nnode)
		default:
			panic(fmt.Sprintf("unknown node type: %T %v", cnode, cnode))
		}
	default:
		panic(fmt.Sprintf("unknown node type: %T %v", node, node))
	}
}

func (mpt *MerklePatriciaTrie) insertAfterPathTraversal(value MPTSerializable, node Node) (Node, Key, error) {
	switch nodeImpl := node.(type) {
	case *FullNode:
		// The value of the branch needs to be updated
		nnode := nodeImpl.Clone().(*FullNode)
		nnode.SetValue(value)
		return mpt.insertNode(node, nnode)
	case *LeafNode:
		if len(nodeImpl.Path) == 0 { // the value of an existing node needs updated
			return mpt.insertLeaf(node, value, nodeImpl.Prefix, nodeImpl.Path)
		}
		// an existing leaf node needs to become a branch + leafnode (with one less path element as it's stored on the new branch) with value on the new branch
		_, ckey, err := mpt.insertLeaf(nil, nodeImpl.GetValue(),
			concat(nodeImpl.Prefix, nodeImpl.Path[:1]...), nodeImpl.Path[1:])
		if err != nil {
			return nil, nil, err
		}

		nnode := NewFullNode(value)
		nnode.PutChild(nodeImpl.Path[0], ckey)
		return mpt.insertNode(node, nnode)
	case *ExtensionNode:
		// an existing extension node becomes a branch + extension node (with one less path element as it's stored in the new branch) with value on the new branch
		_, ckey, err := mpt.insertExtension(nil, nodeImpl.Path[1:], nodeImpl.NodeKey)
		if err != nil {
			return nil, nil, err
		}
		nnode := NewFullNode(value)
		nnode.PutChild(nodeImpl.Path[0], ckey)
		return mpt.insertNode(node, nnode)
	default:
		panic(fmt.Sprintf("unknown node type: %T %v", node, node))
	}
}

func (mpt *MerklePatriciaTrie) deleteAfterPathTraversal(node Node) (Node, Key, error) {
	switch nodeImpl := node.(type) {
	case *FullNode:
		// The value of the branch needs to be updated
		nnode := nodeImpl.Clone().(*FullNode)
		nnode.SetValue(nil)
		if nodeImpl.HasValue() {
			mpt.ChangeCollector.DeleteChange(nodeImpl.Value)
		}
		return mpt.insertNode(node, nnode)
	case *LeafNode:
		if nodeImpl.HasValue() {
			mpt.ChangeCollector.DeleteChange(nodeImpl.Value)
		}
		if err := mpt.deleteNode(node); err != nil {
			return nil, nil, err
		}
		return nil, nil, nil
	case *ExtensionNode:
		panic("this should not happen!")
	default:
		panic(fmt.Sprintf("unknown node type: %T %v", node, node))
	}
}

func (mpt *MerklePatriciaTrie) iterate(ctx context.Context, path Path, key Key, handler MPTIteratorHandler, visitNodeTypes byte) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	node, err := mpt.db.GetNode(key)
	if err != nil {
		if herr := handler(ctx, path, key, node); herr != nil {
			return herr
		}
		return err
	}
	switch nodeImpl := node.(type) {
	case *LeafNode:
		if IncludesNodeType(visitNodeTypes, NodeTypeLeafNode) {
			if err := handler(ctx, path, key, node); err != nil {
				return err
			}
		}
		npath := append(path, nodeImpl.Path...)
		if IncludesNodeType(visitNodeTypes, NodeTypeValueNode) && nodeImpl.HasValue() {
			if err := handler(ctx, npath, nil, nodeImpl.Value); err != nil {
				return err
			}
		}
	case *FullNode:
		if IncludesNodeType(visitNodeTypes, NodeTypeFullNode) {
			if err := handler(ctx, path, key, node); err != nil {
				return err
			}
		}
		if IncludesNodeType(visitNodeTypes, NodeTypeValueNode) && nodeImpl.HasValue() {
			if err := handler(ctx, path, nil, nodeImpl.Value); err != nil {
				return err
			}
		}
		var ecount = 0
		for i := byte(0); i < 16; i++ {
			pe := nodeImpl.indexToByte(i)
			child := nodeImpl.GetChild(pe)
			if child == nil {
				continue
			}
			npath := append(path, pe)
			if err := mpt.iterate(ctx, npath, child, handler, visitNodeTypes); err != nil {
				if err == ErrNodeNotFound || err == ErrIteratingChildNodes {
					ecount++
				} else {
					Logger.Error("iterate - child node", zap.Error(err))
					return err
				}
			}
		}
		if ecount != 0 {
			return ErrIteratingChildNodes
		}
	case *ExtensionNode:
		if IncludesNodeType(visitNodeTypes, NodeTypeExtensionNode) {
			if err := handler(ctx, path, key, node); err != nil {
				return err
			}
		}
		npath := append(path, nodeImpl.Path...)
		return mpt.iterate(ctx, npath, nodeImpl.NodeKey, handler, visitNodeTypes)
	}
	return nil
}

func (mpt *MerklePatriciaTrie) insertNode(oldNode Node, newNode Node) (Node, Key, error) {
	if DebugMPTNode {
		ohash := ""
		if oldNode != nil {
			ohash = oldNode.GetHash()
		}
		Logger.Info("insert node", zap.String("nn", newNode.GetHash()), zap.String("on", ohash))
	}

	newNode.SetOrigin(mpt.Version)
	ckey := newNode.GetHashBytes()
	if err := mpt.db.PutNode(ckey, newNode); err != nil {
		return nil, nil, err
	}
	//If same node is inserted by client, don't add them into change collector
	if oldNode == nil {
		mpt.ChangeCollector.AddChange(oldNode, newNode)
	} else {
		okey := oldNode.GetHashBytes()
		if !bytes.Equal(okey, ckey) { //delete previous node only if it isn`t the same as new one
			mpt.ChangeCollector.AddChange(oldNode, newNode)
			//NOTE: since leveldb is initiaized with propagate deletes as false, only newly created nodes will get deleted
			if err := mpt.db.DeleteNode(okey); err != nil {
				return nil, nil, err
			}
		}
	}
	return newNode, ckey, nil
}

func (mpt *MerklePatriciaTrie) deleteNode(node Node) error {
	if DebugMPTNode {
		Logger.Info("delete node", zap.String("dn", node.GetHash()))
	}
	//Logger.Debug("delete node", zap.Any("version", mpt.Version), zap.String("key", node.GetHash()))
	mpt.ChangeCollector.DeleteChange(node)
	return mpt.db.DeleteNode(node.GetHashBytes())
}

func (mpt *MerklePatriciaTrie) matchingPrefix(p1 Path, p2 Path) Path {
	idx := 0
	for ; idx < len(p1) && idx < len(p2) && p1[idx] == p2[idx]; idx++ {
	}
	return p1[:idx]
}

func (mpt *MerklePatriciaTrie) indent(w io.Writer, depth byte) {
	for i := byte(0); i < depth; i++ {
		_, _ = w.Write([]byte(" "))
	}
}

func (mpt *MerklePatriciaTrie) pp(w io.Writer, key Key, depth byte, initpad bool) error {
	if initpad {
		mpt.indent(w, depth)
	}
	node, err := mpt.db.GetNode(key)
	if err != nil {
		_, _ = fmt.Fprintf(w, "err %v %v\n", ToHex(key), err)
		return err
	}
	switch nodeImpl := node.(type) {
	case *LeafNode:
		_, _ = fmt.Fprintf(w, "L:%v (prefix:%v path:%v, origin:%v, version:%v)\n", ToHex(key), string(nodeImpl.Prefix), string(nodeImpl.Path), node.GetOrigin(), node.GetVersion())
	case *ExtensionNode:
		_, _ = fmt.Fprintf(w, "E:%v (path:%v,child:%v, origin:%v, version:%v)\n", ToHex(key), string(nodeImpl.Path), ToHex(nodeImpl.NodeKey), node.GetOrigin(), node.GetVersion())
		_ = mpt.pp(w, nodeImpl.NodeKey, depth+2, true)
	case *FullNode:
		_, _ = w.Write([]byte("F:"))
		_, _ = fmt.Fprintf(w, "%v (,origin:%v, version:%v)", ToHex(key), node.GetOrigin(), node.GetVersion())
		_, _ = w.Write([]byte("\n"))
		for idx, cnode := range nodeImpl.Children {
			if cnode == nil {
				continue
			}
			mpt.indent(w, depth+1)
			_, _ = w.Write([]byte(fmt.Sprintf("%.2d ", idx)))
			_ = mpt.pp(w, cnode, depth+2, false)
		}
	}
	return nil
}

/*UpdateVersion - updates the origin of all the nodes in this tree to the given origin */
func (mpt *MerklePatriciaTrie) UpdateVersion(ctx context.Context, version Sequence, missingNodeHander MPTMissingNodeHandler) error {
	mpt.mutex.RLock()
	defer mpt.mutex.RUnlock()

	ps := GetPruneStats(ctx)
	if ps != nil {
		ps.Version = version
	}
	keys := make([]Key, 0, BatchSize)
	values := make([]Node, 0, BatchSize)
	var count int64
	var missingNodes int64
	handler := func(ctx context.Context, path Path, key Key, node Node) error {
		if node == nil {
			_ = missingNodeHander(ctx, path, key)
			missingNodes++
			return nil
		}
		if node.GetVersion() >= version {
			return nil
		}
		count++
		node.SetVersion(version)
		tkey := make([]byte, len(key))
		copy(tkey, key)
		keys = append(keys, tkey)
		values = append(values, node)
		if len(keys) == BatchSize {
			err := mpt.db.MultiPutNode(keys, values)
			keys = keys[:0]
			values = values[:0]
			if err != nil {
				Logger.Error("update version - multi put", zap.String("path", string(path)), zap.String("key", ToHex(key)), zap.Any("old_version", node.GetVersion()), zap.Any("new_version", version), zap.Error(err))
			}
			return err
		}
		return nil
	}
	err := mpt.Iterate(ctx, handler, NodeTypeLeafNode|NodeTypeFullNode|NodeTypeExtensionNode)
	if ps != nil {
		ps.BelowVersion = count
		ps.MissingNodes = missingNodes
	}
	if err == nil || err == ErrNodeNotFound || err == ErrIteratingChildNodes {
		if len(keys) > 0 {
			if err := mpt.db.MultiPutNode(keys, values); err != nil {
				Logger.Error("update version - multi put - last batch", zap.Error(err))
				return err
			}
		}
	}
	return err
}

// FindMissingNodes returns the paths and keys of missing nodes
func (mpt *MerklePatriciaTrie) FindMissingNodes(ctx context.Context) ([]Path, []Key, error) {
	paths := make([]Path, 0, BatchSize)
	keys := make([]Key, 0, BatchSize)
	handler := func(ctx context.Context, path Path, key Key, node Node) error {
		if node == nil {
			paths = append(paths, path)
			keys = append(keys, key)
		}
		return nil
	}

	st := time.Now()
	// TODO: may have dead lock for the iterate
	err := mpt.Iterate(ctx, handler, NodeTypeLeafNode|NodeTypeFullNode|NodeTypeExtensionNode)
	if err != nil {
		switch err {
		case ErrNodeNotFound, ErrIteratingChildNodes:
			Logger.Debug("Find missing nodes err", zap.Error(err))
		default:
			Logger.Error("Find missing node with unexpected err", zap.Error(err))
			return nil, nil, err
		}
	}

	Logger.Debug("Find missing nodes iteration time", zap.Any("duration", time.Since(st)))

	return paths, keys, nil
}

// HasMissingNodes returns immediately when a missing node is detected
func (mpt *MerklePatriciaTrie) HasMissingNodes(ctx context.Context) (bool, error) {
	paths := make([]Path, 0, BatchSize)
	keys := make([]Key, 0, BatchSize)
	handler := func(ctx context.Context, path Path, key Key, node Node) error {
		if node == nil {
			paths = append(paths, path)
			keys = append(keys, key)
			return ErrMissingNodes
		}
		return nil
	}

	st := time.Now()
	err := mpt.Iterate(ctx, handler, NodeTypeLeafNode|NodeTypeFullNode|NodeTypeExtensionNode)
	switch err {
	case nil:
		Logger.Debug("Find missing nodes iteration time", zap.Any("duration", time.Since(st)))
		// full state
		return false, nil
	case ErrMissingNodes, ErrNodeNotFound, ErrIteratingChildNodes:
		// find missing nodes
		Logger.Debug("Find missing nodes iteration time", zap.Any("duration", time.Since(st)))
		return true, nil
	default:
		Logger.Error("Find missing node with unexpected err", zap.Error(err))
		return false, err
	}
}

/*IsMPTValid - checks if the merkle tree is in valid state or not */
func IsMPTValid(mpt MerklePatriciaTrieI) error {
	return mpt.Iterate(context.TODO(), func(ctxt context.Context, path Path, key Key, node Node) error { return nil }, NodeTypeLeafNode|NodeTypeFullNode|NodeTypeExtensionNode)
}

//Validate - implement interface - any sort of validations that can tell if the MPT is in a sane state
func (mpt *MerklePatriciaTrie) Validate() error {
	mpt.mutex.RLock()
	defer mpt.mutex.RUnlock()

	if err := mpt.ChangeCollector.Validate(); err != nil {
		return err
	}
	changes := mpt.ChangeCollector.GetChanges()
	db := mpt.getNodeDB()
	switch dbImpl := db.(type) {
	case *MemoryNodeDB:
	case *LevelNodeDB:
		db = dbImpl.GetCurrent()
	case *PNodeDB:
		return nil
	}
	for _, c := range changes {
		if c.Old == nil {
			continue
		}
		if _, err := db.GetNode(c.Old.GetHashBytes()); err == nil {
			return fmt.Errorf(FmtIntermediateNodeExists, c.Old, c.Old.GetHash(), c.New, c.New.GetHash())
		}
	}
	return nil
}

// MergeMPTChanges - implement interface.
func (mpt *MerklePatriciaTrie) MergeMPTChanges(mpt2 MerklePatriciaTrieI) error {
	if bytes.Equal(mpt.GetRoot(), mpt2.GetRoot()) {
		//Logger.Debug("MergeMPTChanges - MPT merge changes with the same root")
		return nil
	}

	if DebugMPTNode {
		if err := mpt2.Validate(); err != nil {
			Logger.Error("MergeMPTChanges - MPT validate", zap.Error(err))
		}
	}

	newDB := mpt2.GetNodeDB()
	newLNDB, ok := newDB.(*LevelNodeDB)
	if !ok {
		Logger.Error("MergeMPTChanges, new MPT's DB is not a LevelNodeDB")
		return errors.New("invalid mpt db")
	}

	preDB := newLNDB.GetPrev()
	if preDB != mpt.GetNodeDB() {
		Logger.Error("MergeMPTChanges does not merge direct child mpt")
		return errors.New("mpt does not merge changes from its child")
	}

	mpt.mutex.Lock()
	defer mpt.mutex.Unlock()
	db, ok := mpt.db.(*LevelNodeDB)
	if ok {
		db.version = newLNDB.version
	} else {
		Logger.Warn("MergeMPTChanges - mpt db is not *LevelNodeDB",
			zap.Int64("version", int64(mpt.GetVersion())))
	}

	newRoot, changes, deletes, startRoot := mpt2.GetChanges()
	if err := mpt.mergeChanges(newRoot, changes, deletes, startRoot); err != nil {
		return err
	}

	return nil
}

// MergeChanges - implement interface.
func (mpt *MerklePatriciaTrie) MergeChanges(newRoot Key, changes []*NodeChange, deletes []Node, startRoot Key) error {
	mpt.mutex.Lock()
	defer mpt.mutex.Unlock()
	return mpt.mergeChanges(newRoot, changes, deletes, startRoot)
}

func (mpt *MerklePatriciaTrie) mergeChanges(newRoot Key, changes []*NodeChange, deletes []Node, startRoot Key) error {
	if bytes.Equal(mpt.root, newRoot) {
		Logger.Error("MergeMPTChanges - MPT merge changes with the same root")
		return nil
	}

	if !bytes.Equal(mpt.root, startRoot) {
		Logger.Error("MergeMPTChanges - optimistic lock failure")
		return errors.New("optimistic lock failure")
	}

	for _, c := range changes {
		if _, _, err := mpt.insertNode(c.Old, c.New); err != nil {
			return err
		}
	}

	for _, d := range deletes {
		if err := mpt.deleteNode(d); err != nil {
			logging.Logger.Error("delete node failed", zap.Error(err))
		}
	}

	mpt.setRoot(newRoot)
	return nil
}

// MergeDB - merges the state changes from the node db directly
func (mpt *MerklePatriciaTrie) MergeDB(ndb NodeDB, root Key) error {
	mpt.mutex.Lock()
	defer mpt.mutex.Unlock()
	handler := func(ctx context.Context, key Key, node Node) error {
		_, _, err := mpt.insertNode(nil, node)
		return err
	}
	mpt.root = root
	return ndb.Iterate(context.TODO(), handler)
}

func (mpt *MerklePatriciaTrie) GetChangeCollector() (cc ChangeCollectorI) {
	mpt.mutex.Lock()
	cc = mpt.ChangeCollector.Clone()
	mpt.mutex.Unlock()
	return
}

func (mpt *MerklePatriciaTrie) SetChangeCollector(cc ChangeCollectorI) {
	mpt.mutex.Lock()
	mpt.ChangeCollector = cc
	mpt.mutex.Unlock()
}
