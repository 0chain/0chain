 #!/bin/bash

#Run
#../reliable/restart_from_scratch.sh

# after it complete run

# /base/logs_blobbers.sh
# /base/logs_miners.sh
# /base/logs_sharders.sh

# Then run 
#../reliable/create_allocation.sh

# /base/logs_miners.sh

# $ ./logs_miners.sh zzzz
# ./docker.local/miner1/log/0chain.log:2021-03-12T20:01:56.592Z   INFO    storagesc/allocation.go:39   getAllocationsListzzzzzzzzzzzzz
# ./docker.local/miner2/log/0chain.log:2021-03-12T20:01:56.592Z   INFO    storagesc/allocation.go:39   getAllocationsListzzzzzzzzzzzzz
# ./docker.local/miner3/log/0chain.log:2021-03-12T20:01:56.574Z   INFO    storagesc/allocation.go:39   getAllocationsListzzzzzzzzzzzzz


# /base/logs_sharders.sh

# $ ./logs_sharders.sh zzz
# 2021-03-12T20:01:57.837Z        INFO    storagesc/allocation.go:39      getAllocationsListzzzzzzzzzzzzz


#--------------------------------------------------

# Let's create one more allocation
#../reliable/create_allocation.sh

# $ ./logs_miners.sh zzzz
# ./docker.local/miner1/log/0chain.log:2021-03-12T20:01:56.592Z   INFO    storagesc/allocation.go:39   getAllocationsListzzzzzzzzzzzzz
# ./docker.local/miner1/log/0chain.log:2021-03-12T20:05:20.014Z   INFO    storagesc/allocation.go:39   getAllocationsListzzzzzzzzzzzzz
# ./docker.local/miner1/log/0chain.log:2021-03-12T20:05:20.025Z   INFO    storagesc/allocation.go:39   getAllocationsListzzzzzzzzzzzzz
# ./docker.local/miner2/log/0chain.log:2021-03-12T20:01:56.592Z   INFO    storagesc/allocation.go:39   getAllocationsListzzzzzzzzzzzzz
# ./docker.local/miner2/log/0chain.log:2021-03-12T20:05:19.997Z   INFO    storagesc/allocation.go:39   getAllocationsListzzzzzzzzzzzzz
# ./docker.local/miner2/log/0chain.log:2021-03-12T20:05:20.014Z   INFO    storagesc/allocation.go:39   getAllocationsListzzzzzzzzzzzzz
# ./docker.local/miner3/log/0chain.log:2021-03-12T20:01:56.574Z   INFO    storagesc/allocation.go:39   getAllocationsListzzzzzzzzzzzzz
# ./docker.local/miner3/log/0chain.log:2021-03-12T20:05:19.998Z   INFO    storagesc/allocation.go:39   getAllocationsListzzzzzzzzzzzzz
# ./docker.local/miner3/log/0chain.log:2021-03-12T20:05:20.014Z   INFO    storagesc/allocation.go:39   getAllocationsListzzzzzzzzzzzzz


# $ ./logs_sharders.sh zzz
# 2021-03-12T20:01:57.837Z        INFO    storagesc/allocation.go:39      getAllocationsListzzzzzzzzzzzzz
# 2021-03-12T20:05:21.264Z        INFO    storagesc/allocation.go:39      getAllocationsListzzzzzzzzzzzzz

# OK.
# To each request there is a function call.


#--------------------------------------------------

# Now let's call
# ./zwallet faucet --methodName pour --input "{Pay day}"

# Execute faucet smart contract success with txn :  01d1b55807333e361bb04db1821660eaabd4602dc98a992b39104e348576be04
# OK


#--------------------------------------------------
# Now let's upload a test files

# $ ./upload_file_to_current_alloc.sh

