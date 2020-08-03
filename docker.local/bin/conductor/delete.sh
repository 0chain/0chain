#!/bin/sh

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

try_five_times_on_error ./zboxcli/zbox --wallet testing.json delete \
    --allocation "$(cat ~/.zcn/allocation.txt)" \
    --remotepath /remote/random.bin
