package magmasc

import (
	"encoding/json"
	"math/big"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

type (
	Billing struct {
		Amount      int64            `json:"amount"`
		DataUsage   *DataUsage       `json:"data_usage"`
		SessionID   datastore.Key    `json:"session_id"`
		CompletedAt common.Timestamp `json:"completed_at,omitempty"`
	}
)

var (
	// Make sure tokenPool implements Serializable interface.
	_ util.Serializable = (*Billing)(nil)
)

// CalcAmount calculates and sets the billing Amount value by given price.
// NOTE: the cost value must be represented in token units per mega byte.
// NOTE: math/big must be used to avoid inaccuracies of floating point operations.
func (m *Billing) CalcAmount(terms ProviderTerms) {
	var amount int64

	price := terms.GetPrice()
	if price > 0 {
		bps := big.NewFloat(0).Add( // data usage summary: UploadBytes + DownloadBytes
			big.NewFloat(0).SetUint64(m.DataUsage.UploadBytes),
			big.NewFloat(0).SetUint64(m.DataUsage.DownloadBytes),
		)

		amount, _ = big.NewFloat(0).Mul(
			big.NewFloat(0).Quo(bps, big.NewFloat(million)), // data usage in mega bytes
			big.NewFloat(0).SetInt64(price),                 // cost per mega byte
		).Int64() // rounded of amount for data usage multiplied by cost
	}

	if minCost := terms.GetMinCost(); amount < minCost {
		amount = minCost
	}

	m.Amount = amount
}

// Decode implements util.Serializable interface.
func (m *Billing) Decode(blob []byte) error {
	var bill Billing
	if err := json.Unmarshal(blob, &bill); err != nil {
		return errDecodeData.WrapErr(err)
	}

	if bill.DataUsage != nil {
		if err := bill.DataUsage.validate(); err != nil {
			return errDecodeData.WrapErr(err)
		}
		m.DataUsage = bill.DataUsage
	}

	m.Amount = bill.Amount
	m.SessionID = bill.SessionID
	m.CompletedAt = bill.CompletedAt

	return nil
}

// Encode implements util.Serializable interface.
func (m *Billing) Encode() []byte {
	blob, _ := json.Marshal(m)
	return blob
}

// uid returns uniq id used to saving billing data into chain state.
func (m *Billing) uid(scID datastore.Key) datastore.Key {
	return "sc:" + scID + ":datausage:" + m.SessionID
}

// validate checks given data usage is correctness for the billing.
func (m *Billing) validate(dataUsage *DataUsage) (err error) {
	switch {
	case dataUsage == nil:
		err = errNew(errCodeBadRequest, "data usage required")

	case m.SessionID != dataUsage.SessionID:
		err = errNew(errCodeBadRequest, "invalid session_id")

	case m.DataUsage == nil:
		return nil // is valid: have no data usage yet

	case m.DataUsage.SessionTime > dataUsage.SessionTime:
		err = errNew(errCodeBadRequest, "invalid session_time")

	case m.DataUsage.UploadBytes > dataUsage.UploadBytes:
		err = errNew(errCodeBadRequest, "invalid upload_bytes")

	case m.DataUsage.DownloadBytes > dataUsage.DownloadBytes:
		err = errNew(errCodeBadRequest, "invalid download_bytes")

	default:
		return nil // is valid - everything is ok
	}

	return errInvalidDataUsage.WrapErr(err)
}
