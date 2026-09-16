// Package mavlinkdoctor provides hardware-agnostic diagnostics, discovery,
// stream reassembly, and telemetry metrics tracking for MAVLink networks.
//
// The CLI utility is located in cmd/mavlink-doctor.
//
// Subpackages:
//   - internal/discovery: Network interface, port, route, ping, and serial candidate discovery
//   - internal/transport: UDP, TCP, Serial, and chunked stream reassembly endpoints
//   - internal/decoder: MAVLink v1/v2 frame inspection, dialect management, and payload detection
//   - internal/metrics: Sequence gap detection, packet loss %, throughput, and jitter analysis
//   - internal/reporter: Terminal UI, JSON serialization, and sanitized field support export
//   - internal/simulator: MAVLink mock sender and chunked stream simulator
package mavlinkdoctor
