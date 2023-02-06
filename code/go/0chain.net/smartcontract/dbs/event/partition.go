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

	raw := fmt.Sprintf("ALTER TABLE %v DROP PARTITION %v_%v", table, table, toDrop)
	return edb.Store.Get().Exec(raw).Error
}
