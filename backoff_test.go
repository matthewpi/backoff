// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package backoff_test

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/matthewpi/backoff"
)

const (
	_maxAttempts uint    = 3
	_factor      float64 = 2
	_min                 = 1 * time.Second
	_max                 = 5 * time.Second
)

func newBackoffWithMockTimer(maxAttempts uint, factor float64, min, max time.Duration) *backoff.Backoff {
	b := backoff.New(maxAttempts, factor, min, max)
	b.Timer = newMockTimer()
	return b
}

func TestNew(t *testing.T) {
	b := newBackoffWithMockTimer(_maxAttempts, _factor, _min, _max)
	if b == nil {
		t.Error("expected backoff to not be nil")
		return
	}

	for i, tc := range []struct {
		field  string
		expect any
		value  any
	}{
		{
			field:  "MaxAttempts",
			expect: _maxAttempts,
			value:  b.MaxAttempts,
		},
		{
			field:  "Factor",
			expect: _factor,
			value:  b.Factor,
		},
		{
			field:  "Min",
			expect: _min,
			value:  b.Min,
		},
		{
			field:  "Max",
			expect: _max,
			value:  b.Max,
		},
	} {
		if tc.expect != tc.value {
			t.Errorf("Test #%d: expected %s to be \"%s\", but got \"%s\"", i+1, tc.field, tc.expect, tc.value)
			continue
		}
	}
}

func TestBackoff_Attempt(t *testing.T) {
	b := newBackoffWithMockTimer(0, 0, 0, 0)
	if b == nil {
		t.Fatal("expected backoff to not be nil")
		return
	}

	// Ensure attempt defaults to 0.
	if b.Attempt() != 0 {
		t.Errorf("Test #0: expected attempt to be \"%d\", but got \"%d\"", 0, b.Attempt())
		return
	}

	// Run the first (0) attempt.
	b.Next(context.Background())

	// Ensure Next increments the attempt for the next run.
	if b.Attempt() != 1 {
		t.Errorf("Test #1: expected attempt to be \"%d\", but got \"%d\"", 1, b.Attempt())
		return
	}
}

func TestBackoff_Duration(t *testing.T) {
	t.Run("Duration", func(t *testing.T) {
		b := newBackoffWithMockTimer(0, 2, 500*time.Millisecond, 3*time.Second)
		if b == nil {
			t.Fatal("expected backoff to not be nil")
			return
		}

		// Ensure first duration is 0.
		if duration := b.Duration(); duration != 0 {
			t.Errorf("Test #0: expected duration to be \"%s\", but got \"%s\"", time.Duration(0), duration)
			return
		}

		// Run the first attempt.
		b.Next(context.Background())

		// Ensure the next duration is set correctly.
		expect := time.Duration(b.Factor * float64(b.Min))
		if duration := b.Duration(); duration != expect {
			t.Errorf("Test #1: expected duration to be \"%s\", but got \"%s\"", expect, duration)
			return
		}
	})

	t.Run("Duration does not drop below Min", func(t *testing.T) {
		b := newBackoffWithMockTimer(0, 0.25, 1*time.Second, 5*time.Second)
		if b == nil {
			t.Fatal("expected backoff to not be nil")
			return
		}

		// Ensure first duration is 0.
		if duration := b.Duration(); duration != 0 {
			t.Errorf("Test #0: expected duration to be \"%s\", but got \"%s\"", time.Duration(0), duration)
			return
		}

		// Run the first attempt.
		b.Next(context.Background())

		// Ensure the next duration does not go below Min.
		if duration := b.Duration(); duration != b.Min {
			t.Errorf("Test #1: expected duration to be \"%s\", but got \"%s\"", b.Min, duration)
			return
		}
	})

	t.Run("Duration does not exceed Max", func(t *testing.T) {
		b := newBackoffWithMockTimer(0, 2, 3*time.Second, 500*time.Millisecond)
		if b == nil {
			t.Fatal("expected backoff to not be nil")
			return
		}

		// Ensure first duration is 0.
		if duration := b.Duration(); duration != 0 {
			t.Errorf("Test #0: expected duration to be \"%s\", but got \"%s\"", time.Duration(0), duration)
			return
		}

		// Run the first attempt.
		b.Next(context.Background())

		// Ensure the next duration is capped by Max.
		if duration := b.Duration(); duration != b.Max {
			t.Errorf("Test #1: expected duration to be \"%s\", but got \"%s\"", b.Max, duration)
			return
		}
	})

	t.Run("Duration does not exceed Max when overflowing float64", func(t *testing.T) {
		b := newBackoffWithMockTimer(0, math.MaxFloat64, 3*time.Second, 500*time.Millisecond)
		if b == nil {
			t.Fatal("expected backoff to not be nil")
			return
		}

		// Ensure first duration is 0.
		if duration := b.Duration(); duration != 0 {
			t.Errorf("Test #0: expected duration to be \"%s\", but got \"%s\"", time.Duration(0), duration)
			return
		}

		// Run the first attempt.
		b.Next(context.Background())

		// Ensure the next duration is capped by Max.
		if duration := b.Duration(); duration != b.Max {
			t.Errorf("Test #1: expected duration to be \"%s\", but got \"%s\"", b.Max, duration)
			return
		}
	})
}

