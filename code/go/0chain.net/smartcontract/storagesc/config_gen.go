package storagesc

// Code generated by github.com/tinylib/msgp DO NOT EDIT.

import (
	"github.com/tinylib/msgp/msgp"
)

// MarshalMsg implements msgp.Marshaler
func (z *Config) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 35
	// string "TimeUnit"
	o = append(o, 0xde, 0x0, 0x23, 0xa8, 0x54, 0x69, 0x6d, 0x65, 0x55, 0x6e, 0x69, 0x74)
	o = msgp.AppendDuration(o, z.TimeUnit)
	// string "MaxMint"
	o = append(o, 0xa7, 0x4d, 0x61, 0x78, 0x4d, 0x69, 0x6e, 0x74)
	o, err = z.MaxMint.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "MaxMint")
		return
	}
	// string "Minted"
	o = append(o, 0xa6, 0x4d, 0x69, 0x6e, 0x74, 0x65, 0x64)
	o, err = z.Minted.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "Minted")
		return
	}
	// string "MinAllocSize"
	o = append(o, 0xac, 0x4d, 0x69, 0x6e, 0x41, 0x6c, 0x6c, 0x6f, 0x63, 0x53, 0x69, 0x7a, 0x65)
	o = msgp.AppendInt64(o, z.MinAllocSize)
	// string "MinAllocDuration"
	o = append(o, 0xb0, 0x4d, 0x69, 0x6e, 0x41, 0x6c, 0x6c, 0x6f, 0x63, 0x44, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e)
	o = msgp.AppendDuration(o, z.MinAllocDuration)
	// string "ChallengeCompletionTime"
	o = append(o, 0xb7, 0x43, 0x68, 0x61, 0x6c, 0x6c, 0x65, 0x6e, 0x67, 0x65, 0x43, 0x6f, 0x6d, 0x70, 0x6c, 0x65, 0x74, 0x69, 0x6f, 0x6e, 0x54, 0x69, 0x6d, 0x65)
	o = msgp.AppendDuration(o, z.ChallengeCompletionTime)
	// string "MinOfferDuration"
	o = append(o, 0xb0, 0x4d, 0x69, 0x6e, 0x4f, 0x66, 0x66, 0x65, 0x72, 0x44, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e)
	o = msgp.AppendDuration(o, z.MinOfferDuration)
	// string "MinBlobberCapacity"
	o = append(o, 0xb2, 0x4d, 0x69, 0x6e, 0x42, 0x6c, 0x6f, 0x62, 0x62, 0x65, 0x72, 0x43, 0x61, 0x70, 0x61, 0x63, 0x69, 0x74, 0x79)
	o = msgp.AppendInt64(o, z.MinBlobberCapacity)
	// string "ReadPool"
	o = append(o, 0xa8, 0x52, 0x65, 0x61, 0x64, 0x50, 0x6f, 0x6f, 0x6c)
	if z.ReadPool == nil {
		o = msgp.AppendNil(o)
	} else {
		// map header, size 1
		// string "MinLock"
		o = append(o, 0x81, 0xa7, 0x4d, 0x69, 0x6e, 0x4c, 0x6f, 0x63, 0x6b)
		o, err = z.ReadPool.MinLock.MarshalMsg(o)
		if err != nil {
			err = msgp.WrapError(err, "ReadPool", "MinLock")
			return
		}
	}
	// string "WritePool"
	o = append(o, 0xa9, 0x57, 0x72, 0x69, 0x74, 0x65, 0x50, 0x6f, 0x6f, 0x6c)
	if z.WritePool == nil {
		o = msgp.AppendNil(o)
	} else {
		// map header, size 1
		// string "MinLock"
		o = append(o, 0x81, 0xa7, 0x4d, 0x69, 0x6e, 0x4c, 0x6f, 0x63, 0x6b)
		o, err = z.WritePool.MinLock.MarshalMsg(o)
		if err != nil {
			err = msgp.WrapError(err, "WritePool", "MinLock")
			return
		}
	}
	// string "StakePool"
	o = append(o, 0xa9, 0x53, 0x74, 0x61, 0x6b, 0x65, 0x50, 0x6f, 0x6f, 0x6c)
	if z.StakePool == nil {
		o = msgp.AppendNil(o)
	} else {
		// map header, size 2
		// string "MinLock"
		o = append(o, 0x82, 0xa7, 0x4d, 0x69, 0x6e, 0x4c, 0x6f, 0x63, 0x6b)
		o, err = z.StakePool.MinLock.MarshalMsg(o)
		if err != nil {
			err = msgp.WrapError(err, "StakePool", "MinLock")
			return
		}
		// string "MinLockPeriod"
		o = append(o, 0xad, 0x4d, 0x69, 0x6e, 0x4c, 0x6f, 0x63, 0x6b, 0x50, 0x65, 0x72, 0x69, 0x6f, 0x64)
		o = msgp.AppendDuration(o, z.StakePool.MinLockPeriod)
	}
	// string "ValidatorReward"
	o = append(o, 0xaf, 0x56, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x6f, 0x72, 0x52, 0x65, 0x77, 0x61, 0x72, 0x64)
	o = msgp.AppendFloat64(o, z.ValidatorReward)
	// string "BlobberSlash"
	o = append(o, 0xac, 0x42, 0x6c, 0x6f, 0x62, 0x62, 0x65, 0x72, 0x53, 0x6c, 0x61, 0x73, 0x68)
	o = msgp.AppendFloat64(o, z.BlobberSlash)
	// string "MaxBlobbersPerAllocation"
	o = append(o, 0xb8, 0x4d, 0x61, 0x78, 0x42, 0x6c, 0x6f, 0x62, 0x62, 0x65, 0x72, 0x73, 0x50, 0x65, 0x72, 0x41, 0x6c, 0x6c, 0x6f, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e)
	o = msgp.AppendInt(o, z.MaxBlobbersPerAllocation)
	// string "MaxReadPrice"
	o = append(o, 0xac, 0x4d, 0x61, 0x78, 0x52, 0x65, 0x61, 0x64, 0x50, 0x72, 0x69, 0x63, 0x65)
	o, err = z.MaxReadPrice.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "MaxReadPrice")
		return
	}
	// string "MaxWritePrice"
	o = append(o, 0xad, 0x4d, 0x61, 0x78, 0x57, 0x72, 0x69, 0x74, 0x65, 0x50, 0x72, 0x69, 0x63, 0x65)
	o, err = z.MaxWritePrice.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "MaxWritePrice")
		return
	}
	// string "MinWritePrice"
	o = append(o, 0xad, 0x4d, 0x69, 0x6e, 0x57, 0x72, 0x69, 0x74, 0x65, 0x50, 0x72, 0x69, 0x63, 0x65)
	o, err = z.MinWritePrice.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "MinWritePrice")
		return
	}
	// string "CancellationCharge"
	o = append(o, 0xb2, 0x43, 0x61, 0x6e, 0x63, 0x65, 0x6c, 0x6c, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x43, 0x68, 0x61, 0x72, 0x67, 0x65)
	o = msgp.AppendFloat64(o, z.CancellationCharge)
	// string "FailedChallengesToCancel"
	o = append(o, 0xb8, 0x46, 0x61, 0x69, 0x6c, 0x65, 0x64, 0x43, 0x68, 0x61, 0x6c, 0x6c, 0x65, 0x6e, 0x67, 0x65, 0x73, 0x54, 0x6f, 0x43, 0x61, 0x6e, 0x63, 0x65, 0x6c)
	o = msgp.AppendInt(o, z.FailedChallengesToCancel)
	// string "FailedChallengesToRevokeMinLock"
	o = append(o, 0xbf, 0x46, 0x61, 0x69, 0x6c, 0x65, 0x64, 0x43, 0x68, 0x61, 0x6c, 0x6c, 0x65, 0x6e, 0x67, 0x65, 0x73, 0x54, 0x6f, 0x52, 0x65, 0x76, 0x6f, 0x6b, 0x65, 0x4d, 0x69, 0x6e, 0x4c, 0x6f, 0x63, 0x6b)
	o = msgp.AppendInt(o, z.FailedChallengesToRevokeMinLock)
	// string "MaxTotalFreeAllocation"
	o = append(o, 0xb6, 0x4d, 0x61, 0x78, 0x54, 0x6f, 0x74, 0x61, 0x6c, 0x46, 0x72, 0x65, 0x65, 0x41, 0x6c, 0x6c, 0x6f, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e)
	o, err = z.MaxTotalFreeAllocation.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "MaxTotalFreeAllocation")
		return
	}
	// string "MaxIndividualFreeAllocation"
	o = append(o, 0xbb, 0x4d, 0x61, 0x78, 0x49, 0x6e, 0x64, 0x69, 0x76, 0x69, 0x64, 0x75, 0x61, 0x6c, 0x46, 0x72, 0x65, 0x65, 0x41, 0x6c, 0x6c, 0x6f, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e)
	o, err = z.MaxIndividualFreeAllocation.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "MaxIndividualFreeAllocation")
		return
	}
	// string "FreeAllocationSettings"
	o = append(o, 0xb6, 0x46, 0x72, 0x65, 0x65, 0x41, 0x6c, 0x6c, 0x6f, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x65, 0x74, 0x74, 0x69, 0x6e, 0x67, 0x73)
	o, err = z.FreeAllocationSettings.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "FreeAllocationSettings")
		return
	}
	// string "ChallengeEnabled"
	o = append(o, 0xb0, 0x43, 0x68, 0x61, 0x6c, 0x6c, 0x65, 0x6e, 0x67, 0x65, 0x45, 0x6e, 0x61, 0x62, 0x6c, 0x65, 0x64)
	o = msgp.AppendBool(o, z.ChallengeEnabled)
	// string "MaxChallengesPerGeneration"
	o = append(o, 0xba, 0x4d, 0x61, 0x78, 0x43, 0x68, 0x61, 0x6c, 0x6c, 0x65, 0x6e, 0x67, 0x65, 0x73, 0x50, 0x65, 0x72, 0x47, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e)
	o = msgp.AppendInt(o, z.MaxChallengesPerGeneration)
	// string "ValidatorsPerChallenge"
	o = append(o, 0xb6, 0x56, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x6f, 0x72, 0x73, 0x50, 0x65, 0x72, 0x43, 0x68, 0x61, 0x6c, 0x6c, 0x65, 0x6e, 0x67, 0x65)
	o = msgp.AppendInt(o, z.ValidatorsPerChallenge)
	// string "ChallengeGenerationRate"
	o = append(o, 0xb7, 0x43, 0x68, 0x61, 0x6c, 0x6c, 0x65, 0x6e, 0x67, 0x65, 0x47, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x61, 0x74, 0x65)
	o = msgp.AppendFloat64(o, z.ChallengeGenerationRate)
	// string "MinStake"
	o = append(o, 0xa8, 0x4d, 0x69, 0x6e, 0x53, 0x74, 0x61, 0x6b, 0x65)
	o, err = z.MinStake.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "MinStake")
		return
	}
	// string "MaxStake"
	o = append(o, 0xa8, 0x4d, 0x61, 0x78, 0x53, 0x74, 0x61, 0x6b, 0x65)
	o, err = z.MaxStake.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "MaxStake")
		return
	}
	// string "MaxDelegates"
	o = append(o, 0xac, 0x4d, 0x61, 0x78, 0x44, 0x65, 0x6c, 0x65, 0x67, 0x61, 0x74, 0x65, 0x73)
	o = msgp.AppendInt(o, z.MaxDelegates)
	// string "MaxCharge"
	o = append(o, 0xa9, 0x4d, 0x61, 0x78, 0x43, 0x68, 0x61, 0x72, 0x67, 0x65)
	o = msgp.AppendFloat64(o, z.MaxCharge)
	// string "BlockReward"
	o = append(o, 0xab, 0x42, 0x6c, 0x6f, 0x63, 0x6b, 0x52, 0x65, 0x77, 0x61, 0x72, 0x64)
	if z.BlockReward == nil {
		o = msgp.AppendNil(o)
	} else {
		o, err = z.BlockReward.MarshalMsg(o)
		if err != nil {
			err = msgp.WrapError(err, "BlockReward")
			return
		}
	}
	// string "ExposeMpt"
	o = append(o, 0xa9, 0x45, 0x78, 0x70, 0x6f, 0x73, 0x65, 0x4d, 0x70, 0x74)
	o = msgp.AppendBool(o, z.ExposeMpt)
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
func (z *Config) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "TimeUnit":
			z.TimeUnit, bts, err = msgp.ReadDurationBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "TimeUnit")
				return
			}
		case "MaxMint":
			bts, err = z.MaxMint.UnmarshalMsg(bts)
			if err != nil {
				err = msgp.WrapError(err, "MaxMint")
				return
			}
		case "Minted":
			bts, err = z.Minted.UnmarshalMsg(bts)
			if err != nil {
				err = msgp.WrapError(err, "Minted")
				return
			}
		case "MinAllocSize":
			z.MinAllocSize, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "MinAllocSize")
				return
			}
		case "MinAllocDuration":
			z.MinAllocDuration, bts, err = msgp.ReadDurationBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "MinAllocDuration")
				return
			}
		case "ChallengeCompletionTime":
			z.ChallengeCompletionTime, bts, err = msgp.ReadDurationBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "ChallengeCompletionTime")
				return
			}
		case "MinOfferDuration":
			z.MinOfferDuration, bts, err = msgp.ReadDurationBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "MinOfferDuration")
				return
			}
		case "MinBlobberCapacity":
			z.MinBlobberCapacity, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "MinBlobberCapacity")
				return
			}
		case "ReadPool":
			if msgp.IsNil(bts) {
				bts, err = msgp.ReadNilBytes(bts)
				if err != nil {
					return
				}
				z.ReadPool = nil
			} else {
				if z.ReadPool == nil {
					z.ReadPool = new(readPoolConfig)
				}
				var zb0002 uint32
				zb0002, bts, err = msgp.ReadMapHeaderBytes(bts)
				if err != nil {
					err = msgp.WrapError(err, "ReadPool")
					return
				}
				for zb0002 > 0 {
					zb0002--
					field, bts, err = msgp.ReadMapKeyZC(bts)
					if err != nil {
						err = msgp.WrapError(err, "ReadPool")
						return
					}
					switch msgp.UnsafeString(field) {
					case "MinLock":
						bts, err = z.ReadPool.MinLock.UnmarshalMsg(bts)
						if err != nil {
							err = msgp.WrapError(err, "ReadPool", "MinLock")
							return
						}
					default:
						bts, err = msgp.Skip(bts)
						if err != nil {
							err = msgp.WrapError(err, "ReadPool")
							return
						}
					}
				}
			}
		case "WritePool":
			if msgp.IsNil(bts) {
				bts, err = msgp.ReadNilBytes(bts)
				if err != nil {
					return
				}
				z.WritePool = nil
			} else {
				if z.WritePool == nil {
					z.WritePool = new(writePoolConfig)
				}
				var zb0003 uint32
				zb0003, bts, err = msgp.ReadMapHeaderBytes(bts)
				if err != nil {
					err = msgp.WrapError(err, "WritePool")
					return
				}
				for zb0003 > 0 {
					zb0003--
					field, bts, err = msgp.ReadMapKeyZC(bts)
					if err != nil {
						err = msgp.WrapError(err, "WritePool")
						return
					}
					switch msgp.UnsafeString(field) {
					case "MinLock":
						bts, err = z.WritePool.MinLock.UnmarshalMsg(bts)
						if err != nil {
							err = msgp.WrapError(err, "WritePool", "MinLock")
							return
						}
					default:
						bts, err = msgp.Skip(bts)
						if err != nil {
							err = msgp.WrapError(err, "WritePool")
							return
						}
					}
				}
			}
		case "StakePool":
			if msgp.IsNil(bts) {
				bts, err = msgp.ReadNilBytes(bts)
				if err != nil {
					return
				}
				z.StakePool = nil
			} else {
				if z.StakePool == nil {
					z.StakePool = new(stakePoolConfig)
				}
				var zb0004 uint32
				zb0004, bts, err = msgp.ReadMapHeaderBytes(bts)
				if err != nil {
					err = msgp.WrapError(err, "StakePool")
					return
				}
				for zb0004 > 0 {
					zb0004--
					field, bts, err = msgp.ReadMapKeyZC(bts)
					if err != nil {
						err = msgp.WrapError(err, "StakePool")
						return
					}
					switch msgp.UnsafeString(field) {
					case "MinLock":
						bts, err = z.StakePool.MinLock.UnmarshalMsg(bts)
						if err != nil {
							err = msgp.WrapError(err, "StakePool", "MinLock")
							return
						}
					case "MinLockPeriod":
						z.StakePool.MinLockPeriod, bts, err = msgp.ReadDurationBytes(bts)
						if err != nil {
							err = msgp.WrapError(err, "StakePool", "MinLockPeriod")
							return
						}
					default:
						bts, err = msgp.Skip(bts)
						if err != nil {
							err = msgp.WrapError(err, "StakePool")
							return
						}
					}
				}
			}
		case "ValidatorReward":
			z.ValidatorReward, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "ValidatorReward")
				return
			}
		case "BlobberSlash":
			z.BlobberSlash, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "BlobberSlash")
				return
			}
		case "MaxBlobbersPerAllocation":
			z.MaxBlobbersPerAllocation, bts, err = msgp.ReadIntBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "MaxBlobbersPerAllocation")
				return
			}
		case "MaxReadPrice":
			bts, err = z.MaxReadPrice.UnmarshalMsg(bts)
			if err != nil {
				err = msgp.WrapError(err, "MaxReadPrice")
				return
			}
		case "MaxWritePrice":
			bts, err = z.MaxWritePrice.UnmarshalMsg(bts)
			if err != nil {
				err = msgp.WrapError(err, "MaxWritePrice")
				return
			}
		case "MinWritePrice":
			bts, err = z.MinWritePrice.UnmarshalMsg(bts)
			if err != nil {
				err = msgp.WrapError(err, "MinWritePrice")
				return
			}
		case "CancellationCharge":
			z.CancellationCharge, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "CancellationCharge")
				return
			}
		case "FailedChallengesToCancel":
			z.FailedChallengesToCancel, bts, err = msgp.ReadIntBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "FailedChallengesToCancel")
				return
			}
		case "FailedChallengesToRevokeMinLock":
			z.FailedChallengesToRevokeMinLock, bts, err = msgp.ReadIntBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "FailedChallengesToRevokeMinLock")
				return
			}
		case "MaxTotalFreeAllocation":
			bts, err = z.MaxTotalFreeAllocation.UnmarshalMsg(bts)
			if err != nil {
				err = msgp.WrapError(err, "MaxTotalFreeAllocation")
				return
			}
		case "MaxIndividualFreeAllocation":
			bts, err = z.MaxIndividualFreeAllocation.UnmarshalMsg(bts)
			if err != nil {
				err = msgp.WrapError(err, "MaxIndividualFreeAllocation")
				return
			}
		case "FreeAllocationSettings":
			bts, err = z.FreeAllocationSettings.UnmarshalMsg(bts)
			if err != nil {
				err = msgp.WrapError(err, "FreeAllocationSettings")
				return
			}
		case "ChallengeEnabled":
			z.ChallengeEnabled, bts, err = msgp.ReadBoolBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "ChallengeEnabled")
				return
			}
		case "MaxChallengesPerGeneration":
			z.MaxChallengesPerGeneration, bts, err = msgp.ReadIntBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "MaxChallengesPerGeneration")
				return
			}
		case "ValidatorsPerChallenge":
			z.ValidatorsPerChallenge, bts, err = msgp.ReadIntBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "ValidatorsPerChallenge")
				return
			}
		case "ChallengeGenerationRate":
			z.ChallengeGenerationRate, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "ChallengeGenerationRate")
				return
			}
		case "MinStake":
			bts, err = z.MinStake.UnmarshalMsg(bts)
			if err != nil {
				err = msgp.WrapError(err, "MinStake")
				return
			}
		case "MaxStake":
			bts, err = z.MaxStake.UnmarshalMsg(bts)
			if err != nil {
				err = msgp.WrapError(err, "MaxStake")
				return
			}
		case "MaxDelegates":
			z.MaxDelegates, bts, err = msgp.ReadIntBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "MaxDelegates")
				return
			}
		case "MaxCharge":
			z.MaxCharge, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "MaxCharge")
				return
			}
		case "BlockReward":
			if msgp.IsNil(bts) {
				bts, err = msgp.ReadNilBytes(bts)
				if err != nil {
					return
				}
				z.BlockReward = nil
			} else {
				if z.BlockReward == nil {
					z.BlockReward = new(blockReward)
				}
				bts, err = z.BlockReward.UnmarshalMsg(bts)
				if err != nil {
					err = msgp.WrapError(err, "BlockReward")
					return
				}
			}
		case "ExposeMpt":
			z.ExposeMpt, bts, err = msgp.ReadBoolBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "ExposeMpt")
				return
			}
		case "OwnerId":
			z.OwnerId, bts, err = msgp.ReadStringBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "OwnerId")
				return
			}
		case "Cost":
			var zb0005 uint32
			zb0005, bts, err = msgp.ReadMapHeaderBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Cost")
				return
			}
			if z.Cost == nil {
				z.Cost = make(map[string]int, zb0005)
			} else if len(z.Cost) > 0 {
				for key := range z.Cost {
					delete(z.Cost, key)
				}
			}
			for zb0005 > 0 {
				var za0001 string
				var za0002 int
				zb0005--
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
func (z *Config) Msgsize() (s int) {
	s = 3 + 9 + msgp.DurationSize + 8 + z.MaxMint.Msgsize() + 7 + z.Minted.Msgsize() + 13 + msgp.Int64Size + 17 + msgp.DurationSize + 24 + msgp.DurationSize + 17 + msgp.DurationSize + 19 + msgp.Int64Size + 9
	if z.ReadPool == nil {
		s += msgp.NilSize
	} else {
		s += 1 + 8 + z.ReadPool.MinLock.Msgsize()
	}
	s += 10
	if z.WritePool == nil {
		s += msgp.NilSize
	} else {
		s += 1 + 8 + z.WritePool.MinLock.Msgsize()
	}
	s += 10
	if z.StakePool == nil {
		s += msgp.NilSize
	} else {
		s += 1 + 8 + z.StakePool.MinLock.Msgsize() + 14 + msgp.DurationSize
	}
	s += 16 + msgp.Float64Size + 13 + msgp.Float64Size + 25 + msgp.IntSize + 13 + z.MaxReadPrice.Msgsize() + 14 + z.MaxWritePrice.Msgsize() + 14 + z.MinWritePrice.Msgsize() + 19 + msgp.Float64Size + 25 + msgp.IntSize + 32 + msgp.IntSize + 23 + z.MaxTotalFreeAllocation.Msgsize() + 28 + z.MaxIndividualFreeAllocation.Msgsize() + 23 + z.FreeAllocationSettings.Msgsize() + 17 + msgp.BoolSize + 27 + msgp.IntSize + 23 + msgp.IntSize + 24 + msgp.Float64Size + 9 + z.MinStake.Msgsize() + 9 + z.MaxStake.Msgsize() + 13 + msgp.IntSize + 10 + msgp.Float64Size + 12
	if z.BlockReward == nil {
		s += msgp.NilSize
	} else {
		s += z.BlockReward.Msgsize()
	}
	s += 10 + msgp.BoolSize + 8 + msgp.StringPrefixSize + len(z.OwnerId) + 5 + msgp.MapHeaderSize
	if z.Cost != nil {
		for za0001, za0002 := range z.Cost {
			_ = za0002
			s += msgp.StringPrefixSize + len(za0001) + msgp.IntSize
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *blockReward) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 10
	// string "BlockReward"
	o = append(o, 0x8a, 0xab, 0x42, 0x6c, 0x6f, 0x63, 0x6b, 0x52, 0x65, 0x77, 0x61, 0x72, 0x64)
	o, err = z.BlockReward.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "BlockReward")
		return
	}
	// string "BlockRewardChangePeriod"
	o = append(o, 0xb7, 0x42, 0x6c, 0x6f, 0x63, 0x6b, 0x52, 0x65, 0x77, 0x61, 0x72, 0x64, 0x43, 0x68, 0x61, 0x6e, 0x67, 0x65, 0x50, 0x65, 0x72, 0x69, 0x6f, 0x64)
	o = msgp.AppendInt64(o, z.BlockRewardChangePeriod)
	// string "BlockRewardChangeRatio"
	o = append(o, 0xb6, 0x42, 0x6c, 0x6f, 0x63, 0x6b, 0x52, 0x65, 0x77, 0x61, 0x72, 0x64, 0x43, 0x68, 0x61, 0x6e, 0x67, 0x65, 0x52, 0x61, 0x74, 0x69, 0x6f)
	o = msgp.AppendFloat64(o, z.BlockRewardChangeRatio)
	// string "QualifyingStake"
	o = append(o, 0xaf, 0x51, 0x75, 0x61, 0x6c, 0x69, 0x66, 0x79, 0x69, 0x6e, 0x67, 0x53, 0x74, 0x61, 0x6b, 0x65)
	o, err = z.QualifyingStake.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "QualifyingStake")
		return
	}
	// string "SharderWeight"
	o = append(o, 0xad, 0x53, 0x68, 0x61, 0x72, 0x64, 0x65, 0x72, 0x57, 0x65, 0x69, 0x67, 0x68, 0x74)
	o = msgp.AppendFloat64(o, z.SharderWeight)
	// string "MinerWeight"
	o = append(o, 0xab, 0x4d, 0x69, 0x6e, 0x65, 0x72, 0x57, 0x65, 0x69, 0x67, 0x68, 0x74)
	o = msgp.AppendFloat64(o, z.MinerWeight)
	// string "BlobberWeight"
	o = append(o, 0xad, 0x42, 0x6c, 0x6f, 0x62, 0x62, 0x65, 0x72, 0x57, 0x65, 0x69, 0x67, 0x68, 0x74)
	o = msgp.AppendFloat64(o, z.BlobberWeight)
	// string "TriggerPeriod"
	o = append(o, 0xad, 0x54, 0x72, 0x69, 0x67, 0x67, 0x65, 0x72, 0x50, 0x65, 0x72, 0x69, 0x6f, 0x64)
	o = msgp.AppendInt64(o, z.TriggerPeriod)
	// string "Gamma"
	o = append(o, 0xa5, 0x47, 0x61, 0x6d, 0x6d, 0x61)
	// map header, size 3
	// string "Alpha"
	o = append(o, 0x83, 0xa5, 0x41, 0x6c, 0x70, 0x68, 0x61)
	o = msgp.AppendFloat64(o, z.Gamma.Alpha)
	// string "A"
	o = append(o, 0xa1, 0x41)
	o = msgp.AppendFloat64(o, z.Gamma.A)
	// string "B"
	o = append(o, 0xa1, 0x42)
	o = msgp.AppendFloat64(o, z.Gamma.B)
	// string "Zeta"
	o = append(o, 0xa4, 0x5a, 0x65, 0x74, 0x61)
	// map header, size 3
	// string "I"
	o = append(o, 0x83, 0xa1, 0x49)
	o = msgp.AppendFloat64(o, z.Zeta.I)
	// string "K"
	o = append(o, 0xa1, 0x4b)
	o = msgp.AppendFloat64(o, z.Zeta.K)
	// string "Mu"
	o = append(o, 0xa2, 0x4d, 0x75)
	o = msgp.AppendFloat64(o, z.Zeta.Mu)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *blockReward) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "BlockReward":
			bts, err = z.BlockReward.UnmarshalMsg(bts)
			if err != nil {
				err = msgp.WrapError(err, "BlockReward")
				return
			}
		case "BlockRewardChangePeriod":
			z.BlockRewardChangePeriod, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "BlockRewardChangePeriod")
				return
			}
		case "BlockRewardChangeRatio":
			z.BlockRewardChangeRatio, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "BlockRewardChangeRatio")
				return
			}
		case "QualifyingStake":
			bts, err = z.QualifyingStake.UnmarshalMsg(bts)
			if err != nil {
				err = msgp.WrapError(err, "QualifyingStake")
				return
			}
		case "SharderWeight":
			z.SharderWeight, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "SharderWeight")
				return
			}
		case "MinerWeight":
			z.MinerWeight, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "MinerWeight")
				return
			}
		case "BlobberWeight":
			z.BlobberWeight, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "BlobberWeight")
				return
			}
		case "TriggerPeriod":
			z.TriggerPeriod, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "TriggerPeriod")
				return
			}
		case "Gamma":
			var zb0002 uint32
			zb0002, bts, err = msgp.ReadMapHeaderBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Gamma")
				return
			}
			for zb0002 > 0 {
				zb0002--
				field, bts, err = msgp.ReadMapKeyZC(bts)
				if err != nil {
					err = msgp.WrapError(err, "Gamma")
					return
				}
				switch msgp.UnsafeString(field) {
				case "Alpha":
					z.Gamma.Alpha, bts, err = msgp.ReadFloat64Bytes(bts)
					if err != nil {
						err = msgp.WrapError(err, "Gamma", "Alpha")
						return
					}
				case "A":
					z.Gamma.A, bts, err = msgp.ReadFloat64Bytes(bts)
					if err != nil {
						err = msgp.WrapError(err, "Gamma", "A")
						return
					}
				case "B":
					z.Gamma.B, bts, err = msgp.ReadFloat64Bytes(bts)
					if err != nil {
						err = msgp.WrapError(err, "Gamma", "B")
						return
					}
				default:
					bts, err = msgp.Skip(bts)
					if err != nil {
						err = msgp.WrapError(err, "Gamma")
						return
					}
				}
			}
		case "Zeta":
			var zb0003 uint32
			zb0003, bts, err = msgp.ReadMapHeaderBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Zeta")
				return
			}
			for zb0003 > 0 {
				zb0003--
				field, bts, err = msgp.ReadMapKeyZC(bts)
				if err != nil {
					err = msgp.WrapError(err, "Zeta")
					return
				}
				switch msgp.UnsafeString(field) {
				case "I":
					z.Zeta.I, bts, err = msgp.ReadFloat64Bytes(bts)
					if err != nil {
						err = msgp.WrapError(err, "Zeta", "I")
						return
					}
				case "K":
					z.Zeta.K, bts, err = msgp.ReadFloat64Bytes(bts)
					if err != nil {
						err = msgp.WrapError(err, "Zeta", "K")
						return
					}
				case "Mu":
					z.Zeta.Mu, bts, err = msgp.ReadFloat64Bytes(bts)
					if err != nil {
						err = msgp.WrapError(err, "Zeta", "Mu")
						return
					}
				default:
					bts, err = msgp.Skip(bts)
					if err != nil {
						err = msgp.WrapError(err, "Zeta")
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
func (z *blockReward) Msgsize() (s int) {
	s = 1 + 12 + z.BlockReward.Msgsize() + 24 + msgp.Int64Size + 23 + msgp.Float64Size + 16 + z.QualifyingStake.Msgsize() + 14 + msgp.Float64Size + 12 + msgp.Float64Size + 14 + msgp.Float64Size + 14 + msgp.Int64Size + 6 + 1 + 6 + msgp.Float64Size + 2 + msgp.Float64Size + 2 + msgp.Float64Size + 5 + 1 + 2 + msgp.Float64Size + 2 + msgp.Float64Size + 3 + msgp.Float64Size
	return
}

// MarshalMsg implements msgp.Marshaler
func (z blockRewardGamma) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 3
	// string "Alpha"
	o = append(o, 0x83, 0xa5, 0x41, 0x6c, 0x70, 0x68, 0x61)
	o = msgp.AppendFloat64(o, z.Alpha)
	// string "A"
	o = append(o, 0xa1, 0x41)
	o = msgp.AppendFloat64(o, z.A)
	// string "B"
	o = append(o, 0xa1, 0x42)
	o = msgp.AppendFloat64(o, z.B)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *blockRewardGamma) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "Alpha":
			z.Alpha, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Alpha")
				return
			}
		case "A":
			z.A, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "A")
				return
			}
		case "B":
			z.B, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "B")
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
func (z blockRewardGamma) Msgsize() (s int) {
	s = 1 + 6 + msgp.Float64Size + 2 + msgp.Float64Size + 2 + msgp.Float64Size
	return
}

