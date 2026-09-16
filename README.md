# mavlink-doctor

[![CI](https://github.com/khairnar2960/mavlink-doctor/actions/workflows/ci.yml/badge.svg)](https://github.com/khairnar2960/mavlink-doctor/actions/workflows/ci.yml)
[![Release](https://github.com/khairnar2960/mavlink-doctor/actions/workflows/release.yml/badge.svg)](https://github.com/khairnar2960/mavlink-doctor/actions/workflows/release.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/khairnar2960/mavlink-doctor)](https://golang.org)
[![Conventional Commits](https://img.shields.io/badge/Conventional%20Commits-1.0.0-yellow.svg)](https://conventionalcommits.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

`mavlink-doctor` is a hardware-agnostic, production-grade diagnostic instrument and telemetry metrics engine for MAVLink connectivity between ground stations, autopilots, flight controllers, companion computers, radios, air units, gimbals, and simulators.

The tool does not assume a specific vendor or link (SIYI, Microhard, RFD900, Herelink, ArduPilot, PX4, or fixed IP/port). It automatically inspects the local network, discovers ports and serial candidates, reassembles chunked datagram streams, decodes live MAVLink v1/v2 frames, measures link health metrics (packet loss, sequence gaps, throughput, jitter), detects payloads (gimbals, cameras), and delivers evidence-based diagnostics.

---

## Key Capabilities

- **Hardware-Agnostic Connectivity**: Works with UDP, TCP, Serial, and UDP Broadcast links across client and server modes.
- **Chunked UDP Reassembly**: Resolves serial-to-UDP fragmentation (e.g. 115-byte radio chunks split across packet boundaries) in both UDP client and UDP server modes.
- **Cross-Platform Discovery**: Native interface, socket ownership, routing, and serial device detection for Linux, macOS, and Windows.
- **Deep MAVLink Decoding**: Detects wire protocol (MAVLink 1 vs 2), verifies CRCs, counts parse errors, and supports selectable dialects (`common`, `ardupilotmega`, `standard`, `all`, `raw`).
- **Autopilot & Vehicle Telemetry**: Translates protocol enums for Autopilots (ArduPilot, PX4, INAV, etc.), Vehicle Types (Multirotor, Plane, Helicopter, VTOL, Rover, Boat, Sub), and Flight States.
- **Payload & Gimbal Confirmation**: Detects gimbal heartbeats, MAVLink Gimbal Protocol v2 (`GIMBAL_DEVICE_ATTITUDE_STATUS`, `GIMBAL_DEVICE_INFORMATION`), Mount Orientation, and Camera components.
- **Real-Time Telemetry Metrics**: Computes per-source sequence gap counts, packet loss %, message rates (Hz), data throughput (bytes/sec), inter-frame jitter, and heartbeat interval stability.
- **Flexible Reporting**:
  - Interactive colorized terminal UI with status badges (`✅`, `⚠️`, `❌`).
  - Machine-readable JSON output (`-json`) for automated test rigs and CI pipelines.
  - Sanitized field support report exports (`-export-report`, `-redact`) masking sensitive IPs and MACs.
- **Opt-In Subnet Prober**: Bounded, rate-limited active scan (`-probe-subnet 192.168.1.0/24`) to discover responsive MAVLink endpoints.

---

## Quick Start

### 1. Installation

#### Pre-built Binaries
Download the latest pre-compiled binary for Linux, macOS, or Windows from the [Releases](https://github.com/khairnar2960/mavlink-doctor/releases) page.

#### From Source
```bash
git clone https://github.com/khairnar2960/mavlink-doctor.git
cd mavlink-doctor
make build
./bin/mavlink-doctor -version
```

---

## Usage Examples

### 1. Automatic Local Discovery & Listener
Inspect local interfaces, socket conflicts, serial candidates, and listen for incoming UDP MAVLink:
```bash
mavlink-doctor
```

### 2. UDP Air Units & Radios (SIYI, Microhard, DoodleLabs)
Air units that push telemetry datagrams to the computer:
```bash
# Server mode with UDP stream reassembly (tolerates chunked packets)
mavlink-doctor -transport udp -mode server -port 19856 -stream-server=true
```

Connecting directly to an Air Unit or Radio listener:
```bash
# Client mode with continuous stream reassembly
mavlink-doctor -transport udp -mode client -address 192.168.144.12:19856
```

### 3. Serial / USB-TTL Telemetry Radios (RFD900, SiK, FTDI)
```bash
# Linux
mavlink-doctor -transport serial -serial /dev/ttyUSB0 -baud 57600

# macOS
mavlink-doctor -transport serial -serial /dev/cu.usbserial-1410 -baud 115200

# Windows
mavlink-doctor.exe -transport serial -serial COM3 -baud 57600
```

### 4. TCP Companion Computers & Routers (MAVProxy, mavlink-router)
```bash
# Connecting to onboard companion computer router
mavlink-doctor -transport tcp -mode client -address 192.168.1.50:5760

# Listening for incoming TCP telemetry connections
mavlink-doctor -transport tcp -mode server -port 5760
```

### 5. Automated CI & Scripting (JSON Output)
```bash
mavlink-doctor -listen 5 -json > telemetry_audit.json
```

### 6. Exporting Field Support Reports with Redaction
Generate a sanitized diagnostic bundle to share with hardware vendors without revealing private internal IPs:
```bash
mavlink-doctor -listen 15 -export-report field_report.json -redact
```

---

## Command-Line Options

| Option | Default | Description |
| :--- | :---: | :--- |
| `-listen` | `10` | Duration in seconds for diagnostic receive tests. |
| `-transport` | `auto` | Transport protocol: `auto`, `udp`, `tcp`, `serial`, or `broadcast`. |
| `-mode` | `auto` | Endpoint mode: `auto`, `server`, or `client`. |
| `-address` | empty | Remote `host:port` for client connections. |
| `-listen-address` | `0.0.0.0` | Local IPv4 address to bind for server mode. |
| `-port` | `19856` | Local UDP/TCP port for server mode. |
| `-serial` | empty | Serial device path (e.g. `/dev/ttyUSB0`, `COM3`). |
| `-baud` | `57600` | Serial baud rate (`57600`, `115200`, `921600`, etc.). |
| `-dialect` | `common` | MAVLink dialect: `common`, `ardupilotmega`, `standard`, `all`, or `raw`. |
| `-gcs-heartbeat` | `true` | Emit diagnostic GCS heartbeat (System ID 255, Component ID 190). |
| `-request-streams`| `false`| Emit automatic telemetry data-stream requests. |
| `-stream-server` | `true` | Enable UDP stream reassembly server for chunked packets. |
| `-no-gomavlib` | `false` | Run raw socket tests only without MAVLink decoding. |
| `-probe-subnet` | empty | Opt-in bounded subnet probe for active endpoints (e.g. `192.168.1.0/24`). |
| `-json` | `false` | Output results in machine-readable JSON format. |
| `-export-report` | empty | Save detailed diagnostic support report to specified JSON file. |
| `-redact` | `false` | Sanitize IP and MAC addresses in exported support report. |
| `-version` | `false` | Print tool version, commit, build timestamp, and exit. |

---

## How to Interpret Diagnostic Output

1. **Local Bind vs. Network Traffic**:
   - `Local socket test bind succeeded` proves only that your computer can open the port. It does not prove a vehicle is sending data.
   - Evidence of actual communication requires `Decoded MAVLink frame(s) received`.
2. **Serial-over-UDP Framing**:
   - If raw packets are received but zero MAVLink frames decode, the air unit is splitting frames across datagram boundaries.
   - `mavlink-doctor` automatically reassembles stream chunks when `-stream-server=true` or client stream mode is used.
3. **Heartbeat Health**:
   - A single heartbeat confirms presence at a moment in time.
   - Repeated heartbeats with low interval jitter (e.g. ~1000ms ± 50ms) confirm a healthy, stable telemetry connection.
4. **Gimbal Observation**:
   - Gimbal confirmation requires `HEARTBEAT` (type gimbal), `GIMBAL_DEVICE_ATTITUDE_STATUS`, or `MOUNT_ORIENTATION`.
   - Missing gimbal messages during a short test is reported as **unobserved**, not as proof of absence.

---

## Development & CI/CD

### Building Locally
```bash
make all           # Run tidy, fmt, vet, test, and build
make test-race     # Run unit and integration tests with race detector
make cross-build   # Build release binaries for Linux, macOS, and Windows
```

### Conventional Commits
This repository strictly enforces [Conventional Commits](https://www.conventionalcommits.org/):
- `feat:` New features (e.g. `feat(transport): add udp broadcast endpoint`)
- `fix:` Bug fixes (e.g. `fix(metrics): correct sequence wrap gap math`)
- `docs:` Documentation updates
- `perf:` Performance improvements
- `ci:` CI/CD and build workflow modifications

Releases and changelogs are automatically generated via `git-cliff` and GitHub Actions.

---

## License

This project is licensed under the [MIT License](LICENSE).
