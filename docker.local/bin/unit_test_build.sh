#!/bin/sh

docker build -f docker.local/build.unit_test/Dockerfile . -t zchain_unit_test

docker run zchain_unit_test sh -c '
  echo running zchain_unit_test
  for mod_file in $(find * -name go.mod -maxdepth 2); do
      mod_dir=$(dirname $mod_file)
      (cd $mod_dir; go test -tags bn256 $mod_dir/...)
      if [ $? -ne 0 ]; then
        exit 1
      fi
  done
  exit 0
  '
if [ $? -ne 0 ]
  then exit 1;
fi

