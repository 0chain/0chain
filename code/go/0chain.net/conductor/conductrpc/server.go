package conductrpc

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"strings"
	"sync"

	"0chain.net/conductor/cases"
	"0chain.net/conductor/conductrpc/stats"
	"0chain.net/conductor/config"
	"0chain.net/conductor/types"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

var ErrShutdown = errors.New("server shutdown")

// type aliases
type (
	NodeID    = config.NodeID
	NodeName  = config.NodeName
	Round     = config.Round
	Phase     = config.Phase
	RoundName = config.RoundName
	Number    = config.Number
)

// ViewChangeEvent represents view change information.
type ViewChangeEvent struct {
	Sender   NodeName   // node that sends the VC
	Round    Round      // view change round
	Number   Number     // magic block number
	Miners   []NodeName // magic block miners
	Sharders []NodeName // magic block sharders
}

// PhaseEvent represents phase switching.
type PhaseEvent struct {
	Sender NodeName //
	Phase  Phase    //
}

// AddMinerEvent in miner SC.
type AddMinerEvent struct {
	Sender NodeName // event emitter
	Miner  NodeName // the added miner
}

// AddSharderEvent in miner SC.
type AddSharderEvent struct {
	Sender  NodeName // event emitter
	Sharder NodeName // the added sharder
}

// AddBlobberEvent in miner SC.
type AddBlobberEvent struct {
	Sender  NodeName // event emitter
	Blobber NodeName // the added blobber
}

// AddValidatorEvent in miner SC.
type AddValidatorEvent struct {
	Sender  NodeName // event emitter
	Validator NodeName // the added validator
}

// AddAuthorizerEvent in miner SC.
type AddAuthorizerEvent struct {
	Sender     NodeName // event emitter
	Authorizer NodeName // the added blobber
}

// SharderKeepEvent in miner SC.
type SharderKeepEvent struct {
	Sender  NodeName // event emitter
	Sharder NodeName // the sharder to keep
}

// Round proceed in pay_fees of Miner SC.
type RoundEvent struct {
	Sender NodeName // event emitter
	Round  Round    // round number
}

// ContributeMPKEvent where a miner successfully sent its contribution.
type ContributeMPKEvent struct {
	Sender NodeName // event emitter
	Miner  NodeName // miner that contributes
}

// ShareOrSignsSharesEvent where a miner successfully sent its share or sign
type ShareOrSignsSharesEvent struct {
	Sender NodeName // event emitter
	Miner  NodeName // miner that sends
}

type nodeState struct {
	state   *State      // current state
	poll    chan *State // update stat
	counter int         // used when node appears
}

type ValidtorTicket struct {
	ValidatorId string
}

type AggregateMessage struct {
	ProviderType types.ProviderType
	ProviderId string
	Values types.Aggregate
}

type NodeConfig struct {
	Version int
	Map map[string]interface{}
	mutex *sync.RWMutex
}

func (nc *NodeConfig) Get(key string) map[string]interface{} {
	nc.mutex.RLock()
	defer nc.mutex.Unlock()

	return nc.Map
}

func (nc *NodeConfig) Update(config map[string]interface{}) {
	nc.mutex.Lock()
	defer nc.mutex.Unlock()

	for k, v := range config {
		nc.Map[k] = v
	}

	nc.Version++
}

func NewNodeConfig(config map[string]interface{}) *NodeConfig {
	return &NodeConfig{
		Version: 1,
		Map: config,
		mutex: &sync.RWMutex{},
	}
}

