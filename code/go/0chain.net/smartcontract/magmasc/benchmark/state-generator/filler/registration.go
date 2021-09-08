package filler

import (
	"encoding/json"
	"fmt"

	zmc "github.com/0chain/gosdk/zmagmacore/magmasc"

	store "0chain.net/core/ememorystore"
	"0chain.net/smartcontract/magmasc"
	"0chain.net/smartcontract/magmasc/benchmark/rand"
	"0chain.net/smartcontract/magmasc/benchmark/state-generator/bar"
)

func (sf *Filler) registerAll(numConsumers, numProviders int) ([]*zmc.Consumer, []*zmc.Provider, error) {
	fmt.Println("Start registering nodes ...")
	sf.pBar = bar.StartNew(numConsumers+numProviders, sf.sepPBar)

	var (
		consumers []*zmc.Consumer
		providers []*zmc.Provider

		errCh    = make(chan error, 2)
		errCount int
	)
	go func() {
		var err error
		consumers, err = sf.registerConsumers(numConsumers)
		errCh <- err
	}()
	errCount++
	go func() {
		var err error
		providers, err = sf.registerProviders(numProviders)
		errCh <- err
	}()
	errCount++

	for err := range errCh {
		if err != nil {
			sf.pBar.Finish()
			return nil, nil, err
		}
		errCount--
		if errCount == 0 {
			close(errCh)
		}
	}
	sf.pBar.Finish()

	return consumers, providers, nil
}

func (sf *Filler) registerConsumers(num int) ([]*zmc.Consumer, error) {
	var (
		consumers = rand.Consumers(num)
	)

	// store all consumers list
	consumersByt, err := json.Marshal(consumers)
	if err != nil {
		return nil, err
	}
	tx := store.GetTransaction(sf.sc.GetDB())
	if err = tx.Conn.Put([]byte(magmasc.AllConsumersKey), consumersByt); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}

	// insert each in state
	for _, cons := range consumers {
		var (
			consumerType = "consumer"
		)
		if _, err := sf.sci.InsertTrieNode(nodeUID(magmasc.Address, consumerType, cons.ExtID), cons); err != nil {
			return nil, err
		}

		sf.pBar.Increment()
	}

	return consumers, nil
}

func (sf *Filler) registerProviders(num int) ([]*zmc.Provider, error) {
	var (
		providers = rand.Providers(num)
	)

	// store all providers list
	providersByt, err := json.Marshal(providers)
	if err != nil {
		return nil, err
	}
	tx := store.GetTransaction(sf.sc.GetDB())
	if err = tx.Conn.Put([]byte(magmasc.AllProvidersKey), providersByt); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}

	// insert each in state
	for _, prov := range providers {
		key := "sc:" + magmasc.Address + ":" + "provider" + ":" + prov.ExtID
		if _, err := sf.sci.InsertTrieNode(key, prov); err != nil {
			return nil, err
		}
		sf.pBar.Increment()
	}

	return providers, nil
}
