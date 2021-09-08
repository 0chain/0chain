package filler

import (
	"math/rand"
	"sync"

	zmc "github.com/0chain/gosdk/zmagmacore/magmasc"
	ts "github.com/0chain/gosdk/zmagmacore/time"

	"0chain.net/chaincore/block"
	chain "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	tx "0chain.net/chaincore/transaction"
	"0chain.net/core/encryption"
	"0chain.net/smartcontract/magmasc"
	"0chain.net/smartcontract/magmasc/benchmark/sessions"
	"0chain.net/smartcontract/magmasc/benchmark/state-generator/bar"
)

func (sf *Filler) fillSessions(
	nasInState,
	nas,
	nisInState,
	nis int,
	consumers []*zmc.Consumer,
	providers []*zmc.Provider,
) error {
	if nas <= 0 && nis <= 0 {
		return nil
	}

	wg := sync.WaitGroup{}
	println("\nStart filling sessions...")
	sf.pBar = bar.StartNew(nas+nis, sf.pBarSep)

	var nasErr, nisErr error
	if nas > 0 {
		wg.Add(1)
		go func() {
			nasErr = sf.activateSessions(nasInState, nas, consumers, providers)
			wg.Done()
		}()
	}
	if nis > 0 {
		wg.Add(1)
		go func() {
			nisErr = sf.inactivateSessions(nisInState, nis, consumers, providers)
			wg.Done()
		}()
	}

	wg.Wait()
	sf.pBar.Finish()

	switch {
	case nasErr != nil:
		return nasErr
	case nisErr != nil:
		return nisErr
	}

	return nil
}

func (sf *Filler) activateSessions(nasInState, nas int, consumers []*zmc.Consumer, providers []*zmc.Provider) error {
	wg := sync.WaitGroup{}
	wg.Add(nas)
	for i := nasInState; i < nasInState+nas; i++ {
		sf.goroutines <- struct{}{}
		go func(i int) {
			_, err := sf.activateSession(
				consumers[rand.Intn(len(consumers))],
				providers[rand.Intn(len(providers))],
				sessions.GetSessionName(i, true),
			)
			if err != nil {
				panic(err)
			}

			wg.Done()
			sf.pBar.Increment()
			<-sf.goroutines
		}(i)
	}
	wg.Wait()

	return nil
}

func (sf *Filler) activateSession(consumer *zmc.Consumer, provider *zmc.Provider, sessionID string) (*zmc.Acknowledgment, error) {
	ackn := createAcknowledgment(sessionID, consumer, provider)
	sci := chain.NewStateContext(
		&block.Block{},
		sf.sci.GetState(),
		&state.Deserializer{},
		&tx.Transaction{
			ClientID:   ackn.Consumer.ID,
			ToClientID: sf.msc.ID,
		},
		func(*block.Block) []string { return []string{} },
		func() *block.Block { return &block.Block{} },
		func() *block.MagicBlock { return &block.MagicBlock{} },
		func() encryption.SignatureScheme { return &encryption.BLS0ChainScheme{} },
	)

	transfer := state.NewTransfer(ackn.Consumer.ID, magmasc.Address, state.Balance(ackn.Terms.GetAmount()))
	if err := sci.AddTransfer(transfer); err != nil {
		return nil, err
	}

	if _, err := sf.sci.InsertTrieNode(nodeUID(magmasc.Address, acknowledgment, ackn.SessionID), ackn); err != nil {
		return nil, err
	}

	return ackn, nil
}

func (sf *Filler) inactivateSessions(nisInState, nis int, consumers []*zmc.Consumer, providers []*zmc.Provider) error {
	wg := sync.WaitGroup{}
	wg.Add(nis)
	for i := nisInState; i < nisInState+nis; i++ {
		sf.goroutines <- struct{}{}
		go func(i int) {
			ackn, err := sf.activateSession(
				consumers[rand.Intn(len(consumers))],
				providers[rand.Intn(len(providers))],
				sessions.GetSessionName(i, false),
			)
			if err != nil {
				panic(err)
			}
			if err := sf.inactivateSession(ackn); err != nil {
				panic(err)
			}

			wg.Done()
			sf.pBar.Increment()
			<-sf.goroutines
		}(i)
	}
	wg.Wait()

	return nil
}

func (sf *Filler) inactivateSession(ackn *zmc.Acknowledgment) error {
	sci := chain.NewStateContext(
		&block.Block{},
		sf.sci.GetState(),
		&state.Deserializer{},
		&tx.Transaction{
			ClientID:   sf.msc.ID,
			ToClientID: ackn.Consumer.ID,
		},
		func(*block.Block) []string { return []string{} },
		func() *block.Block { return &block.Block{} },
		func() *block.MagicBlock { return &block.MagicBlock{} },
		func() encryption.SignatureScheme { return &encryption.BLS0ChainScheme{} },
	)
	if err := spend(sci.GetTransaction(), &ackn.Billing, sci, *ackn.TokenPool); err != nil {
		return err
	}

	ackn.Billing.CompletedAt = ts.Now()
	if _, err := sf.sci.InsertTrieNode(nodeUID(magmasc.Address, acknowledgment, ackn.SessionID), ackn); err != nil {
		return err
	}

	return nil
}
