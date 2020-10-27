Deploy via SSH
==============

Deploy 3 sharders and 8 miners via SSH to a server (single instance
only supported for now).

# Setup remote server.

Connect to SSH server and add the user to docker group. Setup remote Ubunut
server, for example (use your ssh command):

```
sh docker.local/deploy-ssh/remote-setup-ubuntu.sh 'ssh user@host'
```

# Upload or update uploaded images.

+ Build miners and sharders as usual.
+ Upload images via SSH.

```
sh docker.local/deploy-ssh/deploy-ssh-images.sh 'ssh user@host'
```

The 'ssh user@host' is your SSH command to connect to the server.

# Minimal services

Deploy and expand minimal 0chain to the server (configs and scripts only). The
'address' is external address of the server. The external address will be used
by 0chain nodes as 'host'. E.g. it can be DNS-name or IP address. It can be
'localhost', this way nodes will communicate through loopback.

```
./docker.local/bin/deploy-ssh.sh 'ssh user@host' 'address'
```

# Use on the server.

```
sudo systemctl start sharder1 # 2, 3
sudo systemctl start miner1 # 2, 3, 4, 5, 6, 7, 8
```

And the same with 'stop/status/enable/disable'.

Also, `systemctl list-units {sharder,miner}*.service` to list.

The `deploy-ssh.sh` command stops all remote nodes and cleans up remote BC.
Then it uploads or updates minimal 0chain with configs. It never starts a unit.
And it never updates docker images, use `deploy-ssh-images.sh` to update miners
and sharders images on remote host.
