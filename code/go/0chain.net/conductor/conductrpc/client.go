package conductrpc

import (
	"net/rpc"
	"sync"

	"0chain.net/conductor/config"
)

// Client of the conductor RPC server.
type Client struct {
	address string      // RPC server address
	client  *rpc.Client // RPC client

	mutex sync.RWMutex // state mutex
	state *State       // current client state
}

// NewClient creates new client will be interacting
// with server with given address.
func NewClient(address string) (c *Client, err error) {
	if address, err = Host(address); err != nil {
		return
	}
	c = new(Client)
	if c.client, err = rpc.Dial("tcp", address); err != nil {
		return nil, err
	}
	c.address = address
	return
}

func (c *Client) dial() (err error) {
	c.client, err = rpc.Dial("tcp", c.address)
	return
}

// Address of RPC server.
func (c *Client) Address() string {
	return c.address
}

//
// miner SC RPC
//

func (c *Client) Phase(phase *PhaseEvent) (err error) {
	err = c.client.Call("Server.Phase", phase, &struct{}{})
	if err == rpc.ErrShutdown {
		if err = c.dial(); err != nil {
			return
		}
		err = c.client.Call("Server.Phase", phase, &struct{}{})
	}
	return
}

// ViewChange notification.
func (c *Client) ViewChange(viewChange *ViewChangeEvent) (err error) {
	err = c.client.Call("Server.ViewChange", viewChange, &struct{}{})
	if err == rpc.ErrShutdown {
		if err = c.dial(); err != nil {
			return
		}
		err = c.client.Call("Server.ViewChange", viewChange, &struct{}{})
	}
	return
}

// AddMiner notification.
func (c *Client) AddMiner(add *AddMinerEvent) (err error) {
	err = c.client.Call("Server.AddMiner", add, &struct{}{})
	if err == rpc.ErrShutdown {
		if err = c.dial(); err != nil {
			return
		}
		err = c.client.Call("Server.AddMiner", add, &struct{}{})
	}
	return
}

// AddSharder notification.
func (c *Client) AddSharder(add *AddSharderEvent) (err error) {
	err = c.client.Call("Server.AddSharder", add, &struct{}{})
	if err == rpc.ErrShutdown {
		if err = c.dial(); err != nil {
			return
		}
		err = c.client.Call("Server.AddSharder", add, &struct{}{})
	}
	return
}

// NodeReady phase notification.
func (c *Client) NodeReady(nodeID NodeID) (join bool, err error) {
	err = c.client.Call("Server.NodeReady", nodeID, &join)
	if err == rpc.ErrShutdown {
		if err = c.dial(); err != nil {
			return
		}
		err = c.client.Call("Server.NodeReady", nodeID, &join)
	}
	return
}

// Round notification.
func (c *Client) Round(re *RoundEvent) (err error) {
	err = c.client.Call("Server.Round", re, &struct{}{})
	if err == rpc.ErrShutdown {
		if err = c.dial(); err != nil {
			return
		}
		err = c.client.Call("Server.Round", re, &struct{}{})
	}
	return
}

// ContributeMPK phase notification.
func (c *Client) ContributeMPK(cmpke *ContributeMPKEvent) (err error) {
	err = c.client.Call("Server.ContributeMPK", cmpke, &struct{}{})
	if err == rpc.ErrShutdown {
		if err = c.dial(); err != nil {
			return
		}
		err = c.client.Call("Server.ContributeMPK", cmpke, &struct{}{})
	}
	return
}

// ShareOrSignsShares phase notification.
func (c *Client) ShareOrSignsShares(sosse *ShareOrSignsSharesEvent) (err error) {
	err = c.client.Call("Server.ShareOrSignsShares", sosse, &struct{}{})
	if err == rpc.ErrShutdown {
		if err = c.dial(); err != nil {
			return
		}
		err = c.client.Call("Server.ShareOrSignsShares", sosse, &struct{}{})
	}
	return
}

// State requests current client state using long polling strategy. E.g.
// when the state had updated, then the method returns.
func (c *Client) State(me NodeID) (state *State, err error) {
	err = c.client.Call("Server.State", me, &state)
	for err == rpc.ErrShutdown {
		err = c.client.Call("Server.State", me, &state)
	}
	return
}

//
// state (long polling)
//

// State is current node state.
type State struct {
	// Nodes maps NodeID -> NodeName.
	Nodes map[NodeID]NodeName

	IsMonitor  bool // send monitor events (round, phase, etc)
	Lock       bool // node locked
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

	// internal, used by RPC server
	counter int
}

func (s *State) Name(id NodeID) NodeName {
	return s.Nodes[id] // id -> name (or empty string)
}

func (s *State) IsSendShareFor(id NodeID) bool {
	var name, ok = s.Nodes[id]
	if !ok {
		return false
	}
	return s.Shares == nil || isInList(s.Shares.Good, id)
}

func (e *Entity) IsSendBadShareFor(id string) bool {
	var name, ok = s.Nodes[id]
	if !ok {
		return false
	}
	return s.Shares == nil || isInList(s.Shares.Bad, id)
}
