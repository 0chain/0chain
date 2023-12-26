#!/bin/bash

set -e
success_percent=$1

# Randomly return 0 or 1 based on the success_percent
if [ $((RANDOM % 100)) -lt $success_percent ]; then
  exit 0
else
  exit 1
fi
