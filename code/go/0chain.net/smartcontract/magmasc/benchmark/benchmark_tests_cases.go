package benchmark

import (
	"testing"

	zmc "github.com/0chain/gosdk/zmagmacore/magmasc"

	chain "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	bk "0chain.net/smartcontract/benchmark"
	"0chain.net/smartcontract/magmasc"
	"0chain.net/smartcontract/magmasc/benchmark/rand"
)

const (
	magmaName = "Magma"
	del       = "_"
)

type (
	benchTestBase struct {
		input []byte

		txn *transaction.Transaction

		sc *magmasc.MagmaSmartContract

		name string

		funcName string
	}
)

func newBenchTestBase(input []byte, txn *transaction.Transaction, sc *magmasc.MagmaSmartContract, name, funcName string) *benchTestBase {
	return &benchTestBase{
		input:    input,
		txn:      txn,
		sc:       sc,
		name:     name,
		funcName: funcName,
	}
}

func (b benchTestBase) Name() string {
	return b.name
}

func (b benchTestBase) Transaction() *transaction.Transaction {
	return b.txn
}

func (b benchTestBase) Run(sci chain.StateContextI, _ *testing.B) {
	_, err := b.sc.Execute(b.txn, b.funcName, b.input, sci)
	panicIfErr(err)
}

type (
	// consumerRegisterBenchTest represents simple case for executing `consumer_register`.
	consumerRegisterBenchTest struct {
		*benchTestBase
	}
)

var (
	// Ensure consumerRegisterBenchTest implements interface.
	_ bk.BenchTestI = (*consumerRegisterBenchTest)(nil)
)

func newConsumerRegisterBenchTest() *consumerRegisterBenchTest {
	var (
		cons = rand.Consumer()

		txn = transaction.Transaction{
			ClientID: cons.ID,
		}
	)
	return &consumerRegisterBenchTest{
		benchTestBase: newBenchTestBase(
			cons.Encode(),
			&txn,
			sc.magma,
			magmaName+del+zmc.ConsumerRegisterFuncName,
			zmc.ConsumerRegisterFuncName,
		),
	}
}

type (
	// consumerRegisterStressBenchTest represents stress case for executing `consumer_register`.
	consumerRegisterStressBenchTest struct {
		*consumerRegisterBenchTest

		benchData bk.BenchData
	}
)

var (
	// Ensure consumerRegisterStressBenchTest implements interface.
	_ bk.BenchTestI = (*consumerRegisterStressBenchTest)(nil)
)

func newConsumerRegisterStressBenchTest(benchData bk.BenchData) *consumerRegisterStressBenchTest {
	bt := newConsumerRegisterBenchTest()
	bt.sc = scStress.magma
	bt.name = bt.name + stressNamePostfix

	return &consumerRegisterStressBenchTest{
		consumerRegisterBenchTest: bt,
		benchData:                 benchData,
	}
}

func (c *consumerRegisterStressBenchTest) Run(_ chain.StateContextI, b *testing.B) {
	b.StopTimer()
	_, sci := getBalances(
		c.txn,
		extractMpt(sciStress.mpt, sciStress.mpt.GetRoot()),
		c.benchData,
	)

	b.StartTimer()
	c.benchTestBase.Run(sci, b)
}

type (
	// providerRegisterBenchTest represents simple case for executing `provider_register`.
	providerRegisterBenchTest struct {
		*benchTestBase
	}
)

var (
	// Ensure providerRegisterBenchTest implements interface.
	_ bk.BenchTestI = (*providerRegisterBenchTest)(nil)
)

func newProviderRegisterBenchTest() *providerRegisterBenchTest {
	var (
		prov = rand.Provider()

		txn = transaction.Transaction{
			ClientID: prov.ID,
		}
	)
	return &providerRegisterBenchTest{
		benchTestBase: newBenchTestBase(
			prov.Encode(),
			&txn,
			sc.magma,
			magmaName+del+zmc.ProviderRegisterFuncName,
			zmc.ProviderRegisterFuncName,
		),
	}
}

type (
	// providerRegisterStressBenchTest represents stress case for executing `provider_register`.
	providerRegisterStressBenchTest struct {
		*providerRegisterBenchTest

		benchData bk.BenchData
	}
)

