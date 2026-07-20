# 0Chain Events Database Schema Documentation

## Overview

The `events_db` is a PostgreSQL 14 database used by 0chain sharders to store blockchain events, transactions, blocks, and provider data. This database is essential for:

1. **Chain Synchronization** - Storing finalized blocks and transactions
2. **Kafka Event Publishing** - Publishing blockchain events to Kafka for external consumers
3. **API Queries** - Serving data to 0box and other services via REST APIs
4. **Analytics** - Historical data for dashboards and reporting

---

## Entity Relationship Diagram

```
┌─────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                    0CHAIN EVENTS DATABASE SCHEMA                                            │
└─────────────────────────────────────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                         BLOCKCHAIN CORE                                                      │
├─────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                                              │
│    ┌──────────────────┐         ┌──────────────────┐         ┌──────────────────┐                           │
│    │     blocks       │         │   transactions   │         │     events       │                           │
│    │  (partitioned)   │         │   (partitioned)  │         │   (partitioned)  │                           │
│    ├──────────────────┤         ├──────────────────┤         ├──────────────────┤                           │
│    │ id               │         │ id               │         │ id               │                           │
│    │ hash ────────────┼────┐    │ hash             │         │ block_number ────┼──┐                        │
│    │ round            │    │    │ block_hash ──────┼────┐    │ tx_hash          │  │                        │
│    │ miner_id ────────┼─┐  │    │ round            │    │    │ type             │  │                        │
│    │ creation_date    │ │  │    │ client_id        │    │    │ tag              │  │                        │
│    │ merkle_tree_root │ │  │    │ to_client_id     │    │    │ index            │  │                        │
│    │ state_hash       │ │  │    │ value            │    │    │ is_published     │  │  ──► Kafka             │
│    │ num_txns         │ │  │    │ fee              │    │    │ sequence_number  │  │                        │
│    │ prev_hash        │ │  │    │ nonce            │    │    └──────────────────┘  │                        │
│    │ signature        │ │  └────┼──────────────────┼────┘                          │                        │
│    └──────────────────┘ │       │ transaction_type │                               │                        │
│                         │       │ status           │                               │                        │
│                         │       └──────────────────┘                               │                        │
│                         │                                                          │                        │
│    ┌──────────────────┐ │       ┌──────────────────┐                               │                        │
│    │    snapshots     │ │       │      users       │                               │                        │
│    │  (partitioned)   │ │       ├──────────────────┤                               │                        │
│    ├──────────────────┤ │       │ id               │                               │                        │
│    │ round (PK)       │ │       │ user_id          │◄──────────────────────────────┼────────────┐           │
│    │ total_mint       │ │       │ balance          │                               │            │           │
│    │ zcn_supply       │ │       │ nonce            │                               │            │           │
│    │ total_staked     │ │       │ mint_nonce       │                               │            │           │
│    │ total_rewards    │ │       └──────────────────┘                               │            │           │
│    │ transactions_cnt │ │                                                          │            │           │
│    │ block_count      │ │                                                          │            │           │
│    └──────────────────┘ │                                                          │            │           │
│                         │                                                          │            │           │
└─────────────────────────┼──────────────────────────────────────────────────────────┼────────────┼───────────┘
                          │                                                          │            │
┌─────────────────────────┼──────────────────────────────────────────────────────────┼────────────┼───────────┐
│                         │              NETWORK PROVIDERS                           │            │           │
├─────────────────────────┼──────────────────────────────────────────────────────────┼────────────┼───────────┤
│                         │                                                          │            │           │
│    ┌──────────────────┐ │       ┌──────────────────┐         ┌──────────────────┐  │            │           │
│    │     miners       │◄┘       │    sharders      │         │   authorizers    │  │            │           │
│    ├──────────────────┤         ├──────────────────┤         ├──────────────────┤  │            │           │
│    │ id (PK)          │         │ id (PK)          │         │ id (PK)          │  │            │           │
│    │ delegate_wallet  │         │ delegate_wallet  │         │ delegate_wallet  │  │            │           │
│    │ service_charge   │         │ service_charge   │         │ service_charge   │  │            │           │
│    │ total_stake ─────┼────┐    │ total_stake ─────┼────┐    │ total_stake ─────┼──┼───┐        │           │
│    │ n2n_host         │    │    │ n2n_host         │    │    │ url              │  │   │        │           │
│    │ host             │    │    │ host             │    │    │ total_mint       │  │   │        │           │
│    │ port             │    │    │ port             │    │    │ total_burn       │  │   │        │           │
│    │ fees             │    │    │ fees             │    │    │ creation_round   │  │   │        │           │
│    │ active           │    │    │ active           │    │    │ is_killed        │  │   │        │           │
│    │ creation_round   │    │    │ creation_round   │    │    │ is_shutdown      │  │   │        │           │
│    │ is_killed        │    │    │ is_killed        │    │    └──────────────────┘  │   │        │           │
│    │ is_shutdown      │    │    │ is_shutdown      │    │                          │   │        │           │
│    └──────────────────┘    │    └──────────────────┘    │                          │   │        │           │
│             │              │             │              │                          │   │        │           │
│             ▼              │             ▼              │                          │   │        │           │
│    ┌──────────────────┐    │    ┌──────────────────┐    │    ┌──────────────────┐  │   │        │           │
│    │ miner_aggregates │    │    │sharder_aggregates│    │    │authorizer_aggr.  │  │   │        │           │
│    │  (partitioned)   │    │    │  (partitioned)   │    │    │  (partitioned)   │  │   │        │           │
│    └──────────────────┘    │    └──────────────────┘    │    └──────────────────┘  │   │        │           │
│             │              │             │              │             │            │   │        │           │
│             ▼              │             ▼              │             ▼            │   │        │           │
│    ┌──────────────────┐    │    ┌──────────────────┐    │    ┌──────────────────┐  │   │        │           │
│    │ miner_snapshots  │    │    │sharder_snapshots │    │    │authorizer_snap.  │  │   │        │           │
│    └──────────────────┘    │    └──────────────────┘    │    └──────────────────┘  │   │        │           │
│                            │                           │                          │   │        │           │
└────────────────────────────┼───────────────────────────┼──────────────────────────┼───┼────────┼───────────┘
                             │                           │                          │   │        │
                             │                           │                          │   │        │
┌────────────────────────────┼───────────────────────────┼──────────────────────────┼───┼────────┼───────────┐
│                            │   STAKING & REWARDS       │                          │   │        │           │
├────────────────────────────┼───────────────────────────┼──────────────────────────┼───┼────────┼───────────┤
│                            │                           │                          │   │        │           │
│    ┌──────────────────┐    │                           │                          │   │        │           │
│    │  delegate_pools  │◄───┴───────────────────────────┴──────────────────────────┴───┘        │           │
│    ├──────────────────┤                                                                        │           │
│    │ id (PK)          │                                                                        │           │
│    │ pool_id          │                                                                        │           │
│    │ provider_type    │  (1=Miner, 2=Sharder, 3=Blobber, 4=Validator, 5=Authorizer)           │           │
│    │ provider_id ─────┼────► References miners/sharders/blobbers/validators/authorizers        │           │
│    │ delegate_id ─────┼────► References users                                                  │           │
│    │ balance          │                                                                        │           │
│    │ reward           │                                                                        │           │
│    │ total_reward     │                                                                        │           │
│    │ total_penalty    │                                                                        │           │
│    │ status           │  (0=Pending, 1=Active, 2=Deleting)                                     │           │
│    │ staked_at        │                                                                        │           │
│    └──────────────────┘                                                                        │           │
│             │                                                                                  │           │
│             ▼                                                                                  │           │
│    ┌──────────────────┐    ┌──────────────────┐    ┌──────────────────┐                        │           │
│    │ provider_rewards │    │  reward_mints    │    │ reward_providers │                        │           │
│    ├──────────────────┤    ├──────────────────┤    ├──────────────────┤                        │           │
│    │ provider_id (UK) │    │ client_id        │    │ provider_id      │                        │           │
│    │ rewards          │    │ pool_id          │    │ reward_type      │                        │           │
│    │ total_rewards    │    │ provider_type    │    │ allocation_id    │                        │           │
│    └──────────────────┘    │ amount           │    │ amount           │                        │           │
│                            └──────────────────┘    └──────────────────┘                        │           │
│                                                                                                │           │
│    ┌──────────────────┐    ┌──────────────────┐    ┌──────────────────┐                        │           │
│    │ reward_delegates │    │  user_aggregates │    │  user_snapshots  │◄───────────────────────┘           │
│    ├──────────────────┤    │   (partitioned)  │    ├──────────────────┤                                    │
│    │ pool_id          │    ├──────────────────┤    │ user_id (UK)     │                                    │
│    │ provider_id      │    │ user_id          │    │ collected_reward │                                    │
│    │ reward_type      │    │ round            │    │ total_stake      │                                    │
│    │ amount           │    │ collected_reward │    │ read_pool_total  │                                    │
│    └──────────────────┘    │ total_stake      │    │ write_pool_total │                                    │
│                            └──────────────────┘    └──────────────────┘                                    │
│                                                                                                             │
└─────────────────────────────────────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                         STORAGE SYSTEM                                                       │
├─────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                                              │
│    ┌──────────────────┐                           ┌──────────────────┐                                       │
│    │    blobbers      │                           │   validators     │                                       │
│    ├──────────────────┤                           ├──────────────────┤                                       │
│    │ id (PK)          │◄──────────────────────┐   │ id (PK)          │                                       │
│    │ base_url (UK)    │                       │   │ base_url         │                                       │
│    │ delegate_wallet  │                       │   │ delegate_wallet  │                                       │
│    │ service_charge   │                       │   │ service_charge   │                                       │
│    │ total_stake      │                       │   │ total_stake      │                                       │
│    │ capacity         │                       │   │ creation_round   │                                       │
│    │ allocated        │                       │   │ is_killed        │                                       │
│    │ used             │                       │   │ is_shutdown      │                                       │
│    │ read_price       │                       │   └──────────────────┘                                       │
│    │ write_price      │                       │            │                                                 │
│    │ challenges_passed│                       │            ▼                                                 │
│    │ open_challenges  │                       │   ┌──────────────────┐    ┌──────────────────┐               │
│    │ creation_round   │                       │   │validator_aggreg. │    │validator_snapshot│               │
│    │ is_killed        │                       │   │  (partitioned)   │    └──────────────────┘               │
│    │ is_shutdown      │                       │   └──────────────────┘                                       │
│    └──────────────────┘                       │                                                              │
│             │                                 │                                                              │
│             │                                 │                                                              │
│    ┌────────┴────────┐                        │                                                              │
│    ▼                 ▼                        │                                                              │
│  ┌──────────────────┐ ┌──────────────────┐    │                                                              │
│  │blobber_aggregates│ │blobber_snapshots │    │                                                              │
│  │  (partitioned)   │ └──────────────────┘    │                                                              │
│  └──────────────────┘                         │                                                              │
│                                               │                                                              │
│    ┌──────────────────┐                       │                                                              │
│    │   allocations    │                       │                                                              │
│    ├──────────────────┤                       │                                                              │
│    │ id (PK)          │◄──────────────────────┼───────────────────────────────────────────┐                  │
│    │ allocation_id(UK)│◄──────────────────────┼────────────────────────────┐              │                  │
│    │ owner ───────────┼────► users            │                            │              │                  │
│    │ data_shards      │                       │                            │              │                  │
│    │ parity_shards    │                       │                            │              │                  │
│    │ size             │                       │                            │              │                  │
│    │ expiration       │                       │                            │              │                  │
│    │ write_pool       │                       │                            │              │                  │
│    │ used_size        │                       │                            │              │                  │
│    │ finalized        │                       │                            │              │                  │
│    │ cancelled        │                       │                            │              │                  │
│    └──────────────────┘                       │                            │              │                  │
│             │                                 │                            │              │                  │
│             ▼                                 │                            │              │                  │
│    ┌──────────────────┐                       │                            │              │                  │
│    │alloc_blobber_term│                       │                            │              │                  │
│    ├──────────────────┤                       │                            │              │                  │
│    │ alloc_id (FK)────┼───► allocations(id)   │                            │              │                  │
│    │ blobber_id (FK)──┼───────────────────────┘                            │              │                  │
│    │ read_price       │                                                    │              │                  │
│    │ write_price      │                                                    │              │                  │
│    └──────────────────┘                                                    │              │                  │
│                                                                            │              │                  │
│    ┌──────────────────┐    ┌──────────────────┐    ┌──────────────────┐    │              │                  │
│    │  write_markers   │    │  read_markers    │    │ challenge_pools  │    │              │                  │
│    ├──────────────────┤    ├──────────────────┤    ├──────────────────┤    │              │                  │
│    │ allocation_id(FK)┼────┼──────────────────┼────┼──────────────────┼────┘              │                  │
│    │ blobber_id (FK)  │    │ blobber_id (FK)  │    │ allocation_id(UK)│                   │                  │
│    │ transaction_id   │    │ transaction_id   │    │ balance          │                   │                  │
│    │ allocation_root  │    │ read_counter     │    │ expiration       │                   │                  │
│    │ size             │    │ read_size        │    │ finalized        │                   │                  │
│    │ block_number     │    │ block_number     │    └──────────────────┘                   │                  │
│    └──────────────────┘    └──────────────────┘                                           │                  │
│                                                                                           │                  │
│    ┌──────────────────┐                                                                   │                  │
│    │   challenges     │                                                                   │                  │
│    ├──────────────────┤                                                                   │                  │
│    │ challenge_id(UK) │                                                                   │                  │
│    │ allocation_id ───┼───────────────────────────────────────────────────────────────────┘                  │
│    │ blobber_id       │                                                                                      │
│    │ validators_id    │                                                                                      │
│    │ passed           │                                                                                      │
│    │ responded        │                                                                                      │
│    │ round_created_at │                                                                                      │
│    └──────────────────┘                                                                                      │
│                                                                                                              │
│    ┌──────────────────┐                                                                                      │
│    │   read_pools     │                                                                                      │
│    ├──────────────────┤                                                                                      │
│    │ user_id (PK)     │────► users                                                                           │
│    │ balance          │                                                                                      │
│    └──────────────────┘                                                                                      │
│                                                                                                              │
└─────────────────────────────────────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                         BRIDGE & ERRORS                                                      │
├─────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                                              │
│    ┌──────────────────┐    ┌──────────────────┐    ┌──────────────────┐                                      │
│    │  burn_tickets    │    │     errors       │    │transaction_errors│                                      │
│    ├──────────────────┤    ├──────────────────┤    ├──────────────────┤                                      │
│    │ ethereum_address │    │ transaction_id   │    │ transaction_out  │                                      │
│    │ hash             │    │ error            │    │ count            │                                      │
│    │ nonce            │    └──────────────────┘    └──────────────────┘                                      │
│    │ amount           │                                                                                      │
│    └──────────────────┘                                                                                      │
│                                                                                                              │
└─────────────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## Table Categories

### 1. Blockchain Core Tables

| Table | Partitioned | Purpose |
|-------|-------------|---------|
| `blocks` | Yes (round) | Stores finalized block headers |
| `transactions` | Yes (round) | Stores all transactions |
| `events` | Yes (block_number) | Events for Kafka publishing |
| `snapshots` | Yes (round) | Global chain state at each round |
| `users` | No | User accounts and balances |

### 2. Network Provider Tables

| Provider | Main Table | Aggregates | Snapshots |
|----------|------------|------------|-----------|
| Miners | `miners` | `miner_aggregates` | `miner_snapshots` |
| Sharders | `sharders` | `sharder_aggregates` | `sharder_snapshots` |
| Blobbers | `blobbers` | `blobber_aggregates` | `blobber_snapshots` |
| Validators | `validators` | `validator_aggregates` | `validator_snapshots` |
| Authorizers | `authorizers` | `authorizer_aggregates` | `authorizer_snapshots` |

### 3. Staking & Rewards Tables

| Table | Purpose |
|-------|---------|
| `delegate_pools` | Staking pools for all provider types |
| `provider_rewards` | Accumulated provider rewards |
| `reward_mints` | Token minting events |
| `reward_providers` | Provider reward distributions |
| `reward_delegates` | Delegate reward distributions |
| `user_aggregates` | User staking history |
| `user_snapshots` | Current user staking state |

### 4. Storage System Tables

| Table | Purpose |
|-------|---------|
| `allocations` | Storage allocation metadata |
| `allocation_blobber_terms` | Blobber pricing per allocation |
| `challenges` | Storage challenges |
| `challenge_pools` | Challenge pool balances |
| `read_markers` | Read operation records |
| `write_markers` | Write operation records |
| `read_pools` | User read pool balances |

### 5. Bridge & Error Tables

| Table | Purpose |
|-------|---------|
| `burn_tickets` | ZCN burn events for Ethereum bridge |
| `errors` | Transaction errors |
| `transaction_errors` | Aggregated error counts |

---

## Partitioning Strategy

The database uses **range partitioning** for high-volume tables to:
- Improve query performance
- Enable efficient data pruning
- Reduce index sizes

### Partition Configuration

```yaml
# From 0chain.yaml
dbs:
  events:
    partition_change_period: 100000    # New partition every 100k rounds
    partition_keep_count: 20           # Keep last 20 partitions
