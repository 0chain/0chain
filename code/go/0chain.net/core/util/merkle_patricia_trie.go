package util

import (
	"bytes"
	"context"
	"fmt"
	"io"

	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

var DebugMPTNode = false

/*MerklePatriciaTrie - it's a merkle tree and a patricia trie */
type MerklePatriciaTrie struct {
	Root            Key
	DB              NodeDB
	ChangeCollector ChangeCollectorI
	Version         Sequence
}

/*NewMerklePatriciaTrie - create a new patricia merkle trie */
func NewMerklePatriciaTrie(db NodeDB, version Sequence) *MerklePatriciaTrie {
	mpt := &MerklePatriciaTrie{DB: db}
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
	mpt.DB = ndb
}

/*GetNodeDB - implement interface */
func (mpt *MerklePatriciaTrie) GetNodeDB() NodeDB {
	return mpt.DB
}

//SetVersion - implement interface
func (mpt *MerklePatriciaTrie) SetVersion(version Sequence) {
	mpt.Version = version
}

//GetVersion - implement interface
func (mpt *MerklePatriciaTrie) GetVersion() Sequence {
	return mpt.Version
}

/*SetRoot - implement interface */
func (mpt *MerklePatriciaTrie) SetRoot(root Key) {
	mpt.Root = root
}

/*GetRoot - implement interface */
func (mpt *MerklePatriciaTrie) GetRoot() Key {
	return mpt.Root
}

/*GetNodeValue - get the value for a given path */
func (mpt *MerklePatriciaTrie) GetNodeValue(path Path) (Serializable, error) {
	rootKey := []byte(mpt.Root)
	if rootKey == nil || len(rootKey) == 0 {
		return nil, ErrValueNotPresent
	}
	rootNode, err := mpt.DB.GetNode(rootKey)
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
		return mpt.Delete(path)
	}
	eval := value.Encode()
	if eval == nil || len(eval) == 0 {
		return mpt.Delete(path)
	}
	_, newRootHash, err := mpt.insert(value, []byte(mpt.Root), path)
	if err != nil {
		return nil, err
	}
	mpt.SetRoot(newRootHash)
	return newRootHash, nil
}

/*Delete - delete a value from the trie */
func (mpt *MerklePatriciaTrie) Delete(path Path) (Key, error) {
	_, newRootHash, err := mpt.delete(Key(mpt.Root), path)
	if err != nil {
		return nil, err
	}
	mpt.SetRoot(newRootHash)
	return newRootHash, nil
}

//GetPathNodes - implement interface */
func (mpt *MerklePatriciaTrie) GetPathNodes(path Path) ([]Node, error) {
	nodes, err := mpt.getPathNodes([]byte(mpt.Root), path)
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
	node, err := mpt.DB.GetNode(key)
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
	return mpt.ChangeCollector
}

/*ResetChangeCollector - implement interface */
func (mpt *MerklePatriciaTrie) ResetChangeCollector(root Key) {
	mpt.ChangeCollector = NewChangeCollector()
	if root != nil {
		mpt.SetRoot(root)
	}
}

/*SaveChanges - implement interface */
func (mpt *MerklePatriciaTrie) SaveChanges(ndb NodeDB, includeDeletes bool) error {
	cc := mpt.ChangeCollector
	err := cc.UpdateChanges(ndb, mpt.Version, includeDeletes)
	if err != nil {
		return err
	}
	return nil
}