func TestBackoff_Next(t *testing.T) {
	t.Run("Aborts before the first attempt when context is cancelled immediately", func(t *testing.T) {
		b := newBackoffWithMockTimer(0, 0, 0, 0)
		if b == nil {
			t.Fatal("expected backoff to not be nil")
			return
		}

		ctx, cancel := context.WithCancel(context.Background())
		go func(ctx context.Context) {
			for b.Next(ctx) {
				t.Error("backoff ran even though context was immediately cancelled")
			}
		}(ctx)

		cancel()
	})

	t.Run("Aborts between attempts when context is cancelled", func(t *testing.T) {
		// This test sets time parameters to test the other branch of Next.
		// Next has two logic paths, one for when there is no duration and
		// another for when there is a duration.
		b := newBackoffWithMockTimer(0, 3, 1*time.Second, 5*time.Second)
		if b == nil {
			t.Fatal("expected backoff to not be nil")
			return
		}

		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		defer close(done)
		go func(ctx context.Context, done chan<- struct{}) {
			for b.Next(ctx) {
				if b.Attempt() > 1 {
					t.Error("backoff continued to run after context was cancelled")
					return
				}
				cancel()
			}

			done <- struct{}{}
		}(ctx, done)

		<-done
	})

	t.Run("Runs with MaxAttempts set to zero", func(t *testing.T) {
		b := newBackoffWithMockTimer(0, 0, 0, 0)
		if b == nil {
			t.Fatal("expected backoff to not be nil")
			return
		}

		// Check if Next returns false, this means it will not continue running.
		if !b.Next(context.Background()) {
			t.Error("Next doesn't run with MaxAttempts set to zero")
		}
	})

	t.Run("Aborts when MaxAttempt limit is reached", func(t *testing.T) {
		b := newBackoffWithMockTimer(5, 0, 0, 0)
		if b == nil {
			t.Fatal("expected backoff to not be nil")
			return
		}

		var i uint
		ctx := context.Background()
		for b.Next(ctx) {
			i++
		}

		if i != b.MaxAttempts {
			t.Errorf("expected number of attempts to be \"%d\", but got \"%d\"", b.MaxAttempts, i)
		}
	})

	t.Run("Waits between attempts", func(t *testing.T) {
		b := newBackoffWithMockTimer(3, 2, 5*time.Millisecond, 50*time.Millisecond)
		if b == nil {
			t.Fatal("expected backoff to not be nil")
			return
		}

		var (
			i            uint
			lastDuration = b.Duration()
		)
		ctx := context.Background()
		for b.Next(ctx) {
			d := b.Duration()
			if lastDuration >= d {
				t.Error("duration was expected to increase from the previous attempt")
				return
			}
			i++
			lastDuration = d
		}
	})
}

func TestBackoff_Reset(t *testing.T) {
	b := newBackoffWithMockTimer(0, 0, 0, 0)
	if b == nil {
		t.Fatal("expected backoff to not be nil")
		return
	}

	// Run next to ensure the backoff is not in its default state.
	ctx := context.Background()
	b.Next(ctx)
	b.Next(ctx)

	if b.Attempt() == 0 {
		t.Error("backoff attempt count is still at zero after being ran twice")
		return
	}

	// Reset the backoff.
	b.Reset()

	if b.Attempt() != 0 {
		t.Error("backoff attempt count was not reset to zero")
		return
	}
}
