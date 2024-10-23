package chain

// Code generated by github.com/tinylib/msgp DO NOT EDIT.

import (
	"github.com/tinylib/msgp/msgp"
)

// MarshalMsg implements msgp.Marshaler
func (z LfbRound) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 3
	// string "r"
	o = append(o, 0x83, 0xa1, 0x72)
	o = msgp.AppendInt64(o, z.Round)
	// string "b"
	o = append(o, 0xa1, 0x62)
	o = msgp.AppendString(o, z.Hash)
	// string "mb_num"
	o = append(o, 0xa6, 0x6d, 0x62, 0x5f, 0x6e, 0x75, 0x6d)
	o = msgp.AppendInt64(o, z.MagicBlockNumber)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *LfbRound) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	_ = field
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		switch msgp.UnsafeString(field) {
		case "r":
			z.Round, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Round")
				return
			}
		case "b":
			z.Hash, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Hash")
				return
			}
		case "mb_num":
			z.MagicBlockNumber, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "MagicBlockNumber")
				return
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				err = msgp.WrapError(err)
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z LfbRound) Msgsize() (s int) {
	s = 1 + 2 + msgp.Int64Size + 2 + msgp.StringPrefixSize + len(z.Hash) + 7 + msgp.Int64Size
	return
}
