#!/bin/sh

set -e

try_five_times_on_error () {
  n=0
  until [ "$n" -ge 5 ]
  do
    case $("$@" 2>&1) in 
      *"consensus failed on sharders"*)
        echo "REPEAT COMMAND"
        ;;
      *)
        return $? # any other error or success
        ;;
    esac
    n=$((n+1)) 
  done
}

# add to read pools
try_five_times_on_error ./zboxcli/zbox --wallet testing.json rp-lock \
    --duration=1h --allocation "$(cat ~/.zcn/allocation.txt)" --tokens 2.0

# add to write pools
try_five_times_on_error ./zboxcli/zbox --wallet testing.json wp-lock \
    --duration=1h --allocation "$(cat ~/.zcn/allocation.txt)" --tokens 2.0

# auth user
try_five_times_on_error ./zboxcli/zbox --wallet testing-auth.json rp-lock \
    --duration=1h --allocation "$(cat ~/.zcn/allocation.txt)" --tokens 2.0
