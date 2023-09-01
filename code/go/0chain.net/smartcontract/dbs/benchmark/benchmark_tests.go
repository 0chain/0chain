package benchmark

import (
	"testing"

	"0chain.net/smartcontract/benchmark/main/cmd/log"

	"0chain.net/smartcontract/dbs/event"

	"0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	bk "0chain.net/smartcontract/benchmark"
	"golang.org/x/net/context"
)

type DbTest struct {
	name      string
	events    []event.Event
	blockSize int
	ctx       context.Context
}

func (et DbTest) Name() string {
	return et.name
}

func (et DbTest) Transaction() *transaction.Transaction {
	return nil
}

func (et DbTest) Run(sCtx state.TimedQueryStateContext, _ *testing.B) error {
	if len(et.events) == 0 {
		return nil
	}
	be, _, err := sCtx.GetEventDB().MergeEvents(
		et.events,
		et.events[0].BlockNumber,
		"mock block hash"+et.name,
		et.blockSize,
	)
	if err != nil {
		return err
	}
	var gs *event.Snapshot
	p := int64(-1)
	_, err = event.Work(et.ctx, gs, be, &p)
	if err != nil {
		return err
	}
	return nil
}

type DbEventTest struct{ DbTest }

func (et DbEventTest) Run(sCtx state.TimedQueryStateContext, _ *testing.B) error {
	if len(et.events) == 0 {
		return nil
	}
	be, _, err := sCtx.GetEventDB().MergeEvents(
		et.events,
		et.events[0].BlockNumber,
		"mock block hash"+et.name,
		et.blockSize,
	)
	if err != nil {
		return err
	}
	p := int64(-1)
	_, err = sCtx.GetEventDB().WorkEvents(et.ctx, be, &p)
	if err != nil {
		return err
	}
	return nil
}

type DbAggregateTest struct{ DbTest }

func (et DbAggregateTest) Run(sCtx state.TimedQueryStateContext, _ *testing.B) error {
	if len(et.events) == 0 {
		return nil
	}
	be, _, err := sCtx.GetEventDB().MergeEvents(
		et.events,
		et.events[0].BlockNumber,
		"mock block hash"+et.name,
		et.blockSize,
	)
	if err != nil {
		return err
	}
	var gs *event.Snapshot
	_, err = sCtx.GetEventDB().WorkAggregates(gs, be)
	if err != nil {
		return err
	}
	return nil
}

func GetBenchmarkTestSuite(eventsMap map[string][]event.Event, source bk.Source) bk.TestSuite {
	var edbTests []bk.BenchTestI
	for key, events := range eventsMap {
		switch source {
		case bk.EventDatabase:
			edbTests = append(edbTests, DbTest{
				name:      key,
				events:    events,
				blockSize: 1,
				ctx:       context.TODO(),
			})
		case bk.EventDatabaseEvents:
			edbTests = append(edbTests, DbEventTest{
				DbTest{
					name:      key,
					events:    events,
					blockSize: 1,
					ctx:       context.TODO(),
				},
			})
		case bk.EventDatabaseAggregates:
			edbTests = append(edbTests, DbAggregateTest{
				DbTest{
					name:      key,
					events:    events,
					blockSize: 1,
					ctx:       context.TODO(),
				},
			})
		default:
			log.Fatal("invalid source for event database benchmark", source)
		}

	}
	return bk.TestSuite{
		Source:     bk.EventDatabase,
		Benchmarks: edbTests,
		ReadOnly:   true,
	}
}
