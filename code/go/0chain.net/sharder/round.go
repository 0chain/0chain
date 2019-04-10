package sharder

import (
	"context"

	"0chain.net/chaincore/round"
	"0chain.net/core/datastore"
	"0chain.net/core/ememorystore"
)

type RoundSummaries struct {
	datastore.IDField
	RSummaryList []*round.Round `json:round_summaries`
}

var roundSummariesEntityMetadata *datastore.EntityMetadataImpl

/*NewRoundSummaries - create a new RoundSummaries entity */
func NewRoundSummaries() *RoundSummaries {
	rs := datastore.GetEntityMetadata("round_summaries").Instance().(*RoundSummaries)
	return rs
}

/*RoundSummariesProvider - a round summaries instance provider */
func RoundSummariesProvider() datastore.Entity {
	rs := &RoundSummaries{}
	return rs
}

/*GetEntityMetadata - implement interface */
func (rs *RoundSummaries) GetEntityMetadata() datastore.EntityMetadata {
	return roundSummariesEntityMetadata
}

/*SetupRoundSummaries - setup the round summaries entity */
func SetupRoundSummaries() {
	roundSummariesEntityMetadata = datastore.MetadataProvider()
	roundSummariesEntityMetadata.Name = "round_summaries"
	roundSummariesEntityMetadata.Provider = RoundSummariesProvider
	roundSummariesEntityMetadata.IDColumnName = "id"
	datastore.RegisterEntityMetadata("round_summaries", roundSummariesEntityMetadata)
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

/*GetMostRecentRoundFromDB - gives the most recent round added to db*/
func (sc *Chain) GetMostRecentRoundFromDB(ctx context.Context) (*round.Round, error) {
	remd := datastore.GetEntityMetadata("round")
	rctx := ememorystore.WithEntityConnection(ctx, remd)
	defer ememorystore.Close(rctx)
	c := ememorystore.GetEntityCon(rctx, remd)
	r := remd.Instance().(*round.Round)
	iterator := c.Conn.NewIterator(c.ReadOptions)
	defer iterator.Close()
	iterator.SeekToLast()
	if iterator.Valid() {
		datastore.FromJSON(iterator.Value().Data(), r)
	}
	return r, iterator.Err()
}
