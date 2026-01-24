# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

0chain (Züs) is a high-performance blockchain-based distributed cloud storage network written in Go. It provides configurable redundancy, privacy, and uptime using erasure coding and a distributed architecture across two node types:

- **Miners**: Create and validate blocks, run consensus protocol
- **Sharders**: Store blocks, maintain blockchain history, provide REST APIs

## Build Commands

### Prerequisites
Build the base Docker image first (required for all other builds):
```bash
./docker.local/bin/build.base.sh
```

### Generate Mocks (Required Before Testing)
```bash
make build-mocks
```

### Build Nodes
```bash
./docker.local/bin/build.miners.sh     # Build miner containers
./docker.local/bin/build.sharders.sh   # Build sharder containers
```

### Running Tests

**Unit Tests:**
```bash
./docker.local/bin/unit_test.sh                    # Run all unit tests
./docker.local/bin/unit_test.sh [packages]         # Run specific packages
./docker.local/bin/unit_test.sh --no-mocks         # Skip mock generation
make run-test                                       # Run tests with coverage (requires bn256 tag)
```

**Single Test (local Go):**
```bash
cd code/go/0chain.net
go test -tags bn256 -v ./path/to/package -run TestName
```

**Integration Tests:**
```bash
./docker.local/bin/build.miners-integration-tests.sh
./docker.local/bin/build.sharders-integration-tests.sh
./docker.local/bin/start.conductor.sh miners              # Standard miner tests
./docker.local/bin/start.conductor.sh sharders            # Standard sharder tests
./docker.local/bin/start.conductor.sh view-change.byzantine
./docker.local/bin/start.conductor.sh no-view-change.fault-tolerance
```

### Other Useful Commands
```bash
make go-mod                            # Tidy and download dependencies
make install-msgp && make msgp         # Generate msgp serialization code
make swagger                           # Generate all API documentation
./docker.local/bin/sync_clock.sh       # Sync container clocks (fixes validation errors)
./docker.local/bin/clean.sh            # Clean up blockchain data
```

## Code Architecture

### Directory Structure
```
code/go/0chain.net/
├── chaincore/           # Core blockchain: blocks, chain, rounds, transactions, state (MPT)
├── miner/               # Miner node implementation
│   ├── protocol_*.go    # Protocol handlers (block, round, receive)
│   └── mocks/           # Auto-generated mocks (not in git)
├── sharder/             # Sharder node implementation
│   ├── protocol_*.go    # Protocol handlers
│   ├── blockdb/         # Block database
│   └── blockstore/      # Block storage
├── smartcontract/       # Smart contracts
│   ├── minersc/         # Miner staking, rewards, fees
│   ├── storagesc/       # Storage providers, allocations
│   ├── zcnsc/           # ZCN token contract
│   ├── stakepool/       # Delegation and staking
│   └── dbs/             # Database operations
├── core/                # Utilities: cache, encryption, config, datastore
└── conductor/           # Integration test orchestrator (RPC server)
```

### Key Patterns

**File Naming:**
- `protocol_*.go` - Protocol implementation
- `*_main.go` - Production entry points
- `*_integration_tests.go` - Integration test handlers (built with `integration_tests` tag)
- `*_gen.go` - Auto-generated msgp serialization
- `mocks/` - Auto-generated mock interfaces (not committed to git)

**Build Tags:**
- `bn256` - Production cryptography (BLS curve)
- `integration_tests` - Enable integration test code
- `development` - Enable dev features like n2n delays

**Local Compilation:**
```bash
cd code/go/0chain.net/miner
go build -tags "bn256 development"
```

## Testing Guidelines

- Use table-driven tests with subtests
- Use `require` (not `assert`) for immediate failure
- Run tests in parallel with `t.Parallel()` where appropriate
- Mock external dependencies via interfaces
- Mocks are auto-generated - run `make build-mocks` first
- Keep databases in-memory, avoid filesystem
- Don't send real HTTP requests - use `httptest` or mock interfaces

## Dependencies

Key external libraries:
- `herumi/bls` - BLS threshold signatures for consensus
- `linxGnu/grocksdb` - RocksDB key-value store
- `stretchr/testify` - Testing assertions and mocks
- `vektra/mockery` - Mock code generation
- `tinylib/msgp` - MessagePack serialization (uses 0chain fork)

## Contributing

1. Branch from `staging` (not main)
2. Target PR to sprint branch: `sprint-$month-$numberOfWeek` (e.g., `sprint-june-4`)
3. Add tests for new code
4. **Before each commit, verify the build locally:**
   ```bash
   ./docker.local/bin/build.miners.sh    # For miner changes
   ./docker.local/bin/build.sharders.sh  # For sharder changes
   ```
