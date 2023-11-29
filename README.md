            # Züs TestNet Setup with Docker Containers

[![Build](https://github.com/0chain/0chain/actions/workflows/build-&-publish-docker-image.yml/badge.svg)](https://github.com/0chain/0chain/actions/workflows/build-&-publish-docker-image.yml)
[![Test](https://github.com/0chain/0chain/actions/workflows/unit-test.yml/badge.svg)](https://github.com/0chain/0chain/actions/workflows/unit-test.yml)
[![GoDoc](https://godoc.org/github.com/0chain/0chain?status.png)](https://godoc.org/github.com/0chain/0chain)
[![codecov](https://codecov.io/gh/0chain/0chain/branch/staging/graph/badge.svg)](https://codecov.io/gh/0chain/0chain)

## Table of Contents
- [Züs Overview](#züs-overview)
- [Changelog](#changelog)
- [Quickstart](#quickstart)
- [Get Started](#get-started)
  - [1. Network Setup](#1-network-setup)
  - [2. Directory Setup for Miners and Sharders](#2-directory-setup-for-miners-and-sharders)
  - [3. Build and Start 0dns](#3-build-and-start-0dns)
  - [4. Setup Network](#4-setup-network)
  - [5. Building the Miner and Sharder Nodes](#5-building-the-miner-and-sharder-nodes)
  - [6. Configuring the Miner and Sharder Nodes](#6-configuring-the-miner-and-sharder-nodes)
  - [7. Starting the Miner and Sharder Nodes](#7-starting-the-miner-and-sharder-nodes)
  - [8. Building and Starting Blobber Nodes](#8-building-and-starting-blobber-nodes)
  - [Check Chain Status](#check-chain-status)
  - [Restarting the Nodes](#restarting-the-nodes)
  - [Cleanup](#cleanup)
- [Run 0chain on ec2 / vm / bare metal](https://github.com/0chain/0chain/blob/master/docker.aws/README.md)
- [Run 0chain on ec2 / vm / bare metal over https](https://github.com/0chain/0chain/blob/master/https/README.md)
- [Development](#development)
  - [Installing msgp](#installing-msgp)
  - [Dependencies for local compilation](#dependencies-for-local-compilation)
  - [Debugging](#debugging)
  - [Unit tests](#unit-tests)
  - [Creating The Magic Block](#creating-the-magic-block)
  - [Initial states](#initial-states)
  - [Minio Setup](#minio)
  - [Integration tests ](#integration-tests)
    - [Architecture](#architecture)
    - [Running Integration Tests](#running-integration-tests)
    - [Running Standard Tests](#running-standard-tests)
    - [Running complex scenario suites](#running-complex-scenario-suites)
    - [Running Blobber Tests](#running-blobber-tests)
    - [Adding new Tests](#adding-new-tests)
    - [Supported Conductor Commands](#supported-conductor-commands)
    - [Creating Custom Conductor Commands](#creating-custom-conductor-commands) 
  - [Benchmarks](#benchmarks)
  - [Swagger documentation](#swagger-documentation)

## Züs Overview 
[Züs](https://zus.network/) is a high-performance cloud on a fast blockchain offering privacy and configurable uptime. It is an alternative to traditional cloud S3 and has shown better performance on a test network due to its parallel data architecture. The technology uses erasure code to distribute the data between data and parity servers. Züs storage is configurable to provide flexibility for IT managers to design for desired security and uptime, and can design a hybrid or a multi-cloud architecture with a few clicks using [Blimp's](https://blimp.software/) workflow, and can change redundancy and providers on the fly.

For instance, the user can start with 10 data and 5 parity providers and select where they are located globally, and later decide to add a provider on-the-fly to increase resilience, performance, or switch to a lower cost provider.

Users can also add their own servers to the network to operate in a hybrid cloud architecture. Such flexibility allows the user to improve their regulatory, content distribution, and security requirements with a true multi-cloud architecture. Users can also construct a private cloud with all of their own servers rented across the globe to have a better content distribution, highly available network, higher performance, and lower cost.

[The QoS protocol](https://medium.com/0chain/qos-protocol-weekly-debrief-april-12-2023-44524924381f) is time-based where the blockchain challenges a provider on a file that the provider must respond within a certain time based on its size to pass. This forces the provider to have a good server and data center performance to earn rewards and income.

The [privacy protocol](https://zus.network/build) from Züs is unique where a user can easily share their encrypted data with their business partners, friends, and family through a proxy key sharing protocol, where the key is given to the providers, and they re-encrypt the data using the proxy key so that only the recipient can decrypt it with their private key.

Züs has ecosystem apps to encourage traditional storage consumption such as [Blimp](https://blimp.software/), a S3 server and cloud migration platform, and [Vult](https://vult.network/), a personal cloud app to store encrypted data and share privately with friends and family, and [Chalk](https://chalk.software/), a high-performance story-telling storage solution for NFT artists.

Other apps are [Bolt](https://bolt.holdings/), a wallet that is very secure with air-gapped 2FA split-key protocol to prevent hacks from compromising your digital assets, and it enables you to stake and earn from the storage providers; [Atlus](https://atlus.cloud/), a blockchain explorer and [Chimney](https://demo.chimney.software/), which allows anyone to join the network and earn using their server or by just renting one, with no prior knowledge required.

## Changelog
[CHANGELOG.md](CHANGELOG.md)

## Quickstart

Quickstart with a convenient bash script for deploying a Züs testnet locally. follow the guide mentioned below:
- [Deploy Züs testnet locally](https://docs.zus.network/guides/setup-a-blockchain/step-1-set-up-the-project)

## Get Started

### Required OS and Software Dependencies

 - Linux (Ubuntu Preferred) Version: 20.04 and Above
 - Mac(Apple Silicon or Intel) Version: Big Sur and Above
 - Windows Version: Windows 11 or 10 version 2004 and later requires WSL2. Instructions for installing WSL with docker can be found [here](https://github.com/0chain/0chain/blob/hm90121-patch-1/standalone_guides.md#install-wsl-with-docker).
 - Docker and Go must be installed to run the testnet containers. Instructions for installing Docker can be found [here](https://github.com/0chain/0chain/blob/hm90121-patch-1/standalone_guides.md#install-docker-desktop) and for Go find installation instructions [here](https://github.com/0chain/0chain/blob/hm90121-patch-1/standalone_guides.md#install-go). 
 
### 1. Network setup

1.1 Open terminal and clone the 0chain repo:
```
git clone https://github.com/0chain/0chain.git
```

1.2 Navigate to 0chain directory.

```
cd 0chain
```
#### MacOS
```bash
./macos_network.sh
```
#### Ubuntu/WSL2
Run the following script
```bash
./wsl_ubuntu_network_iptables.sh
```
### 2. Directory setup for Miners and Sharders

2.1) Inside the 0chain directory, run the following command:

```
sudo ./docker.local/bin/init.setup.sh
```

Response: The response will intialize 8 miner and 4 sharder directories in `0chain/docker.local/`
```
~/0chain/docker.local$ ls
Makefile    build.benchmarks  build.sc_unit_test     build.unit_test  miner2  miner6    sharder2
benchmarks  build.genkeys     build.sharder          config           miner3  miner7    sharder3
bin         build.magicBlock  build.swagger          docker-clean     miner4  miner8    sharder4
build.base  build.miner       build.test.multisigsc  miner1           miner5  sharder1  sql_script
```

### 3. Build and start 0dns

0dns service is responsible for connecting to the network and fetching all the magic blocks from the network which are saved in the DB. For building and starting 0dns:

3.1) Open another terminal window, clone the 0dns repo and navigate to 0dns directory using the command below:

```
git clone https://github.com/0chain/0dns.git
cd 0dns
```

3.2) For miner and sharder URLs to work locally, update `0dns/docker.local/config/0dns.yaml` file and set both `use_https` and `use_path` to `false`.

3.3) Then run the following command

```
./docker.local/bin/build.sh
```

3.4) Run the container using

```
./docker.local/bin/start.sh
```
### 4. Setup Network

4.1) Inside the git/0chain directory:
   
```
cd 0chain
``` 
4.2) Set up a network called testnet0 for each of these node containers to talk to each other.

**_Note: The config file should be providing the IP address of the nodes as per the IP addresses in this network._**

```
./docker.local/bin/setup.network.sh
```

## 5. Building the Miner and Sharder Nodes

5.1) Navigate to 0chain directory:
```
cd 0chain
``` 
5.2) First build the base containers, zchain_build_base and zchain_run_base

 ```
 ./docker.local/bin/build.base.sh
 ```
5.3) Build mocks from the Makefile in the repo, from git/0chain directory run:
   
   ```
    make build-mocks 
   ```
   Note: Mocks have to be built once in the beginning. Building mocks require mockery which can be installed from [here](https://github.com/0chain/0chain/blob/hm90121-patch-1/standalone_guides.md#install-brew-and-mockery).
   
5.4) Building the miners and sharders. From the git/0chain directory:

   5.4.1) To build the miner containers

   ```
   ./docker.local/bin/build.miners.sh
   ```

   5.4.2) To build the sharder containers

   ```
   ./docker.local/bin/build.sharders.sh
   ```

   5.4.3)(Optional)Syncing time (the host and the containers are being offset by a few seconds that throws validation errors as we accept transactions that are within 5 seconds of creation). This step is needed 
    periodically when you see the validation error.

   ```
   ./docker.local/bin/sync_clock.sh
   ```

## 6. Configuring the Miner and Sharder Nodes

6.1) Use `./docker.local/config/0chain.yaml` to configure the blockchain properties. The default options are set up for running the blockchain fast in development.

  6.1.1) If you want the logs to appear on the console - change `logging.console` from `false` to `true`

  6.1.2) If you want the debug statements in the logs to appear - change `logging.level` from `"info"` to `"debug"`

  6.1.3) If you want to change the block size, set the value of `server_chain.block.size`

  6.1.4) If you want to adjust the network relay time, set the value of `network.relay_time`

  6.1.5) If you want to turn off fees adjust `server_chain.smart_contract.miner` from `true` to `false`

**_Note: Remove sharder72 and miner75 from docker.local/config/b0snode2_keys.txt and docker.local/config/b0mnode5_keys.txt respectively if you are joining to local network._**

## 7. Starting the Miner and Sharder Nodes

7.1) For starting the nodes open 4 terminal tabs. Use the 1st terminal tab and be in the sharder1 (`0chain/docker.local/sharder1`) directory. On other 3 terminal tabs be in the miner directory(0chain/docker.local/miner|i)(miner1/2/3). 

Start sharder first because miners need the genesis magic block. On the sharder terminal tab, use the command below to start the sharders:

```
../bin/start.b0sharder.sh
```

Wait till the cassandra is started and the sharder is ready to listen to requests.

On the respective miner terminal tabs, use the command below to start the miners:

```
../bin/start.b0miner.sh
```

Note: The above commands will run 1 sharder and 3 miners for minimal setup. For running more sharders and blobbers repeat the process in more terminal tabs. 

## 8. Building and Starting Blobber Nodes

For detailed steps on building and starting blobbers, follow the guides below:

- [Directory Setup for Blobbers](https://github.com/0chain/blobber/tree/hm90121-patch-2#directory-setup-for-blobbers)
- [Building and Starting the Blobber Nodes](https://github.com/0chain/blobber/tree/hm90121-patch-2#building-and-starting-the-nodes)
 
8.1) After starting blobbers check whether the blobber has registered to the blockchain by running the zbox command below:

```
./zbox ls-blobbers
```
Note: In case you have not installed and configured zbox for testnet yet, follow the guides below:

 - [Install zboxcli](https://github.com/0chain/zboxcli/tree/hm90121-patch-1-1#1-installation)
 - [Configure zbox network](https://github.com/0chain/zboxcli/tree/hm90121-patch-1-1#2-configure-network) 

In the command response you should see the local blobbers mentioned with their urls for example `http://198.18.0.91:5051` and `http://198.18.0.92:5052`

Sample Response:
```
- id:                    7a90e6790bcd3d78422d7a230390edc102870fe58c15472073922024985b1c7d
  url:                   http://198.18.0.92:5052
  used / total capacity: 0 B / 1.0 GiB
  last_health_check:	  1635347427
  terms:
    read_price:          10.000 mZCN / GB
    write_price:         100.000 mZCN / GB / time_unit
    min_lock_demand:     0.1
    cct:                 2m0s
    max_offer_duration:  744h0m0s
- id:                    f65af5d64000c7cd2883f4910eb69086f9d6e6635c744e62afcfab58b938ee25
  url:                   http://198.18.0.91:5051
  used / total capacity: 0 B / 1.0 GiB
  last_health_check:	  1635347950
  terms:
    read_price:          10.000 mZCN / GB
    write_price:         100.000 mZCN / GB / time_unit
    min_lock_demand:     0.1
    cct:                 2m0s
    max_offer_duration:  744h0m0s
```

Note: When starting multiple blobbers, it could happen that blobbers are not being registered properly (not returned on `zbox ls-blobbers`). 
   
Blobber registration takes some time and adding at least 5 second wait before starting the next blobber usually avoids the issue.
  
8.2) Now you can create allocations on blobber and store files. For creating allocations you need tokens into your wallet, Running the command below in zwallet will give 1 token to wallet.

```sh
./zwallet faucet --methodName pour --input "need token"
```

You can specify the number of tokens required using the following command  for adding 5 tokens:

```sh
./zwallet faucet --methodName pour --input "need token" --tokens 5
```
Sample output from `faucet` prints the transaction.

```
Execute faucet smart contract success with txn:  d25acd4a339f38a9ce4d1fa91b287302fab713ef4385522e16d18fd147b2ebaf
```
To check wallet balance run `./zwallet getbalance` command

Response:
```
Balance: 5 ZCN (4.2299999999999995 USD)
```
8.3) Lock some tokens in blobber stake pools, use the commands below to lock tokens into stake pool: 

```
export BLOBBER1=f65af5d64000c7cd2883f4910eb69086f9d6e6635c744e62afcfab58b938ee25
export BLOBBER2=7a90e6790bcd3d78422d7a230390edc102870fe58c15472073922024985b1c7d
export BLOBBER3=2f051ca6447d8712a020213672bece683dbd0d23a81fdf93ff273043a0764d18
export BLOBBER4=2a4d5a5c6c0976873f426128d2ff23a060ee715bccf0fd3ca5e987d57f25b78e

./zbox sp-lock --blobber_id $BLOBBER1 --tokens 1
./zbox sp-lock --blobber_id $BLOBBER2 --tokens 1
./zbox sp-lock --blobber_id $BLOBBER3 --tokens 1
./zbox sp-lock --blobber_id $BLOBBER4 --tokens 1

```
Note: Atleast have 4 ZCN balance in your wallet before locking tokens into stake pool using the command above.

8.4) Then create new allocation using the command below:

```
./zbox newallocation --lock 0.5
```
Note: Atleast have 1 ZCN balance in your wallet before running the command above.

Now, you can store files in allocated space and execute a variety of operations using zboxcli. For a comprehensive list of zbox commands and their respective functionalities, please refer to the documentation [here](https://github.com/0chain/zboxcli/tree/hm90121-patch-1-1#commands-table).

## Check Chain Status

1. Ensure the port mapping is all correct:

```
docker ps
```

This should display a few containers and should include containers with images miner1_miner, miner2_miner and miner3_miner, and they should have the ports mapped like "0.0.0.0:7071->7071/tcp"

2. Confirming the servers are up and running. From a browser, visit

- http://localhost:7071/_diagnostics

- http://localhost:7072/_diagnostics

- http://localhost:7073/_diagnostics

to see the status of the miners.

Similarly, following links can be used to see the status of the sharders

- http://localhost:7171/_diagnostics

- http://localhost:7172/_diagnostics

- http://localhost:7173/_diagnostics

3. Connecting to redis servers running within the containers (you are within the appropriate miner directories)

Default redis (used for clients and state):

```
../bin/run.miner.sh redis redis-cli
```

Redis used for transactions:

```
../bin/run.miner.sh redis_txns redis-cli
```

4. Connecting to cassandra used in the sharder (you are within the appropriate sharder directories)

```
../bin/run.sharder.sh cassandra cqlsh
```

## Restarting the nodes

To reflect a change in config files 0chain.yaml and sc.yaml, just restart the miner or sharder to take the new configuration. If you're doing a code change locally or pulling updates from GitHub, you need to build.
```
git pull
docker.local/bin/build.base.sh && docker.local/bin/build.sharders.sh && docker.local/bin/build.miners.sh
```
For existing code and if you have tried running once, make sure there are no previous files and processes.
```
docker stop $(docker ps -a -q)
docker.local/bin/clean.sh
docker.local/bin/init.setup.sh
docker.local/bin/sync_clock.sh
```
Then go to individual miner/sharder:
```
../bin/start.b0sharder.sh (start sharders first!)
../bin/start.b0miner.sh
```
## Cleanup

1. If you want to restart the blockchain from the beginning

```
./docker.local/bin/clean.sh
```

This cleans up the directories within docker.local/miner* and docker.local/sharder*

**_Note: this script can take a while if the blockchain generated a lot of blocks as the script deletes
the databases and also all the blocks that are stored by the sharders. Since each block is stored as a
separate file, deleting thousands of such files will take some time._**

2. If you want to get rid of old unused docker resources:

```
docker system prune
```

### Running on systems with SELinux enabled

Library by `herumi` for working with BLS threshold signatures requires this flag turned on:

```
setsebool -P selinuxuser_execheap 1
```

If you are curious about the reasons for this, this thread sheds some light on the topic:

https://github.com/herumi/xbyak/issues/9

## Setting up Cassandra Schema

The following is no longer required as the schema is automatically loaded.

Start the sharder service that also brings up the cassandra service. To run commands on cassandra, use the following command

```
../bin/run.sharder.sh cassandra cqlsh
```

1. To create zerochain keyspace, do the following

```
../bin/run.sharder.sh cassandra cqlsh -f /0chain/sql/zerochain_keyspace.sql
```

2. To create the tables, do the following

```
../bin/run.sharder.sh cassandra cqlsh -k zerochain -f /0chain/sql/txn_summary.sql
```

3. When you want to truncate existing data (use caution), do the following

```
../bin/run.sharder.sh cassandra cqlsh -k zerochain -f /0chain/sql/truncate_tables.sql
```

## Development

### Installing msgp

Run the following command to install the msgp tool:

```sh
make install-msgp
```

We are using [msgp](https://github.com/0chain/msgp) to encode/decode data that store in MPT, it is unnecessary
to touch it unless there are data struct changes or new type of data structs need to store in MPT.


When we need to add a new data struct to MPT, for example:

```go
//go:generate msgp -io=false -tests=false -v
package main

type Foo struct {
	Name string
}

```

Note:
1. `msgp` does not support system type alias, so please do not use `datastore.Key` in MPT data struct, it is an alias of
system type `string`.
2. The `//go:generate msgp -io=false ...` works on file level, i.e, we only need to define it once a file,
so please check if it is already defined before adding.

Then run the following command from the project root to generate methods for serialization.

```sh
make msgp
```

A new file will then be generated as {file}_gen.go in the same dir where the data struct is defined.

### Dependencies for local compilation

You need to install `rocksdb` and `herumi/bls`, refer to `docker.local/build.base/Dockerfile.build_base` for necessary steps.

For local compilation it should be enough of `go build` from a submodule folder, e.g.
```
cd code/go/0chain.net/miner
go build
```

You can pass tag `development` if you want to simulate n2n delays.
And you also need tag `bn256` to build the same code as in production:
```
go build -tags "bn256 development"
```

## Debugging

### Bringing up the chain faster
```bash
./0chain_dev_deployment.sh
```

### Debug builds of 0chain

If you want to run a debug 0chain build you can follow the details contained in the
[`0chain/local` folder](https://github.com/0chain/0chain/blob/debug_builds/local/README.md).

Only one miner and one sharder can be run on any single machine, so you will need at least
three machines to for a working 0chain.

### Log files

The logs of the nodes are stored in log directory (/0chain/log on the container and docker.local/miner|sharder[n]/log in the host). The 0chain.log contains all the logs related to the protocol and the n2n.log contains all the node to node communication logs. The typical issues that need to be debugged is errors in the log, why certain things have not happened which requires reviewing the timestamp of a sequence of events in the network. Here is an example set of commands to do some debugging.

Find errors in all the miner nodes (from git/0chain)

```
grep ERROR docker.local/miner*/log/0chain.log
```

This gives a set of errors in the log. Say an error indicates a problem for a specific block, say abc, then

```
grep abc docker.local/miner*/log/0chain.log
```

gives all the logs related to block 'abc'

To get the start time of all the rounds

```
grep 'starting round' docker.local/miner*/log/0chain.log
```

This gives the start timestamps that can be used to correlate the events and their timings.

## Unit tests

 0chain unit tests verify the behaviour of individual parts of the program. A config for the base docker image can be provided on run to execute general unit tests.


![unit testing uml](https://user-images.githubusercontent.com/65766301/120052862-0b4ffd00-c045-11eb-83c8-977dfdb3038e.png)


Navigate to 0chain folder and run the script to build base docker image for unit testing :

```
cd 0chain
./docker.local/bin/build.base.sh
```

The base image includes all the dependencies required to test the 0chain code.

### Running Tests

Now run the script containing unit tests .

```
./docker.local/bin/unit_test.sh
```
OR to run the unit tests without the mocks,
```
./docker.local/bin/unit_test.sh --no-mocks 
```

The list of packages is optional, and if provided runs only the tests from those packages. The command for running unit tests with specific packages.

```
./docker.local/bin/unit_test.sh [<packages>]
```

###  Testing Steps

Unit testing happens over a series of steps one after the other.

#### Step 1: FROM zchain_build_base

This `FROM`step does the required preparation and specifies the underlying OS architecture to use the build image. Here we are using the base image created in the build phase.

#### Step 2: ENV SRC_DIR=/0chain

 The SRC_DIR  variable is a reference to a filepath which contains the code from your pull request. Here `/0chain` directory is specified as it is the one which was cloned.

#### Step 3: Setting the `GO111Module` variable to `ON`

`GO111MODULE` is an environment variable that can be set when using `go` for changing how Go imports packages. It was introduced to help ensure a smooth transition to the module system.

`GO111MODULE=on` will force using Go modules even if the project is in your GOPATH. Requires `go.mod` to work.

 Note: The default behavior in Go 1.16 is now **GO111MODULE**=on

#### Step 4: COPY ./code/go/0chain.net $SRC_DIR/go/0chain.net

This step copies the code from the source path to the destination path.

#### Step 5: RUN cd $SRC_DIR/go/0chain.net &&  go mod download

The RUN command is an image build step which allows installing of application and packages requited for testing while the`go mod download` downloads the specific module versions you've specified in the `go.mod`file.

#### Step 6: RUN cd $GOPATH/pkg/mod/github.com/valyala/gozstd@v1.5. &&     chmod -R +w . &&  make clean libzstd.a

This step runs the gozstd package and provides write permissions to the directory. gozstd which is a go wrapper for zstd (library) provides Go bindings for the libzstd C library. The `make clean` is run in the last to clean up the code and remove all the compiled object files from the source code

#### Step 7: WORKDIR $SRC_DIR/go

This step defines the working directory for running unit tests which is (0chain/code/go/0chain.net/).For all the running general unit tests their code coverage will be defined in the terminal like this

```
ok      0chain.net/chaincore/block      0.128s  coverage: 98.9% of statements
```

The above output shows 98.9% of code statements was covered with tests.

## Creating The Magic Block

First build the magic block image.

```
./docker.local/bin/build.magic_block.sh
```

Next, set the configuration file. To do this edit the docker.local/build.magicBlock/docker-compose.yml file. On line 13 is a flag "--config_file" set it to the magic block configuration file you want to use.

To create the magic block.

```
./docker.local/bin/create.magic_block.sh
```

The magic block and the dkg summary json files will appear in the docker.local/config under the name given in the configuration file.

The magic_block_file setting in the 0chain.yaml file needs to be updated with the new name of the magic block created.

Update the miner config file, so it is set to the new dkg summaries. To do this edit the docker.local/build.miner/b0docker-compose.yml file. On line 55 is a flag "--dkg_file" set it to the dkg summary files created with the magic block.

## Initial states

The balance for the various nodes is set up in a `initial_state.yaml` file.
This file is a list of node ids and token amounts.

The initial state yaml file is entered as a command line argument when
running a sharder or miner, falling that the `0chain.yaml`
`network.inital_states` entry is used to find the initial state file.

An example, that can be used with the preset ids, can be found at
[0chain/docker.local/config/initial_state.yaml`](https://github.com/0chain/0chain/blob/master/docker.local/config/initial_state.yaml)


### Minio

- You can use the inbuilt minio support to store blocks on cloud

You have to update minio_config file with the cloud creds data, The file can be found at `docker.local/config/minio_config.txt`.
The following order is used for the content :

```
CONNECTION_URL
ACCESS_KEY_ID
SECRET_ACCESS_KEY
BUCKET_NAME
REGION
```

- Your minio config file is then used in the docker-compose while starting the sharder node

```
--minio_file config/minio_config.txt
```

- You can either update the setting in the same file which is given above or create a new one with you config and use that as

```
--minio_file config/your_new_minio_config_file.txt
```

\*\*\_Note: Do not forget to put the file in the same config folder OR mount your new folder.

- Apart from private connection config, There are other options as well in the 0chain.yaml file to manage minio settings.

Sample config

```
minio:
  # Enable or disable minio backup, Do not enable with deep scan ON
  enabled: false
  # In Seconds, The frequency at which the worker should look for files, Ex: 3600 means it will run every 3600 seconds
  worker_frequency: 3600
  # Number of workers to run in parallel, Just to make execution faster we can have mutiple workers running simultaneously
  num_workers: 5
  # Use SSL for connection or not
  use_ssl: false
  # How old the block should be to be considered for moving to cloud
  old_block_round_range: 20000000
  # Delete local copy of block once it's moved to cloud
  delete_local_copy: true
```
- In minio the folders do not get deleted and will cause a slight increase in volume over time.


## Benchmarks
Benchmark 0chain smart-contract endpoints.

Runs testing.Benchmark on each 0chain endpoint. The blockchain database used in these tests is constructed from the parameters in the benchmark.yaml. file. Smartcontracts do not (or should not) access tha chain so a populated MPT database is enough to give a realistic benchmark.

More info in [read.me](code/go/0chain.net/smartcontract/benchmark/main/readme.md)


## Integration tests 

Integration testing combines individual 0chain modules and test them as a group. Integration testing evaluates the compliance of a system for specific functional requirements and usually occurs after unit testing .

For integration testing, A conductor which is RPC(Remote Procedure Call) server is implemented to control behaviour of nodes .To know more about the conductor refer to the [conductor documentation](https://github.com/0chain/0chain/blob/master/code/go/0chain.net/conductor/README.md)

### Architecture
A conductor requires the nodes to be built in a certain order to control them during the tests. A config file is defined in [conductor.config.yaml](https://github.com/0chain/0chain/blob/master/docker.local/config/conductor.config.yaml) which contains important details such as details of all nodes used and custom commands used in integration testing.

![integration testing](https://user-images.githubusercontent.com/65766301/120053178-6a624180-c046-11eb-8255-ac9b4e202e32.png)

For running multiple test cases,conductor uses a test suite which contains multiple sets of tests .A test suites can be categorized into 3 types of tests

`standard tests` - Checks whether chain continue to function properly despite bad miner and sharder participants

`view-change tests` - Checks whether addition and removal of nodes is working

.`blobber tests` - Checks whether storage functions continue to work properly despite bad or lost blobber, and confirms expected storage function failures

Below is an example of conductor test suite.

```
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

### Running Integration Tests

#### Prerequisites

Docker and Git must be installed to run the tests .

Install Git using the following command:

```
sudo apt install git
```

Docker installation instructions can be found [here](https://docs.docker.com/engine/install/).

#### Cloning the repository and Building Base Image

Clone the 0chain repository:

```
git clone https://github.com/0chain/0chain.git
```

Build miner docker image for integration test

```
(cd 0chain && ./docker.local/bin/build.miners-integration-tests.sh)
```

Build sharder docker image for integration test

```
(cd 0chain && ./docker.local/bin/build.sharders-integration-tests.sh)
```

NOTE: The miner and sharder images are designed for integration tests only. If wanted to run chain normally, rebuild the original images.

```
(cd 0chain && ./docker.local/bin/build.sharders.sh && ./docker.local/bin/build.miners.sh)
```

Confirm that view change rounds are set to 50 on `0chain/docker.local/config.yaml`

```
    start_rounds: 50
    contribute_rounds: 50
    share_rounds: 50
    publish_rounds: 50
    wait_rounds: 50
```

### Running standard tests

Run miners test

```
(cd 0chain && ./docker.local/bin/start.conductor.sh miners)
```

Run sharders test

```
(cd 0chain && ./docker.local/bin/start.conductor.sh sharders)
```

### Running complex scenario suites

1. These 2 scripts should be run with `view_change: false` in `0chain/docker.local/config.yaml`
  1.1. `(cd 0chain && ./docker.local/bin/start.conductor.sh no-view-change.byzantine)`
  1.2. `(cd 0chain && ./docker.local/bin/start.conductor.sh no-view-change.fault-tolerance)`
2. Set `view_change: true` in `0chain/docker.local/config.yaml` for the following 2 scripts
  2.1. `(cd 0chain && ./docker.local/bin/start.conductor.sh view-change.byzantine)`
  2.2. `(cd 0chain && ./docker.local/bin/start.conductor.sh view-change.fault-tolerance*)`

### Running blobber tests

Refer to [conductor documentation](https://github.com/0chain/0chain/blob/master/code/go/0chain.net/conductor/README.md#blobber)

### Adding new Tests

New tests can be easily added  to the conductor check [Updating conductor tests](https://github.com/0chain/0chain/blob/master/code/go/0chain.net/conductor/README.md#updating-conductor-tests) in the conductor documentation for more information.

### Enabling or Disabling Tests

Check [Temporarily disabling tests](https://github.com/0chain/0chain/blob/master/code/go/0chain.net/conductor/README.md#temporarily-disabling-tests) in the conductor documentation for more information

### Supported Conductor Commands
Check the [supported directives](https://github.com/0chain/0chain/blob/master/code/go/0chain.net/conductor/README.md#supported-directives) in the conductor documentation for more information.

### Creating Custom Conductor Commands

Check [Custom Commands](https://github.com/0chain/0chain/blob/master/code/go/0chain.net/conductor/README.md#custom-commands) in the conductor documentation for more information

## Swagger documentation

To generate swagger documentation you need go-swagger installed, visit https://goswagger.io/install.html for details.

You then need to run the makefile
```bash
make swagger
```
The documentation will be in `docs/swagger.md` and `docs/swagger.yaml`.
