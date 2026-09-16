package transport

import (
	"context"
	"net"

	"github.com/bluenviron/gomavlib/v4"
)

// NewUDPStreamClient creates a gomavlib endpoint using custom connected UDP socket
// with IsDatagram=false so gomavlib treats the UDP datagrams as a continuous byte stream,
// reconstructing MAVLink frames split across packet boundaries (e.g. 115-byte chunks).
func NewUDPStreamClient(address string) gomavlib.Endpoint {
	return &gomavlib.EndpointCustomClient{
		Connect: func(ctx context.Context) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "udp4", address)
		},
		Label:      "udp-stream:" + address,
		IsDatagram: false,
	}
}
