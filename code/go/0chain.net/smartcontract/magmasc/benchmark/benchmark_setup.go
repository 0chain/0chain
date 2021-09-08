package benchmark

import (
	zmc "github.com/0chain/gosdk/zmagmacore/magmasc"
	ts "github.com/0chain/gosdk/zmagmacore/time"
	magma "github.com/magma/augmented-networks/accounting/protos"

	"0chain.net/chaincore/block"
	chain "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
	"0chain.net/smartcontract/magmasc"
	"0chain.net/smartcontract/magmasc/benchmark/mpt"
)

var (
	sc *SC

	scStress          *SC
	sciStress         *SCI
	stressNamePostfix string
)

func Setup(sci chain.StateContextI) {
	// setup smart contract and default sci for simple using
	sc = makeSC(mkTempDir())

	registerStaticConsumer(sc.magma, sci)
	registerStaticProvider(sc.magma, sci)
	initSession(sc.magma, sci, staticAcknowledgmentForInitializedSession())
	startSession(sc.magma, sci, staticAcknowledgmentForStartedSession())

	// setup smart contract and sci for stress using
	dbDir, sciDbDir, sciLogDir := createTempDirsForStress()
	scStress = makeSC(dbDir)

	stressMptRoot, err := mpt.GetRoot(scStress.magma.GetDB())
	panicIfErr(err)
	sciStress = makeSCI(sciDbDir, sciLogDir, stressMptRoot)

	registerStaticConsumer(scStress.magma, sciStress.sci)
	registerStaticProvider(scStress.magma, sciStress.sci)
	initSession(scStress.magma, sciStress.sci, staticAcknowledgmentForInitializedSession())
	startSession(scStress.magma, sciStress.sci, staticAcknowledgmentForStartedSession())
	stressNamePostfix = getSourcePostfix(scStress.magma, sciStress.sci)
}

func registerStaticConsumer(sc *magmasc.MagmaSmartContract, sci chain.StateContextI) {
	var (
		statCons = staticConsumer()

		txn = transaction.Transaction{
			ClientID: statCons.ID,
		}
	)

	_, err := sc.Execute(&txn, zmc.ConsumerRegisterFuncName, statCons.Encode(), sci)
	panicIfErr(err)
}

func staticConsumer() *zmc.Consumer {
	return &zmc.Consumer{
		ID:    encryption.Hash("consumer-static-id"),
		ExtID: "consumer-static-ext-id",
		Host:  "consumer-static-host",
	}
}

func registerStaticProvider(sc *magmasc.MagmaSmartContract, sci chain.StateContextI) {
	var (
		statProv = staticProvider()

		txn = transaction.Transaction{
			ClientID: statProv.ID,
		}
	)
	_, err := sc.Execute(&txn, zmc.ProviderRegisterFuncName, statProv.Encode(), sci)
	panicIfErr(err)
}

func staticProvider() *zmc.Provider {
	return &zmc.Provider{
		ID:    encryption.Hash("provider-static-id"),
		ExtID: "provider-static-ext-id",
		Host:  "provider-static-host",
	}
}

func initSession(sc *magmasc.MagmaSmartContract, sci chain.StateContextI, ackn *zmc.Acknowledgment) {
	var (
		txn = transaction.Transaction{
			ClientID: ackn.Provider.ID,
		}
	)
	_, err := sc.Execute(&txn, "provider_session_init", ackn.Encode(), sci) // todo change func on const
	panicIfErr(err)

	_, err = sci.GetState().Insert(
		util.Path(ackn.Consumer.ID),
		createState(state.Balance(ackn.Terms.GetAmount())),
	)
	panicIfErr(err)
}

func createState(bal state.Balance) *state.State {
	balance := &state.State{}
	err := balance.SetTxnHash("0000000000000000000000000000000000000000000000000000000000000000")
	panicIfErr(err)
	balance.Balance = bal
	return balance
}

func staticAcknowledgmentForInitializedSession() *zmc.Acknowledgment {
	apid := "ackn-access-point-static-id"
	sessID := "ackn-initialized-static-id"
	return &zmc.Acknowledgment{
		SessionID:     sessID,
		AccessPointID: apid,
		Billing: zmc.Billing{
			DataUsage: zmc.DataUsage{
				DownloadBytes: 3000000,
				UploadBytes:   2000000,
				SessionID:     sessID,
				SessionTime:   1 * 60, // 1 minute
			},
		},
		Consumer: staticConsumer(),
		Provider: staticProvider(),
		Terms: zmc.ProviderTerms{
			AccessPointID:   apid,
			Price:           0.1,
			PriceAutoUpdate: 0.001,
			MinCost:         0.5,
			Volume:          0,
			QoS: &magma.QoS{
				DownloadMbps: 5.4321,
				UploadMbps:   1.2345,
			},
			QoSAutoUpdate: &zmc.QoSAutoUpdate{
				DownloadMbps: 0.001,
				UploadMbps:   0.001,
			},
			ProlongDuration: 1 * 60 * 60,              // 1 hour
			ExpiredAt:       ts.Now() + (1 * 60 * 60), // 1 hour from now
		},
	}
}

func startSession(sc *magmasc.MagmaSmartContract, sci chain.StateContextI, ackn *zmc.Acknowledgment) {
	initSession(sc, sci, ackn)

	var (
		txn = transaction.Transaction{
			ClientID: ackn.Consumer.ID,
		}

		txnSci = chain.NewStateContext(
			&block.Block{},
			sci.GetState(),
			&state.Deserializer{},
			&txn,
			func(*block.Block) []string { return []string{} },
			func() *block.Block { return &block.Block{} },
			func() *block.MagicBlock { return &block.MagicBlock{} },
			func() encryption.SignatureScheme { return &encryption.BLS0ChainScheme{} },
		)
	)

	_, err := sc.Execute(&txn, zmc.ConsumerSessionStartFuncName, ackn.Encode(), txnSci)
	panicIfErr(err)
}

func staticAcknowledgmentForStartedSession() *zmc.Acknowledgment {
	apid := "ackn-access-point-static-id"
	sessID := "ackn-started-static-id"
	return &zmc.Acknowledgment{
		SessionID:     sessID,
		AccessPointID: apid,
		Billing: zmc.Billing{
			DataUsage: zmc.DataUsage{
				DownloadBytes: 3000000,
				UploadBytes:   2000000,
				SessionID:     sessID,
				SessionTime:   1 * 60, // 1 minute
			},
		},
		Consumer: staticConsumer(),
		Provider: staticProvider(),
		Terms: zmc.ProviderTerms{
			AccessPointID:   apid,
			Price:           0.1,
			PriceAutoUpdate: 0.001,
			MinCost:         0.5,
			Volume:          0,
			QoS: &magma.QoS{
				DownloadMbps: 5.4321,
				UploadMbps:   1.2345,
			},
			QoSAutoUpdate: &zmc.QoSAutoUpdate{
				DownloadMbps: 0.001,
				UploadMbps:   0.001,
			},
			ProlongDuration: 1 * 60 * 60,              // 1 hour
			ExpiredAt:       ts.Now() + (1 * 60 * 60), // 1 hour from now
		},
	}
}
