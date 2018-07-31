package miner

import (
	"time"

	"0chain.net/block"
	"0chain.net/chain"
	"0chain.net/common"
	"0chain.net/node"
)

const (
	MessageStartRound         = 0
	MessageVerify             = 1
	MessageVerificationTicket = 2
	MessageNotarization       = 3
	MessageNotarizedBlock     = 4
)

/*BlockMessage - Used for the various messages that need to be handled to generate a block */
type BlockMessage struct {
	Type                    int
	Sender                  *node.Node
	Round                   *Round
	Block                   *block.Block
	BlockVerificationTicket *block.BlockVerificationTicket
	Notarization            *Notarization
	Timestamp               time.Time
	RetryCount              int8
}

/*NewBlockMessage - create a new block message */
func NewBlockMessage(messageType int, sender *node.Node, round *Round, block *block.Block) *BlockMessage {
	bm := &BlockMessage{}
	bm.Type = messageType
	bm.Sender = sender
	bm.Round = round
	bm.Block = block
	bm.Timestamp = time.Now()
	return bm
}

var messageLookups = common.CreateLookups("start_round", "Start Round", "verify_block", "Verify Block", "verification_ticket", "Verification Ticket", "notarization", "Notarization", "notarized_block", "Notarized Block")

/*GetMessageLookup - get the message type lookup */
func GetMessageLookup(msgType int) *common.Lookup {
	return messageLookups[msgType]
}

//ShouldRetry - tells whether this message should be retried by putting back into the channel
func (bm *BlockMessage) ShouldRetry() bool {
	if bm.RetryCount < 5 || time.Since(bm.Timestamp) < 5*chain.DELTA {
		return true
	}
	return false
}

//Retry - retry the block message by putting it back into the channel
func (bm *BlockMessage) Retry(bmc chan *BlockMessage) {
	go func() {
		duration := time.Since(bm.Timestamp)
		if duration < time.Millisecond {
			duration = 10 * time.Millisecond
		}
		duration *= 2
		time.Sleep(duration)
		bm.RetryCount++
		bmc <- bm
	}()
}
