package faucetsc

// Code generated by github.com/tinylib/msgp DO NOT EDIT.

import (
	"github.com/tinylib/msgp/msgp"
)

// MarshalMsg implements msgp.Marshaler
func (z *FaucetConfig) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 8
	// string "PourAmount"
	o = append(o, 0x88, 0xaa, 0x50, 0x6f, 0x75, 0x72, 0x41, 0x6d, 0x6f, 0x75, 0x6e, 0x74)
	o, err = z.PourAmount.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "PourAmount")
		return
	}
	// string "MaxPourAmount"
	o = append(o, 0xad, 0x4d, 0x61, 0x78, 0x50, 0x6f, 0x75, 0x72, 0x41, 0x6d, 0x6f, 0x75, 0x6e, 0x74)
	o, err = z.MaxPourAmount.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "MaxPourAmount")
		return
	}
	// string "PeriodicLimit"
	o = append(o, 0xad, 0x50, 0x65, 0x72, 0x69, 0x6f, 0x64, 0x69, 0x63, 0x4c, 0x69, 0x6d, 0x69, 0x74)
	o, err = z.PeriodicLimit.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "PeriodicLimit")
		return
	}
	// string "GlobalLimit"
	o = append(o, 0xab, 0x47, 0x6c, 0x6f, 0x62, 0x61, 0x6c, 0x4c, 0x69, 0x6d, 0x69, 0x74)
	o, err = z.GlobalLimit.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "GlobalLimit")
		return
	}
	// string "IndividualReset"
	o = append(o, 0xaf, 0x49, 0x6e, 0x64, 0x69, 0x76, 0x69, 0x64, 0x75, 0x61, 0x6c, 0x52, 0x65, 0x73, 0x65, 0x74)
	o = msgp.AppendDuration(o, z.IndividualReset)
	// string "GlobalReset"
	o = append(o, 0xab, 0x47, 0x6c, 0x6f, 0x62, 0x61, 0x6c, 0x52, 0x65, 0x73, 0x65, 0x74)
	o = msgp.AppendDuration(o, z.GlobalReset)
	// string "OwnerId"
	o = append(o, 0xa7, 0x4f, 0x77, 0x6e, 0x65, 0x72, 0x49, 0x64)
	o = msgp.AppendString(o, z.OwnerId)
	// string "Cost"
	o = append(o, 0xa4, 0x43, 0x6f, 0x73, 0x74)
	o = msgp.AppendMapHeader(o, uint32(len(z.Cost)))
	keys_za0001 := make([]string, 0, len(z.Cost))
	for k := range z.Cost {
		keys_za0001 = append(keys_za0001, k)
	}
	msgp.Sort(keys_za0001)
	for _, k := range keys_za0001 {
		za0002 := z.Cost[k]
		o = msgp.AppendString(o, k)
		o = msgp.AppendInt(o, za0002)
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *FaucetConfig) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "PourAmount":
			bts, err = z.PourAmount.UnmarshalMsg(bts)
			if err != nil {
				err = msgp.WrapError(err, "PourAmount")
				return
			}
		case "MaxPourAmount":
			bts, err = z.MaxPourAmount.UnmarshalMsg(bts)
			if err != nil {
				err = msgp.WrapError(err, "MaxPourAmount")
				return
			}
		case "PeriodicLimit":
			bts, err = z.PeriodicLimit.UnmarshalMsg(bts)
			if err != nil {
				err = msgp.WrapError(err, "PeriodicLimit")
				return
			}
		case "GlobalLimit":
			bts, err = z.GlobalLimit.UnmarshalMsg(bts)
			if err != nil {
				err = msgp.WrapError(err, "GlobalLimit")
				return
			}
		case "IndividualReset":
			z.IndividualReset, bts, err = msgp.ReadDurationBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "IndividualReset")
				return
			}
		case "GlobalReset":
			z.GlobalReset, bts, err = msgp.ReadDurationBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "GlobalReset")
				return
			}
		case "OwnerId":
			z.OwnerId, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "OwnerId")
				return
			}
		case "Cost":
			var zb0002 uint32
			zb0002, bts, err = msgp.ReadMapHeaderBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Cost")
				return
			}
			if z.Cost == nil {
				z.Cost = make(map[string]int, zb0002)
			} else if len(z.Cost) > 0 {
				for key := range z.Cost {
					delete(z.Cost, key)
				}
			}
			for zb0002 > 0 {
				var za0001 string
				var za0002 int
				zb0002--
				za0001, bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					err = msgp.WrapError(err, "Cost")
					return
				}
				za0002, bts, err = msgp.ReadIntBytes(bts)
				if err != nil {
					err = msgp.WrapError(err, "Cost", za0001)
					return
				}
				z.Cost[za0001] = za0002
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
func (z *FaucetConfig) Msgsize() (s int) {
	s = 1 + 11 + z.PourAmount.Msgsize() + 14 + z.MaxPourAmount.Msgsize() + 14 + z.PeriodicLimit.Msgsize() + 12 + z.GlobalLimit.Msgsize() + 16 + msgp.DurationSize + 12 + msgp.DurationSize + 8 + msgp.StringPrefixSize + len(z.OwnerId) + 5 + msgp.MapHeaderSize
	if z.Cost != nil {
		for za0001, za0002 := range z.Cost {
			_ = za0002
			s += msgp.StringPrefixSize + len(za0001) + msgp.IntSize
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z Setting) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	o = msgp.AppendInt(o, int(z))
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Setting) UnmarshalMsg(bts []byte) (o []byte, err error) {
	{
		var zb0001 int
		zb0001, bts, err = msgp.ReadIntBytes(bts)
		if err != nil {
			err = msgp.WrapError(err)
			return
		}
		(*z) = Setting(zb0001)
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z Setting) Msgsize() (s int) {
	s = msgp.IntSize
	return
}
