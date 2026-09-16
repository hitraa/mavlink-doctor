package metrics

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/bluenviron/gomavlib/v4/pkg/frame"
)

// Tracker coordinates real-time statistics calculation across all received frames.
type Tracker struct {
	mu               sync.RWMutex
	startTime        time.Time
	totalFrames      uint64
	totalBytes       uint64
	totalParseErrors uint64
	versionCounts    map[string]uint64
	messageCounts    map[string]uint64
	sources          map[EndpointKey]*SourceMetrics
	sysLastSeq       map[uint8]uint8
	sysHasSeq        map[uint8]bool
}

// NewTracker initializes a new telemetry metrics tracker.
func NewTracker() *Tracker {
	return &Tracker{
		startTime:     time.Now(),
		versionCounts: make(map[string]uint64),
		messageCounts: make(map[string]uint64),
		sources:       make(map[EndpointKey]*SourceMetrics),
		sysLastSeq:    make(map[uint8]uint8),
		sysHasSeq:     make(map[uint8]bool),
	}
}

// RecordParseError increments the parse error counter.
func (t *Tracker) RecordParseError() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.totalParseErrors++
}

// RecordFrame processes an incoming decoded MAVLink frame and updates telemetry metrics.
func (t *Tracker) RecordFrame(fr frame.Frame, versionStr string, approxBytes int) *SourceMetrics {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	t.totalFrames++
	t.totalBytes += uint64(approxBytes)
	t.versionCounts[versionStr]++

	msg := fr.GetMessage()
	msgName := fmt.Sprintf("%T", msg)
	t.messageCounts[msgName]++

	sysID := fr.GetSystemID()
	compID := fr.GetComponentID()
	key := EndpointKey{SystemID: sysID, ComponentID: compID}

	sm, exists := t.sources[key]
	if !exists {
		sm = &SourceMetrics{
			SystemID:        sysID,
			ComponentID:     compID,
			FirstSeen:       now,
			LastSeen:        now,
			LastArrivalTime: now,
		}
		t.sources[key] = sm
	}

	sm.TotalFrames++
	sm.TotalBytes += uint64(approxBytes)
	sm.LastSeen = now

	// Sequence gap & packet loss tracking across the link channel for this system
	currentSeq := fr.GetSequenceNumber()
	if t.sysHasSeq[sysID] {
		expectedSeq := byte((int(t.sysLastSeq[sysID]) + 1) % 256)
		if currentSeq != expectedSeq {
			diff := (256 + int(currentSeq) - int(t.sysLastSeq[sysID]) - 1) % 256
			sm.DroppedFrames += uint64(diff)
		}

		// Calculate inter-frame arrival jitter
		interval := now.Sub(sm.LastArrivalTime)
		if sm.LastInterval > 0 {
			diff := math.Abs(float64(interval-sm.LastInterval) / float64(time.Millisecond))
			if sm.JitterMs == 0 {
				sm.JitterMs = diff
			} else {
				// Exponential moving average for jitter
				sm.JitterMs = 0.8*sm.JitterMs + 0.2*diff
			}
		}
		sm.LastInterval = interval
	}

	sm.LastSeq = currentSeq
	sm.HasLastSeq = true
	sm.LastArrivalTime = now

	t.sysLastSeq[sysID] = currentSeq
	t.sysHasSeq[sysID] = true

	// Total expected = received + dropped
	totalExpected := sm.TotalFrames + sm.DroppedFrames
	if totalExpected > 0 {
		sm.PacketLossPct = (float64(sm.DroppedFrames) / float64(totalExpected)) * 100.0
	}

	elapsed := now.Sub(sm.FirstSeen).Seconds()
	if elapsed > 0.05 {
		sm.MsgRateHz = float64(sm.TotalFrames) / elapsed
		sm.DataRateBytesSec = float64(sm.TotalBytes) / elapsed
	}

	return sm
}

