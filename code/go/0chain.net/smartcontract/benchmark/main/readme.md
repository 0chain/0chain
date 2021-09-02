WIP 

Benchmark 0chain smart-contract endpoints.

To run
```bash
go build -tags bn256
./main benchmark | column -t -s,
```

To run only a subset of the test suits
```bash
go build -tags bn256
./main benchmark benchmark --tests "miner, storage" | column -t -s,
```

To only print out the comma delimited data use teh `-verbose false` flat
```bash
go build -tags bn256
./main benchmark  --verbose false | column -t -s,
```

Setup parameters in [benchmark.yaml](https://github.com/0chain/0chain/blob/bench-sc/code/go/0chain.net/smartcontract/benchmark/main/config/benchmark.yaml).