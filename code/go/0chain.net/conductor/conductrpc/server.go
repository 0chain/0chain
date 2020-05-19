package conductrpc

import (
	"sync"

	"github.com/valyala/gorpc"
)

func init() {
	gorpc.RegisterType(MinerID(""))
	gorpc.RegisterType(SharderID(""))
	gorpc.RegisterType(ViewChange{})
	gorpc.RegisterType(Lock(false))
}

// common types
type (
	MinerID   string
	SharderID string
)

// ViewChange represents view change information.
type ViewChange struct {
	Round    int64       // view change round
	Miners   []MinerID   // magic block miners
	Sharders []SharderID // magic block sharders
}

// Lock miner or sharder to join BC.
type Lock bool

// known locks
const (
	Locked   = false // should wait
	Unlocked = true  // can join
)

type Server struct {
	server *gorpc.Server
	disp   *gorpc.Dispatcher

	// server events

	// onViewChange occurs where BC made VC (round == view change round)
	onViewChange chan ViewChange
	// onAddMiner occurs where miner SC proceed add_miner function
	onAddMiner chan MinerID
	// onAddSharder occurs where miner SC proceed add_sharder function
	onAddSharder chan SharderID

	// onMinerReady used by miners to notify the server that miner has started
	// and ready to register (if needed) in miner SC and start it work. E.g.
	// the miner has started and waits the conductor to enter BC.
	onMinerReady chan MinerID
	// onSharderReady used by sharders to notify the server that sharder has
	// started and ready to register (if needed) in miner SC and start it work.
	//  E.g. the sharder has started and waits the conductor to enter BC.
	onSharderReady chan SharderID

	// add / lock  miner / sharder
	mutex    sync.Mutex
	miners   map[MinerID]Lock   // expected miner -> unlocked
	sharders map[SharderID]Lock // expected sharder -> unlocked

	quitOnce sync.Once
	quit     chan struct{}
}

// NewServer Conductor RPC server.
func NewServer(address string) (s *Server) {
	s = new(Server)
	s.quit = make(chan struct{})

	// without a buffer
	s.onViewChange = make(chan ViewChange)
	s.onAddMiner = make(chan MinerID)
	s.onAddSharder = make(chan SharderID)
	s.onMinerReady = make(chan MinerID)
	s.onSharderReady = make(chan SharderID)

	s.disp = gorpc.NewDispatcher()
	s.disp.AddFunc("onViewChange", s.onViewChangeHandler)
	s.disp.AddFunc("onAddMiner", s.onAddMinerHandler)
	s.disp.AddFunc("onAddSharder", s.onAddSharderHandler)
	s.disp.AddFunc("onMinerReady", s.onMinerReadyHandler)
	s.disp.AddFunc("onSharderReady", s.onSharderReadyHandler)

	s.server = gorpc.NewTCPServer(address, s.disp.NewHandlerFunc())
	return
}

//
// add/lock miner/sharder
//

// AddMiner adds and, optionally, locks expected miner.
func (s *Server) AddMiner(minerID MinerID, lock Lock) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.miners[minerID] = lock
}

// AddSharder adds and, optionally, locks expected sharder.
func (s *Server) AddSharder(sharderID SharderID, lock Lock) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.sharders[sharderID] = lock
}

// UnlockMiner unlocks a miner.
func (s *Server) UnlockMiner(minerID MinerID) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.miners[minerID] = true // unlocked
}

// UnlockSharder unlocks a sharder.
func (s *Server) UnlockSharder(sharderID SharderID) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.sharders[sharderID] = true // unlocked
}

func (s *Server) minerLock(minerID MinerID) (lock Lock) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.miners[minerID]
}

func (s *Server) sharderLock(sharderID SharderID) (lock Lock) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.sharders[sharderID]
}

// events handling

// OnViewChange events channel. The event occurs where
// BC made VC (round == view change round).
func (s *Server) OnViewChange() chan ViewChange {
	return s.onViewChange
}

// OnAddMiner events channel. The event occurs
// where miner SC proceed add_miner function.
func (s *Server) OnAddMiner() chan MinerID {
	return s.onAddMiner
}

// OnAddSharder events channel. The event occurs
// where miner SC proceed add_sharder function.
func (s *Server) OnAddSharder() chan SharderID {
	return s.onAddSharder
}

// OnMinerReady used by miners to notify the server that miner has started
// and ready to register (if needed) in miner SC and start it work. E.g.
// the miner has started and waits the conductor to enter BC.
func (s *Server) OnMinerReady() chan MinerID {
	return s.onMinerReady
}

// OnSharderReady used by sharders to notify the server that sharder has
// started and ready to register (if needed) in miner SC and start it work.
//  E.g. the sharder has started and waits the conductor to enter BC.
func (s *Server) OnSharderReady() chan SharderID {
	return s.onSharderReady
}

//
// handlers
//

func (s *Server) onViewChangeHandler(viewChange ViewChange) {
	select {
	case s.onViewChange <- viewChange:
	case <-s.quit:
	}
}

func (s *Server) onAddMinerHandler(minerID MinerID) {
	select {
	case s.onAddMiner <- minerID:
	case <-s.quit:
	}
}

func (s *Server) onAddSharderHandler(sharderID SharderID) {
	select {
	case s.onAddSharder <- sharderID:
	case <-s.quit:
	}
}

func (s *Server) onMinerReadyHandler(minerID MinerID) (join Lock) {
	select {
	case s.onMinerReady <- minerID:
		return s.minerLock(minerID)
	case <-s.quit:
	}
	return
}

func (s *Server) onSharderReadyHandler(sharderID SharderID) (join Lock) {
	select {
	case s.onSharderReady <- sharderID:
		return s.sharderLock(sharderID)
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
