package util

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"

	"0chain.net/core/encryption"
)

const (
	NodeTypeValueNode     = 1
	NodeTypeLeafNode      = 2
	NodeTypeFullNode      = 4
	NodeTypeExtensionNode = 8
	NodeTypesAll          = NodeTypeValueNode | NodeTypeLeafNode | NodeTypeFullNode | NodeTypeExtensionNode
)

//Separator - used to separate fields when creating data array to hash
const Separator = ':'

//ErrInvalidEncoding - error to indicate invalid encoding
var ErrInvalidEncoding = errors.New("invalid node encoding")

//PathElements - all the bytes that can be used as path elements as ascii characters
var PathElements = []byte("0123456789abcdef")

/*Node - a node interface */
type Node interface {
	Clone() Node
	GetNodeType() byte
	SecureSerializableValueI
	OriginTrackerI
	GetOriginTracker() OriginTrackerI
	SetOriginTracker(ot OriginTrackerI)
}

//OriginTrackerNode - a node that implements origin tracking
type OriginTrackerNode struct {
	OriginTracker OriginTrackerI `json:"o,omitempty"`
}

//NewOriginTrackerNode - create a new origin tracker node
func NewOriginTrackerNode() *OriginTrackerNode {
	otn := &OriginTrackerNode{}
	otn.OriginTracker = &OriginTracker{}
	return otn
}

//Clone - clone the given origin tracker node
func (otn *OriginTrackerNode) Clone() *OriginTrackerNode {
	clone := NewOriginTrackerNode()
	clone.OriginTracker.SetOrigin(otn.GetOrigin())
	clone.OriginTracker.SetVersion(otn.GetVersion())
	return clone
}

//SetOriginTracker - implement interface
func (otn *OriginTrackerNode) SetOriginTracker(ot OriginTrackerI) {
	otn.OriginTracker = ot
}

//SetOrigin - implement interface
func (otn *OriginTrackerNode) SetOrigin(origin Sequence) {
	otn.OriginTracker.SetOrigin(origin)
}

//GetOrigin - implement interface
func (otn *OriginTrackerNode) GetOrigin() Sequence {
	return otn.OriginTracker.GetOrigin()
}

//SetVersion - implement interface
func (otn *OriginTrackerNode) SetVersion(version Sequence) {
	otn.OriginTracker.SetVersion(version)
}

//GetVersion - implement interface
func (otn *OriginTrackerNode) GetVersion() Sequence {
	return otn.OriginTracker.GetVersion()
}

//Write - implement interface
func (otn *OriginTrackerNode) Write(w io.Writer) error {
	return otn.OriginTracker.Write(w)
}

//Read - implement interface
func (otn *OriginTrackerNode) Read(r io.Reader) error {
	return otn.OriginTracker.Read(r)
}

//GetOriginTracker - implement interface
func (otn *OriginTrackerNode) GetOriginTracker() OriginTrackerI {
	return otn.OriginTracker
}

/*ValueNode - any node that holds a value should implement this */
type ValueNode struct {
	Value              Serializable `json:"v"`
	*OriginTrackerNode `json:"o,omitempty"`
}

//NewValueNode - create a new value node
func NewValueNode() *ValueNode {
	vn := &ValueNode{}
	//vn.OriginTrackerNode = NewOriginTrackerNode()
	return vn
}

/*Clone - implement interface */
func (vn *ValueNode) Clone() Node {
	clone := NewValueNode()
	clone.OriginTrackerNode = vn.OriginTrackerNode.Clone()
	clone.SetValue(vn.GetValue())
	return clone
}

/*GetNodeType - implement interface */
func (vn *ValueNode) GetNodeType() byte {
	return NodeTypeValueNode
}

/*GetHash - implements SecureSerializableValue interface */
func (vn *ValueNode) GetHash() string {
	return ToHex(vn.GetHashBytes())
}

/*GetHashBytes - implement SecureSerializableValue interface */
func (vn *ValueNode) GetHashBytes() []byte {
	if vn.Value == nil {
		return nil
	}
	return encryption.RawHash(vn.Value.Encode())
}

/*GetValue - get the value store in this node */
func (vn *ValueNode) GetValue() Serializable {
	return vn.Value
}

