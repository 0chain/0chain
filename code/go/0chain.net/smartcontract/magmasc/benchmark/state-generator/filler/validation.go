package filler

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"reflect"

	zmc "github.com/0chain/gosdk/zmagmacore/magmasc"

	"0chain.net/smartcontract/magmasc/benchmark/sessions"
)

func (sf *Filler) validate(consumers []*zmc.Consumer, providers []*zmc.Provider, numActiveSessions, numInactiveSessions int) error {
	if err := sf.validateRegisteredConsumers(consumers); err != nil {
		return err
	}
	log.Println("Consumers registration is valid: num ", len(consumers))

	if err := sf.validateRegisteredProviders(providers); err != nil {
		return err
	}
	log.Println("Providers registration is valid: num ", len(providers))

	if err := sf.validateSessions(numActiveSessions, numInactiveSessions); err != nil {
		return err
	}
	return nil
}

func (sf *Filler) validateRegisteredConsumers(consumers []*zmc.Consumer) error {
	handlAll := sf.sc.RestHandlers["/allConsumers"]
	if output, err := handlAll(nil, nil, nil); err != nil {
		return err
	} else {
		outputConsumers := output.([]*zmc.Consumer)
		if len(outputConsumers) != len(consumers) {
			return fmt.Errorf("validating consumer registration failed: consumers registered %d; expected %d", len(outputConsumers), len(consumers))
		}
	}

	handlOne := sf.sc.RestHandlers["/consumerFetch"]
	for ind, cons := range consumers {
		vals := url.Values{}
		vals.Set("ext_id", cons.ExtID)

		if output, err := handlOne(nil, vals, sf.sci); err != nil {
			return fmt.Errorf("got error while making '/consumerFetch' with ext_id '%s' and ind '%d': %w", cons.ExtID, ind, err)
		} else {
			outputConsumer := output.(*zmc.Consumer)
			if !reflect.DeepEqual(outputConsumer, cons) {
				return errors.New("validating consumer with ext_id '" + cons.ExtID + "' failed")
			}
		}
	}

	return nil
}

func (sf *Filler) validateRegisteredProviders(providers []*zmc.Provider) error {
	handlAll := sf.sc.RestHandlers["/allProviders"]
	if output, err := handlAll(nil, nil, nil); err != nil {
		return err
	} else {
		outputProviders := output.([]*zmc.Provider)
		if len(outputProviders) != len(providers) {
			return fmt.Errorf("validating providers registration failed: providers registered %d; expected %d", len(outputProviders), len(providers))
		}
	}

	handlOne := sf.sc.RestHandlers["/providerFetch"]
	for ind, prov := range providers {
		vals := url.Values{}
		vals.Set("ext_id", prov.ExtID)

		if output, err := handlOne(nil, vals, sf.sci); err != nil {
			return fmt.Errorf("got error while making '/providerFetch' with ext_id '%s' and ind '%d': %w", prov.ExtID, ind, err)
		} else {
			outputProvider := output.(*zmc.Provider)
			if !reflect.DeepEqual(outputProvider, prov) {
				return errors.New("validating provider with ext_id '" + prov.ExtID + "' failed")
			}
		}
	}

	return nil
}

func (sf *Filler) validateSessions(activeNum, inactiveNum int) error {
	activeInStateNum, inactiveInStateNum, err := sessions.Count(sf.sc, sf.sci)
	if err != nil {
		return err
	}
	fmt.Printf("Sessions in state: %d active, %d inactive \n", activeInStateNum, inactiveInStateNum)
	if activeInStateNum != activeNum {
		return fmt.Errorf("active sessions missmatch: %d active in state; %d expected", activeInStateNum, activeNum)
	}
	if inactiveInStateNum != inactiveNum {
		return fmt.Errorf("inactive sessions missmatch: %d in state; %d expected", inactiveInStateNum, inactiveNum)
	}

	return nil
}
