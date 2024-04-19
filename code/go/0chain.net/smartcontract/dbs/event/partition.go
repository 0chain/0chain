package event

import (
	"context"
	"fmt"
	"time"

	"github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

func (edb *EventDb) addPartition(current int64, table string) error {
	from := current * edb.settings.PartitionChangePeriod
	to := (current + 1) * edb.settings.PartitionChangePeriod

	timeout, cancelFunc := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancelFunc()
	raw := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %v_%v PARTITION OF %v FOR VALUES FROM (%v) TO (%v)", table, current, table, from, to)
	return edb.Store.Get().WithContext(timeout).Exec(raw).Error
}

func (edb *EventDb) dropPartition(current int64, table string) error {
	toDrop := current - edb.settings.PartitionKeepCount
	if toDrop < 0 {
		return nil
	}
	timeout, cancelFunc := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancelFunc()

	raw := fmt.Sprintf("DROP TABLE IF EXISTS %v_%v", table, toDrop)
	return edb.Store.Get().WithContext(timeout).Exec(raw).Error
}

func (edb *EventDb) addPermanentPartition(current int64, table string) error {
	from := current * edb.settings.PermanentPartitionChangePeriod
	to := (current + 1) * edb.settings.PermanentPartitionChangePeriod

	timeout, cancelFunc := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancelFunc()
	raw := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %v_%v PARTITION OF %v FOR VALUES FROM (%v) TO (%v)", table, current, table, from, to)
	logging.Logger.Debug("adding partition for",
		zap.String("table", table),
		zap.Int64("current", current),
		zap.Int64("from", from),
		zap.Int64("to", to))
	return edb.Store.Get().WithContext(timeout).Exec(raw).Error
}

func (edb *EventDb) movePartitionToSlowTableSpace(current int64, table string) error {
	toMove := current - edb.settings.PermanentPartitionKeepCount
	if toMove < 0 {
		return nil
	}

	timeout, cancelFunc := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancelFunc()

	tablespace := edb.dbConfig.Slowtablespace
	// identify the partition table that needs to be moved to slow partition
	raw := fmt.Sprintf("ALTER TABLE %v_%v SET TABLESPACE %v", table, toMove, tablespace)
	return edb.Store.Get().WithContext(timeout).Exec(raw).Error
}
