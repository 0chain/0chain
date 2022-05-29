Benchmark 0chain smart-contract endpoints.

Runs [testing.Benchmark](https://pkg.go.dev/testing#Benchmark) on each 0chain endpoint. 
The blockchain database used in these tests is constructed from the parameters in the
[benchmark.yaml](https://github.com/0chain/0chain/blob/staging/code/go/0chain.net/smartcontract/benchmark/main/config/benchmark.yaml).
file. Smartcontracts do not (or should not) access tha chain so a populated 
MPT database is enough to give a realistic benchmark.

## To run
### DOCKER
1. run init
```shell
./docker.local/bin/init.setup.sh
```
2. build base image
```shell
./docker.local/bin/build.base.sh
```
3. build docker image
```shell
./docker.local/bin/build.benchmark.sh
```
4. change dir to benchmarks
```shell
cd docker.local/benchmarks
```
5. run tests
```shell
../bin/start.benchmarks.sh
```

Script can be run with different options:
- load 
- tests
- config
- verbose
- omit

### BARE METAL
```bash
go build -tags bn256 && ./main benchmark | column -t -s,
```

It can take a long time to generate a MPT for the simulation. To help with this 
it is possible to save a MPT for use later, set the options.save_path key in
[benchmark.yaml](https://github.com/0chain/0chain/blob/staging/code/go/0chain.net/smartcontract/benchmark/main/config/benchmark.yaml).
```yaml
options:
  save_path: ./saved_data
```
You can now reuse this database using the load option in the command line
```bash
go build -tags bn256 && ./main benchmark --load saved_data  | column -t -s,
```

-

To run only a subset of the test suits
```bash
go build -tags bn256
./main benchmark benchmark --tests "miner, storage" | column -t -s,
```

To only print out the comma delimited data without any trace outputs, use the `--verbose=false` flag
```bash
go build -tags bn256
./main benchmark  --verbose=false | column -t -s,
```

To filter out test from the benchmark use the `-ommit` option,
and enter them in a comma delimited list.
```bash
go build -tags bn256
./main benchmark --omit "storage_rest.allocation, storage_rest.allocations" | column -t -s,
```

To use the event database you need a to set up a local postgreSQL database. Login in parameters
are read from the benchmark yaml, dbs.events section.
- MacOS
  1. brew install postgres
  2. initdb /usr/local/var/postgres
  3. pg_ctl -D /usr/local/var/postgres start
  4. /usr/local/opt/postgres/bin/createuser -s postgres

Create zchain_user
```sql
CREATE ROLE zchain_user WITH
LOGIN
NOSUPERUSER
NOCREATEDB
NOCREATEROLE
INHERIT
NOREPLICATION
CONNECTION LIMIT -1
PASSWORD 'zchian';
```

Create events_ds database
```sql
CREATE DATABASE events_db
WITH
OWNER = zchain_user
ENCODING = 'UTF8'
CONNECTION LIMIT = -1;
```
Add connectivity details to the config 
```yaml
dbs:
  events:
    enabled: true
    name: events_db
    user: zchain_user
    password: zchian
    host: localhost
    port: 5432
    max_idle_conns: 100
    max_open_conns: 200
    conn_max_lifetime: 20s
```
Set enabled to false if you have not setup a postgreSQL database. Some of the Rest Api
endpoint will not work without an event database.

You can also set all these options in the
[benchmark.yaml](https://github.com/0chain/0chain/blob/staging/code/go/0chain.net/smartcontract/benchmark/main/config/benchmark.yaml).
file. The command line options will take precedence over those in the `.yaml` file.

The benchmark results are unlikely to be false positives but could  be false negatives, 
if benchmark parameters are such that a particularly long running block of code 
is accidentally skipped.

The output results are coloured, red > `50ms`, purple `>10ms`, yellow >`1ms` 
otherwise green. To turn off, set colour=false in
[benchmark.yaml](https://github.com/0chain/0chain/blob/staging/code/go/0chain.net/smartcontract/benchmark/main/config/benchmark.yaml).
or use `--verbose=false`.

For best results try to choose parameters so that benchmark timings are below a second.
