package services

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"0chain.net/conductor/stores"
	"0chain.net/conductor/types"
	"0chain.net/conductor/utils"
)

type (
	ProviderType = types.ProviderType
	Aggregate = types.Aggregate
	Monotonicity = types.Monotonicity
	Comparison = types.Comparison
)

const (
	aggregateServiceSyncInterval = 1 // How often should we run the sync command

	Miner = types.Miner
	Sharder = types.Sharder
	Blobber = types.Blobber
	Validator = types.Validator
	Authorizer = types.Authorizer
	User = types.User
	Global = types.Global

	Increase Monotonicity = types.Increase
	Decrease Monotonicity = types.Decrease

	EQ Comparison = types.EQ
	LT Comparison = types.LT
	LTE Comparison = types.LTE
	GT Comparison = types.GT
	GTE Comparison = types.GTE
)

var aggStore = stores.GetAggregateStore()

type AggregateService struct {
	recv_aggrs chan *types.Aggregate
	baseUrl string
}

func NewAggregateService(baseUrl string) *AggregateService {
	return &AggregateService{
		recv_aggrs: make(chan *types.Aggregate),
		baseUrl: baseUrl,
	}
}

func (s *AggregateService) SyncLatestAggregate(ptype types.ProviderType, pid string) (error) {
	resp, err := s.getRemoteAggregate(ptype, pid)
	if err != nil {
		log.Printf("Error getting aggregates: %v, %v, %v\n", ptype, pid, err)
		return err
	}

	log.Printf("Got aggregate for (%v, %v): %v\n", ptype, pid, resp)

	err = aggStore.Add(*resp, ptype, pid)
	if err != nil {
		log.Printf("Error adding aggregate: %v\n", err)
		return err
	}

	return nil
}

func (s *AggregateService) SyncLatestAggregates(ptype types.ProviderType, pids []string) (error) {
	var (
		aggrs []types.Aggregate
		err error
	)

	switch ptype {
	case Miner, Sharder, Blobber, Validator, Authorizer:
		aggrs, err = s.getRemoteAggregates(ptype, pids)
		if err != nil {
			return fmt.Errorf("Error getting aggregates: %v, %v, %v\n", ptype, pids, err)
		}
	case User:
		for _, pid := range pids {
			resp, err := s.getRemoteAggregate(ptype, pid)
			if err != nil {
				log.Printf("Error getting aggregates: %v, %v, %v\n", ptype, pid, err)
				continue
			}

			aggrs = append(aggrs, *resp)
		}
	case Global:
		aggr, err := s.getRemoteSnapshot()
		if err != nil {
			return fmt.Errorf("Error getting snapshot: %v\n", err)
		}

		aggrs = append(aggrs, *aggr)
	default:
		return fmt.Errorf("Unknown provider type: %v\n", ptype)
	}

	idKey := fmt.Sprintf("%v_id", ptype)
	for _, aggr := range aggrs {
		id := aggr[idKey]
		if id == nil {
			log.Printf("Provider id not found in aggregate: %v\n", aggr)
			continue
		}
		
		pid, ok := id.(string)
		if !ok {
			log.Printf("Unknown type of provider id: %v %T\n", id, id)
			continue
		}

		err = aggStore.Add(aggr, ptype, pid)
		if err != nil {
			log.Printf("Error adding aggregate: %v\n", err)
			continue
		}
	}

	return nil
}

func (s *AggregateService) CheckAggregateValueChange(ptype ProviderType, pid string, key string, monotonicity Monotonicity, tm time.Duration) (bool, error) {
	prev, err := aggStore.GetLatest(ptype, pid, key)
	if err != nil && err != types.ErrNoStoredAggregates {
		return false, err
	}

	t := time.NewTicker(aggregateServiceSyncInterval * time.Second)
	defer t.Stop()
	ts := time.Now()
	
	cancel := make(chan struct{})
	go func() {
		for range t.C {
			if time.Since(ts) > tm {
				close(cancel)
				return
			}

			resp, err := s.getRemoteAggregate(ptype, pid)
			if err != nil {
				log.Printf("Error getting aggregates: %v, %v, %v\n", ptype, pid, err)
				continue
			}

			s.recv_aggrs <- resp
		}
	}()

	for {
		select {
		case agg := <-s.recv_aggrs:
			log.Printf("Got aggregate for (%v, %v): %v\n", ptype, pid, agg)
			check, err := s.checkAggKeyValueChange(prev, *agg, key, monotonicity)
			if err != nil {
				return false, err
			}

			if check {
				return true, nil
			}

			err = aggStore.Add(*agg, ptype, pid)
			if err != nil {
				log.Printf("Error adding aggregate: %v\n", err)
			}

			prev = *agg
		case <-cancel:
			return false, nil
		}
	}
}

