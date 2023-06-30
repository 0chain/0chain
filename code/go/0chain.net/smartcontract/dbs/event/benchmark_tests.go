package event

import (
	"testing"

	"0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	bk "0chain.net/smartcontract/benchmark"
	"golang.org/x/net/context"
)

type DbTest struct {
	name      string
	events    []Event
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
	be, tx, err := sCtx.GetEventDB().mergeEvents(
		et.ctx,
		et.events,
		et.events[0].BlockNumber,
		"mock block hash"+et.name,
		et.blockSize,
	)
	if err != nil {
		return err
	}
	var gs *Snapshot
	p := int64(-1)
	_, err = sCtx.GetEventDB().work(et.ctx, gs, be, &p)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func GetBenchmarkTestSuite(eventsMap map[string][]Event) bk.TestSuite {
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

/* piers remove
func GetBenchmarkTests(
	eventsMap map[string][]Event,
) func(bk.BenchData, bk.SignatureScheme) bk.TestSuite {
	var edbTests []bk.BenchTestI
	for key, events := range eventsMap {
		edbTests = append(edbTests, DbTest{
			name:      key,
			events:    events,
			blockSize: 1,
			ctx:       context.TODO(),
		})
	}
	return func(_ bk.BenchData, _ bk.SignatureScheme) bk.TestSuite {
		return bk.TestSuite{
			Source:     bk.EventDatabase,
			Benchmarks: edbTests,
			ReadOnly:   true,
		}
	}
}
*/
