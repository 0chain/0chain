package wallet

import (
	"0chain.net/node"
	"time"
)

var PutClientSender node.EntitySendHandler
var GetClientSender node.EntitySendHandler
var TransactionSender node.EntitySendHandler

func SetupC2MSenders() {
	options := &node.SendOptions{Timeout: 2 * time.Second, MaxRelayLength: 0, CurrentRelayLength: 0, Compress: false}
	GetClientSender = node.SendEntityHandler("/v1/client/get", options)

	options = &node.SendOptions{Timeout: 2 * time.Second, MaxRelayLength: 0, CurrentRelayLength: 0, Compress: false}
	PutClientSender = node.SendEntityHandler("/v1/client/put", options)

	options = &node.SendOptions{Timeout: 2 * time.Second, MaxRelayLength: 0, CurrentRelayLength: 0, Compress: false}
	TransactionSender = node.SendEntityHandler("/v1/transaction/put", options)
}
