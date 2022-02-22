package state

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"

	"0chain.net/core/encryption"
	"0chain.net/core/util"
)

//msgp:ignore State

//go:generate msgp -io=false -tests=false -v
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
	// if s.TxnHashBytes are not set, the State can't be deserialized later
	if s.TxnHashBytes == nil {
		panic(errors.New("State isn't properly initialized"))
	}
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
		return errors.New("invalid state")
	}
	binary.Read(buf, binary.LittleEndian, &origin)
	binary.Read(buf, binary.LittleEndian, &balance)
	s.Round = origin
	s.Balance = Balance(balance)
	return nil
}

func (s *State) MarshalMsg([]byte) ([]byte, error) {
	return s.Encode(), nil
}

func (s *State) UnmarshalMsg(data []byte) ([]byte, error) {
	err := s.Decode(data)
	return nil, err
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
