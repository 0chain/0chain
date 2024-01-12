package state

// Code generated by github.com/tinylib/msgp DO NOT EDIT.

import (
	"github.com/tinylib/msgp/msgp"
)

// MarshalMsg implements msgp.Marshaler
func (z HardFork) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 2
	// string "name"
	o = append(o, 0x82, 0xa4, 0x6e, 0x61, 0x6d, 0x65)
	o = msgp.AppendString(o, z.name)
	// string "round"
	o = append(o, 0xa5, 0x72, 0x6f, 0x75, 0x6e, 0x64)
	o = msgp.AppendInt64(o, z.round)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *HardFork) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "name":
			z.name, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "name")
				return
			}
		case "round":
			z.round, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "round")
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
func (z HardFork) Msgsize() (s int) {
	s = 1 + 5 + msgp.StringPrefixSize + len(z.name) + 6 + msgp.Int64Size
	return
}