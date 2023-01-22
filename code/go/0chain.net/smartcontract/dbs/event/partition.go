package event

import "fmt"

func (edb *EventDb) addPartition(round int64, table string) {
	number := round / edb.settings.PartitionChangePeriod
	from := number * edb.settings.PartitionChangePeriod
	to := (number + 1) * edb.settings.PartitionChangePeriod

	raw := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %v_? PARTITION OF %v FOR VALUES FROM (?) TO (?)", table, table)
	edb.Store.Get().Exec(raw, number, from, to)

}

func (edb *EventDb) dropPartition(round int64, table string) {
	number := round / edb.settings.PartitionChangePeriod
	toDrop := number - edb.settings.PartitionKeepCount
	if toDrop < 0 {
		return
	}

	raw := fmt.Sprintf("DROP TABLE %v_?", table)
	edb.Store.Get().Exec(raw, toDrop)
}
