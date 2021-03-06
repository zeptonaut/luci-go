// Copyright 2017 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package swarming

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/julienschmidt/httprouter"
	"golang.org/x/net/context"

	"github.com/luci/gae/impl/memory"
	"github.com/luci/luci-go/common/clock/testclock"
	"github.com/luci/luci-go/milo/common"
	"github.com/luci/luci-go/server/auth"
	"github.com/luci/luci-go/server/auth/authtest"
	"github.com/luci/luci-go/server/auth/identity"
	"github.com/luci/luci-go/server/router"
	"github.com/luci/luci-go/server/templates"

	. "github.com/smartystreets/goconvey/convey"
)

func requestCtx(c context.Context, params ...httprouter.Param) *router.Context {
	p := httprouter.Params(params)
	r := &http.Request{URL: &url.URL{Path: "/foobar"}}
	c = common.WithRequest(c, r)
	w := httptest.NewRecorder()
	return &router.Context{
		Context: c,
		Params:  p,
		Writer:  w,
		Request: r,
	}
}

func TestHtml(t *testing.T) {
	c := memory.UseWithAppID(context.Background(), "dev~luci-milo")
	c, _ = testclock.UseTime(c, testclock.TestTimeUTC)
	c = templates.Use(c, common.GetTemplateBundle("../../frontend/appengine/templates"))
	c = auth.WithState(c, &authtest.FakeState{Identity: identity.AnonymousIdentity})

	Convey(`HTML handler tests`, t, func() {
		Convey(`Build pages`, func() {
			Convey(`Empty request`, func() {
				rc := requestCtx(c)
				BuildHandler(rc)
				response := rc.Writer.(*httptest.ResponseRecorder)
				So(response.Code, ShouldEqual, http.StatusBadRequest)
			})
			Convey(`With id foo`, func() {
				rc := requestCtx(c, httprouter.Param{"id", "foo"})
				BuildHandler(rc)
				response := rc.Writer.(*httptest.ResponseRecorder)
				So(response.Code, ShouldEqual, http.StatusBadRequest)
			})
		})

		Convey(`Log pages`, func() {
			Convey(`Empty request`, func() {
				rc := requestCtx(c)
				LogHandler(rc)
				response := rc.Writer.(*httptest.ResponseRecorder)
				So(response.Code, ShouldEqual, http.StatusBadRequest)
			})
		})
	})
}
