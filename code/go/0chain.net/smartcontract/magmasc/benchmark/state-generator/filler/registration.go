package filler

import (
	"encoding/json"
	"fmt"
	"sync"

	store "0chain.net/core/ememorystore"
	"0chain.net/smartcontract/magmasc"
	"0chain.net/smartcontract/magmasc/benchmark/rand"
	"0chain.net/smartcontract/magmasc/benchmark/state-generator/bar"
)

func (sf *Filler) registerAll(nc, np int) error {
	if nc <= 0 && np <= 0 {
		return nil
	}

	wg := sync.WaitGroup{}
	fmt.Println("Start registering nodes...")
	sf.pBar = bar.StartNew(nc+np, sf.pBarSep)

	var crErr, prErr error
	if nc > 0 {
		wg.Add(1)
		go func() {
			crErr = sf.registerConsumers(nc)
			wg.Done()
		}()
	}

	if nc > 0 {
		wg.Add(1)
		go func() {
			prErr = sf.registerProviders(np)
			wg.Done()
		}()
	}

	wg.Wait()
	sf.pBar.Finish()

	switch {
	case crErr != nil:
		return crErr
	case prErr != nil:
		return prErr
	}

	return nil
}

func (sf *Filler) registerConsumers(num int) error {
	if num <= 0 {
		return nil
	}

	consumers := rand.Consumers(num)
	blob, err := json.Marshal(consumers)
	if err != nil {
		return err
	}

	tx := store.GetTransaction(sf.msc.GetDB())
	if err = tx.Conn.Put([]byte(magmasc.AllConsumersKey), blob); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}

	// insert each in state
	for _, cons := range consumers {
		var (
			consumerType = "consumer"
		)
		if _, err = sf.sci.InsertTrieNode(nodeUID(magmasc.Address, consumerType, cons.ExtID), cons); err != nil {
			return err
		}

		sf.pBar.Increment()
	}

	return nil
}

func (sf *Filler) registerProviders(num int) error {
	if num <= 0 {
		return nil
	}

	providers := rand.Providers(num)
	blob, err := json.Marshal(providers)
	if err != nil {
		return err
	}

	tx := store.GetTransaction(sf.msc.GetDB())
	if err = tx.Conn.Put([]byte(magmasc.AllProvidersKey), blob); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}

	// insert each in state
	for _, prov := range providers {
		key := "sc:" + magmasc.Address + ":" + "provider" + ":" + prov.ExtID
		if _, err = sf.sci.InsertTrieNode(key, prov); err != nil {
			return err
		}
		sf.pBar.Increment()
	}

	return nil
}
