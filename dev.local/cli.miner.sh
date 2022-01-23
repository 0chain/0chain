# clean all thing for current miner
clean_miner () {
    echo "clean miner $i"

    cd $root

    rm -rf "./data/miner$i"
}

# mkdir,copy and update config for current miner
setup_miner_runtime() {
    echo ""
    echo "Prepare miner $i: config, files, data, log .."
    cd $root
    [ -d ./data/miner$i ] || mkdir -p ./data/miner$i

    [ -d ./data/miner$i/config ] && rm -rf ./data/miner$i/config

    cp -r ../docker.local/config "./data/miner$i/"

    cd  ./data/miner$i

    find ./config -name "0chain.yaml" -exec sed -i '' 's/level: "debug"/level: "error"/g' {} \;
    find ./config -name "0chain.yaml" -exec sed -i '' "s/console: false/console: true/g" {} \;
    find ./config -name "0chain.yaml" -exec sed -i '' "s/#    host: cassandra/    host: 127.0.0.1/g" {} \;
    find ./config -name "0chain.yaml" -exec sed -i '' "s/#    port: 9042/    port: 904$i/g" {} \;
    find ./config -name "0chain.yaml" -exec sed -i '' 's/threshold_by_count: 66/threshold_by_count: 40/g' {} \;


    find ./config -name "b0magicBlock_4_miners_1_sharder.json" -exec sed -i '' 's/198.18.0.71/127.0.0.1/g' {} \;
    find ./config -name "b0magicBlock_4_miners_1_sharder.json" -exec sed -i '' "s/198.18.0.72/127.0.0.1/g" {} \;
    find ./config -name "b0magicBlock_4_miners_1_sharder.json" -exec sed -i '' "s/198.18.0.73/127.0.0.1/g" {} \;
    find ./config -name "b0magicBlock_4_miners_1_sharder.json" -exec sed -i '' "s/198.18.0.74/127.0.0.1/g" {} \;
    find ./config -name "b0magicBlock_4_miners_1_sharder.json" -exec sed -i '' "s/198.18.0.81/127.0.0.1/g" {} \;
    find ./config -name "b0magicBlock_4_miners_1_sharder.json" -exec sed -i '' "s/198.18.0.82/127.0.0.1/g" {} \;
    

    [ -d ./data/rocksdb/state/dkg ] || mkdir -p ./data/rocksdb/state/dkg
    [ -d ./data/rocksdb/mb ] || mkdir -p ./data/rocksdb/mb
    
    [ -d ./log ] || mkdir ./log
    [ -d ./tmp ] || mkdir ./tmp

   

    cd $root
}

# build and start miner
start_miner(){

    cd $code

    # Build libzstd with local repo
    # FIXME: Change this after https://github.com/valyala/gozstd/issues/6 is fixed.
    # find . -name "go.mod" -exec sed -i '' "/replace github.com\/valyala\/gozstd/d" {} \;
    # echo "replace github.com/valyala/gozstd => ../../../../../valyala/gozstd" >> ./go.mod


    cd ./miner/miner

    # Build bls with CGO_LDFLAGS and CGO_CPPFLAGS to fix `ld: library not found for -lcrypto`
    export CGO_LDFLAGS="-L/usr/local/opt/openssl@1.1/lib"
    export CGO_CPPFLAGS="-I/usr/local/opt/openssl@1.1/include"

    GIT_COMMIT="cli"
    go build -o $root/data/miner$i/miner -v -tags "bn256 development dev" -gcflags "all=-N -l" -ldflags "-X 0chain.net/core/build.BuildTag=$GIT_COMMIT" 


    keys_file=$root/data/miner$i/config/b0mnode${i}_keys.txt
    dkg_file=$root/data/miner$i/config/b0mnode${i}_dkg.json
    redis_port=63${i}0
    redis_txns_port=63${i}1

    echo $keys_file
    echo $dkg_file
    echo $redis_port
    echo $redis_txns_port

    cd $root/data/miner$i/
    ./miner --deployment_mode 0 --keys_file $keys_file --dkg_file $dkg_file --work_dir $root/data/miner$i --redis_host 127.0.0.1 --redis_port $redis_port --redis_txns_host 127.0.0.1 --redis_txns_port $redis_txns_port
}