package config

import (
	"strings"
	"time"
)

// ExpectMagicBlock represents expected magic block.
type ExpectMagicBlock struct {
	// Number is expected Magic Block number. Use of MB number is
	// more stable for the tests, since miners can vote for restart
	// DKG process from start.
	Number Number `json:"number" yaml:"number" mapstructure:"number"`
	// Round ignored if it's zero. If set a positive value, then this
	// round is expected.
	Round Round `json:"round" yaml:"round" mapstructure:"round"`
	// RoundNextVCAfter used in combination with wait_view_change.remember_round
	// that remember round with some name. This directive expects next VC round
	// after the remembered one. For example, if round 340 has remembered as
	// "enter_miner5", then "round_next_vc_after": "enter_miner5", expects
	// 500 round (next VC after the remembered round). Empty string ignored.
	RoundNextVCAfter RoundName `json:"round_next_vc_after" yaml:"round_next_vc_after" mapstructure:"round_next_vc_after"`
	// Sharders expected in MB.
	Sharders []NodeName `json:"sharders" yaml:"sharders" mapstructure:"sharders"`
	// Miners expected in MB.
	Miners []NodeName `json:"miners" yaml:"miners" mapstructure:"miners"`
	// Sharders Count expected in MB.
	ShardersCount int `json:"sharders_count" yaml:"sharders_count" mapstructure:"sharders_count"`
	// Miners Count expected in MB.
	MinersCount int `json:"miners_count" yaml:"miners_count" mapstructure:"miners_count"`
}

// IsZero returns true if the MagicBlock is empty.
func (emb *ExpectMagicBlock) IsZero() bool {
	return emb.Number == 0 &&
		emb.Round == 0 &&
		emb.RoundNextVCAfter == "" &&
		len(emb.Sharders) == 0 &&
		len(emb.Miners) == 0
}

// WaitViewChange flow configuration.
type WaitViewChange struct {
	RememberRound    RoundName        `json:"remember_round" yaml:"remember_round" mapstructure:"remember_round"`
	ExpectMagicBlock ExpectMagicBlock `json:"expect_magic_block" yaml:"expect_magic_block" mapstructure:"expect_magic_block"`
}

// IsZero returns true if the ViewChagne is empty.
func (vc *WaitViewChange) IsZero() bool {
	return vc.RememberRound == "" &&
		vc.ExpectMagicBlock.IsZero()
}

// WaitPhase flow configuration.
type WaitPhase struct {
	// Phase to wait for (number).
	Phase Phase `json:"phase" yaml:"phase" mapstructure:"phase"`
	// ViewChangeRound after which the phase expected (and before next VC),
	// value can be an empty string for any VC.
	ViewChangeRound RoundName `json:"view_change_round" yaml:"view_change_round" mapstructure:"view_change_round"`
}

// IsZero returns true if the WaitPhase is empty.
func (wp *WaitPhase) IsZero() bool {
	return wp.Phase == 0 && wp.ViewChangeRound == ""
}

// WaitRound waits a round.
type WaitRound struct {
	Round        Round     `json:"round" yaml:"round" mapstructure:"round"`
	Name         RoundName `json:"name" yaml:"name" mapstructure:"name"`
	Shift        Round     `json:"shift" yaml:"shift" mapstructure:"shift"`
	ForbidBeyond bool      `json:"forbid_beyond" yaml:"forbid_beyond" mapstructure:"forbid_beyond"`
}

func (wr *WaitRound) IsZero() bool {
	return wr.Round == 0 && wr.Name == "" && wr.Shift == 0
}

// WaitContibuteMpk wait for MPK contributing of a node.
type WaitContributeMpk struct {
	Miner NodeName `json:"miner" yaml:"miner" mapstructure:"miner"`
}

func (wcm *WaitContributeMpk) IsZero() bool {
	return wcm.Miner == ""
}

// WaitShareSignsOrShares waits for SOSS of a node.
type WaitShareSignsOrShares struct {
	Miner NodeName `json:"miner" yaml:"miner" mapstructure:"miner"`
}

func (wssos *WaitShareSignsOrShares) IsZero() bool {
	return wssos.Miner == ""
}

// WaitAdd used to wait for add_miner and add_sharder SC calls.
type WaitAdd struct {
	Miners      []NodeName `json:"miners" yaml:"miners" mapstructure:"miners"`
	Sharders    []NodeName `json:"sharders" yaml:"sharders" mapstructure:"sharders"`
	Blobbers    []NodeName `json:"blobbers" yaml:"blobbers" mapstructure:"blobbers"`
	Authorizers []NodeName `json:"authorizers" yaml:"authorizers" mapstructure:"authorizers"`
	Start       bool       `json:"start" yaml:"start" mapstructure:"start"`
}

func (wa *WaitAdd) IsZero() bool {
	return len(wa.Miners) == 0 && len(wa.Sharders) == 0 && len(wa.Blobbers) == 0 && len(wa.Authorizers) == 0
}

func (wa *WaitAdd) Take(name NodeName) (ok bool) {
	if strings.Contains(string(name), "miner") {
		return wa.TakeMiner(name)
	} else if strings.Contains(string(name), "sharder") {
		return wa.TakeSharder(name)
	} else if strings.Contains(string(name), "blobber") {
		return wa.TakeBlobber(name)
	} else if strings.Contains(string(name), "authorizer") {
		return wa.TakeAuthorizer(name)
	}

	return false
}

func (wa *WaitAdd) TakeMiner(name NodeName) (ok bool) {
	for i, minerName := range wa.Miners {
		if minerName == name {
			wa.Miners = append(wa.Miners[:i], wa.Miners[i+1:]...)
			return true
		}
	}
	return // false
}

func (wa *WaitAdd) TakeSharder(name NodeName) (ok bool) {
	for i, sharderName := range wa.Sharders {
		if sharderName == name {
			wa.Sharders = append(wa.Sharders[:i], wa.Sharders[i+1:]...)
			return true
		}
	}
	return
}

func (wa *WaitAdd) TakeBlobber(name NodeName) (ok bool) {
	for i, blobberName := range wa.Blobbers {
		if blobberName == name {
			wa.Blobbers = append(wa.Blobbers[:i], wa.Blobbers[i+1:]...)
			return true
		}
	}
	return
}

func (wa *WaitAdd) TakeAuthorizer(name NodeName) (ok bool) {
	for i, authorizerName := range wa.Authorizers {
		if authorizerName == name {
			wa.Authorizers = append(wa.Authorizers[:i], wa.Authorizers[i+1:]...)
			return true
		}
	}
	return
}

type WaitNoProgress struct {
	Start time.Time
	Until time.Time
}

func (wnp *WaitNoProgress) IsZero() bool {
	return (*wnp) == (WaitNoProgress{})
}

type WaitNoViewChainge struct {
	Round Round `json:"round" yaml:"round" mapstructure:"round"`
}

func (wnvc *WaitNoViewChainge) IsZero() bool {
	return (*wnvc) == (WaitNoViewChainge{})
}

// WaitSharderKeep used to wait for sharder_keep
// SC successful function call.
type WaitSharderKeep struct {
	Sharders []NodeName `json:"sharders" yaml:"sharders" mapstructure:"sharders"`
}

func (wsk *WaitSharderKeep) IsZero() bool {
	return len(wsk.Sharders) == 0
}

func (wsk *WaitSharderKeep) TakeSharder(name NodeName) (ok bool) {
	for i, sharderName := range wsk.Sharders {
		if sharderName == name {
			wsk.Sharders = append(wsk.Sharders[:i], wsk.Sharders[i+1:]...)
			return true
		}
	}
	return
}
