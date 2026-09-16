# Changelog

All notable changes to `mavlink-doctor` will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html) and [Conventional Commits](https://www.conventionalcommits.org/).

## [0.1.0] - 2026-09-16

### Features
- **core**: Modular production-grade architecture decoupling discovery, transport, decoders, metrics, reporters, and simulator into clean internal packages.
- **discovery**: Cross-platform network interface scanning, IPv4/IPv6 candidate resolution, and MTU extraction.
- **discovery**: Multi-OS UDP socket ownership inspection supporting Linux (`ss`), macOS (`lsof`), and Windows (`netstat`).
- **discovery**: Multi-OS routing table analysis and non-blocking ICMP reachability checks.
- **discovery**: Serial port candidate auto-discovery for Linux (`/dev/ttyUSB*`, `/dev/ttyACM*`), macOS (`/dev/cu.*`), and Windows (`COMx`).
- **discovery**: Opt-in bounded subnet probe (`-probe-subnet`) with concurrency control and strict timeouts.
- **transport**: Hardware-agnostic endpoint factory supporting UDP (server/client), TCP (server/client), Serial, and UDP Broadcast.
- **transport**: Chunked UDP stream reassembly client and server (`-stream-server`) allowing continuous byte-stream reconstruction across packet boundaries (e.g. 115-byte radio chunks).
- **decoder**: Dialect loader with selectable dialects: `common`, `ardupilotmega`, `standard`, `all`, and `raw`.
- **decoder**: Human-readable translation for Autopilots, Vehicle Types, System Status, and Component IDs.
- **decoder**: Specialized Gimbal payload detection supporting `HEARTBEAT` (type gimbal), `GIMBAL_DEVICE_ATTITUDE_STATUS`, `GIMBAL_MANAGER_STATUS`, `GIMBAL_DEVICE_INFORMATION`, and `MOUNT_ORIENTATION`.
- **metrics**: Real-time sequence gap detection and packet loss % calculation per source endpoint.
- **metrics**: Live throughput calculation (message rate Hz and data rate bytes/sec).
- **metrics**: Inter-packet arrival jitter tracking (exponential moving average).
- **metrics**: Periodic heartbeat statistics (arrival count, min/max/average intervals, jitter, and link health evaluation).
- **reporter**: Interactive terminal UI with clean status badges (✅, ⚠️, ❌), formatted metrics tables, and diagnostic recommendations.
- **reporter**: Machine-readable JSON output mode (`-json`) for automated toolchain integration.
- **reporter**: Structured diagnostic support report file export (`-export-report`) with optional IP/MAC address redaction (`-redact`).
- **simulator**: Built-in mock MAVLink sender and 115-byte chunked UDP simulator for offline testing and CI.
- **build**: Cross-platform multi-architecture build script (`build.sh`) and developer `Makefile`.
- **ci**: GitHub Actions CI workflow with latest action versions testing on Linux, macOS, and Windows.
- **release**: Automated GitHub release workflow with `git-cliff` conventional commits changelog generation.
