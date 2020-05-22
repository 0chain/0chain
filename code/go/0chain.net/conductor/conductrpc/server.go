package conductrpc

import (
	"sync"

	"github.com/valyala/gorpc"

	"0chain.net/conductor/config"
)

func init() {
	gorpc.RegisterType(NodeID(""))
	gorpc.RegisterType(ViewChangeEvent{})
	gorpc.RegisterType(PhaseEvent{})
}

// type aliases
type (
	NodeID    = config.NodeID
	Round     = config.Round
	Phase     = config.Phase
	RoundName = config.RoundName
)

// ViewChangeEvent represents view change information.
type ViewChangeEvent struct {
	Node     NodeID   // node that sends the VC
	Round    Round    // view change round
	Miners   []NodeID // magic block miners
	Sharders []NodeID // magic block sharders
}

// PhaseEvent represents phase switching.
type PhaseEvent struct {
	NodeID NodeID //
	Phase  Phase  //
}

// known locks
const (
	Locked   = false // should wait
	Unlocked = true  // can join
)

type nodeLock struct {
	lock    bool //
	counter int  //
}

type Server struct {
	server *gorpc.Server
	disp   *gorpc.Dispatcher

	// server events

	// onViewChange occurs where BC made VC (round == view change round)
	onViewChange chan ViewChangeEvent
	// onPhase occurs for every phase change
	onPhase chan PhaseEvent
	// onAddMiner occurs where miner SC proceed add_miner function
	onAddMiner chan NodeID
	// onAddSharder occurs where miner SC proceed add_sharder function
	onAddSharder chan NodeID

	// onNodeReady used by miner/sharder to notify the server that the node
	// has started and ready to register (if needed) in miner SC and start
	// it work. E.g. the node has started and waits the conductor to enter BC.
	onNodeReady chan NodeID

	// add / lock  miner / sharder
	mutex sync.Mutex
	locks map[NodeID]*nodeLock // expected miner/sharder -> locked/unlocked

	quitOnce sync.Once
	quit     chan struct{}
}

// NewServer Conductor RPC server.
func NewServer(address string) (s *Server) {
	s = new(Server)
	s.quit = make(chan struct{})

	// without a buffer
	s.onViewChange = make(chan ViewChangeEvent)
	s.onPhase = make(chan PhaseEvent)
	s.onAddMiner = make(chan NodeID)
	s.onAddSharder = make(chan NodeID)
	s.onNodeReady = make(chan NodeID)

	s.disp = gorpc.NewDispatcher()
	s.disp.AddFunc("onViewChange", s.onViewChangeHandler)
	s.disp.AddFunc("onPhase", s.onPhaseHandler)
	s.disp.AddFunc("onAddMiner", s.onAddMinerHandler)
	s.disp.AddFunc("onAddSharder", s.onAddSharderHandler)
	s.disp.AddFunc("onNodeReady", s.onNodeReadyHandler)

	s.server = gorpc.NewTCPServer(address, s.disp.NewHandlerFunc())
	return
}

//
// add/lock miner/sharder
//

// AddNode adds miner of sharder and, optionally, locks it.
func (s *Server) AddNode(nodeID NodeID, lock bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.locks[nodeID] = &nodeLock{counter: 0, lock: lock}
}

// UnlockNode unlocks a miner.
func (s *Server) UnlockNode(nodeID NodeID) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.locks[nodeID] = &nodeLock{counter: 0, lock: Unlocked}
}

func (s *Server) nodeLock(nodeID NodeID) (lock, ok bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var nl *nodeLock
	nl, ok = s.locks[nodeID]
	if !ok {
		return // false, false
	}
	return nl.lock, ok // lock, true
}

// events handling

// OnViewChange events channel. The event occurs where
// BC made VC (round == view change round).
func (s *Server) OnViewChange() chan ViewChangeEvent {
	return s.onViewChange
}

// OnPhase events channel. The event occurs where miner SC changes its phase.
func (s *Server) OnPhase() chan PhaseEvent {
	return s.onPhase
}

// OnAddMiner events channel. The event occurs
// where miner SC proceed add_miner function.
func (s *Server) OnAddMiner() chan NodeID {
	return s.onAddMiner
}

// OnAddSharder events channel. The event occurs
// where miner SC proceed add_sharder function.
func (s *Server) OnAddSharder() chan NodeID {
	return s.onAddSharder
}

// OnNodeReady used by nodes to notify the server that the node has started
// and ready to register (if needed) in miner SC and start it work. E.g.
// the node has started and waits the conductor to enter BC.
func (s *Server) OnNodeReady() chan NodeID {
	return s.onNodeReady
}

//
// handlers
//

func (s *Server) onViewChangeHandler(viewChange ViewChangeEvent) {
	select {
	case s.onViewChange <- viewChange:
	case <-s.quit:
	}
}

func (s *Server) onPhaseHandler(phase PhaseEvent) {
	select {
	case s.onPhase <- phase:
	case <-s.quit:
	}
}

func (s *Server) onAddMinerHandler(minerID NodeID) {
	select {
	case s.onAddMiner <- minerID:
	case <-s.quit:
	}
}

func (s *Server) onAddSharderHandler(sharderID NodeID) {
	select {
	case s.onAddSharder <- sharderID:
	case <-s.quit:
	}
}

func (s *Server) onNodeReadyHandler(nodeID NodeID) (join bool) {

	var ok bool
	if join, ok = s.nodeLock(nodeID); ok {
		return // don't trigger onNodeReady twice or more times
	}

	select {
	case s.onNodeReady <- nodeID:
	case <-s.quit:
	}
	return
}

//
// flow
//

// Serve starts the server blocking.
func (s *Server) Serve() (err error) {
	return s.server.Serve()
}

// Close the server waiting.
func (s *Server) Close() {
	s.quitOnce.Do(func() { close(s.quit) })
	s.server.Stop()
}
