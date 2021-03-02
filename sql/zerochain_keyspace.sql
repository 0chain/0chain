-- noinspection SqlDialectInspectionForFile

CREATE  KEYSPACE IF NOT EXISTS zerochain
WITH REPLICATION = { 'class' : 'SimpleStrategy', 'replication_factor' : 1 }
AND DURABLE_WRITES = true;
