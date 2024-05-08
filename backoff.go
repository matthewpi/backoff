// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package backoff

import (
	"context"
	"math"
	"time"
)

// maxInt64 is used to avoid overflowing a time.Duration (int64) value.
const maxInt64 = float64(math.MaxInt64 - 512)

// Backoff represents an exponential backoff.
type Backoff struct {
	// n is the current attempt and defaults to 0. The first attempt will not
	// be delayed before it runs.
	n uint

	// MaxAttempts is the max number of attempts that can occur. If set to 0
	// the number of attempts will not be limited.
	MaxAttempts uint
	// Factor is the factor at which Min will increase after each failed attempt.
	Factor float64
	// Min is the initial backoff time to wait after the first failed attempt.
	Min time.Duration
	// Max is the maximum time to wait before retrying.
	Max time.Duration

	// Timer is used for mocking in unit tests. For normal use, this should
	// always be set to the result of `NewRealTimer()`, if you are creating
	// a Backoff using the `New` function, this will be set by default.
	Timer Timer
}

// New returns a new Backoff instance.
func New(maxAttempts uint, factor float64, min, max time.Duration) *Backoff {
	return &Backoff{
		n: 0,

		MaxAttempts: maxAttempts,
		Factor:      factor,
		Min:         min,
		Max:         max,

		Timer: NewRealTimer(),
	}
}

// Attempt returns the current attempt.
func (b *Backoff) Attempt() uint {
	return b.n
}

// Duration returns the duration to wait for the current attempt. Useful for
// logging when the next attempt will occur.
func (b *Backoff) Duration() time.Duration {
	return b.duration(b.n)
}

// duration returns the time.Duration to wait before running the given attempt.
func (b *Backoff) duration(attempt uint) time.Duration {
	// The first attempt should never have a delay.
	if attempt == 0 {
		return 0
	}

	factor := math.Pow(b.Factor, float64(attempt))
	durF := float64(b.Min) * factor
	if durF > maxInt64 {
		return b.Max
	}

	dur := time.Duration(durF)
	if dur < b.Min {
		return b.Min
	}
	if dur > b.Max {
		return b.Max
	}
	return dur
}

// Next increments the attempt, then waits for the duration of the attempt.
// Once the duration has passed, Next returns true. Next will return false if
// the attempt will exceed the MaxAttempts limit or if the given context has
// been cancelled.
//
// This function was designed to be used as follows:
//
//	for b.Next(ctx) {
//		// Do work, `continue` on soft-failure, `break` on success or non-retryable error.
//	}
func (b *Backoff) Next(ctx context.Context) bool {
	if b.MaxAttempts != 0 && b.n >= b.MaxAttempts {
		return false
	}
	d := b.Duration()
	b.n++

	// If the duration is zero, bypass the timer.
	if d == 0 {
		select {
		case <-ctx.Done():
			return false
		default:
			return true
		}
	}

	b.Timer.Start(d)
	select {
	case <-ctx.Done():
		// Stop the timer to release resources and prevent it from sending to a
		// channel we are not listening to anymore.
		if !b.Timer.Stop() {
			// Drain the channel as per Go's documentation.
			<-b.Timer.C()
		}
		return false
	case <-b.Timer.C():
		return true
	}
}

// Reset resets the backoff back to 0, so it can be re-used.
func (b *Backoff) Reset() {
	b.n = 0
}
