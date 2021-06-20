#!/bin/sh

. ./paths.sh

#----------------------------------------------
# docker.local/bin/stop_all.sharder.sh

./stop_sharders.sh

#----------------------------------------------
cd "$zChain_Root" || exit

sudo rm -rf docker.local/sharder*/log/*


#----------------------------------------------

# !!! start.b0sharder.sh - For now just a single sharder.

cd "$zWorkflows_Base" || exit

./start_sharders.sh



