package node

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"0chain.net/core/build"
	"0chain.net/core/common"
	"0chain.net/core/encryption"
)

/*Self represents the node of this instance */
var Self = newSelfNode()

/*SelfNode -- self node type*/
type SelfNode struct {
	mx sync.RWMutex
	*Node
	signatureScheme encryption.SignatureScheme
}

func newSelfNode() *SelfNode {
	node := &SelfNode{
		Node: &Node{},
	}
	return node
}

// Underlying returns underlying Node instance.
func (sn *SelfNode) Underlying() *Node {
	sn.mx.RLock()
	defer sn.mx.RUnlock()

	return sn.Node
}

/*SetSignatureScheme - getter */
func (sn *SelfNode) GetSignatureScheme() encryption.SignatureScheme {
	sn.mx.RLock()
	defer sn.mx.RUnlock()
	return sn.signatureScheme
}

/*SetSignatureScheme - setter */
func (sn *SelfNode) SetSignatureScheme(signatureScheme encryption.SignatureScheme) {
	sn.mx.Lock()
	defer sn.mx.Unlock()
	sn.signatureScheme = signatureScheme
	sn.Node.SetPublicKey(signatureScheme.GetPublicKey())
}

/*Sign - sign the given hash */
func (sn *SelfNode) Sign(hash string) (string, error) {
	sn.mx.RLock()
	defer sn.mx.RUnlock()
	return sn.signatureScheme.Sign(hash)
}

/*TimeStampSignature - get timestamp based signature */
func (sn *SelfNode) TimeStampSignature() (string, string, string, error) {
	sn.mx.RLock()
	defer sn.mx.RUnlock()
	data := fmt.Sprintf("%v:%v", sn.Node.GetKey(), common.Now())
	hash := encryption.Hash(data)
	signature, err := sn.signatureScheme.Sign(hash)
	if err != nil {
		return "", "", "", err
	}
	return data, hash, signature, err
}

/*ValidateSignatureTime - validate if the time stamp used in the signature is valid */
func ValidateSignatureTime(data string) (bool, error) {
	segs := strings.Split(data, ":")
	if len(segs) < 2 {
		return false, errors.New("invalid data")
	}
	ts, err := strconv.ParseInt(segs[1], 10, 64)
	if err != nil {
		return false, err
	}
	if !common.Within(ts, 3) {
		return false, errors.New("timestamp not within tolerance")
	}
	return true, nil
}

// IsEqual returns true if given node ID is equal to
// ID of underlying Node.
func (sn *SelfNode) IsEqual(node *Node) bool {
	sn.mx.RLock()
	defer sn.mx.RUnlock()

	if node == nil || sn.Node == nil {
		return false
	}

	return sn.Node.ID == node.ID
}

func (sn *SelfNode) SetNodeIfPublicKeyIsEqual(node *Node) {
	sn.mx.Lock()
	defer sn.mx.Unlock()

	if sn.Node.PublicKey != node.PublicKey {
		return
	}

	sn.Node = node
	sn.Node.Info.StateMissingNodes = -1
	sn.Node.Info.BuildTag = build.BuildTag
	sn.Node.Status = NodeStatusActive
}
