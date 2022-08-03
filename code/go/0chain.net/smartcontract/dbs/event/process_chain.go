package event

import "fmt"

func (edb *EventDb) addChainEvent(event Event) error {
	switch EventTag(event.Tag) {
	case TagAddBlock:
		block, ok := fromEvent[Block](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		return edb.addBlock(*block)
	default:
		return fmt.Errorf("unrecognised event %v", event)
	}
}
