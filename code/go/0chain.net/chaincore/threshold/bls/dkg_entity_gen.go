package bls

// THIS FILE WAS PRODUCED BY THE MSGP CODE GENERATION TOOL (github.com/dchenk/msgp).
// DO NOT EDIT.

import (
	"github.com/dchenk/msgp/msgp"
)

// MarshalMsg implements msgp.Marshaler
func (z *DKGKeyShare) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 4
	// string "IDField"
	o = append(o, 0x84, 0xa7, 0x49, 0x44, 0x46, 0x69, 0x65, 0x6c, 0x64)
	o, err = z.IDField.MarshalMsg(o)
	if err != nil {
		return
	}
	// string "Message"
	o = append(o, 0xa7, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65)
	o = msgp.AppendString(o, z.Message)
	// string "Share"
	o = append(o, 0xa5, 0x53, 0x68, 0x61, 0x72, 0x65)
	o = msgp.AppendString(o, z.Share)
	// string "Sign"
	o = append(o, 0xa4, 0x53, 0x69, 0x67, 0x6e)
	o = msgp.AppendString(o, z.Sign)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *DKGKeyShare) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "IDField":
			bts, err = z.IDField.UnmarshalMsg(bts)
			if err != nil {
				return
			}
		case "Message":
			z.Message, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "Share":
			z.Share, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				return
			}
		case "Sign":
			z.Sign, bts, err = msgp.ReadStringBytes(bts)
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
func (z *DKGKeyShare) Msgsize() (s int) {
	s = 1 + 8 + z.IDField.Msgsize() + 8 + msgp.StringPrefixSize + len(z.Message) + 6 + msgp.StringPrefixSize + len(z.Share) + 5 + msgp.StringPrefixSize + len(z.Sign)
	return
}
