package control

// THIS FILE WAS PRODUCED BY THE MSGP CODE GENERATION TOOL (github.com/dchenk/msgp).
// DO NOT EDIT.

import (
	"github.com/dchenk/msgp/msgp"
)

// MarshalMsg implements msgp.Marshaler
func (z item) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 1
	// string "Field"
	o = append(o, 0x81, 0xa5, 0x46, 0x69, 0x65, 0x6c, 0x64)
	o = msgp.AppendInt64(o, z.Field)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *item) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch string(field) {
		case "Field":
			z.Field, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				return
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z item) Msgsize() (s int) {
	s = 1 + 6 + msgp.Int64Size
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *itemArray) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 1
	// string "Fields"
	o = append(o, 0x81, 0xa6, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Fields)))
	for za0001 := range z.Fields {
		o = msgp.AppendInt64(o, z.Fields[za0001])
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *itemArray) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var field []byte
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		return
	}
	for zb0001 > 0 {
		zb0001--
		field, bts, err = msgp.ReadMapKeyZC(bts)
		if err != nil {
			return
		}
		switch string(field) {
		case "Fields":
			var zb0002 uint32
			zb0002, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Fields) >= int(zb0002) {
				z.Fields = (z.Fields)[:zb0002]
			} else {
				z.Fields = make([]int64, zb0002)
			}
			for za0001 := range z.Fields {
				z.Fields[za0001], bts, err = msgp.ReadInt64Bytes(bts)
				if err != nil {
					return
				}
			}
		default:
			bts, err = msgp.Skip(bts)
			if err != nil {
				return
			}
		}
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *itemArray) Msgsize() (s int) {
	s = 1 + 7 + msgp.ArrayHeaderSize + (len(z.Fields) * (msgp.Int64Size))
	return
}
