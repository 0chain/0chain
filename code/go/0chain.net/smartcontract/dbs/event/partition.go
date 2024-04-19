package event

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"time"
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
	rawCurrentPartitionRangeQuery := `SELECT  partition_expression
		FROM (
			SELECT pt.relname AS partition_name,
				   pg_get_expr(pt.relpartbound, pt.oid, true) AS partition_expression
			FROM pg_class base_tb 
			JOIN pg_inherits i ON i.inhparent = base_tb.oid 
			JOIN pg_class pt ON pt.oid = i.inhrelid
			WHERE base_tb.oid = 'public.transactions'::regclass
		) AS subquery
		ORDER BY CAST(REGEXP_REPLACE(partition_expression, '\D', '', 'g') AS BIGINT) desc limit 1;`

	var currentPartitionRange, extractedNumberString string
	if err := edb.Store.Get().Raw(rawCurrentPartitionRangeQuery).Scan(&currentPartitionRange).Error; err != nil {
		return err
	}

	numberPattern := regexp.MustCompile(`FOR VALUES FROM \('(\d+)'\) TO \('\d+'\)`)

	matches := numberPattern.FindStringSubmatch(currentPartitionRange)

	if len(matches) > 1 {
		extractedNumberString = matches[1]
	}

	if extractedNumberString == "" {
		return fmt.Errorf("could not extract number from partition range")
	}
	currentPartitionEnd, err := strconv.ParseInt(extractedNumberString, 10, 64)
	if err != nil {
		return err
	}

	from := current * edb.settings.PermanentPartitionChangePeriod
	to := (current + 1) * edb.settings.PermanentPartitionChangePeriod

	if currentPartitionEnd < from {
		// Determine the number of partitions to create
		numPartitions := (from - currentPartitionEnd) / edb.settings.PermanentPartitionChangePeriod

		// Iterate to create each partition
		for i := int64(0); i < numPartitions; i++ {
			partitionStart := currentPartitionEnd + (i * edb.settings.PermanentPartitionChangePeriod)
			partitionEnd := partitionStart + edb.settings.PermanentPartitionChangePeriod

			// Generate SQL query for creating the partition
			raw := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %v_%v PARTITION OF %v FOR VALUES FROM (%v) TO (%v)", table, partitionStart, table, partitionStart, partitionEnd)

			// Execute the SQL query
			if err := edb.Store.Get().Exec(raw).Error; err != nil {
				return err
			}
		}

		// If the loop completes and newCurrentPartitionEnd is still less than from
		if newCurrentPartitionEnd := currentPartitionEnd + (numPartitions * edb.settings.PermanentPartitionChangePeriod); newCurrentPartitionEnd < from {
			// Create the partition for the remaining range
			raw := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %v_%v PARTITION OF %v FOR VALUES FROM (%v) TO (%v)", table, from, table, newCurrentPartitionEnd, from)
			return edb.Store.Get().Exec(raw).Error
		}
	}

	timeout, cancelFunc := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancelFunc()
	raw := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %v_%v PARTITION OF %v FOR VALUES FROM (%v) TO (%v)", table, current, table, from, to)
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
