package magmasc

import (
	"encoding/json"

	chain "0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

type (
	// Consumer represents consumers node stored in block chain.
	Consumer struct {
		ID    datastore.Key `json:"id"`
		ExtID datastore.Key `json:"ext_id"`
		Host  datastore.Key `json:"host,omitempty"`
	}
)

var (
	// Make sure Consumer implements Serializable interface.
	_ util.Serializable = (*Consumer)(nil)
)

// Decode implements util.Serializable interface.
func (m *Consumer) Decode(blob []byte) error {
	var consumer Consumer
	if err := json.Unmarshal(blob, &consumer); err != nil {
		return errDecodeData.WrapErr(err)
	}
	if err := consumer.validate(); err != nil {
		return errDecodeData.WrapErr(err)
	}

	m.ID = consumer.ID
	m.ExtID = consumer.ExtID
	m.Host = consumer.Host

	return nil
}

// Encode implements util.Serializable interface.
func (m *Consumer) Encode() []byte {
	blob, _ := json.Marshal(m)
	return blob
}

// GetType returns Consumer's type.
func (m *Consumer) GetType() string {
	return consumerType
}

// validate checks Consumer for correctness.
// If it is not return errInvalidConsumer.
func (m *Consumer) validate() (err error) {
	switch { // is invalid
	case m.ExtID == "":
		err = errNew(errCodeBadRequest, "consumer external id is required")

	default:
		return nil // is valid
	}

	return errInvalidConsumer.WrapErr(err)
}

// consumerFetch extracts Consumer stored in state.StateContextI
// or returns error if blockchain state does not contain it.
func consumerFetch(scID, id datastore.Key, sci chain.StateContextI) (*Consumer, error) {
	data, err := sci.GetTrieNode(nodeUID(scID, id, consumerType))
	if err != nil {
		return nil, err
	}

	consumer := Consumer{}
	if err = consumer.Decode(data.Encode()); err != nil {
		return nil, errDecodeData.WrapErr(err)
	}

	return &consumer, nil
}
