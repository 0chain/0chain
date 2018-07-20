package util

import (
	"bytes"
	"context"
	"fmt"
	"io"
)

/*MerklePatriciaTrie - it's a merkle tree and a patricia trie */
type MerklePatriciaTrie struct {
	Root Key
	DB   NodeDB
}

/*NewMerklePatriciaTrie - create a new patricia merkle trie */
func NewMerklePatriciaTrie(db NodeDB) *MerklePatriciaTrie {
	mpt := &MerklePatriciaTrie{DB: db}
	return mpt
}

/*GetNodeDB - implement interface */
func (mpt *MerklePatriciaTrie) GetNodeDB() NodeDB {
	return mpt.DB
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
	return mpt.getNodeKey(path, rootNode)
}

/*Insert - inserts (updates) a value into this trie and updates the trie all the way up and produces a new root */
func (mpt *MerklePatriciaTrie) Insert(path Path, value Serializable, cc ChangeCollectorI) (Key, error) {
	if value == nil {
		return mpt.Delete(path, cc)
	}
	eval := value.Encode()
	if eval == nil || len(eval) == 0 {
		return mpt.Delete(path, cc)
	}
	_, newRootHash, err := mpt.insert(cc, value, []byte(mpt.Root), path)
	if err != nil {
		return nil, err
	}
	return newRootHash, nil
}

/*Delete - delete a value from the trie */
func (mpt *MerklePatriciaTrie) Delete(path Path, cc ChangeCollectorI) (Key, error) {
	_, newRootHash, err := mpt.delete(cc, Key(mpt.Root), path)
	if err != nil {
		return nil, err
	}
	return newRootHash, nil
}

/*Iterate - iterate the entire trie */
func (mpt *MerklePatriciaTrie) Iterate(ctx context.Context, handler MPTIteratorHandler, visitNodeTypes byte) error {
	rootKey := Key(mpt.Root)
	if rootKey == nil || len(rootKey) == 0 {
		return nil
	}
	return mpt.iterate(ctx, Path{}, rootKey, handler, visitNodeTypes)
}

/*PrettyPrint - print this trie */
func (mpt *MerklePatriciaTrie) PrettyPrint(w io.Writer) error {
	return mpt.pp(w, Key(mpt.Root), 0, false)
}

func (mpt *MerklePatriciaTrie) getNodeKey(path Path, node Node) (Serializable, error) {
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
		if err != nil {
			return nil, ErrNodeNotFound
		}
		return mpt.getNodeKey(path[1:], nnode)
	case *ExtensionNode:
		prefix := mpt.matchingPrefix(path, nodeImpl.Path)
		if bytes.Compare(nodeImpl.Path, prefix) == 0 {
			nnode, err := mpt.DB.GetNode(nodeImpl.NodeKey)
			if err != nil {
				return nil, ErrNodeNotFound
			}
			return mpt.getNodeKey(path[len(prefix):], nnode)
		}
		return nil, ErrValueNotPresent
	default:
		panic("unknown node type")
	}
}

func (mpt *MerklePatriciaTrie) insert(cc ChangeCollectorI, value Serializable, key Key, path Path) (Node, Key, error) {
	if key == nil || len(key) == 0 {
		cnode := NewLeafNode(path, value)
		return mpt.insertNode(cc, nil, cnode)
	}

	node, err := mpt.DB.GetNode(key)
	if err != nil {
		return nil, nil, err
	}
	if len(path) == 0 {
		return mpt.insertAfterPathTraversal(cc, value, node)
	}
	return mpt.insertAtNode(cc, value, node, path)

}

func (mpt *MerklePatriciaTrie) delete(cc ChangeCollectorI, key Key, path Path) (Node, Key, error) {
	node, err := mpt.DB.GetNode(key)
	if err != nil {
		return nil, nil, err
	}
	if len(path) == 0 {
		return mpt.deleteAfterPathTraversal(cc, node)
	}
	return mpt.deleteAtNode(cc, node, path)
}

