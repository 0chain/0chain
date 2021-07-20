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

    cd  ./data/sharder$i


    find ./config -name "0chain.yaml" -exec sed -i '' 's/level: "debug"/level: "error"/g' {} \;
    find ./config -name "0chain.yaml" -exec sed -i '' "s/console: false/console: true/g" {} \;
    find ./config -name "0chain.yaml" -exec sed -i '' "s/#    host: cassandra/    host: 127.0.0.1/g" {} \;
    find ./config -name "0chain.yaml" -exec sed -i '' "s/#    port: 9042/    port: 904$i/g" {} \;


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
    find . -name "go.mod" -exec sed -i '' "/replace github.com\/valyala\/gozstd/d" {} \;
    echo "replace github.com/valyala/gozstd => ../../../../../valyala/gozstd" >> ./go.mod


    cd ./sharder/sharder

    # Build bls with CGO_LDFLAGS and CGO_CPPFLAGS to fix `ld: library not found for -lcrypto`
    export CGO_LDFLAGS="-L/usr/local/opt/openssl@1.1/lib"
    export CGO_CPPFLAGS="-I/usr/local/opt/openssl@1.1/include"

    GIT_COMMIT=$GIT_COMMIT
    go build -o $root/data/sharder$i/sharder -v -tags bn256 -gcflags "all=-N -l" -ldflags "-X 0chain.net/core/build.BuildTag=$GIT_COMMIT" 

    cd $root/data/sharder$i/
    ./sharder --deployment_mode 0 --keys_file $root/data/sharder$i/config/b0snode${i}_keys.txt --minio_file $root/data/sharder$i/config/minio_config.txt --work_dir $root/data/sharder$i
}
