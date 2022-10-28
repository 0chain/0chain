package event

import (
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
	"golang.org/x/net/context"
)

func (edb *EventDb) eventBlockController(_ context.Context) {
	var (
		currentRound   = int64(1)
		partialBlocks  = make(map[int64]*blockEvents)
		completeBlocks = make(map[int64]*blockEvents)
	)
	logging.Logger.Info("piers eventBlockController start")
	for {
		es := <-edb.blockEventChannel
		logging.Logger.Info("piers eventBlockController just caught block",
			zap.Int64("round", es.round))
		block, foundPartial := partialBlocks[es.round]
		//_, foundComplete := completeBlocks[es.round]
		//logging.Logger.Info("piers eventBlockController",
		//	zap.Int64("round", es.round),
		//	zap.Bool("found partial", foundPartial),
		//	zap.Bool("found complete", foundComplete),
		//	zap.Int("len partial", len(partialBlocks)),
		//	zap.Int("len complete", len(completeBlocks)),
		//)

		if es.round < currentRound {
			logging.Logger.Error("piers events for previous round",
				zap.Any("current round", currentRound),
				zap.Any("out of order events", es),
				zap.Int64("es round", es.round),
				zap.Int64("current round", currentRound),
			)
			continue
		}

		if !foundPartial {
			partialBlocks[es.round] = &es
			//logging.Logger.Info("piers eventBlockController",
			//	zap.Bool("foundPartial", foundPartial))
			continue
		}

		block.events = append(block.events, es.events...)
		delete(partialBlocks, es.round)

		if es.round > currentRound {
			completeBlocks[es.round] = block
			//logging.Logger.Info("piers eventBlockController",
			//	zap.Int64("es round", es.round),
			//	zap.Int64("current round", currentRound))
			continue
		}

		// We receive exactly two packets of events for each block
		//select {
		//case edb.blockEventChannel <- *block:
		//}
		//logging.Logger.Info("piers eventBlockController about to send",
		//	zap.Int64("round", block.round))
		edb.eventsChannel <- *block
		//logging.Logger.Info("piers eventBlockController just sent",
		//	zap.Int64("round", block.round))

		currentRound++
		//logging.Logger.Info("piers eventBlockController", zap.Int64("new round", currentRound))
		for b, found := completeBlocks[currentRound]; found; currentRound++ {
			logging.Logger.Info("piers eventBlockController in loop",
				zap.Int64("current round", currentRound),
				zap.Bool("found", found))
			edb.eventsChannel <- *b
			delete(completeBlocks, currentRound)
		}
		//logging.Logger.Info("piers eventBlockController end", zap.Int64("current round", currentRound))
	}
}
