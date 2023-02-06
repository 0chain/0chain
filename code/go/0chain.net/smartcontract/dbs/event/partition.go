package event

import (
	"fmt"
)

func (edb *EventDb) addPartition(round int64, table string) error {
	number := round / edb.settings.PartitionChangePeriod
	from := number * edb.settings.PartitionChangePeriod
	to := (number + 1) * edb.settings.PartitionChangePeriod

	raw_create := fmt.Sprintf("CREATE TABLE IF NOT EXISTS public.%v_%v PARTITION OF public.%v FOR VALUES FROM (%v) TO (%v)", table, number, table, from, to)
	err := edb.Store.Get().Exec(raw_create).Error
	if err != nil {
		return err
	}
	raw_grant := fmt.Sprintf("ALTER TABLE public.%v_%v OWNER TO zchain_user;", table, number)
	return edb.Store.Get().Exec(raw_grant).Error
}

func (edb *EventDb) dropPartition(round int64, table string) error {
	number := round / edb.settings.PartitionChangePeriod
	toDrop := number - edb.settings.PartitionKeepCount
	if toDrop < 0 {
		return nil
	}

	raw := fmt.Sprintf("DROP TABLE public.%v_%v", table, toDrop)
	return edb.Store.Get().Exec(raw).Error
}
