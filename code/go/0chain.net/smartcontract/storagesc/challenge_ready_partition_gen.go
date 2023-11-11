package storagesc

// Code generated by github.com/tinylib/msgp DO NOT EDIT.

import (
	"github.com/tinylib/msgp/msgp"
)

// MarshalMsg implements msgp.Marshaler
func (z *ChallengeReadyBlobber) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 3
	// string "BlobberID"
	o = append(o, 0x83, 0xa9, 0x42, 0x6c, 0x6f, 0x62, 0x62, 0x65, 0x72, 0x49, 0x44)
	o = msgp.AppendString(o, z.BlobberID)
	// string "Stake"
	o = append(o, 0xa5, 0x53, 0x74, 0x61, 0x6b, 0x65)
	o, err = z.Stake.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "Stake")
		return
	}
	// string "UsedCapacity"
	o = append(o, 0xac, 0x55, 0x73, 0x65, 0x64, 0x43, 0x61, 0x70, 0x61, 0x63, 0x69, 0x74, 0x79)
	o = msgp.AppendUint64(o, z.UsedCapacity)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *ChallengeReadyBlobber) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "BlobberID":
			z.BlobberID, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "BlobberID")
				return
			}
		case "Stake":
			bts, err = z.Stake.UnmarshalMsg(bts)
			if err != nil {
				err = msgp.WrapError(err, "Stake")
				return
			}
		case "UsedCapacity":
			z.UsedCapacity, bts, err = msgp.ReadUint64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "UsedCapacity")
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
func (z *ChallengeReadyBlobber) Msgsize() (s int) {
	s = 1 + 10 + msgp.StringPrefixSize + len(z.BlobberID) + 6 + z.Stake.Msgsize() + 13 + msgp.Uint64Size
	return
}
