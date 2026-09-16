package discovery

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

// ProbeSubnet attempts to discover responding MAVLink/UDP endpoints within a CIDR.
// It is strictly bounded by maxHosts and per-probe timeouts to avoid network saturation.
func ProbeSubnet(cidr string, targetPort int, maxHosts int) ([]string, error) {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR %q: %w", cidr, err)
	}

	ips := generateIPs(ipnet, maxHosts)
	if len(ips) == 0 {
		return nil, fmt.Errorf("no host IPs generated for %s", cidr)
	}

	var foundMu sync.Mutex
	var found []string
	sem := make(chan struct{}, 20) // concurrency limit

	var wg sync.WaitGroup
	for _, ip := range ips {
		wg.Add(1)
		sem <- struct{}{}

		go func(targetIP string) {
			defer wg.Done()
			defer func() { <-sem }()

			addr := net.JoinHostPort(targetIP, fmt.Sprint(targetPort))
			ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
			defer cancel()

			var d net.Dialer
			conn, err := d.DialContext(ctx, "udp4", addr)
			if err != nil {
				return
			}
			defer conn.Close()

			// Send empty datagram to test routability
			_, _ = conn.Write([]byte{})

			foundMu.Lock()
			found = append(found, addr)
			foundMu.Unlock()
		}(ip.String())
	}

	wg.Wait()
	return found, nil
}

func generateIPs(ipnet *net.IPNet, maxHosts int) []net.IP {
	var ips []net.IP
	ip := ipnet.IP.Mask(ipnet.Mask)

	for ipnet.Contains(ip) {
		inc(ip)
		if !ipnet.Contains(ip) {
			break
		}
		// Skip network and broadcast (.0 and .255 for IPv4 /24)
		if ip[len(ip)-1] == 0 || ip[len(ip)-1] == 255 {
			continue
		}
		dup := make(net.IP, len(ip))
		copy(dup, ip)
		ips = append(ips, dup)
		if len(ips) >= maxHosts {
			break
		}
	}
	return ips
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}
