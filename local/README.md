# Debug TestNet Setup

> Warning!
This is intended for developers to construct debug builds. Other users
are advised to use the docker files as outlined in the main 
[readme.md](https://github.com/0chain/0chain/tree/debug_builds#initial-setup)


## Table of contents

- [Introduction](#introduction)
- [One time machine configuration](#one-time-machine-configuration)
- [New debug session setup](#new-debug-session-setup)

## Introduction

0Chain setup for running directly on machines. This allows the
for debugging running chains. The documentation is presented
from the perspective of an Ubuntu machine but should apply equally to
a Mac or alternative Linux OS.

As 0chains need at least three or four miners. When running outside docker containers
only one miner and one sharder is permitted on each machine, each such 0chain needs
to be run over at least three machines. 

## One time machine configuration

Each machine needs to be configured so that 0chain executables can be built, outlined in 
[build_environment.md](https://github.com/0chain/0chain/blob/debug_builds/local/build_environment.md)
and the database 0Chain uses installed as in
[instal_dbs.md](https://github.com/0chain/0chain/blob/debug_builds/local/install_dbs.md)

If you are using an IDE such as
[Goland](https://www.jetbrains.com/go/promo/?gclid=CjwKCAiAm-2BBhANEiwAe7eyFHLK4O3pHcNb0Vi_q4l5pOkSoeLN4XTYNFXJYeJbFBWQ0NzEeTEixBoCAEoQAvD_BwE),
you will want to set up you debug environments as outline in 
[debug_environment.md](https://github.com/0chain/0chain/blob/debug_builds/local/dubug_environment.md)

Each time you start a new chain you will to ensure all the various configuration 
files across the 0chian network's machines are correctly setup, as outlined in
[debug_environment.md](https://github.com/0chain/0chain/blob/debug_builds/local/dubug_environment.md#debug-config-files)

## New debug session setup

Each time you start a debug session you will want to do something similar to:

If you are running a miner
```shell
sudo 0chain/local/bin/reset_redis.sh
```

If you are running a sharder
```shell
reset_cassandra.sh
```
Wait for `cqlish` to come up then
```shell
cqlsh
Connected to Test Cluster at 127.0.0.1:9042.
[cqlsh 5.0.1 | Cassandra 3.11.10 | CQL spec 3.4.4 | Native protocol v4]
Use HELP for help.
cqlsh> --file 0chian\docker.local\config\cassandra\init.cql
cqlsh> --file 0chain\sql\zerochain_keyspace.sql
cqlsh> --file magic_block_map.sql`
cqlsh> --file txn_summary.sql
cqlsh> quit
```
Now you can clear and rebuild the runtime directories
```shell
0chain/local/bin/clean.sh
0chain/local/bin/init.setup.sh
```
You can enter your IDE run your 0chain apps; Remembering to start sharders first.
