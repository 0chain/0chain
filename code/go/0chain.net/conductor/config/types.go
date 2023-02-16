package config

import "github.com/mitchellh/mapstructure"

// AdversarialAuthorizer represents the adversarial_authorizer directive state.
type AdversarialAuthorizer struct {
	ID              string `json:"id" yaml:"id" mapstructure:"id"`
	SendFakedTicket bool   `json:"send_faked_ticket" yaml:"send_faked_ticket" mapstructure:"send_faked_ticket"`
}

// NewAdversarialAuthorizer returns an entity of AdversarialAuthorizer
func NewAdversarialAuthorizer() *AdversarialAuthorizer {
	return &AdversarialAuthorizer{}
}

// Decode implements MapDecoder interface.
func (n *AdversarialAuthorizer) Decode(val interface{}) error {
	return mapstructure.Decode(val, n)
}

// LockNotarizationAndSendNextRoundVRF represents the lock_notarization_and_send_next_round_vrf directive state.
type LockNotarizationAndSendNextRoundVRF struct {
	Round       int    `json:"round" yaml:"round" mapstructure:"round"`
	Adversarial string `json:"adversarial" yaml:"adversarial" mapstructure:"adversarial"`
}

// NewLockNotarizationAndSendNextRoundVRF returns an entity of LockNotarizationAndSendNextRoundVRF
func NewLockNotarizationAndSendNextRoundVRF() *LockNotarizationAndSendNextRoundVRF {
	return &LockNotarizationAndSendNextRoundVRF{}
}

// Decode implements MapDecoder interface.
func (n *LockNotarizationAndSendNextRoundVRF) Decode(val interface{}) error {
	return mapstructure.Decode(val, n)
}

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

// AdversarialValidator represents the blobber_delete directive state.
type AdversarialValidator struct {
	ID                 string `json:"id" yaml:"id" mapstructure:"id"`
	FailValidChallenge bool   `json:"fail_valid_challenge" yaml:"fail_valid_challenge" mapstructure:"fail_valid_challenge"`
	DenialOfService    bool   `json:"denial_of_service" yaml:"denial_of_service" mapstructure:"denial_of_service"`
	PassAllChallenges  bool   `json:"pass_all_challenges" yaml:"pass_all_challenges" mapstructure:"pass_all_challenges"`
}

// NewAdversarialValidator returns an entity of AdversarialValidator
func NewAdversarialValidator() *AdversarialValidator {
	return &AdversarialValidator{}
}

// Decode implements MapDecoder interface.
func (n *AdversarialValidator) Decode(val interface{}) error {
	return mapstructure.Decode(val, n)
}

// CollectVerificationTicketsWhenMissedVRF represents the collect_verification_tickets_when_missing_vrf directive state.
type CollectVerificationTicketsWhenMissedVRF struct {
	Miner string `json:"miner" yaml:"miner" mapstructure:"miner"`
	Round int    `json:"round" yaml:"round" mapstructure:"round"`
}

// NewCollectVerificationTicketsWhenMissedVRF returns an entity of CollectVerificationTicketsWhenMissedVRF
func NewCollectVerificationTicketsWhenMissedVRF() *CollectVerificationTicketsWhenMissedVRF {
	return &CollectVerificationTicketsWhenMissedVRF{}
}

// Decode implements MapDecoder interface.
func (n *CollectVerificationTicketsWhenMissedVRF) Decode(val interface{}) error {
	return mapstructure.Decode(val, n)
}
