package decoder

import (
	"fmt"

	"github.com/bluenviron/gomavlib/v4/pkg/dialects/common"
	"github.com/bluenviron/gomavlib/v4/pkg/message"
)

// GimbalEvidence records observed gimbal indicators.
type GimbalEvidence struct {
	Observed      bool     `json:"observed"`
	GimbalType    string   `json:"gimbal_type,omitempty"`
	SystemID      uint8    `json:"system_id,omitempty"`
	ComponentID   uint8    `json:"component_id,omitempty"`
	MessagesSeen  []string `json:"messages_seen"`
	AttitudeRoll  float32  `json:"attitude_roll,omitempty"`
	AttitudePitch float32  `json:"attitude_pitch,omitempty"`
	AttitudeYaw   float32  `json:"attitude_yaw,omitempty"`
	Details       string   `json:"details,omitempty"`
}

// CheckGimbalMessage inspects a MAVLink message for gimbal presence.
func CheckGimbalMessage(msg message.Message, sysID, compID uint8, evidence *GimbalEvidence) bool {
	if evidence.MessagesSeen == nil {
		evidence.MessagesSeen = make([]string, 0)
	}

	found := false
	msgName := fmt.Sprintf("%T", msg)

	switch typed := msg.(type) {
	case *common.MessageHeartbeat:
		if typed.Type == common.MAV_TYPE_GIMBAL || compID == 154 || compID == 155 {
			found = true
			evidence.Observed = true
			evidence.SystemID = sysID
			evidence.ComponentID = compID
			evidence.GimbalType = "Payload Gimbal via HEARTBEAT"
			evidence.Details = fmt.Sprintf("Gimbal heartbeat received from sys=%d comp=%d", sysID, compID)
			appendUnique(&evidence.MessagesSeen, "HEARTBEAT(MAV_TYPE_GIMBAL)")
		}

	case *common.MessageGimbalDeviceAttitudeStatus:
		found = true
		evidence.Observed = true
		evidence.SystemID = sysID
		evidence.ComponentID = compID
		evidence.GimbalType = "MAVLink v2 Gimbal Protocol v2"
		evidence.Details = fmt.Sprintf("Gimbal device attitude reported (target sys=%d comp=%d flags=0x%X)",
			typed.TargetSystem, typed.TargetComponent, typed.Flags)
		appendUnique(&evidence.MessagesSeen, "GIMBAL_DEVICE_ATTITUDE_STATUS")

	case *common.MessageGimbalDeviceInformation:
		found = true
		evidence.Observed = true
		evidence.SystemID = sysID
		evidence.ComponentID = compID
		evidence.GimbalType = fmt.Sprintf("Gimbal: %s (Vendor: %s)", typed.ModelName, typed.VendorName)
		evidence.Details = fmt.Sprintf("Hardware UID: %d", typed.Uid)
		appendUnique(&evidence.MessagesSeen, "GIMBAL_DEVICE_INFORMATION")

	case *common.MessageGimbalManagerStatus:
		found = true
		evidence.Observed = true
		evidence.SystemID = sysID
		evidence.ComponentID = compID
		evidence.Details = fmt.Sprintf("Gimbal Manager sys=%d comp=%d device_id=%d", sysID, compID, typed.GimbalDeviceId)
		appendUnique(&evidence.MessagesSeen, "GIMBAL_MANAGER_STATUS")

	case *common.MessageGimbalManagerInformation:
		found = true
		evidence.Observed = true
		evidence.SystemID = sysID
		evidence.ComponentID = compID
		evidence.Details = fmt.Sprintf("Gimbal Manager Cap Flags=0x%X", typed.CapFlags)
		appendUnique(&evidence.MessagesSeen, "GIMBAL_MANAGER_INFORMATION")

	case *common.MessageMountOrientation:
		found = true
		evidence.Observed = true
		evidence.SystemID = sysID
		evidence.ComponentID = compID
		evidence.GimbalType = "Mount Orientation Protocol"
		evidence.Details = fmt.Sprintf("Mount roll=%.1f pitch=%.1f yaw=%.1f", typed.Roll, typed.Pitch, typed.Yaw)
		appendUnique(&evidence.MessagesSeen, "MOUNT_ORIENTATION")
	default:
		// Check by component ID if not already matched
		if (compID == 154 || compID == 155) && !found {
			found = true
			evidence.Observed = true
			evidence.SystemID = sysID
			evidence.ComponentID = compID
			appendUnique(&evidence.MessagesSeen, msgName)
		}
	}

	return found
}

func appendUnique(slice *[]string, val string) {
	for _, item := range *slice {
		if item == val {
			return
		}
	}
	*slice = append(*slice, val)
}
