#!/bin/bash

. ../base/paths.sh


#------------------------------------------------------



$zCLI_Root/zbox upload --localpath $zCLI_Root/zbox \
--remotepath / --allocation "$(cat ~/.zcn/allocation.txt)"

$zCLI_Root/zbox upload --localpath $zWallet_Root/zwallet \
--remotepath / --allocation "$(cat ~/.zcn/allocation.txt)"