func (mpt *MerklePatriciaTrie) insertAtNode(cc ChangeCollectorI, value Serializable, node Node, path Path) (Node, Key, error) {
	var err error
	switch nodeImpl := node.(type) {
	case *FullNode:
		ckey := nodeImpl.GetChild(path[0])
		if ckey == nil {
			cnode := NewLeafNode(path[1:], value)
			_, ckey, err = mpt.insertNode(cc, nil, cnode)
		} else {
			_, ckey, err = mpt.insert(cc, value, ckey, path[1:])
		}
		if err != nil {
			return nil, nil, err
		}
		nnode := nodeImpl.Clone().(*FullNode)
		nnode.PutChild(path[0], ckey)
		return mpt.insertNode(cc, node, nnode)
	case *LeafNode:
		if len(nodeImpl.Path) == 0 {
			cnode := NewLeafNode(path[1:], value)
			_, ckey, err := mpt.insertNode(cc, nil, cnode)
			if err != nil {
				return nil, nil, err
			}
			nnode := NewFullNode(nodeImpl.GetValue())
			nnode.PutChild(path[0], ckey)
			return mpt.insertNode(cc, node, nnode)
		}
		// updating an existing leaf
		if bytes.Compare(path, nodeImpl.Path) == 0 {
			nnode := NewLeafNode(nodeImpl.Path, value)
			return mpt.insertNode(cc, node, nnode)
		}

		prefix := mpt.matchingPrefix(path, nodeImpl.Path)
		plen := len(prefix)
		cnode := &FullNode{}

		if bytes.Compare(prefix, path) == 0 { // path is a prefix of the existing leaf (node.Path = "hello world", path = "hello")
			gcnode2 := NewLeafNode(nodeImpl.Path[plen+1:], nodeImpl.GetValue())
			_, gckey2, err := mpt.insertNode(cc, nil, gcnode2)
			if err != nil {
				return nil, nil, err
			}
			cnode.PutChild(nodeImpl.Path[plen], gckey2)
			cnode.SetValue(value)
		} else if bytes.Compare(prefix, nodeImpl.Path) == 0 { // existing leaf path is a prefix of the path (node.Path = "hello", path = "hello world")
			gcnode1 := NewLeafNode(path[plen+1:], value)
			_, gckey1, err := mpt.insertNode(cc, nil, gcnode1)
			if err != nil {
				return nil, nil, err
			}
			cnode.PutChild(path[plen], gckey1)
			cnode.SetValue(nodeImpl.GetValue())
		} else { // existing leaf path and the given path have a prefix (or or more) and separate suffixes (node.Path = "hello world", path = "hello earth")
			// a full node that would contain children with indexes "w" and "e" , an extension node with path "hello "
			gcnode1 := NewLeafNode(path[plen+1:], value)
			_, gckey1, err := mpt.insertNode(cc, nil, gcnode1)
			if err != nil {
				return nil, nil, err
			}
			gcnode2 := NewLeafNode(nodeImpl.Path[plen+1:], nodeImpl.GetValue())
			_, gckey2, err := mpt.insertNode(cc, nil, gcnode2)
			if err != nil {
				return nil, nil, err
			}
			cnode.PutChild(path[plen], gckey1)
			cnode.PutChild(nodeImpl.Path[plen], gckey2)
		}
		var prevNode Node
		if plen == 0 { // node.Path = "world" and path = "earth" => old leaf node is replaced with new branch node
			prevNode = node
		}
		_, ckey, err := mpt.insertNode(cc, prevNode, cnode)
		if err != nil {
			return nil, nil, err
		}
		if plen == 0 {
			return cnode, ckey, nil
		}
		// if there is a matching prefix, it becomes an extension node
		// node.Path == "hello world", path = "hello earth", enode.Path = "hello " and replaces the old node.
		// enode.NodeKey points to ckey which is a new branch node that contains "world" and "earth" paths
		enode := &ExtensionNode{Path: prefix, NodeKey: ckey}
		return mpt.insertNode(cc, node, enode)
	case *ExtensionNode:
		// updating an existing extension node with value
		if bytes.Compare(path, nodeImpl.Path) == 0 {
			_, ckey, err := mpt.insert(cc, value, nodeImpl.NodeKey, Path{})
			if err != nil {
				return nil, nil, err
			}
			nnode := &ExtensionNode{Path: path, NodeKey: ckey}
			return mpt.insertNode(cc, node, nnode)
		}

		prefix := mpt.matchingPrefix(path, nodeImpl.Path)
		plen := len(prefix)

		if bytes.Compare(prefix, nodeImpl.Path) == 0 { // existing branch path is a prefix of the path (node.Path = "hello", path = "hello world")
			_, gckey1, err := mpt.insert(cc, value, nodeImpl.NodeKey, path[plen:])
			if err != nil {
				return nil, nil, err
			}
			nnode := nodeImpl.Clone().(*ExtensionNode)
			nnode.NodeKey = gckey1
			return mpt.insertNode(cc, node, nnode)
		}

		cnode := &FullNode{}

		// existing leaf path and the given path have a prefix (one or more) and separate suffixes (node.Path = "hello world", path = "hello earth")
		// a full node that would contain children with indexes "w" and "e" , an extension node with path "hello "
		if bytes.Compare(prefix, path) != 0 {
			gcnode1 := NewLeafNode(path[plen+1:], value)
			_, gckey1, err := mpt.insertNode(cc, nil, gcnode1)
			if err != nil {
				return nil, nil, err
			}
			cnode.PutChild(path[plen], gckey1)
		} else {
			// path is a prefix of the existing extension (node.Path = "hello world", path = "hello")
			cnode.SetValue(value)
		}
		gcnode2 := &ExtensionNode{Path: nodeImpl.Path[plen+1:], NodeKey: nodeImpl.NodeKey}
		_, gckey2, err := mpt.insertNode(cc, nil, gcnode2)
		if err != nil {
			return nil, nil, err
		}
		cnode.PutChild(nodeImpl.Path[plen], gckey2)
		var prevNode Node
		if plen == 0 { // node.Path = "world" and path = "earth" => old leaf node is replaced with new branch node
			prevNode = node
		}
		_, ckey, err := mpt.insertNode(cc, prevNode, cnode)
		if err != nil {
			return nil, nil, err
		}
		if plen > 0 { // if there is a matching prefix, it becomes an extension node
			// node.Path == "hello world", path = "hello earth", enode.Path = "hello " and replaces the old node.
			// enode.NodeKey points to ckey which is a new branch node that contains "world" and "earth" paths
			enode := &ExtensionNode{Path: prefix, NodeKey: ckey}
			return mpt.insertNode(cc, node, enode)
		}
		return cnode, ckey, nil
	default:
		panic("uknown node type")
	}
}

