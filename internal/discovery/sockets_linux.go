//go:build linux

package discovery

import (
	"os/exec"
	"strings"
)

// ScanUDPSockets inspects active UDP sockets using ss on Linux.
func ScanUDPSockets() ([]SocketInfo, error) {
	cmd := exec.Command("ss", "-H", "-u", "-a", "-n", "-p")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Fallback to netstat if ss is unavailable
		cmd = exec.Command("netstat", "-unpa")
		output, err = cmd.CombinedOutput()
		if err != nil {
			return nil, err
		}
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var sockets []SocketInfo

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		info := SocketInfo{
			Protocol: "UDP",
			Raw:      line,
		}
		if len(fields) >= 4 {
			info.LocalAddr = fields[3]
		}
		if len(fields) >= 5 {
			info.ForeignAddr = fields[4]
		}
		if len(fields) >= 6 {
			info.Process = strings.Join(fields[5:], " ")
		}
		sockets = append(sockets, info)
	}

	return sockets, nil
}
