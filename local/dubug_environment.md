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
[Goland](https://www.jetbrains.com/go/promo/?gclid=CjwKCAiAm-2BBhANEiwAe7eyFHLK4O3pHcNb0Vi_q4l5pOkSoeLN4XTYNFXJYeJbFBWQ0NzEeTEixBoCAEoQAvD_BwE),
adapt for the IDE of your choice.

## custom build tags

In goland these are set up in the settings panel.  
![pierses image](https://github.com/0chain/0chain/blob/debug_builds/local/goland%20settins.png?raw=true)


## miner

![pierses image](https://github.com/0chain/0chain/blob/debug_builds/local/goland%20miner.png?raw=true)

## sharder

![pierses image](https://github.com/0chain/0chain/blob/debug_builds/local/goland%20sharder.png?raw=true)
