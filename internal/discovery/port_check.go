package discovery

import (
	"fmt"
	"net"
)

// CheckPort examines whether local port can be bound and inspects ownership.
func CheckPort(listenAddress string, port int, sockets []SocketInfo) PortStatus {
	status := PortStatus{
		Port:         port,
		Protocol:     "UDP",
		Available:    true,
		BoundSockets: make([]SocketInfo, 0),
	}

	portStr := fmt.Sprintf(":%d", port)
	for _, s := range sockets {
		if stringsContainsPort(s.LocalAddr, portStr) || stringsContainsPort(s.Raw, portStr) {
			status.BoundSockets = append(status.BoundSockets, s)
			status.Available = false
		}
	}

	conn, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   parseIP(listenAddress),
		Port: port,
	})

	if err != nil {
		status.BindTestOk = false
		status.BindError = err.Error()
		status.Available = false
	} else {
		status.BindTestOk = true
		conn.Close()
	}

	// Test ephemeral port allocation
	econn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err == nil {
		status.EphemeralTest = econn.LocalAddr().String()
		econn.Close()
	}

	return status
}

func parseIP(addr string) net.IP {
	ip := net.ParseIP(addr)
	if ip == nil {
		return net.IPv4zero
	}
	return ip
}

func stringsContainsPort(str, portStr string) bool {
	return len(str) >= len(portStr) && (str == portStr ||
		str[len(str)-len(portStr):] == portStr ||
		findSubstr(str, portStr))
}

func findSubstr(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
