# clean all thing for current sharder
clean_sharder () {
    echo "clean sharder $i"

    cd $root

    rm -rf "./data/sharder$i"
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


    find ./config -name "b0magicBlock_4_miners_2_sharders.json" -exec sed -i '' 's/198.18.0.71/127.0.0.1/g' {} \;
    find ./config -name "b0magicBlock_4_miners_2_sharders.json" -exec sed -i '' "s/198.18.0.72/127.0.0.1/g" {} \;
    find ./config -name "b0magicBlock_4_miners_2_sharders.json" -exec sed -i '' "s/198.18.0.73/127.0.0.1/g" {} \;
    find ./config -name "b0magicBlock_4_miners_2_sharders.json" -exec sed -i '' "s/198.18.0.74/127.0.0.1/g" {} \;
    find ./config -name "b0magicBlock_4_miners_2_sharders.json" -exec sed -i '' "s/198.18.0.81/127.0.0.1/g" {} \;
    find ./config -name "b0magicBlock_4_miners_2_sharders.json" -exec sed -i '' "s/198.18.0.82/127.0.0.1/g" {} \;
    

    

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

    export LIBRARY_PATH="/usr/local/lib"
    export LD_LIBRARY_PATH="/usr/local/lib:/usr/local/opt/openssl@1.1/lib"
    export CGO_LDFLAGS="-L/usr/local/opt/openssl@1.1/lib -lrocksdb -lstdc++ -lm -lz -lbz2 -lsnappy -llz4"
    export CGO_CFLAGS="-I/usr/local/opt/openssl@1.1/include -I/usr/local/include"
    export CGO_CPPFLAGS="-I/usr/local/opt/openssl@1.1/include -I/usr/local/include"

    GIT_COMMIT="cli"
    go build -mod mod -o $root/data/sharder$i/sharder -v -tags bn256 -gcflags "all=-N -l" -ldflags "-X 0chain.net/core/build.BuildTag=$GIT_COMMIT" 

    cd $root/data/sharder$i/
    keys_file=$root/data/sharder$i/config/b0snode${i}_keys.txt
    minio_file=$root/data/sharder$i/config/minio_config.txt
    echo $keys_file


    ./sharder --deployment_mode 0 --keys_file $keys_file --minio_file $minio_file --work_dir $root/data/sharder$i
}