#!/bin/bash

ts=$(date +%s)

echo "Random number: $ts"

if (( ts % 2 == 0 )); then
  exit 0
else
  exit 1
fi