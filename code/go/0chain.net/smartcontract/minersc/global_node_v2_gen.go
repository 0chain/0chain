package minersc

// Code generated by github.com/tinylib/msgp DO NOT EDIT.

import (
	"github.com/tinylib/msgp/msgp"
)

// MarshalMsg implements msgp.Marshaler
func (z *globalNodeV2) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 4
	// string "globalNodeV1"
	o = append(o, 0x84, 0xac, 0x67, 0x6c, 0x6f, 0x62, 0x61, 0x6c, 0x4e, 0x6f, 0x64, 0x65, 0x56, 0x31)
	o, err = z.globalNodeV1.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "globalNodeV1")
		return
	}
	// string "version"
	o = append(o, 0xa7, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e)
	o = msgp.AppendString(o, z.Version)
	// string "name"
	o = append(o, 0xa4, 0x6e, 0x61, 0x6d, 0x65)
	o = msgp.AppendString(o, z.Name)
	// string "vc_phase_rounds"
	o = append(o, 0xaf, 0x76, 0x63, 0x5f, 0x70, 0x68, 0x61, 0x73, 0x65, 0x5f, 0x72, 0x6f, 0x75, 0x6e, 0x64, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.VCPhaseRounds)))
	for za0001 := range z.VCPhaseRounds {
		o = msgp.AppendInt(o, z.VCPhaseRounds[za0001])
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *globalNodeV2) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "globalNodeV1":
			bts, err = z.globalNodeV1.UnmarshalMsg(bts)
			if err != nil {
				err = msgp.WrapError(err, "globalNodeV1")
				return
			}
		case "version":
			z.Version, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Version")
				return
			}
		case "name":
			z.Name, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Name")
				return
			}
		case "vc_phase_rounds":
			var zb0002 uint32
			zb0002, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "VCPhaseRounds")
				return
			}
			if cap(z.VCPhaseRounds) >= int(zb0002) {
				z.VCPhaseRounds = (z.VCPhaseRounds)[:zb0002]
			} else {
				z.VCPhaseRounds = make([]int, zb0002)
			}
			for za0001 := range z.VCPhaseRounds {
				z.VCPhaseRounds[za0001], bts, err = msgp.ReadIntBytes(bts)
				if err != nil {
					err = msgp.WrapError(err, "VCPhaseRounds", za0001)
					return
				}
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
func (z *globalNodeV2) Msgsize() (s int) {
	s = 1 + 13 + z.globalNodeV1.Msgsize() + 8 + msgp.StringPrefixSize + len(z.Version) + 5 + msgp.StringPrefixSize + len(z.Name) + 16 + msgp.ArrayHeaderSize + (len(z.VCPhaseRounds) * (msgp.IntSize))
	return
}