func (mpt *MerklePatriciaTrie) deleteAtNode(cc ChangeCollectorI, node Node, path Path) (Node, Key, error) {
	switch nodeImpl := node.(type) {
	case *FullNode:
		_, ckey, err := mpt.delete(cc, nodeImpl.GetChild(path[0]), path[1:])
		if err != nil {
			return nil, nil, err
		}
		if ckey == nil {
			numChildren := nodeImpl.GetNumChildren()
			if numChildren == 1 {
				if nodeImpl.HasValue() { // a full node with no children anymore but with a value becomes a leaf node
					nnode := NewLeafNode(nil, nodeImpl.GetValue())
					return mpt.insertNode(cc, node, nnode)
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
						nnode := &ExtensionNode{Path: npath, NodeKey: otherChildKey}
						return mpt.insertNode(cc, nil, nnode)
					case *LeafNode:
						if onodeImpl.Path != nil {
							npath = append(npath, onodeImpl.Path...)
						}
						lnode := ochild.Clone().(*LeafNode)
						lnode.Path = npath
						nnode = lnode
					case *ExtensionNode:
						if onodeImpl.Path != nil {
							npath = append(npath, onodeImpl.Path...)
						}
						lnode := ochild.Clone().(*ExtensionNode)
						lnode.Path = npath
						nnode = lnode
					default:
						panic("invalid node type")
					}
					return mpt.insertNode(cc, ochild, nnode)
				}
			}
		}
		nnode := nodeImpl.Clone().(*FullNode)
		nnode.PutChild(path[0], ckey)
		return mpt.insertNode(cc, node, nnode)
	case *LeafNode:
		if bytes.Compare(path, nodeImpl.Path) != 0 {
			return node, node.GetHashBytes(), ErrValueNotPresent // There is nothing to delete
		}
		return mpt.deleteAfterPathTraversal(cc, node)
	case *ExtensionNode:
		prefix := mpt.matchingPrefix(path, nodeImpl.Path)
		if bytes.Compare(prefix, nodeImpl.Path) != 0 {
			return node, node.GetHashBytes(), ErrValueNotPresent // There is nothing to delete
		}
		cnode, ckey, err := mpt.delete(cc, nodeImpl.NodeKey, path[len(nodeImpl.Path):])
		if err != nil {
			return nil, nil, err
		}
		if lnode, ok := cnode.(*LeafNode); ok { // if extension child changes from full node to leaf, convert the extension into a leaf node
			nnode := cnode.Clone().(*LeafNode)
			nnode.Path = append(nodeImpl.Path, lnode.Path...)
			nnode.SetValue(lnode.GetValue())
			return mpt.insertNode(cc, cnode, nnode)
		}
		nnode := nodeImpl.Clone().(*ExtensionNode)
		nnode.NodeKey = ckey
		return mpt.insertNode(cc, node, nnode)
	default:
		panic(fmt.Sprintf("uknown node type: %T %v", node, node))
	}
}

