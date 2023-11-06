#!/bin/bash

path=$1
recreate_script=$2
recreate_script_workdir=$3

curdir="$(pwd)"

rm -rf $path;
cd $recreate_script_workdir;
"./$recreate_script";
cd $curdir;