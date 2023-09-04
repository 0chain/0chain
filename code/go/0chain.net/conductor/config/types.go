package config

import (
	"errors"
	"fmt"
	"time"

	"0chain.net/conductor/conductrpc/stats"
	"github.com/mitchellh/mapstructure"
)

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

type GenerateChallege struct {
	BlobberID         string        `json:"blobber_id" mapstructure:"blobber_id"`
	ChallengeDuration time.Duration `json:"chal_dur" mapstructure:"chal_dur"`
	ExpectedStatus    int           `json:"expected_status" mapstructure:"expected_status"` // 1 -> "pass" or 0-> "fail"
	// Id of a miner so that only this miner will generate challenge
	MinerID                   string `json:"miner" mapstructure:"miner"`
	WaitOnBlobberCommit       bool
	WaitOnChallengeGeneration bool
	WaitForChallengeStatus    bool
}

func (g *GenerateChallege) Decode(val interface{}) error {
	dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook:       mapstructure.StringToTimeDurationHookFunc(),
		WeaklyTypedInput: true,
		Result:           g,
	})
	if err != nil {
		return err
	}

	err = dec.Decode(val)
	if err != nil {
		return err
	}

	if g.ExpectedStatus == 0 || g.ExpectedStatus == 1 {
		return nil
	}

	return fmt.Errorf("expected either '0' or '1', got: %d", g.ExpectedStatus)
}

func NewGenerateChallenge() *GenerateChallege {
	return &GenerateChallege{}
}

type CheckFileMetaRoot struct {
	RequireSameRoot bool `mapstructure:"require_same_root"`
}

func (c *CheckFileMetaRoot) Decode(val interface{}) error {
	if c == nil {
		return errors.New("cannot decode into nil pointer")
	}
	return mapstructure.Decode(val, c)
}

func NewCheckFileMetaRoot() *CheckFileMetaRoot {
	return &CheckFileMetaRoot{}
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

type NotifyOnBlockGeneration struct {
	Enable bool	`json:"enable" yaml:"enable" mapstructure:"enable"`
}

func (nbg *NotifyOnBlockGeneration) Decode(val interface{}) error {
	return mapstructure.Decode(val, nbg)
}

type RenameCommitControl struct {
	Fail bool
	Nodes []NodeID
}

func BuildFailRenameCommit(nodes []NodeID) *RenameCommitControl {
	return &RenameCommitControl{
		Fail: true,
		Nodes: nodes,
	}
}

func BuildDisableFailRenameCommit(nodes []NodeID) *RenameCommitControl {
	return &RenameCommitControl{
		Fail: false,
		Nodes: nodes,
	}
}

type WaitValidatorTicket struct {
	ValidatorName string `json:"validator_name" yaml:"validator_name" mapstructure:"validator_name"`
	ValidatorId string `json:"-" yaml:"-" mapstructure:"-"`
}

func NewWaitValidatorTicket() *WaitValidatorTicket {
	return &WaitValidatorTicket{}
}

type MonitorAggregates struct {
	SharderIds []string `json:"sharders" yaml:"sharders" mapstructure:"sharders"`
	MinerIds []string	`json:"miners" yaml:"miners" mapstructure:"miners"`
	BlobberIds []string	`json:"blobbers" yaml:"blobbers" mapstructure:"blobbers"`
	ValidatorIds []string	`json:"validators" yaml:"validators" mapstructure:"validators"`
	AuthorizerIds []string	`json:"authorizers" yaml:"authorizers" mapstructure:"authorizers"`
	MonitorGlobal bool	`json:"global" yaml:"global" mapstructure:"global"`

	SharderFields []string	`json:"sharder_fields" yaml:"sharder_fields" mapstructure:"sharder_fields"`
	MinerFields []string	`json:"miner_fields" yaml:"miner_fields" mapstructure:"miner_fields"`
	BlobberFields []string	`json:"blobber_fields" yaml:"blobber_fields" mapstructure:"blobber_fields"`
	ValidatorFields []string	`json:"validator_fields" yaml:"validator_fields" mapstructure:"validator_fields"`
	AuthorizerFields []string	`json:"authorizer_fields" yaml:"authorizer_fields" mapstructure:"authorizer_fields"`
	GlobalFields []string	`json:"global_fields" yaml:"global_fields" mapstructure:"global_fields"`
}

type CheckAggregateChange struct {
	ProviderType stats.ProviderType `json:"provider_type" yaml:"provider_type" mapstructure:"provider_type"`
	ProviderId string `json:"provider_id" yaml:"provider_id" mapstructure:"provider_id"`
	Key string `json:"key" yaml:"key" mapstructure:"key"`
	Monotonicity stats.Monotonicity `json:"monotonicity" yaml:"monotonicity" mapstructure:"monotonicity"`
}

type CheckAggregateComparison struct {
	ProviderType stats.ProviderType `json:"provider_type" yaml:"provider_type" mapstructure:"provider_type"`
	ProviderId string `json:"provider_id" yaml:"provider_id" mapstructure:"provider_id"`
	Key string `json:"key" yaml:"key" mapstructure:"key"`
	Comparison stats.Comparison `json:"comparison" yaml:"comparison" mapstructure:"comparison"`
	RValue int64 `json:"rvalue" yaml:"rvalue" mapstructure:"rvalue"`
}