package benchmark

// THIS FILE WAS PRODUCED BY THE MSGP CODE GENERATION TOOL (github.com/dchenk/msgp).
// DO NOT EDIT.

import (
	"github.com/dchenk/msgp/msgp"
)

// MarshalMsg implements msgp.Marshaler
func (z *BenchDataMpt) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 4
	// string "Clients"
	o = append(o, 0x84, 0xa7, 0x43, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Clients)))
	for za0001 := range z.Clients {
		o = msgp.AppendString(o, z.Clients[za0001])
	}
	// string "PublicKeys"
	o = append(o, 0xaa, 0x50, 0x75, 0x62, 0x6c, 0x69, 0x63, 0x4b, 0x65, 0x79, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.PublicKeys)))
	for za0002 := range z.PublicKeys {
		o = msgp.AppendString(o, z.PublicKeys[za0002])
	}
	// string "PrivateKeys"
	o = append(o, 0xab, 0x50, 0x72, 0x69, 0x76, 0x61, 0x74, 0x65, 0x4b, 0x65, 0x79, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.PrivateKeys)))
	for za0003 := range z.PrivateKeys {
		o = msgp.AppendString(o, z.PrivateKeys[za0003])
	}
	// string "Sharders"
	o = append(o, 0xa8, 0x53, 0x68, 0x61, 0x72, 0x64, 0x65, 0x72, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Sharders)))
	for za0004 := range z.Sharders {
		o = msgp.AppendString(o, z.Sharders[za0004])
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *BenchDataMpt) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "Clients":
			var zb0002 uint32
			zb0002, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Clients) >= int(zb0002) {
				z.Clients = (z.Clients)[:zb0002]
			} else {
				z.Clients = make([]string, zb0002)
			}
			for za0001 := range z.Clients {
				z.Clients[za0001], bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					return
				}
			}
		case "PublicKeys":
			var zb0003 uint32
			zb0003, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.PublicKeys) >= int(zb0003) {
				z.PublicKeys = (z.PublicKeys)[:zb0003]
			} else {
				z.PublicKeys = make([]string, zb0003)
			}
			for za0002 := range z.PublicKeys {
				z.PublicKeys[za0002], bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					return
				}
			}
		case "PrivateKeys":
			var zb0004 uint32
			zb0004, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.PrivateKeys) >= int(zb0004) {
				z.PrivateKeys = (z.PrivateKeys)[:zb0004]
			} else {
				z.PrivateKeys = make([]string, zb0004)
			}
			for za0003 := range z.PrivateKeys {
				z.PrivateKeys[za0003], bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					return
				}
			}
		case "Sharders":
			var zb0005 uint32
			zb0005, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				return
			}
			if cap(z.Sharders) >= int(zb0005) {
				z.Sharders = (z.Sharders)[:zb0005]
			} else {
				z.Sharders = make([]string, zb0005)
			}
			for za0004 := range z.Sharders {
				z.Sharders[za0004], bts, err = msgp.ReadStringBytes(bts)
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
func (z *BenchDataMpt) Msgsize() (s int) {
	s = 1 + 8 + msgp.ArrayHeaderSize
	for za0001 := range z.Clients {
		s += msgp.StringPrefixSize + len(z.Clients[za0001])
	}
	s += 11 + msgp.ArrayHeaderSize
	for za0002 := range z.PublicKeys {
		s += msgp.StringPrefixSize + len(z.PublicKeys[za0002])
	}
	s += 12 + msgp.ArrayHeaderSize
	for za0003 := range z.PrivateKeys {
		s += msgp.StringPrefixSize + len(z.PrivateKeys[za0003])
	}
	s += 9 + msgp.ArrayHeaderSize
	for za0004 := range z.Sharders {
		s += msgp.StringPrefixSize + len(z.Sharders[za0004])
	}
	return
}
