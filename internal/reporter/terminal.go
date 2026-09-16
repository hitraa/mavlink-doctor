package reporter

import (
	"fmt"
	"strings"
	"time"

	"github.com/hitraa/mavlink-doctor/internal/decoder"
	"github.com/hitraa/mavlink-doctor/internal/discovery"
	"github.com/hitraa/mavlink-doctor/internal/metrics"
	"github.com/hitraa/mavlink-doctor/internal/transport"
)

// Section prints a formatted section banner.
func Section(title string) {
	fmt.Printf("\n============================================================\n")
	fmt.Printf(" %s\n", title)
	fmt.Printf("============================================================\n")
}

// Ok prints success checkmark.
func Ok(msg string) {
	fmt.Printf("✅ %s\n", msg)
}

// Warn prints warning indicator.
func Warn(msg string) {
	fmt.Printf("⚠️  %s\n", msg)
}

// Fail prints error indicator.
func Fail(msg string) {
	fmt.Printf("❌ %s\n", msg)
}

// Info prints indented context line.
func Info(msg string) {
	fmt.Printf("   %s\n", msg)
}

// PrintDiscoverySummary outputs a concise, non-bloated environment summary.
func PrintDiscoverySummary(ifaces []discovery.InterfaceInfo, portStatus *discovery.PortStatus, serialCandidates []string, route *discovery.RouteResult, verbose bool) {
	fmt.Println("\n[1/3] Environment & Discovery")

	// 1. IP Candidates
	var ips []string
	for _, iface := range ifaces {
		for _, ip := range iface.IPv4Bind {
			ips = append(ips, fmt.Sprintf("%s (%s)", ip, iface.Name))
		}
	}
	if len(ips) > 0 {
		fmt.Printf("  ✅ Local IP Candidates : %s\n", strings.Join(ips, ", "))
	} else {
		fmt.Printf("  ⚠️  Local IP Candidates : None found\n")
	}

	// 2. Port status
	if portStatus != nil {
		if portStatus.Available {
			fmt.Printf("  ✅ Port UDP %-10d : Available (No socket conflicts)\n", portStatus.Port)
		} else {
			fmt.Printf("  ❌ Port UDP %-10d : Conflict (In use by another process)\n", portStatus.Port)
			for _, s := range portStatus.BoundSockets {
				fmt.Printf("       Conflicting: %s\n", s.Raw)
			}
		}
	}

	// 3. Routing
	if route != nil {
		if route.IsDirect {
			fmt.Printf("  ✅ Route to %s : Direct link via %s\n", route.TargetHost, route.Interface)
		} else if route.Gateway != "" {
			fmt.Printf("  ℹ️  Route to %s : Gateway %s (dev %s)\n", route.TargetHost, route.Gateway, route.Interface)
		}
		if route.PingResponded {
			fmt.Printf("  ✅ Ping to %s  : Responded in %s\n", route.TargetHost, route.PingDuration.Round(time.Millisecond))
		}
	}

	// 4. Serial candidates
	if len(serialCandidates) > 0 {
		fmt.Printf("  ✅ Serial Ports        : %s\n", strings.Join(serialCandidates, ", "))
	} else {
		fmt.Printf("  ℹ️  Serial Ports        : None detected\n")
	}

	// Verbose dump only if requested
	if verbose {
		fmt.Println("\n--- Verbose Interface & Socket Details ---")
		for _, iface := range ifaces {
			fmt.Printf("Interface: %s (MTU: %d, Flags: %s, Addrs: %v)\n", iface.Name, iface.MTU, iface.Flags, iface.Addresses)
		}
		if portStatus != nil && len(portStatus.BoundSockets) > 0 {
			fmt.Println("Active UDP Sockets:")
			for _, s := range portStatus.BoundSockets {
				fmt.Println("  " + s.Raw)
			}
		}
	}
}

// PrintRawPacketSummary outputs concise raw UDP sniffer results.
func PrintRawPacketSummary(summary *transport.PacketSummary, verbose bool) {
	fmt.Println("\n[2/3] Raw Packet Sniffer")
	if summary.TotalPackets == 0 {
		fmt.Printf("  ❌ Inbound Traffic    : 0 UDP packets on port %d (%s window)\n", summary.Port, summary.Duration)
		fmt.Printf("     Possible causes: target host not sending to this IP/port, or firewall drop.\n")
		return
	}

	pct := 0
	if summary.TotalPackets > 0 {
		pct = (summary.MAVLinkLike * 100) / summary.TotalPackets
	}

	fmt.Printf("  ✅ Inbound Traffic    : %d UDP datagrams (%d bytes) from %s\n",
		summary.TotalPackets, summary.TotalBytes, strings.Join(summary.UniqueSenders, ", "))
	if summary.MAVLinkLike > 0 {
		fmt.Printf("  ✅ MAVLink Framing    : %d%% of packets match MAVLink v1/v2 magic bytes\n", pct)
	} else {
		fmt.Printf("  ⚠️  MAVLink Framing    : Packets arrived without leading 0xFE/0xFD magic (chunked stream)\n")
	}

	if verbose {
		fmt.Printf("Sample packet sizes: %v\n", summary.PacketSizes)
	}
}

