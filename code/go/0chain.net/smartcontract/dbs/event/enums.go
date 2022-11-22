package event

type (
	EventType int
	EventTag  int
)

const GB = 1024 * 1024 * 1024
const period = 10
const pageLimit = int64(50)

const (
	TypeNone EventType = iota
	TypeError
	TypeChain
	TypeStats
)

var TypeSting = []string{
	"none", "error", "chain", "stats",
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
	TagAddValidator
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
	TagCollectProviderReward
	TagSendTransfer
	TagReceiveTransfer
	TagLockStakePool
	TagUnlockStakePool
	TagLockWritePool
	TagUnlockWritePool
	TagLockReadPool
	TagUnlockReadPool
	TagToChallengePool
	TagFromChallengePool
	TagAddMint
	TagBurn
	TagAllocValueChange
	TagAllocBlobberValueChange
	TagUpdateBlobberOpenChallenges
	TagUpdateValidatorStakeTotal
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
	"TagCollectProviderReward",
	"TagSendTransfer",
	"TagReceiveTransfer",
	"TagLockStakePool",
	"TagUnlockStakePool",
	"TagLockWritePool",
	"TagUnlockWritePool",
	"TagLockReadPool",
	"TagUnlockReadPool",
	"TagToChallengePool",
	"TagFromChallengePool",
	"TagAddMint",
	"TagBurn",
	"TagAllocValueChange",
	"TagAllocBlobberValueChange",
	"TagUpdateBlobberOpenChallenges",
	"TagUpdateValidatorStakeTotal",
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
