# 0chain on EC2 instance over https


## Prerequisite

- Ec2 instance with docker installed

## Initial Setup

### Directory Setup for Miners & Sharders

In the git/0chain run the following command

```
$ ./docker.local/bin/init.setup.sh
```

### Setup Network

Setup a network called testnet0 for each of these node containers to talk to each other.

```
$ ./docker.local/bin/setup_network.sh
```
## Modify Config and Keys files

1. Update `network . dns_url` in `./docker.local/config/0chain.yaml` to point to `https://<network_url>/dns`

2. Edit `docker.local/config/b0snode2_keys.txt` and replace `localhost` and `198.18.0.83` with your domain.

3. Edit `docker.local/config/b0mnode5_keys.txt` and replace `localhost` and `198.18.0.83` with your domain.

## Building the Nodes

1. Open 2 terminal tabs.

1.1) First build the base containers, zchain_build_base and zchain_run_base

```
$ ./docker.local/bin/build.base.sh
```

2. Building the miners and sharders. From the git/0chain directory use

2.1) To build the miner containers

```
$ ./docker.local/bin/build.miners.sh
```

2.2) To build the sharder containers

```
$ ./docker.local/bin/build.sharders.sh
```

2.3) Syncing time (the host and the containers are being offset by a few seconds that throws validation errors as we accept transactions that are within 5 seconds of creation). This step is needed periodically when you see the validation error.

```
$ ./docker.local/bin/sync_clock.sh

```

## Starting the nodes

1. To start sharder container `cd docker.local/sharder2`

```
$ ../bin/start.b0sharder.sh
```

Wait till the cassandra is started and the sharder is ready to listen to requests.

2. To start sharder container `cd docker.local/miner5` in other terminal.


```
$ ../bin/start.b0miner.sh
```



## Configuring Https

1. Go to https directory in 0chain repo.
```
cd /0chain/https
```

2. Edit docker-compose.yml and replace <your_email>, <your_domain> with your email and domain. Make sure to add route53 A type record for your domain and ip address


3. Deploy nginx and certbot using the following command
```
docker-compose up -d
```

4. Check certbot logs and see if certificate is generated. You will find "Congratulations! Your certificate and chain have been saved at: /etc/letsencrypt/live/<your_domain>/fullchain.pem" in the logs if the certificate is generated properly.

```
docker logs -f https_certbot_1 
```

4. Edit /conf.d/nginx.conf to uncomment required locations in config for port 80. Uncomment all lines in server config for port 443 and comment locations which are not required. You can add more locations if required.

5. Restart docker compose and you will be able to access 0chain over https.

```
docker-compose restart
```
