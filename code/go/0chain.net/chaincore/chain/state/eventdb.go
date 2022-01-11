package state

import (
	"0chain.net/chaincore/state"
	"0chain.net/smartcontract/dbs/event"
	"context"
	"encoding/json"
	"fmt"
)

func (sc *StateContext) addTransferToDb(t *state.Transfer) error {
	if sc.GetEventDB() == nil {
		return event.ErrNoEventDb
	}
	var events []event.Event
	toUser := event.User{
		UserID:        t.ToClientID,
		Amount:        t.Amount,
		TransactionID: sc.txn.Hash,
	}
	data, err := json.Marshal(toUser)
	if err != nil {
		return fmt.Errorf("marshalling to client: %v", err)
	}
	events = append(events, event.Event{
		BlockNumber: sc.block.Round,
		TxHash:      sc.txn.Hash,
		Type:        int(event.TypeStats),
		Tag:         int(event.TagIncreaseUserBalanceByAmount),
		Index:       toUser.UserID,
		Data:        string(data),
	})

	fromUser := event.User{
		UserID:        t.ClientID,
		Amount:        t.Amount,
		TransactionID: sc.txn.Hash,
	}
	data, err = json.Marshal(fromUser)
	if err != nil {
		return fmt.Errorf("marshalling from client: %v", err)
	}

	events = append(events, event.Event{
		BlockNumber: sc.block.Round,
		TxHash:      sc.txn.Hash,
		Type:        int(event.TypeStats),
		Tag:         int(event.TagDecreaseUserBalanceByAmount),
		Index:       fromUser.UserID,
		Data:        string(data),
	})

	sc.GetEventDB().AddEvents(context.TODO(), events)

	return nil
}

func (sc *StateContext) addMintToDb(m *state.Mint) error {

	if sc.GetEventDB() == nil {
		return event.ErrNoEventDb
	}

	toUser := event.User{
		UserID:        m.ToClientID,
		Amount:        m.Amount,
		TransactionID: sc.txn.Hash,
	}
	data, err := json.Marshal(toUser)
	if err != nil {
		return fmt.Errorf("marshalling to client: %v", err)
	}

	var events = []event.Event{{
		BlockNumber: sc.block.Round,
		TxHash:      sc.txn.Hash,
		Type:        int(event.TypeStats),
		Tag:         int(event.TagIncreaseUserBalanceByAmount),
		Index:       toUser.UserID,
		Data:        string(data),
	},
	}

	sc.GetEventDB().AddEvents(context.TODO(), events)

	return nil
}
