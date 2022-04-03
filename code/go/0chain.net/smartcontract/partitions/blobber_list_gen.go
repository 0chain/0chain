package partitions

// Code generated by github.com/tinylib/msgp DO NOT EDIT.

import (
	"github.com/tinylib/msgp/msgp"
)

// MarshalMsg implements msgp.Marshaler
func (z BlobberNode) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 2
	// string "ID"
	o = append(o, 0x82, 0xa2, 0x49, 0x44)
	o = msgp.AppendString(o, z.ID)
	// string "Url"
	o = append(o, 0xa3, 0x55, 0x72, 0x6c)
	o = msgp.AppendString(o, z.Url)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *BlobberNode) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "ID":
			z.ID, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "ID")
				return
			}
		case "Url":
			z.Url, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Url")
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
func (z BlobberNode) Msgsize() (s int) {
	s = 1 + 3 + msgp.StringPrefixSize + len(z.ID) + 4 + msgp.StringPrefixSize + len(z.Url)
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *blobberItemList) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 3
	// string "Key"
	o = append(o, 0x83, 0xa3, 0x4b, 0x65, 0x79)
	o = msgp.AppendString(o, z.Key)
	// string "Items"
	o = append(o, 0xa5, 0x49, 0x74, 0x65, 0x6d, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.Items)))
	for za0001 := range z.Items {
		// map header, size 2
		// string "ID"
		o = append(o, 0x82, 0xa2, 0x49, 0x44)
		o = msgp.AppendString(o, z.Items[za0001].ID)
		// string "Url"
		o = append(o, 0xa3, 0x55, 0x72, 0x6c)
		o = msgp.AppendString(o, z.Items[za0001].Url)
	}
	// string "Changed"
	o = append(o, 0xa7, 0x43, 0x68, 0x61, 0x6e, 0x67, 0x65, 0x64)
	o = msgp.AppendBool(o, z.Changed)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *blobberItemList) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "Key":
			z.Key, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Key")
				return
			}
		case "Items":
			var zb0002 uint32
			zb0002, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Items")
				return
			}
			if cap(z.Items) >= int(zb0002) {
				z.Items = (z.Items)[:zb0002]
			} else {
				z.Items = make([]BlobberNode, zb0002)
			}
			for za0001 := range z.Items {
				var zb0003 uint32
				zb0003, bts, err = msgp.ReadMapHeaderBytes(bts)
				if err != nil {
					err = msgp.WrapError(err, "Items", za0001)
					return
				}
				for zb0003 > 0 {
					zb0003--
					field, bts, err = msgp.ReadMapKeyZC(bts)
					if err != nil {
						err = msgp.WrapError(err, "Items", za0001)
						return
					}
					switch msgp.UnsafeString(field) {
					case "ID":
						z.Items[za0001].ID, bts, err = msgp.ReadStringBytes(bts)
						if err != nil {
							err = msgp.WrapError(err, "Items", za0001, "ID")
							return
						}
					case "Url":
						z.Items[za0001].Url, bts, err = msgp.ReadStringBytes(bts)
						if err != nil {
							err = msgp.WrapError(err, "Items", za0001, "Url")
							return
						}
					default:
						bts, err = msgp.Skip(bts)
						if err != nil {
							err = msgp.WrapError(err, "Items", za0001)
							return
						}
					}
				}
			}
		case "Changed":
			z.Changed, bts, err = msgp.ReadBoolBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Changed")
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
func (z *blobberItemList) Msgsize() (s int) {
	s = 1 + 4 + msgp.StringPrefixSize + len(z.Key) + 6 + msgp.ArrayHeaderSize
	for za0001 := range z.Items {
		s += 1 + 3 + msgp.StringPrefixSize + len(z.Items[za0001].ID) + 4 + msgp.StringPrefixSize + len(z.Items[za0001].Url)
	}
	s += 8 + msgp.BoolSize
	return
}
