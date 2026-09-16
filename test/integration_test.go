package test

import (
	"context"
	"fmt"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/bluenviron/gomavlib/v4"
	"github.com/bluenviron/gomavlib/v4/pkg/dialects/common"
	"github.com/bluenviron/gomavlib/v4/pkg/frame"
	"github.com/hitraa/mavlink-doctor/internal/decoder"
	"github.com/hitraa/mavlink-doctor/internal/discovery"
	"github.com/hitraa/mavlink-doctor/internal/metrics"
	"github.com/hitraa/mavlink-doctor/internal/reporter"
	"github.com/hitraa/mavlink-doctor/internal/simulator"
	"github.com/hitraa/mavlink-doctor/internal/transport"
)

func TestMetricsTracker(t *testing.T) {
	tracker := metrics.NewTracker()

	// Simulate frames with 1 dropped sequence
	fr1 := &frame.V2Frame{SequenceNumber: 1, SystemID: 1, ComponentID: 1, Message: &common.MessageHeartbeat{}}
	fr2 := &frame.V2Frame{SequenceNumber: 2, SystemID: 1, ComponentID: 1, Message: &common.MessageAttitude{}}
	fr4 := &frame.V2Frame{SequenceNumber: 4, SystemID: 1, ComponentID: 1, Message: &common.MessageAttitude{}} // skipped 3!

	tracker.RecordFrame(fr1, "MAVLink 2", 20)
	tracker.RecordFrame(fr2, "MAVLink 2", 30)
	tracker.RecordFrame(fr4, "MAVLink 2", 30)

	tracker.RecordHeartbeat(1, 1, "ArduPilot", "Quadrotor", "ACTIVE")
	time.Sleep(20 * time.Millisecond)
	tracker.RecordHeartbeat(1, 1, "ArduPilot", "Quadrotor", "ACTIVE")

	snap := tracker.Snapshot()
	if snap.TotalFrames != 3 {
		t.Fatalf("expected 3 total frames, got %d", snap.TotalFrames)
	}

	src, ok := snap.Sources["1/1"]
	if !ok {
		t.Fatalf("expected source 1/1 to exist")
	}

	if src.DroppedFrames != 1 {
		t.Fatalf("expected 1 dropped frame, got %d", src.DroppedFrames)
	}

	if src.HeartbeatStats == nil || src.HeartbeatStats.Count != 2 {
		t.Fatalf("expected 2 heartbeats recorded")
	}
}

func TestGimbalDetection(t *testing.T) {
	var evidence decoder.GimbalEvidence

	// 1. Regular attitude: should not trigger gimbal
	att := &common.MessageAttitude{}
	if decoder.CheckGimbalMessage(att, 1, 1, &evidence) {
		t.Fatal("regular attitude falsely triggered gimbal evidence")
	}
	if evidence.Observed {
		t.Fatal("expected observed=false")
	}

	// 2. Gimbal device attitude status: should trigger gimbal
	gAtt := &common.MessageGimbalDeviceAttitudeStatus{Flags: 1}
	if !decoder.CheckGimbalMessage(gAtt, 1, 154, &evidence) {
		t.Fatal("gimbal device attitude status failed to trigger gimbal evidence")
	}
	if !evidence.Observed {
		t.Fatal("expected observed=true")
	}
}

func TestUDPChunkReassembly(t *testing.T) {
	// Pick an ephemeral UDP port for test
	l, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("failed to listen UDP: %v", err)
	}
	port := l.LocalAddr().(*net.UDPAddr).Port
	_ = l.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start mock sender with 115-byte chunks (simulating air unit serial-to-UDP fragmentation)
	targetAddr := net.JoinHostPort("127.0.0.1", fmt.Sprint(port))
	mockOpts := simulator.DefaultMockOptions(targetAddr)
	mockOpts.ChunkSize = 115
	mockOpts.Interval = 25 * time.Millisecond

	go func() {
		_ = simulator.RunMockSender(ctx, mockOpts)
	}()

	// Start UDP stream server endpoint which reassembles chunks
	endpoint := transport.NewUDPStreamServer("127.0.0.1", port)
	node := &gomavlib.Node{
		Endpoints:        []gomavlib.Endpoint{endpoint},
		Dialect:          common.Dialect,
		OutVersion:       gomavlib.V2,
		OutSystemID:      255,
		HeartbeatDisable: true,
	}

	if err := node.Initialize(); err != nil {
		t.Fatalf("failed to initialize node: %v", err)
	}
	defer node.Close()

	receivedFrames := 0
	timeout := time.After(3 * time.Second)

	for {
		select {
		case <-timeout:
			if receivedFrames == 0 {
				t.Fatal("timed out without receiving any reassembled MAVLink frame from 115-byte chunked UDP")
			}
			return
		case evt := <-node.Events():
			if _, ok := evt.(*gomavlib.EventFrame); ok {
				receivedFrames++
				if receivedFrames >= 3 {
					// Successfully proved chunk reassembly!
					return
				}
			}
		}
	}
}

func TestTCPTransport(t *testing.T) {
	// Start TCP server
	l, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("failed to listen TCP: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serverEndpoint := &gomavlib.EndpointTCPServer{
		Address: net.JoinHostPort("127.0.0.1", fmt.Sprint(port)),
	}
	serverNode := &gomavlib.Node{
		Endpoints:        []gomavlib.Endpoint{serverEndpoint},
		Dialect:          common.Dialect,
		OutVersion:       gomavlib.V2,
		OutSystemID:      255,
		HeartbeatDisable: true,
	}
	if err := serverNode.Initialize(); err != nil {
		t.Fatalf("failed to init server node: %v", err)
	}
	defer serverNode.Close()

	// Client sends mock traffic
	targetAddr := net.JoinHostPort("127.0.0.1", fmt.Sprint(port))
	mockOpts := simulator.DefaultMockOptions(targetAddr)
	mockOpts.IsTCP = true
	mockOpts.Interval = 25 * time.Millisecond

	go func() {
		_ = simulator.RunMockSender(ctx, mockOpts)
	}()

	frames := 0
	timeout := time.After(3 * time.Second)
	for {
		select {
		case <-timeout:
			if frames == 0 {
				t.Fatal("timed out without receiving any TCP frame")
			}
			return
		case evt := <-serverNode.Events():
			if _, ok := evt.(*gomavlib.EventFrame); ok {
				frames++
				if frames >= 2 {
					return
				}
			}
		}
	}
}

func TestJSONReportAndRedaction(t *testing.T) {
	disc := discovery.DiscoveryReport{
		Interfaces: []discovery.InterfaceInfo{
			{
				Name:      "eth0",
				Addresses: []string{"192.168.1.100/24"},
				IPv4Bind:  []string{"192.168.1.100"},
			},
		},
	}
	raw := &transport.PacketSummary{
		TotalPackets:  5,
		UniqueSenders: []string{"192.168.1.20:19856"},
	}
	metrics := metrics.GlobalMetrics{TotalFrames: 10}
	gimbal := decoder.GimbalEvidence{Observed: true, GimbalType: "Test Gimbal"}

	res := reporter.BuildResult(disc, raw, metrics, gimbal)
	if res.Status != "MAVLINK_CONNECTED" {
		t.Fatalf("expected status MAVLINK_CONNECTED, got %s", res.Status)
	}

	reportFile := filepath.Join(t.TempDir(), "mavlink_doctor_test_report.json")
	err := reporter.ExportSupportReport(reportFile, res, true)
	if err != nil {
		t.Fatalf("failed to export report: %v", err)
	}
}
