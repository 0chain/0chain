package state

import (
	"0chain.net/core/encryption"
	. "0chain.net/core/logging"
	"0chain.net/core/util"
	"bytes"
	"encoding/binary"
	"encoding/hex"
)

//Balance - any quantity that is represented as an integer in the lowest denomination
type Balance int64

//State - state that needs consensus within the blockchain.
type State struct {
	/* Note: origin is way to parallelize state pruning with state saving. That is, when a leaf node is deleted and added later, but the pruning logic of
	marking the nodes by origin is complete and before the sweeping the nodes to delete, the same leaf node comes back, it gets deleted. However, by
	having the origin (round in the blockchain) part of the state ensures that the same logical leaf has a new hash and avoid this issue. We are getting
	parallelism without explicit locks with this approach.
	*/
	TxnHash      string  `json:"txn" msgpack:"-"`
	TxnHashBytes []byte  `json:"-" msgpack:"t"`
	Round        int64   `json:"round" msgpack:"r"`
	Balance      Balance `json:"balance" msgpack:"b"`
}

/*GetHash - implement SecureSerializableValueI interface */
func (s *State) GetHash() string {
	return util.ToHex(s.GetHashBytes())
}

/*GetHashBytes - implement SecureSerializableValueI interface */
func (s *State) GetHashBytes() []byte {
	return encryption.RawHash(s.Encode())
}

/*Encode - implement SecureSerializableValueI interface */
func (s *State) Encode() []byte {
	buf := bytes.NewBuffer(nil)
	buf.Write(s.TxnHashBytes)
	binary.Write(buf, binary.LittleEndian, s.Round)
	binary.Write(buf, binary.LittleEndian, s.Balance)
	return buf.Bytes()
}

/*Decode - implement SecureSerializableValueI interface */
func (s *State) Decode(data []byte) error {
	buf := bytes.NewBuffer(data)
	var origin int64
	var balance Balance
	s.TxnHashBytes = make([]byte, 32)
	if n, err := buf.Read(s.TxnHashBytes); err != nil || n != 32 {
		Logger.Error("invalid state")
	}
	binary.Read(buf, binary.LittleEndian, &origin)
	binary.Read(buf, binary.LittleEndian, &balance)
	s.Round = origin
	s.Balance = Balance(balance)
	return nil
}

//ComputeProperties - logic to compute derived properties
func (s *State) ComputeProperties() {
	s.TxnHash = hex.EncodeToString(s.TxnHashBytes)
}

/*SetRound - set the round for this state to make it unique if the same logical state is arrived again in a different round */
func (s *State) SetRound(round int64) {
	s.Round = round
}

//SetTxnHash - set the hash of the txn that's modifying this state
func (s *State) SetTxnHash(txnHash string) error {
	hashBytes, err := hex.DecodeString(txnHash)
	if err != nil {
		return err
	}
	s.TxnHash = txnHash
	s.TxnHashBytes = hashBytes
	return nil
}

//Deserializer - a deserializer to convert raw serialized data to a state object
type Deserializer struct {
}

//Deserialize - implement interface
func (bd *Deserializer) Deserialize(sv util.Serializable) util.Serializable {
	s := &State{}
	s.Decode(sv.Encode())
	return s
}
