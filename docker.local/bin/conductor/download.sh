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


rm -f got.bin

try_five_times_on_error ./zboxcli/zbox --wallet testing.json download \
    --allocation "$(cat ~/.zcn/allocation.txt)" \
    --localpath=got.bin \
    --remotepath /remote/random.bin
