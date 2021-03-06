// Copyright 2017 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package ensure

import (
	"bytes"
	"testing"

	. "github.com/luci/luci-go/common/testing/assertions"
	. "github.com/smartystreets/goconvey/convey"
)

var badEnsureFiles = []struct {
	name string
	file string
	err  string
}{
	{
		"too many tokens",
		"this has too many tokens",
		"bad version",
	},

	{
		"no version",
		"just/a/package",
		"bad version",
	},

	{
		"empty directive",
		"@ foobar",
		`unknown @directive: "@"`,
	},

	{
		"unknown directive",
		"@nerbs foobar",
		`unknown @directive: "@nerbs"`,
	},

	{
		"windows subdir",
		"@subdir folder\\thing",
		`bad subdir: backslashes not allowed`,
	},

	{
		"messy subdir",
		"@subdir folder/../something",
		`bad subdir: "folder/../something" (should be "something")`,
	},

	{
		"relative subdir",
		"@subdir ../../something",
		`bad subdir: invalid ".": "../../something"`,
	},

	{
		"absolute subdir",
		"@subdir /etc",
		`bad subdir: absolute paths not allowed`,
	},

	{
		"extra slashes",
		"@subdir //foo/bar/baz",
		`bad subdir`,
	},

	{
		"windows style",
		"@subdir c:/foo/bar/baz",
		`bad subdir`,
	},

	{
		"empty setting",
		"$ something",
		`unknown $setting: "$"`,
	},

	{
		"bad url",
		"$serviceurl ://sad.url",
		"url is invalid",
	},

	{
		"too many urls",
		f(
			"$serviceurl https://something.example.com",
			"$serviceurl https://something.else.example.com",
		),
		"$ServiceURL may only be set once per file",
	},

	{
		"bad setting",
		"$nurbs thingy",
		`unknown $setting: "$nurbs"`,
	},

	{
		"bad template",
		"foo/bar/${not_good} version",
		"failed to resolve package template",
	},

	{
		"bad template (2)",
		"foo/bar/$not_good version",
		"unable to process some variables",
	},

	{
		"duplicate package (literal)",
		f(
			"some/package/something version",
			"some/package/something latest",
		),
		`duplicate package in subdir "": "some/package/something": defined on line 1 and 2`,
	},

	{
		"duplicate package (template)",
		f(
			"some/package/${arch} version",
			"some/other/package canary",
			"some/package/test_arch latest",
		),
		`duplicate package in subdir "": "some/package/test_arch": defined on line 1 and 3`,
	},

	{
		"bad version resolution",
		f(
			"some/package/something error_version",
		),
		`failed to resolve package version (line 1)`,
	},

	{
		"late setting (pkg)",
		f(
			"some/package version",
			"",
			"$ServiceURL https://something.example.com",
		),
		`$setting found after non-$setting statements`,
	},

	{
		"late setting (directive)",
		f(
			"@Subdir some/path",
			"",
			"$ServiceURL https://something.example.com",
		),
		`$setting found after non-$setting statements`,
	},
}

func TestBadEnsureFiles(t *testing.T) {
	t.Parallel()

	Convey("bad ensure files", t, func() {
		for _, tc := range badEnsureFiles {
			Convey(tc.name, func() {
				buf := bytes.NewBufferString(tc.file)
				f, err := ParseFile(buf)
				if err != nil {
					So(f, ShouldBeNil)
					So(err, ShouldErrLike, tc.err)
				} else {
					So(f, ShouldNotBeNil)
					rf, err := f.ResolveWith(testResolver, map[string]string{
						"os":       "test_os",
						"arch":     "test_arch",
						"platform": "test_os-test_arch",
					})
					So(rf, ShouldBeNil)
					So(err, ShouldErrLike, tc.err)
				}
			})
		}
	})
}
