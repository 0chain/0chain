#!/bin/bash
set -e

# Define the installation directory and repository URL
INSTALL_DIR=$(go env GOPATH)/src/github.com/0chain/msgp
REPO_URL=https://github.com/0chain/msgp.git
VERSION=v1.2.0

# Check if the directory exists; if not, clone the repository
if [ ! -d "$INSTALL_DIR" ]; then
  echo "Cloning the repository..."
  git clone $REPO_URL $INSTALL_DIR
fi

# Navigate to the repository directory
cd $INSTALL_DIR

# Checkout the desired version
git fetch --tags
echo "Checking out version $VERSION..."
git checkout $VERSION

# Build and install the package
echo "Building and installing msgp..."
go install ./...

echo "msgp installed successfully."
