package metrics

import "time"

// EndpointKey uniquely identifies a MAVLink source (SystemID / ComponentID).
type EndpointKey struct {
	SystemID    uint8
	ComponentID uint8
}

// SourceMetrics tracks telemetry statistics for a specific (SystemID, ComponentID).
type SourceMetrics struct {
	SystemID         uint8           `json:"system_id"`
	ComponentID      uint8           `json:"component_id"`
	FirstSeen        time.Time       `json:"first_seen"`
	LastSeen         time.Time       `json:"last_seen"`
	TotalFrames      uint64          `json:"total_frames"`
	TotalBytes       uint64          `json:"total_bytes"`
	DroppedFrames    uint64          `json:"dropped_frames"`
	PacketLossPct    float64         `json:"packet_loss_pct"`
	MsgRateHz        float64         `json:"msg_rate_hz"`
	DataRateBytesSec float64         `json:"data_rate_bytes_sec"`
	JitterMs         float64         `json:"jitter_ms"`
	LastSeq          uint8           `json:"last_seq"`
	HasLastSeq       bool            `json:"-"`
	LastArrivalTime  time.Time       `json:"-"`
	LastInterval     time.Duration   `json:"-"`
	HeartbeatStats   *HeartbeatStats `json:"heartbeat_stats,omitempty"`
}

// HeartbeatStats records heartbeat frequency, intervals, and stability.
type HeartbeatStats struct {
	Count         uint64    `json:"count"`
	FirstArrival  time.Time `json:"first_arrival"`
	LastArrival   time.Time `json:"last_arrival"`
	MinIntervalMs float64   `json:"min_interval_ms"`
	MaxIntervalMs float64   `json:"max_interval_ms"`
	AvgIntervalMs float64   `json:"avg_interval_ms"`
	JitterMs      float64   `json:"jitter_ms"`
	RateHz        float64   `json:"rate_hz"`
	IsHealthy     bool      `json:"is_healthy"`
	Autopilot     string    `json:"autopilot,omitempty"`
	VehicleType   string    `json:"vehicle_type,omitempty"`
	SystemStatus  string    `json:"system_status,omitempty"`
}

// GlobalMetrics captures cumulative metrics across all streams.
type GlobalMetrics struct {
	StartTime            time.Time                 `json:"start_time"`
	DurationSeconds      float64                   `json:"duration_seconds"`
	TotalFrames          uint64                    `json:"total_frames"`
	TotalBytes           uint64                    `json:"total_bytes"`
	TotalParseErrors     uint64                    `json:"total_parse_errors"`
	VersionCounts        map[string]uint64         `json:"version_counts"`
	MessageCounts        map[string]uint64         `json:"message_counts"`
	Sources              map[string]*SourceMetrics `json:"sources"`
	OverallThroughputBps float64                   `json:"overall_throughput_bps"`
	OverallMsgRateHz     float64                   `json:"overall_msg_rate_hz"`
}
