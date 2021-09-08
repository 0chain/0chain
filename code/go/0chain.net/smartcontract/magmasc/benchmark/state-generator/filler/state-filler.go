package filler

import (
	"github.com/0chain/gosdk/zmagmacore/errors"
	zmc "github.com/0chain/gosdk/zmagmacore/magmasc"

	chain "0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/magmasc"
	"0chain.net/smartcontract/magmasc/benchmark/sessions"
	"0chain.net/smartcontract/magmasc/benchmark/state-generator/bar"
)

type (
	// Filler fills state by executing magma's smart contract functions.
	Filler struct {
		sci        chain.StateContextI
		msc        *magmasc.MagmaSmartContract
		chAckn     chan *zmc.Acknowledgment
		pBar       *bar.ProgressBar
		pBarSep    bool
		goroutines chan struct{}
	}
)

func New(sci chain.StateContextI, msc *magmasc.MagmaSmartContract, goroutines int, sep bool) *Filler {
	return &Filler{
		sci:        sci,
		msc:        msc,
		chAckn:     make(chan *zmc.Acknowledgment),
		pBarSep:    sep,
		goroutines: make(chan struct{}, goroutines),
	}
}

// Fill registers nodes, starts and stops sessions by executing magma sc functions.
func (sf *Filler) Fill(nc, np, nas, nis int) error {
	if err := sf.registerAll(nc, np); err != nil {
		return err
	}

	nasInState, err := sessions.CountActive(sf.msc, sf.sci)
	if err != nil {
		return err
	}
	nisInState, err := sessions.CountInactive(sf.msc, sf.sci)
	if err != nil {
		return err
	}

	var data interface{}
	data, err = sf.msc.RestHandlers["/allConsumers"](nil, nil, sf.sci)
	if err != nil {
		return err
	}
	consumers, ok := data.([]*zmc.Consumer)
	if !ok || len(consumers) == 0 {
		return errors.New("internal", "empty registered consumers' list")
	}

	data, err = sf.msc.RestHandlers["/allProviders"](nil, nil, sf.sci)
	if err != nil {
		return err
	}
	providers, ok := data.([]*zmc.Provider)
	if !ok || len(providers) == 0 {
		return errors.New("internal", "empty registered providers' list")
	}

	if err = sf.fillSessions(nasInState, nas, nisInState, nis, consumers, providers); err != nil {
		return err
	}

	return sf.validate(consumers, providers, nasInState+nas, nisInState+nis)
}
