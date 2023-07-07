package benchmark

import (
	"testing"

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
	be, tx, err := sCtx.GetEventDB().MergeEvents(
		et.ctx,
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
	_, err = sCtx.GetEventDB().Work(et.ctx, gs, be, &p)
	if err != nil {
		return err
	}
	tx = tx           //piers
	err = tx.Commit() //piers
	return err
}

func GetBenchmarkTestSuite(eventsMap map[string][]event.Event) bk.TestSuite {
	var edbTests []bk.BenchTestI
	for key, events := range eventsMap {
		edbTests = append(edbTests, DbTest{
			name:      key,
			events:    events,
			blockSize: 1,
			ctx:       context.TODO(),
		})
	}
	return bk.TestSuite{
		Name:       "events",
		Source:     bk.EventDatabase,
		Benchmarks: edbTests,
		ReadOnly:   true,
	}
}