/*SetValue - set the value stored in this node */
func (vn *ValueNode) SetValue(value Serializable) {
	vn.Value = value
}

/*HasValue - check if the value stored is empty */
func (vn *ValueNode) HasValue() bool {
	if vn.Value == nil {
		return false
	}
	encoding := vn.Value.Encode()
	if encoding == nil || len(encoding) == 0 {
		return false
	}
	return true
}

/*Encode - overwrite interface method */
func (vn *ValueNode) Encode() []byte {
	buf := bytes.NewBuffer(nil)
	writeNodePrefix(buf, vn)
	if vn.HasValue() {
		buf.Write(vn.GetValue().Encode())
	}
	return buf.Bytes()
}

/*Decode - overwrite interface method */
func (vn *ValueNode) Decode(buf []byte) error {
	pspv := &SecureSerializableValue{}
	err := pspv.Decode(buf)
	if err != nil {
		return err
	}
	vn.SetValue(pspv)
	return nil
}

/*LeafNode - a node that represents the leaf that contains a value and an optional path */
type LeafNode struct {
	Path               Path       `json:"p,omitempty"`
	Value              *ValueNode `json:"v"`
	*OriginTrackerNode `json:"o"`
}

/*NewLeafNode - create a new leaf node */
func NewLeafNode(path Path, origin Sequence, value Serializable) *LeafNode {
	ln := &LeafNode{}
	ln.OriginTrackerNode = NewOriginTrackerNode()
	ln.Path = path
	ln.SetOrigin(origin)
	ln.SetValue(value)
	return ln
}

/*GetHash - implements SecureSerializableValue interface */
func (ln *LeafNode) GetHash() string {
	return ToHex(ln.GetHashBytes())
}

/*GetHashBytes - implement interface */
func (ln *LeafNode) GetHashBytes() []byte {
	buf := bytes.NewBuffer(nil)
	binary.Write(buf, binary.LittleEndian, ln.GetOrigin())
	ln.encode(buf)
	return encryption.RawHash(buf.Bytes())
}

/*Clone - implement interface */
func (ln *LeafNode) Clone() Node {
	clone := &LeafNode{}
	clone.OriginTrackerNode = NewOriginTrackerNode()
	clone.Path = ln.Path // path will never be updated inplace and so ok
	clone.SetValue(ln.GetValue())
	return clone
}

/*GetNodeType - implement interface */
func (ln *LeafNode) GetNodeType() byte {
	return NodeTypeLeafNode
}

/*Encode - implement interface */
func (ln *LeafNode) Encode() []byte {
	buf := bytes.NewBuffer(nil)
	writeNodePrefix(buf, ln)
	ln.encode(buf)
	return buf.Bytes()
}

func (ln *LeafNode) encode(buf *bytes.Buffer) {
	if len(ln.Path) > 0 {
		buf.Write(ln.Path)
	}
	buf.WriteByte(Separator)
	if ln.HasValue() {
		buf.Write(ln.GetValue().Encode())
	}
}

/*Decode - implement interface */
func (ln *LeafNode) Decode(buf []byte) error {
	idx := bytes.IndexByte(buf, Separator)
	if idx < 0 {
		return ErrInvalidEncoding
	}
	ln.Path = buf[:idx]
	buf = buf[idx+1:]
	if len(buf) == 0 {
		ln.SetValue(nil)
	} else {
		vn := NewValueNode()
		vn.Decode(buf)
		ln.Value = vn
	}
	return nil
}

/*HasValue - implement interface */
func (ln *LeafNode) HasValue() bool {
	return ln.Value != nil && ln.Value.HasValue()
}

/*GetValue - implement interface */
func (ln *LeafNode) GetValue() Serializable {
	if !ln.HasValue() {
		return nil
	}
	return ln.Value.GetValue()
}

/*SetValue - implement interface */
func (ln *LeafNode) SetValue(value Serializable) {
	if ln.Value == nil {
		ln.Value = NewValueNode()
	}
	ln.Value.SetValue(value)
}

/*FullNode - a branch node that can contain 16 children and a value */
type FullNode struct {
	Children           [16][]byte `json:"c"`
	Value              *ValueNode `json:"v,omitempty"` // This may not be needed as our path is fixed in size
	*OriginTrackerNode `json:"o,omitempty"`
}

