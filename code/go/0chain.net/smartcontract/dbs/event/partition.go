package event

import (
	"fmt"
)

type TableInfo struct {
	Name string
	Size uint64
}

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
	var results []TableInfo
	err := edb.Store.Get().Table("pg_tables").Where("tablename LIKE ?", fmt.Sprintf("'%v_%%'", table)).Find(&results).Error
	if err != nil {
		return err
	}

	tablespace := edb.dbConfig.Slowtablespace
	for _, partionedTable := range results {
		if partionedTable.Size > edb.dbConfig.PartitionedTableMaxSize {
			// identify the partition table that needs to be moved to slow partition
			raw := fmt.Sprintf("ALTER TABLE %v SET TABLESPACE %v", partionedTable.Name, tablespace)
			return edb.Store.Get().Exec(raw).Error
		}
	}
	return nil
}
