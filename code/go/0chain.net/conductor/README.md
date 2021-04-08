# Conductor

Conductor is an RPC server used for integrations tests. 
It can control nodes to make them behave badly in the network as required.
As an RPC server, it also receives events from the nodes to reliably manage the test progressions.

The conductor is automated as much as it can be. 

## How it works

The conductor requires the nodes to be built on a certain way in order to control them during the tests. 

The conductor uses a test suite which defined the tests to run.

The suite contains multiple sets of tests and each contains multiple test cases.

The individual test cases cannot be run in parallel which is the reason the tests run for hours.

### Suite sample and explanation

```yaml
# This enumerates the sets that are enabled.
enable: 
  - "Miner down/up"
  - "Blobber tests"

# Test sets defines the test cases it covers.
sets: 
  - name: "Miner down/up" 
    tests:
      - "Miner: 50 (switch to contribute)"
      - "Miner: 100 (switch to share)"
  - name: "Blobber tests"
    tests:
      - "All blobber tests"

# Test cases defines the execution flow for the tests.
tests: 
  - name: "Miner: 50 (switch to contribute)"
    flow: 
    # Flow is a series of commands.
    # The command can either be built-in in the conductor 
    # or custom defined in `conductor.config.yaml`
      - set_monitor: "sharder-1" # Most commands refer to nodes, these are defined in `conductor.config.yaml` 
      - cleanup_bc: {} # A sample built-in command that triggers stop on all nodes and clean up.
      - start: ['sharder-1']
      - start: ['miner-1', 'miner-2', 'miner-3']
      - wait_phase: 
          phase: 'contribute'
      - stop: ['miner-1']
      - start: ['miner-1']
      - wait_view_change:
          timeout: '5m'
          expect_magic_block:
            miners: ['miner-1', 'miner-2', 'miner-3']
            sharders: ['sharder-1']
  - name: "Miner: 100 (switch to share)"
    flow:
    ...
  - name: "All blobber tests"
    flow:
      - command:
          name: 'build_test_blobbers' # Sample custom command that executes `build_test_blobbers`
    ...
...
```

Jump to [Updating conductor tests](#updating-conductor-tests)

## Type of conductor tests

The conductor test suites are configured on yaml files. These test suites can be categorized into 3. 

1. `standard tests` - confirms chain continue to function properly despite bad miner and sharder participants
- docker.local/config/conductor.miners.yaml
- docker.local/config/conductor.sharders.yaml
2. `view-change tests` - confirms view change (addition and removal of nodes) is working
- docker.local/config/conductor.view-change-1.yaml
- docker.local/config/conductor.view-change-2.yaml
- docker.local/config/conductor.view-change-3.yaml
3. `blobber tests` - confirms storage functions continue to work despite bad or lost blobber, and confirms expected storage function failures
- docker.local/config/conductor.blobber-1.yaml
- docker.local/config/conductor.blobber-2.yaml

## Required setup

1. Git clone [0chain](https://github.com/0chain/0chain)
2. Build miner docker image for integration test
```sh
(cd 0chain && ./docker.local/bin/build.miners-integration-tests.sh)
```
2. Build sharder docker image for integration test
```sh
(cd 0chain && ./docker.local/bin/build.sharders-integration-tests.sh)
```

NOTE: The miner and sharder images are designed for integration tests only. If wanted to run chain normally, rebuild the original images.

```sh
(cd 0chain && ./docker.local/bin/build.sharders.sh && ./docker.local/bin/build.miners.sh)
```

3. Confirm that view change rounds are set to 50 on `0chain/docker.local/config.yaml`
```yaml
    start_rounds: 50
    contribute_rounds: 50
    share_rounds: 50
    publish_rounds: 50
    wait_rounds: 50
```

## Running standards tests
1. Run miners test
```sh
(cd 0chain && ./docker.local/bin/start.conductor.sh miners)
```
2. Run sharders test
```sh
(cd 0chain && ./docker.local/bin/start.conductor.sh sharders)
```

## Running view-change tests
1. Set `view_change: true` on `0chain/docker.local/config.yaml` 
2. Run view-change tests
```sh
(cd 0chain && ./docker.local/bin/start.conductor.sh view-change-1)
(cd 0chain && ./docker.local/bin/start.conductor.sh view-change-2)
(cd 0chain && ./docker.local/bin/start.conductor.sh view-change-3)
````

## Running blobber tests

Blobber tests require more setup.

1. Git clone [blobber](https://github.com/0chain/blobber)
2. Git clone [zboxcli](https://github.com/0chain/zboxcli)
3. Git clone [zwalletcli](https://github.com/0chain/blobber)
4. Git clone [0dns](https://github.com/0chain/0dns)
5. Confirm directories 
```
0chain/
blobber/
zboxcli/
zwalletcli/
0dns/
```

6. Install zboxcli
```sh
(cd zboxcli && make install)
```
7. Install zwalletcli
```sh
(cd zwalletcli && make install)
```
8. Patch 0dns
```sh
(cd 0dns && git apply --check ../0chain/docker.local/bin/conductor/0dns-local.patch)
(cd 0dns && git apply ../0chain/docker.local/bin/conductor/0dns-local.patch)
```
9. Patch blobbers
```sh
(cd blobber && git apply --check ../0chain/docker.local/bin/conductor/blobber-tests.patch)
(cd blobber && git apply ../0chain/docker.local/bin/conductor/blobber-tests.patch)
```
10. Add `~/.zcn/config.yaml` as follows
```yaml
block_worker: http://127.0.0.1:9091
signature_scheme: bls0chain
min_submit: 50
min_confirmation: 50
confirmation_chain_length: 3
max_txn_query: 5
query_sleep_time: 5
```
11. Apply if on Ubuntu 18.04

https://github.com/docker/for-linux/issues/563#issuecomment-547962928

The bug in Ubuntu 18.04 related. It relates to docker-credential-secretservice
package required by docker-compose and used by docker. A docker process
(a build, for example) can sometimes fails due to the bug. Some tests have
internal docker builds and can fail due to this bug.

12. Run blobber tests
```sh
(cd 0chain && ./docker.local/bin/start.conductor.sh blobber-1)
(cd 0chain && ./docker.local/bin/start.conductor.sh blobber-2) (edited)
```

## Updating conductor tests

### Changing the tests

### Temporarily disabling tests
To test a specific test, simply comment out the other tests on `enable` part of the conductor test yaml.

For example, only run `Miner down/up`

```yaml
enable:
  - "Miner down/up"
#  - "Sharder down/up"
#  - "All miners down/up"
#  - "All sharders down/up"
#  - "All nodes down/up"
```

### Common command settings
- `timeout` - all command support a timeout out of the box. valid values in time duration format (eg. `1s` for 1 second, `10m` for 10 minutes) 

### Supported commands

#### Built-in commands

TODO 

#### Custom commands

The list is available on [conductor.config.yaml](https://github.com/0chain/0chain/blob/c93e6022bee40e76eb35c408d8117dfb41b30bf7/docker.local/config/conductor.config.yaml#L117).

### Adding new command

#### To add a new command that executes a CLI command, simply update `docker.local/config/conductor.config.yaml`

Add a new command under `commands`
```yaml
  your_command_name:
    work_dir: "../blobber" # working directory where the command will be called is relative to ./0chain folder 
    exec: "../blobber/docker.local/bin/docker-clean.sh" ## CLI command to execute
    can_fail: true #
```

To use, simply provide the command name on test suite.

```yaml
  - name: "All blobber tests"
    flow:
      - command:
          name: 'your_command_name' 
```



