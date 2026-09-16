//go:build !linux && !darwin && !windows

package discovery

// CheckRouting fallback for other operating systems.
func CheckRouting(remoteAddress string) (*RouteResult, error) {
	return nil, nil
}
