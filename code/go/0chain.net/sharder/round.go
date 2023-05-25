package sharder

import (
	"context"

	"0chain.net/chaincore/round"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/ememorystore"
)

// RoundSummaries -
type RoundSummaries struct {
	datastore.IDField
	RSummaryList []*round.Round `json:"round_summaries"`
}

// HealthyRound -
type HealthyRound struct {
	datastore.IDField
	Number int64
}

var roundSummariesEntityMetadata *datastore.EntityMetadataImpl
var healthyRoundEntityMetadata *datastore.EntityMetadataImpl

// NewRoundSummaries - create a new RoundSummaries entity
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

/*NewHealthyRound - create a new HealthyRound entity */
func NewHealthyRound() *HealthyRound {
	hr := datastore.GetEntityMetadata("healthy_round").Instance().(*HealthyRound)
	return hr
}

/*HealthyRoundProvider - a HealthyRoundProvider instance provider */
func HealthyRoundProvider() datastore.Entity {
	hr := &HealthyRound{}
	return hr
}

/*GetEntityMetadata - implement interface */
func (hr *HealthyRound) GetEntityMetadata() datastore.EntityMetadata {
	return healthyRoundEntityMetadata
}

/*SetupHealthyRound - setup the healthy round entity */
func (sc *Chain) SetupHealthyRound() {
	healthyRoundEntityMetadata = datastore.MetadataProvider()
	healthyRoundEntityMetadata.Name = "healthy_round"
	healthyRoundEntityMetadata.DB = sc.GetConfigInfoDB()
	healthyRoundEntityMetadata.Provider = HealthyRoundProvider
	healthyRoundEntityMetadata.Store = sc.GetConfigInfoStore()
	healthyRoundEntityMetadata.IDColumnName = "id"
	datastore.RegisterEntityMetadata("healthy_round", healthyRoundEntityMetadata)
}

// GetKey -
func (hr *HealthyRound) GetKey() datastore.Key {
	return datastore.ToKey(hr.GetEntityMetadata().GetName())
}

// SharderRoundFactory Factory for Sharder Round.
type SharderRoundFactory struct{}

// CreateRoundF the interface{} here returns generic round
func (mrf SharderRoundFactory) CreateRoundF(roundNum int64) round.RoundI {
	mr := round.NewRound(roundNum)
	return mr
}

/*StoreRound - persists given round to ememory(rocksdb)*/
func (sc *Chain) StoreRound(r *round.Round) error {
	roundEntityMetadata := r.GetEntityMetadata()
	rctx := ememorystore.WithEntityConnection(common.GetRootContext(), roundEntityMetadata)
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

func (sc *Chain) StoreRoundNoCommit(r *round.Round) (func() error, error) {
	roundEntityMetadata := r.GetEntityMetadata()
	rctx := ememorystore.WithEntityConnection(common.GetRootContext(), roundEntityMetadata)
	defer ememorystore.Close(rctx)
	err := r.Write(rctx)
	if err != nil {
		return nil, err
	}

	return func() error {
		return ememorystore.GetEntityCon(rctx, roundEntityMetadata).Commit()
	}, nil
}

// ReadHealthyRound -
func (sc *Chain) ReadHealthyRound(ctx context.Context) (*HealthyRound, error) {
	hr := datastore.GetEntity("healthy_round").(*HealthyRound)
	healthyRoundEntityMetadata := hr.GetEntityMetadata()
	hrStore := healthyRoundEntityMetadata.GetStore()
	hrctx := ememorystore.WithEntityConnection(ctx, healthyRoundEntityMetadata)
	defer ememorystore.Close(hrctx)
	err := hrStore.Read(hrctx, hr.GetKey(), hr)
	return hr, err
}

// WriteHealthyRound -
func (sc *Chain) WriteHealthyRound(ctx context.Context, hr *HealthyRound) error {
	healthyRoundEntityMetadata := hr.GetEntityMetadata()
	hrStore := healthyRoundEntityMetadata.GetStore()
	hrctx := ememorystore.WithEntityConnection(ctx, healthyRoundEntityMetadata)
	defer ememorystore.Close(hrctx)
	err := hrStore.Write(hrctx, hr)
	if err != nil {
		return err
	}
	con := ememorystore.GetEntityCon(hrctx, healthyRoundEntityMetadata)
	err = con.Commit()
	if err != nil {
		return err
	}
	return nil
}