func (mpt *MerklePatriciaTrie) insertAfterPathTraversal(cc ChangeCollectorI, value Serializable, node Node) (Node, Key, error) {
	switch nodeImpl := node.(type) {
	case *FullNode:
		// The value of the branch needs to be updated
		nnode := nodeImpl.Clone().(*FullNode)
		nnode.SetValue(value)
		return mpt.insertNode(cc, node, nnode)
	case *LeafNode:
		if len(nodeImpl.Path) == 0 { // the value of an existing needs updated
			nnode := nodeImpl.Clone().(*LeafNode)
			nnode.SetValue(value)
			return mpt.insertNode(cc, node, nnode)
		}
		// an existing leaf node needs to become a branch + leafnode (with one less path element as it's stored on the new branch) with value on the new branch
		cnode := NewLeafNode(nodeImpl.Path[1:], nodeImpl.GetValue())
		_, ckey, err := mpt.insertNode(cc, nil, cnode)
		if err != nil {
			return nil, nil, err
		}
		nnode := NewFullNode(value)
		nnode.PutChild(nodeImpl.Path[0], ckey)
		return mpt.insertNode(cc, node, nnode)
	case *ExtensionNode:
		// an existing extension node becomes a branch + extension node (with one less path element as it's stored in the new branch) with value on the new branch
		cnode := nodeImpl.Clone().(*ExtensionNode)
		cnode.Path = nodeImpl.Path[1:]
		_, ckey, err := mpt.insertNode(cc, nil, cnode)
		if err != nil {
			return nil, nil, err
		}
		nnode := NewFullNode(value)
		nnode.PutChild(nodeImpl.Path[0], ckey)
		return mpt.insertNode(cc, node, nnode)
	default:
		panic("uknown node type")
	}
}

func (mpt *MerklePatriciaTrie) deleteAfterPathTraversal(cc ChangeCollectorI, node Node) (Node, Key, error) {
	switch nodeImpl := node.(type) {
	case *FullNode:
		// The value of the branch needs to be updated
		nnode := nodeImpl.Clone().(*FullNode)
		nnode.SetValue(nil)
		cc.AddChange(node, nnode)
		if nodeImpl.HasValue() {
			cc.DeleteChange(nodeImpl.Value)
		}
		return mpt.insertNode(cc, node, nnode)
	case *LeafNode:
		cc.DeleteChange(node)
		if nodeImpl.HasValue() {
			cc.DeleteChange(nodeImpl.Value)
		}
		return nil, nil, nil
	case *ExtensionNode:
		panic("this should not happen!")
	default:
		panic("uknown node type")
	}
}

