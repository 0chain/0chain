package event

type (
	EventType int
	EventTag  int
)

const (
	TypeNone EventType = iota
	TypeError
	TypeStats
)

var TypeSting = []string{
	"none", "error", "stats",
}

func (t EventType) String() string {
	if int(t) < len(TagString) && int(t) >= 0 {
		return TagString[t]
	}

	return "unknown tag"
}

func (t EventType) Int() int {
	return int(t)
}

const (
	TagNone EventTag = iota
	TagAddBlobber
	TagUpdateBlobber
	TagUpdateBlobberAllocatedHealth
	TagUpdateBlobberTotalStake
	TagUpdateBlobberTotalOffers
	TagDeleteBlobber
	TagAddAuthorizer
	TagUpdateAuthorizer
	TagDeleteAuthorizer
	TagAddTransactions
	TagAddOrOverwriteUser
	TagAddWriteMarker
	TagAddBlock
	TagAddOrOverwriteValidator
	TagUpdateValidator
	TagAddReadMarker
	TagAddOrOverwriteMiner
	TagUpdateMiner
	TagDeleteMiner
	TagAddOrOverwriteSharder
	TagUpdateSharder
	TagDeleteSharder
	TagAddOrOverwriteCurator
	TagRemoveCurator
	TagStakePoolReward
	TagAddDelegatePool
	TagUpdateDelegatePool
	TagAddAllocation
	TagUpdateAllocationStakes
	TagUpdateAllocation
	TagMintReward
	TagAddChallenge
	TagUpdateChallenge
	TagUpdateBlobberChallenge
	TagUpdateAllocationChallenge
	TagAddChallengeToAllocation
	TagAddOrOverwriteAllocationBlobberTerm
	TagUpdateAllocationBlobberTerm
	TagDeleteAllocationBlobberTerm
	TagAddOrUpdateChallengePool
	TagUpdateAllocationStat
	TagUpdateBlobberStat
	TagUpdateValidatorStakeTotal
	TagCollectProviderReward
	NumberOfTags
)

var TagString = []string{
	"None",
	"AddBlobber",
	"UpdateBlobber",
	"UpdateBlobberAllocatedHealth",
	"UpdateBlobberTotalStake",
	"UpdateBlobberTotalOffers",
	"DeleteBlobber",
	"AddAuthorizer",
	"UpdateAuthorizer",
	"DeleteAuthorizer",
	"AddTransactions",
	"AddOrOverwriteUser",
	"AddWriteMarker",
	"AddBlock",
	"AddOrOverwriteValidator",
	"UpdateValidator",
	"AddReadMarker",
	"AddOrOverwriteMiner",
	"UpdateMiner",
	"DeleteMiner",
	"AddOrOverwriteSharder",
	"UpdateSharder",
	"DeleteSharder",
	"AddOrOverwriteCurator",
	"RemoveCurator",
	"StakePoolReward",
	"AddDelegatePool",
	"UpdateDelegatePool",
	"AddAllocation",
	"UpdateAllocationStakes",
	"UpdateAllocation",
	"MintReward",
	"AddChallenge",
	"UpdateChallenge",
	"UpdateBlobberChallenge",
	"UpdateAllocationChallenge",
	"AddChallengeToAllocation",
	"AddOrOverwriteAllocationBlobberTerm",
	"UpdateAllocationBlobberTerm",
	"DeleteAllocationBlobberTerm",
	"AddOrUpdateChallengePool",
	"UpdateAllocationStat",
	"UpdateBlobberStat",
	"UpdateValidatorStakeTotal",
	"TagCollectProviderReward",
	"Invalid",
}

func (tag EventTag) String() string {
	if int(tag) < len(TagString) && int(tag) >= 0 {
		return TagString[tag]
	}
	return "unknown tag"
}

func (tag EventTag) Int() int {
	return int(tag)
}
