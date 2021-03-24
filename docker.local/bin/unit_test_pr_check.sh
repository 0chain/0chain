#!/bin/sh

docker build -f docker.local/build.unit_test/Dockerfile . -t zchain_unit_test

docker run zchain_unit_test sh -c '
  for test_file in $(find * -name  *_test.go); do
    echo about to test $test_file
    test_dir=$(dirname $test_file)
    test_name=$(basename $test_file)
    (cd $test_dir; go test -tags bn256 $test_dir -run $test_name)
    if [ $? -ne 0 ]; then exit 1; fi
  done;
  exit 0
  '

if [ $? -ne 0 ]; then exit 1; fi

