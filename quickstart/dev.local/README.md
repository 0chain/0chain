# 0Chain Developer Guide
This is a guide for configuring your local machine for 0chain development.  


## build docker images

please checkout 0chain:fix/developer_guide and 0dns:fix/developer_guide first

```
./docker.build.sh

```

## start docker container
- start 0dns
```
./docker.install.sh
```
- start 1 sharder
```
./docker.install.sharder.sh
```
- start 2 miners
```
./docker.install.miners.sh
```


## start debugger in vscode
update settings in your `0chain/code/go/0chain.net/.vscode/launch.json`

```
{
    // Use IntelliSense to learn about possible attributes.
    // Hover to view descriptions of existing attributes.
    // For more information, visit: https://go.microsoft.com/fwlink/?linkid=830387
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Debug:sharder1",
            "type": "go",
            "request": "attach",
            "mode": "remote",
            "remotePath": "/0chain/go/0chain.net/",
            "port": 2341,
            "host": "127.0.0.1",
            "showLog": true
        },

    ]
}
```


