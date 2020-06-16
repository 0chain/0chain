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
	// Of nodes (sing only competing blocks of this nodes, for example)
	Of []NodeName `json:"of" yaml:"of" mapstructure:"of"`
}

// Unmarshal with given name and from given map[interface{}]interface{}
// by mapstructure package.
func (b *Bad) Unmarshal(name string, val interface{}) (err error)) {
	if err = mapstructure.Decode(val, b); err != nil {
		return fmt.Errorf("invalid '%s' argument type: %T, "+
			"decoding error: %v", name, val, err)
	}
	if len(b.By) == 0{
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

// IsGood returns true if the Bad is nil or given name is in Good list.
func (b *Bad) IsGood(name NodeName) bool {
	return b == nil || isInList(b.Good, name)
}

// IsBad returns true if the Bad is nil or given name is in Bad list.
func (b *Bad) IsBad(name NodeName) bool {
	return b == nil || isInList(b.Bad, name)
}

// IsBy returns true if given name is in By list.
func (b *Bad) IsBy(name NodeName) bool {
	return isInList(b.By, name)
}

// IsBad returns true if the Bad is nil or given name is in Of list.
func (b *Bad) IsOf(name NodeName) bool {
	return b == nil || isInList(b.Of, name)
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
	// Byzantine view change
	MPK        struct{ Bad } // mpk
	Shares     struct{ Bad } // shares
	Signatures struct{ Bad } // signatures
	Publish    struct{ Bad } // publish
)
