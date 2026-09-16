//go:build windows

package discovery

import (
	"fmt"
	"os/exec"
	"strings"
)

// SerialCandidates lists available COM ports on Windows.
func SerialCandidates() []string {
	// Query COM ports using PowerShell or registry
	cmd := exec.Command("powershell", "-Command", "[System.IO.Ports.SerialPort]::GetPortNames()")
	output, err := cmd.CombinedOutput()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		var ports []string
		for _, line := range lines {
			p := strings.TrimSpace(line)
			if p != "" {
				ports = append(ports, p)
			}
		}
		if len(ports) > 0 {
			return ports
		}
	}

	// Fallback heuristic: check COM1 to COM32
	var candidates []string
	for i := 1; i <= 32; i++ {
		portName := fmt.Sprintf("COM%d", i)
		// We avoid opening here to prevent resetting DTR/RTS, but return common candidates
		candidates = append(candidates, portName)
	}
	return candidates[:4]
}
