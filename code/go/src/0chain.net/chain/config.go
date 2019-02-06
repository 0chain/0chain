package chain

import (
	"time"

	"0chain.net/datastore"
)

const (
	BlockProposalWaitStatic  = 0
	BlockProposalWaitDynamic = iota
)

//Config - chain Configuration
type Config struct {
	OwnerID               datastore.Key `json:"owner_id"`                  // Client who created this chain
	ParentChainID         datastore.Key `json:"parent_chain_id,omitempty"` // Chain from which this chain is forked off
	GenesisBlockHash      string        `json:"genesis_block_hash"`
	Decimals              int8          `json:"decimals"`                // Number of decimals allowed for the token on this chain
	BlockSize             int32         `json:"block_size"`              // Number of transactions in a block
	MinBlockSize          int32         `json:"min_block_size"`          // Number of transactions a block needs to have
	MaxByteSize           int64         `json:"max_byte_size"`           // Max number of bytes a block can have
	NumGenerators         int           `json:"num_generators"`          // Number of block generators
	NumReplicators        int           `json:"num_replicators"`         // Number of sharders that can store the block
	ThresholdByCount      int           `json:"threshold_by_count"`      // Threshold count for a block to be notarized
	ThresholdByStake      int           `json:"threshold_by_stake"`      // Stake threshold for a block to be notarized
	ValidationBatchSize   int           `json:"validation_size"`         // Batch size of txns for crypto verification
	TxnMaxPayload         int           `json:"transaction_max_payload"` // Max payload allowed in the transaction
	PruneStateBelowCount  int           `json:"prune_state_below_count"` // Prune state below these many rounds
	RoundRange            int64         `json:"round_range"`             // blocks are stored in separate directory for each range of rounds
	BlocksToSharder       int           `json:"blocks_to_sharder"`       // send finalized or notarized blocks to sharder
	VerificationTicketsTo int           `json:"verification_tickets_to"` // send verification tickets to generator or all miners

	BlockProposalMaxWaitTime time.Duration `json:"block_proposal_max_wait_time"` // max time to wait to receive a block proposal
	BlockProposalWaitMode    int8          `json:"block_proposal_wait_mode"`     // wait time for the block proposal is static (0) or dynamic (1)

	ReuseTransactions bool `json:"reuse_txns"` // indicates if transactions from unrelated blocks can be reused

	ClientSignatureScheme string `json:"client_signature_scheme"` // indicates which signature scheme is being used

	MinActiveSharders    int `json:"min_active_sharders"`    // Minimum active sharders required to validate blocks
	MinActiveReplicators int `json:"min_active_replicators"` // Minimum active replicators of a block that should be active to verify the block
}
