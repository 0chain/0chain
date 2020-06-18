package conductrpc

import (
	"0chain.net/conductor/config"
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
	// Byzantine state. Below, if a value is nil, then node behaves as usual
	// for it.
	//
	// Byzantine blockchain
	VRFS                        *config.VRFS
	RoundTimeout                *config.RoundTimeout
	CompetingBlock              *config.CompetingBlock
	SignOnlyCompetingBlocks     *config.SignOnlyCompetingBlocks
	DoubleSpendTransaction      *config.DoubleSpendTransaction
	WrongBlockSignHash          *config.WrongBlockSignHash
	WrongBlockSignKey           *config.WrongBlockSignKey
	WrongBlockHash              *config.WrongBlockHash
	VerificationTicket          *config.VerificationTicket
	WrongVerificationTicketHash *config.WrongVerificationTicketHash
	WrongVerificationTicketKey  *config.WrongVerificationTicketKey
	WrongNotarizedBlockHash     *config.WrongNotarizedBlockHash
	WrongNotarizedBlockKey      *config.WrongNotarizedBlockKey
	NotarizeOnlyCompetingBlock  *config.NotarizeOnlyCompetingBlock
	NotarizedBlock              *config.NotarizedBlock
	// Byzantine View Change
	MPK        *config.MPK
	Shares     *config.Shares
	Signatures *config.Signatures
	Publish    *config.Publish
}

// Name returns NodeName by given NodeID.
func (s *State) Name(id NodeID) NodeName {
	return s.Nodes[id] // id -> name (or empty string)
}

func (s *State) copy() (cp *State) {
	cp = new(State)
	(*cp) = (*s)
	return

}

func (s *State) send(poll chan *State) {
	go func(state *State) {
		poll <- state
	}(s.copy())
}
