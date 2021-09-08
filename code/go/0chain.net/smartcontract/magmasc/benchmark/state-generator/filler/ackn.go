package filler

import (
	"time"

	zmc "github.com/0chain/gosdk/zmagmacore/magmasc"
	ts "github.com/0chain/gosdk/zmagmacore/time"
	magma "github.com/magma/augmented-networks/accounting/protos"

	"0chain.net/core/encryption"
	"0chain.net/smartcontract/magmasc"
	"0chain.net/smartcontract/magmasc/benchmark/rand"
)

func createAcknowledgment(sessionID string, consumer *zmc.Consumer, provider *zmc.Provider) *zmc.Acknowledgment {
	var (
		now   = time.Now().Format(time.RFC3339Nano)
		apID  = "ap-id" + now
		terms = zmc.ProviderTerms{
			AccessPointID:   apID,
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
		}
	)
	return &zmc.Acknowledgment{
		SessionID:     sessionID,
		AccessPointID: apID,
		Consumer:      consumer,
		Provider:      provider,
		Terms:         terms,
		TokenPool: &zmc.TokenPool{
			ID:      sessionID,
			Balance: terms.GetAmount(),
			PayerID: consumer.ID,
			PayeeID: provider.ID,
			Transfers: []zmc.TokenPoolTransfer{
				{
					TxnHash:    encryption.Hash(rand.String(32)),
					ToPool:     sessionID,
					Value:      terms.GetAmount(),
					FromClient: consumer.ID,
					ToClient:   magmasc.Address,
				},
			},
		},
	}
}
