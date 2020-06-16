package conductrpc

import (
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/spf13/viper"
)

// Entity represents client long polling instance.
// The Client used in Miner SC (one instance) and
// in miners and sharders code (another instance,
// the Entity). The Entity uses long polling
// methods.
type Entity struct {
	id     NodeID  // this
	client *Client // RPC client

	state atomic.Value // the last state (can be nil first time)

	quitOnce sync.Once     //
	quit     chan struct{} //
}

// State returns current state.
func (e *Entity) State() (state *State) {
	if val := e.state.Load(); val != nil {
		return val.(*State)
	}
	return // nil, not polled yet
}

// SetState sets current state.
func (e *Entity) SetState(state *State) {
	e.state.Store(state) // update
}

var globalEntity *Entity

// Init integration tests
func Init(id string) {

	var (
		client, err = NewClient(viper.GetString("integration_tests.address"))
		interval    = viper.GetDuration("integration_tests.lock_interval")
		state       *State
	)
	if err != nil {
		log.Fatalf("creating RPC client: %v", err)
	}

	globalEntity = new(Entity)
	globalEntity.id = NodeID(id)
	globalEntity.client = client
	globalEntity.quit = make(chan struct{})

	// initial state polling and wait node unlock
	for {
		// state polling can't return nil-State if err is nil
		state, err = client.State(NodeID(id))
		if err != nil {
			panic("requesting RPC (State): " + err.Error())
		}
		globalEntity.SetState(state)
		if !state.IsLock {
			break // can join blockchain
		}
		// otherwise, have to wait, retry after the interval

		// the joining is expected, since we can simply use the time.Sleep
		// instead of select with time.After and context.Done for tests
		time.Sleep(interval)
	}

	// start state polling
	go globalEntity.pollState()

}

func (e *Entity) pollState() {
	for {
		select {
		case <-e.quit:
			return
		default:
		}
		var state, err = e.client.State(e.id)
		if err != nil {
			log.Printf("polling State: %v", err)
			continue
		}
		e.SetState(state)
	}
}

func (e *Entity) shutdown() {
	e.quitOnce.Do(func() {
		close(e.quit)
	})
}

func isInList(ids []NodeID, id NodeID) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

func (e *Entity) isSendShareFor(id string) bool {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if len(e.only) == 0 {
		if isInList(e.bad, NodeID(id)) {
			return false // bad share will be sent, skip
		}
		return true // allow all
	}

	if isInList(e.only, NodeID(id)) {
		if isInList(e.bad, NodeID(id)) {
			return false // bad share will be sent, skip
		}
		return true // send for this node
	}

	return false
}

func (e *Entity) isSendBadShareFor(id string) bool {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	if len(e.bad) == 0 {
		return false // don't send bad share
	}

	return isInList(e.bad, NodeID(id))
}

func (e *Entity) isRevealed() bool {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	return e.revealed
}

// Shutdown Entity.
func Shutdown() {
	globalEntity.shutdown()
}

// IsSendShareFor returns true if this node should send share for given one.
func IsSendShareFor(id string) bool {
	return globalEntity.isSendShareFor(id)
}

// IsSendShareFor returns true if this node should send bad share for given one.
func IsSendBadShareFor(id string) bool {
	return globalEntity.isSendBadShareFor(id)
}

func IsRevealed() bool {
	return globalEntity.isRevealed()
}
