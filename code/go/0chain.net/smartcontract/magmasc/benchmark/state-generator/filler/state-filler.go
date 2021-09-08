package filler

import (
	zmc "github.com/0chain/gosdk/zmagmacore/magmasc"

	chain "0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/magmasc"
	"0chain.net/smartcontract/magmasc/benchmark/sessions"
	"0chain.net/smartcontract/magmasc/benchmark/state-generator/bar"
)

type (
	// Filler fills state by executing magma's smart contract functions.
	Filler struct {
		sci chain.StateContextI
		sc  *magmasc.MagmaSmartContract

		acknCh chan *zmc.Acknowledgment

		pBar    *bar.ProgressBar
		sepPBar bool

		activeGoroutines chan struct{}
	}
)

func New(sci chain.StateContextI, sc *magmasc.MagmaSmartContract, numGoroutines int, separatePBar bool) *Filler {
	return &Filler{
		sci:              sci,
		sc:               sc,
		acknCh:           make(chan *zmc.Acknowledgment),
		sepPBar:          separatePBar,
		activeGoroutines: make(chan struct{}, numGoroutines),
	}
}

// Fill registers nodes, starts and stops sessions by executing magma sc functions.
func (sf *Filler) Fill(numConsumers, numProviders, numActiveSessions, numInactiveSessions int) error {
	consumers, providers, err := sf.registerAll(numConsumers, numProviders)
	if err != nil {
		return err
	}

	actInState, inactInState, err := sessions.Count(sf.sc, sf.sci)
	if err != nil {
		return err
	}

	if err := sf.fillSessions(actInState, numActiveSessions, inactInState, numInactiveSessions, consumers, providers); err != nil {
		return err
	}

	return sf.validate(consumers, providers, actInState+numActiveSessions, inactInState+numInactiveSessions)
}
