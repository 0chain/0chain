#!/bin/sh
cd code/go/test || exit
go build miner_stress.go
cd - || exit
