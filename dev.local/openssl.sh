#!/bin/bash


brew install openssl
#brew unlink openssl && brew link openssl --force
echo '#OpenSSL 1.1' >> ~/.zshrc
echo 'export PATH="/usr/local/opt/openssl@1.1/bin:$PATH"' >> ~/.zshrc
echo 'export CGO_LDFLAGS="-L/usr/local/opt/openssl@1.1/lib"' >> ~/.zshrc
echo 'export CGO_CPPFLAGS="-I/usr/local/opt/openssl@1.1/include"' >> ~/.zshrc
echo 'export PKG_CONFIG_PATH="/usr/local/opt/openssl@1.1/lib/pkgconfig"' >> ~/.zshrc
echo 'export OPENSSL_ROOT_DIR="/usr/local/opt/openssl@1.1"' >> ~/.zshrc
echo 'export OPENSSL_CRYPTO_LIBRARY="/usr/local/opt/openssl@1.1/lib"' >> ~/.zshrc
echo 'export OPENSSL_LIBRARIES="/usr/local/opt/openssl@1.1/lib"' >> ~/.zshrc
echo 'export OPENSSL_INCLUDE_DIR="/usr/local/opt/openssl@1.1/include"' >> ~/.zshrc