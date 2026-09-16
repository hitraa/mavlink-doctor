//go:build linux

package discovery

import "path/filepath"

// SerialCandidates lists serial device candidates on Linux.
func SerialCandidates() []string {
	patterns := []string{
		"/dev/ttyUSB*",
		"/dev/ttyACM*",
		"/dev/serial/by-id/*",
		"/dev/serial/by-path/*",
	}

	seen := make(map[string]struct{})
	var result []string
	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		for _, match := range matches {
			if _, exists := seen[match]; !exists {
				seen[match] = struct{}{}
				result = append(result, match)
			}
		}
	}
	return result
}
