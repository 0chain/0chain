# Debug TestNet Setup

> Warning!
This is intended for developers to construct debug builds. Other users
are advised to use the docker files as outlined in the main 
[readme.mn](https://github.com/0chain/0chain/tree/debug_builds#initial-setup)


## Table of contents

- [Introduction](#introduction)
- [One time machine configuration](#one-time-machine-configuration)
- [Before each debug run](#before-each-debug-run)

## Introduction

0Chain setup for running directly on machines. This allows the
for debugging running chains. The documentation is presented
from the perspective of an Ubuntu machine but should apply equally to
a Mac or alternative Linux OS.

As 0chains need at least three or four miners. When running outside docker containers
only one miner and one sharder is permitted on each machine, each such 0chain needs
to be run over at least three machines. 

### One time machine configuration

Each machine needs to be configured so that 0chain executables can be built, outlined in 
[build_environment.md](https://github.com/0chain/0chain/blob/debug_builds/local/build_environment.md)
and the database 0Chain uses installed as in
[instal_dbs.md](https://github.com/0chain/0chain/blob/debug_builds/local/install_dbs.md)

If you are using an IDE such as
[Goland](https://www.jetbrains.com/go/promo/?gclid=CjwKCAiAm-2BBhANEiwAe7eyFHLK4O3pHcNb0Vi_q4l5pOkSoeLN4XTYNFXJYeJbFBWQ0NzEeTEixBoCAEoQAvD_BwE),
you will want to set up you debug environments as outline in 
[debug_environment.md](https://github.com/0chain/0chain/blob/debug_builds/local/dubug_environment.md)

### Before each debug run

Each time you start a new chain you will to set up all the various configuration across the
0chian network's machines, such a proccess is document in 
[debug_environment.md](https://github.com/0chain/0chain/blob/debug_builds/local/dubug_environment.md#debug-config-files)