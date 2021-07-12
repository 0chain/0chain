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
func (m *Billing) CalcAmount(cost uint64) {
	var amount int64
	if cost > 0 {
		bps := big.NewFloat(0).Add( // data usage summary: UploadBytes + DownloadBytes
			big.NewFloat(0).SetUint64(m.DataUsage.UploadBytes),
			big.NewFloat(0).SetUint64(m.DataUsage.DownloadBytes),
		)

		amount, _ = big.NewFloat(0).Mul(
			big.NewFloat(0).Quo(bps, big.NewFloat(million)), // data usage in mega bytes
			big.NewFloat(0).SetUint64(cost),                 // cost per mega byte
		).Int64() // rounded of amount for data usage multiplied by cost
	}

	m.Amount = amount
}

// Decode implements util.Serializable interface.
func (m *Billing) Decode(blob []byte) error {
	var bill Billing
	if err := json.Unmarshal(blob, &bill); err != nil {
		return errDecodeData.WrapErr(err)
	}
	if err := bill.DataUsage.validate(); err != nil {
		return errDecodeData.WrapErr(err)
	}

	m.Amount = bill.Amount
	m.DataUsage = bill.DataUsage
	m.SessionID = bill.SessionID

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
func (m *Billing) validate(dataUsage *DataUsage) error {
	switch {
	case dataUsage == nil: // is invalid: data usage cannon be nil
	case m.SessionID != dataUsage.SessionID: // is invalid: invalid session id

	case m.DataUsage == nil: // is valid: have no data usage yet
		return nil

	// is invalid cases
	case m.DataUsage.SessionTime > dataUsage.SessionTime:
	case m.DataUsage.UploadBytes > dataUsage.UploadBytes:
	case m.DataUsage.DownloadBytes > dataUsage.DownloadBytes:

	default: // is valid: everything is ok
		return nil
	}

	return errDataUsageInvalid
}
