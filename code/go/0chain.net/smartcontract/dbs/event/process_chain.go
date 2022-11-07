package event

import (
	"fmt"

	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

func (edb *EventDb) addChainEvent(event Event) error {
	switch EventTag(event.Tag) {
	case TagAddBlock:
		block, ok := fromEvent[Block](event.Data)
		if !ok {
			return ErrInvalidEventData
		}
		logging.Logger.Debug("saving block event", zap.String("id", block.Hash))

		return edb.addOrUpdateBlock(*block)
	default:
		return fmt.Errorf("unrecognised event %v", event)
	}
}
