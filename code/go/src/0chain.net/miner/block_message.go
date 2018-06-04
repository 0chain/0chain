package miner

import (
	"0chain.net/block"
	"0chain.net/node"
	"0chain.net/round"
)

const (
	MessageStartRound         = 1
	MessageVerify             = 2
	MessageVerificationTicket = 3
	MessageConsensus          = 4
)

/*BlockMessage - Used for the various messages that need to be handled to generate a block */
type BlockMessage struct {
	Type                    int
	Sender                  *node.Node
	Round                   *round.Round
	Block                   *block.Block
	BlockVerificationTicket *block.BlockVerificationTicket
	Consensus               *Consensus
}