// PrintLiveStart announces start of telemetry listening.
func PrintLiveStart(transport, mode string, port int, address string) {
	fmt.Println("\n[3/3] Live Telemetry Monitor")
	if address != "" {
		fmt.Printf("  Connecting to %s %s at %s...\n", transport, mode, address)
	} else {
		fmt.Printf("  Listening on %s %s 0.0.0.0:%d...\n", transport, mode, port)
	}
}

// PrintMAVLinkSummary outputs a structured, concise table of telemetry metrics.
func PrintMAVLinkSummary(global metrics.GlobalMetrics, gimbal decoder.GimbalEvidence, reportFile string, verbose bool) {
	Section("TELEMETRY HEALTH & METRICS")

	if global.TotalFrames == 0 {
		Fail("No MAVLink frames were decoded during the test window.")
		Info("Check dialect compatibility (-dialect common/ardupilotmega/all) and transport settings.")
		if reportFile != "" {
			fmt.Printf("\n📁 Detailed diagnostic report saved to: %s\n", reportFile)
		}
		return
	}

	Ok(fmt.Sprintf("Decoded %d frames across %d source(s) in %.1fs (%.1f msgs/s | %.1f B/s)",
		global.TotalFrames, len(global.Sources), global.DurationSeconds, global.OverallMsgRateHz, global.OverallThroughputBps))

	if global.TotalParseErrors > 0 {
		Warn(fmt.Sprintf("Parse errors: %d frame(s) failed CRC or framing check", global.TotalParseErrors))
	}

	fmt.Println()
	fmt.Printf("%-10s %-16s %-24s %-10s %-10s %s\n",
		"SOURCE", "COMPONENT", "TYPE / AUTOPILOT", "MSG RATE", "HB RATE", "STATUS")
	fmt.Println(strings.Repeat("-", 80))

	for _, src := range global.Sources {
		compName := shortComponentName(src.ComponentID)
		typeStr := "Telemetry"
		hbRateStr := "n/a"
		statusStr := fmt.Sprintf("Active (loss: %.1f%%)", src.PacketLossPct)

		if src.HeartbeatStats != nil {
			hb := src.HeartbeatStats
			hbRateStr = fmt.Sprintf("%.2f Hz", hb.RateHz)
			typeStr = shortTypeString(hb.VehicleType)
			if hb.IsHealthy {
				statusStr = fmt.Sprintf("Healthy (%s)", shortStatusString(hb.SystemStatus))
			} else {
				statusStr = fmt.Sprintf("Unstable (%s)", shortStatusString(hb.SystemStatus))
			}
		}

		fmt.Printf("%-10s %-16s %-24s %-10s %-10s %s\n",
			fmt.Sprintf("%d / %d", src.SystemID, src.ComponentID),
			compName,
			typeStr,
			fmt.Sprintf("%.1f Hz", src.MsgRateHz),
			hbRateStr,
			statusStr,
		)
	}

	// Gimbal observation
	fmt.Println()
	if gimbal.Observed {
		Ok(fmt.Sprintf("Gimbal Payload : Confirmed (%s)", strings.Join(gimbal.MessagesSeen, ", ")))
		if gimbal.Details != "" {
			Info(gimbal.Details)
		}
	} else {
		Info("Gimbal Payload : Not observed in this window (unproven)")
	}

	if reportFile != "" {
		fmt.Printf("\n📁 Full diagnostic report exported to: %s\n", reportFile)
	}
	if !verbose {
		fmt.Println("💡 Tip: Use -v / -verbose for full frame stream and raw socket dump.")
	}
}

// PrintFinalRecommendation prints actionable configuration snippets and next steps.
func PrintFinalRecommendation(port int) {
	fmt.Print(`
Troubleshooting Quick Reference:
  • Air Unit sends to PC  : mavlink-doctor -transport udp -mode server -port ` + fmt.Sprint(port) + ` -stream-server=true
  • Connect to Air Unit   : mavlink-doctor -transport udp -mode client -address <device-ip>:<device-port>
  • Serial / USB-TTL Radio: mavlink-doctor -transport serial -serial /dev/ttyUSB0 -baud 57600
  • Packet capture check  : ` + transport.FormatTcpdumpCommand(port, "") + `
`)
}

func shortComponentName(compID uint8) string {
	switch compID {
	case 1:
		return "Autopilot"
	case 154:
		return "Gimbal 1"
	case 155:
		return "Gimbal 2"
	case 100:
		return "Camera 1"
	case 191:
		return "Companion PC"
	case 190:
		return "GCS"
	default:
		return fmt.Sprintf("Component %d", compID)
	}
}

func shortTypeString(vehType string) string {
	if strings.Contains(vehType, "Quadrotor") {
		return "Quadrotor"
	}
	if strings.Contains(vehType, "Gimbal") {
		return "Payload Gimbal"
	}
	if strings.Contains(vehType, "Fixed Wing") {
		return "Fixed Wing"
	}
	if strings.Contains(vehType, "Hexarotor") {
		return "Hexarotor"
	}
	if len(vehType) > 22 {
		return vehType[:22]
	}
	return vehType
}

func shortStatusString(status string) string {
	if strings.Contains(status, "STANDBY") {
		return "STANDBY"
	}
	if strings.Contains(status, "ACTIVE") {
		return "ACTIVE"
	}
	if strings.Contains(status, "UNINIT") {
		return "UNINIT"
	}
	if strings.Contains(status, "BOOT") {
		return "BOOT"
	}
	return status
}
