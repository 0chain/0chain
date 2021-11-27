package miner

import (
	"sync"
	"time"

	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/round"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/core/common"
)

// MessageVRFShare -
const (
	MessageVRFShare           = 0
	MessageVerify             = iota
	MessageVerificationTicket = iota
	MessageNotarization       = iota
	MessageNotarizedBlock     = iota
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
	VRFShare                *round.VRFShare
	VRFShares               map[string]*round.VRFShare
}

/*NewBlockMessage - create a new block message */
func NewBlockMessage(messageType int, sender *node.Node, round *Round, block *block.Block, vrfShares map[string]*round.VRFShare) *BlockMessage {
	bm := &BlockMessage{}
	bm.Type = messageType
	bm.Sender = sender
	bm.Round = round
	bm.Block = block
	bm.VRFShares = vrfShares
	bm.Timestamp = time.Now()
	return bm
}

var messageLookups = common.CreateLookups("vrf_share", "VRF Share",
	"verify_block", "Verify Block",
	"verification_ticket", "Verification Ticket",
	"notarization", "Notarization",
	"notarized_block", "Notarized Block")

// lock for protecting the messageLookups
var messageLock sync.RWMutex

/*GetMessageLookup - get the message type lookup */
func GetMessageLookup(msgType int) *common.Lookup {
	messageLock.RLock()
	msg := messageLookups[msgType]
	messageLock.RUnlock()
	return msg
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
		if duration > time.Second {
			duration = time.Second
		}
		time.Sleep(duration)
		bm.RetryCount++
		bmc <- bm
	}()
}
