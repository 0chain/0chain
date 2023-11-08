package conductrpc

import (
	"0chain.net/conductor/config"
	"0chain.net/conductor/config/cases"
)

//
// state (long polling)
//

// State is current node state.
type State struct {
	// Nodes maps NodeID -> NodeName.
	Nodes map[NodeID]NodeName

	IsMonitor  bool // send monitor events (round, phase, etc)
	IsLock     bool // node locked
	IsRevealed bool // revealed shares
	// Failure emulation
	GeneratorsFailureRoundNumber Round // all generators fail on start of this round
	// Byzantine state. Below, if a value is nil, then node behaves as usual
	// for it.
	//
	// Byzantine blockchain
	VRFS                                    *config.Bad
	RoundTimeout                            *config.Bad
	CompetingBlock                          *config.Bad
	SignOnlyCompetingBlocks                 *config.Bad
	DoubleSpendTransaction                  *config.Bad
	DoubleSpendTransactionHash              string // internal variable to ignore this transaction in ChainHasTransaction()
	WrongBlockSignHash                      *config.Bad
	WrongBlockSignKey                       *config.Bad
	WrongBlockHash                          *config.Bad
	WrongBlockRandomSeed                    *config.Bad
	WrongBlockDDoS                          *config.Bad
	VerificationTicketGroup                 *config.Bad
	WrongVerificationTicketHash             *config.Bad
	WrongVerificationTicketKey              *config.Bad
	WrongNotarizedBlockHash                 *config.Bad
	WrongNotarizedBlockKey                  *config.Bad
	NotarizeOnlyCompetingBlock              *config.Bad
	NotarizedBlock                          *config.Bad
	CollectVerificationTicketsWhenMissedVRF *config.CollectVerificationTicketsWhenMissedVRF
	// Byzantine blockchain sharders
	FinalizedBlock    *config.Bad
	MagicBlock        *config.Bad
	VerifyTransaction *config.Bad
	// Byzantine View Change
	MPK        *config.Bad
	Shares     *config.Bad
	Signatures *config.Bad
	Publish    *config.Bad

	ExtendNotNotarisedBlock               *cases.NotNotarisedBlockExtension
	SendDifferentBlocksFromFirstGenerator *cases.SendDifferentBlocksFromFirstGenerator
	SendDifferentBlocksFromAllGenerators  *cases.SendDifferentBlocksFromAllGenerators
	BreakingSingleBlock                   *cases.BreakingSingleBlock
	SendInsufficientProposals             *cases.SendInsufficientProposals
	VerifyingNonExistentBlock             *cases.VerifyingNonExistentBlock
	NotarisingNonExistentBlock            *cases.NotarisingNonExistentBlock
	ResendProposedBlock                   *cases.ResendProposedBlock
	ResendNotarisation                    *cases.ResendNotarisation
	BadTimeoutVRFS                        *cases.BadTimeoutVRFS
	HalfNodesDown                         *cases.HalfNodesDown
	BlockStateChangeRequestor             *cases.BlockStateChangeRequestor
	MinerNotarisedBlockRequestor          *cases.MinerNotarisedBlockRequestor
	FBRequestor                           *cases.FBRequestor
	MissingLFBTicket                      *cases.MissingLFBTickets
	RoundHasFinalizedConfig               *cases.RoundHasFinalized
	RoundRandomSeed                       *cases.RoundRandomSeed

	LockNotarizationAndSendNextRoundVRF *config.LockNotarizationAndSendNextRoundVRF

	// Blobbers related states
	StorageTree             *config.Bad // blobber sends bad files/tree responses
	ValidatorProof          *config.Bad // blobber sends invalid proof to validators
	Challenges              *config.Bad // blobber ignores challenges
	BlobberList             *config.BlobberList
	BlobberDownload         *config.BlobberDownload
	BlobberUpload           *config.BlobberUpload
	BlobberDelete           *config.BlobberDelete
	GenerateAllChallenges   bool
	GenerateChallenge       *config.GenerateChallege
	StopWMCommit            bool
	StopChallengeGeneration bool
	BlobberCommittedWM      bool
	GetFileMetaRoot         bool
	FailRenameCommit        []NodeID
	FailUploadCommit        []NodeID
	// Validators related states
	CheckChallengeIsValid *cases.CheckChallengeIsValid
	AdversarialValidator  *config.AdversarialValidator

	// Authorizers related states
	AdversarialAuthorizer *config.AdversarialAuthorizer

	ServerStatsCollectorEnabled        bool
	ClientStatsCollectorEnabled        bool
	NotifyOnBlockGeneration            bool
	NotifyOnValidationTicketGeneration bool
}

// Name returns NodeName by given NodeID.
func (s *State) Name(id NodeID) NodeName {
	return s.Nodes[id] // id -> name (or empty string)
}

func (s *State) copy() (cp *State) {
	cp = new(State)
	*cp = *s
	return

}

func (s *State) send(poll chan *State) {
	poll <- s.copy()
}

type IsGoodOrBad interface {
	IsGood(state config.Namer, id string) bool
	IsBad(state config.Namer, id string) bool
}

type IsBy interface {
	IsBy(state config.Namer, id string) bool
}
