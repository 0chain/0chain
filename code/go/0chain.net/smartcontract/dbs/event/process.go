package event

import (
	"fmt"

	"0chain.net/core/logging"
	"go.uber.org/zap"
)

const (
	TypeError = "error"
	TypeStats = "stats"
)

const (
	TagNewChallenge    = "new_challenge"
	TagRemoveChallenge = "remove_challenge"
)

func (edb *EventDb) AddEvents(events []Event) {
	newEvents := edb.removeDuplicate(events)
	logging.Logger.Info("piers processing events",
		zap.Any("events", newEvents))

	edb.addEvents(newEvents)
	for _, event := range newEvents {
		var err error = nil
		switch event.Type {
		case TypeStats:
			err = edb.addStat(event)
		default:
		}
		if err != nil {
			logging.Logger.Error(
				"event could not be processed",
				zap.Any("event", event),
				zap.Error(err),
			)
		}
	}
}

func (edb *EventDb) addStat(event Event) error {
	switch event.Tag {
	case TagNewChallenge:
		var challenge Challenge
		logging.Logger.Info("piers event db adding",
			zap.Any("challenge", event.Data))
		return challenge.Add(edb, []byte(event.Data))
	case TagRemoveChallenge:
		logging.Logger.Info("piers event db removing",
			zap.Any("challenge", event.Data))
		return edb.removeChallenge(event.Data)
	default:
		return fmt.Errorf("unrecognised event %v", event)
	}
}
