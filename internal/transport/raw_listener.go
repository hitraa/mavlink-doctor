package transport

import (
	"encoding/hex"
	"fmt"
	"net"
	"time"

	"github.com/khairnar2960/mavlink-doctor/internal/decoder"
)

// PacketSummary stores raw observation metrics from the raw UDP socket listener.
type PacketSummary struct {
	ListenAddress string        `json:"listen_address"`
	Port          int           `json:"port"`
	Duration      time.Duration `json:"duration"`
	TotalPackets  int           `json:"total_packets"`
	TotalBytes    int           `json:"total_bytes"`
	MAVLinkLike   int           `json:"mavlink_like_packets"`
	UniqueSenders []string      `json:"unique_senders"`
	PacketSizes   []int         `json:"sample_packet_sizes,omitempty"`
	ErrorMessage  string        `json:"error,omitempty"`
}

// RawPacketCallback is invoked on every received UDP packet.
type RawPacketCallback func(timestamp time.Time, remoteAddr string, bytesReceived int, hexPreview string, isMavlink bool)

// ListenRawUDP listens on UDP port for duration and analyzes packets.
func ListenRawUDP(duration time.Duration, listenAddress string, port int, cb RawPacketCallback) (*PacketSummary, error) {
	summary := &PacketSummary{
		ListenAddress: listenAddress,
		Port:          port,
		Duration:      duration,
		UniqueSenders: make([]string, 0),
		PacketSizes:   make([]int, 0),
	}

	conn, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.ParseIP(listenAddress),
		Port: port,
	})
	if err != nil {
		summary.ErrorMessage = err.Error()
		return summary, fmt.Errorf("unable to listen on raw UDP socket %s:%d: %w", listenAddress, port, err)
	}
	defer conn.Close()

	_ = conn.SetReadDeadline(time.Now().Add(duration))
	buf := make([]byte, 65535)
	sendersMap := make(map[string]struct{})

	for {
		n, remote, err := conn.ReadFromUDP(buf)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				break
			}
			summary.ErrorMessage = err.Error()
			break
		}

		summary.TotalPackets++
		summary.TotalBytes += n

		if len(summary.PacketSizes) < 10 {
			summary.PacketSizes = append(summary.PacketSizes, n)
		}

		senderStr := remote.String()
		if _, exists := sendersMap[senderStr]; !exists {
			sendersMap[senderStr] = struct{}{}
			summary.UniqueSenders = append(summary.UniqueSenders, senderStr)
		}

		data := buf[:n]
		isMav := decoder.LooksLikeMAVLink(data)
		if isMav {
			summary.MAVLinkLike++
		}

		if cb != nil {
			previewLen := n
			if previewLen > 24 {
				previewLen = 24
			}
			previewHex := hex.EncodeToString(data[:previewLen])
			if previewLen < n {
				previewHex += "..."
			}
			cb(time.Now(), senderStr, n, previewHex, isMav)
		}
	}

	return summary, nil
}
