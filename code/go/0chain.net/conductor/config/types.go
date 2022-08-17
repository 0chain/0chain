package config

import "github.com/mitchellh/mapstructure"

// BlobberList represents the blobber_list directive state.
type BlobberList struct {
	ReturnError       bool   `json:"return_error" yaml:"return_error" mapstructure:"return_error"`
	SendWrongData     bool   `json:"send_wrong_data" yaml:"send_wrong_data" mapstructure:"send_wrong_data"`
	SendWrongMetadata bool   `json:"send_wrong_metadata" yaml:"send_wrong_metadata" mapstructure:"send_wrong_metadata"`
	NotRespond        bool   `json:"not_respond" yaml:"not_respond" mapstructure:"not_respond"`
	Adversarial       string `json:"adversarial" yaml:"adversarial" mapstructure:"adversarial"`
}

// NewBlobberList returns an entity of BlobberList
func NewBlobberList() *BlobberList {
	return &BlobberList{}
}

// Decode implements MapDecoder interface.
func (n *BlobberList) Decode(val interface{}) error {
	return mapstructure.Decode(val, n)
}

// BlobberDownload represents the blobber_download directive state.
type BlobberDownload struct {
	ReturnError bool   `json:"return_error" yaml:"return_error" mapstructure:"return_error"`
	NotRespond  bool   `json:"not_respond" yaml:"not_respond" mapstructure:"not_respond"`
	Adversarial string `json:"adversarial" yaml:"adversarial" mapstructure:"adversarial"`
}

// NewBlobberDownload returns an entity of BlobberDownload
func NewBlobberDownload() *BlobberDownload {
	return &BlobberDownload{}
}

// Decode implements MapDecoder interface.
func (n *BlobberDownload) Decode(val interface{}) error {
	return mapstructure.Decode(val, n)
}

// BlobberUpload represents the blobber_upload directive state.
type BlobberUpload struct {
	ReturnError bool   `json:"return_error" yaml:"return_error" mapstructure:"return_error"`
	NotRespond  bool   `json:"not_respond" yaml:"not_respond" mapstructure:"not_respond"`
	Adversarial string `json:"adversarial" yaml:"adversarial" mapstructure:"adversarial"`
}

// NewBlobberUpload returns an entity of BlobberUpload
func NewBlobberUpload() *BlobberUpload {
	return &BlobberUpload{}
}

// Decode implements MapDecoder interface.
func (n *BlobberUpload) Decode(val interface{}) error {
	return mapstructure.Decode(val, n)
}

// BlobberDelete represents the blobber_delete directive state.
type BlobberDelete struct {
	ReturnError bool   `json:"return_error" yaml:"return_error" mapstructure:"return_error"`
	NotRespond  bool   `json:"not_respond" yaml:"not_respond" mapstructure:"not_respond"`
	Adversarial string `json:"adversarial" yaml:"adversarial" mapstructure:"adversarial"`
}

// NewBlobberDelete returns an entity of BlobberDelete
func NewBlobberDelete() *BlobberDelete {
	return &BlobberDelete{}
}

// Decode implements MapDecoder interface.
func (n *BlobberDelete) Decode(val interface{}) error {
	return mapstructure.Decode(val, n)
}
