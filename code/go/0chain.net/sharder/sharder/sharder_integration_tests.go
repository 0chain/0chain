//go:build integration_tests
// +build integration_tests

package main

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/node"
	crpc "0chain.net/conductor/conductrpc" // integration tests
	"0chain.net/conductor/conductrpc/stats"
	"0chain.net/smartcontract/dbs/event"
	"github.com/0chain/common/core/logging"
)

// start lock, where the sharder is ready to connect to blockchain (BC)
func initIntegrationsTests() {
	crpc.Init()
}

func registerInConductor(id string) {
	crpc.Client().Register(id)
}

func shutdownIntegrationTests() {
	crpc.Shutdown()
}

func readMagicBlock(magicBlockConfig string) (*block.MagicBlock, error) {
	magicBlockFromConductor := crpc.Client().MagicBlock()

	if magicBlockFromConductor != "" {
		return chain.ReadMagicBlockFile(magicBlockFromConductor)
	}

	return chain.ReadMagicBlockFile(magicBlockConfig)
}

func notifyConductor(block *block.Block) error {
	logging.Logger.Debug("[conductor] notifyConductor",
		zap.String("sharder", node.Self.ID),
		zap.String("miner", block.MinerID),
		zap.Int64("round", block.Round),
		zap.String("hash", block.Hash),
	)
	if crpc.Client().State().NotifyOnBlockGeneration {
		return crpc.Client().NotifyOnSharderBlock(&stats.BlockFromSharder{
			Round: block.Round,
			Hash: block.Hash,
			GeneratorId: block.MinerID,
			SenderId: node.Self.ID,
		})
	}
	return nil
}

