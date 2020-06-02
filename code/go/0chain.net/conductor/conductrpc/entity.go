package conductrpc

import (
	"log"
	"sync"
	"time"

	"github.com/spf13/viper"
)

// Entity represents client long polling instance.
// The Client used in Miner SC (one instance) and
// in miners and sharders code (another instance,
// the Entity). The Entity uses long polling
// methods.
type Entity struct {
	id NodeID

	client *Client

	mutex    sync.Mutex
	only     []NodeID // send share only for this nodes
	bad      []NodeID // send bad share for this nodes (regardless the 'only' list)
	revealed bool     // default is false

	quitOnce sync.Once
	quit     chan struct{}
}

var globalEntity *Entity

// Init integration tests
func Init(id string) {

	var (
		client, err = NewClient(viper.GetString("integration_tests.address"))
		interval    = viper.GetDuration("integration_tests.lock_interval")
		join        bool
	)
	if err != nil {
		log.Fatalf("creating RPC client: %v", err)
	}

	globalEntity = new(Entity)
	globalEntity.id = NodeID(id)
	globalEntity.client = client
	globalEntity.quit = make(chan struct{})

	go globalEntity.pollSendShareOnly()
	go globalEntity.pollSendShareBad()
	go globalEntity.pollIsRevealed()

	for {
		join, err = client.NodeReady(NodeID(id))
		if err != nil {
			panic("requesting RPC (NodeReady): " + err.Error())
		}
		if join {
			return // can join blockchain
		}
		// otherwise, have to wait, retry after the interval

		// the joining is expected, since we can simply use the time.Sleep
		// instead of select with time.After and context.Done for tests
		time.Sleep(interval)
	}

}

func (e *Entity) setShareOnly(only []NodeID) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	e.only = only
}

func (e *Entity) setShareBad(bad []NodeID) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	e.bad = bad
}

func (e *Entity) setRevealed(pin bool) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	e.revealed = pin
}

func (e *Entity) pollSendShareOnly() {
	for {
		select {
		case <-e.quit:
			return
		default:
		}
		var only, err = e.client.SendShareOnly(e.id)
		if err != nil {
			log.Printf("polling SendShareOnly: %v", err)
			continue
		}
		e.setShareOnly(only)
	}
}

func (e *Entity) pollSendShareBad() {
	for {
		select {
		case <-e.quit:
			return
		default:
		}
		var bad, err = e.client.SendShareBad(e.id)
		if err != nil {
			log.Printf("polling SendShareBad: %v", err)
			continue
		}
		e.setShareBad(bad)
	}
}

func (e *Entity) pollIsRevealed() {
	for {
		select {
		case <-e.quit:
			return
		default:
		}
		var pin, err = e.client.IsRevealed(e.id)
		if err != nil {
			log.Printf("polling IsRevealed: %v", err)
			continue
		}
		e.setRevealed(pin)
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
