 #!/bin/bash

./start_sharders.sh
sleep 10 # It Depends on your cpu speed
./start_miners.sh
sleep 10 # It Depends on your cpu speed
./start_blobbers.sh
sleep 10


