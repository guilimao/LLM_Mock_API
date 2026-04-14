package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// FaultSimulator handles fault simulation during streaming
type FaultSimulator struct {
	rng *rand.Rand
}

// NewFaultSimulator creates a new fault simulator
func NewFaultSimulator() *FaultSimulator {
	return &FaultSimulator{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// ShouldTrigger checks if a fault should be triggered based on configuration
func (fs *FaultSimulator) ShouldTrigger(fault *FaultConfig) bool {
	if fault == nil {
		return false
	}
	return fs.rng.Float64() < fault.Probability
}

// ApplyFault applies a fault to data and returns the faulted data
func (fs *FaultSimulator) ApplyFault(data string, fault *FaultConfig, bytesSent int) (string, bool, error) {
	if fault == nil {
		return data, false, nil
	}
	
	// Check if we should delay triggering fault until certain bytes are sent
	if fault.AfterBytes > 0 && bytesSent < fault.AfterBytes {
		return data, false, nil
	}
	
	switch fault.Type {
	case FaultTypeDelay:
		return fs.applyDelay(data, fault)
	case FaultTypeInterrupt:
		return fs.applyInterrupt(data, fault)
	case FaultTypePacketLoss:
		return fs.applyPacketLoss(data, fault)
	case FaultTypeCorruption:
		return fs.applyCorruption(data, fault)
	case FaultTypePartialJSON:
		return fs.applyPartialJSON(data, fault)
	case FaultTypeMalformedJSON:
		return fs.applyMalformedJSON(data, fault)
	case FaultTypeTimeout:
		return fs.applyTimeout(data, fault)
	default:
		return data, false, nil
	}
}

func (fs *FaultSimulator) applyDelay(data string, fault *FaultConfig) (string, bool, error) {
	delay := fault.Duration
	if delay == 0 {
		delay = 2 * time.Second
	}
	time.Sleep(delay)
	return data, false, nil
}

func (fs *FaultSimulator) applyInterrupt(data string, fault *FaultConfig) (string, bool, error) {
	// Return data truncated at a random point to simulate interruption
	if len(data) > 0 {
		truncationPoint := fs.rng.Intn(len(data))
		return data[:truncationPoint], true, fmt.Errorf("connection interrupted")
	}
	return data, true, fmt.Errorf("connection interrupted")
}

func (fs *FaultSimulator) applyPacketLoss(data string, fault *FaultConfig) (string, bool, error) {
	// Randomly drop characters to simulate packet loss
	if len(data) == 0 {
		return data, false, nil
	}
	
	lossRate := fault.CorruptionLevel
	if lossRate == 0 {
		lossRate = 0.3
	}
	
	var result strings.Builder
	for _, ch := range data {
		if fs.rng.Float64() > lossRate {
			result.WriteRune(ch)
		}
	}
	
	return result.String(), false, nil
}

func (fs *FaultSimulator) applyCorruption(data string, fault *FaultConfig) (string, bool, error) {
	// Corrupt random characters in the data
	if len(data) == 0 {
		return data, false, nil
	}
	
	corruptionRate := fault.CorruptionLevel
	if corruptionRate == 0 {
		corruptionRate = 0.2
	}
	
	chars := []rune(data)
	for i := range chars {
		if fs.rng.Float64() < corruptionRate {
			// Replace with random character
			chars[i] = rune(fs.rng.Intn(256))
		}
	}
	
	return string(chars), false, nil
}

func (fs *FaultSimulator) applyPartialJSON(data string, fault *FaultConfig) (string, bool, error) {
	// Return incomplete JSON to simulate truncation during transmission
	if len(data) <= 10 {
		return data, false, nil
	}
	
	// Truncate at a random point
	truncationPoint := len(data) - fs.rng.Intn(len(data)/2) - 1
	truncated := data[:truncationPoint]
	
	// Try to make it look like valid partial JSON
	// Count braces and brackets
	openBraces := strings.Count(truncated, "{") - strings.Count(truncated, "}")
	openBrackets := strings.Count(truncated, "[") - strings.Count(truncated, "]")
	
	// Add closing braces/brackets but leave it incomplete
	for i := 0; i < openBraces; i++ {
		truncated += "}"
	}
	for i := 0; i < openBrackets; i++ {
		truncated += "]"
	}
	
	// Add incomplete field to make it truly partial
	truncated = truncated[:len(truncated)-1] + `,"incomplete_fie`
	
	return truncated, true, fmt.Errorf("partial json")
}

func (fs *FaultSimulator) applyMalformedJSON(data string, fault *FaultConfig) (string, bool, error) {
	// Corrupt the JSON structure
	malformedOptions := []string{
		// Missing quotes around keys
		`{id: "test", value: 123}`,
		// Trailing comma
		`{"id": "test", "value": 123,}`,
		// Unescaped quotes in string
		`{"message": "He said "hello" to me"}`,
		// Invalid escape sequence
		`{"path": "C:\\user\\test"}`,
		// Mismatched braces
		`{"id": "test", "nested": {"key": "value"}`,
		// Invalid unicode
		`{"text": "Hello\x00World"}`,
		// Control characters
		`{"text": "Hello` + "\n\r\t" + `World"}`,
	}
	
	idx := fs.rng.Intn(len(malformedOptions))
	return malformedOptions[idx], false, fmt.Errorf("malformed json")
}

func (fs *FaultSimulator) applyTimeout(data string, fault *FaultConfig) (string, bool, error) {
	delay := fault.Duration
	if delay == 0 {
		delay = 30 * time.Second
	}
	
	// Sleep for a long time to simulate timeout
	time.Sleep(delay)
	return "", true, fmt.Errorf("request timeout")
}

// StreamingFaultInjector injects faults during streaming
func (fs *FaultSimulator) StreamingFaultInjector(fault *FaultConfig, bytesSent *int) func(string) (string, error) {
	return func(data string) (string, error) {
		*bytesSent += len(data)
		
		if !fs.ShouldTrigger(fault) {
			return data, nil
		}
		
		modified, interrupted, err := fs.ApplyFault(data, fault, *bytesSent)
		if interrupted {
			return modified, err
		}
		return modified, err
	}
}

// NetworkSimulator simulates various network conditions
type NetworkSimulator struct {
	fs       *FaultSimulator
	latency  time.Duration
	jitter   time.Duration
	lossRate float64
}

// NewNetworkSimulator creates a network simulator
func NewNetworkSimulator(latency, jitter time.Duration, lossRate float64) *NetworkSimulator {
	return &NetworkSimulator{
		fs:       NewFaultSimulator(),
		latency:  latency,
		jitter:   jitter,
		lossRate: lossRate,
	}
}

// SimulateLatency simulates network latency with jitter
func (ns *NetworkSimulator) SimulateLatency() {
	if ns.latency > 0 {
		actualLatency := ns.latency
		if ns.jitter > 0 {
			jitterAmount := time.Duration(ns.fs.rng.Int63n(int64(ns.jitter*2))) - ns.jitter
			actualLatency += jitterAmount
		}
		if actualLatency > 0 {
			time.Sleep(actualLatency)
		}
	}
}

// ShouldDropPacket determines if a packet should be dropped
func (ns *NetworkSimulator) ShouldDropPacket() bool {
	return ns.fs.rng.Float64() < ns.lossRate
}

// RecoverySimulator simulates recovery scenarios
type RecoverySimulator struct {
	fs          *FaultSimulator
	retryCount  int
	maxRetries  int
	recoveryTime time.Duration
}

// NewRecoverySimulator creates a recovery simulator
func NewRecoverySimulator(maxRetries int, recoveryTime time.Duration) *RecoverySimulator {
	return &RecoverySimulator{
		fs:           NewFaultSimulator(),
		retryCount:   0,
		maxRetries:   maxRetries,
		recoveryTime: recoveryTime,
	}
}

// AttemptRecovery attempts to recover from a fault
func (rs *RecoverySimulator) AttemptRecovery() (bool, time.Duration) {
	if rs.retryCount >= rs.maxRetries {
		return false, 0
	}
	
	rs.retryCount++
	
	// Exponential backoff
	backoff := time.Duration(1<<uint(rs.retryCount-1)) * time.Second
	if backoff > rs.recoveryTime {
		backoff = rs.recoveryTime
	}
	
	// Add some jitter
	jitter := time.Duration(rs.fs.rng.Int63n(int64(backoff / 2)))
	backoff += jitter
	
	return true, backoff
}

// Reset resets the retry count
func (rs *RecoverySimulator) Reset() {
	rs.retryCount = 0
}

// FaultPresets provides common fault configurations
var FaultPresets = map[string]*FaultConfig{
	"mild_delay": {
		Type:        FaultTypeDelay,
		Probability: 0.3,
		Duration:    100 * time.Millisecond,
	},
	"heavy_delay": {
		Type:        FaultTypeDelay,
		Probability: 0.7,
		Duration:    2 * time.Second,
	},
	"intermittent": {
		Type:        FaultTypeInterrupt,
		Probability: 0.2,
		AfterBytes:  100,
	},
	"packet_loss": {
		Type:            FaultTypePacketLoss,
		Probability:     0.3,
		CorruptionLevel: 0.1,
	},
	"corrupted_data": {
		Type:            FaultTypeCorruption,
		Probability:     0.4,
		CorruptionLevel: 0.2,
	},
	"json_error": {
		Type:            FaultTypeMalformedJSON,
		Probability:     0.5,
		AfterBytes:      50,
	},
	"timeout": {
		Type:        FaultTypeTimeout,
		Probability: 0.1,
		Duration:    10 * time.Second,
	},
	"recovery_test": {
		Type:          FaultTypeInterrupt,
		Probability:   0.3,
		AfterBytes:    200,
		RecoveryAfter: 3 * time.Second,
	},
}

// CreateStreamChunkWithFault creates a stream chunk with potential faults applied
func CreateStreamChunkWithFault(chunk ChatStreamChunk, fault *FaultConfig, fs *FaultSimulator) (string, error) {
	data, err := json.Marshal(chunk)
	if err != nil {
		return "", err
	}
	
	if fault != nil && fs.ShouldTrigger(fault) {
		modified, _, err := fs.ApplyFault(string(data), fault, 0)
		if err != nil {
			return modified, err
		}
		data = []byte(modified)
	}
	
	return "data: " + string(data) + "\n\n", nil
}

// ValidateJSON checks if data is valid JSON
func ValidateJSON(data string) bool {
	var v interface{}
	return json.Unmarshal([]byte(data), &v) == nil
}

// AttemptJSONRepair tries to repair malformed JSON
func AttemptJSONRepair(data string) string {
	// Remove trailing commas
	data = strings.TrimSpace(data)
	data = strings.TrimSuffix(data, ",")
	
	// Close unclosed braces/brackets
	openBraces := strings.Count(data, "{") - strings.Count(data, "}")
	openBrackets := strings.Count(data, "[") - strings.Count(data, "]")
	
	for i := 0; i < openBraces; i++ {
		data += "}"
	}
	for i := 0; i < openBrackets; i++ {
		data += "]"
	}
	
	// Try to add missing quotes
	if !strings.HasPrefix(data, "{") && !strings.HasPrefix(data, "[") {
		data = "{\"content\":\"" + data + "\"}"
	}
	
	return data
}
