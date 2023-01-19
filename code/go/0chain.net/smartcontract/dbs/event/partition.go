package event

import "fmt"

func (edb *EventDb) addPartition(round int64, table string) {
	number := round / edb.settings.PartitionChanePeriod
	from := number * edb.settings.PartitionChanePeriod
	to := (number + 1) * edb.settings.PartitionChanePeriod

	raw := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %v_? PARTITION OF %v FOR VALUES FROM (?) TO (?)", table, table)
	edb.Store.Get().Exec(raw, number, from, to)

}

func (edb *EventDb) dropPartition(round int64, table string) {
	number := round / edb.settings.PartitionChanePeriod
	toDrop := number - edb.settings.PartitionKeepCount
	if toDrop < 0 {
		return
	}

	raw := fmt.Sprintf("DROP TABLE %v_?", table)
	edb.Store.Get().Exec(raw, toDrop)
}
