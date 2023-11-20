package cases

import (
	"encoding/json"
)

type (
	// RoundInfo represents simple struct for reports containing round's information
	// needed for making tests checks.
	RoundInfo struct {
		Num                int64        `json:"num"`
		GeneratorsNum      int          `json:"generators_num"`
		RankedMiners       []string     `json:"ranked_miners"`
		FinalisedBlockHash string       `json:"finalised_block_hash"`
		ProposedBlocks     []*BlockInfo `json:"proposed_blocks"`
		NotarisedBlocks    []*BlockInfo `json:"notarised_blocks"`
		TimeoutCount       int          `json:"timeout_count"`
		RoundRandomSeed    int64        `json:"round_random_seed"`
		IsFinalised        bool         `json:"is_finalised"`
	}

	// BlockInfo represents simple struct for reports containing round's information
	// needed for making tests checks.
	BlockInfo struct {
		Hash                string                    `json:"hash"`
		PrevHash            string                    `json:"prev_hash"`
		Notarised           bool                      `json:"notarised"`
		VerificationStatus  int                       `json:"verification_status"`
		Rank                int                       `json:"rank"`
		VerificationTickets []*VerificationTicketInfo `json:"verification_tickets"`
	}

	// VerificationTicketInfo represents simple struct for reports containing verification ticket's information
	// needed for making tests checks.
	VerificationTicketInfo struct {
		VerifierID string `json:"verifier_id"`
	}
)

func (r *RoundInfo) blocks() map[string]*BlockInfo {
	blocks := make(map[string]*BlockInfo)
	for _, bl := range r.ProposedBlocks {
		blocks[bl.Hash] = bl
	}
	for _, bl := range r.NotarisedBlocks {
		blocks[bl.Hash] = bl
	}
	return blocks
}

// getNotarisedBlocks return ID of the node with provided parameters.
// If node with provided parameters is not found, returns "".
//
// 	Explaining type rank example:
//		RoundInfo.GeneratorsNum = 2
// 		len(RoundInfo.RankedMiners) = 4
// 		Generator0:	rank = 0; generator = true;	typeRank = 0.
// 		Generator1:	rank = 1; generator = true; typeRank = 1.
// 		Replica0:	rank = 2; generator = false; typeRank = 0.
// 		Replica0:	rank = 3; generator = false; typeRank = 1.
func (r *RoundInfo) getNodeID(generator bool, typeRank int) string {
	for rank, rankedMiner := range r.RankedMiners {
		isGenerator := rank < r.GeneratorsNum
		currTypeRank := rank
		if !isGenerator {
			currTypeRank = rank - r.GeneratorsNum
		}

		if isGenerator == generator && currTypeRank == typeRank {
			return rankedMiner
		}
	}

	return ""
}

// Encode encodes RoundInfo to bytes.
func (r *RoundInfo) Encode() ([]byte, error) {
	return json.Marshal(r)
}

// Decode decodes RoundInfo from bytes.
func (r *RoundInfo) Decode(blob []byte) error {
	return json.Unmarshal(blob, r)
}

// Encode encodes BlockInfo to bytes.
func (b *BlockInfo) Encode() ([]byte, error) {
	return json.Marshal(b)
}

// Decode decodes BlockInfo from bytes.
func (b *BlockInfo) Decode(blob []byte) error {
	return json.Unmarshal(blob, b)
}

type (
	// NotarisationInfo represents simple struct for reports containing notarisation's information
	// needed for making tests checks.
	NotarisationInfo struct {
		VerificationTickets []*VerificationTicketInfo `json:"verification_tickets"`
		BlockID             string                    `json:"block_id"`
		Round               int64                     `json:"round"`
	}
)

// Encode encodes NotarisationInfo to bytes.
func (n *NotarisationInfo) Encode() ([]byte, error) {
	return json.Marshal(n)
}

// Decode decodes NotarisationInfo from bytes.
func (n *NotarisationInfo) Decode(blob []byte) error {
	return json.Unmarshal(blob, n)
}