type Server struct {
	server  *rpc.Server
	address string
	l       net.Listener

	// server events

	// onViewChange occurs where BC made VC (round == view change round)
	onViewChange chan *ViewChangeEvent
	// onPhase occurs for every phase change
	onPhase chan *PhaseEvent
	// onAddMiner occurs where miner SC proceed add_miner function
	onAddMiner chan *AddMinerEvent
	// onAddSharder occurs where miner SC proceed add_sharder function
	onAddSharder chan *AddSharderEvent
	// onAddBlobber occurs where blobber added in storage SC
	onAddBlobber chan *AddBlobberEvent
	// onAddValidator occurs where blobber added in storage SC
	onAddValidator chan *AddValidatorEvent
	// onAddAuthorizer occurs where authorizer added in storage SC
	onAddAuthorizer chan *AddAuthorizerEvent
	// onSharderKeep occurs where miner SC proceed sharder_keep function
	onSharderKeep chan *SharderKeepEvent
	// onSharderBlock occurs when the sharder is send an add_block request
	onSharderBlock chan *stats.BlockFromSharder
	// onValidatorTicket occurs when the validator sends a ticket
	onValidatorTicket chan *ValidtorTicket
	// onAggregate occurs when the sharder sends an aggregate if the aggregates are being monitored
	onAggregate chan *AggregateMessage

	// onNodeReady used by miner/sharder to notify the server that the node
	// has started and ready to register (if needed) in miner SC and start
	// it work. E.g. the node has started and waits the conductor to enter BC.
	onNodeReady               chan NodeName
	onRoundEvent              chan *RoundEvent
	onContributeMPKEvent      chan *ContributeMPKEvent
	onShareOrSignsSharesEvent chan *ShareOrSignsSharesEvent
	// onChallengeGeneration will notify server that challenge has been generated
	// May need to add fields in struct rather than only struct{}
	onChallengeGeneration chan string
	onBlobberCommit       chan string
	onChallengeStatus     chan map[string]interface{}
	onGettingFileMetaRoot chan map[string]string
	CurrentTest           cases.TestCase

	magicBlock string
	// nodes lock/unlock/shares sending (send only, send bad)
	mutex sync.RWMutex
	nodes map[NodeName]*nodeState

	// node id -> node name mapping
	names map[NodeID]NodeName
	nodeCustomConfig map[NodeID]*NodeConfig

	NodesServerStatsCollector *stats.NodesServerStats
	NodesClientStatsCollector *stats.NodesClientStats

	quitOnce sync.Once
	quit     chan struct{}
}

// NewServer Conductor RPC server.
func NewServer(address string, names map[NodeID]NodeName) (s *Server, err error) {
	s = &Server{
		quit:                      make(chan struct{}),
		names:                     names,
		nodeCustomConfig: 		   make(map[config.NodeID]*NodeConfig),
		onViewChange:              make(chan *ViewChangeEvent, 10),
		onPhase:                   make(chan *PhaseEvent, 10),
		onAddMiner:                make(chan *AddMinerEvent, 10),
		onAddSharder:              make(chan *AddSharderEvent, 10),
		onAddBlobber:              make(chan *AddBlobberEvent, 10),
		onAddAuthorizer:           make(chan *AddAuthorizerEvent, 10),
		onSharderKeep:             make(chan *SharderKeepEvent, 10),
		onSharderBlock: 		   make(chan *stats.BlockFromSharder),
		onValidatorTicket: 		   make(chan *ValidtorTicket, 10),
		onNodeReady:               make(chan NodeName, 10),
		onRoundEvent:              make(chan *RoundEvent, 100),
		onContributeMPKEvent:      make(chan *ContributeMPKEvent, 10),
		onShareOrSignsSharesEvent: make(chan *ShareOrSignsSharesEvent, 10),
		onChallengeGeneration:     make(chan string, 1),
		onBlobberCommit:           make(chan string, 1),
		onChallengeStatus:         make(chan map[string]interface{}, 1),
		onGettingFileMetaRoot:     make(chan map[string]string, 1),
		nodes:                     make(map[NodeName]*nodeState),
		address:                   address,
		server:                    rpc.NewServer(),
	}

	if err = s.server.Register(s); err != nil {
		return nil, err
	}

	return
}

func (s *Server) Serve() (err error) {
	var l net.Listener
	if l, err = net.Listen("tcp", s.address); err != nil {
		return
	}
	go s.server.Accept(l)
	return
}

//
// add/lock miner/sharder
//

