package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/khairnar2960/mavlink-doctor/internal/version"
)

const (
	DefaultPort = 19856
	DefaultBaud = 57600
)

// Config holds all user-specified and resolved CLI options.
type Config struct {
	ListenDuration time.Duration
	ListenAddress  string
	Port           int
	Transport      string // auto, udp, tcp, serial, broadcast
	Mode           string // auto, server, client
	Address        string // remote host:port
	SerialDevice   string
	Baud           int
	Dialect        string // common, ardupilotmega, standard, all, raw
	GCSHeartbeat   bool
	RequestStreams bool
	StreamServer   bool
	SkipGomavlib   bool
	ShowVersion    bool
	JSONOutput     bool
	ExportReport   string
	RedactReport   bool
	ProbeSubnet    string
}

// ParseFlags parses and validates command line arguments.
func ParseFlags(args []string) (*Config, error) {
	fs := flag.NewFlagSet("mavlink-doctor", flag.ContinueOnError)

	listenSeconds := fs.Int("listen", 10, "Duration in seconds for diagnostic receive tests")
	listenAddress := fs.String("listen-address", "0.0.0.0", "Local IPv4/IPv6 address to bind for server mode")
	port := fs.Int("port", DefaultPort, "Local UDP/TCP server port")
	transport := fs.String("transport", "auto", "Transport protocol: auto, udp, tcp, serial, or broadcast")
	mode := fs.String("mode", "auto", "Endpoint mode: auto, server, or client")
	address := fs.String("address", "", "Remote host:port for UDP/TCP client mode")
	serialDevice := fs.String("serial", "", "Serial device path (e.g. /dev/ttyUSB0, COM3)")
	baud := fs.Int("baud", DefaultBaud, "Serial baud rate (e.g. 57600, 115200, 921600)")
	dialect := fs.String("dialect", "common", "MAVLink dialect: common, ardupilotmega, standard, all, or raw")
	gcsHeartbeat := fs.Bool("gcs-heartbeat", true, "Emit diagnostic GCS heartbeat (System ID 255, Component ID 190)")
	requestStreams := fs.Bool("request-streams", false, "Emit automatic MAVLink telemetry stream requests")
	streamServer := fs.Bool("stream-server", true, "Enable UDP stream reassembly server for chunked packets")
	skipGomavlib := fs.Bool("no-gomavlib", false, "Skip MAVLink frame decoding and run raw transport tests only")
	showVersion := fs.Bool("version", false, "Print version information and exit")
	jsonOutput := fs.Bool("json", false, "Emit output in machine-readable JSON format")
	exportReport := fs.String("export-report", "", "Save detailed diagnostic support report to specified JSON file path")
	redactReport := fs.Bool("redact", false, "Redact sensitive IP and MAC addresses in exported support report")
	probeSubnet := fs.String("probe-subnet", "", "Opt-in bounded subnet probe for active MAVLink endpoints (e.g. 192.168.1.0/24)")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "mavlink-doctor - Hardware-agnostic MAVLink diagnostics & telemetry metrics utility\n\n")
		fmt.Fprintf(os.Stderr, "Usage: mavlink-doctor [options]\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if *showVersion {
		return &Config{ShowVersion: true}, nil
	}

	t := strings.ToLower(strings.TrimSpace(*transport))
	if t != "auto" && t != "udp" && t != "tcp" && t != "serial" && t != "broadcast" {
		return nil, fmt.Errorf("invalid -transport %q: must be auto, udp, tcp, serial, or broadcast", *transport)
	}

	m := strings.ToLower(strings.TrimSpace(*mode))
	if m != "auto" && m != "server" && m != "client" {
		return nil, fmt.Errorf("invalid -mode %q: must be auto, server, or client", *mode)
	}

	if t == "serial" && m == "server" {
		return nil, fmt.Errorf("serial transport operates in client mode only")
	}

	if (t == "tcp" || t == "udp") && m == "client" && strings.TrimSpace(*address) == "" {
		return nil, fmt.Errorf("-address host:port is required for explicit %s client mode", t)
	}

	d := strings.ToLower(strings.TrimSpace(*dialect))
	if d != "common" && d != "ardupilotmega" && d != "standard" && d != "all" && d != "raw" {
		return nil, fmt.Errorf("invalid -dialect %q: must be common, ardupilotmega, standard, all, or raw", *dialect)
	}

	dur := time.Duration(*listenSeconds) * time.Second
	if dur <= 0 {
		dur = 10 * time.Second
	}

	return &Config{
		ListenDuration: dur,
		ListenAddress:  *listenAddress,
		Port:           *port,
		Transport:      t,
		Mode:           m,
		Address:        strings.TrimSpace(*address),
		SerialDevice:   strings.TrimSpace(*serialDevice),
		Baud:           *baud,
		Dialect:        d,
		GCSHeartbeat:   *gcsHeartbeat,
		RequestStreams: *requestStreams,
		StreamServer:   *streamServer,
		SkipGomavlib:   *skipGomavlib,
		ShowVersion:    *showVersion,
		JSONOutput:     *jsonOutput,
		ExportReport:   strings.TrimSpace(*exportReport),
		RedactReport:   *redactReport,
		ProbeSubnet:    strings.TrimSpace(*probeSubnet),
	}, nil
}

// PrintBanner outputs the application header to standard output.
func PrintBanner() {
	fmt.Println("============================================================")
	fmt.Printf(" %s\n", version.Short())
	fmt.Println(" Hardware-Agnostic MAVLink Diagnostics & Metrics Engine")
	fmt.Println("============================================================")
}
