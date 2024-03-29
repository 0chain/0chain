package multisigsc

// Code generated by github.com/tinylib/msgp DO NOT EDIT.

import (
	"github.com/tinylib/msgp/msgp"
)

// MarshalMsg implements msgp.Marshaler
func (z *Wallet) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 6
	// string "ClientID"
	o = append(o, 0x86, 0xa8, 0x43, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x49, 0x44)
	o = msgp.AppendString(o, z.ClientID)
	// string "SignatureScheme"
	o = append(o, 0xaf, 0x53, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x53, 0x63, 0x68, 0x65, 0x6d, 0x65)
	o = msgp.AppendString(o, z.SignatureScheme)
	// string "PublicKey"
	o = append(o, 0xa9, 0x50, 0x75, 0x62, 0x6c, 0x69, 0x63, 0x4b, 0x65, 0x79)
	o = msgp.AppendString(o, z.PublicKey)
	// string "SignerThresholdIDs"
	o = append(o, 0xb2, 0x53, 0x69, 0x67, 0x6e, 0x65, 0x72, 0x54, 0x68, 0x72, 0x65, 0x73, 0x68, 0x6f, 0x6c, 0x64, 0x49, 0x44, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.SignerThresholdIDs)))
	for za0001 := range z.SignerThresholdIDs {
		o = msgp.AppendString(o, z.SignerThresholdIDs[za0001])
	}
	// string "SignerPublicKeys"
	o = append(o, 0xb0, 0x53, 0x69, 0x67, 0x6e, 0x65, 0x72, 0x50, 0x75, 0x62, 0x6c, 0x69, 0x63, 0x4b, 0x65, 0x79, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.SignerPublicKeys)))
	for za0002 := range z.SignerPublicKeys {
		o = msgp.AppendString(o, z.SignerPublicKeys[za0002])
	}
	// string "NumRequired"
	o = append(o, 0xab, 0x4e, 0x75, 0x6d, 0x52, 0x65, 0x71, 0x75, 0x69, 0x72, 0x65, 0x64)
	o = msgp.AppendInt(o, z.NumRequired)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *Wallet) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "ClientID":
			z.ClientID, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "ClientID")
				return
			}
		case "SignatureScheme":
			z.SignatureScheme, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "SignatureScheme")
				return
			}
		case "PublicKey":
			z.PublicKey, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "PublicKey")
				return
			}
		case "SignerThresholdIDs":
			var zb0002 uint32
			zb0002, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "SignerThresholdIDs")
				return
			}
			if cap(z.SignerThresholdIDs) >= int(zb0002) {
				z.SignerThresholdIDs = (z.SignerThresholdIDs)[:zb0002]
			} else {
				z.SignerThresholdIDs = make([]string, zb0002)
			}
			for za0001 := range z.SignerThresholdIDs {
				z.SignerThresholdIDs[za0001], bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					err = msgp.WrapError(err, "SignerThresholdIDs", za0001)
					return
				}
			}
		case "SignerPublicKeys":
			var zb0003 uint32
			zb0003, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "SignerPublicKeys")
				return
			}
			if cap(z.SignerPublicKeys) >= int(zb0003) {
				z.SignerPublicKeys = (z.SignerPublicKeys)[:zb0003]
			} else {
				z.SignerPublicKeys = make([]string, zb0003)
			}
			for za0002 := range z.SignerPublicKeys {
				z.SignerPublicKeys[za0002], bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					err = msgp.WrapError(err, "SignerPublicKeys", za0002)
					return
				}
			}
		case "NumRequired":
			z.NumRequired, bts, err = msgp.ReadIntBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "NumRequired")
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
func (z *Wallet) Msgsize() (s int) {
	s = 1 + 9 + msgp.StringPrefixSize + len(z.ClientID) + 16 + msgp.StringPrefixSize + len(z.SignatureScheme) + 10 + msgp.StringPrefixSize + len(z.PublicKey) + 19 + msgp.ArrayHeaderSize
	for za0001 := range z.SignerThresholdIDs {
		s += msgp.StringPrefixSize + len(z.SignerThresholdIDs[za0001])
	}
	s += 17 + msgp.ArrayHeaderSize
	for za0002 := range z.SignerPublicKeys {
		s += msgp.StringPrefixSize + len(z.SignerPublicKeys[za0002])
	}
	s += 12 + msgp.IntSize
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *expirationQueue) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 2
	// string "Head"
	o = append(o, 0x82, 0xa4, 0x48, 0x65, 0x61, 0x64)
	// map header, size 2
	// string "ClientID"
	o = append(o, 0x82, 0xa8, 0x43, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x49, 0x44)
	o = msgp.AppendString(o, z.Head.ClientID)
	// string "ProposalID"
	o = append(o, 0xaa, 0x50, 0x72, 0x6f, 0x70, 0x6f, 0x73, 0x61, 0x6c, 0x49, 0x44)
	o = msgp.AppendString(o, z.Head.ProposalID)
	// string "Tail"
	o = append(o, 0xa4, 0x54, 0x61, 0x69, 0x6c)
	// map header, size 2
	// string "ClientID"
	o = append(o, 0x82, 0xa8, 0x43, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x49, 0x44)
	o = msgp.AppendString(o, z.Tail.ClientID)
	// string "ProposalID"
	o = append(o, 0xaa, 0x50, 0x72, 0x6f, 0x70, 0x6f, 0x73, 0x61, 0x6c, 0x49, 0x44)
	o = msgp.AppendString(o, z.Tail.ProposalID)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *expirationQueue) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "Head":
			var zb0002 uint32
			zb0002, bts, err = msgp.ReadMapHeaderBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Head")
				return
			}
			for zb0002 > 0 {
				zb0002--
				field, bts, err = msgp.ReadMapKeyZC(bts)
				if err != nil {
					err = msgp.WrapError(err, "Head")
					return
				}
				switch msgp.UnsafeString(field) {
				case "ClientID":
					z.Head.ClientID, bts, err = msgp.ReadStringBytes(bts)
					if err != nil {
						err = msgp.WrapError(err, "Head", "ClientID")
						return
					}
				case "ProposalID":
					z.Head.ProposalID, bts, err = msgp.ReadStringBytes(bts)
					if err != nil {
						err = msgp.WrapError(err, "Head", "ProposalID")
						return
					}
				default:
					bts, err = msgp.Skip(bts)
					if err != nil {
						err = msgp.WrapError(err, "Head")
						return
					}
				}
			}
		case "Tail":
			var zb0003 uint32
			zb0003, bts, err = msgp.ReadMapHeaderBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Tail")
				return
			}
			for zb0003 > 0 {
				zb0003--
				field, bts, err = msgp.ReadMapKeyZC(bts)
				if err != nil {
					err = msgp.WrapError(err, "Tail")
					return
				}
				switch msgp.UnsafeString(field) {
				case "ClientID":
					z.Tail.ClientID, bts, err = msgp.ReadStringBytes(bts)
					if err != nil {
						err = msgp.WrapError(err, "Tail", "ClientID")
						return
					}
				case "ProposalID":
					z.Tail.ProposalID, bts, err = msgp.ReadStringBytes(bts)
					if err != nil {
						err = msgp.WrapError(err, "Tail", "ProposalID")
						return
					}
				default:
					bts, err = msgp.Skip(bts)
					if err != nil {
						err = msgp.WrapError(err, "Tail")
						return
					}
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
func (z *expirationQueue) Msgsize() (s int) {
	s = 1 + 5 + 1 + 9 + msgp.StringPrefixSize + len(z.Head.ClientID) + 11 + msgp.StringPrefixSize + len(z.Head.ProposalID) + 5 + 1 + 9 + msgp.StringPrefixSize + len(z.Tail.ClientID) + 11 + msgp.StringPrefixSize + len(z.Tail.ProposalID)
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *proposal) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 9
	// string "ProposalID"
	o = append(o, 0x89, 0xaa, 0x50, 0x72, 0x6f, 0x70, 0x6f, 0x73, 0x61, 0x6c, 0x49, 0x44)
	o = msgp.AppendString(o, z.ProposalID)
	// string "ExpirationDate"
	o = append(o, 0xae, 0x45, 0x78, 0x70, 0x69, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x44, 0x61, 0x74, 0x65)
	o, err = z.ExpirationDate.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "ExpirationDate")
		return
	}
	// string "Next"
	o = append(o, 0xa4, 0x4e, 0x65, 0x78, 0x74)
	// map header, size 2
	// string "ClientID"
	o = append(o, 0x82, 0xa8, 0x43, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x49, 0x44)
	o = msgp.AppendString(o, z.Next.ClientID)
	// string "ProposalID"
	o = append(o, 0xaa, 0x50, 0x72, 0x6f, 0x70, 0x6f, 0x73, 0x61, 0x6c, 0x49, 0x44)
	o = msgp.AppendString(o, z.Next.ProposalID)
	// string "Prev"
	o = append(o, 0xa4, 0x50, 0x72, 0x65, 0x76)
	// map header, size 2
	// string "ClientID"
	o = append(o, 0x82, 0xa8, 0x43, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x49, 0x44)
	o = msgp.AppendString(o, z.Prev.ClientID)
	// string "ProposalID"
	o = append(o, 0xaa, 0x50, 0x72, 0x6f, 0x70, 0x6f, 0x73, 0x61, 0x6c, 0x49, 0x44)
	o = msgp.AppendString(o, z.Prev.ProposalID)
	// string "Transfer"
	o = append(o, 0xa8, 0x54, 0x72, 0x61, 0x6e, 0x73, 0x66, 0x65, 0x72)
	o, err = z.Transfer.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "Transfer")
		return
	}
	// string "SignerThresholdIDs"
	o = append(o, 0xb2, 0x53, 0x69, 0x67, 0x6e, 0x65, 0x72, 0x54, 0x68, 0x72, 0x65, 0x73, 0x68, 0x6f, 0x6c, 0x64, 0x49, 0x44, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.SignerThresholdIDs)))
	for za0001 := range z.SignerThresholdIDs {
		o = msgp.AppendString(o, z.SignerThresholdIDs[za0001])
	}
	// string "SignerSignatures"
	o = append(o, 0xb0, 0x53, 0x69, 0x67, 0x6e, 0x65, 0x72, 0x53, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x73)
	o = msgp.AppendArrayHeader(o, uint32(len(z.SignerSignatures)))
	for za0002 := range z.SignerSignatures {
		o = msgp.AppendString(o, z.SignerSignatures[za0002])
	}
	// string "ClientSignature"
	o = append(o, 0xaf, 0x43, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x53, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65)
	o = msgp.AppendString(o, z.ClientSignature)
	// string "ExecutedInTxnHash"
	o = append(o, 0xb1, 0x45, 0x78, 0x65, 0x63, 0x75, 0x74, 0x65, 0x64, 0x49, 0x6e, 0x54, 0x78, 0x6e, 0x48, 0x61, 0x73, 0x68)
	o = msgp.AppendString(o, z.ExecutedInTxnHash)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *proposal) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "ProposalID":
			z.ProposalID, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "ProposalID")
				return
			}
		case "ExpirationDate":
			bts, err = z.ExpirationDate.UnmarshalMsg(bts)
			if err != nil {
				err = msgp.WrapError(err, "ExpirationDate")
				return
			}
		case "Next":
			var zb0002 uint32
			zb0002, bts, err = msgp.ReadMapHeaderBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Next")
				return
			}
			for zb0002 > 0 {
				zb0002--
				field, bts, err = msgp.ReadMapKeyZC(bts)
				if err != nil {
					err = msgp.WrapError(err, "Next")
					return
				}
				switch msgp.UnsafeString(field) {
				case "ClientID":
					z.Next.ClientID, bts, err = msgp.ReadStringBytes(bts)
					if err != nil {
						err = msgp.WrapError(err, "Next", "ClientID")
						return
					}
				case "ProposalID":
					z.Next.ProposalID, bts, err = msgp.ReadStringBytes(bts)
					if err != nil {
						err = msgp.WrapError(err, "Next", "ProposalID")
						return
					}
				default:
					bts, err = msgp.Skip(bts)
					if err != nil {
						err = msgp.WrapError(err, "Next")
						return
					}
				}
			}
		case "Prev":
			var zb0003 uint32
			zb0003, bts, err = msgp.ReadMapHeaderBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Prev")
				return
			}
			for zb0003 > 0 {
				zb0003--
				field, bts, err = msgp.ReadMapKeyZC(bts)
				if err != nil {
					err = msgp.WrapError(err, "Prev")
					return
				}
				switch msgp.UnsafeString(field) {
				case "ClientID":
					z.Prev.ClientID, bts, err = msgp.ReadStringBytes(bts)
					if err != nil {
						err = msgp.WrapError(err, "Prev", "ClientID")
						return
					}
				case "ProposalID":
					z.Prev.ProposalID, bts, err = msgp.ReadStringBytes(bts)
					if err != nil {
						err = msgp.WrapError(err, "Prev", "ProposalID")
						return
					}
				default:
					bts, err = msgp.Skip(bts)
					if err != nil {
						err = msgp.WrapError(err, "Prev")
						return
					}
				}
			}
		case "Transfer":
			bts, err = z.Transfer.UnmarshalMsg(bts)
			if err != nil {
				err = msgp.WrapError(err, "Transfer")
				return
			}
		case "SignerThresholdIDs":
			var zb0004 uint32
			zb0004, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "SignerThresholdIDs")
				return
			}
			if cap(z.SignerThresholdIDs) >= int(zb0004) {
				z.SignerThresholdIDs = (z.SignerThresholdIDs)[:zb0004]
			} else {
				z.SignerThresholdIDs = make([]string, zb0004)
			}
			for za0001 := range z.SignerThresholdIDs {
				z.SignerThresholdIDs[za0001], bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					err = msgp.WrapError(err, "SignerThresholdIDs", za0001)
					return
				}
			}
		case "SignerSignatures":
			var zb0005 uint32
			zb0005, bts, err = msgp.ReadArrayHeaderBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "SignerSignatures")
				return
			}
			if cap(z.SignerSignatures) >= int(zb0005) {
				z.SignerSignatures = (z.SignerSignatures)[:zb0005]
			} else {
				z.SignerSignatures = make([]string, zb0005)
			}
			for za0002 := range z.SignerSignatures {
				z.SignerSignatures[za0002], bts, err = msgp.ReadStringBytes(bts)
				if err != nil {
					err = msgp.WrapError(err, "SignerSignatures", za0002)
					return
				}
			}
		case "ClientSignature":
			z.ClientSignature, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "ClientSignature")
				return
			}
		case "ExecutedInTxnHash":
			z.ExecutedInTxnHash, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "ExecutedInTxnHash")
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
func (z *proposal) Msgsize() (s int) {
	s = 1 + 11 + msgp.StringPrefixSize + len(z.ProposalID) + 15 + z.ExpirationDate.Msgsize() + 5 + 1 + 9 + msgp.StringPrefixSize + len(z.Next.ClientID) + 11 + msgp.StringPrefixSize + len(z.Next.ProposalID) + 5 + 1 + 9 + msgp.StringPrefixSize + len(z.Prev.ClientID) + 11 + msgp.StringPrefixSize + len(z.Prev.ProposalID) + 9 + z.Transfer.Msgsize() + 19 + msgp.ArrayHeaderSize
	for za0001 := range z.SignerThresholdIDs {
		s += msgp.StringPrefixSize + len(z.SignerThresholdIDs[za0001])
	}
	s += 17 + msgp.ArrayHeaderSize
	for za0002 := range z.SignerSignatures {
		s += msgp.StringPrefixSize + len(z.SignerSignatures[za0002])
	}
	s += 16 + msgp.StringPrefixSize + len(z.ClientSignature) + 18 + msgp.StringPrefixSize + len(z.ExecutedInTxnHash)
	return
}

// MarshalMsg implements msgp.Marshaler
func (z proposalRef) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 2
	// string "ClientID"
	o = append(o, 0x82, 0xa8, 0x43, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x49, 0x44)
	o = msgp.AppendString(o, z.ClientID)
	// string "ProposalID"
	o = append(o, 0xaa, 0x50, 0x72, 0x6f, 0x70, 0x6f, 0x73, 0x61, 0x6c, 0x49, 0x44)
	o = msgp.AppendString(o, z.ProposalID)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *proposalRef) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "ClientID":
			z.ClientID, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "ClientID")
				return
			}
		case "ProposalID":
			z.ProposalID, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "ProposalID")
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
func (z proposalRef) Msgsize() (s int) {
	s = 1 + 9 + msgp.StringPrefixSize + len(z.ClientID) + 11 + msgp.StringPrefixSize + len(z.ProposalID)
	return
}
