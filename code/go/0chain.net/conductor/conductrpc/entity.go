package conductrpc

import (
	"log"
	"sync"
	"time"

	"0chain.net/conductor/conductrpc/stats"
	"0chain.net/core/viper"
)

// Entity represents client long polling instance.
type Entity struct {
	id     NodeID  // this
	client *client // RPC client

	stateMu sync.Mutex
	state   *State // the last state (can be nil first time)

	quitOnce sync.Once     //
	quit     chan struct{} //
}

// State returns current state.
func (e *Entity) State() (state *State) {
	e.stateMu.Lock()
	defer e.stateMu.Unlock()

	return e.state
}

// MagicBlock returns the location path of the magic block configuration.
func (e *Entity) MagicBlock() string {
	magicBlock, err := e.client.magicBlock()
	if err != nil {
		log.Fatalf("failed getting magic block: %v", err)
	}

	return *magicBlock
}

// SetState sets current state.
func (e *Entity) SetState(state *State) {
	e.stateMu.Lock()
	defer e.stateMu.Unlock()

	e.state = state
}

// Register registers node in conductor server
func (e *Entity) Register(id string) {
	var (
		interval = viper.GetDuration("integration_tests.lock_interval")
	)

	e.id = NodeID(id)

	// initial state polling and wait node unlock
	for {
		// state polling can't return nil-State if err is nil
		state, err := e.client.state(NodeID(id))
		if err != nil {
			panic("requesting RPC (State): " + err.Error())
		}
		e.SetState(state)
		if !state.IsLock {
			break // can join blockchain
		}
		// otherwise, have to wait, retry after the interval

		// the joining is expected, since we can simply use the time.Sleep
		// instead of select with time.After and context.Done for tests
		time.Sleep(interval)
	}

	// start state polling
	go e.pollState()
}

// NewEntity creates RPC client for integration tests.
func NewEntity() (e *Entity) {
	var (
		client, err = newClient(viper.GetString("integration_tests.address"))
	)
	if err != nil {
		log.Fatalf("creating RPC client: %v", err)
	}

	e = new(Entity)
	e.client = client
	e.quit = make(chan struct{})

	return
}

func (e *Entity) pollState() {
	for {
		select {
		case <-e.quit:
			return
		default:
		}
		var state, err = e.client.state(e.id)
		if err != nil {
			log.Printf("polling State: %v", err)
			continue
		}
		e.SetState(state)
	}
}

func (e *Entity) Shutdown() {
	e.quitOnce.Do(func() {
		close(e.quit)
	})
}

func (e *Entity) isMonitor() bool {
	var state = e.State()
	return state != nil && state.IsMonitor
}

//
// RPC methods (events notification)
//

func (e *Entity) Phase(phase *PhaseEvent) (err error) {
	if !e.isMonitor() {
		return // not a monitor
	}
	return e.client.phase(phase)
}

func (e *Entity) ViewChange(viewChange *ViewChangeEvent) (err error) {
	if !e.isMonitor() {
		return // not a monitor
	}
	return e.client.viewChange(viewChange)
}

func (e *Entity) AddMiner(add *AddMinerEvent) (err error) {
	if !e.isMonitor() {
		return // not a monitor
	}
	return e.client.addMiner(add)
}

func (e *Entity) AddSharder(add *AddSharderEvent) (err error) {
	if !e.isMonitor() {
		return // not a monitor
	}
	return e.client.addSharder(add)
}

func (e *Entity) AddBlobber(add *AddBlobberEvent) (err error) {
	if !e.isMonitor() {
		return // not a monitor
	}
	return e.client.addBlobber(add)
}

func (e *Entity) SharderKeep(sk *SharderKeepEvent) (err error) {
	if !e.isMonitor() {
		return // not a monitor
	}
	return e.client.sharderKeep(sk)
}

func (e *Entity) Round(re *RoundEvent) (err error) {
	if !e.isMonitor() {
		return // not a monitor
	}
	return e.client.round(re)
}

func (e *Entity) ContributeMPK(cmpke *ContributeMPKEvent) (err error) {
	if !e.isMonitor() {
		return // not a monitor
	}
	return e.client.contributeMPK(cmpke)
}

func (e *Entity) ShareOrSignsShares(sosse *ShareOrSignsSharesEvent) (
	err error) {

	if !e.isMonitor() {
		return // not a monitor
	}
	return e.client.shareOrSignsShares(sosse)
}

//
// global
//

// checks

func (e *Entity) ConfigureTestCase(blob []byte) error {
	return e.client.configureTestCase(blob)
}

func (e *Entity) AddTestCaseResult(blob []byte) error {
	return e.client.addTestCaseResult(blob)
}

// stats

func (e *Entity) AddBlockServerStats(ss *stats.BlockRequest) error {
	return e.client.addBlockServerStats(ss)
}

func (e *Entity) AddVRFSServerStats(ss *stats.VRFSRequest) error {
	return e.client.addVRFSServerStats(ss)
}

func (e *Entity) AddBlockClientStats(rs *stats.BlockRequest, reqType stats.BlockRequestor) error {
	br := newBlockRequest(rs, reqType)
	blob, err := br.Encode()
	if err != nil {
		return err
	}

	return e.client.addBlockClientStats(blob)
}

var global *Entity

// Init creates global Entity and locks until unlocked.
func Init() {
	global = NewEntity()
}

// Shutdown the global Entity.
func Shutdown() {
	if global != nil {
		global.Shutdown()
	}
}

// Client returns global Entity to interact with. Use it, for example,
//
//     var state = conductrpc.Client().State()
//     for _, minerID := range miners {
//         if state.VRFS.IsBad(state, minerID) {
//             // send bad VRFS to this miner
//         } else if state.VRFS.IsGood(state, minerID) {
//             // send good VRFS to this miner
//         } else {
//             // don't send a VRFS to this miner
//         }
//     }
//
func Client() *Entity {
	return global
}
