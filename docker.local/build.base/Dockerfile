FROM iron/go:dev
RUN apk add --update --no-cache build-base linux-headers git cmake bash perl
RUN apk add --update --no-cache zlib zlib-dev bzip2 bzip2-dev snappy snappy-dev lz4 lz4-dev jemalloc jemalloc-dev

# Install golang dep
RUN curl -fsSL -o /usr/local/bin/dep https://github.com/golang/dep/releases/download/v0.5.0/dep-linux-amd64 && \
    chmod +x /usr/local/bin/dep

# Install latest gflags
RUN cd /tmp && \
    curl -L -o gflags.tar.gz https://github.com/gflags/gflags/archive/v2.2.1.tar.gz && \
    tar xf gflags.tar.gz && \
    cd gflags-* && \
    mkdir build && \
    cd build && \
    cmake -DBUILD_SHARED_LIBS=1 -DGFLAGS_INSTALL_SHARED_LIBS=1 .. && \
    make install -j $(nproc) && \
    cd /tmp && \
    rm -R /tmp/gflags*

# Install RocksDB
RUN cd /tmp && \
    curl -L -o rocksdb.tar.gz https://github.com/facebook/rocksdb/archive/5.13.fb.tar.gz && \
    tar xf rocksdb.tar.gz && \
    cd rocksdb-* && \
    make shared_lib -j $(nproc) && \
    mkdir -p /usr/local/rocksdb/lib && \
    mkdir /usr/local/rocksdb/include && \
    cp librocksdb.so* /usr/local/rocksdb/lib && \
    cp /usr/local/rocksdb/lib/librocksdb.so* /usr/lib/ && \
    cp -r include /usr/local/rocksdb/ && \
    cp -r include/* /usr/include/ && \
    rm -R /tmp/rocksdb*

# Install Mitsunari Shigeo's BLS threshold crypto lib
# It has a homebrew build system that requires all four codebases be in the same dir.
RUN apk --update --no-cache add gmp-dev openssl-dev
RUN printf "#!/bin/sh\necho x86_64\n" > /usr/local/bin/arch && \
    chmod +x /usr/local/bin/arch && \
    cd /tmp && \
    curl -L -o mcl.tar.gz https://github.com/herumi/mcl/archive/release20170402.tar.gz && \
    curl -L -o cybozulib.tar.gz https://github.com/herumi/cybozulib/archive/release20170401.tar.gz && \
    curl -L -o xbyak.tar.gz https://github.com/herumi/xbyak/archive/release20170401.tar.gz && \
    tar xf mcl.tar.gz && mv mcl-* mcl && \
    tar xf cybozulib.tar.gz && mv cybozulib-* cybozulib && \
    tar xf xbyak.tar.gz && mv xbyak-* xbyak && \
    cd /tmp/mcl && \
      make lib/libmcl.a && \
      cp lib/*.a /usr/local/lib && \
    cd /tmp && \
    curl -L -o bls.tar.gz https://github.com/herumi/bls/archive/release20170402.tar.gz && \
    tar xf bls.tar.gz && mv bls-* bls && \
    cd /tmp/bls && \
      make lib/libbls.a lib/libbls_if.a && \
      cp lib/*.a /usr/local/lib && \
      cp include/*.h /usr/local/include && \
    rm -R /tmp/bls* && \
    rm -R /tmp/cybozulib* && \
    rm -R /tmp/mcl* && \
    rm -R /tmp/xbyak* && \
    rm /usr/local/bin/arch
