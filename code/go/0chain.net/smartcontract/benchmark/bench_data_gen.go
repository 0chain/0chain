package benchmark

// Code generated by github.com/tinylib/msgp DO NOT EDIT.

import (
	"github.com/tinylib/msgp/msgp"
)

// MarshalMsg implements msgp.Marshaler
func (z *BenchDataMpt) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 12
	// string "Clients"
	o = append(o, 0x8c, 0xa7, 0x43, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x73)
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
	// string "Miners"
	o = append(o, 0xa6, 0x4d, 0x69, 0x6e, 0x65, 0x72, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Miners)))
	for za0004 := range z.Miners {
		o = msgp.AppendString(o, z.Miners[za0004])
	}
	// string "Sharders"
	o = append(o, 0xa8, 0x53, 0x68, 0x61, 0x72, 0x64, 0x65, 0x72, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Sharders)))
	for za0005 := range z.Sharders {
		o = msgp.AppendString(o, z.Sharders[za0005])
	}
	// string "SharderKeys"
	o = append(o, 0xab, 0x53, 0x68, 0x61, 0x72, 0x64, 0x65, 0x72, 0x4b, 0x65, 0x79, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.SharderKeys)))
	for za0006 := range z.SharderKeys {
		o = msgp.AppendString(o, z.SharderKeys[za0006])
	}
	// string "ValidatorIds"
	o = append(o, 0xac, 0x56, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x6f, 0x72, 0x49, 0x64, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.ValidatorIds)))
	for za0007 := range z.ValidatorIds {
		o = msgp.AppendString(o, z.ValidatorIds[za0007])
	}
	// string "ValidatorPublicKeys"
	o = append(o, 0xb3, 0x56, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x6f, 0x72, 0x50, 0x75, 0x62, 0x6c, 0x69, 0x63, 0x4b, 0x65, 0x79, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.ValidatorPublicKeys)))
	for za0008 := range z.ValidatorPublicKeys {
		o = msgp.AppendString(o, z.ValidatorPublicKeys[za0008])
	}
	// string "ValidatorPrivateKeys"
	o = append(o, 0xb4, 0x56, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x6f, 0x72, 0x50, 0x72, 0x69, 0x76, 0x61, 0x74, 0x65, 0x4b, 0x65, 0x79, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.ValidatorPrivateKeys)))
	for za0009 := range z.ValidatorPrivateKeys {
		o = msgp.AppendString(o, z.ValidatorPrivateKeys[za0009])
	}
	// string "InactiveSharder"
	o = append(o, 0xaf, 0x49, 0x6e, 0x61, 0x63, 0x74, 0x69, 0x76, 0x65, 0x53, 0x68, 0x61, 0x72, 0x64, 0x65, 0x72)
	o = msgp.AppendString(o, z.InactiveSharder)
	// string "InactiveSharderPK"
	o = append(o, 0xb1, 0x49, 0x6e, 0x61, 0x63, 0x74, 0x69, 0x76, 0x65, 0x53, 0x68, 0x61, 0x72, 0x64, 0x65, 0x72, 0x50, 0x4b)
	o = msgp.AppendString(o, z.InactiveSharderPK)
	// string "Now"
	o = append(o, 0xa3, 0x4e, 0x6f, 0x77)
	o, err = z.Now.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "Now")
		return
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *BenchDataMpt) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "Clients":
			var zb0002 uint32
			zb0002, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Clients")
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
					err = msgp.WrapError(err, "Clients", za0001)
					return
				}
			}
		case "PublicKeys":
			var zb0003 uint32
			zb0003, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "PublicKeys")
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
					err = msgp.WrapError(err, "PublicKeys", za0002)
					return
				}
			}
		case "PrivateKeys":
			var zb0004 uint32
			zb0004, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "PrivateKeys")
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
					err = msgp.WrapError(err, "PrivateKeys", za0003)
					return
				}
			}
		case "Miners":
			var zb0005 uint32
			zb0005, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Miners")
				return
			}
			if cap(z.Miners) >= int(zb0005) {
				z.Miners = (z.Miners)[:zb0005]
			} else {
				z.Miners = make([]string, zb0005)
			}
			for za0004 := range z.Miners {
				z.Miners[za0004], bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					err = msgp.WrapError(err, "Miners", za0004)
					return
				}
			}
		case "Sharders":
			var zb0006 uint32
			zb0006, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Sharders")
				return
			}
			if cap(z.Sharders) >= int(zb0006) {
				z.Sharders = (z.Sharders)[:zb0006]
			} else {
				z.Sharders = make([]string, zb0006)
			}
			for za0005 := range z.Sharders {
				z.Sharders[za0005], bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					err = msgp.WrapError(err, "Sharders", za0005)
					return
				}
			}
		case "SharderKeys":
			var zb0007 uint32
			zb0007, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "SharderKeys")
				return
			}
			if cap(z.SharderKeys) >= int(zb0007) {
				z.SharderKeys = (z.SharderKeys)[:zb0007]
			} else {
				z.SharderKeys = make([]string, zb0007)
			}
			for za0006 := range z.SharderKeys {
				z.SharderKeys[za0006], bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					err = msgp.WrapError(err, "SharderKeys", za0006)
					return
				}
			}
		case "ValidatorIds":
			var zb0008 uint32
			zb0008, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "ValidatorIds")
				return
			}
			if cap(z.ValidatorIds) >= int(zb0008) {
				z.ValidatorIds = (z.ValidatorIds)[:zb0008]
			} else {
				z.ValidatorIds = make([]string, zb0008)
			}
			for za0007 := range z.ValidatorIds {
				z.ValidatorIds[za0007], bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					err = msgp.WrapError(err, "ValidatorIds", za0007)
					return
				}
			}
		case "ValidatorPublicKeys":
			var zb0009 uint32
			zb0009, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "ValidatorPublicKeys")
				return
			}
			if cap(z.ValidatorPublicKeys) >= int(zb0009) {
				z.ValidatorPublicKeys = (z.ValidatorPublicKeys)[:zb0009]
			} else {
				z.ValidatorPublicKeys = make([]string, zb0009)
			}
			for za0008 := range z.ValidatorPublicKeys {
				z.ValidatorPublicKeys[za0008], bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					err = msgp.WrapError(err, "ValidatorPublicKeys", za0008)
					return
				}
			}
		case "ValidatorPrivateKeys":
			var zb0010 uint32
			zb0010, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "ValidatorPrivateKeys")
				return
			}
			if cap(z.ValidatorPrivateKeys) >= int(zb0010) {
				z.ValidatorPrivateKeys = (z.ValidatorPrivateKeys)[:zb0010]
			} else {
				z.ValidatorPrivateKeys = make([]string, zb0010)
			}
			for za0009 := range z.ValidatorPrivateKeys {
				z.ValidatorPrivateKeys[za0009], bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					err = msgp.WrapError(err, "ValidatorPrivateKeys", za0009)
					return
				}
			}
		case "InactiveSharder":
			z.InactiveSharder, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "InactiveSharder")
				return
			}
		case "InactiveSharderPK":
			z.InactiveSharderPK, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "InactiveSharderPK")
				return
			}
		case "Now":
			bts, err = z.Now.UnmarshalMsg(bts)
			if err != nil {
				err = msgp.WrapError(err, "Now")
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
	s += 7 + msgp.ArrayHeaderSize
	for za0004 := range z.Miners {
		s += msgp.StringPrefixSize + len(z.Miners[za0004])
	}
	s += 9 + msgp.ArrayHeaderSize
	for za0005 := range z.Sharders {
		s += msgp.StringPrefixSize + len(z.Sharders[za0005])
	}
	s += 12 + msgp.ArrayHeaderSize
	for za0006 := range z.SharderKeys {
		s += msgp.StringPrefixSize + len(z.SharderKeys[za0006])
	}
	s += 13 + msgp.ArrayHeaderSize
	for za0007 := range z.ValidatorIds {
		s += msgp.StringPrefixSize + len(z.ValidatorIds[za0007])
	}
	s += 20 + msgp.ArrayHeaderSize
	for za0008 := range z.ValidatorPublicKeys {
		s += msgp.StringPrefixSize + len(z.ValidatorPublicKeys[za0008])
	}
	s += 21 + msgp.ArrayHeaderSize
	for za0009 := range z.ValidatorPrivateKeys {
		s += msgp.StringPrefixSize + len(z.ValidatorPrivateKeys[za0009])
	}
	s += 16 + msgp.StringPrefixSize + len(z.InactiveSharder) + 18 + msgp.StringPrefixSize + len(z.InactiveSharderPK) + 4 + z.Now.Msgsize()
	return
}
