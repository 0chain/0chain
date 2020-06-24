package conductrpc

import (
	"0chain.net/chaincore/node"
	"0chain.net/core/encryption"

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
	VRFS                        *config.Bad
	RoundTimeout                *config.Bad
	CompetingBlock              *config.Bad
	SignOnlyCompetingBlocks     *config.Bad
	DoubleSpendTransaction      *config.Bad
	WrongBlockSignHash          *config.Bad
	WrongBlockSignKey           *config.Bad
	WrongBlockHash              *config.Bad
	VerificationTicketGroup     *config.Bad
	WrongVerificationTicketHash *config.Bad
	WrongVerificationTicketKey  *config.Bad
	WrongNotarizedBlockHash     *config.Bad
	WrongNotarizedBlockKey      *config.Bad
	NotarizeOnlyCompetingBlock  *config.Bad
	NotarizedBlock              *config.Bad
	// Byzantine blockchain sharders
	FinalizedBlock    *config.Bad
	MagicBlock        *config.Bad
	VerifyTransaction *config.Bad
	// Byzantine View Change
	MPK        *config.Bad
	Shares     *config.Bad
	Signatures *config.Bad
	Publish    *config.Bad

	// persistent (persistent fields)
	signature encryption.SignatureScheme
}

func (s *State) Update(prev *State) {
	if prev != nil && prev.signature != nil {
		s.signature = prev.signature // keep signature scheme unchanged
		return
	}
	s.signature = encryption.NewBLS0ChainScheme()
	var err error
	if err = s.signature.GenerateKeys(); err != nil {
		panic(err)
	}
}

// Sign by internal ("wrong") secret key generated randomly once client created.
func (s *State) Sign(hash string) (sign string, err error) {
	return s.signature.Sign(hash)
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

type IsGoodBader interface {
	IsGood(state config.Namer, id string) bool
	IsBad(state config.Namer, id string) bool
}

// Split nodes list by given IsGoodBader.
func (s *State) Split(igb IsGoodBader, nodes []*node.Node) (
	good, bad []*node.Node) {

	for _, n := range nodes {
		if igb.IsBad(s, n.GetKey()) {
			bad = append(bad, n)
		} else if igb.IsGood(s, n.GetKey()) {
			good = append(good, n)
		}
	}
	return
}

type IsByer interface {
	IsBy(state config.Namer, id string) bool
}

// Filter return IsBy nodes only.
func (s *State) Filter(ib IsByer, nodes []*node.Node) (rest []*node.Node) {
	for _, n := range nodes {
		if ib.IsBy(s, n.GetKey()) {
			rest = append(rest, n)
		}
	}
	return
}
