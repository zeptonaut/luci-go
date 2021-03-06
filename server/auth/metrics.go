// Copyright 2015 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package auth

import (
	"golang.org/x/net/context"

	"github.com/luci/luci-go/common/clock"
	"github.com/luci/luci-go/common/errors"
	"github.com/luci/luci-go/common/tsmon/distribution"
	"github.com/luci/luci-go/common/tsmon/field"
	"github.com/luci/luci-go/common/tsmon/metric"
	"github.com/luci/luci-go/common/tsmon/types"
)

var (
	authenticateDuration = metric.NewCumulativeDistribution(
		"luci/auth/methods/authenticate",
		"Distribution of 'Authenticate' call durations per result.",
		&types.MetricMetadata{Units: types.Microseconds},
		distribution.DefaultBucketer,
		field.String("result"))

	mintDelegationTokenDuration = metric.NewCumulativeDistribution(
		"luci/auth/methods/mint_delegation_token",
		"Distribution of 'MintDelegationToken' call durations per result.",
		&types.MetricMetadata{Units: types.Microseconds},
		distribution.DefaultBucketer,
		field.String("result"))

	mintAccessTokenDuration = metric.NewCumulativeDistribution(
		"luci/auth/methods/mint_access_token",
		"Distribution of 'MintAccessTokenForServiceAccount' call durations per result.",
		&types.MetricMetadata{Units: types.Microseconds},
		distribution.DefaultBucketer,
		field.String("result"))
)

func durationReporter(c context.Context, m metric.CumulativeDistribution) func(error, string) {
	startTs := clock.Now(c)
	return func(err error, result string) {
		// We should report context errors as such. It doesn't matter at which stage
		// the deadline happens, thus we ignore 'result' if seeing a context error.
		switch errors.Unwrap(err) {
		case context.DeadlineExceeded:
			result = "CONTEXT_DEADLINE_EXCEEDED"
		case context.Canceled:
			result = "CONTEXT_CANCELED"
		}
		m.Add(c, float64(clock.Since(c, startTs).Nanoseconds()/1000), result)
	}
}
