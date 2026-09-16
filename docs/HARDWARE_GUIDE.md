# Hardware Integration & Diagnostic Guide

`mavlink-doctor` is hardware-agnostic and supports telemetry connections across any radio link, air unit, companion computer, autopilot, or gimbal.

## Hardware Topologies

### 1. SIYI HM30 / MK15 / MK32 Air Units
- **Connection**: Ethernet / Wi-Fi to Ground Unit
- **Default IP**: Ground unit is usually `192.168.144.25` or `192.168.144.11`, Air unit `192.168.144.12`
- **Telemetry Port**: UDP `19856`
- **Gotcha**: Air unit reads serial MAVLink from the flight controller and forwards it in short 115-byte UDP datagrams. A MAVLink frame is frequently split across two datagrams!
- **Diagnostic Command**:
  ```bash
  mavlink-doctor -transport udp -mode client -address 192.168.144.12:19856
  ```
  Or if the air unit is configured to send datagrams to your PC:
  ```bash
  mavlink-doctor -transport udp -mode server -port 19856 -stream-server=true
  ```

### 2. Microhard P900 / pDDL / Nano
- **Connection**: Ethernet bridge or USB/RS232
- **Network Mode**: Point-to-point or Point-to-Multipoint IP link
- **Diagnostic Command**:
  ```bash
  mavlink-doctor -transport udp -mode server -port 14550
  ```

### 3. RFD900x / RFD868x / SiK Radios (Serial / FTDI)
- **Connection**: USB-to-UART FTDI cable or direct telemetry port
- **Default Baud**: Usually 57600 baud (ArduPilot default) or 115200 baud
- **Diagnostic Command**:
  ```bash
  # Linux
  mavlink-doctor -transport serial -serial /dev/ttyUSB0 -baud 57600

  # Windows
  mavlink-doctor.exe -transport serial -serial COM3 -baud 57600
  ```

### 4. CubePilot Herelink
- **Connection**: Wi-Fi hotspot or USB tethering from Ground Station
- **MAVLink Port**: UDP `14550` or `14551`
- **Diagnostic Command**:
  ```bash
  mavlink-doctor -transport udp -mode server -port 14550
  ```

### 5. Companion Computers (Raspberry Pi, Jetson Nano, Radxa)
- Running `mavlink-router`, `MAVProxy`, or `cmavnode`
- Often routes telemetry over TCP port `5760` or UDP port `14550`:
  ```bash
  # Connecting as TCP client to companion router
  mavlink-doctor -transport tcp -mode client -address 192.168.1.50:5760

  # Listening for companion router UDP forward
  mavlink-doctor -transport udp -mode server -port 14550
  ```

### 6. Payloads & Gimbals (Gremsy, SIYI, Viewpro, SimpleBGC)
- **MAVLink Gimbal Protocol v2**: Emits `GIMBAL_DEVICE_ATTITUDE_STATUS` and `GIMBAL_DEVICE_INFORMATION`.
- **System / Component IDs**: Usually Component ID `154` (Gimbal 1) or `155` (Gimbal 2).
- `mavlink-doctor` automatically highlights gimbal presence in the report. If no gimbal frames appear, ensure the flight controller has `MNT1_TYPE` configured or stream rates enabled with `-request-streams=true`.
