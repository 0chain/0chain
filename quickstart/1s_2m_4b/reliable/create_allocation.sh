 
#!/bin/bash

. ../base/paths.sh


#---------------------------------------------------

$zCLI_Root/zbox sp-lock --blobber_id 2a4d5a5c6c0976873f426128d2ff23a060ee715bccf0fd3ca5e987d57f25b78e --tokens 0.5
$zCLI_Root/zbox sp-lock --blobber_id 2f051ca6447d8712a020213672bece683dbd0d23a81fdf93ff273043a0764d18 --tokens 0.5
$zCLI_Root/zbox sp-lock --blobber_id 7a90e6790bcd3d78422d7a230390edc102870fe58c15472073922024985b1c7d --tokens 0.5
$zCLI_Root/zbox sp-lock --blobber_id f65af5d64000c7cd2883f4910eb69086f9d6e6635c744e62afcfab58b938ee25 --tokens 0.5

sleep 10

$zCLI_Root/zbox newallocation --lock 1