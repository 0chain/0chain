#!/bin/sh
set -e

cmd="build"

for arg in "$@"
do
    case $arg in
        -m1|--m1|m1)
        echo "The build will be performed for Apple M1 chip"
        cmd="buildx build --platform linux/amd64"
        shift
        ;;
    esac
done

echo "checking if go.mod is upto date
pushd code/go/0chain.net
go mod tidy
if [[ -z $(git status -s) ]]
then
  echo "go mod is clean"
else
  echo "change detected. please run 'go mod tidy' and commit the changes"
  exit
fi
popd

docker $cmd -f docker.local/build.unit_test/Dockerfile . -t zchain_unit_test

#Set this to a value higher than the current number of linter errors. We should lower this number over time
docker run zchain_unit_test bash -c '(cd 0chain.net; go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.52.2; golangci-lint run --build-tags bn256 --timeout 10m0s --skip-files '_test.go' --out-format tab)'