var (
	// Ensure providerRegisterStressBenchTest implements interface.
	_ bk.BenchTestI = (*providerRegisterStressBenchTest)(nil)
)

func newProviderRegisterStressBenchTest(benchData bk.BenchData) *providerRegisterStressBenchTest {
	bt := newProviderRegisterBenchTest()
	bt.sc = scStress.magma
	bt.name = bt.name + stressNamePostfix

	return &providerRegisterStressBenchTest{
		providerRegisterBenchTest: bt,
		benchData:                 benchData,
	}
}

func (c *providerRegisterStressBenchTest) Run(_ chain.StateContextI, b *testing.B) {
	b.StopTimer()
	_, sci := getBalances(
		c.txn,
		extractMpt(sciStress.mpt, sciStress.mpt.GetRoot()),
		c.benchData,
	)

	b.StartTimer()
	c.benchTestBase.Run(sci, b)
}

type (
	// consumerUpdateBenchTest represents a simple case for executing `consumer_update`.
	consumerUpdateBenchTest struct {
		*benchTestBase
	}
)

var (
	// Ensure consumerUpdateBenchTest implements interface.
	_ bk.BenchTestI = (*consumerUpdateBenchTest)(nil)
)

func newConsumerUpdateBenchTest() *consumerUpdateBenchTest {
	var (
		cons = staticConsumer()

		txn = transaction.Transaction{
			ClientID: cons.ID,
		}
	)
	cons.Host += "-updated"
	return &consumerUpdateBenchTest{
		benchTestBase: newBenchTestBase(
			cons.Encode(),
			&txn,
			sc.magma,
			magmaName+del+zmc.ConsumerUpdateFuncName,
			zmc.ConsumerUpdateFuncName,
		),
	}
}

type (
	// consumerUpdateStressBenchTest represents a stress case for executing `consumer_update`.
	consumerUpdateStressBenchTest struct {
		*consumerUpdateBenchTest

		benchData bk.BenchData
	}
)

var (
	// Ensure consumerUpdateBenchTest implements interface.
	_ bk.BenchTestI = (*consumerUpdateStressBenchTest)(nil)
)

func newConsumerUpdateStressBenchTest(benchData bk.BenchData) *consumerUpdateStressBenchTest {
	bt := newConsumerUpdateBenchTest()
	bt.sc = scStress.magma
	bt.name = bt.name + stressNamePostfix

	return &consumerUpdateStressBenchTest{
		consumerUpdateBenchTest: bt,
		benchData:               benchData,
	}
}

func (c consumerUpdateStressBenchTest) Run(_ chain.StateContextI, b *testing.B) {
	b.StopTimer()
	_, sci := getBalances(
		c.txn,
		extractMpt(sciStress.mpt, sciStress.mpt.GetRoot()),
		c.benchData,
	)

	b.StartTimer()

	c.benchTestBase.Run(sci, b)
}

type (
	// providerUpdateBenchTest represents a simple case for executing `provider_update`.
	providerUpdateBenchTest struct {
		*benchTestBase
	}
)

var (
	// Ensure providerUpdateBenchTest implements interface.
	_ bk.BenchTestI = (*providerUpdateBenchTest)(nil)
)

func newProviderUpdateBenchTest() *providerUpdateBenchTest {
	var (
		prov = staticProvider()

		txn = transaction.Transaction{
			ClientID: prov.ID,
		}
	)
	prov.Host += "-updated"
	return &providerUpdateBenchTest{
		benchTestBase: newBenchTestBase(
			prov.Encode(),
			&txn,
			sc.magma,
			magmaName+del+zmc.ProviderUpdateFuncName,
			zmc.ProviderUpdateFuncName,
		),
	}
}

type (
	// providerUpdateStressBenchTest represents a stress case for executing `provider_update`.
	providerUpdateStressBenchTest struct {
		*providerUpdateBenchTest

		benchData bk.BenchData
	}
)

var (
	// Ensure providerUpdateStressBenchTest implements interface.
	_ bk.BenchTestI = (*providerUpdateStressBenchTest)(nil)
)

func newProviderUpdateStressBenchTest(benchData bk.BenchData) *providerUpdateStressBenchTest {
	bt := newProviderUpdateBenchTest()
	bt.sc = scStress.magma
	bt.name = bt.name + stressNamePostfix

	return &providerUpdateStressBenchTest{
		providerUpdateBenchTest: bt,
		benchData:               benchData,
	}
}