func notifyOnAggregates(ctx context.Context, edb *event.EventDb, round int64) error {
	monitorAggregates := crpc.Client().State().MonitorAggregates
	if monitorAggregates == nil {
		return nil
	}
	
	tx, err := edb.Begin(ctx)
	if err != nil {
		return err
	}

	if len(monitorAggregates.MinerIds) > 0 && len(monitorAggregates.MinerFields) > 0 {
		logging.Logger.Debug("[conductor] processing miner aggregates", zap.Any("ids", monitorAggregates.MinerIds), zap.Any("fields", monitorAggregates.MinerFields))
		res, err := getProviderAggregatesForRound(tx, "miner", round, monitorAggregates.MinerIds, monitorAggregates.MinerFields)
		if err != nil {
			return err
		}

		logging.Logger.Debug("[conductor] found miner aggregates", zap.Any("round", round), zap.Any("aggregates", res))
	
		for _, agg := range res {
			pid , ok := agg["miner_id"].(string)
			logging.Logger.Debug("[conductor] sending miner aggregate to c.s.", zap.Any("agg", agg))
			if !ok {
				logging.Logger.Warn("[conductor] aggregate without id or id not string", zap.Any("agg", agg))
			}

			delete(agg, "miner_id")
			crpc.Client().SendAggregate(&crpc.AggregateMessage{
				ProviderType: stats.Miner,
				ProviderId: pid,
				Values: agg,
			})
		}
	}

	if len(monitorAggregates.SharderIds) > 0 && len(monitorAggregates.SharderFields) > 0 {
		logging.Logger.Debug("[conductor] processing sharder aggregates", zap.Any("ids", monitorAggregates.SharderIds), zap.Any("fields", monitorAggregates.SharderFields))
		res, err := getProviderAggregatesForRound(tx, "sharder", round, monitorAggregates.SharderIds, monitorAggregates.SharderFields)
		if err != nil {
			return err
		}

		logging.Logger.Debug("[conductor] found sharder aggregates", zap.Any("round", round), zap.Any("aggregates", res))
	
		for _, agg := range res {
			logging.Logger.Debug("[conductor] sending sharder aggregate to c.s.", zap.Any("agg", agg))
			pid , ok := agg["sharder_id"].(string)
			if !ok {
				logging.Logger.Warn("[conductor] aggregate without id or id not string", zap.Any("agg", agg))
			}

			delete(agg, "sharder_id")
			crpc.Client().SendAggregate(&crpc.AggregateMessage{
				ProviderType: stats.Sharder,
				ProviderId: pid,
				Values: agg,
			})
		}
	}

	if len(monitorAggregates.BlobberIds) > 0 && len(monitorAggregates.BlobberFields) > 0 {
		logging.Logger.Debug("[conductor] processing blobber aggregates", zap.Any("ids", monitorAggregates.BlobberIds), zap.Any("fields", monitorAggregates.BlobberFields))
		res, err := getProviderAggregatesForRound(tx, "blobber", round, monitorAggregates.BlobberIds, monitorAggregates.BlobberFields)
		if err != nil {
			return err
		}

		logging.Logger.Debug("[conductor] found blobber aggregates", zap.Any("round", round), zap.Any("aggregates", res))
	
		for _, agg := range res {
			logging.Logger.Debug("[conductor] sending blobber aggregate to c.s.", zap.Any("agg", agg))
			pid , ok := agg["blobber_id"].(string)
			if !ok {
				logging.Logger.Warn("[conductor] aggregate without id or id not string", zap.Any("agg", agg))
			}

			delete(agg, "blobber_id")
			crpc.Client().SendAggregate(&crpc.AggregateMessage{
				ProviderType: stats.Blobber,
				ProviderId: pid,
				Values: agg,
			})
		}
	}

	if len(monitorAggregates.ValidatorIds) > 0 && len(monitorAggregates.ValidatorIds) > 0 {
		logging.Logger.Debug("[conductor] processing validator aggregates", zap.Any("ids", monitorAggregates.ValidatorIds), zap.Any("fields", monitorAggregates.ValidatorFields))
		res, err := getProviderAggregatesForRound(tx, "validator", round, monitorAggregates.ValidatorIds, monitorAggregates.ValidatorFields)
		if err != nil {
			return err
		}

		logging.Logger.Debug("[conductor] found validator aggregates", zap.Any("round", round), zap.Any("aggregates", res))
	
		for _, agg := range res {
			logging.Logger.Debug("[conductor] sending validator aggregate to c.s.", zap.Any("agg", agg))
			pid , ok := agg["validator_id"].(string)
			if !ok {
				logging.Logger.Warn("[conductor] aggregate without id or id not string", zap.Any("agg", agg))
			}

			delete(agg, "validator_id")
			crpc.Client().SendAggregate(&crpc.AggregateMessage{
				ProviderType: stats.Validator,
				ProviderId: pid,
				Values: agg,
			})
		}
	}

	if len(monitorAggregates.AuthorizerIds) > 0 && len(monitorAggregates.AuthorizerFields) > 0 {
		logging.Logger.Debug("[conductor] processing authorizer aggregates", zap.Any("ids", monitorAggregates.AuthorizerIds), zap.Any("fields", monitorAggregates.AuthorizerFields))
		res, err := getProviderAggregatesForRound(tx, "authorizer", round, monitorAggregates.AuthorizerIds, monitorAggregates.AuthorizerFields)
		if err != nil {
			return err
		}

		logging.Logger.Debug("[conductor] found authorizer aggregates", zap.Any("round", round), zap.Any("aggregates", res))
	
		for _, agg := range res {
			logging.Logger.Debug("[conductor] sending authorizer aggregate to c.s.", zap.Any("agg", agg))
			pid , ok := agg["authorizer_id"].(string)
			if !ok {
				logging.Logger.Warn("[conductor] aggregate without id or id not string", zap.Any("agg", agg))
			}

			delete(agg, "authorizer_id")
			crpc.Client().SendAggregate(&crpc.AggregateMessage{
				ProviderType: stats.Authorizer,
				ProviderId: pid,
				Values: agg,
			})
		}
	}

	if len(monitorAggregates.UserIds) > 0 && len(monitorAggregates.UserFields) > 0 {
		logging.Logger.Debug("[conductor] processing user aggregates", zap.Any("ids", monitorAggregates.UserIds), zap.Any("fields", monitorAggregates.UserFields))
		res, err := getProviderAggregatesForRound(tx, "user", round, monitorAggregates.UserIds, monitorAggregates.UserFields)
		if err != nil {
			return err
		}

		logging.Logger.Debug("[conductor] found user aggregates", zap.Any("round", round), zap.Any("aggregates", res))

		for _, agg := range res {
			logging.Logger.Debug("[conductor] sending user aggregate to c.s.", zap.Any("agg", agg))
			pid , ok := agg["user_id"].(string)
			if !ok {
				logging.Logger.Warn("[conductor] aggregate without id or id not string", zap.Any("agg", agg))
			}

			delete(agg, "user_id")
			crpc.Client().SendAggregate(&crpc.AggregateMessage{
				ProviderType: stats.User,
				ProviderId: pid,
				Values: agg,
			})
		}
	}

	if monitorAggregates.MonitorGlobal && len(monitorAggregates.GlobalFields) > 0 {
		logging.Logger.Debug("[conductor] processing global aggregates", zap.Any("fields", monitorAggregates.GlobalFields))
		res, err := getGloablSnapshot(tx, round, monitorAggregates.GlobalFields)
		if err != nil {
			return err
		}

		logging.Logger.Debug("[conductor] found global aggregates", zap.Any("round", round), zap.Any("aggregates", res))

		if res != nil {
			logging.Logger.Debug("[conductor] sending global aggregate to c.s.", zap.Any("agg", res))
			crpc.Client().SendAggregate(&crpc.AggregateMessage{
				ProviderType: stats.Global,
				ProviderId: "global",
				Values: res,
			})
		}
	}

	return tx.Rollback()
}

func getProviderAggregatesForRound(tx *event.EventDb, provider string, round int64, ids []string, keys []string) ([]stats.Aggregate, error) {
	tableName := fmt.Sprintf("%v_aggregates", provider)
	idString := fmt.Sprintf("%v_id", provider)

	var results []stats.Aggregate
	q := tx.Get().Table(tableName).Select(append(keys, idString)).Where("round", round).Where(fmt.Sprintf("%v in ?", idString), ids).Scan(&results)
	logging.Logger.Debug("[conductor] getProviderAggregatesForRound query", zap.Any("query", tx.Get().ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Table(tableName).Select(append(keys, idString)).Where("round", round).Where(fmt.Sprintf("%v in ?", idString), ids).Scan(&results)
	})))

	err := q.Error
	if err != nil {
		return nil, err
	}

	return results, nil
}

func getGloablSnapshot(tx *event.EventDb, round int64, keys []string) (stats.Aggregate, error) {
	var result stats.Aggregate
	err := tx.Get().Table("snapshots").Select(keys).Where("round", round).Scan(&result).Error
	if err != nil {
		return nil, err
	}

	return result, nil
}