package sharder

import (
	"context"

	"0chain.net/chaincore/round"
	"0chain.net/core/ememorystore"
)

// SharderRoundFactory Factory for Sharder Round
type SharderRoundFactory struct{}

// CreateRoundF the interface{} here returns generic round
func (mrf SharderRoundFactory) CreateRoundF(roundNum int64) interface{} {
	mr := round.NewRound(roundNum)
	return mr
}

/*StoreRound - persists given round to ememory(rocksdb)*/
func (sc *Chain) StoreRound(ctx context.Context, r *round.Round) error {
	roundEntityMetadata := r.GetEntityMetadata()
	rctx := ememorystore.WithEntityConnection(ctx, roundEntityMetadata)
	defer ememorystore.Close(rctx)
	err := r.Write(rctx)
	if err != nil {
		return err
	}
	con := ememorystore.GetEntityCon(rctx, roundEntityMetadata)
	err = con.Commit()
	if err != nil {
		return err
	}
	return nil
}
