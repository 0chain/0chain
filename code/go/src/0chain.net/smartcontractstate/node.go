package smartcontractstate

import (
	"bytes"
	"io"
	"io/ioutil"

	"0chain.net/common"
	"0chain.net/encryption"
	"0chain.net/util"
)

//Separator - used to separate fields when creating data array to hash
const Separator = ':'

//ErrInvalidEncoding - error to indicate invalid encoding
var ErrInvalidEncoding = common.NewError("invalid_node_encoding", "invalid node encoding")

/*Node - a node interface */
type Node interface {
	Clear()
	util.SecureSerializableValueI
}

/*ValueNode - any node that holds a value should implement this */
type ValueNode struct {
	Value util.Serializable
}

/*Clear - implement interface */
func (vn *ValueNode) Clear() {
	vn.Value = nil
}

/*GetHash - implements SecureSerializableValue interface */
func (vn *ValueNode) GetHash() string {
	return util.ToHex(vn.GetHashBytes())
}

/*GetHashBytes - implement SecureSerializableValue interface */
func (vn *ValueNode) GetHashBytes() []byte {
	if vn.Value == nil {
		return nil
	}
	return encryption.RawHash(vn.Value.Encode())
}

/*GetValue - get the value store in this node */
func (vn *ValueNode) GetValue() util.Serializable {
	return vn.Value
}

/*SetValue - set the value stored in this node */
func (vn *ValueNode) SetValue(value util.Serializable) {
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
	if vn.HasValue() {
		buf.Write(vn.GetValue().Encode())
	}
	return buf.Bytes()
}

/*Decode - overwrite interface method */
func (vn *ValueNode) Decode(buf []byte) error {
	pspv := &util.SecureSerializableValue{}
	err := pspv.Decode(buf)
	if err != nil {
		return err
	}
	vn.SetValue(pspv)
	return nil
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
	var node Node
	node = &ValueNode{}
	buf, err = ioutil.ReadAll(r)
	err = node.Decode(buf)
	return node, err
}