```

### Partition Lifecycle

1. **Creation**: New partitions created automatically when round reaches threshold
2. **Retention**: Old partitions kept based on `partition_keep_count`
3. **Archival**: Old partitions can be moved to slow tablespace (`hdd_tablespace`)
4. **Deletion**: Partitions beyond retention period are dropped

---

## What's Needed for Blockchain to Operate Well

### Critical Components

#### 1. **PostgreSQL Database**
```
✅ REQUIRED for sharder operation
- Stores all blockchain events
- Provides data for APIs
- Enables Kafka event publishing
```

#### 2. **Core Tables Required for Chain Sync**

| Table | Why Critical |
|-------|--------------|
| `blocks` | Block finalization tracking |
| `transactions` | Transaction history |
| `events` | Event publishing to Kafka |
| `snapshots` | Global state tracking |

#### 3. **Provider Tables Required**

| Tables | Why Critical |
|--------|--------------|
| `miners` | Active miner tracking |
| `sharders` | Active sharder tracking |
| `delegate_pools` | Staking state |

### Minimum Database Requirements

```sql
-- Essential tables that MUST have data for chain to operate:

1. blocks           -- At least genesis block
2. transactions     -- Genesis transactions
3. miners           -- Active miners in magic block
4. sharders         -- Active sharders in magic block
5. delegate_pools   -- Staking pools (if staking enabled)
6. events           -- For Kafka (if enabled)
```

### Database Health Checklist

```sql
-- Run these queries to verify database health:

