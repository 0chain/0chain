package storagesc

// Code generated by github.com/tinylib/msgp DO NOT EDIT.

import (
	"github.com/tinylib/msgp/msgp"
)

// MarshalMsg implements msgp.Marshaler
func (z *ReadPool) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 1
	// string "Pools"
	o = append(o, 0x81, 0xa5, 0x50, 0x6f, 0x6f, 0x6c, 0x73)
	o, err = z.Pools.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "Pools")
		return
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *ReadPool) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "Pools":
			bts, err = z.Pools.UnmarshalMsg(bts)
			if err != nil {
				err = msgp.WrapError(err, "Pools")
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
func (z *ReadPool) Msgsize() (s int) {
	s = 1 + 6 + z.Pools.Msgsize()
	return
}