5. Ensure CI passes
6. **Do NOT include `Co-Authored-By: Claude` in commit messages**

## Local Development

1. Start sharder first (provides genesis): `cd docker.local/sharder1 && ../bin/start.b0sharder.sh`
2. Start miners after: `cd docker.local/miner1 && ../bin/start.b0miner.sh`
3. Diagnostics: `http://localhost:7071/_diagnostics` (miners), `http://localhost:7171/_diagnostics` (sharders)

## Mainnet Miners

Get the current miner list from: `https://mainnet.zus.network/dns/network`

**Current mainnet miners (each has unique domain):**
1. https://msb01.datauber.net/miner01
2. https://msb01.safestor.net/miner01
3. https://viewpoint.ddns.net/miner01
4. https://miner5.0chain.net/miner01
5. https://m.sdredfox.com/miner01
6. https://miner4.0chain.net/miner01
7. https://hel.sdredfox.com/miner01
8. https://fi-m.th0r.eu/miner01
9. https://msb02.datauber.net/miner01
10. https://miner8.nodely.store/miner01
11. https://es-m.th0r.eu/miner01
12. https://0.fra.zcn.zeroservices.eu/miner01
13. https://mnr.zus-storage.com/miner01
14. https://miner1.bytepatch.io/miner01
15. https://miner9.0chain.net/miner01
16. https://mb01.0chainstaking.net/miner01
17. https://miner8.0chain.net/miner01
18. https://mining.zus-network.com/miner01

## Debugging Miner Diagnostics

When investigating network consensus issues on live miners:

**Key Diagnostic Endpoints:**
- `/_diagnostics` - Main status page (phase, round, VRF status, LFMB)
- `/_diagnostics/logs?detail=3` - Error logs with timestamps
- `/_diagnostics/n2n_logs?detail=3` - Network message send/receive activity
- `/_diagnostics/round_info` - Verification ticket details per block

**Important Rules:**

1. **Build Tag Location**: The `build_tag` is on line 4 of the main diagnostics page. Format: `build_tag:ea3b77059caf5c1f333b1f4695285650fcdc3384`. Use this to verify which code version a miner is running.

2. **Message Activity**: Check `/_diagnostics/n2n_logs?detail=3` for actual send/receive message counts - NOT the main diagnostics page. The main page may show misleading "0 sent/0 received" even when the node is actively communicating.

**Key Metrics to Monitor:**
- `vrfs`: Number of VRF shares collected (threshold: 17)
- `verification_tickets`: Number of tickets for notarization (threshold: 15)
- `LFMB`: Latest Finalized Merkle Block - should match across healthy miners
- `gn_mb_magic_block_number`: Current magic block being used for DKG keys

## DKG Recovery Diagnostics

**Endpoints for monitoring and recovering DKG keys:**

| Endpoint | Method | Description | Modifies State |
|----------|--------|-------------|----------------|
| `/_diagnostics/dkg/status` | GET | Show DKG status for current and previous MB | No |
| `/_diagnostics/dkg/test_recovery` | GET | Dry run - test if recovery is possible | No |
| `/_diagnostics/dkg/force_recovery` | GET | Force DKG recovery with backup | **Yes** |
| `/_diagnostics/dkg/backups` | GET | List available backup files | No |
| `/_diagnostics/dkg/restore?file=<path>` | GET | Restore DKG from backup file | **Yes** |

**Example Usage:**
```bash
# Check DKG status
curl https://<miner-host>/miner01/_diagnostics/dkg/status

# Test recovery (dry run - no changes)
curl https://<miner-host>/miner01/_diagnostics/dkg/test_recovery

# Force recovery (creates backup first)
curl https://<miner-host>/miner01/_diagnostics/dkg/force_recovery

# List backups
curl https://<miner-host>/miner01/_diagnostics/dkg/backups

# Restore from backup
curl "https://<miner-host>/miner01/_diagnostics/dkg/restore?file=data/dkg_backup/dkg_summary_19_20260123_120000.json"
```

**Security:** State-modifying endpoints (`force_recovery`, `restore`) are **localhost-only**. To use them remotely, SSH into the miner container first:
```bash
# SSH into miner container, then call locally
docker exec -it miner1 curl http://localhost:7071/_diagnostics/dkg/force_recovery
```

**Backups:** Stored in `data/dkg_backup/` with format `dkg_summary_{mb_number}_{timestamp}.json`
