package config

// send_vrf
// send_round_timeout
// send_competing_block
// send_block
// send_verification_ticket
// send_notarized_block

// The BadOnly is common bad / only sending configuration.
type BadOnly struct {
	// By these nodes.
	By []NodeName `json:"by" yaml:"by" mapstructure:"by"`
	// Good to these nodes.
	Good []NodeName `json:"good" yaml:"good" mapstructure:"good"`
	// Bad to these nodes.
	Bad []NodeName `json:"bad" yaml:"bad" mapstructure:"bad"`
}

// common Byzantine scenarios
type (
	VRF                               struct{ Bad }
	RoundTimeoutCounter               struct{ Bad } // ?
	CompetingBlock                    struct{ Bad }
	Block                             struct{ Bad }
	VerificationTicket                struct{ Bad }
	NotarizedBlock                    struct{ Bad }
	Share                             struct{ Bad }
	Signature                         struct{ Bad }
	Signatures                        struct{ Bad } // ?
	RoundTimeout                      struct{ Bad } // ?
	CompetingBlockWhenNotAGenerator   struct{ Bad }
	SignsTheCompetingBlocks           struct{ Bad }
	DoubleSpendATransaction           struct{ Bad }
	DifferentHashThanTheBlockHash     struct{ Bad }
	DifferentPrivateKeyToSignTheBlock struct{ Bad }
	HashTheBlockIncorrectly           struct{ Bad }
	CompetingNotarizedBlock           struct{ Bad }
)
