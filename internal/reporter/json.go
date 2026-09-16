package reporter

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/khairnar2960/mavlink-doctor/internal/decoder"
	"github.com/khairnar2960/mavlink-doctor/internal/discovery"
	"github.com/khairnar2960/mavlink-doctor/internal/metrics"
	"github.com/khairnar2960/mavlink-doctor/internal/transport"
	"github.com/khairnar2960/mavlink-doctor/internal/version"
)

// DiagnosticResult encapsulates the complete diagnostic execution outcome.
type DiagnosticResult struct {
	ToolVersion    string                    `json:"tool_version"`
	Discovery      discovery.DiscoveryReport `json:"discovery"`
	RawPacketTest  *transport.PacketSummary  `json:"raw_packet_test,omitempty"`
	Metrics        metrics.GlobalMetrics     `json:"metrics"`
	GimbalEvidence decoder.GimbalEvidence    `json:"gimbal_evidence"`
	Status         string                    `json:"overall_status"`
	SummaryNotes   []string                  `json:"summary_notes"`
}

// WriteJSON encodes DiagnosticResult to writer with clean 2-space indentation.
func WriteJSON(w io.Writer, res DiagnosticResult) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(res)
}

// PrintJSON outputs the result to standard out.
func PrintJSON(res DiagnosticResult) error {
	return WriteJSON(os.Stdout, res)
}

// BuildResult compiles all subsystem reports into a unified DiagnosticResult.
func BuildResult(disc discovery.DiscoveryReport, raw *transport.PacketSummary, global metrics.GlobalMetrics, gimbal decoder.GimbalEvidence) DiagnosticResult {
	status := "NO_TRAFFIC"
	var notes []string

	if global.TotalFrames > 0 {
		status = "MAVLINK_CONNECTED"
		notes = append(notes, fmt.Sprintf("Successfully received %d MAVLink frames", global.TotalFrames))
		healthyHB := false
		for _, s := range global.Sources {
			if s.HeartbeatStats != nil && s.HeartbeatStats.IsHealthy {
				healthyHB = true
				break
			}
		}
		if healthyHB {
			notes = append(notes, "Periodic healthy heartbeat observed from vehicle/autopilot")
		} else {
			notes = append(notes, "Heartbeat not yet established or unstable")
		}
	} else if raw != nil && raw.TotalPackets > 0 {
		status = "PACKETS_WITHOUT_MAVLINK"
		notes = append(notes, fmt.Sprintf("Received %d UDP packets but no valid MAVLink frames decoded", raw.TotalPackets))
	} else {
		notes = append(notes, "No UDP packets or MAVLink frames received during diagnostic window")
	}

	if gimbal.Observed {
		notes = append(notes, fmt.Sprintf("Gimbal detected: %s", gimbal.GimbalType))
	}

	return DiagnosticResult{
		ToolVersion:    version.Short(),
		Discovery:      disc,
		RawPacketTest:  raw,
		Metrics:        global,
		GimbalEvidence: gimbal,
		Status:         status,
		SummaryNotes:   notes,
	}
}
