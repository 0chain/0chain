package magmasc

import (
	"encoding/json"
	"math/big"
	"time"

	magma "github.com/magma/augmented-networks/accounting/protos"

	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/util"
)

type (
	// ProviderTerms represents data of provider and services terms.
	ProviderTerms struct {
		Terms
		QoS magma.QoS `json:"qos"`
	}

	// Terms represents data of provider terms for a session.
	Terms struct {
		Price           float32          `json:"price"`             // tokens per Megabyte
		MinCost         float32          `json:"min_cost"`          // minimal cost for a session
		Volume          int64            `json:"volume"`            // bytes per a session
		AutoUpdatePrice float32          `json:"auto_update_price"` // price change on auto update
		AutoUpdateQoS   AutoUpdateQoS    `json:"auto_update_qos"`   // qos change on auto update
		ProlongDuration time.Duration    `json:"prolong_duration"`  // duration in seconds to prolong the terms
		ExpiredAt       common.Timestamp `json:"expired_at"`        // timestamp till a session valid
	}

	// AutoUpdateQoS represents data of qos terms on auto update.
	AutoUpdateQoS struct {
		DownloadMbps float32 `json:"download_mbps"` //
		UploadMbps   float32 `json:"upload_mbps"`   //
	}
)

var (
	// Make sure ProviderTerms implements Serializable interface.
	_ util.Serializable = (*ProviderTerms)(nil)
)

// Decode implements util.Serializable interface.
func (m *ProviderTerms) Decode(blob []byte) error {
	var terms ProviderTerms
	if err := json.Unmarshal(blob, &terms); err != nil {
		return errDecodeData.WrapErr(err)
	}
	if err := terms.validate(); err != nil {
		return errDecodeData.WrapErr(err)
	}

	m.Price = terms.Price
	m.MinCost = terms.MinCost
	m.Volume = terms.Volume
	m.AutoUpdatePrice = terms.AutoUpdatePrice
	m.AutoUpdateQoS = terms.AutoUpdateQoS
	m.ProlongDuration = terms.ProlongDuration
	m.ExpiredAt = terms.ExpiredAt
	m.QoS.UploadMbps = terms.QoS.UploadMbps
	m.QoS.DownloadMbps = terms.QoS.DownloadMbps

	return nil
}

// Encode implements util.Serializable interface.
func (m *ProviderTerms) Encode() []byte {
	blob, _ := json.Marshal(m)
	return blob
}

// GetAmount returns calculated amount value of provider terms.
// NOTE: math/big must be used to avoid inaccuracies of floating point operations.
func (m *ProviderTerms) GetAmount() (amount state.Balance) {
	price := m.GetPrice()
	if price > 0 {
		amount = state.Balance(price * m.GetVolume())
	}

	return amount
}

// GetMinCost returns calculated min cost value of provider terms.
// NOTE: math/big must be used to avoid inaccuracies of floating point operations.
func (m *ProviderTerms) GetMinCost() (cost int64) {
	if m.MinCost > 0 {
		cost, _ = big.NewFloat(0).Mul( // convert to token units price
			big.NewFloat(billion),
			big.NewFloat(float64(m.MinCost)),
		).Int64() // rounded value of price multiplied by volume
	}

	return cost
}

// GetPrice returns calculated price value of provider terms.
// NOTE: the price value will be represented in token units per mega byte.
// NOTE: math/big must be used to avoid inaccuracies of floating point operations.
func (m *ProviderTerms) GetPrice() (price int64) {
	if m.Price > 0 {
		price, _ = big.NewFloat(0).Mul( // convert to token units price
			big.NewFloat(billion),
			big.NewFloat(float64(m.Price)),
		).Int64() // rounded value of price multiplied by volume
	}

	return price
}

// GetVolume returns value of the provider terms volume.
// If the Volume is empty it will be calculates by the provider terms.
// NOTE: math/big must be used to avoid inaccuracies of floating point operations.
func (m *ProviderTerms) GetVolume() int64 {
	if m.Volume == 0 {
		mbps := big.NewFloat(0).Add( // provider terms summary: UploadMbps + DownloadMbps
			big.NewFloat(float64(m.QoS.UploadMbps)),
			big.NewFloat(float64(m.QoS.DownloadMbps)),
		)

		m.Volume, _ = big.NewFloat(0).Mul(
			big.NewFloat(0).Quo(mbps, big.NewFloat(octet)),            // mega bytes per second
			big.NewFloat(0).SetInt64(int64(m.ExpiredAt-common.Now())), // duration in seconds
		).Int64() // rounded of bytes per second multiplied by duration
	}

	return m.Volume
}

