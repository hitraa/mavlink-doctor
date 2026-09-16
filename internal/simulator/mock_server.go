package simulator

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/bluenviron/gomavlib/v4"
	"github.com/bluenviron/gomavlib/v4/pkg/dialects/common"
)

// MockOptions configures the simulator behavior.
type MockOptions struct {
	TargetAddress string
	IsTCP         bool
	ChunkSize     int // if > 0, chunks encoded frames into fixed size UDP datagrams
	SendGimbal    bool
	UseMavlink1   bool
	Interval      time.Duration
	SystemID      uint8
	ComponentID   uint8
	Autopilot     common.MAV_AUTOPILOT
	VehicleType   common.MAV_TYPE
}

// DefaultMockOptions returns standard simulator parameters.
func DefaultMockOptions(targetAddr string) MockOptions {
	return MockOptions{
		TargetAddress: targetAddr,
		IsTCP:         false,
		ChunkSize:     0,
		SendGimbal:    true,
		UseMavlink1:   false,
		Interval:      100 * time.Millisecond,
		SystemID:      1,
		ComponentID:   1,
		Autopilot:     common.MAV_AUTOPILOT_ARDUPILOTMEGA,
		VehicleType:   common.MAV_TYPE_QUADROTOR,
	}
}

// RunMockSender runs a mock MAVLink sender in the background until context is cancelled.
func RunMockSender(ctx context.Context, opts MockOptions) error {
	if opts.ChunkSize > 0 && !opts.IsTCP {
		return runChunkedUDPSender(ctx, opts)
	}

	var endpoint gomavlib.Endpoint
	if opts.IsTCP {
		endpoint = &gomavlib.EndpointTCPClient{Address: opts.TargetAddress}
	} else {
		endpoint = &gomavlib.EndpointUDPClient{Address: opts.TargetAddress}
	}

	outVer := gomavlib.V2
	if opts.UseMavlink1 {
		outVer = gomavlib.V1
	}

	node := &gomavlib.Node{
		Endpoints:      []gomavlib.Endpoint{endpoint},
		Dialect:        common.Dialect,
		OutVersion:     outVer,
		OutSystemID:    opts.SystemID,
		OutComponentID: opts.ComponentID,
	}

	if err := node.Initialize(); err != nil {
		return fmt.Errorf("mock node init error: %w", err)
	}
	defer node.Close()

	ticker := time.NewTicker(opts.Interval)
	defer ticker.Stop()

	seq := 0
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			seq++
			// 1. Emit Heartbeat
			_ = node.WriteMessageAll(&common.MessageHeartbeat{
				Type:           opts.VehicleType,
				Autopilot:      opts.Autopilot,
				BaseMode:       common.MAV_MODE_FLAG_MANUAL_INPUT_ENABLED | common.MAV_MODE_FLAG_STABILIZE_ENABLED,
				CustomMode:     0,
				SystemStatus:   common.MAV_STATE_ACTIVE,
				MavlinkVersion: 3,
			})

			// 2. Emit Attitude
			_ = node.WriteMessageAll(&common.MessageAttitude{
				TimeBootMs: uint32(seq * 100),
				Roll:       0.05,
				Pitch:      -0.02,
				Yaw:        1.57,
			})

			// 3. Optional Gimbal Status
			if opts.SendGimbal && seq%2 == 0 {
				_ = node.WriteMessageAll(&common.MessageGimbalDeviceAttitudeStatus{
					TargetSystem:    255,
					TargetComponent: 190,
					Flags:           1,
					Q:               [4]float32{1.0, 0.0, 0.0, 0.0},
				})
			}
		}
	}
}

// pipeConn implements net.Conn for writing into an io.PipeWriter.
type pipeConn struct {
	pr        *io.PipeReader
	pw        *io.PipeWriter
	raddr     net.Addr
	closeOnce sync.Once
}

func (c *pipeConn) Read(b []byte) (n int, err error)  { return c.pr.Read(b) }
func (c *pipeConn) Write(b []byte) (n int, err error) { return c.pw.Write(b) }
func (c *pipeConn) Close() error {
	c.closeOnce.Do(func() {
		_ = c.pw.Close()
		_ = c.pr.Close()
	})
	return nil
}
func (c *pipeConn) LocalAddr() net.Addr                { return c.raddr }
func (c *pipeConn) RemoteAddr() net.Addr               { return c.raddr }
func (c *pipeConn) SetDeadline(t time.Time) error      { return nil }
func (c *pipeConn) SetReadDeadline(t time.Time) error  { return nil }
func (c *pipeConn) SetWriteDeadline(t time.Time) error { return nil }

// runChunkedUDPSender serializes MAVLink messages via gomavlib and splits the byte stream
// into fixed-size UDP packets (e.g. 115 bytes) crossing frame boundaries to simulate chunked serial-over-UDP radio bridges.
func runChunkedUDPSender(ctx context.Context, opts MockOptions) error {
	raddr, err := net.ResolveUDPAddr("udp", opts.TargetAddress)
	if err != nil {
		return err
	}
	udpConn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		return err
	}
	defer udpConn.Close()

	prNode, pwPipe := io.Pipe()
	prPipe, pwNode := io.Pipe()

	connForNode := &pipeConn{pr: prNode, pw: pwNode, raddr: raddr}

	endpoint := &gomavlib.EndpointCustomClient{
		Connect: func(ctx context.Context) (net.Conn, error) {
			return connForNode, nil
		},
		Label:      "mock-chunker",
		IsDatagram: false,
	}

	node := &gomavlib.Node{
		Endpoints:      []gomavlib.Endpoint{endpoint},
		Dialect:        common.Dialect,
		OutVersion:     gomavlib.V2,
		OutSystemID:    opts.SystemID,
		OutComponentID: opts.ComponentID,
	}

	if err := node.Initialize(); err != nil {
		return err
	}
	defer node.Close()

	// Chunk reading goroutine
	go func() {
		defer pwPipe.Close()
		buf := make([]byte, opts.ChunkSize)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, err := prPipe.Read(buf)
				if err != nil {
					return
				}
				if n > 0 {
					_, _ = udpConn.Write(buf[:n])
				}
			}
		}
	}()

	ticker := time.NewTicker(opts.Interval)
	defer ticker.Stop()

	seq := 0
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			seq++
			_ = node.WriteMessageAll(&common.MessageHeartbeat{
				Type:           opts.VehicleType,
				Autopilot:      opts.Autopilot,
				BaseMode:       common.MAV_MODE_FLAG_MANUAL_INPUT_ENABLED,
				SystemStatus:   common.MAV_STATE_ACTIVE,
				MavlinkVersion: 3,
			})
			_ = node.WriteMessageAll(&common.MessageAttitude{
				TimeBootMs: uint32(seq * 100),
				Roll:       0.01,
				Pitch:      0.02,
				Yaw:        0.03,
			})
		}
	}
}