func (c providerUpdateStressBenchTest) Run(_ chain.StateContextI, b *testing.B) {
	b.StopTimer()
	_, sci := getBalances(
		c.txn,
		extractMpt(sciStress.mpt, sciStress.mpt.GetRoot()),
		c.benchData,
	)

	b.StartTimer()
	c.benchTestBase.Run(sci, b)
}

type (
	// providerSessionInitBenchTest represents a simple case for executing `provider_session_init`.
	providerSessionInitBenchTest struct {
		*benchTestBase
	}
)

var (
	// Ensure providerSessionInitBenchTest implements interface.
	_ bk.BenchTestI = (*providerSessionInitBenchTest)(nil)
)

func newProviderSessionInitBenchTest() *providerSessionInitBenchTest {
	var (
		cons = staticConsumer()
		prov = staticProvider()
		ackn = rand.Acknowledgment(cons, prov)

		txn = transaction.Transaction{
			ClientID: prov.ID,
		}

		funcName = "provider_session_init" // todo change on constant func name
	)
	return &providerSessionInitBenchTest{
		benchTestBase: newBenchTestBase(
			ackn.Encode(),
			&txn,
			sc.magma,
			magmaName+del+funcName,
			funcName,
		),
	}
}

type (
	// providerSessionInitStressBenchTest represents a stress case for executing `provider_session_init`.
	providerSessionInitStressBenchTest struct {
		*providerSessionInitBenchTest

		benchData bk.BenchData
	}
)

var (
	// Ensure providerSessionInitStressBenchTest implements interface.
	_ bk.BenchTestI = (*providerSessionInitStressBenchTest)(nil)
)

func newProviderSessionInitStressBenchTest(bd bk.BenchData) *providerSessionInitStressBenchTest {
	bt := newProviderSessionInitBenchTest()
	bt.sc = scStress.magma
	bt.name = bt.name + stressNamePostfix

	return &providerSessionInitStressBenchTest{
		providerSessionInitBenchTest: bt,
		benchData:                    bd,
	}
}

func (p providerSessionInitStressBenchTest) Run(_ chain.StateContextI, b *testing.B) {
	b.StopTimer()
	_, sci := getBalances(
		p.txn,
		extractMpt(sciStress.mpt, sciStress.mpt.GetRoot()),
		p.benchData,
	)

	b.StartTimer()
	p.benchTestBase.Run(sci, b)
}

type (
	// consumerSessionStartBenchTest represents a simple case for executing `consumer_session_start`.
	consumerSessionStartBenchTest struct {
		*benchTestBase
	}
)

var (
	// Ensure consumerSessionStartBenchTest implements interface.
	_ bk.BenchTestI = (*consumerSessionStartBenchTest)(nil)
)

func newConsumerSessionStartBenchTest() *consumerSessionStartBenchTest {
	var (
		statAckn = staticAcknowledgmentForInitializedSession()

		txn = transaction.Transaction{
			ClientID: statAckn.Consumer.ID,
		}
	)
	return &consumerSessionStartBenchTest{
		benchTestBase: newBenchTestBase(
			statAckn.Encode(),
			&txn,
			sc.magma,
			magmaName+del+zmc.ConsumerSessionStartFuncName,
			zmc.ConsumerSessionStartFuncName,
		),
	}
}

type (
	// consumerSessionStartStressBenchTest represents a stress case for executing `consumer_session_start`.
	consumerSessionStartStressBenchTest struct {
		*consumerSessionStartBenchTest

		benchData bk.BenchData
	}
)

var (
	// Ensure consumerSessionStartStressBenchTest implements interface.
	_ bk.BenchTestI = (*consumerSessionStartStressBenchTest)(nil)
)

func newConsumerSessionStartStressBenchTest(bd bk.BenchData) *consumerSessionStartStressBenchTest {
	bt := newConsumerSessionStartBenchTest()
	bt.sc = scStress.magma
	bt.name = bt.name + stressNamePostfix

	return &consumerSessionStartStressBenchTest{
		consumerSessionStartBenchTest: bt,
		benchData:                     bd,
	}
}

