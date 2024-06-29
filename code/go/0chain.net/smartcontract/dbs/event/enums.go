package event

type (
	EventType    int
	EventTag     int
	EventVersion string
)

const (
	Version1 EventVersion = "1.0"
)

const (
	TypeNone EventType = iota
	TypeError
	TypeChain
	TypeStats
	NumberOfTypes
)

// ProviderTable Table names of all providers in events_db, used in general provider handlers
type ProviderTable string

const (
	MinerTable      ProviderTable = "miners"
	SharderTable    ProviderTable = "sharders"
	BlobberTable    ProviderTable = "blobbers"
	AuthorizerTable ProviderTable = "authorizers"
	ValidatorTable  ProviderTable = "validators"
)

func (t EventType) String() string {
	if int(t) < len(TypeString) && int(t) >= 0 {
		return TypeString[t]
	}

	return "unknown type"
}

func (t EventType) Int() int {
	return int(t)
}

const (
	TagNone EventTag = iota
	TagAddBlobber
	TagUpdateBlobber
	TagUpdateBlobberAllocatedSavedHealth
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
	TagFinalizeBlock
	TagAddOrOverwiteValidator
	TagUpdateValidator
	TagAddReadMarker
	TagAddMiner
	TagUpdateMiner
	TagDeleteMiner
	TagAddSharder
	TagUpdateSharder
	TagDeleteSharder
	TagStakePoolReward
	TagStakePoolPenalty
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
	TagLockStakePool
	TagUnlockStakePool
	TagLockWritePool
	TagUnlockWritePool
	TagLockReadPool
	TagUnlockReadPool
	TagToChallengePool
	TagFromChallengePool
	TagUpdateValidatorStakeTotal
	TagUpdateMinerTotalStake
	TagUpdateSharderTotalStake
	TagUpdateAuthorizerTotalStake
	TagUniqueAddress
	TagMinerHealthCheck
	TagSharderHealthCheck
	TagBlobberHealthCheck
	TagAuthorizerHealthCheck
	TagValidatorHealthCheck
	TagUpdateUserPayedFees
	TagUpdateUserCollectedRewards
	TagAddBurnTicket
	TagAuthorizerBurn
	TagAddBridgeMint
	TagKillProvider
	TagShutdownProvider
	TagInsertReadpool
	TagUpdateReadpool
	NumberOfTags
)

var (
	TypeString []string
	TagString  []string
)

func init() {
	initTypeString()
	initTagString()
}

func initTypeString() {
	TypeString = make([]string, NumberOfTypes+1)
	TypeString[TypeNone] = "none"
	TypeString[TypeError] = "error"
	TypeString[TypeChain] = "chain"
	TypeString[TypeStats] = "stats"
	TypeString[NumberOfTypes] = "invalid"
}

