//go:build linux

package discovery

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"
)

// CheckRouting inspects route table and reachability on Linux.
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

	// 1. ip route get <host>
	cmd := exec.Command("ip", "route", "get", host)
	output, err := cmd.CombinedOutput()
	if err == nil {
		outStr := strings.TrimSpace(string(output))
		res.RouteOutput = outStr
		res.IsDirect = !strings.Contains(outStr, " via ")

		fields := strings.Fields(outStr)
		for i := 0; i < len(fields)-1; i++ {
			if fields[i] == "dev" {
				res.Interface = fields[i+1]
			}
			if fields[i] == "via" {
				res.Gateway = fields[i+1]
			}
		}
	} else {
		res.RouteOutput = fmt.Sprintf("ip route get error: %v (%s)", err, strings.TrimSpace(string(output)))
	}

	// 2. Ping check (1 packet, 2s timeout)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	start := time.Now()
	pingCmd := exec.CommandContext(ctx, "ping", "-c", "1", "-W", "2", host)
	pingOut, pingErr := pingCmd.CombinedOutput()
	res.PingDuration = time.Since(start)
	res.PingOutput = strings.TrimSpace(string(pingOut))
	res.PingResponded = (pingErr == nil)

	return res, nil
}
