# clean all thing for current sharder
clean_sharder () {
    echo "clean sharder $i"

    cd $root

    [ -d "./data/sharder$i/data/blocks" ] && rm -rf "./data/sharder$i/data/blocks" && mkdir -p "./data/sharder$i/data/blocks" && echo " > clean blocks"
    [ -d "./data/sharder$i/data/rocksdb" ] && rm -rf "./data/sharder$i/data/rocksdb" && mkdir -p "./data/sharder$i/data/rocksdb" && echo " > clean rocksdb"
    [ -d "./data/sharder$i/log" ] && rm -rf "./data/sharder$i/log" && mkdir -p "./data/sharder$i/log" && echo " > clean logs"
    [ -d "./data/sharder$i/tmp" ] && rm -rf "./data/sharder$i/tmp" && mkdir -p "./data/sharder$i/tmp" && echo " > clean tmp"
    [ -d "./data/sharder$i/postgres" ] && rm -rf "./data/sharder$i/postgres"  && mkdir -p "./data/sharder$i/postgres" && \
    [ -d "./data/sharder$i/cassandra" ] && rm -rf "./data/sharder$i/cassandra" && mkdir -p "./data/sharder$i/cassandra" && \
    ./cli.sharder.db.sh $i  && echo " > clean [postgres, cassandra]"
}

# mkdir,copy and update config for current sharder
setup_sharder_runtime() {
    echo ""
    echo "Prepare sharder $i: config, files, data, log .."
    cd $root
    [ -d ./data/sharder$i ] || mkdir -p ./data/sharder$i

    cp -r ../docker.local/config "./data/sharder$i/"
    cp -r ../docker.local/sql_script "./data/sharder$i/"

    cd  ./data/sharder$i

    find ./config -name "0chain.yaml" -exec sed -i '' 's/level: "debug"/level: "error"/g' {} \;
    find ./config -name "0chain.yaml" -exec sed -i '' "s/console: false/console: true/g" {} \;
    find ./config -name "0chain.yaml" -exec sed -i '' "s/#   host: cassandra/    host: 127.0.0.1/g" {} \;
    find ./config -name "0chain.yaml" -exec sed -i '' "s/#   port: 9042/    port: 904$i/g" {} \;
    find ./config -name "0chain.yaml" -exec sed -i '' 's/threshold_by_count: 66/threshold_by_count: 40/g' {} \;
    find ./config -name "0chain.yaml" -exec sed -i '' "s/    host: postgres/    host: 127.0.0.1/g" {} \;
    find ./config -name "0chain.yaml" -exec sed -i '' "s/    port: 5432/    port: 553$i/g" {} \;
    
    find ./config -name "*.json" -exec sed -i '' "s/198.18.0.71/127.0.0.1/g" {} \;
    find ./config -name "*.json" -exec sed -i '' "s/198.18.0.72/127.0.0.1/g" {} \;
    find ./config -name "*.json" -exec sed -i '' "s/198.18.0.73/127.0.0.1/g" {} \;
    find ./config -name "*.json" -exec sed -i '' "s/198.18.0.74/127.0.0.1/g" {} \;
    find ./config -name "*.json" -exec sed -i '' "s/198.18.0.81/127.0.0.1/g" {} \;
    find ./config -name "*.json" -exec sed -i '' "s/198.18.0.82/127.0.0.1/g" {} \;
    

    [ -d ./data/blocks ] || mkdir -p ./data/blocks
    [ -d ./data/rocksdb ] || mkdir -p ./data/rocksdb
    [ -d ./log ] || mkdir ./log
    [ -d ./tmp ] || mkdir ./tmp

    cd $root
}

# build and start sharder
start_sharder(){

    cd $code

    # Build libzstd with local repo
    # FIXME: Change this after https://github.com/valyala/gozstd/issues/6 is fixed.
    # find . -name "go.mod" -exec sed -i '' "/replace github.com\/valyala\/gozstd/d" {} \;
    # echo "replace github.com/valyala/gozstd => ../../../../../valyala/gozstd" >> ./go.mod


    cd ./sharder/sharder

    snappy=$(brew --prefix snappy)
    lz4=$(brew --prefix lz4)
    gmp=$(brew --prefix gmp)
    openssl=$(brew --prefix openssl@1.1)

    export LIBRARY_PATH="/usr/local/lib:${openssl}/lib:${snappy}/lib:${lz4}/lib:${gmp}/lib"
    export LD_LIBRARY_PATH="/usr/local/lib:${openssl}/lib:${snappy}/lib:${lz4}/lib:${gmp}/lib"
    export DYLD_LIBRARY_PATH="/usr/local/lib:${openssl}/lib:${snappy}/lib:${lz4}/lib:${gmp}/lib"
    export CGO_LDFLAGS="-L/usr/local/lib -lrocksdb -lstdc++ -lm -lz -lbz2 -lsnappy -llz4"
    export CGO_CFLAGS="-I/usr/local/include"
    export CGO_CPPFLAGS="-I/usr/local/include"
    export LDFLAGS="-L/usr/local/lib -lrocksdb -lstdc++ -lm -lz -lbz2 -lsnappy -llz4"
    export CFLAGS="-I/usr/local/include"
    export CPPFLAGS="-I/usr/local/include"

    GIT_COMMIT="cli"
    go build -mod mod -o $root/data/sharder$i/sharder -v -tags "bn256 development dev" -gcflags "all=-N -l" -ldflags "-X 0chain.net/core/build.BuildTag=$GIT_COMMIT" 

    cd $root/data/sharder$i/
    keys_file=$root/data/sharder$i/config/b0snode${i}_keys.txt
    minio_file=$root/data/sharder$i/config/minio_config.txt

    ./sharder --deployment_mode 0 --keys_file $keys_file --work_dir $root/data/sharder$i
}
