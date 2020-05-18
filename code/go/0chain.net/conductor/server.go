package conductor

import (
	"sync"

	"github.com/valyala/gorpc"
)

func init() {
	gorpc.RegisterType(MinerID(""))
	gorpc.RegisterType(SharderID(""))
	gorpc.RegisterType(ViewChange{})
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

func (s *Server) onMinerReadyHandler(minerID MinerID) {
	select {
	case s.onMinerReady <- minerID:
	case <-s.quit:
	}
}

func (s *Server) onSharderReadyHandler(sharderID SharderID) {
	select {
	case s.onSharderReady <- sharderID:
	case <-s.quit:
	}
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
