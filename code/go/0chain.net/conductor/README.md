# Conductor

Conductor is a program used for orchestrate a 0chain network and running integrations tests agains it.
It can control nodes to make them behave badly in the network as required.

A RPC server is exposed, so the communication between the conductor and the network nodes is possible. Through this channel of the communication the conductor receives information about the nodes state and events that happen in the blockchain in order to proceed in the tests. The client queries the conductor to know if it should behave in a malicious way.

Currently, the conductor tests cover miners, sharders, blobbers and authorizers. They also allow to test client tools like `zboxcli` and `zwalletcli`.

## How it works

The conductor tests are present in `yaml` configuration files located in `docker.local/config`. The conductor server searches in this location for a file with the name used as argument in the start conductor command.

The conductor requires the nodes to be built on a certain way in order to control them during the tests.
Particularly, when miners, sharders, blobbers and authorizers are built, it uses a tag `integration_tests`.
The `go build` will use the go files ending with `_integration_tests.go` instead of `_main.go` files.
The `_integration_tests.go` copy communicates with the conductor through RPC.

During run time, the conductor loads a yaml file for its config and uses a test suite which defines the tests.

The test runs successfully if the conductor is able to run all the steps without issues. A timeout is associated with most of the steps that when reached make the test fail.

### Conductor config

