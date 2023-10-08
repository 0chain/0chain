package event

import "encoding/json"

type EventMessage struct {
	Event
	MessageID int64 `json:"message_id"`
}


func NewEventMessage(event Event, counter int64) *EventMessage {
	em := &EventMessage{
		Event: event,
	}

	em.MessageID = em.generateId(counter)
	return em
}

func (em *EventMessage) Encode() ([]byte, error) {
	var (
		err error
		raw []byte
	)

	if raw, err = json.Marshal(em); err != nil {
		return nil, err
	}

	return raw, nil
}

func (em *EventMessage) generateId(counter int64) int64 {
	return counter
}
