#!/bin/sh
set -e

docker build -f docker.local/build.unit_test/Dockerfile . -t zchain_unit_test

#Set this to a value higher than the current number of linter errors. We should lower this number over time
docker run zchain_unit_test bash -c '
    current_lint_count=516
    apk add curl
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.38.0
    golangci-lint --version
    lint_issues=0
    for mod_file in $(find * -name go.mod); do
        mod_dir=$(dirname $mod_file)
        (cd $mod_dir; go mod download; golangci-lint run --build-tags bn256 --timeout 10m0s | tee lint_report.out)
        (cd $mod_dir; echo piers wc; wc -l lint_report.out)
        issues=`grep ".go" "$mod_dir"/lint_report.out | wc -l`
        echo "The number of lint requests in $mod_dir is $issues"
        lint_issues=`expr "$lint_issues" + "$issues"`
        echo "running total of lint issues is $lint_issues"
    done
    echo "total lint issues is $lint_issues"
    if [ $lint_issues -le $current_lint_count ]
    then
      echo "The number of errors is within the threshold of $current_lint_count"
      exit 0
    else
      echo "The number of errors exceeds the threshold of $current_lint_count" >&2
      exit 1
    fi
'
if [ $? -ne 0 ]
  then exit 1;
fi
