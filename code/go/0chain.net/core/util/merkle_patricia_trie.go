package util

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

//DebugMPTNode - for detailed debugging
var DebugMPTNode = false

/*MerklePatriciaTrie - it's a merkle tree and a patricia trie */
type MerklePatriciaTrie struct {
	mutex           *sync.RWMutex
	Root            Key
	db              NodeDB
	ChangeCollector ChangeCollectorI
	Version         Sequence
	rootHashTrack   map[string]struct{}
}

/*NewMerklePatriciaTrie - create a new patricia merkle trie */
func NewMerklePatriciaTrie(db NodeDB, version Sequence) *MerklePatriciaTrie {
	mpt := &MerklePatriciaTrie{
		mutex:         &sync.RWMutex{},
		db:            db,
		rootHashTrack: map[string]struct{}{},
	}
	mpt.ResetChangeCollector(nil)
	mpt.SetVersion(version)
	return mpt
}

//CloneMPT - clone an existing MPT so it can go off of a different root
func CloneMPT(mpt MerklePatriciaTrieI) *MerklePatriciaTrie {
	clone := NewMerklePatriciaTrie(mpt.GetNodeDB(), mpt.GetVersion())
	clone.SetRoot(mpt.GetRoot())
	return clone
}

/*SetNodeDB - implement interface */
func (mpt *MerklePatriciaTrie) SetNodeDB(ndb NodeDB) {
	mpt.mutex.Lock()
	defer mpt.mutex.Unlock()
	Logger.Debug("MPT SetNodeDB")
	mpt.db = ndb
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

/*SetRoot - implement interface */
func (mpt *MerklePatriciaTrie) SetRoot(root Key) {
	mpt.mutex.Lock()
	defer mpt.mutex.Unlock()
	mpt.setRoot(root)
}

/*setRoot - implement interface */
func (mpt *MerklePatriciaTrie) setRoot(root Key) {
	rs := hex.EncodeToString(root)
	if bytes.Compare(mpt.Root, root) != 0 {
		if _, ok := mpt.rootHashTrack[rs]; ok {
			Logger.DPanic("MPT set root back, which should not happen",
				zap.String("root key", rs))
		}
	}
	mpt.Root = root
	mpt.rootHashTrack[rs] = struct{}{}
}

/*GetRoot - implement interface */
func (mpt *MerklePatriciaTrie) GetRoot() Key {
	mpt.mutex.RLock()
	defer mpt.mutex.RUnlock()
	return mpt.Root
}

/*GetNodeValue - get the value for a given path */
func (mpt *MerklePatriciaTrie) GetNodeValue(path Path) (Serializable, error) {
	mpt.mutex.RLock()
	defer mpt.mutex.RUnlock()

	rootKey := []byte(mpt.Root)
	if rootKey == nil || len(rootKey) == 0 {
		return nil, ErrValueNotPresent
	}

	rootNode, err := mpt.db.GetNode(rootKey)
	if err != nil {
		return nil, err
	}
	if rootNode == nil {
		return nil, ErrNodeNotFound
	}
	v, err := mpt.getNodeValue(path, rootNode)
	if err != nil {
		return nil, err
	}
	if v == nil { // This can happen if path given is partial that aligns with a full node that has no value
		return nil, ErrValueNotPresent
	}
	return v, err
}

/*Insert - inserts (updates) a value into this trie and updates the trie all the way up and produces a new root */
func (mpt *MerklePatriciaTrie) Insert(path Path, value Serializable) (Key, error) {
	if value == nil {
		Logger.Debug("Insert nil value, delete data on path:",
			zap.String("path", string(path)))
		return mpt.Delete(path)
	}
	eval := value.Encode()
	if eval == nil || len(eval) == 0 {
		Logger.Debug("Insert encoded nil value, delete data on path:",
			zap.String("path", string(path)))
		return mpt.Delete(path)
	}

	mpt.mutex.Lock()
	defer mpt.mutex.Unlock()
	var err error
	var newRootHash Key
	if mpt.Root == nil {
		_, newRootHash, err = mpt.insertLeaf(nil, value, path)
	} else {
		_, newRootHash, err = mpt.insert(value, []byte(mpt.Root), path)
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

	_, newRootHash, err := mpt.delete(mpt.Root, path)
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

	nodes, err := mpt.getPathNodes(mpt.Root, path)
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
		if bytes.Compare(nodeImpl.Path, path) == 0 {
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
		if bytes.Compare(nodeImpl.Path, prefix) == 0 {
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

/*GetChangeCollector - implement interface */
func (mpt *MerklePatriciaTrie) GetChangeCollector() ChangeCollectorI {
	mpt.mutex.RLock()
	defer mpt.mutex.RUnlock()
	return mpt.ChangeCollector
}

func (mpt *MerklePatriciaTrie) getChangeCollector() ChangeCollectorI {
	return mpt.ChangeCollector
}

/*ResetChangeCollector - implement interface */
func (mpt *MerklePatriciaTrie) ResetChangeCollector(root Key) {
	mpt.mutex.Lock()
	defer mpt.mutex.Unlock()
	mpt.ChangeCollector = NewChangeCollector()
	if root != nil {
		mpt.setRoot(root)
	}
}

/*SaveChanges - implement interface */
func (mpt *MerklePatriciaTrie) SaveChanges(ndb NodeDB, includeDeletes bool) error {
	mpt.mutex.RLock()
	defer mpt.mutex.RUnlock()
	cc := mpt.ChangeCollector
	err := cc.UpdateChanges(ndb, mpt.Version, includeDeletes)
	if err != nil {
		return err
	}
	return nil
}

/*Iterate - iterate the entire trie */
func (mpt *MerklePatriciaTrie) Iterate(ctx context.Context, handler MPTIteratorHandler, visitNodeTypes byte) error {
	mpt.mutex.RLock()
	defer mpt.mutex.RUnlock()

	rootKey := mpt.Root
	if rootKey == nil || len(rootKey) == 0 {
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
	return mpt.pp(w, Key(mpt.Root), 0, false)
}

func (mpt *MerklePatriciaTrie) getNodeValue(path Path, node Node) (Serializable, error) {
	switch nodeImpl := node.(type) {
	case *LeafNode:
		if bytes.Compare(nodeImpl.Path, path) == 0 {
			return nodeImpl.GetValue(), nil
		}
		return nil, ErrValueNotPresent
	case *FullNode:
		if len(path) == 0 {
			return nodeImpl.GetValue(), nil
		}
		ckey := nodeImpl.GetChild(path[0])
		if ckey == nil {
			return nil, ErrValueNotPresent
		}

		nnode, err := mpt.db.GetNode(ckey)
		if err != nil || nnode == nil {
			if err != nil {
				Logger.Error("full node get node failed",
					zap.Any("version", mpt.Version),
					zap.Int("path len", len(path)),
					zap.String("path", string(path)),
					zap.String("key", hex.EncodeToString(ckey)),
					zap.String("root key", hex.EncodeToString(mpt.GetRoot())),
					zap.String("node hash", node.GetHash()),
					zap.Error(err))
			}
			return nil, ErrNodeNotFound
		}
		return mpt.getNodeValue(path[1:], nnode)
	case *ExtensionNode:
		prefix := mpt.matchingPrefix(path, nodeImpl.Path)
		if len(prefix) == 0 {
			return nil, ErrValueNotPresent
		}
		if bytes.Compare(nodeImpl.Path, prefix) == 0 {
			nnode, err := mpt.db.GetNode(nodeImpl.NodeKey)
			if err != nil || nnode == nil {
				if err != nil {
					Logger.Error("extension node get node failed", zap.Error(err))
				}
				return nil, ErrNodeNotFound
			}
			return mpt.getNodeValue(path[len(prefix):], nnode)
		}
		return nil, ErrValueNotPresent
	default:
		panic(fmt.Sprintf("unknown node type: %T %v", node, node))
	}
}

func (mpt *MerklePatriciaTrie) insert(value Serializable, key Key, path Path) (Node, Key, error) {
	node, err := mpt.db.GetNode(key)
	if err != nil {
		return nil, nil, err
	}
	if len(path) == 0 {
		return mpt.insertAfterPathTraversal(value, node)
	}
	return mpt.insertAtNode(value, node, path)

}

func (mpt *MerklePatriciaTrie) insertLeaf(oldNode Node, value Serializable, path Path) (Node, Key, error) {
	n, k, err := mpt.insertNode(oldNode, NewLeafNode(path, mpt.Version, value))
	if oldNode != nil {
		Logger.Debug("insertLeaf",
			zap.String("path", string(path)),
			zap.String("key", hex.EncodeToString(k)),
			zap.String("old node key", oldNode.GetHash()))
	} else {
		Logger.Debug("insertLeaf",
			zap.String("path", string(path)),
			zap.String("key", hex.EncodeToString(k)))
	}

	return n, k, err
}

func (mpt *MerklePatriciaTrie) insertExtension(oldNode Node, path Path, key Key) (Node, Key, error) {
	return mpt.insertNode(oldNode, NewExtensionNode(path, key))
}

func (mpt *MerklePatriciaTrie) delete(key Key, path Path) (Node, Key, error) {
	node, err := mpt.db.GetNode(key)
	if err != nil {
		return nil, nil, err
	}
	if len(path) == 0 {
		return mpt.deleteAfterPathTraversal(node)
	}
	return mpt.deleteAtNode(node, path)
}

func (mpt *MerklePatriciaTrie) insertAtNode(value Serializable, node Node, path Path) (Node, Key, error) {
	var err error
	switch nodeImpl := node.(type) {
	case *FullNode:
		ckey := nodeImpl.GetChild(path[0])
		if ckey == nil {
			_, ckey, err = mpt.insertLeaf(nil, value, path[1:])
		} else {
			_, ckey, err = mpt.insert(value, ckey, path[1:])
		}
		if err != nil {
			return nil, nil, err
		}
		nnode := nodeImpl.Clone().(*FullNode)
		nnode.PutChild(path[0], ckey)
		return mpt.insertNode(node, nnode)
	case *LeafNode:
		if len(nodeImpl.Path) == 0 {
			if _, ckey, err := mpt.insertLeaf(nil, value, path[1:]); err != nil {
				return nil, nil, err
			} else {
				nnode := NewFullNode(nodeImpl.GetValue())
				nnode.PutChild(path[0], ckey)
				return mpt.insertNode(node, nnode)
			}
		}
		// updating an existing leaf
		if bytes.Compare(path, nodeImpl.Path) == 0 {
			return mpt.insertLeaf(node, value, nodeImpl.Path)
		}

		prefix := mpt.matchingPrefix(path, nodeImpl.Path)
		plen := len(prefix)
		cnode := NewFullNode(nil)
		if bytes.Compare(prefix, path) == 0 { // path is a prefix of the existing leaf (node.Path = "hello world", path = "hello")
			if _, gckey2, err := mpt.insertLeaf(nil, nodeImpl.GetValue(), nodeImpl.Path[plen+1:]); err == nil {
				cnode.PutChild(nodeImpl.Path[plen], gckey2)
				cnode.SetValue(value)
			} else {
				return nil, nil, err
			}
		} else if bytes.Compare(prefix, nodeImpl.Path) == 0 { // existing leaf path is a prefix of the path (node.Path = "hello", path = "hello world")
			if _, gckey1, err := mpt.insertLeaf(nil, value, path[plen+1:]); err == nil {
				cnode.PutChild(path[plen], gckey1)
				cnode.SetValue(nodeImpl.GetValue())
			} else {
				return nil, nil, err
			}
		} else { // existing leaf path and the given path have a prefix (one or more) and separate suffixes (node.Path = "hello world", path = "hello earth")
			// a full node that would contain children with indexes "w" and "e" , an extension node with path "hello "
			_, gckey1, err := mpt.insertLeaf(nil, value, path[plen+1:])
			if err != nil {
				return nil, nil, err
			}
			_, gckey2, err := mpt.insertLeaf(nil, nodeImpl.GetValue(), nodeImpl.Path[plen+1:])
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
		if _, ckey, err := mpt.insertNode(nil, cnode); err != nil {
			return nil, nil, err
		} else {
			return mpt.insertExtension(node, prefix, ckey)
		}
	case *ExtensionNode:
		// updating an existing extension node with value
		if bytes.Compare(path, nodeImpl.Path) == 0 {
			if _, ckey, err := mpt.insert(value, nodeImpl.NodeKey, Path{}); err != nil {
				return nil, nil, err
			} else {
				return mpt.insertExtension(node, path, ckey)
			}
		}
		prefix := mpt.matchingPrefix(path, nodeImpl.Path)
		plen := len(prefix)
		if bytes.Compare(prefix, nodeImpl.Path) == 0 { // existing branch path is a prefix of the path (node.Path = "hello", path = "hello world")
			if _, gckey1, err := mpt.insert(value, nodeImpl.NodeKey, path[plen:]); err != nil {
				return nil, nil, err
			} else {
				nnode := nodeImpl.Clone().(*ExtensionNode)
				nnode.NodeKey = gckey1
				return mpt.insertNode(node, nnode)
			}
		}
		cnode := NewFullNode(nil)
		if bytes.Compare(prefix, path) == 0 {
			// path is a prefix of the existing extension (node.Path = "hello world", path = "hello")
			cnode.SetValue(value)
		} else {
			// existing branch path and the given path have a prefix (one or more) and separate suffixes (node.Path = "hello world", path = "hello earth")
			// a full node that would contain children with indexes "w" and "e" , an extension node with path "hello "
			if _, gckey1, err := mpt.insertLeaf(nil, value, path[plen+1:]); err != nil {
				return nil, nil, err
			} else {
				cnode.PutChild(path[plen], gckey1)
			}
		}
		var gckey2 Key
		if len(nodeImpl.Path) == plen+1 {
			gckey2 = nodeImpl.NodeKey
		} else {
			if _, gckey2, err = mpt.insertExtension(nil, nodeImpl.Path[plen+1:], nodeImpl.NodeKey); err != nil {
				return nil, nil, err
			}
		}
		cnode.PutChild(nodeImpl.Path[plen], gckey2)
		if plen == 0 { // node.Path = "world" and path = "earth" => old leaf node is replaced with new branch node
			if _, ckey, err := mpt.insertNode(node, cnode); err != nil {
				return nil, nil, err
			} else {
				return cnode, ckey, nil
			}
		}
		if _, ckey, err := mpt.insertNode(nil, cnode); err != nil {
			return nil, nil, err
		} else {
			// if there is a matching prefix, it becomes an extension node
			// node.Path == "hello world", path = "hello earth", enode.Path = "hello " and replaces the old node.
			// enode.NodeKey points to ckey which is a new branch node that contains "world" and "earth" paths
			return mpt.insertExtension(node, prefix, ckey)
		}
	default:
		panic(fmt.Sprintf("uknown node type: %T %v", node, node))
	}
}

func (mpt *MerklePatriciaTrie) deleteAtNode(node Node, path Path) (Node, Key, error) {
	switch nodeImpl := node.(type) {
	case *FullNode:
		_, ckey, err := mpt.delete(nodeImpl.GetChild(path[0]), path[1:])
		if err != nil {
			return nil, nil, err
		}
		if ckey == nil {
			numChildren := nodeImpl.GetNumChildren()
			if numChildren == 1 {
				if nodeImpl.HasValue() { // a full node with no children anymore but with a value becomes a leaf node
					return mpt.insertLeaf(node, nodeImpl.GetValue(), nil)
				}
				// a full node with no children anymore and no value should be removed
				return nil, nil, nil
			}
			if numChildren == 2 {
				if !nodeImpl.HasValue() { // a full node with a single child and no value should lift up the child
					tempNode := nodeImpl.Clone().(*FullNode)
					tempNode.PutChild(path[0], nil) // clear the child being deleted
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
					npath := []byte{nodeImpl.indexToByte(byte(oidx))}
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
						nnode = lnode
						mpt.deleteNode(ochild)
					case *ExtensionNode:
						if onodeImpl.Path != nil {
							npath = append(npath, onodeImpl.Path...)
						}
						enode := ochild.Clone().(*ExtensionNode)
						enode.Path = npath
						nnode = enode
						mpt.deleteNode(ochild)
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
		if bytes.Compare(path, nodeImpl.Path) != 0 {
			return node, node.GetHashBytes(), ErrValueNotPresent // There is nothing to delete
		}
		return mpt.deleteAfterPathTraversal(node)
	case *ExtensionNode:
		prefix := mpt.matchingPrefix(path, nodeImpl.Path)
		if bytes.Compare(prefix, nodeImpl.Path) != 0 {
			return node, node.GetHashBytes(), ErrValueNotPresent // There is nothing to delete
		}
		cnode, ckey, err := mpt.delete(nodeImpl.NodeKey, path[len(nodeImpl.Path):])
		if err != nil {
			return nil, nil, err
		}
		switch cnodeImpl := cnode.(type) {
		case *LeafNode: // if extension child changes from full node to leaf, convert the extension into a leaf node
			nnode := cnode.Clone().(*LeafNode)
			nnode.SetOrigin(mpt.Version)
			nnode.Path = append(nodeImpl.Path, cnodeImpl.Path...)
			nnode.SetValue(cnodeImpl.GetValue())
			mpt.deleteNode(cnode)
			return mpt.insertNode(node, nnode)
		case *FullNode:
			nnode := nodeImpl.Clone().(*ExtensionNode)
			nnode.NodeKey = ckey
			return mpt.insertNode(node, nnode)
		case *ExtensionNode: // if extension child changes from full node to extension node, merge the extensions
			nnode := nodeImpl.Clone().(*ExtensionNode)
			nnode.Path = append(nnode.Path, cnodeImpl.Path...)
			nnode.NodeKey = cnodeImpl.NodeKey
			mpt.deleteNode(cnode)
			return mpt.insertNode(node, nnode)
		default:
			panic(fmt.Sprintf("unknown node type: %T %v", cnode, cnode))
		}
	default:
		panic(fmt.Sprintf("unknown node type: %T %v", node, node))
	}
}

func (mpt *MerklePatriciaTrie) insertAfterPathTraversal(value Serializable, node Node) (Node, Key, error) {
	switch nodeImpl := node.(type) {
	case *FullNode:
		// The value of the branch needs to be updated
		nnode := nodeImpl.Clone().(*FullNode)
		nnode.SetValue(value)
		return mpt.insertNode(node, nnode)
	case *LeafNode:
		if len(nodeImpl.Path) == 0 { // the value of an existing node needs updated
			return mpt.insertLeaf(node, value, nodeImpl.Path)
		}
		// an existing leaf node needs to become a branch + leafnode (with one less path element as it's stored on the new branch) with value on the new branch
		if _, ckey, err := mpt.insertLeaf(nil, nodeImpl.GetValue(), nodeImpl.Path[1:]); err != nil {
			return nil, nil, err
		} else {
			nnode := NewFullNode(value)
			nnode.PutChild(nodeImpl.Path[0], ckey)
			return mpt.insertNode(node, nnode)
		}
	case *ExtensionNode:
		// an existing extension node becomes a branch + extension node (with one less path element as it's stored in the new branch) with value on the new branch
		if _, ckey, err := mpt.insertExtension(nil, nodeImpl.Path[1:], nodeImpl.NodeKey); err != nil {
			return nil, nil, err
		} else {
			nnode := NewFullNode(value)
			nnode.PutChild(nodeImpl.Path[0], ckey)
			return mpt.insertNode(node, nnode)
		}
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
	node, err := mpt.db.GetNode(key)
	if err != nil {
		Logger.Error("iterate - get node error", zap.Error(err))
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
			handler(ctx, path, nil, nodeImpl.Value)
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
		Logger.Debug("insert node",
			zap.Any("version", mpt.Version),
			zap.Any("root", hex.EncodeToString(mpt.Root)),
			zap.String("old node key", ohash),
			zap.String("new node key", newNode.GetHash()))
	}

	ckey := newNode.GetHashBytes()
	if err := mpt.db.PutNode(ckey, newNode); err != nil {
		return nil, nil, err
	}
	//If same node is inserted by client, don't add them into change collector
	if oldNode == nil {
		mpt.ChangeCollector.AddChange(oldNode, newNode)
	} else if bytes.Compare(oldNode.GetHashBytes(), ckey) != 0 { //delete previous node only if it isn`t the same as new one
		mpt.ChangeCollector.AddChange(oldNode, newNode)
		//NOTE: since leveldb is initiaized with propagate deletes as false, only newly created nodes will get deleted
		mpt.db.DeleteNode(oldNode.GetHashBytes())
	}
	return newNode, ckey, nil
}

func (mpt *MerklePatriciaTrie) deleteNode(node Node) error {
	if DebugMPTNode {
		Logger.Debug("delete node", zap.Any("version", mpt.Version), zap.String("key", node.GetHash()))
	}

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
		w.Write([]byte(" "))
	}
}

func (mpt *MerklePatriciaTrie) pp(w io.Writer, key Key, depth byte, initpad bool) error {
	if initpad {
		mpt.indent(w, depth)
	}
	node, err := mpt.db.GetNode(key)
	if err != nil {
		fmt.Fprintf(w, "err %v %v\n", ToHex(key), err)
		return err
	}
	switch nodeImpl := node.(type) {
	case *LeafNode:
		fmt.Fprintf(w, "L:%v (%v,%v)\n", ToHex(key), string(nodeImpl.Path), node.GetOrigin())
	case *ExtensionNode:
		fmt.Fprintf(w, "E:%v (%v,%v,%v)\n", ToHex(key), string(nodeImpl.Path), ToHex(nodeImpl.NodeKey), node.GetOrigin())
		mpt.pp(w, nodeImpl.NodeKey, depth+2, true)
	case *FullNode:
		w.Write([]byte("F:"))
		fmt.Fprintf(w, "%v (,%v)", ToHex(key), node.GetOrigin())
		w.Write([]byte("\n"))
		for idx, cnode := range nodeImpl.Children {
			if cnode == nil {
				continue
			}
			mpt.indent(w, depth+1)
			w.Write([]byte(fmt.Sprintf("%.2d ", idx)))
			mpt.pp(w, cnode, depth+2, false)
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
			missingNodeHander(ctx, path, key)
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

/*IsMPTValid - checks if the merkle tree is in valid state or not */
func IsMPTValid(mpt MerklePatriciaTrieI) error {
	return mpt.Iterate(context.TODO(), func(ctxt context.Context, path Path, key Key, node Node) error { return nil }, NodeTypeLeafNode|NodeTypeFullNode|NodeTypeExtensionNode)
}

/*GetChanges - get the list of changes */
func GetChanges(ctx context.Context, ndb NodeDB, start Sequence, end Sequence) (map[Sequence]MerklePatriciaTrieI, error) {
	mpts := make(map[Sequence]MerklePatriciaTrieI, int64(end-start+1))
	handler := func(ctx context.Context, key Key, node Node) error {
		origin := node.GetOrigin()
		if !(start <= origin && origin <= end) {
			return nil
		}
		mpt, ok := mpts[origin]
		if !ok {
			mndb := NewMemoryNodeDB()
			mpt = NewMerklePatriciaTrie(mndb, origin)
			mpts[origin] = mpt
		}
		mpt.GetNodeDB().PutNode(key, node)
		return nil
	}
	ndb.Iterate(ctx, handler)
	for _, mpt := range mpts {
		root := mpt.GetNodeDB().(*MemoryNodeDB).ComputeRoot()
		if root != nil {
			mpt.SetRoot(root.GetHashBytes())
		}
	}
	return mpts, nil
}

//Validate - implement interface - any sort of validations that can tell if the MPT is in a sane state
func (mpt *MerklePatriciaTrie) Validate() error {
	mpt.mutex.RLock()
	defer mpt.mutex.RUnlock()

	changes := mpt.getChangeCollector().GetChanges()
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
	if bytes.Compare(mpt.GetRoot(), mpt2.GetRoot()) == 0 {
		Logger.Debug("MergeMPTChanges - MPT merge changes with the same root")
		return nil
	}

	changes := mpt2.GetChangeCollector().GetChanges()
	deletes := mpt2.GetChangeCollector().GetDeletes()

	if DebugMPTNode {
		if err := mpt2.GetChangeCollector().Validate(); err != nil {
			Logger.Error("MergeMPTChanges - change collector validate", zap.Error(err))
		}

		Logger.Debug("MergeMPTChanges",
			zap.Int("change num", len(changes)),
			zap.Int("delete num", len(deletes)),
			zap.Any("old mpt version", mpt.Version),
			zap.Any("new mpt version", mpt2.GetVersion()),
			zap.String("old mpt root key", hex.EncodeToString(mpt.GetRoot())),
			zap.String("new mpt root key", hex.EncodeToString(mpt2.GetRoot())))
	}

	mpt.mutex.Lock()
	defer mpt.mutex.Unlock()

	for _, c := range changes {
		if _, _, err := mpt.insertNode(c.Old, c.New); err != nil {
			return err
		}
	}
	for _, d := range deletes {
		if err := mpt.deleteNode(d); err != nil {
			return err
		}
	}
	mpt.setRoot(mpt2.GetRoot())
	return nil
}

// MergeDB - implement interface.
func (mpt *MerklePatriciaTrie) MergeDB(ndb NodeDB, root Key) error {
	handler := func(ctx context.Context, key Key, node Node) error {
		mpt.mutex.Lock()
		_, _, err := mpt.insertNode(nil, node)
		mpt.mutex.Unlock()
		return err
	}
	mpt.SetRoot(root)
	return ndb.Iterate(context.TODO(), handler)
}
