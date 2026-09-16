# Architecture & Technical Design

`mavlink-doctor` is an extensible, transport-neutral diagnostic instrument and telemetry metrics engine for MAVLink networks.

```
                  +----------------------------------------------+
                  |               CLI Entrypoint                 |
                  |           (cmd/mavlink-doctor)               |
                  +----------------------+-----------------------+
                                         |
     +-------------------+---------------+-------------------+
     |                   |                                   |
+----+----+     +--------+--------+                 +--------+--------+
|Discovery|     |    Transport    |                 |     Reporter    |
| Package |     |     Package     |                 |     Package     |
+----+----+     +--------+--------+                 +--------+--------+
     |                   |                                   ^
     |          +--------+--------+                          |
     |          | MAVLink Node    |                          |
     |          | (gomavlib v4)   |                          |
     |          +--------+--------+                          |
     |                   |                                   |
     |          +--------+--------+        +-----------------+
     |          | Decoder Package +------->| Metrics Package |
     |          | (Enums, Gimbal) |        | (Gaps, Jitter)  |
     |          +-----------------+        +-----------------+
```

## Core Modules

### 1. Discovery Subsystem (`internal/discovery`)
- **Network Interfaces**: Inspects host adapters (`net.Interfaces`), flags (UP, BROADCAST, MULTICAST), MTU, and candidate IPv4/IPv6 addresses.
- **Port Availability**: Uses platform-native tools (Linux `ss`/`netstat`, macOS `lsof`, Windows `netstat`) to verify if another process owns the target port.
- **Reachability & Routing**: Determines direct local links vs gateway hops (Linux `ip route get`, macOS `route get`, Windows `route print`) and tests reachability with non-blocking ICMP pings.
- **Serial Discovery**: Dynamically scans OS serial device patterns (`/dev/ttyUSB*`, `/dev/ttyACM*`, `/dev/cu.*`, Windows COM ports).
- **Subnet Prober**: Bounded, rate-limited active UDP probe for candidate vehicles across an IPv4 CIDR block.

### 2. Transport & Reassembly (`internal/transport`)
- Supports **UDP** (server and client), **TCP** (server and client), **Serial** (client), and **UDP Broadcast**.
- **Chunked UDP Reassembly**: Common air units (e.g. SIYI HM30/MK15, Microhard, RFD900) package serial byte streams into fixed-length UDP datagrams (often 115 bytes). If a MAVLink frame is split across datagram boundaries, standard datagram sockets fail to decode it. `mavlink-doctor` provides custom stream clients and stream servers that demux packets per remote sender and reassemble continuous byte streams before feeding the MAVLink parser.
- **Raw Sniffer**: Captures raw datagrams and profiles packet size distributions, arrival timestamps, and magic byte markers (`0xFE` for MAVLink 1, `0xFD` for MAVLink 2).

### 3. Decoder & Enum Engine (`internal/decoder`)
- **Dialect Management**: Runtime selection between `common`, `ardupilotmega`, `standard`, `all`, or `raw`.
- **Enum Translation**: Converts numeric protocol enums into human-readable strings:
  - Autopilots: ArduPilot, PX4, INAV, Paparazzi, etc.
  - Vehicles: Quadrotor, Hexarotor, Fixed Wing, Helicopter, VTOL, Rover, Boat, Submarine, Antenna Tracker, Gimbal.
  - States: Boot, Calibrating, Standby, Active, Critical, Emergency, Poweroff.
  - Components: Flight Controller, Companion Computer, Camera, Gimbal, ADSB, Obstacle Avoidance.
- **Gimbal Engine**: Detects gimbal evidence across multiple protocol specifications:
  - MAV_TYPE_GIMBAL in `HEARTBEAT`
  - Component ID 154 / 155
  - `GIMBAL_DEVICE_ATTITUDE_STATUS` (Gimbal Protocol v2)
  - `GIMBAL_DEVICE_INFORMATION`
  - `GIMBAL_MANAGER_STATUS`
  - `MOUNT_ORIENTATION`

### 4. Telemetry Metrics Engine (`internal/metrics`)
- **Sequence Gap Analysis**: Tracks sequence numbers modulo 256 for each distinct `(SystemID, ComponentID)` stream. Computes exact dropped frame counts and packet loss percentage.
- **Throughput Profiling**: Computes real-time message frequency (Hz) and byte throughput (bytes/sec).
- **Jitter Calculation**: Computes exponential moving average of inter-frame arrival variance.
- **Heartbeat Stability**: Tracks heartbeat arrival intervals, minimum/maximum intervals, average period, and marks link health as healthy, unstable, or irregular.

### 5. Reporters & Support Bundles (`internal/reporter`)
- **Interactive Terminal**: Colorized, structured output with clear status indicators and copy-pasteable `tcpdump` / `tshark` debugging commands.
- **Machine-Readable JSON**: Complete diagnostic tree exported for CI pipelines, automated QA rigs, and flight line scripts.
- **Redacted Support Export**: Generates sanitized diagnostic reports (`-redact`) masking private IP and MAC addresses for sharing with vendors or open-source forums.
