package discovery

import (
	"net"
	"time"
)

// InterfaceInfo contains network interface attributes.
type InterfaceInfo struct {
	Name      string   `json:"name"`
	Index     int      `json:"index"`
	MTU       int      `json:"mtu"`
	Flags     string   `json:"flags"`
	Addresses []string `json:"addresses"`
	IPv4Bind  []string `json:"ipv4_bind_candidates"`
}

// SocketInfo represents an active socket listening on the host.
type SocketInfo struct {
	Protocol    string `json:"protocol"`
	LocalAddr   string `json:"local_addr"`
	ForeignAddr string `json:"foreign_addr"`
	PID         string `json:"pid,omitempty"`
	Process     string `json:"process,omitempty"`
	Raw         string `json:"raw"`
}

// RouteResult holds routing table analysis for remote destination.
type RouteResult struct {
	TargetHost    string        `json:"target_host"`
	TargetPort    string        `json:"target_port"`
	Interface     string        `json:"interface,omitempty"`
	Gateway       string        `json:"gateway,omitempty"`
	IsDirect      bool          `json:"is_direct"`
	RouteOutput   string        `json:"route_output"`
	PingResponded bool          `json:"ping_responded"`
	PingOutput    string        `json:"ping_output,omitempty"`
	PingDuration  time.Duration `json:"ping_duration,omitempty"`
}

// PortStatus details local port availability and ownership.
type PortStatus struct {
	Port          int          `json:"port"`
	Protocol      string       `json:"protocol"`
	Available     bool         `json:"available"`
	BoundSockets  []SocketInfo `json:"bound_sockets"`
	BindTestOk    bool         `json:"bind_test_ok"`
	BindError     string       `json:"bind_error,omitempty"`
	EphemeralTest string       `json:"ephemeral_test,omitempty"`
}

// DiscoveryReport aggregates all discovered system attributes.
type DiscoveryReport struct {
	Timestamp        time.Time       `json:"timestamp"`
	OS               string          `json:"os"`
	Arch             string          `json:"arch"`
	Interfaces       []InterfaceInfo `json:"interfaces"`
	UDPSockets       []SocketInfo    `json:"udp_sockets"`
	PortStatus       *PortStatus     `json:"port_status,omitempty"`
	Route            *RouteResult    `json:"route,omitempty"`
	SerialCandidates []string        `json:"serial_candidates"`
	SubnetProbe      []string        `json:"subnet_probe_results,omitempty"`
}

// ExtractIP returns net.IP from net.Addr.
func ExtractIP(addr net.Addr) net.IP {
	switch v := addr.(type) {
	case *net.IPNet:
		return v.IP
	case *net.IPAddr:
		return v.IP
	default:
		return nil
	}
}
