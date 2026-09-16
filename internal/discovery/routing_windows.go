//go:build windows

package discovery

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"
)

// CheckRouting inspects route table and reachability on Windows.
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

	// route print <host>
	cmd := exec.Command("route", "print", host)
	output, err := cmd.CombinedOutput()
	if err == nil {
		res.RouteOutput = strings.TrimSpace(string(output))
		res.IsDirect = !strings.Contains(res.RouteOutput, "0.0.0.0")
	} else {
		res.RouteOutput = fmt.Sprintf("route print error: %v (%s)", err, strings.TrimSpace(string(output)))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	start := time.Now()
	pingCmd := exec.CommandContext(ctx, "ping", "-n", "1", "-w", "2000", host)
	pingOut, pingErr := pingCmd.CombinedOutput()
	res.PingDuration = time.Since(start)
	res.PingOutput = strings.TrimSpace(string(pingOut))
	res.PingResponded = (pingErr == nil)

	return res, nil
}
