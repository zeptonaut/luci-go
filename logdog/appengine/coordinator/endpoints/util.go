// Copyright 2016 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package endpoints

import (
	"time"

	"github.com/golang/protobuf/ptypes/duration"
	"github.com/luci/luci-go/common/proto/google"
)

// MinDuration selects the smallest duration that is > 0 from a set of
// duration.Duration protobufs.
//
// If none of the supplied Durations are > 0, 0 will be returned.
func MinDuration(candidates ...*duration.Duration) (exp time.Duration) {
	for _, c := range candidates {
		if cd := google.DurationFromProto(c); cd > 0 && (exp <= 0 || cd < exp) {
			exp = cd
		}
	}
	return
}