// MarshalMsg implements msgp.Marshaler
func (z blockRewardZeta) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 3
	// string "I"
	o = append(o, 0x83, 0xa1, 0x49)
	o = msgp.AppendFloat64(o, z.I)
	// string "K"
	o = append(o, 0xa1, 0x4b)
	o = msgp.AppendFloat64(o, z.K)
	// string "Mu"
	o = append(o, 0xa2, 0x4d, 0x75)
	o = msgp.AppendFloat64(o, z.Mu)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *blockRewardZeta) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "I":
			z.I, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "I")
				return
			}
		case "K":
			z.K, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "K")
				return
			}
		case "Mu":
			z.Mu, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Mu")
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
func (z blockRewardZeta) Msgsize() (s int) {
	s = 1 + 2 + msgp.Float64Size + 2 + msgp.Float64Size + 3 + msgp.Float64Size
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *freeAllocationSettings) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 7
	// string "DataShards"
	o = append(o, 0x87, 0xaa, 0x44, 0x61, 0x74, 0x61, 0x53, 0x68, 0x61, 0x72, 0x64, 0x73)
	o = msgp.AppendInt(o, z.DataShards)
	// string "ParityShards"
	o = append(o, 0xac, 0x50, 0x61, 0x72, 0x69, 0x74, 0x79, 0x53, 0x68, 0x61, 0x72, 0x64, 0x73)
	o = msgp.AppendInt(o, z.ParityShards)
	// string "Size"
	o = append(o, 0xa4, 0x53, 0x69, 0x7a, 0x65)
	o = msgp.AppendInt64(o, z.Size)
	// string "Duration"
	o = append(o, 0xa8, 0x44, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e)
	o = msgp.AppendDuration(o, z.Duration)
	// string "ReadPriceRange"
	o = append(o, 0xae, 0x52, 0x65, 0x61, 0x64, 0x50, 0x72, 0x69, 0x63, 0x65, 0x52, 0x61, 0x6e, 0x67, 0x65)
	o, err = z.ReadPriceRange.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "ReadPriceRange")
		return
	}
	// string "WritePriceRange"
	o = append(o, 0xaf, 0x57, 0x72, 0x69, 0x74, 0x65, 0x50, 0x72, 0x69, 0x63, 0x65, 0x52, 0x61, 0x6e, 0x67, 0x65)
	o, err = z.WritePriceRange.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "WritePriceRange")
		return
	}
	// string "ReadPoolFraction"
	o = append(o, 0xb0, 0x52, 0x65, 0x61, 0x64, 0x50, 0x6f, 0x6f, 0x6c, 0x46, 0x72, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e)
	o = msgp.AppendFloat64(o, z.ReadPoolFraction)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *freeAllocationSettings) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "DataShards":
			z.DataShards, bts, err = msgp.ReadIntBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "DataShards")
				return
			}
		case "ParityShards":
			z.ParityShards, bts, err = msgp.ReadIntBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "ParityShards")
				return
			}
		case "Size":
			z.Size, bts, err = msgp.ReadInt64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Size")
				return
			}
		case "Duration":
			z.Duration, bts, err = msgp.ReadDurationBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "Duration")
				return
			}
		case "ReadPriceRange":
			bts, err = z.ReadPriceRange.UnmarshalMsg(bts)
			if err != nil {
				err = msgp.WrapError(err, "ReadPriceRange")
				return
			}
		case "WritePriceRange":
			bts, err = z.WritePriceRange.UnmarshalMsg(bts)
			if err != nil {
				err = msgp.WrapError(err, "WritePriceRange")
				return
			}
		case "ReadPoolFraction":
			z.ReadPoolFraction, bts, err = msgp.ReadFloat64Bytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "ReadPoolFraction")
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
func (z *freeAllocationSettings) Msgsize() (s int) {
	s = 1 + 11 + msgp.IntSize + 13 + msgp.IntSize + 5 + msgp.Int64Size + 9 + msgp.DurationSize + 15 + z.ReadPriceRange.Msgsize() + 16 + z.WritePriceRange.Msgsize() + 17 + msgp.Float64Size
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *readPoolConfig) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 1
	// string "MinLock"
	o = append(o, 0x81, 0xa7, 0x4d, 0x69, 0x6e, 0x4c, 0x6f, 0x63, 0x6b)
	o, err = z.MinLock.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "MinLock")
		return
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *readPoolConfig) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "MinLock":
			bts, err = z.MinLock.UnmarshalMsg(bts)
			if err != nil {
				err = msgp.WrapError(err, "MinLock")
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
func (z *readPoolConfig) Msgsize() (s int) {
	s = 1 + 8 + z.MinLock.Msgsize()
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *stakePoolConfig) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 2
	// string "MinLock"
	o = append(o, 0x82, 0xa7, 0x4d, 0x69, 0x6e, 0x4c, 0x6f, 0x63, 0x6b)
	o, err = z.MinLock.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "MinLock")
		return
	}
	// string "MinLockPeriod"
	o = append(o, 0xad, 0x4d, 0x69, 0x6e, 0x4c, 0x6f, 0x63, 0x6b, 0x50, 0x65, 0x72, 0x69, 0x6f, 0x64)
	o = msgp.AppendDuration(o, z.MinLockPeriod)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *stakePoolConfig) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "MinLock":
			bts, err = z.MinLock.UnmarshalMsg(bts)
			if err != nil {
				err = msgp.WrapError(err, "MinLock")
				return
			}
		case "MinLockPeriod":
			z.MinLockPeriod, bts, err = msgp.ReadDurationBytes(bts)
			if err != nil {
				err = msgp.WrapError(err, "MinLockPeriod")
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
func (z *stakePoolConfig) Msgsize() (s int) {
	s = 1 + 8 + z.MinLock.Msgsize() + 14 + msgp.DurationSize
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *writePoolConfig) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// map header, size 1
	// string "MinLock"
	o = append(o, 0x81, 0xa7, 0x4d, 0x69, 0x6e, 0x4c, 0x6f, 0x63, 0x6b)
	o, err = z.MinLock.MarshalMsg(o)
	if err != nil {
		err = msgp.WrapError(err, "MinLock")
		return
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *writePoolConfig) UnmarshalMsg(bts []byte) (o []byte, err error) {
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
		case "MinLock":
			bts, err = z.MinLock.UnmarshalMsg(bts)
			if err != nil {
				err = msgp.WrapError(err, "MinLock")
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
func (z *writePoolConfig) Msgsize() (s int) {
	s = 1 + 8 + z.MinLock.Msgsize()
	return
}
