package retry

import (
	"math"
	"time"
)

// CalculateBackoff calculates the exponential backoff delay
// Formula: delay = base^attempts seconds
func CalculateBackoff(attempts int, base float64) time.Duration {
	if attempts < 0 {
		attempts = 0
	}
	if base < 1 {
		base = 2.0 // default base
	}

	// Calculate delay in seconds
	delaySeconds := math.Pow(base, float64(attempts))

	// Cap at reasonable maximum (e.g., 1 hour)
	const maxDelaySeconds = 3600
	if delaySeconds > maxDelaySeconds {
		delaySeconds = maxDelaySeconds
	}

	return time.Duration(delaySeconds) * time.Second
}

// NextRetryTime calculates when the next retry should occur
func NextRetryTime(attempts int, base float64) time.Time {
	delay := CalculateBackoff(attempts, base)
	return time.Now().Add(delay)
}

// GetNextRetryAt returns a pointer to the next retry time
func GetNextRetryAt(attempts int, base float64) *time.Time {
	t := NextRetryTime(attempts, base)
	return &t
}
