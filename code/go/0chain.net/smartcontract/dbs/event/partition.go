package event

import (
	"context"
	"fmt"
	"log"
	"time"
)

func (edb *EventDb) addPartition(current int64, table string) error {
	from := current * edb.settings.PartitionChangePeriod
	to := (current + 1) * edb.settings.PartitionChangePeriod

	log.Printf("addPartition (current, from, to), change_period = (%v, %v, %v) %v\n", current, from, to, edb.settings.PartitionChangePeriod)

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

func (edb *EventDb) movePartitionToSlowTableSpace(current int64, table string) error {
	toMove := current - edb.settings.PartitionKeepCount
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
