//go:build windows

package discovery

import (
	"os/exec"
	"strings"
)

// ScanUDPSockets inspects active UDP sockets using netstat on Windows.
func ScanUDPSockets() ([]SocketInfo, error) {
	cmd := exec.Command("netstat", "-ano", "-p", "udp")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var sockets []SocketInfo

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(strings.ToUpper(line), "UDP") {
			continue
		}
		fields := strings.Fields(line)
		info := SocketInfo{
			Protocol: "UDP",
			Raw:      line,
		}
		if len(fields) >= 2 {
			info.LocalAddr = fields[1]
		}
		if len(fields) >= 3 {
			info.ForeignAddr = fields[2]
		}
		if len(fields) >= 4 {
			info.PID = fields[len(fields)-1]
		}
		sockets = append(sockets, info)
	}

	return sockets, nil
}
