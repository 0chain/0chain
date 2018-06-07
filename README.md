
TestNet Setup with Docker Containers:

1) Directory setup. In the git/0chain/docker.local directory create 3 directories called miner1, miner2 and miner3. Within these directories create data/db/redis directory using

> mkdir -p data/db/redis


2) Docker commands for Miners

*) Create a network called testnet0 where all the nodes have an IP address so they can talk to each other.
   Note: The config file should be providing the IP address of the nodes as per the IP addresses in this network.

> docker network create --driver=bridge --subnet=198.18.0.0/15 --gateway=198.18.0.255 testnet0

*) From the working directory of git/0chain, issue the following commands. Build by removing intermediate containers

> export MINER=1; docker-compose -p miner1 -f docker.local/build.miner/docker-compose.yml  build --force-rm

> export MINER=2; docker-compose -p miner2 -f docker.local/build.miner/docker-compose.yml  build --force-rm

> export MINER=3; docker-compose -p miner3 -f docker.local/build.miner/docker-compose.yml  build --force-rm

*) Syncing time (the host and the containers are being offset by a few seconds that throws validation errors as we accept transactions that are within 5 seconds of creation). This step is needed periodically when you see the validation error.

> docker run --rm --privileged alpine hwclock -s

*)Open 3 terminals and go to the directory miner1 , 2 and 3 respectively that were created under git/0chain/docker.local. From there issue the 3 commands one on each terminal respectively.

> export MINER=1; docker-compose -p miner1 -f ../build.miner/docker-compose.yml up

> export MINER=2; docker-compose -p miner2 -f ../build.miner/docker-compose.yml up

> export MINER=3; docker-compose -p miner3 -f ../build.miner/docker-compose.yml up


3) Troubleshooting:

*) Ensure the port mapping is all correct

> docker ps

This should display a few containers and should include containers with images miner1_miner, miner2_miner and miner3_miner and they should have the ports mapped like "0.0.0.0:7071->7071/tcp"

*) Confirming the servers are up and running. From a browser, visit

http://localhost:7071/

http://localhost:7072/

http://localhost:7073/

to see the status of the servers.


*) Connecting to redis servers running within the containers (you are within git/0chain directory)

Default redis (used for clients and state):

> export MINER=1; docker-compose -p miner1 -f docker.local/build.miner/docker-compose.yml exec redis redis-cli

> export MINER=2; docker-compose -p miner2 -f docker.local/build.miner/docker-compose.yml exec redis redis-cli

> export MINER=3; docker-compose -p miner3 -f docker.local/build.miner/docker-compose.yml exec redis redis-cli



Redis used for transactions:

> export MINER=1; docker-compose -p miner1 -f docker.local/build.miner/docker-compose.yml exec redis_txns redis-cli

> export MINER=2; docker-compose -p miner2 -f docker.local/build.miner/docker-compose.yml exec redis_txns redis-cli

> export MINER=3; docker-compose -p miner3 -f docker.local/build.miner/docker-compose.yml exec redis_txns redis-cli

4) Docker Commands for Sharder (single sharder for now in the local test)

4.1) Build: from git/0chain, issue the following command to build the containers

> export SHARDER=1; docker-compose -p sharder1 -f docker.local/build.sharder/docker-compose.yml  build --force-rm

4.2) Start the sharder: from git/0chain/docker.local/sharder1 directory, issue the following command

> export SHARDER=1; docker-compose -p sharder1 -f ../build.sharder/docker-compose.yml up

5) Miscellaneous

Cleanup

*) Get rid of old unused docker resources :

> docker system prune
