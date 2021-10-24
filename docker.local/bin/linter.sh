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

docker $cmd -f docker.local/build.unit_test/Dockerfile . -t zchain_unit_test

#Set this to a value higher than the current number of linter errors. We should lower this number over time
docker run zchain_unit_test bash -c '
    current_lint_count=281
    apk add curl
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.38.0
    (cd 0chain.net; golangci-lint run --build-tags bn256 --timeout 10m0s | tee lint_report.out)
    lint_issues=`grep ".go" 0chain.net/lint_report.out | wc -l`
    if [ $lint_issues -le $current_lint_count ]
    then
      echo "The number of lint issues, $lint_issues, is within the threshold of $current_lint_count"
      exit 0
    else
      echo "The number of lint issues $lint_issues exceeds the threshold of $current_lint_count"
      echo "The number of lint issues $lint_issues exceeds the threshold of $current_lint_count" >&2
      exit 1
    fi
'