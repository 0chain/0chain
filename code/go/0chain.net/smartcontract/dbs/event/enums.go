package event

type (
	EventType int
	EventTag  int
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
	MinerTable 		ProviderTable 	= "miners"
	SharderTable 	ProviderTable	= "sharders"
	BlobberTable 	ProviderTable	= "blobbers"
	AuthorizerTable ProviderTable   = "authorizers"
	ValidatorTable 	ProviderTable   = "validators"
)

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
	TagUpdateBlobberTotalUnStake
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
	TagUpdateValidatorUnStakeTotal
	TagUpdateMinerTotalStake
	TagUpdateMinerTotalUnStake
	TagUpdateSharderTotalStake
	TagUpdateSharderTotalUnStake
	TagUpdateAuthorizerTotalStake
	TagUpdateAuthorizerTotalUnStake
	TagUniqueAddress
	TagMinerHealthCheck
	TagSharderHealthCheck
	TagBlobberHealthCheck
	TagAuthorizerHealthCheck
	TagValidatorHealthCheck
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
	TagString[TagUpdateBlobberAllocatedHealth] = "TagUpdateBlobberAllocatedHealth"
	TagString[TagUpdateBlobberTotalStake] = "TagUpdateBlobberTotalStake"
	TagString[TagUpdateBlobberTotalUnStake] = "TagUpdateBlobberTotalUnStake"
	TagString[TagUpdateBlobberTotalOffers] = "TagUpdateBlobberTotalOffers"
	TagString[TagDeleteBlobber] = "TagDeleteBlobber"
	TagString[TagAddAuthorizer] = "TagAddAuthorizer"
	TagString[TagUpdateAuthorizer] = "TagUpdateAuthorizer"
	TagString[TagDeleteAuthorizer] = "TagDeleteAuthorizer"
	TagString[TagAddTransactions] = "TagAddTransactions"
	TagString[TagAddOrOverwriteUser] = "TagAddOrOverwriteUser"
	TagString[TagAddWriteMarker] = "TagAddWriteMarker"
	TagString[TagAddBlock] = "TagAddBlock"
	TagString[TagFinalizeBlock] = "TagFinalizeBlock"
	TagString[TagAddOrOverwiteValidator] = "TagAddOrOverwiteValidator"
	TagString[TagUpdateValidator] = "TagUpdateValidator"
	TagString[TagAddReadMarker] = "TagAddReadMarker"
	TagString[TagAddOrOverwriteMiner] = "TagAddOrOverwriteMiner"
	TagString[TagUpdateMiner] = "TagUpdateMiner"
	TagString[TagDeleteMiner] = "TagDeleteMiner"
	TagString[TagAddOrOverwriteSharder] = "TagAddOrOverwriteSharder"
	TagString[TagUpdateSharder] = "TagUpdateSharder"
	TagString[TagDeleteSharder] = "TagDeleteSharder"
	TagString[TagAddOrOverwriteCurator] = "TagAddOrOverwriteCurator"
	TagString[TagRemoveCurator] = "TagRemoveCurator"
	TagString[TagStakePoolReward] = "TagStakePoolReward"
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
	TagString[TagSendTransfer] = "TagSendTransfer"
	TagString[TagReceiveTransfer] = "TagReceiveTransfer"
	TagString[TagLockStakePool] = "TagLockStakePool"
	TagString[TagUnlockStakePool] = "TagUnlockStakePool"
	TagString[TagLockWritePool] = "TagLockWritePool"
	TagString[TagUnlockWritePool] = "TagUnlockWritePool"
	TagString[TagLockReadPool] = "TagLockReadPool"
	TagString[TagUnlockReadPool] = "TagUnlockReadPool"
	TagString[TagToChallengePool] = "TagToChallengePool"
	TagString[TagFromChallengePool] = "TagFromChallengePool"
	TagString[TagAddMint] = "TagAddMint"
	TagString[TagBurn] = "TagBurn"
	TagString[TagAllocValueChange] = "TagAllocValueChange"
	TagString[TagAllocBlobberValueChange] = "TagAllocBlobberValueChange"
	TagString[TagUpdateBlobberOpenChallenges] = "TagUpdateBlobberOpenChallenges"
	TagString[TagUpdateValidatorStakeTotal] = "TagUpdateValidatorStakeTotal"
	TagString[TagUpdateValidatorUnStakeTotal] = "TagUpdateValidatorUnStakeTotal"
	TagString[TagUpdateMinerTotalUnStake] = "TagUpdateMinerTotalUnStake"
	TagString[TagUpdateSharderTotalUnStake] = "TagUpdateSharderTotalUnStake"
	TagString[TagUpdateAuthorizerTotalUnStake] = "TagUpdateAuthorizerTotalUnStake"
	TagString[TagUniqueAddress] = "TagUniqueAddress"
	TagString[TagMinerHealthCheck] = "TagMinerHealthCheck"
	TagString[TagSharderHealthCheck] =  "TagSharderHealthCheck"
	TagString[TagBlobberHealthCheck] =  "TagBlobberHealthCheck"
	TagString[TagAuthorizerHealthCheck] = "TagAuthorizerHealthCheck"
	TagString[TagValidatorHealthCheck] = "TagValidatorHealthCheck"
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
