//go:build darwin

package discovery

import "path/filepath"

// SerialCandidates lists serial device candidates on macOS.
func SerialCandidates() []string {
	patterns := []string{
		"/dev/cu.*",
		"/dev/tty.*",
	}

	seen := make(map[string]struct{})
	var result []string
	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		for _, match := range matches {
			// Skip internal bluetooth devices
			if filepath.Base(match) == "cu.Bluetooth-Incoming-Port" {
				continue
			}
			if _, exists := seen[match]; !exists {
				seen[match] = struct{}{}
				result = append(result, match)
			}
		}
	}
	return result
}
