TestNet Setup with Docker Containers:

Initial Setup:

1) Directory setup for miners & sharders. In the git/0chain run the following command

> ./docker.local/bin/init_setup.sh

2) Setup a network called testnet0 for each of these node containers to talk to each other.

Note: The config file should be providing the IP address of the nodes as per the IP addresses in this network.

> ./docker.local/bin/setup_network.sh

Building and starting the nodes:

1) Open 5 terminal tabs. Use the first one for building the containers by being in git/0chain directory.
Use the next 3 for 3 miners and be in the respective miner<i> directories created above in docker.local.
Use the 5th terminal and be in the sharder1 directory.

2) Building the miners and sharders. From the git/0chain directory use

2.1) To build the miner containers

> ./docker.local/bin/build_miners.sh

2.2) To build the sharder containers

> ./docker.local/bin/build_sharders.sh

for building the 1 sharder.

2.3) Syncing time (the host and the containers are being offset by a few seconds that throws validation errors as we accept transactions that are within 5 seconds of creation). This step is needed periodically when you see the validation error.

> ./docker.local/bin/sync_clock.sh

3) Starting the nodes. On each of the miner terminals use the commands (note the .. at the beginning. This is because, these commands are run from within the docker.local/<miner/sharder|i> directories and the bin is one level above relative to these directories)

> ../bin/miner.start.sh

On the sharder terminal, use

> ../bin/sharder.start.sh

4) Generating Test Transactions:

4.1) To build the miner_stress program from git/0chain directory

> ./docker.local/bin/build_txns_generator.sh

4.2) To run the miner_stress program after starting the 3 miners

> ./docker.local/bin/generate_txns.sh num-txns

If num-txns is not specified, then 25000 transactions are generated for each miner

5) Troubleshooting:

5.1) Ensure the port mapping is all correct

> docker ps

This should display a few containers and should include containers with images miner1_miner, miner2_miner and miner3_miner and they should have the ports mapped like "0.0.0.0:7071->7071/tcp"

5.2) Confirming the servers are up and running. From a browser, visit

http://localhost:7071/

http://localhost:7072/

http://localhost:7073/

to see the status of the miners.

5.3) Connecting to redis servers running within the containers (you are within the appropriate miner directories)

Default redis (used for clients and state):

> ../bin/run.sh redis redis-cli

Redis used for transactions:

> ../bin/run.sh redis_txns redis-cli

6) Miscellaneous

Cleanup

6.1) Get rid of old unused docker resources :

> docker system prune
