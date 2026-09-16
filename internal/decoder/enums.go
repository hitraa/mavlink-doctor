package decoder

import "fmt"

// AutopilotName returns human readable name for MAV_AUTOPILOT enum.
func AutopilotName(val uint8) string {
	switch val {
	case 0:
		return "Generic / Unknown (MAV_AUTOPILOT_GENERIC)"
	case 1:
		return "PIXHAWK / PX4"
	case 2:
		return "SLUGS"
	case 3:
		return "ArduPilot (ArduCopter/ArduPlane/Rover/Sub)"
	case 4:
		return "OpenPilot"
	case 5:
		return "Generic Mission Computer"
	case 6:
		return "FlexiPilot"
	case 7:
		return "AutoQuad"
	case 8:
		return "UDB"
	case 9:
		return "FP-Optimizer"
	case 10:
		return "Armazila"
	case 11:
		return "Aerob"
	case 12:
		return "PX4 Autopilot"
	case 13:
		return "SMACCMPilot"
	case 14:
		return "AutoTeam"
	case 15:
		return "Freefall Clipper"
	case 16:
		return "InertiaLabs"
	case 17:
		return "INAIR"
	case 18:
		return "Inav / Cleanflight / Betaflight"
	default:
		return fmt.Sprintf("Custom / Other (%d)", val)
	}
}

// VehicleTypeName returns human readable name for MAV_TYPE enum.
func VehicleTypeName(val uint8) string {
	switch val {
	case 0:
		return "Generic Micro Air Vehicle (MAV_TYPE_GENERIC)"
	case 1:
		return "Fixed Wing Aircraft"
	case 2:
		return "Quadrotor Multirotor"
	case 3:
		return "Coaxial Helicopter"
	case 4:
		return "Helicopter (Normal single rotor)"
	case 5:
		return "Ground Installation / Antenna Tracker"
	case 6:
		return "Ground Control Station (GCS)"
	case 7:
		return "Airship / Blimp"
	case 8:
		return "Free Balloon"
	case 9:
		return "Rocket"
	case 10:
		return "Ground Rover"
	case 11:
		return "Surface Boat"
	case 12:
		return "Submarine / Underwater ROV"
	case 13:
		return "Hexarotor Multirotor"
	case 14:
		return "Octorotor Multirotor"
	case 15:
		return "Tricopter"
	case 16:
		return "Flapping Wing Ornithopter"
	case 17:
		return "Kite"
	case 18:
		return "Onboard Companion Computer"
	case 19:
		return "Two-Rotor VTOL (Tailsitter / Tiltrotor)"
	case 20:
		return "Quad-Rotor VTOL"
	case 21:
		return "Tiltrotor VTOL"
	case 26:
		return "Payload Gimbal"
	case 27:
		return "Onboard ADSB Transponder"
	case 30:
		return "Camera"
	case 31:
		return "Charging Station"
	default:
		return fmt.Sprintf("Specialized / Unknown (%d)", val)
	}
}

// SystemStatusName returns human readable name for MAV_STATE enum.
func SystemStatusName(val uint8) string {
	switch val {
	case 0:
		return "UNINIT (Uninitialized)"
	case 1:
		return "BOOT (System booting)"
	case 2:
		return "CALIBRATING (Sensors calibrating)"
	case 3:
		return "STANDBY (Grounded, disarmed, ready)"
	case 4:
		return "ACTIVE (Armed and operational)"
	case 5:
		return "CRITICAL (Failsafe or critical alarm active)"
	case 6:
		return "EMERGENCY (System shutdown / emergency landing)"
	case 7:
		return "POWEROFF (Powering off)"
	case 8:
		return "TERMINATION (Flight termination)"
	default:
		return fmt.Sprintf("Unknown state (%d)", val)
	}
}

// ComponentName returns standard MAVLink component name for common IDs.
func ComponentName(compID uint8) string {
	switch compID {
	case 1:
		return "Autopilot 1 (Flight Controller)"
	case 2:
		return "Autopilot 2 (Redundant FC)"
	case 100:
		return "Camera 1"
	case 101:
		return "Camera 2"
	case 140:
		return "Servo Actuator 1"
	case 154:
		return "Gimbal 1"
	case 155:
		return "Gimbal 2"
	case 156:
		return "ADSB Transponder"
	case 158:
		return "Precision Landing / Beacon"
	case 190:
		return "GCS (Ground Control Station)"
	case 191:
		return "Companion Computer 1"
	case 192:
		return "Companion Computer 2"
	case 196:
		return "Obstacle Avoidance System"
	case 200:
		return "Laser Scanner / Lidar"
	case 240:
		return "Telemetry Radio / Air Unit"
	default:
		return fmt.Sprintf("Component %d", compID)
	}
}
