Benchmark 0chain smart-contract endpoints.

Runs [testing.Benchmark](https://pkg.go.dev/testing#Benchmark) on each 0chain endpoint. 
The blockchain database used in these tests is constructed from the parameters in the
[benchmark.yaml](https://github.com/0chain/0chain/blob/staging/code/go/0chain.net/smartcontract/benchmark/main/config/benchmark.yaml).
file. Smartcontracts do not (or should not) access tha chain so a populated 
MPT database is enough to give a realistic benchmark.

To run
```bash
go build -tags bn256 && ./main benchmark | column -t -s,
```

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
