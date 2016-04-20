// Copyright 2015 The Chromium Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package errors

import (
	"errors"
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

type otherMEType []error

func (o otherMEType) Error() string { return "FAIL" }

func TestUpstreamErrors(t *testing.T) {
	t.Parallel()

	Convey("Test MultiError", t, func() {
		Convey("nil", func() {
			me := MultiError(nil)
			So(me.Error(), ShouldEqual, "(0 errors)")
			Convey("single", func() {
				So(SingleError(error(me)), ShouldBeNil)
			})
		})
		Convey("one", func() {
			me := MultiError{errors.New("sup")}
			So(me.Error(), ShouldEqual, "sup")
		})
		Convey("two", func() {
			me := MultiError{errors.New("sup"), errors.New("what")}
			So(me.Error(), ShouldEqual, "sup (and 1 other error)")
		})
		Convey("more", func() {
			me := MultiError{errors.New("sup"), errors.New("what"), errors.New("nerds")}
			So(me.Error(), ShouldEqual, "sup (and 2 other errors)")

			Convey("single", func() {
				So(SingleError(error(me)), ShouldResemble, errors.New("sup"))
			})
		})
	})

	Convey("SingleError passes through", t, func() {
		e := errors.New("unique")
		So(SingleError(e), ShouldEqual, e)
	})

	Convey("Test MultiError Conversion", t, func() {
		ome := otherMEType{errors.New("sup")}
		So(Fix(ome), ShouldHaveSameTypeAs, MultiError{})
	})

	Convey("Fix passes through", t, func() {
		e := errors.New("unique")
		So(Fix(e), ShouldEqual, e)
	})
}

func TestAny(t *testing.T) {
	t.Parallel()

	Convey(`Testing the Any function`, t, func() {
		for _, tc := range []struct {
			err error
			has bool
		}{
			{nil, false},
			{New("test error"), true},
			{New("other error"), false},
			{MultiError{MultiError{New("test error"), nil}, New("other error")}, true},
		} {
			Convey(fmt.Sprintf(`Registers %v for error [%v]`, tc.has, tc.err), func() {
				So(Any(tc.err, func(err error) bool { return err.Error() == "test error" }), ShouldEqual, tc.has)
			})
		}
	})
}
