package publish

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"
)

var ErrRateLimited = errors.New("rate limit exceeded")

const RlnLimiterCapacity = 600
const RlnLimiterRefillInterval = 10 * time.Minute

// RlnRateLimiter is used to rate limit the outgoing messages,
// The capacity and refillInterval comes from RLN contract configuration.
type RlnRateLimiter struct {
	mu             sync.Mutex
	capacity       int
	tokens         int
	refillInterval time.Duration
	lastRefill     time.Time
	updateCh       chan BucketUpdate
}

// BucketUpdate includes the information that need to be persisted in database.
type BucketUpdate struct {
	RemainingTokens int
	LastRefill      time.Time
}

// NewRlnPublishRateLimiter creates a new rate limiter, starts with a full capacity bucket.
func NewRlnRateLimiter(capacity int, refillInterval time.Duration, availableTokens int, lastRefill time.Time, updateCh chan BucketUpdate) *RlnRateLimiter {
	return &RlnRateLimiter{
		capacity:       capacity,
		tokens:         availableTokens, // Start with a full bucket in the first run, then track the remaining tokens in storage
		refillInterval: refillInterval,
		lastRefill:     lastRefill,
		updateCh:       updateCh,
	}
}

// Allow checks if a token can be consumed, and refills the bucket if necessary
func (rl *RlnRateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Refill tokens if the refill interval has passed
	now := time.Now()
	if now.Sub(rl.lastRefill) >= rl.refillInterval {
		rl.tokens = rl.capacity
		rl.lastRefill = now
		rl.sendUpdate()
	}

	// Check if there are tokens available
	if rl.tokens > 0 {
		rl.tokens--
		rl.sendUpdate()
		return true
	}

	return false
}

// sendUpdate sends the latest token state to the update channel.
func (rl *RlnRateLimiter) sendUpdate() {
	rl.updateCh <- BucketUpdate{RemainingTokens: rl.tokens, LastRefill: rl.lastRefill}
}

func (rl *RlnRateLimiter) Check(ctx context.Context, logger *zap.Logger) error {
	if rl.Allow() {
		return nil
	}
	return ErrRateLimited
}
