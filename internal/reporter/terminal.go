package reporter

import (
	"fmt"
	"strings"
	"time"

	"github.com/khairnar2960/mavlink-doctor/internal/decoder"
	"github.com/khairnar2960/mavlink-doctor/internal/discovery"
	"github.com/khairnar2960/mavlink-doctor/internal/metrics"
	"github.com/khairnar2960/mavlink-doctor/internal/transport"
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

// PrintInterfaces outputs scanned network interfaces.
func PrintInterfaces(ifaces []discovery.InterfaceInfo) {
	Section("1. NETWORK INTERFACES")
	if len(ifaces) == 0 {
		Warn("No network interfaces discovered.")
		return
	}

	for _, iface := range ifaces {
		fmt.Printf("\nInterface: %s (Index: %d, MTU: %d, Flags: %s)\n",
			iface.Name, iface.Index, iface.MTU, iface.Flags)
		for _, addr := range iface.Addresses {
			fmt.Printf("  Address: %s\n", addr)
		}
		for _, ip := range iface.IPv4Bind {
			Ok(fmt.Sprintf("IPv4 candidate: %s on %s", ip, iface.Name))
		}
	}
}

// PrintSockets outputs discovered UDP sockets.
func PrintSockets(sockets []discovery.SocketInfo, ifaces []discovery.InterfaceInfo) {
	Section("2. LOCAL IP ADDRESSES & UDP SOCKETS")
	for _, iface := range ifaces {
		for _, ip := range iface.IPv4Bind {
			fmt.Printf("Local UDP bind candidate: %-15s (%s)\n", ip, iface.Name)
		}
	}
	fmt.Println()
	if len(sockets) == 0 {
		Info("No active UDP sockets detected or host tools unavailable.")
		return
	}
	for _, s := range sockets {
		fmt.Printf("UDP socket: %-22s -> %-22s %s\n", s.LocalAddr, s.ForeignAddr, s.Process)
	}
}

// PrintRouting outputs route table and ICMP reachability results.
func PrintRouting(route *discovery.RouteResult) {
	Section("3. NETWORK REACHABILITY & ROUTE")
	if route == nil {
		Info("No remote destination supplied; routing and reachability checks skipped.")
		return
	}

	fmt.Printf("Target destination : %s:%s\n", route.TargetHost, route.TargetPort)
	if route.RouteOutput != "" {
		fmt.Printf("Route details      : %s\n", route.RouteOutput)
	}

	if route.IsDirect {
		Ok("Direct local link / subnet route to target host.")
	} else if route.Gateway != "" {
		Info(fmt.Sprintf("Route directs via gateway: %s (Interface: %s)", route.Gateway, route.Interface))
	}

	if route.PingResponded {
		Ok(fmt.Sprintf("Remote host answered ICMP ping in %s.", route.PingDuration.Round(time.Millisecond)))
	} else {
		Warn("Remote host did not respond to ICMP ping (normal for hardened air units/autopilots).")
	}
}

// PrintPortStatus outputs port availability and bindability.
func PrintPortStatus(status discovery.PortStatus) {
	Section("4. PORT STATUS & BINDABILITY")
	if status.Available {
		Ok(fmt.Sprintf("Port UDP %d is unowned and available for binding.", status.Port))
	} else {
		Fail(fmt.Sprintf("Port UDP %d has conflicts or is owned by an existing socket.", status.Port))
		for _, s := range status.BoundSockets {
			Info(fmt.Sprintf("Conflicting socket: %s", s.Raw))
		}
	}

	if status.BindTestOk {
		Ok(fmt.Sprintf("Local socket test bind on UDP %d succeeded.", status.Port))
	} else {
		Fail(fmt.Sprintf("Local socket test bind failed: %s", status.BindError))
	}

	if status.EphemeralTest != "" {
		Ok(fmt.Sprintf("Ephemeral UDP client port allocation succeeded: %s", status.EphemeralTest))
	}
}

// PrintSerialCandidates outputs discovered serial devices.
func PrintSerialCandidates(candidates []string) {
	Section("5. SERIAL TRANSPORT DISCOVERY")
	if len(candidates) == 0 {
		Info("No serial device candidates discovered on this operating system.")
		Info("If using USB-TTL / radio, check device permissions (e.g. dialout group) or specify -serial /dev/... or COMx.")
		return
	}
	for _, c := range candidates {
		fmt.Printf("Serial candidate: %s\n", c)
	}
}

// PrintRawPacketSummary outputs results from raw UDP listener.
func PrintRawPacketSummary(summary *transport.PacketSummary) {
	Section("6. RAW UDP PACKET SNIFFER")
	fmt.Printf("Listen Target: %s:%d (Duration: %s)\n\n", summary.ListenAddress, summary.Port, summary.Duration)
	if summary.TotalPackets == 0 {
		Fail("NO UDP packets received during test window.")
		Info("Possible causes:")
		Info("  - The remote radio / autopilot is not configured to send to this host IP/port.")
		Info("  - A local firewall (iptables / ufw / Windows Defender) is dropping inbound UDP.")
		Info("  - Network interface link is down or cable disconnected.")
		Info(fmt.Sprintf("Run packet sniffer to verify wire traffic: %s", transport.FormatTcpdumpCommand(summary.Port, "")))
		return
	}

	Ok(fmt.Sprintf("Received %d raw UDP datagram(s) (%d total bytes).", summary.TotalPackets, summary.TotalBytes))
	if summary.MAVLinkLike > 0 {
		Ok(fmt.Sprintf("%d datagram(s) begin with MAVLink magic byte (0xFE / 0xFD).", summary.MAVLinkLike))
	} else {
		Warn("Packets arrived, but none began with MAVLink magic byte 0xFE or 0xFD at byte 0.")
		Info("This strongly indicates serial-over-UDP chunk fragmentation! Air unit is splitting frames across datagrams.")
		Info("Ensure -stream-server=true or UDP client stream mode is active.")
	}

	for _, s := range summary.UniqueSenders {
		Info("Datagram sender source: " + s)
	}
}

// PrintMAVLinkSummary outputs decoded MAVLink statistics and health evaluation.
func PrintMAVLinkSummary(global metrics.GlobalMetrics, gimbal decoder.GimbalEvidence) {
	Section("7. MAVLINK TELEMETRY & LINK METRICS")
	if global.TotalFrames == 0 {
		Fail("No MAVLink frames were decoded during the test window.")
		Info("Check dialect compatibility (-dialect common/ardupilotmega/all) and transport settings.")
		return
	}

	Ok(fmt.Sprintf("Decoded %d MAVLink frame(s) across %d source(s) in %.1fs.",
		global.TotalFrames, len(global.Sources), global.DurationSeconds))

	fmt.Println("\nFrame Version Breakdown:")
	for v, count := range global.VersionCounts {
		fmt.Printf("  • %-16s: %d frame(s)\n", v, count)
	}

	if global.TotalParseErrors > 0 {
		Warn(fmt.Sprintf("Parse errors encountered: %d frame(s) failed CRC or framing check.", global.TotalParseErrors))
	} else {
		Ok("Zero framing/CRC parse errors detected.")
	}

	fmt.Printf("\nOverall Throughput: %.1f msgs/sec | %.1f bytes/sec\n", global.OverallMsgRateHz, global.OverallThroughputBps)

	// Per-Source Telemetry Details
	for _, src := range global.Sources {
		fmt.Printf("\n------------------------------------------------------------\n")
		fmt.Printf("Source System ID: %d | Component ID: %d (%s)\n",
			src.SystemID, src.ComponentID, decoder.ComponentName(src.ComponentID))
		fmt.Printf("------------------------------------------------------------\n")
		fmt.Printf("  • Total frames received : %d\n", src.TotalFrames)
		fmt.Printf("  • Dropped frames        : %d (Packet loss: %.2f%%)\n", src.DroppedFrames, src.PacketLossPct)
		fmt.Printf("  • Message rate          : %.1f Hz\n", src.MsgRateHz)
		fmt.Printf("  • Data throughput       : %.1f bytes/sec\n", src.DataRateBytesSec)
		fmt.Printf("  • Inter-packet jitter   : %.2f ms\n", src.JitterMs)

		if src.HeartbeatStats != nil {
			hb := src.HeartbeatStats
			fmt.Printf("  • Heartbeats observed   : %d\n", hb.Count)
			fmt.Printf("  • Autopilot identity    : %s\n", hb.Autopilot)
			fmt.Printf("  • Vehicle type          : %s\n", hb.VehicleType)
			fmt.Printf("  • System flight status  : %s\n", hb.SystemStatus)
			fmt.Printf("  • Heartbeat frequency   : %.2f Hz (avg interval: %.1f ms, jitter: %.1f ms)\n",
				hb.RateHz, hb.AvgIntervalMs, hb.JitterMs)

			if hb.IsHealthy {
				Ok("Heartbeat link health: HEALTHY & STABLE (regular periodic heartbeats).")
			} else if hb.Count == 1 {
				Warn("Heartbeat link health: UNSTABLE (only 1 heartbeat received; link may be intermittent).")
			} else {
				Warn(fmt.Sprintf("Heartbeat link health: IRREGULAR (interval variation: %.1f - %.1f ms).",
					hb.MinIntervalMs, hb.MaxIntervalMs))
			}
		} else {
			Warn("No HEARTBEAT message observed from this source yet.")
		}
	}

	// Gimbal Evidence Section
	fmt.Printf("\n------------------------------------------------------------\n")
	fmt.Println("Gimbal & Payload Subsystem Observation:")
	fmt.Printf("------------------------------------------------------------\n")
	if gimbal.Observed {
		Ok(fmt.Sprintf("Gimbal payload CONFIRMED: %s (sys=%d comp=%d)", gimbal.GimbalType, gimbal.SystemID, gimbal.ComponentID))
		Info(fmt.Sprintf("Evidence messages: %s", strings.Join(gimbal.MessagesSeen, ", ")))
		if gimbal.Details != "" {
			Info("Details: " + gimbal.Details)
		}
	} else {
		Info("No gimbal heartbeat or status messages were observed in this window.")
		Info("Note: Gimbal absence is NOT proven; some gimbals communicate via CAN or require explicit telemetry streams.")
	}
}

// PrintFinalRecommendation prints actionable configuration snippets and next steps.
func PrintFinalRecommendation(port int) {
	Section("8. DIAGNOSTIC RECOMMENDATIONS & NEXT STEPS")
	fmt.Print(`
Recommended Troubleshooting Order:
  1. Verify the local network interface has an assigned IPv4 address in the vehicle/air unit subnet.
  2. For UDP Air Units (SIYI, Microhard, RFD900, DoodleLabs, Herelink):
     - If the air unit streams to your PC: use server mode with UDP stream reassembly:
       mavlink-doctor -transport udp -mode server -port ` + fmt.Sprint(port) + ` -stream-server=true
     - If the air unit acts as a server: connect in UDP client mode:
       mavlink-doctor -transport udp -mode client -address <device-ip>:<device-port>
  3. For Serial Links (USB-TTL, FTDI, Holybro, CUAV, Pixhawk TELEM):
     mavlink-doctor -transport serial -serial /dev/ttyACM0 -baud 115200
  4. For Telemetry Streams:
     - GCS heartbeat is enabled by default (sys=255, comp=190).
     - If heartbeats are received but data streams are absent, enable stream requests:
       mavlink-doctor -request-streams=true
  5. Packet Capture:
     Run in another shell: ` + transport.FormatTcpdumpCommand(port, "") + `
`)
}
