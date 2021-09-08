package filler

import (
	"fmt"
	"math/rand"
	"sync"

	zmc "github.com/0chain/gosdk/zmagmacore/magmasc"
	ts "github.com/0chain/gosdk/zmagmacore/time"

	"0chain.net/chaincore/block"
	chain "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/encryption"
	"0chain.net/smartcontract/magmasc"
	"0chain.net/smartcontract/magmasc/benchmark/sessions"
	"0chain.net/smartcontract/magmasc/benchmark/state-generator/bar"
)

func (sf *Filler) fillSessions(numActiveSessionsInState, activateNum, numInactiveSessionsInState, inactivateNum int,
	consumers []*zmc.Consumer, providers []*zmc.Provider) (err error) {

	fmt.Println("\nStart filling sessions ...")
	sf.pBar = bar.StartNew(activateNum+inactivateNum, sf.sepPBar)

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		err = sf.activateSessions(numActiveSessionsInState, activateNum, consumers, providers)
		wg.Done()
	}()
	go func() {
		err = sf.inactivateSessions(numInactiveSessionsInState, inactivateNum, consumers, providers)
		wg.Done()
	}()
	wg.Wait()
	sf.pBar.Finish()

	return nil
}

func (sf *Filler) activateSessions(numActiveSessionsInState, activateNum int, consumers []*zmc.Consumer, providers []*zmc.Provider) error {
	wg := sync.WaitGroup{}
	wg.Add(activateNum)
	for i := numActiveSessionsInState; i < numActiveSessionsInState+activateNum; i++ {
		sf.activeGoroutines <- struct{}{}
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
			<-sf.activeGoroutines
		}(i)
	}
	wg.Wait()

	return nil
}

func (sf *Filler) activateSession(consumer *zmc.Consumer, provider *zmc.Provider, sessionID string) (*zmc.Acknowledgment, error) {
	var (
		sci = chain.NewStateContext(
			&block.Block{},
			sf.sci.GetState(),
			&state.Deserializer{},
			&transaction.Transaction{
				ClientID:   consumer.ID,
				ToClientID: consumer.ID,
			},
			func(*block.Block) []string { return []string{} },
			func() *block.Block { return &block.Block{} },
			func() *block.MagicBlock { return &block.MagicBlock{} },
			func() encryption.SignatureScheme { return &encryption.BLS0ChainScheme{} },
		)

		ackn = createAcknowledgment(sessionID, consumer, provider)

		transfer = state.NewTransfer(ackn.Consumer.ID, magmasc.Address, state.Balance(ackn.Terms.GetAmount()))
	)

	if err := sci.AddTransfer(transfer); err != nil {
		return nil, err
	}

	if _, err := sf.sci.InsertTrieNode(nodeUID(magmasc.Address, acknowledgment, ackn.SessionID), ackn); err != nil {
		return nil, err
	}

	return ackn, nil
}

func (sf *Filler) inactivateSessions(numInactiveSessionsInState, inactivateNum int, consumers []*zmc.Consumer, providers []*zmc.Provider) error {
	wg := sync.WaitGroup{}
	wg.Add(inactivateNum)
	for i := numInactiveSessionsInState; i < numInactiveSessionsInState+inactivateNum; i++ {
		sf.activeGoroutines <- struct{}{}
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
			<-sf.activeGoroutines
		}(i)
	}
	wg.Wait()

	return nil
}

func (sf *Filler) inactivateSession(ackn *zmc.Acknowledgment) error {
	pool := &zmc.TokenPool{}
	if err := pool.Decode(ackn.TokenPool.Encode()); err != nil {
		return err
	}

	var (
		txn = transaction.Transaction{
			ClientID:   ackn.Consumer.ID,
			ToClientID: ackn.Consumer.ID,
		}
		sci = chain.NewStateContext(
			&block.Block{},
			sf.sci.GetState(),
			&state.Deserializer{},
			&txn,
			func(*block.Block) []string { return []string{} },
			func() *block.Block { return &block.Block{} },
			func() *block.MagicBlock { return &block.MagicBlock{} },
			func() encryption.SignatureScheme { return &encryption.BLS0ChainScheme{} },
		)
	)
	if err := spend(&txn, &ackn.Billing, sci, *ackn.TokenPool); err != nil {
		return err
	}

	ackn.Billing.CompletedAt = ts.Now()
	if _, err := sf.sci.InsertTrieNode(nodeUID(magmasc.Address, acknowledgment, ackn.SessionID), ackn); err != nil {
		return err
	}

	return nil
}
