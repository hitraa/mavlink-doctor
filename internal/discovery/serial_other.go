//go:build !linux && !darwin && !windows

package discovery

// SerialCandidates lists serial device candidates for unsupported platforms.
func SerialCandidates() []string {
	return nil
}
