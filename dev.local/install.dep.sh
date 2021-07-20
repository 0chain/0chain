#!/bin/bash

set -e

root=$(pwd)

cd ../../../

github=$(pwd)

export PATH="/usr/local/opt/openssl@1.1/bin:$PATH"
export CGO_LDFLAGS="-L/usr/local/opt/openssl@1.1/lib"
export CGO_CPPFLAGS="-I/usr/local/opt/openssl@1.1/include"
export PKG_CONFIG_PATH="/usr/local/opt/openssl@1.1/lib/pkgconfig"
export OPENSSL_ROOT_DIR="/usr/local/opt/openssl@1.1"
export OPENSSL_CRYPTO_LIBRARY="/usr/local/opt/openssl@1.1/lib"
export OPENSSL_LIBRARIES="/usr/local/opt/openssl@1.1/lib"
export OPENSSL_INCLUDE_DIR="/usr/local/opt/openssl@1.1/include"

echo "==============================================="
echo "[1/2] install valyala/gozstd..."
echo "==============================================="
cd $github
[ -d valyala ] || mkdir valyala 
cd valyala 
[ -d gozstd ] || git clone https://github.com/valyala/gozstd.git 
cd $github/valyala/gozstd 
git checkout v1.5.0 

make && make clean libzstd.a

echo "==============================================="
echo "[2/2] install facebook/rocksdb..."
echo "==============================================="

cd $github
[ -d facebook ] || mkdir facebook 
cd facebook 
[ -d rocksdb ] || git clone https://github.com/facebook/rocksdb.git 
cd $github/facebook/rocksdb 
git checkout v5.18.3
make install

cd $root

