package rand

import (
	"encoding/hex"
	"time"

	zmc "github.com/0chain/gosdk/zmagmacore/magmasc"
	ts "github.com/0chain/gosdk/zmagmacore/time"
	magma "github.com/magma/augmented-networks/accounting/protos"
	"golang.org/x/crypto/sha3"

	"0chain.net/core/encryption"
	"0chain.net/smartcontract/magmasc"
)

func Consumers(num int) []*zmc.Consumer {
	consumers := make([]*zmc.Consumer, num)
	for ind := range consumers {
		consumers[ind] = Consumer()
	}
	return consumers
}

func Consumer() *zmc.Consumer {
	id := String(32)
	hash := sha3.Sum256([]byte(id))
	return &zmc.Consumer{
		ID:    hex.EncodeToString(hash[:]),
		ExtID: "id:consumer:external:" + id,
		Host:  "host.consumer.local:" + id,
	}
}

func Providers(num int) []*zmc.Provider {
	provider := make([]*zmc.Provider, num)
	for ind := range provider {
		provider[ind] = Provider()
	}
	return provider
}

func Provider() *zmc.Provider {
	id := String(32)
	hash := sha3.Sum256([]byte(id))
	return &zmc.Provider{
		ID:    hex.EncodeToString(hash[:]),
		ExtID: "id:provider:external:" + id,
		Host:  "host.provider.local:" + id,
	}
}

func Acknowledgment(consumer *zmc.Consumer, provider *zmc.Provider) *zmc.Acknowledgment {
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
		sessionID = String(16)
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
					TxnHash:    encryption.Hash(String(32)),
					ToPool:     sessionID,
					Value:      terms.GetAmount(),
					FromClient: consumer.ID,
					ToClient:   magmasc.Address,
				},
			},
		},
	}
}
