package state

import (
	"0dns.io/core/config"
	"0dns.io/core/logging"
	"github.com/0chain/gosdk/core/block"
	"strconv"
	"strings"
	"sync"
)

const (
	HTTPProtocol  = "http://"
	HTTPSProtocol = "https://"
)

// State defines the latest state of network based on most recent magic block.
type State struct {
	sync.RWMutex
	Miners            []string
	Sharders          []string
	CurrentMagicBlock *block.MagicBlock
}

// state is a global status of 0dns.
var state = &State{}

// Get return a copy of state.
func Get() State {
	state.RLock()
	defer state.RUnlock()

	return State{
		Miners:            state.Miners,
		Sharders:          state.Sharders,
		CurrentMagicBlock: state.CurrentMagicBlock,
	}
}

func SetFromCurrentMagicBlock(c config.Config, b *block.MagicBlock) {
	if b == nil {
		panic("Unexpected missing magic block")
	}

	networkProtocol := HTTPProtocol
	if c.UseHTTPS {
		networkProtocol = HTTPSProtocol
	}

	var miners []string
	for _, miner := range b.Miners.Nodes {
		host := miner.Host
		if strings.Contains(host, "localhost") {
			host = miner.N2NHost
		}

		if c.UsePath {
			miners = append(miners,
				networkProtocol+
					host+
					"/"+
					miner.Path)
		} else {
			miners = append(miners,
				networkProtocol+
					host+
					":"+
					strconv.Itoa(miner.Port))
		}
	}

	var sharders []string
	for _, sharder := range b.Sharders.Nodes {
		host := sharder.Host
		if strings.Contains(host, "localhost") {
			host = sharder.N2NHost
		}
		if c.UsePath {
			sharders = append(sharders,
				networkProtocol+
					host+
					"/"+
					sharder.Path)
		} else {
			sharders = append(sharders,
				networkProtocol+
					host+
					":"+
					strconv.Itoa(sharder.Port))
		}
	}

	logging.Logger.Info("miners: " + strings.Join(miners, ", "))
	logging.Logger.Info("sharders: " + strings.Join(sharders, ", "))

	state.Lock()
	defer state.Unlock()

	state.CurrentMagicBlock = b
	state.Miners = miners
	state.Sharders = sharders
}
