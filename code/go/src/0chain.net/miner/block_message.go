package miner

import "0chain.net/block"

const (
	MessageVerify             = 1
	MessageVerificationTicket = 2
	MessageConsensus          = 3
)

/*BlockMessage - Used for the various messages that need to be handled to generate a block */
type BlockMessage struct {
	Type                    int
	Block                   *block.Block
	BlockVerificationTicket *block.BlockVerificationTicket
	Consensus               *Consensus
}