/*NewFullNode - create a new full node */
func NewFullNode(value Serializable) *FullNode {
	fn := &FullNode{}
	fn.OriginTrackerNode = NewOriginTrackerNode()
	fn.SetValue(value)
	return fn
}

/*GetHash - implements SecureSerializableValue interface */
func (fn *FullNode) GetHash() string {
	return ToHex(fn.GetHashBytes())
}

/*GetHashBytes - implement interface */
func (fn *FullNode) GetHashBytes() []byte {
	buf := bytes.NewBuffer(nil)
	fn.encode(buf)
	return encryption.RawHash(buf.Bytes())
}

/*Encode - implement interface */
func (fn *FullNode) Encode() []byte {
	buf := bytes.NewBuffer(nil)
	writeNodePrefix(buf, fn)
	fn.encode(buf)
	return buf.Bytes()
}

func (fn *FullNode) encode(buf *bytes.Buffer) {
	for i := byte(0); i < 16; i++ {
		child := fn.GetChild(fn.indexToByte(i))
		if child != nil {
			buf.Write([]byte(ToHex(child)))
		}
		buf.WriteByte(Separator)
	}
	if fn.HasValue() {
		buf.Write(fn.GetValue().Encode())
	}
}

/*Decode - implement interface */
func (fn *FullNode) Decode(buf []byte) error {
	for i := byte(0); i < 16; i++ {
		idx := bytes.IndexByte(buf, Separator)
		if idx < 0 {
			return ErrInvalidEncoding
		}
		if idx > 0 {
			key := make([]byte, 32)
			_, err := hex.Decode(key, buf[:idx])
			if err != nil {
				return err
			}
			fn.PutChild(fn.indexToByte(i), key)
		}
		buf = buf[idx+1:]
	}
	if len(buf) == 0 {
		fn.SetValue(nil)
	} else {
		vn := NewValueNode()
		vn.Decode(buf)
		fn.Value = vn
	}
	return nil
}

/*Clone - implement interface */
func (fn *FullNode) Clone() Node {
	clone := &FullNode{}
	clone.OriginTrackerNode = NewOriginTrackerNode()
	for idx, ckey := range fn.Children {
		clone.Children[idx] = ckey // ckey will never be updated inplace and so ok
	}
	if fn.HasValue() {
		clone.SetValue(fn.GetValue())
	}
	return clone
}

/*GetNodeType - implement interface */
func (fn *FullNode) GetNodeType() byte {
	return NodeTypeFullNode
}

func (fn *FullNode) index(c byte) byte {
	if c >= 48 && c <= 57 {
		return c - 48
	}
	if c >= 97 && c <= 102 {
		return 10 + c - 97
	}
	if c >= 65 && c <= 70 {
		return 10 + c - 65
	}
	panic("Invalid byte for index in Patricia Merkle Trie")
}

func (fn *FullNode) indexToByte(idx byte) byte {
	if idx < 10 {
		return 48 + idx
	}
	return 97 + (idx - 10)
}

/*GetNumChildren - get the number of children in this node */
func (fn *FullNode) GetNumChildren() byte {
	var count byte
	for _, child := range fn.Children {
		if child != nil {
			count++
		}
	}
	return count
}

/*GetChild - get the child at the given hex index */
func (fn *FullNode) GetChild(hex byte) []byte {
	return fn.Children[fn.index(hex)]
}

/*PutChild - put the child at the given hex index */
func (fn *FullNode) PutChild(hex byte, child []byte) {
	fn.Children[fn.index(hex)] = child
}

/*HasValue - implement interface */
func (fn *FullNode) HasValue() bool {
	return fn.Value != nil && fn.Value.HasValue()
}

/*GetValue - implement interface */
func (fn *FullNode) GetValue() Serializable {
	if fn.Value == nil {
		return nil
	}
	return fn.Value.GetValue()
}

/*SetValue - implement interface */
func (fn *FullNode) SetValue(value Serializable) {
	if fn.Value == nil {
		fn.Value = NewValueNode()
	}
	fn.Value.SetValue(value)
}

