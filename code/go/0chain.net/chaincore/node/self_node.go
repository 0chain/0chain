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

/*SelfNode -- self node type*/
type SelfNode struct {
	mx sync.RWMutex
	*Node
	signatureScheme encryption.SignatureScheme
}

func newSelfNode() (sn *SelfNode) {
	sn = new(SelfNode)
	sn.Node = new(Node)
	return
}

// Underlying returns underlying Node instance.
func (sn *SelfNode) Underlying() *Node {
	sn.mx.RLock()
	defer sn.mx.RUnlock()

	return sn.Node
}

/*SetSignatureScheme - getter */
func (sn *SelfNode) GetSignatureScheme() encryption.SignatureScheme {
	return sn.signatureScheme
}

/*SetSignatureScheme - setter */
func (sn *SelfNode) SetSignatureScheme(signatureScheme encryption.SignatureScheme) {
	sn.signatureScheme = signatureScheme
	sn.Underlying().SetPublicKey(signatureScheme.GetPublicKey())
}

/*Sign - sign the given hash */
func (sn *SelfNode) Sign(hash string) (string, error) {
	return sn.signatureScheme.Sign(hash)
}

/*TimeStampSignature - get timestamp based signature */
func (sn *SelfNode) TimeStampSignature() (string, string, string, error) {
	data := fmt.Sprintf("%v:%v", sn.Underlying().GetKey(), common.Now())
	hash := encryption.Hash(data)
	signature, err := sn.Sign(hash)
	if err != nil {
		return "", "", "", err
	}
	return data, hash, signature, err
}

/*ValidateSignatureTime - validate if the time stamp used in the signature is valid */
func (sn *SelfNode) ValidateSignatureTime(data string) (bool, error) {
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

// IsEqual returns true if given node pointer is equal to
// pointer to underlying Node.
func (sn *SelfNode) IsEqual(node *Node) bool {
	sn.mx.RLock()
	defer sn.mx.RUnlock()

	return sn.Node == node
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

/*Self represents the node of this instance */
var Self = newSelfNode()
