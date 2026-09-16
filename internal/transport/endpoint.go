package transport

import (
	"errors"
	"fmt"
	"net"

	"github.com/bluenviron/gomavlib/v4"
)

// EndpointOptions holds settings required to initialize a transport endpoint.
type EndpointOptions struct {
	Transport     string // udp, tcp, serial, broadcast
	Mode          string // server, client
	ListenAddress string
	Port          int
	Address       string
	SerialDevice  string
	Baud          int
	StreamServer  bool
}

// CreateEndpoint creates the appropriate gomavlib.Endpoint instance.
func CreateEndpoint(opts EndpointOptions) (gomavlib.Endpoint, error) {
	switch opts.Transport {
	case "udp":
		if opts.Mode == "server" {
			if opts.StreamServer {
				return NewUDPStreamServer(opts.ListenAddress, opts.Port), nil
			}
			return &gomavlib.EndpointUDPServer{
				Address: net.JoinHostPort(opts.ListenAddress, fmt.Sprint(opts.Port)),
			}, nil
		}
		// Client mode: use stream client to tolerate chunked serial-over-UDP frames
		return NewUDPStreamClient(opts.Address), nil

	case "tcp":
		if opts.Mode == "server" {
			return &gomavlib.EndpointTCPServer{
				Address: net.JoinHostPort(opts.ListenAddress, fmt.Sprint(opts.Port)),
			}, nil
		}
		return &gomavlib.EndpointTCPClient{Address: opts.Address}, nil

	case "serial":
		if opts.Mode != "client" && opts.Mode != "auto" {
			return nil, errors.New("serial transport requires client mode")
		}
		if opts.SerialDevice == "" {
			return nil, errors.New("serial device must be specified")
		}
		return &gomavlib.EndpointSerial{
			Device: opts.SerialDevice,
			Baud:   opts.Baud,
		}, nil

	case "broadcast":
		return &gomavlib.EndpointUDPBroadcast{
			BroadcastAddress: opts.Address,
			LocalAddress:     net.JoinHostPort(opts.ListenAddress, fmt.Sprint(opts.Port)),
		}, nil

	default:
		return nil, fmt.Errorf("unsupported transport %q", opts.Transport)
	}
}
