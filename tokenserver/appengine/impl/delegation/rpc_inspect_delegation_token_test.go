// Copyright 2016 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package delegation

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"golang.org/x/net/context"

	"github.com/luci/luci-go/common/clock"
	"github.com/luci/luci-go/common/clock/testclock"
	"github.com/luci/luci-go/server/auth/delegation/messages"
	"github.com/luci/luci-go/server/auth/signing"
	"github.com/luci/luci-go/server/auth/signing/signingtest"

	admin "github.com/luci/luci-go/tokenserver/api/admin/v1"

	. "github.com/smartystreets/goconvey/convey"
)

func TestInspectDelegationToken(t *testing.T) {
	ctx := context.Background()
	ctx, tc := testclock.UseTime(ctx, testclock.TestTimeUTC)

	rpc := InspectDelegationTokenRPC{
		Signer: signingtest.NewSigner(0, &signing.ServiceInfo{
			ServiceAccountName: "service@example.com",
		}),
	}

	original := &messages.Subtoken{
		DelegatedIdentity: "user:delegated@example.com",
		RequestorIdentity: "user:requestor@example.com",
		CreationTime:      clock.Now(ctx).Unix(),
		ValidityDuration:  3600,
		Audience:          []string{"*"},
		Services:          []string{"*"},
	}

	tok, _ := SignToken(ctx, rpc.Signer, original)

	Convey("Happy path", t, func() {
		resp, err := rpc.InspectDelegationToken(ctx, &admin.InspectDelegationTokenRequest{
			Token: tok,
		})
		So(err, ShouldBeNil)

		resp.Envelope.Pkcs1Sha256Sig = nil
		resp.Envelope.SerializedSubtoken = nil
		So(resp, ShouldResemble, &admin.InspectDelegationTokenResponse{
			Valid:      true,
			Signed:     true,
			NonExpired: true,
			Envelope: &messages.DelegationToken{
				SignerId:     "user:service@example.com",
				SigningKeyId: "f9da5a0d0903bda58c6d664e3852a89c283d7fe9",
			},
			Subtoken: original,
		})
	})

	Convey("Not base64", t, func() {
		resp, err := rpc.InspectDelegationToken(ctx, &admin.InspectDelegationTokenRequest{
			Token: "@@@@@@@@@@@@@",
		})
		So(err, ShouldBeNil)
		So(resp, ShouldResemble, &admin.InspectDelegationTokenResponse{
			InvalidityReason: "not base64 - illegal base64 data at input byte 0",
		})
	})

	Convey("Not valid envelope proto", t, func() {
		resp, err := rpc.InspectDelegationToken(ctx, &admin.InspectDelegationTokenRequest{
			Token: "zzzz",
		})
		So(err, ShouldBeNil)
		So(resp, ShouldResemble, &admin.InspectDelegationTokenResponse{
			InvalidityReason: "can't unmarshal the envelope - proto: can't skip unknown wire type 7 for messages.DelegationToken",
		})
	})

	Convey("Bad signature", t, func() {
		env, _, _ := deserializeForTest(ctx, tok, rpc.Signer)
		env.Pkcs1Sha256Sig = []byte("lalala")
		blob, _ := proto.Marshal(env)
		tok := base64.RawURLEncoding.EncodeToString(blob)

		resp, err := rpc.InspectDelegationToken(ctx, &admin.InspectDelegationTokenRequest{
			Token: tok,
		})
		So(err, ShouldBeNil)

		resp.Envelope.Pkcs1Sha256Sig = nil
		resp.Envelope.SerializedSubtoken = nil
		So(resp, ShouldResemble, &admin.InspectDelegationTokenResponse{
			Valid:            false,
			InvalidityReason: "bad signature - crypto/rsa: verification error",
			Signed:           false,
			NonExpired:       true,
			Envelope: &messages.DelegationToken{
				SignerId:     "user:service@example.com",
				SigningKeyId: "f9da5a0d0903bda58c6d664e3852a89c283d7fe9",
			},
			Subtoken: original,
		})
	})

	Convey("Expired", t, func() {
		tc.Add(2 * time.Hour)

		resp, err := rpc.InspectDelegationToken(ctx, &admin.InspectDelegationTokenRequest{
			Token: tok,
		})
		So(err, ShouldBeNil)

		resp.Envelope.Pkcs1Sha256Sig = nil
		resp.Envelope.SerializedSubtoken = nil
		So(resp, ShouldResemble, &admin.InspectDelegationTokenResponse{
			Valid:            false,
			InvalidityReason: "expired",
			Signed:           true,
			NonExpired:       false,
			Envelope: &messages.DelegationToken{
				SignerId:     "user:service@example.com",
				SigningKeyId: "f9da5a0d0903bda58c6d664e3852a89c283d7fe9",
			},
			Subtoken: original,
		})
	})
}
