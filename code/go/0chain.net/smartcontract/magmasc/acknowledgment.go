package magmasc

import (
	"encoding/json"

	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

type (
	// Acknowledgment contains the necessary data obtained when the consumer
	// accepts the provider terms and stores in the state of the blockchain
	// as a result of performing the consumerAcceptTerms MagmaSmartContract function.
	Acknowledgment struct {
		SessionID     datastore.Key `json:"session_id"`
		AccessPointID datastore.Key `json:"access_point_id"`
		Consumer      *Consumer     `json:"consumer"`
		Provider      *Provider     `json:"provider"`
	}
)

var (
	// Make sure Acknowledgment implements Serializable interface.
	_ util.Serializable = (*Acknowledgment)(nil)
)

// Decode implements util.Serializable interface.
func (m *Acknowledgment) Decode(blob []byte) error {
	var ackn Acknowledgment
	if err := json.Unmarshal(blob, &ackn); err != nil {
		return errDecodeData.WrapErr(err)
	}
	if err := ackn.validate(); err != nil {
		return errDecodeData.WrapErr(err)
	}

	m.SessionID = ackn.SessionID
	m.AccessPointID = ackn.AccessPointID
	m.Consumer = ackn.Consumer
	m.Provider = ackn.Provider

	return nil
}

// Encode implements util.Serializable interface.
func (m *Acknowledgment) Encode() []byte {
	blob, _ := json.Marshal(m)
	return blob
}

// uid returns uniq id used to saving Acknowledgment into chain state.
func (m *Acknowledgment) uid(scID datastore.Key) datastore.Key {
	return "sc:" + scID + ":acknowledgment:" + m.SessionID
}

// validate checks Acknowledgment for correctness.
// If it is not return errInvalidAcknowledgment.
func (m *Acknowledgment) validate() (err error) {
	switch { // is invalid
	case m.SessionID == "":
		err = errNew(errCodeBadRequest, "session_id is required")

	case m.AccessPointID == "":
		err = errNew(errCodeBadRequest, "access_point_id is required")

	case m.Consumer == nil || m.Consumer.ExtID == "":
		err = errNew(errCodeBadRequest, "consumer ext_id is required")

	case m.Provider == nil || m.Provider.ExtID == "":
		err = errNew(errCodeBadRequest, "provider ext_id is required")

	default:
		return nil // is valid
	}

	return errInvalidAcknowledgment.WrapErr(err)
}