# miner logs
# ./docker.local/miner3/log/0chain.log:2021-03-12T20:12:57.762Z   INFO    storagesc/allocation.go:24   getAllocationzzzzzzzzzzzzz
# ./docker.local/miner3/log/0chain.log:2021-03-12T20:12:58.167Z   INFO    storagesc/allocation.go:24   getAllocationzzzzzzzzzzzzz
# ./docker.local/miner3/log/0chain.log:2021-03-12T20:12:58.173Z   INFO    storagesc/allocation.go:24   getAllocationzzzzzzzzzzzzz
# ./docker.local/miner3/log/0chain.log:2021-03-12T20:12:58.178Z   INFO    storagesc/allocation.go:24   getAllocationzzzzzzzzzzzzz
# ./docker.local/miner3/log/0chain.log:2021-03-12T20:12:58.178Z   INFO    storagesc/allocation.go:24   getAllocationzzzzzzzzzzzzz
# ./docker.local/miner3/log/0chain.log:2021-03-12T20:12:58.178Z   INFO    storagesc/allocation.go:24   getAllocationzzzzzzzzzzzzz
# ./docker.local/miner3/log/0chain.log:2021-03-12T20:12:58.179Z   INFO    storagesc/allocation.go:24   getAllocationzzzzzzzzzzzzz
# ./docker.local/miner3/log/0chain.log:2021-03-12T20:12:58.179Z   INFO    storagesc/allocation.go:24   getAllocationzzzzzzzzzzzzz
# ./docker.local/miner3/log/0chain.log:2021-03-12T20:13:00.745Z   INFO    storagesc/allocation.go:24   getAllocationzzzzzzzzzzzzz
# ./docker.local/miner3/log/0chain.log:2021-03-12T20:13:00.748Z   INFO    storagesc/allocation.go:24   getAllocationzzzzzzzzzzzzz
# ./docker.local/miner3/log/0chain.log:2021-03-12T20:13:00.749Z   INFO    storagesc/allocation.go:24   getAllocationzzzzzzzzzzzzz
# ./docker.local/miner3/log/0chain.log:2021-03-12T20:13:00.750Z   INFO    storagesc/allocation.go:24   getAllocationzzzzzzzzzzzzz
# ./docker.local/miner3/log/0chain.log:2021-03-12T20:13:00.752Z   INFO    storagesc/allocation.go:24   getAllocationzzzzzzzzzzzzz
# ./docker.local/miner3/log/0chain.log:2021-03-12T20:13:00.754Z   INFO    storagesc/allocation.go:24   getAllocationzzzzzzzzzzzzz
# ./docker.local/miner3/log/0chain.log:2021-03-12T20:13:00.756Z   INFO    storagesc/allocation.go:24   getAllocationzzzzzzzzzzzzz
# ./docker.local/miner3/log/0chain.log:2021-03-12T20:13:01.992Z   INFO    storagesc/allocation.go:24   getAllocationzzzzzzzzzzzzz
# ./docker.local/miner3/log/0chain.log:2021-03-12T20:13:01.998Z   INFO    storagesc/allocation.go:24   getAllocationzzzzzzzzzzzzz
# ./docker.local/miner3/log/0chain.log:2021-03-12T20:13:01.999Z   INFO    storagesc/allocation.go:24   getAllocationzzzzzzzzzzzzz
# ./docker.local/miner3/log/0chain.log:2021-03-12T20:13:02.004Z   INFO    storagesc/allocation.go:24   getAllocationzzzzzzzzzzzzz
# ./docker.local/miner3/log/0chain.log:2021-03-12T20:13:02.004Z   INFO    storagesc/allocation.go:24   getAllocationzzzzzzzzzzzzz
# ./docker.local/miner3/log/0chain.log:2021-03-12T20:13:02.007Z   INFO    storagesc/allocation.go:24   getAllocationzzzzzzzzzzzzz
# ./docker.local/miner3/log/0chain.log:2021-03-12T20:13:02.008Z   INFO    storagesc/allocation.go:24   getAllocationzzzzzzzzzzzzz
# ./docker.local/miner3/log/0chain.log:2021-03-12T20:13:02.867Z   INFO    storagesc/allocation.go:24   getAllocationzzzzzzzzzzzzz
# ./docker.local/miner3/log/0chain.log:2021-03-12T20:13:02.872Z   INFO    storagesc/allocation.go:24   getAllocationzzzzzzzzzzzzz
# ./docker.local/miner3/log/0chain.log:2021-03-12T20:13:02.872Z   INFO    storagesc/allocation.go:24   getAllocationzzzzzzzzzzzzz
# ./docker.local/miner3/log/0chain.log:2021-03-12T20:13:02.873Z   INFO    storagesc/allocation.go:24   getAllocationzzzzzzzzzzzzz
# ./docker.local/miner3/log/0chain.log:2021-03-12T20:13:02.873Z   INFO    storagesc/allocation.go:24   getAllocationzzzzzzzzzzzzz
# ./docker.local/miner3/log/0chain.log:2021-03-12T20:13:02.873Z   INFO    storagesc/allocation.go:24   getAllocationzzzzzzzzzzzzz
# ./docker.local/miner3/log/0chain.log:2021-03-12T20:13:02.877Z   INFO    storagesc/allocation.go:24   getAllocationzzzzzzzzzzzzz




