package formatters

import "encoding/json"

type FreeStorageMarker struct {
	Assigner   string   `json:"assigner"`
	Recipient  string   `json:"recipient"`
	FreeTokens float64  `json:"free_tokens"`
	Nonce      int64    `json:"nonce"`
	Signature  string   `json:"signature"`
	Blobbers   []string `json:"blobbers"`
}

func (frm *FreeStorageMarker) Decode(b []byte) error {
	return json.Unmarshal(b, frm)
}

type FreeStorageAllocationInput struct {
	RecipientPublicKey string   `json:"recipient_public_key"`
	Marker             string   `json:"marker"`
	Blobbers           []string `json:"blobbers"`
}

func (frm *FreeStorageAllocationInput) Decode(b []byte) error {
	return json.Unmarshal(b, frm)
}
