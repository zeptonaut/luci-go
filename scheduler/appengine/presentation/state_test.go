// Copyright 2017 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package presentation

import (
	"testing"

	"golang.org/x/net/context"

	"github.com/golang/protobuf/proto"
	"github.com/luci/luci-go/scheduler/appengine/catalog"
	"github.com/luci/luci-go/scheduler/appengine/engine"
	"github.com/luci/luci-go/scheduler/appengine/messages"
	"github.com/luci/luci-go/scheduler/appengine/task"
	"github.com/luci/luci-go/scheduler/appengine/task/urlfetch"
	. "github.com/smartystreets/goconvey/convey"
)

func TestGetPublicStateKind(t *testing.T) {
	t.Parallel()

	Convey("works", t, func() {
		So(GetPublicStateKind(&engine.Job{
			State: engine.JobState{State: engine.JobStateOverrun},
		}, task.Traits{}), ShouldEqual, PublicStateOverrun)

		So(GetPublicStateKind(&engine.Job{
			State: engine.JobState{State: engine.JobStateSlowQueue},
		}, task.Traits{}), ShouldEqual, PublicStateStarting)

		So(GetPublicStateKind(&engine.Job{
			State: engine.JobState{State: engine.JobStateSlowQueue, InvocationRetryCount: 1},
		}, task.Traits{}), ShouldEqual, PublicStateRetrying)

		So(GetPublicStateKind(&engine.Job{
			Paused: true,
			State:  engine.JobState{State: engine.JobStateSuspended},
		}, task.Traits{}), ShouldEqual, PublicStatePaused)

		So(GetPublicStateKind(&engine.Job{
			State: engine.JobState{State: engine.JobStateQueued, InvocationID: 1},
		}, task.Traits{Multistage: true}), ShouldEqual, PublicStateStarting)

		So(GetPublicStateKind(&engine.Job{
			State: engine.JobState{State: engine.JobStateQueued, InvocationID: 1},
		}, task.Traits{Multistage: false}), ShouldEqual, PublicStateRunning)
	})
}

func TestGetJobTraits(t *testing.T) {
	t.Parallel()
	Convey("works", t, func() {
		cat := catalog.New("scheduler.cfg")
		So(cat.RegisterTaskManager(&urlfetch.TaskManager{}), ShouldBeNil)
		ctx := context.Background()

		Convey("bad task", func() {
			taskBlob, err := proto.Marshal(&messages.TaskDefWrapper{
				Noop: &messages.NoopTask{},
			})
			So(err, ShouldBeNil)
			traits, err := GetJobTraits(ctx, cat, &engine.Job{
				Task: taskBlob,
			})
			So(err, ShouldNotBeNil)
			So(traits, ShouldResemble, task.Traits{})
		})

		Convey("OK task", func() {
			taskBlob, err := proto.Marshal(&messages.TaskDefWrapper{
				UrlFetch: &messages.UrlFetchTask{Url: "http://example.com/path"},
			})
			So(err, ShouldBeNil)
			traits, err := GetJobTraits(ctx, cat, &engine.Job{
				Task: taskBlob,
			})
			So(err, ShouldBeNil)
			So(traits, ShouldResemble, task.Traits{})
		})
	})
}