The config file is defined in [conductor.config.yaml](https://github.com/0chain/0chain/blob/master/docker.local/config/conductor.config.yaml)

The important details in the config file are the following.

- details of all nodes used
- custom commands used in tests
- `stuck_warning_threshold` setting to show additional output when the chain is stuck for more than specified duration

### Conductor test suite

The test suite contains multiple sets of tests and each contains multiple test cases.

The individual test cases cannot be run in parallel which is the reason the tests run for hours.

Below is a sample of test suite.

```yaml
# Under `enable` is the list of sets that will be run.
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
    # Flow is a series of directives.
    # The directive can either be built-in in the conductor
    # or custom command defined in "conductor.config.yaml"
      - set_monitor: "sharder-1" # Most directive refer to node by name, these are defined in `conductor.config.yaml`
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

## Category of conductor tests supported

The conductor test suites are configured on yaml files. These test suites can be categorized into 3.

1. `standard tests` - confirms chain continue to function properly despite bad miner and sharder participants
- [conductor.miners.yaml](https://github.com/0chain/0chain/blob/master/docker.local/config/conductor.miners.yaml)
- [conductor.sharders.yaml](https://github.com/0chain/0chain/blob/master/docker.local/config/conductor.sharders.yaml)
2. `complex scenarios` - confirms chain continues to function properly despite byzantine attacks and faults
- [conductor.no-view-change.byzantine.yaml](https://github.com/0chain/0chain/blob/master/docker.local/config/conductor.no-view-change.byzantine.yaml)
- [conductor.no-view-change.fault-tolerance.yaml](https://github.com/0chain/0chain/blob/master/docker.local/config/conductor.no-view-change.fault-tolerance.yaml)
- [conductor.view-change.byzantine.yaml](https://github.com/0chain/0chain/blob/master/docker.local/config/conductor.view-change.byzantine.yaml)
- conductor.view-change.fault-tolerance*.yaml
3. `blobber tests` - confirms storage functions continue to work despite bad or lost blobber, and confirms expected storage function failures
- [conductor.blobber-1.yaml](https://github.com/0chain/0chain/blob/master/docker.local/config/conductor.blobber-1.yaml)
- [conductor.blobber-2.yaml](https://github.com/0chain/0chain/blob/master/docker.local/config/conductor.blobber-2.yaml)
- [conductor.blobber-3.yaml](https://github.com/0chain/0chain/blob/master/docker.local/config/conductor.blobber-3.yaml)
- [conductor.validator-1.yaml](https://github.com/0chain/0chain/blob/master/docker.local/config/conductor.validator-1.yaml)
4. `authorizer tests` - confirms burns and mints continue to work despite authorizers bad behaviours
- [conductor.authorizer.yaml](https://github.com/0chain/0chain/blob/master/docker.local/config/conductor.authorizer.yaml)

## Test cases covered

To know about the specific test cases covered by the conductor tests, navigate to the test suite files listed above.

## Required setup

Below are the basic setup required to run the test suites.

### 1. Clone the repo
```sh
git clone git@github.com:0chain/0chain.git && cd 0chain
```

### 2. Init Setup
```sh
./docker.local/bin/init.setup.sh
```
this will create folder called sharder* and miner* inside `./docker.local/` folder.

### 3. Setup the network
```sh
./docker.local/bin/setup.network.sh
```

### 4. Build the base image
```sh
./docker.local/bin/build.base.sh
```

### 5. Build miner and sharder docker images for integration test

#### a. Build miner docker image for integration test

```sh
./docker.local/bin/build.miners-integration-tests.sh
```

#### b. Build sharder docker image for integration test

```sh
./docker.local/bin/build.sharders-integration-tests.sh
```

NOTE: The miner and sharder images are designed for integration tests only. If wanted to run chain normally, rebuild the original images by running the folowing:

```sh
./docker.local/bin/build.sharders.sh && ./docker.local/bin/build.miners.sh
```

### 6. Confirm that view change rounds are set to 50 on `0chain/docker.local/config/sc.yaml`

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

1. Set `view_change: true` on `0chain/docker.local/config/0chain.yaml`
2. Run view-change tests

```sh
(cd 0chain && ./docker.local/bin/start.conductor.sh view-change.fault-tolerance)
(cd 0chain && ./docker.local/bin/start.conductor.sh view-change.byzantine)
(cd 0chain && ./docker.local/bin/start.conductor.sh view-change-3)
```

## <a name="blobber"></a>Running blobber tests

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

8. Build 0dns

```sh
(cd 0dns && ./docker.local/bin/init.sh)
(cd 0dns && ./docker.local/bin/build.sh)
```

9. Init setup for blobbers

```sh
(cd blobber && ./docker.local/bin/blobber.init.setup.sh)
```

10. Build blobber base
```sh
(cd blobber && ./docker.local/bin/build.base.sh)
```

11. Add `~/.zcn/config.yaml` as follows

```yaml
block_worker: http://127.0.0.1:9091
signature_scheme: bls0chain
min_submit: 50
min_confirmation: 50
confirmation_chain_length: 3
max_txn_query: 5
query_sleep_time: 5
```

12. Apply if on Ubuntu 18.04

https://github.com/docker/for-linux/issues/563#issuecomment-547962928

The bug in Ubuntu 18.04 related. It relates to docker-credential-secretservice
package required by docker-compose and used by docker. A docker process
(a build, for example) can sometimes fail due to the bug. Some tests have
internal docker builds and can fail due to this bug.

13. Run blobber tests

```sh
(cd 0chain && ./docker.local/bin/start.conductor.sh blobber-1)
(cd 0chain && ./docker.local/bin/start.conductor.sh blobber-2)
(cd 0chain && ./docker.local/bin/start.conductor.sh blobber-3)
(cd 0chain && ./docker.local/bin/start.conductor.sh validator-1)
```

## <a name="authorizer"></a>Running authorizer tests

Blobber tests require more setup.

1. Git clone [authorizer](https://github.com/0chain/token_bridge_authserver)
2. Git clone [zboxcli](https://github.com/0chain/zboxcli)
3. Git clone [zwalletcli](https://github.com/0chain/blobber)
4. Git clone [0dns](https://github.com/0chain/0dns)
5. Confirm directories

```
0chain/
token_bridge_authserver/
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

8. Build 0dns

```sh
(cd 0dns && ./docker.local/bin/init.sh)
(cd 0dns && ./docker.local/bin/build.sh)
```

9. Init setup for authorizer

```sh
(cd token_bridge_authserver && ./docker.local/bin/authorizer.init.setup.sh)
```

10. Build authorizer integration image
```sh
(cd blobber && ./docker.local/bin/build.authorizer-integration-tests.sh)
```

11. Add `~/.zcn/config.yaml` as follows

```yaml
block_worker: http://127.0.0.1:9091
signature_scheme: bls0chain
min_submit: 50
min_confirmation: 50
confirmation_chain_length: 3
max_txn_query: 5
query_sleep_time: 5
```

12. Apply if on Ubuntu 18.04

https://github.com/docker/for-linux/issues/563#issuecomment-547962928

The bug in Ubuntu 18.04 related. It relates to docker-credential-secretservice
package required by docker-compose and used by docker. A docker process
(a build, for example) can sometimes fail due to the bug. Some tests have
internal docker builds and can fail due to this bug.

13. Run authorizer tests

```sh
(cd 0chain && ./docker.local/bin/start.conductor.sh authorizer)
```

## Code structure

The conductor code is located in `code/go/0chain.net/conductor`. Here we highlight the next directories:

* `cases` - has all test cases that should be linked to directives in `config`. These test cases are executed on directive `make_test_case_check`.
* `conductor` - has all features related to reading test file and the directives instructions.
* `conductrpc` - has rpc server and rpc client code to allow communication between conductor and nodes.
* `config` - registry of the directives and instructions to read the test configuration file.

## Updating conductor tests

### Updating the tests

To add more tests, simply create new test cases and add them to existing or new set.
Then be sure to enable the test set if creating a new one.

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

### Directives

The directives are test steps executed by the conductor. Each directive has a set of instructions programmed in the conductor. There are different purposes for creating a new different directive. We may need to inject a new behaviour in some node, waiting for a new event or creating a new test case.

The directives may be used for example for:
1. Starting nodes;
2. Changing conductor nodes state what allows the injection of a new behaviour when the node is accordingly programmed;
3. Waiting for events to happen like reaching a specific round, waiting for a view change, etc;
4. Run commands (bash scripts);
5. Execute verifications (test cases configured).

To insert a new directive that will change some node behaviour you should:
1. Declare directive in `code/go/0chain.net/conductor/config/registry.go`;
2. Create the structures inside `code/go/0chain.net/conductor/config` that will allow to read the configuration defined in the test file;
3. Update the conductor state, so it can be read by the nodes;
4. Use a `_integration_tests.go` to read the conductor state and program the behaviour you want.


### Common directive properties

- `timeout` - All commands support a timeout out of the box. Valid values in time duration format (eg. `1s` for 1 second, `10m` for 10 minutes). The default is 2 minutes.
- `must_fail` - Whether the test should fail if the command throw an error. By default, built-in commands have `must_fail` as false. Custom commands however is configurable with`should_fail` when declared.

### Supported directives

#### Built-in directives

1. **common setups**

- `set_monitor` - initiate the node from where blockchain events will be accepted
- `cleanup_bc` - stop all nodes, reset rounds, and clean up data using `cleanup_command` defined on `conductor.config.yaml`
- `env` - set environment variables that might affect commands to start/stop nodes. e.g. `CLI_ARGS` will effectively add arguments to command in `b0docker-compose.yml`

2. **common nodes control**

- `start` - starts the list of nodes. The start script used is defined on `conductor.config.yaml`
- `stop` - stops the list of nodes. The stop script used is defined on `conductor.config.yaml`
- `start_lock` - starts the list of nodes but lock them such that the nodes do nothing (eg. does not sign, does not generate blocks)
- `unlock` - update the state of the list of nodes to be no longer locked

3. **wait for an event of the monitor**

- `wait_view_change` - wait until a view change occurred
  - properties
    ```yaml
    # Name the round of this view change.
    # UNUSED in any of the tests
    remember_round: <string>
    # expectations on this view change.
    expect_magic_block:
      # Number is expected Magic Block number.
      # Use of MB number is more stable for the tests, since miners can vote for restart DKG process from start.
      number: <int64>
      # Round ignored if it's zero.
      # If set a positive value, then this round is expected.
      # UNUSED in any of the tests
      round: <int64>
      # RoundNextVCAfter used in combination with "remember_round".
      # This directive expects next VC round after the remembered one.
      # Empty string is ignored.
      # UNUSED in any of the tests
      round_next_vc_after: <string>
      # Sharders expected in MB.
      sharders: <array of strings>
      # Miners expected in MB.
      miners: <array of strings>
    ```
- `wait_phase` - wait until a phase ocurred
  - properties
    ```yaml
    # Phase can be any of 'start', 'contribute', 'share', 'publish', 'wait'
    phase: <string>
    # ViewChangeRound is the name of the view change round.
    # UNUSED in any of the tests
    view_change_round: <string>
    ```
- `wait_round` - wait until a round
  - properties
    ```yaml
    # Round is the blockchain round.
    round: <int64>
    # RoundName is the name of the round.
    # UNUSED in any of the tests
    name: <string>
    # Shift is the number of rounds to wait from current or "name" round if provided.
    shift: <int64>
    ```
- `wait_contribute_mpk` - wait for a miner's MPK
  - properties
    ```yaml
    # Miner is the name of the node.
    miner: <string>
    ```
- `wait_share_signs_or_shares` - waits for a miner's share signs or shares
  - properties
    ```yaml
    # Miner is the name of the node.
    miner: <string>
    ```
- `wait_add` - waits for the list of nodes to be added to blockchain
  - properties
    ```yaml
    # Miners are the names of the nodes.
    miners: <array of string>
    # Sharders are the names of the nodes.
    sharders: <array of string>
    # Blobbers are the names of the nodes.
    blobbers: <array of string>
    ```
- `wait_no_progress` - waits to confirm there is no progress on rounds. Anything less than 10 rounds after is acceptable as no progress.
- `wait_no_view_change`- waits to confirm there is no more view change after the round specified.
  ```yaml
  # Round is the blockchain round after which no view change is expected.
  round: <int64>
  ```
- `wait_sharder_keep` - waits for sharder keep on the list of sharders
  - properties
    ```yaml
    # Sharders are the names of the nodes.
    sharders: <array of string>
    ```

4. **control nodes behavior / misbehavior**

- `set_revealed` - reveal the list of nodes. A revealed node sends it share.
- `unset_revealed` - hide the list of nodes. A hidden node does not sends it share.
  - This is currently UNUSED
- `generators_failure` - prevents generators selected at start of the specified round (as in some setups they aren't known beforehand) from generating blocks for the duration of the whole round including all restarts.

5. **Byzantine blockchain**

- `vrfs` - have list of miners send bad VRFS
  - properties
    ```yaml
    # By miners
    by: <array of strings>
    # Good to miners
    good: <array of strings>
    # Bad to miners
    bad: <array of strings>
    ```
- `round_timeout` - have list of miners send bad round timeout
  - properties
    ```yaml
    # By miners
    by: <array of strings>
    # Good to miners
    good: <array of strings>
    # Bad to miners
    bad: <array of strings>
    ```
- `competing_block` - have one on the list of miners generate its own block
  - properties
    ```yaml
    # By miners
    by: <array of strings>
    ```
- `sign_only_competing_blocks` - have list of miners sign the competing blocks
  - properties
    ```yaml
    # By miners
    by: <array of strings>
    ```
- `double_spend_transaction` - have list of miners readd a previous transaction
  - properties
    ```yaml
    # By miners
    by: <array of strings>
    ```
- `wrong_block_sign_hash` - have list of miners use an invalid signature hash
  - properties
    ```yaml
    # By miners
    by: <array of strings>
    ```
- `wrong_block_sign_key` - have list of miners use an invalid secret key
  - properties
    ```yaml
    # By miners
    by: <array of strings>
    ```
- `wrong_block_hash` - have list of miners use an invalid block hash
  - properties
    ```yaml
    # By miners
    by: <array of strings>
    ```
- `verification_ticket_group` - unimplemented
- `wrong_verification_ticket_hash` - have list of miners send invalid verification ticket signature hash
  - properties
    ```yaml
    # By miners
    by: <array of strings>
    # Good to miners
    good: <array of strings>
    # Bad to miners
    bad: <array of strings>
    ```
- `wrong_verification_ticket_key` - have list of miners send invalid verification ticket signature hash (wrong key)
  - properties
    ```yaml
    # By miners
    by: <array of strings>
    # Good to miners
    good: <array of strings>
    # Bad to miners
    bad: <array of strings>
    ```
- `wrong_notarized_block_hash` - unimplemented
- `wrong_notarized_block_key` - unimplemented
- `notarize_only_competing_block` - unimplemented
- `notarized_block` - unimplemented

6. **Byzantine blockchain sharders**

- `finalized_block` - have list of sharders returns a different block hash for last finalized block
  - properties
    ```yaml
    # By sharders
    by: <array of strings>
    ```
- `magic_block` - have list of sharders returns a different block hash for last finalized magic block
  - properties
    ```yaml
    # By sharders
    by: <array of strings>
    ```
- `verify_transaction` - have list of sharders returns a hash and data on transaction verification
  - properties
    ```yaml
    # By sharders
    by: <array of strings>
    ```

7. **Byzantine view change**

- `mpk` - have list of miners send bad MPK
  - properties
    ```yaml
    # By miners
    by: <array of strings>
    # Good to miners
    good: <array of strings>
    # Bad to miners
    bad: <array of strings>
    ```
- `share` - have list of miners send bad DKG
  - properties
    ```yaml
    # By miners
    by: <array of strings>
    # Good to miners
    good: <array of strings>
    # Bad to miners
    bad: <array of strings>
    ```
- `signature` - have list of miners send bad sign share
  - properties
    ```yaml
    # By miners
    by: <array of strings>
    # Good to miners
    good: <array of strings>
    # Bad to miners
    bad: <array of strings>
    ```
- `publish` - have list of miners publish bad sign share
  - properties
    ```yaml
    # By miners
    by: <array of strings>
    # Good to miners
    good: <array of strings>
    # Bad to miners
    bad: <array of strings>
    ```

8. **blobber**

- `storage_tree` - unimplemented
- `validator_proof` - unimplemented
- `challenges` - unimplemented

#### Custom commands

The list is available on [conductor.config.yaml](https://github.com/0chain/0chain/blob/master/docker.local/config/conductor.config.yaml#L146).

### Adding new command

#### To add a new command that executes a CLI command, simply update `docker.local/config/conductor.config.yaml`

Add a new command under `commands`

```yaml
your_command_name:
  work_dir: "../blobber" # working directory where the command will be called is relative to ./0chain folder
  exec: "../blobber/docker.local/bin/docker-clean.sh" ## CLI command to execute
  can_fail: true #
```

To use, simply provide the `command` directive and the custom command name on test suite.

```yaml
- name: "All blobber tests"
  flow:
    - command:
        name: "your_command_name"
```

## Debugging

The output of start conductor command shows the tests that being executed, the tests result and the error description if the test fails.

It is generated the next logs for each node that runs in the conductor test.
* `0chain/conductor/logs` shows logs about the building of the docker images and the docker containers initialization. Normally, you can see here errors if the test is stuck on starting nodes.
* `0chain/docker.local/miner*` shows miners application logs.
* `0chain/docker.local/sharder*` shows sharders application logs.
* `blobber/docker.local/blobber*` shows miners application logs.
* `token_bridge_authserver/docker.local/authorizer*` shows bridges application logs.

The test cases are highly dependent on the configuration used in the network. The files are located in:
* `0chain/docker.local/0chain.yaml`
* `0chain/docker.local/sc.yaml`
* `blobber/docker.local/conductor-config/0chain_blobber.yaml`
* `blobber/docker.local/conductor-config/0chain_validator.yaml`
* `token_bridge_authserver/config/config.yaml`
* `token_bridge_authserver/config/authorizer.yaml`

Some tests also requires the magic block to have only the nodes in use.



