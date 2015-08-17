// Copyright (c) 2015 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package retry

import (
	"time"

	"github.com/luci/luci-go/common/clock"
	"golang.org/x/net/context"
)

// Stop is a sentinel value returned by a Iterator to indicate that no more
// attempts should be made.
const Stop time.Duration = -1

// Callback is a callback function that Retry will invoke every time an
// attempt fails prior to sleeping.
type Callback func(error, time.Duration)

// Iterator describes a stateful implementation of retry logic.
type Iterator interface {
	// Returns the next retry delay, or Stop if no more retries should be made.
	Next(context.Context, error) time.Duration
}

// Retry executes a function. If the function returns an error, it will
// be re-executed according to a retry plan.
//
// If the supplied context is cancelled, retry will stop executing. Retry will
// not execute the supplied function at all if the context is canelled when
// Retry is invoked.
//
// If notify is not nil, it will be invoked if an error occurs (prior to
// sleeping).
func Retry(ctx context.Context, it Iterator, f func() error, callback Callback) (err error) {
	for {
		// If we've been cancelled, don't try/retry.
		select {
		case <-ctx.Done():
			return ctx.Err()

		default:
			break
		}

		// Execute the function.
		err = f()
		if err == nil || it == nil {
			return
		}

		delay := it.Next(ctx, err)
		if delay == Stop {
			return
		}

		// Notify our observer that we are retrying.
		if callback != nil {
			callback(err, delay)
		}

		if delay > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()

			case <-clock.After(ctx, delay):
				break
			}
		}
	}
}