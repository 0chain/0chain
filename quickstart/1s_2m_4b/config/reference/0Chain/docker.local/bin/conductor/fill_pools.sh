#!/bin/sh

set -e

# add to read pools
./zboxcli/zbox --wallet testing.json rp-lock --duration=1h \
     --allocation "$(cat ~/.zcn/allocation.txt)" --tokens 2.0

# add to write pools
./zboxcli/zbox --wallet testing.json wp-lock --duration=1h \
    --allocation "$(cat ~/.zcn/allocation.txt)" --tokens 2.0

# auth user
# ./zboxcli/zbox --wallet testing-auth.json rp-lock --duration=1h \
#    --allocation "$(cat ~/.zcn/allocation.txt)" --tokens 2.0
