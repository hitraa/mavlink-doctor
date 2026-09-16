package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/bluenviron/gomavlib/v4"
	"github.com/bluenviron/gomavlib/v4/pkg/dialects/common"
	"github.com/khairnar2960/mavlink-doctor/internal/cli"
	"github.com/khairnar2960/mavlink-doctor/internal/decoder"
	"github.com/khairnar2960/mavlink-doctor/internal/discovery"
	"github.com/khairnar2960/mavlink-doctor/internal/metrics"
	"github.com/khairnar2960/mavlink-doctor/internal/reporter"
	"github.com/khairnar2960/mavlink-doctor/internal/transport"
	"github.com/khairnar2960/mavlink-doctor/internal/version"
)

func main() {
	cfg, err := cli.ParseFlags(os.Args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if cfg.ShowVersion {
		fmt.Println(version.String())
		return
	}

	if !cfg.JSONOutput {
		cli.PrintBanner()
		fmt.Printf("OS/Arch       : %s/%s\n", runtime.GOOS, runtime.GOARCH)
		fmt.Printf("Transport     : %s (mode: %s)\n", cfg.Transport, cfg.Mode)
		if cfg.Address != "" {
			fmt.Printf("Remote Target : %s\n", cfg.Address)
		} else {
			fmt.Printf("Local Bind    : %s:%d\n", cfg.ListenAddress, cfg.Port)
		}
		if cfg.SerialDevice != "" {
			fmt.Printf("Serial Device : %s @ %d baud\n", cfg.SerialDevice, cfg.Baud)
		}
		fmt.Printf("Dialect       : %s\n", cfg.Dialect)
		fmt.Printf("GCS Heartbeat : %t | Stream Requests: %t\n", cfg.GCSHeartbeat, cfg.RequestStreams)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// 1. Discovery Phase
	discReport := discovery.DiscoveryReport{
		Timestamp: time.Now(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}

	ifaces, err := discovery.ScanInterfaces()
	if err == nil {
		discReport.Interfaces = ifaces
	}
	if !cfg.JSONOutput {
		reporter.PrintInterfaces(discReport.Interfaces)
	}

	sockets, _ := discovery.ScanUDPSockets()
	discReport.UDPSockets = sockets
	if !cfg.JSONOutput {
		reporter.PrintSockets(discReport.UDPSockets, discReport.Interfaces)
	}

	if cfg.Address != "" {
		route, _ := discovery.CheckRouting(cfg.Address)
		discReport.Route = route
		if !cfg.JSONOutput {
			reporter.PrintRouting(discReport.Route)
		}
	} else if !cfg.JSONOutput {
		reporter.PrintRouting(nil)
	}

	if cfg.Transport == "auto" || cfg.Transport == "udp" {
		portStatus := discovery.CheckPort(cfg.ListenAddress, cfg.Port, discReport.UDPSockets)
		discReport.PortStatus = &portStatus
		if !cfg.JSONOutput {
			reporter.PrintPortStatus(portStatus)
		}
	}

	discReport.SerialCandidates = discovery.SerialCandidates()
	if !cfg.JSONOutput {
		reporter.PrintSerialCandidates(discReport.SerialCandidates)
	}

	if cfg.ProbeSubnet != "" {
		if !cfg.JSONOutput {
			reporter.Section("SUBNET ACTIVE PROBE")
			fmt.Printf("Probing subnet %s for active MAVLink endpoints on port %d...\n", cfg.ProbeSubnet, cfg.Port)
		}
		probed, probeErr := discovery.ProbeSubnet(cfg.ProbeSubnet, cfg.Port, 254)
		if probeErr == nil {
			discReport.SubnetProbe = probed
			if !cfg.JSONOutput {
				if len(probed) > 0 {
					reporter.Ok(fmt.Sprintf("Discovered %d active UDP endpoint(s):", len(probed)))
					for _, p := range probed {
						reporter.Info(p)
					}
				} else {
					reporter.Info("No responding hosts found on target subnet.")
				}
			}
		} else if !cfg.JSONOutput {
			reporter.Warn(fmt.Sprintf("Subnet probe error: %v", probeErr))
		}
	}

	// 2. Raw UDP Sniffer (if applicable)
	var rawSummary *transport.PacketSummary
	if (cfg.Transport == "auto" || cfg.Transport == "udp") && cfg.Address == "" {
		if !cfg.JSONOutput {
			reporter.Section("6. RAW UDP PACKET SNIFFER")
			fmt.Printf("Listening for raw UDP packets on %s:%d for %s...\n",
				cfg.ListenAddress, cfg.Port, cfg.ListenDuration)
		}

		rawSummary, _ = transport.ListenRawUDP(cfg.ListenDuration, cfg.ListenAddress, cfg.Port,
			func(ts time.Time, remoteAddr string, bytesReceived int, hexPreview string, isMavlink bool) {
				if !cfg.JSONOutput {
					mavStr := "Non-MAVLink"
					if isMavlink {
						mavStr = "MAVLink magic found"
					}
					fmt.Printf("[%s] %-21s | %4d B | %s | %s\n",
						ts.Format("15:04:05.000"), remoteAddr, bytesReceived, hexPreview, mavStr)
				}
			},
		)

		if !cfg.JSONOutput && rawSummary != nil {
			reporter.PrintRawPacketSummary(rawSummary)
		}
	}

	// 3. MAVLink Protocol & Metrics Test
	tracker := metrics.NewTracker()
	var gimbalEvidence decoder.GimbalEvidence

	if !cfg.SkipGomavlib {
		runMavlinkTest(ctx, cfg, tracker, &gimbalEvidence)
	}

	// 4. Summarize and Report Results
	metricsSnapshot := tracker.Snapshot()
	result := reporter.BuildResult(discReport, rawSummary, metricsSnapshot, gimbalEvidence)

	if cfg.JSONOutput {
		_ = reporter.PrintJSON(result)
	} else {
		reporter.PrintMAVLinkSummary(metricsSnapshot, gimbalEvidence)
		reporter.PrintFinalRecommendation(cfg.Port)
	}

	if cfg.ExportReport != "" {
		if err := reporter.ExportSupportReport(cfg.ExportReport, result, cfg.RedactReport); err != nil {
			if !cfg.JSONOutput {
				reporter.Fail(fmt.Sprintf("Failed to export support report: %v", err))
			}
		} else if !cfg.JSONOutput {
			reporter.Ok(fmt.Sprintf("Exported diagnostic support report to %s (Redacted: %t)", cfg.ExportReport, cfg.RedactReport))
		}
	}
}

func runMavlinkTest(ctx context.Context, cfg *cli.Config, tracker *metrics.Tracker, gimbalEvidence *decoder.GimbalEvidence) {
	selTransport := cfg.Transport
	selMode := cfg.Mode

	if selTransport == "auto" {
		if cfg.Address != "" {
			selTransport = "udp"
			selMode = "client"
		} else {
			selTransport = "udp"
			selMode = "server"
		}
	}

	if selMode == "auto" {
		if cfg.Address != "" {
			selMode = "client"
		} else {
			selMode = "server"
		}
	}

	endpoint, err := transport.CreateEndpoint(transport.EndpointOptions{
		Transport:     selTransport,
		Mode:          selMode,
		ListenAddress: cfg.ListenAddress,
		Port:          cfg.Port,
		Address:       cfg.Address,
		SerialDevice:  cfg.SerialDevice,
		Baud:          cfg.Baud,
		StreamServer:  cfg.StreamServer,
	})

	if err != nil {
		if !cfg.JSONOutput {
			reporter.Fail(fmt.Sprintf("Failed to configure %s %s endpoint: %v", selTransport, selMode, err))
		}
		return
	}

	dialectObj, err := decoder.GetDialect(cfg.Dialect)
	if err != nil {
		if !cfg.JSONOutput {
			reporter.Fail(fmt.Sprintf("Invalid dialect: %v", err))
		}
		return
	}

	node := &gomavlib.Node{
		Endpoints:              []gomavlib.Endpoint{endpoint},
		Dialect:                dialectObj,
		OutVersion:             gomavlib.V2,
		OutSystemID:            255, // Standard GCS identity
		OutComponentID:         190,
		HeartbeatDisable:       !cfg.GCSHeartbeat,
		HeartbeatPeriod:        time.Second,
		StreamRequestEnable:    cfg.RequestStreams,
		StreamRequestFrequency: 4,
	}

	if err := node.Initialize(); err != nil {
		if !cfg.JSONOutput {
			reporter.Fail(fmt.Sprintf("gomavlib node initialization failed: %v", err))
		}
		return
	}
	defer node.Close()

	if !cfg.JSONOutput {
		reporter.Section("7. MAVLINK LIVE DECODER & TELEMETRY MONITOR")
		reporter.Ok(fmt.Sprintf("Listening for MAVLink events via %s %s endpoint...", selTransport, selMode))
	}

	testTimer := time.NewTimer(cfg.ListenDuration)
	defer testTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-testTimer.C:
			return
		case evt, ok := <-node.Events():
			if !ok {
				return
			}
			switch e := evt.(type) {
			case *gomavlib.EventChannelOpen:
				if !cfg.JSONOutput {
					reporter.Ok(fmt.Sprintf("Transport channel opened: %s", e.Channel))
				}
			case *gomavlib.EventFrame:
				verStr := decoder.FrameVersion(e.Frame)
				approxBytes := 12 // minimal header size + message estimate
				msg := e.Message()
				sysID := e.SystemID()
				compID := e.ComponentID()

				sm := tracker.RecordFrame(e.Frame, verStr, approxBytes)
				decoder.CheckGimbalMessage(msg, sysID, compID, gimbalEvidence)

				if hb, isHB := msg.(*common.MessageHeartbeat); isHB {
					apStr := decoder.AutopilotName(uint8(hb.Autopilot))
					vehStr := decoder.VehicleTypeName(uint8(hb.Type))
					statStr := decoder.SystemStatusName(uint8(hb.SystemStatus))
					tracker.RecordHeartbeat(sysID, compID, apStr, vehStr, statStr)

					if !cfg.JSONOutput {
						reporter.Ok(fmt.Sprintf("[%s] HEARTBEAT from sys=%d comp=%d | %s | %s | %s",
							time.Now().Format("15:04:05.000"), sysID, compID, apStr, vehStr, statStr))
					}
				} else if !cfg.JSONOutput && (sm.TotalFrames <= 10 || sm.TotalFrames%20 == 0) {
					fmt.Printf("[%s] FRAME sys=%d comp=%d seq=%-3d ver=%s msg=%T\n",
						time.Now().Format("15:04:05.000"), sysID, compID, e.Frame.GetSequenceNumber(), verStr, msg)
				}
			case *gomavlib.EventParseError:
				tracker.RecordParseError()
				if !cfg.JSONOutput {
					reporter.Warn(fmt.Sprintf("[%s] Parse error on channel %s: %v",
						time.Now().Format("15:04:05.000"), e.Channel, e.Error))
				}
			case *gomavlib.EventStreamRequested:
				if !cfg.JSONOutput {
					reporter.Info(fmt.Sprintf("Sent data stream request to system=%d component=%d", e.SystemID, e.ComponentID))
				}
			case *gomavlib.EventChannelClose:
				if !cfg.JSONOutput {
					reporter.Warn(fmt.Sprintf("Transport channel closed: %v", e.Error))
				}
			}
		}
	}
}
