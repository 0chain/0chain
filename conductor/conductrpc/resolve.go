package conductrpc

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
)

func Host(address string) (addr string, err error) {
	var host, port string
	if host, port, err = net.SplitHostPort(address); err != nil {
		return
	}
	if host != "host.docker.internal" {
		return address, nil // return the passed
	}
	var cmd = exec.Command("sh", "-c",
		"ip -4 route list match 0/0 | cut -d' ' -f3")
	var stdout []byte
	if stdout, err = cmd.Output(); err != nil {
		return
	}
	var ip = strings.TrimSpace(string(stdout))
	if net.ParseIP(ip) == nil {
		return "", fmt.Errorf("invalid 'host.docker.internal' resolution: %s",
			ip)
	}
	return ip + ":" + port, nil // host
}
