package event

import (
	"fmt"
	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

func (edb *EventDb) addPartition(round int64, table string) error {
	number := round / edb.settings.PartitionChangePeriod
	from := number * edb.settings.PartitionChangePeriod
	to := (number + 1) * edb.settings.PartitionChangePeriod

	raw := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %v_%v PARTITION OF %v FOR VALUES FROM (%v) TO (%v)", table, number, table, from, to)
	return edb.Store.Get().Exec(raw).Error
}

func (edb *EventDb) dropPartition(round int64, table string) error {
	number := round / edb.settings.PartitionChangePeriod
	toDrop := number - edb.settings.PartitionKeepCount
	if toDrop < 0 {
		return nil
	}

	raw := fmt.Sprintf("DROP TABLE IF EXISTS %v_%v", table, toDrop)
	return edb.Store.Get().Exec(raw).Error
}

func (edb *EventDb) movePartitionToSlowTableSpace(round int64, table string) error {
	number := round / edb.settings.PartitionChangePeriod
	toMove := number - edb.settings.PartitionKeepCount

	logging.Logger.Info("Jayash movePartitionToSlowTableSpace", zap.Int64("round", round), zap.String("table", table), zap.Int64("toMove", toMove),
		zap.Any("settings", edb.settings.PartitionChangePeriod), zap.Any("settings", edb.settings.PartitionKeepCount))

	if toMove < 0 {
		return nil
	}

	tablespace := edb.dbConfig.Slowtablespace
	// identify the partition table that needs to be moved to slow partition
	raw := fmt.Sprintf("ALTER TABLE %v_%v SET TABLESPACE %v", table, toMove, tablespace)

	logging.Logger.Info("Jayash movePartitionToSlowTableSpace", zap.Int64("round", round), zap.String("table", table), zap.Int64("toMove", toMove),
		zap.Any("settings", edb.settings.PartitionChangePeriod), zap.Any("settings", edb.settings.PartitionKeepCount), zap.String("raw", raw))

	return edb.Store.Get().Exec(raw).Error
}