// AddNode adds miner of sharder and, optionally, locks it.
func (s *Server) AddNode(name NodeName, lock bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var monitor bool

	// if already added (by SetMonitor, for example)
	if ns, ok := s.nodes[name]; ok {
		monitor = ns.state.IsMonitor
	}

	var ns = &nodeState{
		state: &State{
			IsMonitor: monitor,
			Nodes:     s.names,
			IsLock:    lock,
		},
		poll:    make(chan *State, 10),
		counter: 0,
	}

	ns.state.send(ns.poll) // initial state sending
	s.nodes[name] = ns
}

func (s *Server) SetNodeConfig(id NodeID, config map[string]interface{}) (err error) {
	nc, ok := s.nodeCustomConfig[id]
	if !ok {
		s.nodeCustomConfig[id] = NewNodeConfig(config)
		return
	}
	
	nc.Update(config)
	return
}

// not for updating
func (s *Server) nodeState(name NodeName) (ns *nodeState, err error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var ok bool
	if ns, ok = s.nodes[name]; !ok {
		return nil, fmt.Errorf("(node state) unexpected node: %s", name)
	}
	ns.counter++
	return
}

type UpdateStateFunc func(state *State)

func (s *Server) UpdateState(name NodeName, update UpdateStateFunc) (
	err error) {

	s.mutex.Lock()
	defer s.mutex.Unlock()

	var n, ok = s.nodes[name]
	if !ok {
		return fmt.Errorf("(update state) unexpected node: %s", name)
	}

	update(n.state) // update
	n.state.send(n.poll)
	return
}

func (s *Server) UpdateStates(names []NodeName, update UpdateStateFunc) (
	err error) {

	for _, name := range names {
		if err := s.UpdateState(name, update); err != nil {
			logging.Logger.Warn("update state failed", zap.Error(err))
		}
	}
	return
}

func (s *Server) UpdateAllStates(update UpdateStateFunc) (
	err error) {

	for name := range s.nodes {
		if err := s.UpdateState(name, update); err != nil {
			logging.Logger.Warn("update state failed", zap.Error(err))
		}
	}
	return
}

// events handling

// OnViewChange events channel. The event occurs where
// BC made VC (round == view change round).
func (s *Server) OnViewChange() chan *ViewChangeEvent {
	return s.onViewChange
}

// OnPhase events channel. The event occurs where miner SC changes its phase.
func (s *Server) OnPhase() chan *PhaseEvent {
	return s.onPhase
}

// OnAddMiner events channel. The event occurs
// where miner SC proceed add_miner function.
func (s *Server) OnAddMiner() chan *AddMinerEvent {
	return s.onAddMiner
}

// OnAddSharder events channel. The event occurs
// where miner SC proceed add_sharder function.
func (s *Server) OnAddSharder() chan *AddSharderEvent {
	return s.onAddSharder
}

func (s *Server) OnAddBlobber() chan *AddBlobberEvent {
	return s.onAddBlobber
}

func (s *Server) OnAddValidator() chan *AddValidatorEvent {
	return s.onAddValidator
}

func (s *Server) OnAddAuthorizer() chan *AddAuthorizerEvent {
	return s.onAddAuthorizer
}

func (s *Server) OnSharderKeep() chan *SharderKeepEvent {
	return s.onSharderKeep
}

func (s *Server) OnSharderBlock() chan *stats.BlockFromSharder {
	return s.onSharderBlock
}

// OnNodeReady used by nodes to notify the server that the node has started
// and ready to register (if needed) in miner SC and start it work. E.g.
// the node has started and waits the conductor to enter BC.
func (s *Server) OnNodeReady() chan NodeName {
	return s.onNodeReady
}

func (s *Server) OnRound() chan *RoundEvent {
	return s.onRoundEvent
}

func (s *Server) OnContributeMPK() chan *ContributeMPKEvent {
	return s.onContributeMPKEvent
}

func (s *Server) OnShareOrSignsShares() chan *ShareOrSignsSharesEvent {
	return s.onShareOrSignsSharesEvent
}