func (c consumerSessionStartStressBenchTest) Run(_ chain.StateContextI, b *testing.B) {
	b.StopTimer()
	_, sci := getBalances(
		c.txn,
		extractMpt(sciStress.mpt, sciStress.mpt.GetRoot()),
		c.benchData,
	)

	b.StartTimer()
	c.benchTestBase.Run(sci, b)
}

type (
	// providerDataUsageBenchTest represents a simple case for executing `provider_data_usage`.
	providerDataUsageBenchTest struct {
		*benchTestBase
	}
)

var (
	// Ensure providerDataUsageBenchTest implements interface.
	_ bk.BenchTestI = (*providerDataUsageBenchTest)(nil)
)

func newProviderDataUsageBenchTest() *providerDataUsageBenchTest {
	var (
		statAckn = staticAcknowledgmentForStartedSession()

		txn = transaction.Transaction{
			ClientID: statAckn.Provider.ID,
		}
	)
	return &providerDataUsageBenchTest{
		benchTestBase: newBenchTestBase(
			statAckn.Encode(),
			&txn,
			sc.magma,
			magmaName+del+zmc.ProviderDataUsageFuncName,
			zmc.ProviderDataUsageFuncName,
		),
	}
}

type (
	// providerDataUsageStressBenchTest represents a stress case for executing `provider_data_usage`.
	providerDataUsageStressBenchTest struct {
		*providerDataUsageBenchTest

		benchData bk.BenchData
	}
)

var (
	// Ensure providerDataUsageBenchTest implements interface.
	_ bk.BenchTestI = (*providerDataUsageStressBenchTest)(nil)
)

func newProviderDataUsageStressBenchTest(bd bk.BenchData) *providerDataUsageStressBenchTest {
	bt := newProviderDataUsageBenchTest()
	bt.sc = scStress.magma
	bt.name = bt.name + stressNamePostfix

	return &providerDataUsageStressBenchTest{
		providerDataUsageBenchTest: bt,
		benchData:                  bd,
	}
}

func (c providerDataUsageStressBenchTest) Run(_ chain.StateContextI, b *testing.B) {
	b.StopTimer()
	_, sci := getBalances(
		c.txn,
		extractMpt(sciStress.mpt, sciStress.mpt.GetRoot()),
		c.benchData,
	)

	b.StartTimer()
	c.benchTestBase.Run(sci, b)
}

type (
	// consumerSessionStopBenchTest represents a simple case for executing `consumer_session_stop`.
	consumerSessionStopBenchTest struct {
		*benchTestBase
	}
)

var (
	// Ensure consumerSessionStopBenchTest implements interface.
	_ bk.BenchTestI = (*consumerSessionStopBenchTest)(nil)
)

func newConsumerSessionStopBenchTest() *consumerSessionStopBenchTest {
	var (
		statAckn = staticAcknowledgmentForStartedSession()

		txn = transaction.Transaction{
			ClientID: statAckn.Consumer.ID,
		}
	)
	return &consumerSessionStopBenchTest{
		benchTestBase: newBenchTestBase(
			statAckn.Encode(),
			&txn,
			sc.magma,
			magmaName+del+zmc.ConsumerSessionStopFuncName,
			zmc.ConsumerSessionStopFuncName,
		),
	}
}

type (
	// consumerSessionStopStressBenchTest represents a stress case for executing `consumer_session_stop`.
	consumerSessionStopStressBenchTest struct {
		*consumerSessionStopBenchTest

		benchData bk.BenchData
	}
)

var (
	// Ensure consumerSessionStopBenchTest implements interface.
	_ bk.BenchTestI = (*consumerSessionStopStressBenchTest)(nil)
)

func newConsumerSessionStopStressBenchTest(bd bk.BenchData) *consumerSessionStopStressBenchTest {
	bt := newConsumerSessionStopBenchTest()
	bt.sc = scStress.magma
	bt.name = bt.name + stressNamePostfix

	return &consumerSessionStopStressBenchTest{
		consumerSessionStopBenchTest: bt,
		benchData:                    bd,
	}
}

func (c consumerSessionStopStressBenchTest) Run(_ chain.StateContextI, b *testing.B) {
	b.StopTimer()
	_, sci := getBalances(
		c.txn,
		extractMpt(sciStress.mpt, sciStress.mpt.GetRoot()),
		c.benchData,
	)

	b.StartTimer()
	c.benchTestBase.Run(sci, b)
}
