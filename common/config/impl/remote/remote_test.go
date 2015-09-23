// Copyright 2015 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package remote

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/luci/luci-go/common/config"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/net/context"
)

func encodeToB(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func testTools(code int, resp interface{}) (*httptest.Server, config.Interface) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(code)
		w.Header().Set("Content-Type", "application/json")
		marsh, _ := json.Marshal(resp)
		fmt.Fprintln(w, string(marsh))
	}))

	c := Use(context.Background(), nil, server.URL)
	return server, config.Get(c)
}

func TestRemoteCalls(t *testing.T) {
	t.Parallel()

	Convey("Should pass through calls to the generated API", t, func() {
		Convey("GetConfig", func() {
			server, remoteImpl := testTools(200, map[string]string{
				"content":      encodeToB("hi"),
				"content_hash": "bar",
				"revision":     "3",
			})
			defer server.Close()

			res, err := remoteImpl.GetConfig("a", "b")

			So(err, ShouldBeNil)
			So(*res, ShouldResemble, config.Config{
				ConfigSet:   "a",
				Content:     "hi",
				ContentHash: "bar",
				Revision:    "3",
			})
		})
		Convey("GetConfigByHash", func() {
			server, remoteImpl := testTools(200, map[string]string{
				"content": encodeToB("content"),
			})
			defer server.Close()

			res, err := remoteImpl.GetConfigByHash("a")

			So(err, ShouldBeNil)
			So(res, ShouldResemble, "content")
		})
		Convey("GetConfigSetLocation", func() {
			URL, err := url.Parse("http://example.com")
			if err != nil {
				panic(err)
			}

			server, remoteImpl := testTools(200, map[string]interface{}{
				"mappings": [...]interface{}{map[string]string{
					"config_set": "a",
					"location":   URL.String(),
				}},
			})
			defer server.Close()

			res, err := remoteImpl.GetConfigSetLocation("a")

			So(err, ShouldBeNil)
			So(*res, ShouldResemble, *URL)
		})
		Convey("GetProjectConfigs", func() {
			server, remoteImpl := testTools(200, map[string]interface{}{
				"configs": [...]interface{}{map[string]string{
					"config_set":   "a",
					"content":      encodeToB("hi"),
					"content_hash": "bar",
					"revision":     "3",
				}},
			})
			defer server.Close()

			res, err := remoteImpl.GetProjectConfigs("a")

			So(err, ShouldBeNil)
			So(res, ShouldNotBeEmpty)
			So(len(res), ShouldEqual, 1)
			So(*res[0], ShouldResemble, config.Config{
				ConfigSet:   "a",
				Content:     "hi",
				ContentHash: "bar",
				Revision:    "3",
			})
		})
		Convey("GetProjects", func() {
			id := "blink"
			name := "Blink"
			URL, err := url.Parse("http://example.com")
			if err != nil {
				panic(err)
			}

			server, remoteImpl := testTools(200, map[string]interface{}{
				"projects": [...]interface{}{map[string]string{
					"id":        id,
					"name":      name,
					"repo_type": "GITILES",
					"repo_url":  URL.String(),
				}},
			})
			defer server.Close()

			res, err := remoteImpl.GetProjects()

			So(err, ShouldBeNil)
			So(res, ShouldNotBeEmpty)
			So(len(res), ShouldEqual, 1)
			So(*res[0], ShouldResemble, config.Project{
				ID:       id,
				Name:     name,
				RepoType: config.GitilesRepo,
				RepoURL:  URL,
			})
		})
		Convey("GetRefConfigs", func() {
			server, remoteImpl := testTools(200, map[string]interface{}{
				"configs": [...]interface{}{map[string]string{
					"config_set":   "a",
					"content":      encodeToB("hi"),
					"content_hash": "bar",
					"revision":     "3",
				}},
			})
			defer server.Close()

			res, err := remoteImpl.GetRefConfigs("a")

			So(err, ShouldBeNil)
			So(res, ShouldNotBeEmpty)
			So(len(res), ShouldEqual, 1)
			So(*res[0], ShouldResemble, config.Config{
				ConfigSet:   "a",
				Content:     "hi",
				ContentHash: "bar",
				Revision:    "3",
			})
		})
		Convey("GetRefs", func() {
			ref := "refs/heads/master"
			server, remoteImpl := testTools(200, map[string]interface{}{
				"refs": [...]interface{}{map[string]string{
					"name": ref,
				}},
			})
			defer server.Close()

			res, err := remoteImpl.GetRefs("a")

			So(err, ShouldBeNil)
			So(res, ShouldNotBeEmpty)
			So(len(res), ShouldEqual, 1)
			So(res[0], ShouldEqual, ref)
		})
	})

	Convey("Should handle errors well", t, func() {
		Convey("Should enforce GetConfigSetLocation argument is not the empty string.", func() {
			c := Use(context.Background(), nil, "")
			remoteImpl := config.Get(c)

			_, err := remoteImpl.GetConfigSetLocation("")
			So(err, ShouldNotBeNil)
		})

		Convey("Should pass through HTTP errors", func() {
			client := http.Client{
				Transport: failingRoundTripper{},
			}
			c := Use(context.Background(), &client, "")
			remoteImpl := config.Get(c)

			_, err := remoteImpl.GetConfig("a", "b")
			So(err, ShouldNotBeNil)
			_, err = remoteImpl.GetConfigByHash("a")
			So(err, ShouldNotBeNil)
			_, err = remoteImpl.GetConfigSetLocation("a")
			So(err, ShouldNotBeNil)
			_, err = remoteImpl.GetProjectConfigs("a")
			So(err, ShouldNotBeNil)
			_, err = remoteImpl.GetProjects()
			So(err, ShouldNotBeNil)
			_, err = remoteImpl.GetRefConfigs("a")
			So(err, ShouldNotBeNil)
			_, err = remoteImpl.GetRefs("a")
			So(err, ShouldNotBeNil)
		})
	})
}

type failingRoundTripper struct{}

func (t failingRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return nil, fmt.Errorf("IM AM ERRAR\n")
}