/*Iterate - iterate the entire trie */
func (mpt *MerklePatriciaTrie) Iterate(ctx context.Context, handler MPTIteratorHandler, visitNodeTypes byte) error {
	rootKey := Key(mpt.Root)
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
		nnode, err := mpt.DB.GetNode(ckey)
		if err != nil || nnode == nil {
			return nil, ErrNodeNotFound
		}
		return mpt.getNodeValue(path[1:], nnode)
	case *ExtensionNode:
		prefix := mpt.matchingPrefix(path, nodeImpl.Path)
		if len(prefix) == 0 {
			return nil, ErrValueNotPresent
		}
		if bytes.Compare(nodeImpl.Path, prefix) == 0 {
			nnode, err := mpt.DB.GetNode(nodeImpl.NodeKey)
			if err != nil || nnode == nil {
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
	if key == nil || len(key) == 0 {
		cnode := NewLeafNode(path, mpt.Version, value)
		return mpt.insertNode(nil, cnode)
	}

	node, err := mpt.DB.GetNode(key)
	if err != nil {
		return nil, nil, err
	}
	if len(path) == 0 {
		return mpt.insertAfterPathTraversal(value, node)
	}
	return mpt.insertAtNode(value, node, path)

}

func (mpt *MerklePatriciaTrie) delete(key Key, path Path) (Node, Key, error) {
	node, err := mpt.DB.GetNode(key)
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
			cnode := NewLeafNode(path[1:], mpt.Version, value)
			_, ckey, err = mpt.insertNode(nil, cnode)
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
			cnode := NewLeafNode(path[1:], mpt.Version, value)
			_, ckey, err := mpt.insertNode(nil, cnode)
			if err != nil {
				return nil, nil, err
			}
			nnode := NewFullNode(nodeImpl.GetValue())
			nnode.PutChild(path[0], ckey)
			return mpt.insertNode(node, nnode)
		}
		// updating an existing leaf
		if bytes.Compare(path, nodeImpl.Path) == 0 {
			nnode := NewLeafNode(nodeImpl.Path, mpt.Version, value)
			return mpt.insertNode(node, nnode)
		}

		prefix := mpt.matchingPrefix(path, nodeImpl.Path)
		plen := len(prefix)
		cnode := NewFullNode(nil)
		if bytes.Compare(prefix, path) == 0 { // path is a prefix of the existing leaf (node.Path = "hello world", path = "hello")
			gcnode2 := NewLeafNode(nodeImpl.Path[plen+1:], mpt.Version, nodeImpl.GetValue())
			_, gckey2, err := mpt.insertNode(nil, gcnode2)
			if err != nil {
				return nil, nil, err
			}
			cnode.PutChild(nodeImpl.Path[plen], gckey2)
			cnode.SetValue(value)
		} else if bytes.Compare(prefix, nodeImpl.Path) == 0 { // existing leaf path is a prefix of the path (node.Path = "hello", path = "hello world")
			gcnode1 := NewLeafNode(path[plen+1:], mpt.Version, value)
			_, gckey1, err := mpt.insertNode(nil, gcnode1)
			if err != nil {
				return nil, nil, err
			}
			cnode.PutChild(path[plen], gckey1)
			cnode.SetValue(nodeImpl.GetValue())
		} else { // existing leaf path and the given path have a prefix (one or more) and separate suffixes (node.Path = "hello world", path = "hello earth")
			// a full node that would contain children with indexes "w" and "e" , an extension node with path "hello "
			gcnode1 := NewLeafNode(path[plen+1:], mpt.Version, value)
			_, gckey1, err := mpt.insertNode(nil, gcnode1)
			if err != nil {
				return nil, nil, err
			}
			gcnode2 := NewLeafNode(nodeImpl.Path[plen+1:], mpt.Version, nodeImpl.GetValue())
			_, gckey2, err := mpt.insertNode(nil, gcnode2)
			if err != nil {
				return nil, nil, err
			}
			cnode.PutChild(path[plen], gckey1)
			cnode.PutChild(nodeImpl.Path[plen], gckey2)
		}
		if plen == 0 { // node.Path = "world" and path = "earth" => old leaf node is replaced with new branch node
			_, ckey, err := mpt.insertNode(node, cnode)
			if err != nil {
				return nil, nil, err
			}
			return cnode, ckey, nil
		}
		// if there is a matching prefix, it becomes an extension node
		// node.Path == "hello world", path = "hello earth", enode.Path = "hello " and replaces the old node.
		// enode.NodeKey points to ckey which is a new branch node that contains "world" and "earth" paths
		_, ckey, err := mpt.insertNode(nil, cnode)
		if err != nil {
			return nil, nil, err
		}
		enode := NewExtensionNode(prefix, ckey)
		return mpt.insertNode(node, enode)
	case *ExtensionNode:
		// updating an existing extension node with value
		if bytes.Compare(path, nodeImpl.Path) == 0 {
			_, ckey, err := mpt.insert(value, nodeImpl.NodeKey, Path{})
			if err != nil {
				return nil, nil, err
			}
			nnode := NewExtensionNode(path, ckey)
			return mpt.insertNode(node, nnode)
		}

		prefix := mpt.matchingPrefix(path, nodeImpl.Path)
		plen := len(prefix)

		if bytes.Compare(prefix, nodeImpl.Path) == 0 { // existing branch path is a prefix of the path (node.Path = "hello", path = "hello world")
			_, gckey1, err := mpt.insert(value, nodeImpl.NodeKey, path[plen:])
			if err != nil {
				return nil, nil, err
			}
			nnode := nodeImpl.Clone().(*ExtensionNode)
			nnode.NodeKey = gckey1
			return mpt.insertNode(node, nnode)
		}

		cnode := NewFullNode(nil)

		// existing branch path and the given path have a prefix (one or more) and separate suffixes (node.Path = "hello world", path = "hello earth")
		// a full node that would contain children with indexes "w" and "e" , an extension node with path "hello "
		if bytes.Compare(prefix, path) != 0 {
			gcnode1 := NewLeafNode(path[plen+1:], mpt.Version, value)
			_, gckey1, err := mpt.insertNode(nil, gcnode1)
			if err != nil {
				return nil, nil, err
			}
			cnode.PutChild(path[plen], gckey1)
		} else {
			// path is a prefix of the existing extension (node.Path = "hello world", path = "hello")
			cnode.SetValue(value)
		}
		var gckey2 Key
		if len(nodeImpl.Path) == plen+1 {
			gckey2 = nodeImpl.NodeKey
		} else {
			gcnode2 := NewExtensionNode(nodeImpl.Path[plen+1:], nodeImpl.NodeKey)
			_, gckey2, err = mpt.insertNode(nil, gcnode2)
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
		enode := NewExtensionNode(prefix, ckey)
		return mpt.insertNode(node, enode)
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
					nnode := NewLeafNode(nil, mpt.Version, nodeImpl.GetValue())
					return mpt.insertNode(node, nnode)
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
					ochild, err := mpt.DB.GetNode(otherChildKey)
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
						panic(fmt.Sprintf("unknown node type: %T %v %T", ochild, ochild, mpt.DB))
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
		case *ExtensionNode: // if extension child changes from full node to extension node, merge the extensions
			nnode := nodeImpl.Clone().(*ExtensionNode)
			nnode.Path = append(nnode.Path, cnodeImpl.Path...)
			nnode.NodeKey = cnodeImpl.NodeKey
			mpt.deleteNode(cnode)
			return mpt.insertNode(node, nnode)
		}
		nnode := nodeImpl.Clone().(*ExtensionNode)
		nnode.NodeKey = ckey
		return mpt.insertNode(node, nnode)
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
		if len(nodeImpl.Path) == 0 { // the value of an existing needs updated
			nnode := nodeImpl.Clone().(*LeafNode)
			nnode.SetOrigin(mpt.Version)
			nnode.SetValue(value)
			return mpt.insertNode(node, nnode)
		}
		// an existing leaf node needs to become a branch + leafnode (with one less path element as it's stored on the new branch) with value on the new branch
		cnode := NewLeafNode(nodeImpl.Path[1:], mpt.Version, nodeImpl.GetValue())
		_, ckey, err := mpt.insertNode(nil, cnode)
		if err != nil {
			return nil, nil, err
		}
		nnode := NewFullNode(value)
		nnode.PutChild(nodeImpl.Path[0], ckey)
		return mpt.insertNode(node, nnode)
	case *ExtensionNode:
		// an existing extension node becomes a branch + extension node (with one less path element as it's stored in the new branch) with value on the new branch
		cnode := nodeImpl.Clone().(*ExtensionNode)
		cnode.Path = nodeImpl.Path[1:]
		_, ckey, err := mpt.insertNode(nil, cnode)
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
		mpt.deleteNode(node)
		return nil, nil, nil
	case *ExtensionNode:
		panic("this should not happen!")
	default:
		panic(fmt.Sprintf("unknown node type: %T %v", node, node))
	}
}

