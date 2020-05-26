package conductrpc

import (
	"fmt"
	"net"
	"os/exec"
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
	if net.ParseIP(string(stdout)) == nil {
		return "", fmt.Errorf("invalid 'host.docker.internal' resolution: %s",
			string(stdout))
	}
	return string(stdout) + ":" + port, nil // host
}
