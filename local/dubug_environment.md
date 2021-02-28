# Debug environment for 0Chain go executables

## Table of contents

- [Introduction](#introduction)
- [configure files](#configure-files)
    - [Node keys](#node-keys)
    - [Magic block](#magic-block)
    - [0chain.yaml](#0chain-yaml)
- [Reset databases](#reset-databases)
    - [Redis](#redis)
    - [Cassandra](#cassandra)

## Introduction

I will assume that you have set up your 0Chian as in the
[run_environment.md](https://github.com/0chain/0chain/blob/debug_builds/local/run_environment.md)
document and now you want to debug the chain on one of the machine.

I will be explaining this from the perspective of 
![Goland](edit_config_miner.png),
adapt for the IDE of your choice.

## 

## miner

[miner edit configuration](https://github.com/0chain/0chain/blob/debug_builds/local/edit_config_miner.png)