func (mpt *MerklePatriciaTrie) iterate(ctx context.Context, path Path, key Key, handler MPTIteratorHandler, visitNodeTypes byte) error {
	node, err := mpt.DB.GetNode(key)
	if err != nil {
		if DebugMPTNode && Logger != nil {
			Logger.Error("iterate - get node error", zap.Error(err))
		}
		if herr := handler(ctx, path, key, node); herr != nil {
			return herr
		}
		return err
	}
	switch nodeImpl := node.(type) {
	case *LeafNode:
		if IncludesNodeType(visitNodeTypes, NodeTypeLeafNode) {
			err := handler(ctx, path, key, node)
			if err != nil {
				return err
			}
		}
		npath := append(path, nodeImpl.Path...)
		if IncludesNodeType(visitNodeTypes, NodeTypeValueNode) && nodeImpl.HasValue() {
			err := handler(ctx, npath, nil, nodeImpl.Value)
			if err != nil {
				return err
			}
		}
	case *FullNode:
		if IncludesNodeType(visitNodeTypes, NodeTypeFullNode) {
			err := handler(ctx, path, key, node)
			if err != nil {
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
			err := mpt.iterate(ctx, npath, child, handler, visitNodeTypes)
			if err != nil {
				if err == ErrNodeNotFound || err == ErrIteratingChildNodes {
					ecount++
				} else {
					return err
				}
			}
		}
		if ecount != 0 {
			return ErrIteratingChildNodes
		}
	case *ExtensionNode:
		if IncludesNodeType(visitNodeTypes, NodeTypeExtensionNode) {
			err = handler(ctx, path, key, node)
			if err != nil {
				return err
			}
		}
		npath := append(path, nodeImpl.Path...)
		return mpt.iterate(ctx, npath, nodeImpl.NodeKey, handler, visitNodeTypes)
	}
	return nil
}

func (mpt *MerklePatriciaTrie) insertNode(oldNode Node, newNode Node) (Node, Key, error) {
	if DebugMPTNode && Logger != nil {
		ohash := ""
		if oldNode != nil {
			ohash = oldNode.GetHash()
		}
		Logger.Info("insert node", zap.String("nn", newNode.GetHash()), zap.String("on", ohash))
	}
	ckey := newNode.GetHashBytes()
	err := mpt.DB.PutNode(ckey, newNode)
	if err != nil {
		return nil, nil, err
	}
	//If same node is inserted by client, don't add them into change collector
	if oldNode == nil || bytes.Compare(oldNode.GetHashBytes(), ckey) != 0 {
		mpt.ChangeCollector.AddChange(oldNode, newNode)
	}
	if oldNode != nil {
		//NOTE: since leveldb is initiaized with propagate deletes as false, only newly created nodes will get deleted
		mpt.DB.DeleteNode(oldNode.GetHashBytes())
	}
	return newNode, ckey, nil
}

func (mpt *MerklePatriciaTrie) deleteNode(node Node) error {
	if DebugMPTNode && Logger != nil {
		Logger.Info("delete node", zap.String("dn", node.GetHash()))
	}
	mpt.ChangeCollector.DeleteChange(node)
	return mpt.DB.DeleteNode(node.GetHashBytes())
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
	node, err := mpt.DB.GetNode(key)
	if initpad {
		mpt.indent(w, depth)
	}
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
			err := mpt.DB.MultiPutNode(keys, values)
			keys = keys[:0]
			values = values[:0]
			if err != nil {
				if DebugMPTNode && Logger != nil {
					Logger.Error("update version - multi put", zap.String("path", string(path)), zap.String("key", ToHex(key)), zap.Any("old_version", node.GetVersion()), zap.Any("new_version", version), zap.Error(err))
				}
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
			err := mpt.DB.MultiPutNode(keys, values)
			if err != nil {
				if DebugMPTNode && Logger != nil {
					Logger.Error("update version - multi put - last batch", zap.Error(err))
				}
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
		fmt.Printf("root:%T %v\n", root, root.GetHash())
		if root != nil {
			mpt.SetRoot(root.GetHashBytes())
		}
	}
	return mpts, nil
}

//Validate - implement interface - any sort of validations that can tell if the MPT is in a sane state
func (mpt *MerklePatriciaTrie) Validate() error {
	changes := mpt.GetChangeCollector().GetChanges()
	db := mpt.GetNodeDB()
	switch dbImpl := db.(type) {
	case *MemoryNodeDB:
	case *LevelNodeDB:
		db = dbImpl.C
	case *PNodeDB:
		return nil
	}
	for _, c := range changes {
		if c.Old == nil {
			continue
		}
		if _, err := db.GetNode(c.Old.GetHashBytes()); err == nil {
			return fmt.Errorf(ErrIntermediateNodeExists.Error(), c.Old, c.Old.GetHash(), c.New, c.New.GetHash())
		}
	}
	return nil
}

//MergeMPTChanges - implement interface
func (mpt *MerklePatriciaTrie) MergeMPTChanges(mpt2 MerklePatriciaTrieI) error {
	if DebugMPTNode && Logger != nil {
		if err := mpt2.GetChangeCollector().Validate(); err != nil {
			Logger.Error("MergeMPTChanges - change collector validate", zap.Error(err))
		}
	}
	changes := mpt2.GetChangeCollector().GetChanges()
	for _, c := range changes {
		_, _, err := mpt.insertNode(c.Old, c.New)
		if err != nil {
			return err
		}
	}
	deletes := mpt2.GetChangeCollector().GetDeletes()
	for _, d := range deletes {
		err := mpt.deleteNode(d)
		if err != nil {
			return err
		}
	}
	mpt.SetRoot(mpt2.GetRoot())
	return nil
}

//MergeDB - implement interface
func (mpt *MerklePatriciaTrie) MergeDB(ndb NodeDB, root Key) error {
	handler := func(ctx context.Context, key Key, node Node) error {
		_, _, err := mpt.insertNode(nil, node)
		return err
	}
	mpt.SetRoot(root)
	return ndb.Iterate(context.TODO(), handler)
}
