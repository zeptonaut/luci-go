// Copyright 2015 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package common

import (
	"net/http"
	"strings"

	"golang.org/x/net/context"

	"github.com/luci/gae/impl/cloud"
	"github.com/luci/gae/service/info"
	"github.com/luci/luci-go/appengine/gaeauth/server"
	"github.com/luci/luci-go/appengine/gaemiddleware"
	"github.com/luci/luci-go/common/clock"
	"github.com/luci/luci-go/server/analytics"
	"github.com/luci/luci-go/server/auth"
	"github.com/luci/luci-go/server/auth/identity"
	"github.com/luci/luci-go/server/router"
	"github.com/luci/luci-go/server/templates"
)

var authconfig *auth.Config

// GetTemplateBundles is used to render HTML templates. It provides base args
// passed to all templates.  It takes a path to the template folder, relative
// to the path of the binary during runtime.
func GetTemplateBundle(templatePath string) *templates.Bundle {
	return &templates.Bundle{
		Loader:          templates.FileSystemLoader(templatePath),
		DebugMode:       info.IsDevAppServer,
		DefaultTemplate: "base",
		DefaultArgs: func(c context.Context) (templates.Args, error) {
			r := getRequest(c)
			path := r.URL.Path
			loginURL, err := auth.LoginURL(c, path)
			if err != nil {
				return nil, err
			}
			logoutURL, err := auth.LogoutURL(c, path)
			if err != nil {
				return nil, err
			}
			return templates.Args{
				"AppVersion":  strings.Split(info.VersionID(c), ".")[0],
				"IsAnonymous": auth.CurrentIdentity(c) == identity.AnonymousIdentity,
				"User":        auth.CurrentUser(c),
				"LoginURL":    loginURL,
				"LogoutURL":   logoutURL,
				"CurrentTime": clock.Now(c),
				"Analytics":   analytics.Snippet(c),
				"RequestID":   info.RequestID(c),
				"Request":     r,
			}, nil
		},
		FuncMap: funcMap,
	}
}

// Base returns the basic LUCI appengine middlewares.
func Base(templatePath string) router.MiddlewareChain {
	return gaemiddleware.BaseProd().Extend(
		auth.Authenticate(server.CookieAuth),
		withRequestMiddleware,
		templates.WithTemplates(GetTemplateBundle(templatePath)),
	)
}

// FlexBase returns the basic middleware for use on appengine flex.  Flex does not
// allow the use of appengine APIs.
func FlexBase() router.MiddlewareChain {
	// Installs the Info and Datastore services.
	return router.NewMiddlewareChain(func(c *router.Context, next router.Handler) {
		c.Context = cloud.UseFlex(c.Context)
		next(c)
	})
}

// The context key, so that we can embed the http.Request object into
// the context.
var requestKey = "http.request"

// WithRequest returns a context with the http.Request object
// in it.
func WithRequest(c context.Context, r *http.Request) context.Context {
	return context.WithValue(c, &requestKey, r)
}

// withRequestMiddleware is a middleware that installs a request into the context.
// This is used for various things in the default template.
func withRequestMiddleware(c *router.Context, next router.Handler) {
	c.Context = WithRequest(c.Context, c.Request)
	next(c)
}

func getRequest(c context.Context) *http.Request {
	if req, ok := c.Value(&requestKey).(*http.Request); ok {
		return req
	}
	panic("No http.request found in context")
}
