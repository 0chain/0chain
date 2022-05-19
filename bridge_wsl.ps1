$wsl_ip=wsl hostname -I
$port=15210 # conductor server port
netsh int ipv4 reset
netsh interface portproxy add v4tov4 listenport=$port listenaddress=0.0.0.0 connectport=$port connectaddress=$wsl_ip
echo "bridging $wsl_ip"
