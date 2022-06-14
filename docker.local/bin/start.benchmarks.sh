#!/bin/bash

POSITIONAL_ARGS=()

while [[ $# -gt 0 ]]; do
  case $1 in
    -l|--load)
      echo "Processing 'load' option  Input argument is '$2'"
      export LOAD="$2"
      shift # past argument
      shift # past value
      ;;
    -t|--tests)
      echo "Processing 'tests' option  Input argument is '$2'"
      export TESTS="$2"
      shift # past argument
      shift # past value
      ;;
    -o|--omit)
      echo "Processing 'omit' option  Input argument is '$2'"
      export OMIT="$2"
      shift # past argument
      shift # past value
      ;;
    -v|--verbose)
      echo "Processing 'verbose' option  Input argument is '$2'"
      export VERBOSE="$2"
      shift # past argument
      shift # past value
      ;;
    -c|--config)
      echo "Processing 'config' option  Input argument is '$2'"
      export CONFIG="$2"
      shift # past argument
      shift # past value
      ;;
    --default)
      DEFAULT=YES
      shift # past argument
      ;;
    -*|--*)
      echo "Unknown option $1"
      exit 1
      ;;
    *)
      POSITIONAL_ARGS+=("$1") # save positional arg
      shift # past argument
      ;;
  esac
done

set -- "${POSITIONAL_ARGS[@]}" # restore positional parameters

docker-compose -p benchmarks -f ../build.benchmarks/b0docker-compose.yml up --abort-on-container-exit