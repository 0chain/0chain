package miner

import (
	"0chain.net/block"
	"0chain.net/common"
	"0chain.net/node"
	"0chain.net/round"
)

const (
	MessageStartRound         = 0
	MessageVerify             = 1
	MessageVerificationTicket = 2
	MessageNotarization       = 3
)

/*BlockMessage - Used for the various messages that need to be handled to generate a block */
type BlockMessage struct {
	Type                    int
	Sender                  *node.Node
	Round                   *round.Round
	Block                   *block.Block
	BlockVerificationTicket *block.BlockVerificationTicket
	Notarization            *Notarization
}

var messageLookups = common.CreateLookups("start_round", "Start Round", "verify_block", "Verify Block", "verification_ticket", "Verification Ticket", "notarization", "Notarization")

func GetMessageLookup(msgType int) *common.Lookup {
	return messageLookups[msgType]
}
