//go:build darwin

package discovery

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"
)

// CheckRouting inspects route table and reachability on macOS.
func CheckRouting(remoteAddress string) (*RouteResult, error) {
	if remoteAddress == "" {
		return nil, nil
	}

	host, port, err := net.SplitHostPort(remoteAddress)
	if err != nil {
		host = remoteAddress
	}

	res := &RouteResult{
		TargetHost: host,
		TargetPort: port,
	}

	// route -n get <host>
	cmd := exec.Command("route", "-n", "get", host)
	output, err := cmd.CombinedOutput()
	if err == nil {
		outStr := strings.TrimSpace(string(output))
		res.RouteOutput = outStr

		lines := strings.Split(outStr, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "interface:") {
				res.Interface = strings.TrimSpace(strings.TrimPrefix(line, "interface:"))
			}
			if strings.HasPrefix(line, "gateway:") {
				res.Gateway = strings.TrimSpace(strings.TrimPrefix(line, "gateway:"))
			}
		}
		res.IsDirect = (res.Gateway == "")
	} else {
		res.RouteOutput = fmt.Sprintf("route -n get error: %v (%s)", err, strings.TrimSpace(string(output)))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	start := time.Now()
	pingCmd := exec.CommandContext(ctx, "ping", "-c", "1", "-t", "2", host)
	pingOut, pingErr := pingCmd.CombinedOutput()
	res.PingDuration = time.Since(start)
	res.PingOutput = strings.TrimSpace(string(pingOut))
	res.PingResponded = (pingErr == nil)

	return res, nil
}