func (s *Server) OnGenerateChallenge() chan string {
	return s.onChallengeGeneration
}

func (s *Server) OnBlobberCommit() chan string {
	return s.onBlobberCommit
}

func (s *Server) OnChallengeStatus() chan map[string]interface{} {
	return s.onChallengeStatus
}

func (s *Server) OnGettingFileMetaRoot() chan map[string]string {
	return s.onGettingFileMetaRoot
}

func (s *Server) OnValidatorTicket() chan *ValidtorTicket {
	return s.onValidatorTicket
}

func (s *Server) OnAggregate() chan *AggregateMessage {
	return s.onAggregate
}

func (s *Server) Nodes() map[config.NodeName]*nodeState {
	return s.nodes
}

// GetMinersNum returns current miners number.
func (s *Server) GetMinersNum() int {
	var minersNum int
	for nodeName, node := range s.nodes {
		if strings.Contains(string(nodeName), "miner") && node != nil {
			minersNum++
		}
	}
	return minersNum
}

//
// RPC functions/handlers
// Signature Fn(args *Any, Response *Any)error
// Exception of Any is channel. i.e channel can neither be argument nor response
//

func (s *Server) ViewChange(viewChange *ViewChangeEvent, _ *struct{}) (err error) {
	select {
	case s.onViewChange <- viewChange:
	case <-s.quit:
	}
	return
}

func (s *Server) Phase(phase *PhaseEvent, _ *struct{}) (err error) {
	select {
	case s.onPhase <- phase:
	case <-s.quit:
	}
	return
}

func (s *Server) AddMiner(add *AddMinerEvent, _ *struct{}) (err error) {
	select {
	case s.onAddMiner <- add:
	case <-s.quit:
	}
	return
}

func (s *Server) AddSharder(add *AddSharderEvent, _ *struct{}) (err error) {
	select {
	case s.onAddSharder <- add:
	case <-s.quit:
	}
	return
}

func (s *Server) AddBlobber(add *AddBlobberEvent, _ *struct{}) (err error) {
	select {
	case s.onAddBlobber <- add:
	case <-s.quit:
	}
	return
}

func (s *Server) AddAuthorizer(add *AddAuthorizerEvent, _ *struct{}) (err error) {
	select {
	case s.onAddAuthorizer <- add:
	case <-s.quit:
	}
	return
}

func (s *Server) SharderKeep(sk *SharderKeepEvent, _ *struct{}) (err error) {
	select {
	case s.onSharderKeep <- sk:
	case <-s.quit:
	}
	return
}

func (s *Server) SharderBlock(block *stats.BlockFromSharder, _ *struct{}) (err error) {
	select {
	case s.onSharderBlock <- block:
	case <- s.quit:
	}
	return
}

func (s *Server) Round(rnd *RoundEvent, _ *struct{}) (err error) {
	select {
	case s.onRoundEvent <- rnd:
	case <-s.quit:
	}
	return
}

func (s *Server) ContributeMPK(cmpke *ContributeMPKEvent, _ *struct{}) (err error) {
	select {
	case s.onContributeMPKEvent <- cmpke:
	case <-s.quit:
	}
	return
}

func (s *Server) ShareOrSignsShares(soss *ShareOrSignsSharesEvent, _ *struct{}) (err error) {
	select {
	case s.onShareOrSignsSharesEvent <- soss:
	case <-s.quit:
	}
	return
}

func (s *Server) ChallengeGenerated(blobberID *string, _ *struct{}) error {
	select {
	case s.onChallengeGeneration <- *blobberID:
	case <-s.quit:
	}
	return nil
}

func (s *Server) AggregateMessage(agg *AggregateMessage, _ *struct{}) error {
	select {
	case s.onAggregate <- agg:
	case <-s.quit:
	}
	return nil
}

func (s *Server) BlobberCommitted(blobberID *string, _ *struct{}) error {
	select {
	case s.onBlobberCommit <- *blobberID:
	case <-s.quit:
	}
	return nil
}

