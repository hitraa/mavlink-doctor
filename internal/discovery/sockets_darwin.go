//go:build darwin

package discovery

import (
	"os/exec"
	"strings"
)

// ScanUDPSockets inspects active UDP sockets using lsof / netstat on macOS.
func ScanUDPSockets() ([]SocketInfo, error) {
	cmd := exec.Command("lsof", "-nP", "-iUDP")
	output, err := cmd.CombinedOutput()
	if err != nil {
		cmd = exec.Command("netstat", "-an", "-p", "udp")
		output, err = cmd.CombinedOutput()
		if err != nil {
			return nil, err
		}
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var sockets []SocketInfo

	// Skip header line if present
	startIdx := 0
	if len(lines) > 0 && (strings.HasPrefix(lines[0], "COMMAND") || strings.HasPrefix(lines[0], "Active")) {
		startIdx = 1
	}

	for i := startIdx; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		info := SocketInfo{
			Protocol: "UDP",
			Raw:      line,
		}
		if len(fields) >= 9 {
			info.Process = fields[0]
			info.PID = fields[1]
			info.LocalAddr = fields[8]
		} else if len(fields) >= 4 {
			info.LocalAddr = fields[3]
		}
		sockets = append(sockets, info)
	}

	return sockets, nil
}