/*ExtensionNode - a multi-char length path along which there are no branches, at the end of this path there should be full node */
type ExtensionNode struct {
	Path               Path `json:"p"`
	NodeKey            Key  `json:"k"`
	*OriginTrackerNode `json:"o,omitempty"`
}

/*NewExtensionNode - create a new extension node */
func NewExtensionNode(path Path, key Key) *ExtensionNode {
	en := &ExtensionNode{}
	en.OriginTrackerNode = NewOriginTrackerNode()
	en.Path = path
	en.NodeKey = key
	return en
}

/*GetHash - implements SecureSerializableValue interface */
func (en *ExtensionNode) GetHash() string {
	return ToHex(en.GetHashBytes())
}

/*GetHashBytes - implement interface */
func (en *ExtensionNode) GetHashBytes() []byte {
	buf := bytes.NewBuffer(nil)
	en.encode(buf)
	return encryption.RawHash(buf.Bytes())
}

/*Clone - implement interface */
func (en *ExtensionNode) Clone() Node {
	clone := &ExtensionNode{}
	clone.OriginTrackerNode = NewOriginTrackerNode()
	clone.Path = en.Path       // path will never be updated inplace and so ok
	clone.NodeKey = en.NodeKey // nodekey will never be updated inplace and so ok
	return clone
}

/*GetNodeType - implement interface */
func (en *ExtensionNode) GetNodeType() byte {
	return NodeTypeExtensionNode
}

/*Encode - implement interface */
func (en *ExtensionNode) Encode() []byte {
	buf := bytes.NewBuffer(nil)
	writeNodePrefix(buf, en)
	en.encode(buf)
	return buf.Bytes()
}

func (en *ExtensionNode) encode(buf *bytes.Buffer) {
	buf.Write(en.Path)
	buf.WriteByte(Separator)
	buf.Write(en.NodeKey)
}

/*Decode - implement interface */
func (en *ExtensionNode) Decode(buf []byte) error {
	idx := bytes.IndexByte(buf, Separator)
	if idx < 0 {
		return ErrInvalidEncoding
	}
	en.Path = buf[:idx]
	buf = buf[idx+1:]
	en.NodeKey = buf
	return nil
}

/*GetValueNode - get the value node associated with this node*/
func GetValueNode(node Node) *ValueNode {
	if node == nil {
		return nil
	}
	switch nodeImpl := node.(type) {
	case *ValueNode:
		return nodeImpl
	case *LeafNode:
		return nodeImpl.Value
	case *FullNode:
		return nodeImpl.Value
	default:
		return nil
	}
}

/*GetSerializationPrefix - get the serialization prefix */
func GetSerializationPrefix(node Node) byte {
	switch node.(type) {
	case *ValueNode:
		return NodeTypeValueNode
	case *LeafNode:
		return NodeTypeLeafNode
	case *FullNode:
		return NodeTypeFullNode
	case *ExtensionNode:
		return NodeTypeExtensionNode
	default:
		panic("uknown node type")
	}
}

/*IncludesNodeType - checks if the given node type is one of the node types in the mask */
func IncludesNodeType(nodeTypes byte, nodeType byte) bool {
	return (nodeTypes & nodeType) == nodeType
}

/*CreateNode - create a node based on the serialization prefix */
func CreateNode(r io.Reader) (Node, error) {
	buf := []byte{0}
	n, err := r.Read(buf)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, ErrInvalidEncoding
	}
	code := buf[0]
	var node Node
	switch code & NodeTypesAll {
	case NodeTypeValueNode:
		node = NewValueNode()
	case NodeTypeLeafNode:
		node = NewLeafNode(nil, Sequence(0), nil)
	case NodeTypeFullNode:
		node = NewFullNode(nil)
	case NodeTypeExtensionNode:
		node = NewExtensionNode(nil, nil)
	default:
		panic(fmt.Sprintf("unkown node type: %v", code))
	}
	var ot OriginTracker
	ot.Read(r)
	node.SetOriginTracker(&ot)
	buf, err = ioutil.ReadAll(r)
	err = node.Decode(buf)
	return node, err
}

func writeNodePrefix(w io.Writer, node Node) error {
	_, err := w.Write([]byte{GetSerializationPrefix(node)})
	if err != nil {
		return err
	}
	node.GetOriginTracker().Write(w)
	return nil
}
