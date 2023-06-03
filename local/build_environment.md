# Build environment for 0Chain go executables

## Table of contents

- [Introduction](#introduction)
- [Install rocksdb](#install-rocksdb)
- [Install Herumi's cryptography](#install-herumis-cryptography)
- [Build libzstd](#build-libzstd)
- [Build miner](#build-miner)

## Introduction

Assume that you have just run
```shell
git clone https://github.com/0chain
```
check to see if you can compile an 0Chain miner
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
# github.com/linxGnu/grocksdb
cgo: exec gcc: exec: "gcc": executable file not found in $PATH
````
No problem we will go through building herumi, valyala and grocksdb. I will
be assuming your using Ubuntu but Mac works much the same replacing `apt-get` with `brew`.

You can also work it our for yourself, all the details are in the docker files
[build_base](https://github.com/0chain/0chain/blob/master/docker.local/build.base/Dockerfile.build_base)
and [build miner](https://github.com/0chain/0chain/blob/master/docker.local/build.miner/Dockerfile).

## Install rocksdb

You probably already have `make` and `g++` installed, but if not you want
```shell
sudo apt update
sudo apt install -y make
sudo apt install -y build-essential
```
Now install the libraries for RocksDB.
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
That is the preliminaries out the way. Now install RocksDB
```shell
cd ~/Downloads
wget https://github.com/facebook/rocksdb/archive/v6.15.5.tar.gz
tar -xf v6.15.5.tar.gz
cd rocksdb-6.15.5
make OPT=-g0 USE_RTTI=1
sudo make install
```

## Install Herumis cryptography

As before we need to install some libraries first.
```shell
sudo apt-get update -y
sudo apt-get install -y libgmp-dev
sudo apt-get install libssl-dev
```
> Mac: Clang has problems linking to Version 1.1 of openssl. If you have version 1.1 then its recommended
> you downgrade to version 1.0 or upgrade to version 1.1.1j or higher.

```shell
wget -O - https://github.com/herumi/mcl/archive/master.tar.gz | tar xz
mv mcl* mcl
cd mcl
make -j $(nproc) lib/libmclbn256.so
sudo make install
sudo cp lib/libmclbn256.so /usr/local/lib
```
```shell
wget -O - https://github.com/herumi/bls/archive/master.tar.gz | tar xz
mv bls* bls
cd bls
make
sudo make install
```
## Build libzstd

From [Dockerfile](https://github.com/0chain/0chain/blob/master/docker.local/build.miner/Dockerfile);
As https://github.com/valyala/gozstd/issues/6 is still open we have to build `libzstd` as follows.
Do this even if you already have `libzstd` installed.
```shell
cd $HOME/go/pkg/mod/github.com/valyala/gozstd*
chmod -R +w . &&
make clean libzstd.a
```

## Build miner

Now the big test. Run
```shell
cd code/go/0chain.net/miner/miner
go build -tags "bn256 development"
```
If all is well `go build` should work, and you will have a new `miner` executable.
Alternately the result of mistakes or shortcuts are likely to turn up here as errors.
```shell
/usr/bin/ld: /usr/local/lib/librocksdb.a(env_posix.o): in function `rocksdb::(anonymous namespace)::PosixDynamicLibrary::~PosixDynamicLibrary()':
env_posix.cc:(.text+0xf0): undefined reference to `dlclose'

```
Suggests a linker error, probably a problem with your RocksDB and gcc versions. Check you installed
`RocksDB 5.18.3`.
```shell
/usr/bin/ld: /usr/local/lib/librocksdb.a(format.o): in function `rocksdb::LZ4_Uncompress(rocksdb::UncompressionContext const&, char const*, unsigned long, int*, unsigned int, rocksdb::MemoryAllocator*)':
format.cc:(.text._ZN7rocksdb14LZ4_UncompressERKNS_20UncompressionContextEPKcmPijPNS_15MemoryAllocatorE[_ZN7rocksdb14LZ4_UncompressERKNS_20UncompressionContextEPKcmPijPNS_15MemoryAllocatorE]+0xd5): undefined reference to `LZ4_createStreamDecode'
/usr/bin/ld: format.cc:(.text._ZN7rocksdb14LZ4_UncompressERKNS_20UncompressionContextEPKcmPijPNS_15MemoryAllocatorE[_ZN7rocksdb14LZ4_UncompressERKNS_20UncompressionContextEPKcmPijPNS_15MemoryAllocatorE]+0x135): undefined reference to `LZ4_setStreamDecode'
```
If you get stuck there is always the option of copying the librocksdb libraries from
another computer where the 0chain builds are know to work.
The libraries can be put in `usr/local/lib`. You will probably want to make sym links so you
have a `librocksdb.so.5.18.3`, `librockssdb.5.18`, `librocksdb.so.5` and `librocksdb.so`.
```shell