-- 1. Check latest block
SELECT MAX(round) as latest_round FROM blocks;

-- 2. Check active miners
SELECT COUNT(*) FROM miners WHERE active = true AND is_killed = false;

-- 3. Check active sharders  
SELECT COUNT(*) FROM sharders WHERE active = true AND is_killed = false;

-- 4. Check unpublished events (Kafka backlog)
SELECT COUNT(*) FROM events WHERE is_published = false;

-- 5. Check partition health
SELECT schemaname, tablename 
FROM pg_tables 
WHERE tablename LIKE '%_part_%' OR tablename LIKE '%_0' OR tablename LIKE '%_1';
```

### Performance Tuning

#### PostgreSQL Configuration (postgresql.conf)

```ini
# Memory
shared_buffers = 256MB          # 25% of RAM for dedicated DB server
work_mem = 64MB                 # For complex queries
maintenance_work_mem = 128MB    # For VACUUM, CREATE INDEX

# WAL
max_wal_size = 2GB
min_wal_size = 256MB
wal_buffers = 64MB

# Connections
max_connections = 200

# Query Planning
effective_cache_size = 1GB      # 50-75% of RAM
random_page_cost = 1.1          # For SSD storage
```

#### Index Optimization

The schema includes these critical indexes:

```sql
-- Block lookups
CREATE UNIQUE INDEX idx_bhash ON blocks(hash, round);
CREATE INDEX idx_bround ON blocks(round);