# sharder logs
# 2021-03-12T20:13:04.104Z        INFO    storagesc/allocation.go:24      getAllocationzzzzzzzzzzzzz
# 2021-03-12T20:13:04.105Z        INFO    storagesc/allocation.go:24      getAllocationzzzzzzzzzzzzz
# 2021-03-12T20:13:04.111Z        INFO    storagesc/allocation.go:24      getAllocationzzzzzzzzzzzzz
# 2021-03-12T20:13:20.998Z        INFO    storagesc/allocation.go:24      getAllocationzzzzzzzzzzzzz
# 2021-03-12T20:13:21.422Z        INFO    storagesc/allocation.go:24      getAllocationzzzzzzzzzzzzz
# 2021-03-12T20:13:21.845Z        INFO    storagesc/allocation.go:24      getAllocationzzzzzzzzzzzzz
# 2021-03-12T20:13:23.560Z        INFO    storagesc/allocation.go:24      getAllocationzzzzzzzzzzzzz
# 2021-03-12T20:13:29.459Z        INFO    storagesc/allocation.go:24      getAllocationzzzzzzzzzzzzz
# 2021-03-12T20:13:30.271Z        INFO    storagesc/allocation.go:24      getAllocationzzzzzzzzzzzzz
# 2021-03-12T20:13:31.530Z        INFO    storagesc/allocation.go:24      getAllocationzzzzzzzzzzzzz
# 2021-03-12T20:13:34.492Z        INFO    storagesc/allocation.go:24      getAllocationzzzzzzzzzzzzz
# 2021-03-12T20:13:37.465Z        INFO    storagesc/allocation.go:24      getAllocationzzzzzzzzzzzzz
# 2021-03-12T20:13:40.438Z        INFO    storagesc/allocation.go:24      getAllocationzzzzzzzzzzzzz
# 2021-03-12T20:13:42.563Z        INFO    storagesc/allocation.go:24      getAllocationzzzzzzzzzzzzz


#--------------------------------------------------

# Now let's restart miners
# stop_miners.sh
# start_miners.sh

# And then

# $ ./logs_miners.sh zzzz
#empty

# and sharder also toped to query the getAllocationzzzzzzzzzzzzz function

#--------------------------------------------------

# RESULT

# After miners restart
# the storagec functions getAllocation and getAllocations are not called anymore.

# And also
#$ ./zwallet faucet --methodName pour --input "{Pay day}"
# gives and error
# Execute faucet smart contract failed. submit transaction failed. {"code":"entity_not_found","error":"entity_not_found: client not found with id = client:c77fb42ac1c5390788e07d8fe5e0e743ac5a29b770c83fd24bacecb5dfaa3958"}



