package discovery

import (
	"fmt"
	"net"
)

// ScanInterfaces lists and analyzes all network interfaces on the host.
func ScanInterfaces() ([]InterfaceInfo, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("unable to enumerate network interfaces: %w", err)
	}

	var results []InterfaceInfo

	for _, iface := range ifaces {
		info := InterfaceInfo{
			Name:      iface.Name,
			Index:     iface.Index,
			MTU:       iface.MTU,
			Flags:     iface.Flags.String(),
			Addresses: make([]string, 0),
			IPv4Bind:  make([]string, 0),
		}

		addrs, err := iface.Addrs()
		if err == nil {
			for _, addr := range addrs {
				info.Addresses = append(info.Addresses, addr.String())
				ip := ExtractIP(addr)
				if ip != nil && ip.To4() != nil {
					info.IPv4Bind = append(info.IPv4Bind, ip.String())
				}
			}
		}

		results = append(results, info)
	}

	return results, nil
}

// FindAllIPv4Candidates returns all non-loopback or all IPv4 bind candidates.
func FindAllIPv4Candidates(ifaces []InterfaceInfo) []string {
	var candidates []string
	for _, iface := range ifaces {
		for _, ip := range iface.IPv4Bind {
			candidates = append(candidates, fmt.Sprintf("%s (%s)", ip, iface.Name))
		}
	}
	return candidates
}