-- Transaction lookups
CREATE UNIQUE INDEX idx_thash ON transactions(hash, round);
CREATE INDEX idx_tclient_id ON transactions(client_id);
CREATE INDEX idx_tto_client_id ON transactions(to_client_id, client_id);

-- Provider lookups
CREATE INDEX idx_miner_creation_round ON miners(creation_round);
CREATE INDEX idx_sharder_creation_round ON sharders(creation_round);

-- Staking lookups
CREATE INDEX idx_dprov_active ON delegate_pools(provider_id, provider_type, status);
CREATE INDEX idx_dp_total_staked ON delegate_pools(delegate_id, status);
```

---

## Kafka Integration

### Event Publishing Flow

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Sharder   │────►│  events_db  │────►│   Kafka     │────►│  Consumers  │
│  (Go code)  │     │  (events)   │     │  Producer   │     │  (0box etc) │
└─────────────┘     └─────────────┘     └─────────────┘     └─────────────┘
```

### Event Table Fields for Kafka

```sql
-- Events table with Kafka-specific fields
CREATE TABLE events (
    id bigint,
    block_number bigint NOT NULL,
    tx_hash text,
    type bigint,                    -- Event type enum
    tag bigint,                     -- Event tag enum  
    index text,
    is_published boolean,           -- ✅ Kafka publish status
    sequence_number bigint,         -- ✅ Ordering guarantee
    PRIMARY KEY (id, block_number)
);
```

