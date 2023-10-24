package event

import (
	"fmt"
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

func (edb *EventDb) movePartitionToSlowTableSpace(tablespace, table string, round int64) error {
	number := round / edb.settings.PartitionChangePeriod
	toMove := number - edb.settings.PartitionKeepCount
	if toMove < 0 {
		return nil
	}

	// identify the partition table that needs to be moved to slow partition
	raw := fmt.Sprintf("ALTER TABLE %v_%v SET TABLESPACE %v;", table, toMove, tablespace)
	return edb.Store.Get().Exec(raw).Error
}
