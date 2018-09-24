FROM iron/go:dev
RUN apk add --update --no-cache build-base linux-headers git cmake bash perl
RUN apk add --update --no-cache zlib zlib-dev bzip2 bzip2-dev snappy snappy-dev lz4 lz4-dev jemalloc jemalloc-dev

# Install golang dep
RUN curl -fsSL -o /usr/local/bin/dep https://github.com/golang/dep/releases/download/v0.5.0/dep-linux-amd64 && \
    chmod +x /usr/local/bin/dep

# Install latest gflags
RUN cd /tmp && \
    git clone https://github.com/gflags/gflags.git && \
    cd gflags && \
    mkdir build && \
    cd build && \
    cmake -DBUILD_SHARED_LIBS=1 -DGFLAGS_INSTALL_SHARED_LIBS=1 .. && \
    make install && \
    cd /tmp && \
    rm -R /tmp/gflags/

# Install RocksDB
RUN cd /tmp && \
    git clone https://github.com/facebook/rocksdb.git && \
    cd rocksdb && \
    git checkout 5.13.fb && \
    make shared_lib && \
    mkdir -p /usr/local/rocksdb/lib && \
    mkdir /usr/local/rocksdb/include && \
    cp librocksdb.so* /usr/local/rocksdb/lib && \
    cp /usr/local/rocksdb/lib/librocksdb.so* /usr/lib/ && \
    cp -r include /usr/local/rocksdb/ && \
    cp -r include/* /usr/include/ && \
    rm -R /tmp/rocksdb/

RUN go get github.com/tecbot/gorocksdb

# Install Mitsunari Shigeo's BLS threshold crypto lib
RUN apk --update --no-cache add gmp-dev openssl-dev
RUN printf "#!/bin/sh\necho x86_64\n" > /usr/local/bin/arch && \
    chmod +x /usr/local/bin/arch && \
    cd /tmp && \
    git clone git://github.com/pmer/bls.git && \
    git clone git://github.com/pmer/cybozulib.git && \
    git clone git://github.com/pmer/mcl.git && \
    git clone git://github.com/pmer/xbyak.git && \
    cd /tmp/bls && \
    git checkout release20170402 && \
    cd /tmp/cybozulib && \
    git checkout release20170401 && \
    cd /tmp/mcl && \
    git checkout release20170402 && \
    cd /tmp/xbyak && \
    git checkout release20170401 && \
    cd /tmp/mcl && \
    make lib/libmcl.a && \
    cp lib/*.a /usr/local/lib && \
    cd /tmp/bls && \
    make lib/libbls.a lib/libbls_if.a && \
    cp lib/*.a /usr/local/lib && \
    cp include/*.h /usr/local/include && \
    rm -R /tmp/bls/ && \
    rm -R /tmp/cybozulib/ && \
    rm -R /tmp/mcl/ && \
    rm -R /tmp/xbyak/ && \
    rm /usr/local/bin/arch
