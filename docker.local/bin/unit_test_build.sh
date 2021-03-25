#!/bin/sh

docker build -f docker.local/build.unit_test/Dockerfile . -t zchain_unit_test

docker run zchain_unit_test sh -c '
  for mod_file in $(find * -name go.mod); do
      mod_dir=$(dirname $mod_file)
      echo mod_dir $mod_dir
  done
  for test_file in $(find * -name  *_test.go); do
    echo about to test $test_file
    package_name=$(dirname $test_file)
    test_name=$(basename $test_file)
    echo test_name $test_name
    echo test_file $test_file
    echo package_name $package_name
    echo go test -tags bn256 $package_name -run $test_name
    (cd $test_dir; go mod init)
    (cd $test_dir; go test -tags bn256 $package_name -run $test_name)
    if [ $? -ne 0 ]; then exit 1; fi
  done;
  exit 0
  '

if [ $? -ne 0 ]; then exit 1; fi