### Kafka Configuration

```yaml
# From 0chain.yaml
dbs:
  events:
    kafka_enabled: true
    kafka_host: "kafka:9092"
    kafka_trigger_round: 0          # Start publishing from round 0
    kafka_write_timeout: 30s
```

---

## Backup & Recovery

### Critical Files to Backup

```
postgresql/
├── global/pg_control              # MOST CRITICAL
├── base/16569/                    # events_db data
├── pg_wal/                        # WAL for recovery
└── pg_xact/                       # Transaction status
```

### Backup Command

```bash
# Full backup
pg_basebackup -h localhost -U postgres -D /backup/pg_backup -Ft -z -P

# Logical backup (for migration)
pg_dump -h localhost -U zchain_user events_db > events_db_backup.sql
```

### Recovery Steps

1. Stop PostgreSQL
2. Restore data directory from backup
3. Ensure `pg_control` is intact
4. Start PostgreSQL
5. Verify with health queries above

---

## Summary

The `events_db` PostgreSQL database is **essential** for 0chain sharder operation. It stores:

- **Blockchain data**: blocks, transactions, events
- **Provider data**: miners, sharders, blobbers, validators, authorizers
- **Staking data**: delegate pools, rewards
- **Storage data**: allocations, challenges, read/write markers

For the blockchain to operate well, ensure:

1. ✅ PostgreSQL is running and healthy
2. ✅ All core tables exist with proper schema
3. ✅ Partitions are being created/pruned correctly
4. ✅ Indexes are in place for query performance
5. ✅ Kafka events are being published (if enabled)
6. ✅ Regular backups are configured