func initTagString() {
	TagString = make([]string, NumberOfTags+1)
	TagString[TagNone] = "none"
	TagString[TagAddBlobber] = "TagAddBlobber"
	TagString[TagUpdateBlobber] = "TagUpdateBlobber"
	TagString[TagUpdateBlobberAllocatedSavedHealth] = "TagUpdateBlobberAllocatedSavedHealth"
	TagString[TagUpdateBlobberTotalStake] = "TagUpdateBlobberTotalStake"
	TagString[TagUpdateBlobberTotalOffers] = "TagUpdateBlobberTotalOffers"
	TagString[TagDeleteBlobber] = "TagDeleteBlobber"
	TagString[TagAddAuthorizer] = "TagAddAuthorizer"
	TagString[TagUpdateAuthorizer] = "TagUpdateAuthorizer"
	TagString[TagDeleteAuthorizer] = "TagDeleteAuthorizer"
	TagString[TagAddTransactions] = "TagAddTransactions"
	TagString[TagAddOrOverwriteUser] = "TagAddOrOverwriteUser"
	TagString[TagAddWriteMarker] = "TagAddWriteMarker"
	TagString[TagFinalizeBlock] = "TagFinalizeBlock"
	TagString[TagAddOrOverwiteValidator] = "TagAddOrOverwiteValidator"
	TagString[TagUpdateValidator] = "TagUpdateValidator"
	TagString[TagAddReadMarker] = "TagAddReadMarker"
	TagString[TagAddMiner] = "TagAddMiner"
	TagString[TagUpdateMiner] = "TagUpdateMiner"
	TagString[TagDeleteMiner] = "TagDeleteMiner"
	TagString[TagAddSharder] = "TagAddSharder"
	TagString[TagUpdateSharder] = "TagUpdateSharder"
	TagString[TagDeleteSharder] = "TagDeleteSharder"
	TagString[TagStakePoolReward] = "TagStakePoolReward"
	TagString[TagStakePoolPenalty] = "TagStakePoolPenalty"
	TagString[TagAddDelegatePool] = "TagAddDelegatePool"
	TagString[TagUpdateDelegatePool] = "TagUpdateDelegatePool"
	TagString[TagAddAllocation] = "TagAddAllocation"
	TagString[TagUpdateAllocationStakes] = "TagUpdateAllocationStakes"
	TagString[TagUpdateAllocation] = "TagUpdateAllocation"
	TagString[TagMintReward] = "TagMintReward"
	TagString[TagAddChallenge] = "TagAddChallenge"
	TagString[TagUpdateChallenge] = "TagUpdateChallenge"
	TagString[TagUpdateBlobberChallenge] = "TagUpdateBlobberChallenge"
	TagString[TagUpdateAllocationChallenge] = "TagUpdateAllocationChallenge"
	TagString[TagAddChallengeToAllocation] = "TagAddChallengeToAllocation"
	TagString[TagAddOrOverwriteAllocationBlobberTerm] = "TagAddOrOverwriteAllocationBlobberTerm"
	TagString[TagUpdateAllocationBlobberTerm] = "TagUpdateAllocationBlobberTerm"
	TagString[TagDeleteAllocationBlobberTerm] = "TagDeleteAllocationBlobberTerm"
	TagString[TagAddOrUpdateChallengePool] = "TagAddOrUpdateChallengePool"
	TagString[TagUpdateAllocationStat] = "TagUpdateAllocationStat"
	TagString[TagUpdateBlobberStat] = "TagUpdateBlobberStat"
	TagString[TagCollectProviderReward] = "TagCollectProviderReward"
	TagString[TagLockStakePool] = "TagLockStakePool"
	TagString[TagUnlockStakePool] = "TagUnlockStakePool"
	TagString[TagLockWritePool] = "TagLockWritePool"
	TagString[TagUnlockWritePool] = "TagUnlockWritePool"
	TagString[TagLockReadPool] = "TagLockReadPool"
	TagString[TagUnlockReadPool] = "TagUnlockReadPool"
	TagString[TagToChallengePool] = "TagToChallengePool"
	TagString[TagFromChallengePool] = "TagFromChallengePool"
	TagString[TagUpdateValidatorStakeTotal] = "TagUpdateValidatorStakeTotal"
	TagString[TagUniqueAddress] = "TagUniqueAddress"
	TagString[TagMinerHealthCheck] = "TagMinerHealthCheck"
	TagString[TagSharderHealthCheck] = "TagSharderHealthCheck"
	TagString[TagBlobberHealthCheck] = "TagBlobberHealthCheck"
	TagString[TagAuthorizerHealthCheck] = "TagAuthorizerHealthCheck"
	TagString[TagValidatorHealthCheck] = "TagValidatorHealthCheck"
	TagString[TagUpdateUserPayedFees] = "TagUpdateUserPayedFees"
	TagString[TagUpdateUserCollectedRewards] = "TagUpdateUserCollectedRewards"
	TagString[TagAuthorizerBurn] = "TagAuthorizerBurn"
	TagString[TagAddBurnTicket] = "TagAddBurnTicket"
	TagString[TagKillProvider] = "TagKillProvider"
	TagString[TagShutdownProvider] = "TagShutdownProvider"
	TagString[TagInsertReadpool] = "TagInsertReadpool"
	TagString[TagUpdateReadpool] = "TagUpdateReadpool"
	TagString[NumberOfTags] = "invalid"
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