func (mpt *MerklePatriciaTrie) iterate(ctx context.Context, path Path, key Key, handler MPTIteratorHandler, visitNodeTypes byte) error {
	node, err := mpt.DB.GetNode(key)
	if err != nil {
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
		if err != nil {
			return err
		}
		if IncludesNodeType(visitNodeTypes, NodeTypeValueNode) && nodeImpl.HasValue() {
			handler(ctx, npath, nil, nodeImpl.Value)
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
		for i := byte(0); i < 16; i++ {
			pe := nodeImpl.indexToByte(i)
			child := nodeImpl.GetChild(pe)
			if child == nil {
				continue
			}
			npath := append(path, pe)
			err := mpt.iterate(ctx, npath, child, handler, visitNodeTypes)
			if err != nil {
				return err
			}
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

func (mpt *MerklePatriciaTrie) insertNode(cc ChangeCollectorI, oldNode Node, newNode Node) (Node, Key, error) {
	cc.AddChange(oldNode, newNode)
	/* NOTE: This is not required as leaf node stores the encoded value
	valNode := GetValueNode(newNode)
	if valNode != (*ValueNode)(nil) { // golang doesn't allow nil comparision of interfaces when an typed-nil-value is assigned to it and GetValueNode returns *ValueNode
		oldValNode := GetValueNode(oldNode)
		if oldValNode != (*ValueNode)(nil) {
			cc.AddChange(oldValNode, valNode)
		} else {
			cc.AddChange(nil, valNode)
		}
	}*/
	ckey := newNode.GetHashBytes()
	err := mpt.DB.PutNode(ckey, newNode)
	if err != nil {
		return nil, nil, err
	}
	return newNode, ckey, nil
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
	fmt.Printf("%v ", ToHex(key))
	switch nodeImpl := node.(type) {
	case *LeafNode:
		fmt.Fprintf(w, "L:%v (%v)\n", ToHex(nodeImpl.Encode()), string(nodeImpl.Path))
	case *ExtensionNode:
		fmt.Fprintf(w, "E:%v (%v,%v)\n", ToHex(nodeImpl.Encode()), string(nodeImpl.Path), ToHex(nodeImpl.NodeKey))
		mpt.pp(w, nodeImpl.NodeKey, depth+2, true)
	case *FullNode:
		w.Write([]byte("F:"))
		if nodeImpl.HasValue() {
			fmt.Fprintf(w, "%v", GetValueNode(nodeImpl).GetHash())
		}
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

/*UpdateOrigin - updates the origin of all the nodes in this tree to the given origin */
func (mpt *MerklePatriciaTrie) UpdateOrigin(ctx context.Context, origin Origin) error {
	fmt.Printf("updating to origin: %v\n", origin)
	handler := func(ctx context.Context, path Path, key Key, node Node) error {
		if node.GetOrigin() >= origin {
			return nil
		}
		node.SetOrigin(origin)
		fmt.Printf("updating node: %v\n", string(key))
		err := mpt.DB.PutNode(key, node)
		if err != nil {
			fmt.Printf("DEBUG: updated origin to : %v %v\n", origin, err)
		}
		return err
	}
	return mpt.Iterate(ctx, handler, NodeTypeLeafNode|NodeTypeFullNode|NodeTypeExtensionNode)
}

/*PruneBelowOrigin - prune the state below the given origin */
func (mpt *MerklePatriciaTrie) PruneBelowOrigin(ctx context.Context, origin Origin) error {
	fmt.Printf("pruning to origin: %v\n", origin)
	handler := func(ctx context.Context, key Key, node Node) error {
		if node.GetOrigin() >= origin {
			return nil
		}
		err := mpt.DB.DeleteNode(key)
		if err != nil {
			fmt.Printf("DEBUG: deleting node: %v %v\n", node.GetOrigin(), err)
		}
		return err
	}
	return mpt.DB.Iterate(ctx, handler)
}
