#!/bin/sh

# create random file
head -c 5M < /dev/urandom > upload.bin

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

# upload it
try_five_times_on_error ./zboxcli/zbox --wallet testing.json upload \
    --allocation "$(cat ~/.zcn/allocation.txt)" \
    --commit \
    --localpath=upload.bin \
    --remotepath=/remote/upload.bin
