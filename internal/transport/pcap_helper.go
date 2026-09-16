package transport

import "fmt"

// FormatTcpdumpCommand returns copy-pasteable tcpdump command.
func FormatTcpdumpCommand(port int, iface string) string {
	if iface == "" {
		iface = "any"
	}
	return fmt.Sprintf("sudo tcpdump -ni %s udp port %d -vv -X", iface, port)
}

// FormatTsharkCommand returns copy-pasteable tshark command with mavlink dissector.
func FormatTsharkCommand(port int) string {
	return fmt.Sprintf("tshark -f \"udp port %d\" -d udp.port==%d,mavlink_proto -O mavlink_proto", port, port)
}
