package decoder

import (
	"bytes"

	"github.com/bluenviron/gomavlib/v4/pkg/frame"
)

const (
	MagicV1 = 0xFE
	MagicV2 = 0xFD
)

// FrameVersion returns human-readable version of the decoded frame.
func FrameVersion(fr frame.Frame) string {
	switch fr.(type) {
	case *frame.V1Frame:
		return "MAVLink 1"
	case *frame.V2Frame:
		return "MAVLink 2"
	default:
		return "Unknown MAVLink version"
	}
}

// MagicByteName returns description of first byte.
func MagicByteName(b byte) string {
	switch b {
	case MagicV1:
		return "MAVLink 1 (0xFE)"
	case MagicV2:
		return "MAVLink 2 (0xFD)"
	default:
		return "Non-MAVLink"
	}
}

// LooksLikeMAVLink checks if packet starts with MAVLink v1/v2 magic byte.
func LooksLikeMAVLink(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	return data[0] == MagicV1 || data[0] == MagicV2
}

// FindMagicOffset locates the index of the first MAVLink 1 or 2 magic byte in buffer.
// Returns -1 if not found.
func FindMagicOffset(data []byte) int {
	for i, b := range data {
		if b == MagicV1 || b == MagicV2 {
			return i
		}
	}
	return -1
}

// CountMagicBytes counts occurrences of 0xFE and 0xFD inside arbitrary buffer.
func CountMagicBytes(data []byte) (v1Count int, v2Count int) {
	v1Count = bytes.Count(data, []byte{MagicV1})
	v2Count = bytes.Count(data, []byte{MagicV2})
	return
}