func (s *Server) ValidatorTicket(ticket *ValidtorTicket, _ *struct{}) error {
	select {
	case s.onValidatorTicket <- ticket:
	case <-s.quit:
	}
	return nil
}

func (s *Server) SetChallengeStatus(m map[string]interface{}, _ *struct{}) error {
	select {
	case s.onChallengeStatus <- m:
	case <-s.quit:
	}

	return nil
}

func (s *Server) GetFileMetaRoot(m map[string]string, _ *struct{}) error {
	select {
	case s.onGettingFileMetaRoot <- m:
	case <-s.quit:
	}
	return nil
}

// magic block handler
func (s *Server) MagicBlock(_ *struct{}, configFile *string) (err error) {
	(*configFile) = s.magicBlock
	return nil
}

// state polling handler
func (s *Server) State(id NodeID, state *State) (err error) {
	nodeName, ok := s.names[id]
	if !ok {
		return fmt.Errorf("unknown node ID: %s", id)
	}

	var ns *nodeState
	if ns, err = s.nodeState(nodeName); err != nil {
		return
	}

	// trigger the node ready once
	if ns.counter == 1 {
		select {
		case s.onNodeReady <- nodeName:
		case <-s.quit:
			return ErrShutdown
		}
	}

	select {
	case x := <-ns.poll:
		(*state) = (*x)
	case <-s.quit:
		return ErrShutdown
	}
	return
}

func (s *Server) NodeCustomConfig(id NodeID, config *NodeConfig) (err error) {
	cfg, ok := s.nodeCustomConfig[id]
	if !ok {
		log.Printf("[warn] no custom config for node %v\n", id)
		return
	}

	(*config) = (*cfg)
	return
}

//
// checks
//

func (s *Server) ConfigureTestCase(blob []byte, _ *struct{}) error {
	log.Printf("configuring test case: %s", string(blob))
	return s.CurrentTest.Configure(blob)
}

func (s *Server) AddTestCaseResult(blob []byte, _ *struct{}) error {
	log.Printf("adding result to the test case: %s", string(blob))
	return s.CurrentTest.AddResult(blob)
}

//
// stats
//

func (s *Server) AddBlockServerStats(ss *stats.BlockRequest, _ *struct{}) error {
	s.NodesServerStatsCollector.AddBlockStats(ss)
	return nil
}

func (s *Server) AddVRFSServerStats(ss *stats.VRFSRequest, _ *struct{}) error {
	s.NodesServerStatsCollector.AddVRFSStats(ss)
	return nil
}

func (s *Server) AddBlockClientStats(reqBlob []byte, _ *struct{}) error {
	req := new(BlockRequest)
	if err := req.Decode(reqBlob); err != nil {
		return err
	}

	s.NodesClientStatsCollector.AddBlockStats(req.Req, req.ReqType)
	return nil
}

//
// flow
//

// EnableServerStatsCollector initializes Server.NodesServerStatsCollector,
// and updates State.ServerStatsCollectorEnabled for all nodes.
func (s *Server) EnableServerStatsCollector() error {
	s.NodesServerStatsCollector = stats.NewNodesServerStats()
	return s.UpdateAllStates(func(state *State) {
		state.ServerStatsCollectorEnabled = true
	})
}

// EnableClientStatsCollector initializes Server.NodesClientStatsCollector,
// and updates State.ClientStatsCollectorEnabled for all nodes.
func (s *Server) EnableClientStatsCollector() error {
	s.NodesClientStatsCollector = stats.NewNodesClientStats()
	return s.UpdateAllStates(func(state *State) {
		state.ClientStatsCollectorEnabled = true
	})
}

// SetMagicBlock sets magic block in server state
func (s *Server) SetMagicBlock(configFile string) {
	s.magicBlock = configFile
}

// Close the server waiting.
func (s *Server) Close() (err error) {
	s.quitOnce.Do(func() {
		close(s.quit)
		if s.l != nil {
			err = s.l.Close()
		}
	})
	return
}
