package magmasc

import (
	"encoding/json"

	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

type (
	// DataUsage represents session data sage implementation.
	DataUsage struct {
		DownloadBytes uint64        `json:"download_bytes"`
		UploadBytes   uint64        `json:"upload_bytes"`
		SessionID     datastore.Key `json:"session_id"`
		SessionTime   uint32        `json:"session_time"`
	}
)

var (
	// Make sure tokenPool implements Serializable interface.
	_ util.Serializable = (*DataUsage)(nil)
)

// Decode implements util.Serializable interface.
func (m *DataUsage) Decode(blob []byte) error {
	var dataUsage DataUsage
	if err := json.Unmarshal(blob, &dataUsage); err != nil {
		return errDecodeData.WrapErr(err)
	}
	if err := dataUsage.validate(); err != nil {
		return errDecodeData.WrapErr(err)
	}

	m.DownloadBytes = dataUsage.DownloadBytes
	m.UploadBytes = dataUsage.UploadBytes
	m.SessionID = dataUsage.SessionID
	m.SessionTime = dataUsage.SessionTime

	return nil
}

// Encode implements util.Serializable interface.
func (m *DataUsage) Encode() []byte {
	blob, _ := json.Marshal(m)
	return blob
}

// validate checks DataUsage for correctness.
func (m *DataUsage) validate() error {
	switch { // is invalid
	case m.SessionID == "":

	default: // is valid
		return nil
	}

	return errDataUsageInvalid
}
