package storagesc

// Code generated by github.com/tinylib/msgp DO NOT EDIT.

import (
	"github.com/tinylib/msgp/msgp"
)

// MarshalMsg implements msgp.Marshaler
func (z *readPool) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 1
	// string "Balance"
	o = append(o, 0x81, 0xa7, 0x42, 0x61, 0x6c, 0x61, 0x6e, 0x63, 0x65)
	o, err = z.Balance.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "Balance")
		return
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *readPool) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "Balance":
			bts, err = z.Balance.UnmarshalMsg(bts)
			if err != nil {
				err = msgp.WrapError(err, "Balance")
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
func (z *readPool) Msgsize() (s int) {
	s = 1 + 8 + z.Balance.Msgsize()
	return
}

// MarshalMsg implements msgp.Marshaler
func (z readPoolLockRequest) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 2
	// string "TargetId"
	o = append(o, 0x82, 0xa8, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x49, 0x64)
	o = msgp.AppendString(o, z.TargetId)
	// string "MintTokens"
	o = append(o, 0xaa, 0x4d, 0x69, 0x6e, 0x74, 0x54, 0x6f, 0x6b, 0x65, 0x6e, 0x73)
	o = msgp.AppendBool(o, z.MintTokens)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *readPoolLockRequest) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "TargetId":
			z.TargetId, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "TargetId")
				return
			}
		case "MintTokens":
			z.MintTokens, bts, err = msgp.ReadBoolBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "MintTokens")
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
func (z readPoolLockRequest) Msgsize() (s int) {
	s = 1 + 9 + msgp.StringPrefixSize + len(z.TargetId) + 11 + msgp.BoolSize
	return
}
