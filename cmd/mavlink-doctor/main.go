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
		if cfg.Address != "" {
			fmt.Printf("Target: %s %s %s | Dialect: %s | Duration: %s\n",
				cfg.Transport, cfg.Mode, cfg.Address, cfg.Dialect, cfg.ListenDuration)
		} else {
			fmt.Printf("Target: %s %s %s:%d | Dialect: %s | Duration: %s\n",
				cfg.Transport, cfg.Mode, cfg.ListenAddress, cfg.Port, cfg.Dialect, cfg.ListenDuration)
		}
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

	sockets, _ := discovery.ScanUDPSockets()
	discReport.UDPSockets = sockets

	if cfg.Address != "" {
		route, _ := discovery.CheckRouting(cfg.Address)
		discReport.Route = route
	}

	if cfg.Transport == "auto" || cfg.Transport == "udp" {
		portStatus := discovery.CheckPort(cfg.ListenAddress, cfg.Port, discReport.UDPSockets)
		discReport.PortStatus = &portStatus
	}

	discReport.SerialCandidates = discovery.SerialCandidates()

	if cfg.ProbeSubnet != "" {
		probed, probeErr := discovery.ProbeSubnet(cfg.ProbeSubnet, cfg.Port, 254)
		if probeErr == nil {
			discReport.SubnetProbe = probed
		}
	}

	if !cfg.JSONOutput {
		reporter.PrintDiscoverySummary(discReport.Interfaces, discReport.PortStatus, discReport.SerialCandidates, discReport.Route, cfg.Verbose)
	}

	// 2. Raw UDP Sniffer (if applicable in local server mode)
	var rawSummary *transport.PacketSummary
	if (cfg.Transport == "auto" || cfg.Transport == "udp") && cfg.Address == "" {
		rawSummary, _ = transport.ListenRawUDP(cfg.ListenDuration, cfg.ListenAddress, cfg.Port,
			func(ts time.Time, remoteAddr string, bytesReceived int, hexPreview string, isMavlink bool) {
				if cfg.Verbose && !cfg.JSONOutput {
					mavStr := "Non-MAVLink"
					if isMavlink {
						mavStr = "MAVLink magic"
					}
					fmt.Printf("[%s] %-21s | %4d B | %s | %s\n",
						ts.Format("15:04:05.000"), remoteAddr, bytesReceived, hexPreview, mavStr)
				}
			},
		)

		if !cfg.JSONOutput && rawSummary != nil {
			reporter.PrintRawPacketSummary(rawSummary, cfg.Verbose)
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

	if cfg.ExportReport != "" {
		if err := reporter.ExportSupportReport(cfg.ExportReport, result, cfg.RedactReport); err != nil {
			if !cfg.JSONOutput {
				reporter.Fail(fmt.Sprintf("Failed to export report: %v", err))
			}
		}
	}

	if cfg.JSONOutput {
		_ = reporter.PrintJSON(result)
	} else {
		reporter.PrintMAVLinkSummary(metricsSnapshot, gimbalEvidence, cfg.ExportReport, cfg.Verbose)
		if metricsSnapshot.TotalFrames == 0 {
			reporter.PrintFinalRecommendation(cfg.Port)
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
		reporter.PrintLiveStart(selTransport, selMode, cfg.Port, cfg.Address)
	}

	testTimer := time.NewTimer(cfg.ListenDuration)
	defer testTimer.Stop()

	announcedSources := make(map[string]bool)

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
				if cfg.Verbose && !cfg.JSONOutput {
					reporter.Ok(fmt.Sprintf("Transport channel opened: %s", e.Channel))
				}
			case *gomavlib.EventFrame:
				verStr := decoder.FrameVersion(e.Frame)
				approxBytes := 12
				msg := e.Message()
				sysID := e.SystemID()
				compID := e.ComponentID()
				srcKey := fmt.Sprintf("%d/%d", sysID, compID)

				tracker.RecordFrame(e.Frame, verStr, approxBytes)
				decoder.CheckGimbalMessage(msg, sysID, compID, gimbalEvidence)

				if hb, isHB := msg.(*common.MessageHeartbeat); isHB {
					apStr := decoder.AutopilotName(uint8(hb.Autopilot))
					vehStr := decoder.VehicleTypeName(uint8(hb.Type))
					statStr := decoder.SystemStatusName(uint8(hb.SystemStatus))
					tracker.RecordHeartbeat(sysID, compID, apStr, vehStr, statStr)

					if !announcedSources[srcKey] && !cfg.JSONOutput {
						announcedSources[srcKey] = true
						reporter.Ok(fmt.Sprintf("[%s] Online sys=%d comp=%d : %s (%s)",
							time.Now().Format("15:04:05"), sysID, compID, vehStr, statStr))
					}
				} else if !announcedSources[srcKey] && !cfg.JSONOutput {
					announcedSources[srcKey] = true
					reporter.Ok(fmt.Sprintf("[%s] Telemetry sys=%d comp=%d : Active (%s)",
						time.Now().Format("15:04:05"), sysID, compID, decoder.ComponentName(compID)))
				}

				if cfg.Verbose && !cfg.JSONOutput {
					fmt.Printf("[%s] FRAME sys=%d comp=%d seq=%-3d ver=%s msg=%T\n",
						time.Now().Format("15:04:05.000"), sysID, compID, e.Frame.GetSequenceNumber(), verStr, msg)
				}
			case *gomavlib.EventParseError:
				tracker.RecordParseError()
				if cfg.Verbose && !cfg.JSONOutput {
					reporter.Warn(fmt.Sprintf("[%s] Parse error: %v", time.Now().Format("15:04:05.000"), e.Error))
				}
			case *gomavlib.EventStreamRequested:
				if cfg.Verbose && !cfg.JSONOutput {
					reporter.Info(fmt.Sprintf("Stream request sent to sys=%d comp=%d", e.SystemID, e.ComponentID))
				}
			case *gomavlib.EventChannelClose:
				if cfg.Verbose && !cfg.JSONOutput {
					reporter.Warn(fmt.Sprintf("Channel closed: %v", e.Error))
				}
			}
		}
	}
}
