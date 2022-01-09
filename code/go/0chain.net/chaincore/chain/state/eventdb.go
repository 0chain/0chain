package state

import (
	"0chain.net/chaincore/state"
	"0chain.net/smartcontract/dbs/event"
	"encoding/json"
	"errors"
	"fmt"
)

func (sc *StateContext) emitHandleAddTransfer(t *state.Transfer) error {
	if sc.GetEventDB() == nil {
		return errors.New(event.ErrNoEventDb)
	}
	toUser := event.User{
		UserID:        t.ToClientID,
		Amount:        t.Amount,
		TransactionID: sc.txn.Hash,
	}
	data, err := json.Marshal(toUser)
	if err != nil {
		return fmt.Errorf("marshalling to client: %v", err)
	}
	sc.EmitEvent(event.TypeStats, event.TagIncreaseUserBalanceByAmount, toUser.UserID, string(data))

	fromUser := event.User{
		UserID:        t.ClientID,
		Amount:        t.Amount,
		TransactionID: sc.txn.Hash,
	}
	data, err = json.Marshal(fromUser)
	if err != nil {
		return fmt.Errorf("marshalling from client: %v", err)
	}
	sc.EmitEvent(event.TypeStats, event.TagDecreaseUserBalanceByAmount, fromUser.UserID, string(data))

	return nil
}

func (sc *StateContext) emitHandleAddMint(m *state.Mint) error {
	if sc.GetEventDB() == nil {
		return errors.New(event.ErrNoEventDb)
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
	sc.EmitEvent(event.TypeStats, event.TagIncreaseUserBalanceByAmount, toUser.UserID, string(data))

	return nil
}