func (s *AggregateService) CompareAggregateValue(ptype ProviderType, pid string, key string, comparison Comparison, value float64, tm time.Duration) (bool, error) {
	t := time.NewTicker(aggregateServiceSyncInterval * time.Second)
	defer t.Stop()
	ts := time.Now()
	
	cancel := make(chan struct{})
	go func() {
		for range t.C {
			if time.Since(ts) > tm {
				close(cancel)
				return
			}

			resp, err := s.getRemoteAggregate(ptype, pid)
			if err != nil {
				log.Printf("Error getting aggregates: %v, %v, %v\n", ptype, pid, err)
				continue
			}

			s.recv_aggrs <- resp
		}
	}()

	for {
		select {
		case agg := <-s.recv_aggrs:
			log.Printf("Got aggregate for (%v, %v): %v\n", ptype, pid, agg)
			check, err := s.compareAggValue(*agg, key, value, comparison)
			if err != nil {
				return false, err
			}

			if check {
				return true, nil
			}
		case <-cancel:
			return false, nil
		}
	}
}

func (s *AggregateService) getRemoteAggregate(ptype ProviderType, pid string) (*types.Aggregate, error) {
	url := fmt.Sprintf("%v/%v-aggregate?id=%v", s.baseUrl, ptype, pid)

	log.Printf("Getting aggregate from %v\n", url)

	resp, err := utils.HttpGet(url, map[string]string{})
	if err != nil {
		return nil, err
	}

	agg := &types.Aggregate{}
	err = json.Unmarshal(resp, agg)
	if err != nil {
		return nil, err
	}

	return agg, nil
}

func (s *AggregateService) getRemoteAggregates(ptype ProviderType, pids []string) ([]types.Aggregate, error) {
	idsParams := ""
	for i, pid := range pids {
		if i > 0 {
			idsParams += "&"
		}
		idsParams += fmt.Sprintf("ids[]=%v", pid)
	}

	url := fmt.Sprintf("%v/%v-aggregates?%v", s.baseUrl, ptype, idsParams)

	log.Printf("Getting aggregates from %v\n", url)

	resp, err := utils.HttpGet(url, map[string]string{})
	if err != nil {
		return nil, err
	}

	agg := []types.Aggregate{}
	err = json.Unmarshal(resp, &agg)
	if err != nil {
		return nil, err
	}

	return agg, nil
}

func (s *AggregateService) getRemoteSnapshot() (*types.Aggregate, error) {
	url := fmt.Sprintf("%v/latest-snapshot", s.baseUrl)

	log.Printf("Getting snapshot from %v\n", url)

	resp, err := utils.HttpGet(url, map[string]string{})
	if err != nil {
		return nil, err
	}

	agg := &types.Aggregate{}
	err = json.Unmarshal(resp, &agg)
	if err != nil {
		return nil, err
	}

	return agg, nil
}

func (s *AggregateService) checkAggKeyValueChange(prev, cur Aggregate, key string, mono Monotonicity) (bool, error) {
	prevVal, ok := prev[key]
	if !ok {
		log.Printf("key (%v) not found in previous value: %v", key, prev)
		prevVal = float64(0)
	}

	curVal, ok := cur[key]
	if !ok {
		return false, fmt.Errorf("key (%v) not found in currnet value: %v", key, cur)
	}

	f64Prev, ok := prevVal.(float64)
	if !ok {
		return false, fmt.Errorf("unknown type of previous value: %v %T", prevVal, prevVal)
	}

	f64Cur, ok := curVal.(float64)
	if !ok {
		return false, fmt.Errorf("unknown type of current value: %v %T", curVal, curVal)
	}

	switch mono {
		case Increase:
			return f64Cur > f64Prev, nil
		case Decrease:
			return f64Cur < f64Prev, nil
		default:
			return false, fmt.Errorf("unknown monotonicity: %v", mono)
	}
}

func (s *AggregateService) compareAggValue(agg Aggregate, key string, expectedVal float64, comp Comparison) (bool, error) {
	actualVal, ok := agg[key]
	if !ok {
		return false, fmt.Errorf("key (%v) not found in aggregate: %v", key, agg)
	}

	f64Actual, ok := actualVal.(float64)
	if !ok {
		return false, fmt.Errorf("unknown type of actual value: %v %T", actualVal, actualVal)
	}

	switch comp {
		case EQ:
			return f64Actual == expectedVal, nil
		case LT:
			return f64Actual < expectedVal, nil
		case LTE:
			return f64Actual <= expectedVal, nil
		case GT:
			return f64Actual > expectedVal, nil
		case GTE:
			return f64Actual >= expectedVal, nil
		default:
			return false, fmt.Errorf("unknown comparison: %v", comp)
	}
}