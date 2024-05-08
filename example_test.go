// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: Copyright (c) 2024 Matthew Penner

package backoff_test

import (
	"context"
	"time"

	"github.com/matthewpi/backoff"
)

func ExampleNew() {
	b := backoff.New(3, 2, 1*time.Second, 5*time.Second)

	// Avoid generating a new context every time Next is called.
	ctx := context.Background()
	for b.Next(ctx) {
		// Do something.
		//
		// break if successful, continue on failure
	}
}
