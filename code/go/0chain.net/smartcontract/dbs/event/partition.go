package event

import (
	"fmt"
	"sort"
)

type TableInfo struct {
	TableName  string
	Tablespace string
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
	tablespace := edb.dbConfig.Slowtablespace

	var results []TableInfo
	err := edb.Store.Get().Table("pg_tables").
							Select("tablename, tablespace").
							Where("tablename LIKE ?", fmt.Sprintf("'%v_%%'", table)).
							Find(&results).Error
	if err != nil {
		return err
	}

	fmt.Println("results", results)
	// remove tables that are part of cold storage
	var hotTables []TableInfo
	for i, table := range results {
		if table.Tablespace != tablespace {
			hotTables = append(hotTables, results[i])
		}
	}

	fmt.Println("move to hot table only if there are more than 10 tables", hotTables)

	if len(hotTables) < 10 {
		// move to hot table only if there are more than 10 tables
		fmt.Println("move to hot table only if there are more than 10 tables")
		return nil
	}

	// sort by tablename
	sort.Slice(hotTables, func(i, j int) bool {
		return hotTables[i].TableName > hotTables[j].TableName
	})

	for _, table := range hotTables[10:] {
		fmt.Println("moving table to cold storage ", table)
		// identify the partition table that needs to be moved to slow partition
		raw := fmt.Sprintf("ALTER TABLE %v SET TABLESPACE %v", table.TableName, tablespace)
		return edb.Store.Get().Exec(raw).Error
	}
	return nil
}