// RecordHeartbeat updates dedicated heartbeat statistics for a source.
func (t *Tracker) RecordHeartbeat(sysID, compID uint8, autopilot, vehicle, state string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	key := EndpointKey{SystemID: sysID, ComponentID: compID}
	sm, exists := t.sources[key]
	if !exists {
		sm = &SourceMetrics{
			SystemID:        sysID,
			ComponentID:     compID,
			FirstSeen:       now,
			LastSeen:        now,
			LastArrivalTime: now,
		}
		t.sources[key] = sm
	}

	if sm.HeartbeatStats == nil {
		sm.HeartbeatStats = &HeartbeatStats{
			Count:         1,
			FirstArrival:  now,
			LastArrival:   now,
			MinIntervalMs: 999999.0,
			MaxIntervalMs: 0.0,
			Autopilot:     autopilot,
			VehicleType:   vehicle,
			SystemStatus:  state,
		}
		return
	}

	hb := sm.HeartbeatStats
	hb.Count++
	hb.Autopilot = autopilot
	hb.VehicleType = vehicle
	hb.SystemStatus = state

	intervalMs := float64(now.Sub(hb.LastArrival).Microseconds()) / 1000.0
	hb.LastArrival = now

	if intervalMs < hb.MinIntervalMs {
		hb.MinIntervalMs = intervalMs
	}
	if intervalMs > hb.MaxIntervalMs {
		hb.MaxIntervalMs = intervalMs
	}

	totalTimeSec := now.Sub(hb.FirstArrival).Seconds()
	if totalTimeSec > 0 && hb.Count > 1 {
		hb.AvgIntervalMs = (totalTimeSec * 1000.0) / float64(hb.Count-1)
		hb.RateHz = float64(hb.Count-1) / totalTimeSec
		// Heartbeat is considered healthy if at least 2 heartbeats arrived and average rate is between 0.5Hz and 5Hz
		hb.IsHealthy = (hb.Count >= 2) && (hb.AvgIntervalMs >= 200 && hb.AvgIntervalMs <= 2500)
	}
}

// Snapshot returns a copy of current global metrics.
func (t *Tracker) Snapshot() GlobalMetrics {
	t.mu.RLock()
	defer t.mu.RUnlock()

	now := time.Now()
	elapsed := now.Sub(t.startTime).Seconds()

	vCopy := make(map[string]uint64, len(t.versionCounts))
	for k, v := range t.versionCounts {
		vCopy[k] = v
	}

	mCopy := make(map[string]uint64, len(t.messageCounts))
	for k, v := range t.messageCounts {
		mCopy[k] = v
	}

	sCopy := make(map[string]*SourceMetrics, len(t.sources))
	for k, v := range t.sources {
		smVal := *v
		if v.HeartbeatStats != nil {
			hbVal := *v.HeartbeatStats
			smVal.HeartbeatStats = &hbVal
		}
		keyStr := fmt.Sprintf("%d/%d", k.SystemID, k.ComponentID)
		sCopy[keyStr] = &smVal
	}

	var overallThroughput float64
	var overallMsgRate float64
	if elapsed > 0.05 {
		overallThroughput = float64(t.totalBytes) / elapsed
		overallMsgRate = float64(t.totalFrames) / elapsed
	}

	return GlobalMetrics{
		StartTime:            t.startTime,
		DurationSeconds:      elapsed,
		TotalFrames:          t.totalFrames,
		TotalBytes:           t.totalBytes,
		TotalParseErrors:     t.totalParseErrors,
		VersionCounts:        vCopy,
		MessageCounts:        mCopy,
		Sources:              sCopy,
		OverallThroughputBps: overallThroughput,
		OverallMsgRateHz:     overallMsgRate,
	}
}

// MessageStats returns top received message types.
func (t *Tracker) MessageStats() map[string]uint64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	res := make(map[string]uint64, len(t.messageCounts))
	for k, v := range t.messageCounts {
		res[k] = v
	}
	return res
}
