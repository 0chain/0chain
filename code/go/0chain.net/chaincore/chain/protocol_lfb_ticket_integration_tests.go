//go:build integration_tests
// +build integration_tests

package chain

import (
	"context"
	"log"

	"0chain.net/chaincore/node"
	crpc "0chain.net/conductor/conductrpc"
	"0chain.net/core/datastore"
)

func SetupLFBTicketSender() {
	setupLFBTicketSender()

	LFBTicketSender = LFBTicketSenderMiddleWare(LFBTicketSender)
}

func LFBTicketSenderMiddleWare(sender node.EntitySendHandler) node.EntitySendHandler {
	return func(entity datastore.Entity) node.SendHandler {
		lfbTicket, ok := entity.(*LFBTicket)
		if !ok {
			log.Panicf("Conductor: unexpected entity implementation")
		}

		if isIgnoringSendingLFBTicket(lfbTicket) {
			return func(_ context.Context, n *node.Node) bool { return true }
		}

		return sender(entity)
	}
}

func isIgnoringSendingLFBTicket(ticket *LFBTicket) bool {
	missingLFBTicketsCfg := crpc.Client().State().MissingLFBTicket
	if missingLFBTicketsCfg == nil {
		return false
	}

	return ticket.Round >= missingLFBTicketsCfg.OnRound &&
		node.Self.Type == node.NodeTypeSharder
}
