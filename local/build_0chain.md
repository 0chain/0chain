# Build 0Chain go executables

## Table of contents

- [Introduction](#introduction)
- [Install rocksdb](#install-rocksdb)

### Introduction

Assume that you have just run
```shell
git clone https://github.com/0chain
```
check to see if you can compile a 0Chain miner
```shell
cd 0chain/code/go/0chain.net/miner/miner
go build -tags "bn256 development"
```
If this gives no errors your probably good to go. However, be aware that rocksdb's 
compiler will configure build options by inspecting installed libraries. So 
for instance if lz4 is not installed you likely have the wrong version of rocksdb, 
which will come apparent when you run your miner.

More likely you have a response similar to
```shell
# github.com/herumi/bls/ffi/go/bls
cgo: exec gcc: exec: "gcc": executable file not found in $PATH
# github.com/valyala/gozstd
cgo: exec gcc: exec: "gcc": executable file not found in $PATH
# github.com/0chain/gorocksdb
cgo: exec gcc: exec: "gcc": executable file not found in $PATH
````
No problem we will go through building herumi, valyala and gorocksdb. I will 
be assuming your using `ubuntu` but Mac works much the same replacing `apt-get` with `brew`.  

You can also work it our for yourself, all the details are in the docker files
[build_base](https://github.com/0chain/0chain/blob/master/docker.local/build.base/Dockerfile.build_base)
and [build miner](https://github.com/0chain/0chain/blob/master/docker.local/build.miner/Dockerfile).

### Install rocksdb

You probably already have make and g++ installed, but if not you want
```shell
sudo apt install -y make
sudo apt install -y build-essential
```
Now install the required libraries.
```shell
sudo apt-get update -y
sudo apt-get install -y coreutils
```
If you are using a Mac you probably won't want linux-headers.
```shell
sudo apt install linux-headers-$(uname -r)
sudo apt install -y zlib1g-dev
sudo apt-get install -y bzip2
sudo apt-get install liblz4-dev
sudo apt-get install -y libsnappy-dev
sudo apt-get install -y zstd
sudo apt-get install -y libbz2-dev

```
That is the prelimaries out the way. Now install RocksDB. The well
tested docker file wants to install an old version of RocksDB, 
so we will do that.
```shell
cd ~/Downloads
wget https://github.com/facebook/rocksdb/archive/v5.18.3.tar.gz
tar -xf v5.18.3.tar.gz
cd rocksdb-5.18.3
make OPT=-g0 USE_RTTI=1
sudo make install
```
If the `make OPT=-g0 USE_RTTI=1` command fails with 
```shell
...
./db/version_edit.h:86:8: error: implicitly-declared ‘constexpr rocksdb::FileDescriptor::FileDescriptor(const rocksdb::FileDescriptor&)’ is deprecated [-Werror=deprecated-copy]
...
cc1plus: all warnings being treated as errors
make: *** [Makefile:1958: db/builder.o] Error 1
```
then your gcc version is too high. Probably the best thing to do here is to downgrade your gcc.
```shell
sudo apt install g++-7 gcc-7
export CC=/usr/bin/gcc-7
export CXX=/usr/bin/g++-7
make OPT=-g0 USE_RTTI=1
sudo make install
```

### Install Herumi's cryptography

Ad before we need to install some libraries first.
```shell
sudo apt-get update -y
sudo apt-get install -y libgmp-dev
sudo apt-get install libssl-dev
```
> Mac: Clang has problems linking to Version 1.1 of openssl. If you have version 1.1 then its recommended 
> you downgrade to version 1.0 or upgrade to version 1.1.1j or higher.

```shell
wget https://github.com/herumi/mcl/archive/v0.98.tar.gz
tar -xf v0.98.tar.gz
mv mcl* mcl
wget https://github.com/herumi/bls/archive/2e9e496ad85e74ecaee91559e2dcf95ba571382d.tar.gz 
tar -xf 2e9e496ad85e74ecaee91559e2dcf95ba571382d.tar.gz
mv bls* bls 
cd mcl
make -j $(nproc) lib/libmclbn256.so 
sudo make install
sudo cp lib/libmclbn256.so /usr/local/lib 
cd ../bls
make 
sudo make install
```
### Build libzstd

From [docker file](https://github.com/0chain/0chain/blob/master/docker.local/build.miner/Dockerfile);
As https://github.com/valyala/gozstd/issues/6 is still open we have to build libzstd as follows.
```shell
cd $HOME/go/pkg/mod/github.com/valyala/gozstd* 
chmod -R +w . && 
make clean libzstd.a
```