// decrease makes automatically decrease provider terms by config.
// NOTE: math/big must be used to avoid inaccuracies of floating point operations.
func (m *ProviderTerms) decrease() *ProviderTerms {
	if m.AutoUpdateQoS.UploadMbps != 0 { // up the upload mbps quality of service
		upload := big.NewFloat(float64(m.QoS.UploadMbps))
		update := big.NewFloat(float64(m.AutoUpdateQoS.UploadMbps))
		m.QoS.UploadMbps, _ = big.NewFloat(0).Add(upload, update).Float32()
	}
	if m.AutoUpdateQoS.DownloadMbps != 0 { // up the download mbps quality of service
		download := big.NewFloat(float64(m.QoS.DownloadMbps))
		update := big.NewFloat(float64(m.AutoUpdateQoS.DownloadMbps))
		m.QoS.DownloadMbps, _ = big.NewFloat(0).Add(download, update).Float32()
	}
	if m.AutoUpdatePrice != 0 { // prepare price and auto update value
		price := big.NewFloat(float64(m.Price))
		update := big.NewFloat(float64(m.AutoUpdatePrice))
		if price.Cmp(update) == 1 { // check if the price is greater than the value of auto update
			m.Price, _ = big.NewFloat(0).Sub(price, update).Float32() // down the price
		}
	}
	if m.ProlongDuration != 0 { // prolong expire of terms
		m.ExpiredAt += common.Timestamp(m.ProlongDuration)
	}

	m.Volume = 0 // the volume of terms must to be zeroed

	return m
}

// increase makes automatically increase provider terms by config.
// NOTE: math/big must be used to avoid inaccuracies of floating point operations.
func (m *ProviderTerms) increase() *ProviderTerms {
	if m.AutoUpdatePrice != 0 { // up the price of service
		price := big.NewFloat(float64(m.Price))
		update := big.NewFloat(float64(m.AutoUpdatePrice))
		m.Price, _ = big.NewFloat(0).Add(price, update).Float32()
	}
	if m.AutoUpdateQoS.UploadMbps != 0 { // prepare upload mbps quality of service
		upload := big.NewFloat(float64(m.QoS.UploadMbps))
		update := big.NewFloat(float64(m.AutoUpdateQoS.UploadMbps))
		if upload.Cmp(update) == 1 { // down thr upload mbps quality of service
			m.QoS.UploadMbps, _ = big.NewFloat(0).Sub(upload, update).Float32()
		}
	}
	if m.AutoUpdateQoS.DownloadMbps != 0 { // prepare download mbps quality of service
		download := big.NewFloat(float64(m.QoS.DownloadMbps))
		update := big.NewFloat(float64(m.AutoUpdateQoS.DownloadMbps))
		if download.Cmp(update) == 1 { // down the download mbps quality of service
			m.QoS.DownloadMbps, _ = big.NewFloat(0).Sub(download, update).Float32()
		}
	}
	if m.ProlongDuration != 0 { // prolong expire of terms
		m.ExpiredAt += common.Timestamp(m.ProlongDuration)
	}

	m.Volume = 0 // the volume of terms must to be zeroed

	return m
}

// validate checks ProviderTerms for correctness.
// If it is not return errInvalidProviderTerms.
func (m *ProviderTerms) validate() (err error) {
	switch { // is invalid
	case m.QoS.UploadMbps <= 0:
		err = errNew(errCodeBadRequest, "invalid terms qos upload mbps")

	case m.QoS.DownloadMbps <= 0:
		err = errNew(errCodeBadRequest, "invalid terms qos download mbps")

	case m.ExpiredAt < common.Now()+providerTermsExpiredDuration:
		now := time.Now().Add(providerTermsExpiredDuration).Format(time.RFC3339)
		err = errNew(errCodeBadRequest, "expired at must be after "+now)

	default:
		return nil // is valid
	}

	return errInvalidProviderTerms.WrapErr(err)
}
