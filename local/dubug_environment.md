# Debug environment for 0Chain go executables

## Table of contents

- [Introduction](#introduction)
- [Custom build tags](#custom-build-tags)
- [Miner](#miner)     
- [Sharder](#Sharder)
- [Debug config files](#debug-config-files)

## Introduction

I will assume that you have set up your 0Chian as in the
[run_environment.md](https://github.com/0chain/0chain/blob/debug_builds/local/run_environment.md)
document and now you want to debug the chain on one of the machine.

I will be explaining this from the perspective of 
[Goland](https://www.jetbrains.com/go/promo/?gclid=CjwKCAiAm-2BBhANEiwAe7eyFHLK4O3pHcNb0Vi_q4l5pOkSoeLN4XTYNFXJYeJbFBWQ0NzEeTEixBoCAEoQAvD_BwE),
adapt for the IDE of your choice.

## Custom build tags

0Chain uses bn256 as a build tag, you want to add this to all your builds. In goland 
these are set up in the settings panel.  
![pierses image](https://github.com/0chain/0chain/blob/debug_builds/local/goland%20settings.png?raw=true)

## Miner
The builds in goland are set up in the `Run\Debug Configurations` panel accessed from the
`Run\Cnfigurations..` menu item. 
Possible way of populating the panel for building and running the miner would be:
* Run kind: Directory
* Directory: `/home/piers/GolandProjects/0chain/code/go/0chain.net/miner/miner`
* Output directory: `/home/piers/GolandProjects/0chain/local/miner`
* Working directory: `/home/piers/GolandProjects/0chain/local/miner`  
* Environment: `REDIS_HOST=redis;REDIS_TXNS=redis_txns`
* Use custom build tags: ticked
* Program arguments: `--development_mode 0 --keys_file ../config/b0snode1_keys.txt -mino_file ../config/mino_config.txt` 
![pierses image](https://github.com/0chain/0chain/blob/debug_builds/local/goland%20miner.png?raw=true)

## Sharder
The builds in goland are set up in the `Run\Debug Configurations` panel accessed from the
`Run\Cnfigurations..` menu item.
Possible way of populating the panel for building and running the sharder would be:
* Run kind: Directory
* Directory: `/home/piers/GolandProjects/0chain/code/go/0chain.net/sharder/sharder`
* Output directory: `/home/piers/GolandProjects/0chain/local/sharder`
* Working directory: `/home/piers/GolandProjects/0chain/local/sharder`
* Environment: `CASSANDRA_CLUSTER=cassandra`
* Use custom build tags: ticked
* Program arguments: `--development_mode 0 --keys_file ../config/b0snode1_keys.txt --mino_file ../config/mino_config.txt`
![pierses image](https://github.com/0chain/0chain/blob/debug_builds/local/goland%20sharder.png?raw=true)
  
## Debug config files

Each time we debug a new chain we need to set up the miner and sharder working directories.
```shell
0chain\local\bin\clean.sh
0chain\local\bin\init.setup.sh
```
Now edit the [magic block file](https://github.com/0chain/0chain/blob/debug_builds/local/run_environment.md#magic-block) and 
[`0chain.yaml`](https://github.com/0chain/0chain/blob/debug_builds/local/run_environment.md#0chain-yaml) file. Copy them 
into the respective `0chain/local/miner/config` or `0chain/local/sharder/config` directory along with the 
[node keys](https://github.com/0chain/0chain/blob/debug_builds/local/run_environment.md#node-keys)
file that identifies the miner or sharder that is being debuged.
