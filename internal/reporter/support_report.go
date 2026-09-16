package reporter

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"regexp"
)

// ExportSupportReport writes diagnostic results to disk, optionally redacting sensitive IPs.
func ExportSupportReport(path string, res DiagnosticResult, redact bool) error {
	if redact {
		res = redactResult(res)
	}

	data, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal support report: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write support report to %s: %w", path, err)
	}

	return nil
}

var ipRegex = regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)

func redactResult(res DiagnosticResult) DiagnosticResult {
	// Deep copy & redact interface addresses
	for i := range res.Discovery.Interfaces {
		for j := range res.Discovery.Interfaces[i].Addresses {
			res.Discovery.Interfaces[i].Addresses[j] = maskIP(res.Discovery.Interfaces[i].Addresses[j])
		}
		for j := range res.Discovery.Interfaces[i].IPv4Bind {
			res.Discovery.Interfaces[i].IPv4Bind[j] = maskIP(res.Discovery.Interfaces[i].IPv4Bind[j])
		}
	}

	for i := range res.Discovery.UDPSockets {
		res.Discovery.UDPSockets[i].LocalAddr = maskIP(res.Discovery.UDPSockets[i].LocalAddr)
		res.Discovery.UDPSockets[i].ForeignAddr = maskIP(res.Discovery.UDPSockets[i].ForeignAddr)
		res.Discovery.UDPSockets[i].Raw = maskIP(res.Discovery.UDPSockets[i].Raw)
	}

	if res.Discovery.Route != nil {
		res.Discovery.Route.TargetHost = maskIP(res.Discovery.Route.TargetHost)
		res.Discovery.Route.Gateway = maskIP(res.Discovery.Route.Gateway)
		res.Discovery.Route.RouteOutput = maskIP(res.Discovery.Route.RouteOutput)
		res.Discovery.Route.PingOutput = maskIP(res.Discovery.Route.PingOutput)
	}

	if res.RawPacketTest != nil {
		res.RawPacketTest.ListenAddress = maskIP(res.RawPacketTest.ListenAddress)
		for i := range res.RawPacketTest.UniqueSenders {
			res.RawPacketTest.UniqueSenders[i] = maskIP(res.RawPacketTest.UniqueSenders[i])
		}
	}

	return res
}

func maskIP(input string) string {
	return ipRegex.ReplaceAllStringFunc(input, func(ipStr string) string {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			return ipStr
		}
		if ip.IsLoopback() {
			return ipStr
		}
		v4 := ip.To4()
		if v4 != nil {
			return fmt.Sprintf("%d.%d.xxx.xxx", v4[0], v4[1])
		}
		return "xxxx:xxxx:..."
	})
}
