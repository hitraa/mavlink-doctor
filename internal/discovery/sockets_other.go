//go:build !linux && !darwin && !windows

package discovery

// ScanUDPSockets fallback for other operating systems.
func ScanUDPSockets() ([]SocketInfo, error) {
	return nil, nil
}
