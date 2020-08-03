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

# add tokens to write pools
try_five_times_on_error ./zboxcli/zbox --wallet testing.json wp-lock \
    --duration=1h --allocation "$(cat ~/.zcn/allocation.txt)" --tokens 2.0

trap "kill 0" EXIT

go run 0chain/code/go/0chain.net/conductor/sdkproxy/main.go -f uploadMeta &
sleep 3

# create random file
head -c 52428800 < /dev/urandom > random.bin

# upload initial file
HTTP_PROXY="http://0.0.0.0:15211" try_five_times_on_error ./zboxcli/zbox \
    --wallet testing.json upload \
    --allocation "$(cat ~/.zcn/allocation.txt)" \
    --localpath=random.bin \
    --remotepath=/remote/random.bin
