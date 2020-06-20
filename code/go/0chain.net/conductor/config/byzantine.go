package config

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
)

// The Bad is common bad / only sending configuration.
type Bad struct {
	// By these nodes.
	By []NodeName `json:"by" yaml:"by" mapstructure:"by"`
	// Good to these nodes.
	Good []NodeName `json:"good" yaml:"good" mapstructure:"good"`
	// Bad to these nodes.
	Bad []NodeName `json:"bad" yaml:"bad" mapstructure:"bad"`
}

// Unmarshal with given name and from given map[interface{}]interface{}
// by mapstructure package.
func (b *Bad) Unmarshal(name string, val interface{}) (err error) {
	if err = mapstructure.Decode(val, b); err != nil {
		return fmt.Errorf("invalid '%s' argument type: %T, "+
			"decoding error: %v", name, val, err)
	}
	if len(b.By) == 0 {
		return fmt.Errorf("empty 'by' field of '%s'", name)
	}
	return
}

// Is given name in given names list.
func isInList(ids []NodeName, id NodeName) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

type Namer interface {
	Name(NodeID) NodeName
}

// IsGood returns true if the Bad is nil or given name is in Good list.
func (b *Bad) IsGood(state Namer, id string) bool {
	return b == nil || isInList(b.Good, state.Name(NodeID(id)))
}

// IsBad returns true if the Bad is nil or given name is in Bad list.
func (b *Bad) IsBad(state Namer, id string) bool {
	return b == nil || isInList(b.Bad, state.Name(NodeID(id)))
}

// IsBy returns true if given name is in By list.
func (b *Bad) IsBy(state Namer, id string) bool {
	return isInList(b.By, state.Name(NodeID(id)))
}

// IsCompetingRoundGenerator returns true for a group competing block generator.
// The node shouldn't be a generator. For a group of block generators (if the By
// field is a list of many items, they are the group) one of the nodes is
// competing round generator. It uses special algorithm choosing one of the By
// list.
func (b *Bad) IsCompetingRoundGenerator(state Namer, id string,
	round int64) (ok bool) {

	if b == nil {
		return false
	}

	var i = int(round % int64(len(b.By)))
	return b.By[i] == state.Name(NodeID(id))
}

// IsCompetingGroupMember returns true if given id is one of By list. The
// method regards nil, unlike the IsBy.
func (b *Bad) IsCompetingGroupMember(state Namer, id string) (ok bool) {

	if b == nil {
		return true // sign every block
	}

	var name = state.Name(NodeID(id))
	for _, member := range b.By {
		if member == name {
			return true
		}
	}
	return // false
}

// common Byzantine scenarios
type (
	// Byzantine blockchain
	VRFS                        struct{ Bad } // vrfs
	RoundTimeout                struct{ Bad } // round_timeout
	CompetingBlock              struct{ Bad } // competing_block
	SignOnlyCompetingBlocks     struct{ Bad } // sign_only_competing_blocks
	DoubleSpendTransaction      struct{ Bad } // double_spend_transaction
	WrongBlockSignHash          struct{ Bad } // wrong_block_sign_hash
	WrongBlockSignKey           struct{ Bad } // wrong_block_sign_key
	WrongBlockHash              struct{ Bad } // wrong_block_hash
	VerificationTicket          struct{ Bad } // verification_ticket
	WrongVerificationTicketHash struct{ Bad } // wrong_verification_ticket_hash
	WrongVerificationTicketKey  struct{ Bad } // wrong_verification_ticket_key
	WrongNotarizedBlockHash     struct{ Bad } // wrong_notarized_block_hash
	WrongNotarizedBlockKey      struct{ Bad } // wrong_notarized_block_key
	NotarizeOnlyCompetingBlock  struct{ Bad } // notarize_only_competing_block
	NotarizedBlock              struct{ Bad } // notarized_block
	// Byzantine blockchain sharders side
	FinalizedBlock    struct{ Bad } // finalized_block
	MagicBlock        struct{ Bad } // magic_block
	VerifyTransaction struct{ Bad } // verify_transaction (to clients)
	SCState           struct{ Bad } // sc_state (to clients)
	// Byzantine view change
	MPK        struct{ Bad } // mpk
	Shares     struct{ Bad } // shares
	Signatures struct{ Bad } // signatures
	Publish    struct{ Bad } // publish
)
