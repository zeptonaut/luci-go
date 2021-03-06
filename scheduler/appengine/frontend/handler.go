// Copyright 2015 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// Package frontend implements GAE web server for luci-scheduler service.
//
// Due to the way classic GAE imports work, this package can not have
// subpackages (or at least subpackages referenced via absolute import path).
// We can't use relative imports because luci-go will then become unbuildable
// by regular (non GAE) toolset.
//
// See https://groups.google.com/forum/#!topic/google-appengine-go/dNhqV6PBqVc.
package frontend

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/appengine"

	"github.com/luci/gae/service/info"
	tq "github.com/luci/gae/service/taskqueue"

	"github.com/luci/luci-go/grpc/discovery"
	"github.com/luci/luci-go/grpc/grpcmon"
	"github.com/luci/luci-go/grpc/prpc"
	"github.com/luci/luci-go/server/router"

	"github.com/luci/luci-go/appengine/gaemiddleware"

	"github.com/luci/luci-go/common/data/rand/mathrand"
	"github.com/luci/luci-go/common/logging"
	"github.com/luci/luci-go/common/retry/transient"

	"github.com/luci/luci-go/scheduler/api/scheduler/v1"

	"github.com/luci/luci-go/scheduler/appengine/apiservers"
	"github.com/luci/luci-go/scheduler/appengine/catalog"
	"github.com/luci/luci-go/scheduler/appengine/engine"
	"github.com/luci/luci-go/scheduler/appengine/task"
	"github.com/luci/luci-go/scheduler/appengine/task/buildbucket"
	"github.com/luci/luci-go/scheduler/appengine/task/gitiles"
	"github.com/luci/luci-go/scheduler/appengine/task/noop"
	"github.com/luci/luci-go/scheduler/appengine/task/swarming"
	"github.com/luci/luci-go/scheduler/appengine/task/urlfetch"
	"github.com/luci/luci-go/scheduler/appengine/ui"
)

//// Global state. See init().

var (
	globalCatalog catalog.Catalog
	globalEngine  engine.Engine

	// Known kinds of tasks.
	managers = []task.Manager{
		&buildbucket.TaskManager{},
		&gitiles.TaskManager{},
		&noop.TaskManager{},
		&swarming.TaskManager{},
		&urlfetch.TaskManager{},
	}
)

//// Helpers.

// requestContext is used to add helper methods.
type requestContext router.Context

// fail writes error message to the log and the response and sets status code.
func (c *requestContext) fail(code int, msg string, args ...interface{}) {
	body := fmt.Sprintf(msg, args...)
	logging.Errorf(c.Context, "HTTP %d: %s", code, body)
	http.Error(c.Writer, body, code)
}

// err sets status to 500 on transient errors or 202 on fatal ones. Returning
// status code in range [200–299] is the only way to tell Task Queues to stop
// retrying the task.
func (c *requestContext) err(e error, msg string, args ...interface{}) {
	code := 500
	if !transient.Tag.In(e) {
		code = 202
	}
	args = append(args, e)
	c.fail(code, msg+" - %s", args...)
}

// ok sets status to 200 and puts "OK" in response.
func (c *requestContext) ok() {
	c.Writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	c.Writer.WriteHeader(200)
	fmt.Fprintln(c.Writer, "OK")
}

///

var globalInit sync.Once

// initializeGlobalState does one time initialization for stuff that needs
// active GAE context.
func initializeGlobalState(c context.Context) {
	if info.IsDevAppServer(c) {
		// Dev app server doesn't preserve the state of task queues across restarts,
		// need to reset datastore state accordingly, otherwise everything gets stuck.
		if err := globalEngine.ResetAllJobsOnDevServer(c); err != nil {
			logging.Errorf(c, "Failed to reset jobs: %s", err)
		}
	}
}

//// Routes.

func init() {
	// Dev server likes to restart a lot, and upon a restart math/rand seed is
	// always set to 1, resulting in lots of presumably "random" IDs not being
	// very random. Seed it with real randomness.
	mathrand.SeedRandomly()

	// Setup global singletons.
	globalCatalog = catalog.New("")
	for _, m := range managers {
		if err := globalCatalog.RegisterTaskManager(m); err != nil {
			panic(err)
		}
	}
	globalEngine = engine.NewEngine(engine.Config{
		Catalog:              globalCatalog,
		TimersQueuePath:      "/internal/tasks/timers",
		TimersQueueName:      "timers",
		InvocationsQueuePath: "/internal/tasks/invocations",
		InvocationsQueueName: "invocations",
		PubSubPushPath:       "/pubsub",
	})

	// Do global init before handling requests.
	base := gaemiddleware.BaseProd().Extend(
		func(c *router.Context, next router.Handler) {
			globalInit.Do(func() { initializeGlobalState(c.Context) })
			next(c)
		},
	)

	// Setup HTTP routes.
	r := router.New()

	gaemiddleware.InstallHandlersWithMiddleware(r, base)

	ui.InstallHandlers(r, base, ui.Config{
		Engine:        globalEngine,
		Catalog:       globalCatalog,
		TemplatesPath: "templates",
	})

	r.POST("/pubsub", base, pubsubPushHandler) // auth is via custom tokens
	r.GET("/internal/cron/read-config", base.Extend(gaemiddleware.RequireCron), readConfigCron)
	r.POST("/internal/tasks/read-project-config", base.Extend(gaemiddleware.RequireTaskQueue("read-project-config")), readProjectConfigTask)
	r.POST("/internal/tasks/timers", base.Extend(gaemiddleware.RequireTaskQueue("timers")), actionTask)
	r.POST("/internal/tasks/invocations", base.Extend(gaemiddleware.RequireTaskQueue("invocations")), actionTask)

	// Devserver can't accept PubSub pushes, so allow manual pulls instead to
	// simplify local development.
	if appengine.IsDevAppServer() {
		r.GET("/pubsub/pull/:ManagerName/:Publisher", base, pubsubPullHandler)
	}

	// Install RPC servers.
	api := prpc.Server{
		UnaryServerInterceptor: grpcmon.NewUnaryServerInterceptor(nil),
	}
	scheduler.RegisterSchedulerServer(&api, apiservers.SchedulerServer{
		Engine:  globalEngine,
		Catalog: globalCatalog,
	})
	discovery.Enable(&api)
	api.InstallHandlers(r, base)

	http.DefaultServeMux.Handle("/", r)
}

// pubsubPushHandler handles incoming PubSub messages.
func pubsubPushHandler(c *router.Context) {
	rc := requestContext(*c)
	body, err := ioutil.ReadAll(rc.Request.Body)
	if err != nil {
		rc.fail(500, "Failed to read the request: %s", err)
		return
	}
	if err = globalEngine.ProcessPubSubPush(rc.Context, body); err != nil {
		rc.err(err, "Failed to process incoming PubSub push")
		return
	}
	rc.ok()
}

// pubsubPullHandler is called on dev server by developer to pull pubsub
// messages from a topic created for a publisher.
func pubsubPullHandler(c *router.Context) {
	rc := requestContext(*c)
	if !appengine.IsDevAppServer() {
		rc.fail(403, "Not a dev server")
		return
	}
	err := globalEngine.PullPubSubOnDevServer(
		rc.Context, rc.Params.ByName("ManagerName"), rc.Params.ByName("Publisher"))
	if err != nil {
		rc.err(err, "Failed to pull PubSub messages")
	} else {
		rc.ok()
	}
}

// readConfigCron grabs a list of projects from the catalog and datastore and
// dispatches task queue tasks to update each project's cron jobs.
func readConfigCron(c *router.Context) {
	rc := requestContext(*c)
	projectsToVisit := map[string]bool{}

	// Visit all projects in the catalog.
	ctx, _ := context.WithTimeout(rc.Context, 150*time.Second)
	projects, err := globalCatalog.GetAllProjects(ctx)
	if err != nil {
		rc.err(err, "Failed to grab a list of project IDs from catalog")
		return
	}
	for _, id := range projects {
		projectsToVisit[id] = true
	}

	// Also visit all registered projects that do not show up in the catalog
	// listing anymore. It will unregister all jobs belonging to them.
	existing, err := globalEngine.GetAllProjects(rc.Context)
	if err != nil {
		rc.err(err, "Failed to grab a list of project IDs from datastore")
		return
	}
	for _, id := range existing {
		projectsToVisit[id] = true
	}

	// Handle each project in its own task to avoid "bad" projects (e.g. ones with
	// lots of jobs) to slow down "good" ones.
	tasks := make([]*tq.Task, 0, len(projectsToVisit))
	for projectID := range projectsToVisit {
		tasks = append(tasks, &tq.Task{
			Path: "/internal/tasks/read-project-config?projectID=" + url.QueryEscape(projectID),
		})
	}
	if err = tq.Add(rc.Context, "read-project-config", tasks...); err != nil {
		rc.err(transient.Tag.Apply(err), "Failed to add tasks to task queue")
	} else {
		rc.ok()
	}
}

// readProjectConfigTask grabs a list of jobs in a project from catalog, updates
// all changed jobs, adds new ones, disables old ones.
func readProjectConfigTask(c *router.Context) {
	rc := requestContext(*c)
	projectID := rc.Request.URL.Query().Get("projectID")
	if projectID == "" {
		// Return 202 to avoid retry, it is fatal error.
		rc.fail(202, "Missing projectID query attribute")
		return
	}
	ctx, _ := context.WithTimeout(rc.Context, 150*time.Second)
	jobs, err := globalCatalog.GetProjectJobs(ctx, projectID)
	if err != nil {
		rc.err(err, "Failed to query for a list of jobs")
		return
	}
	if err = globalEngine.UpdateProjectJobs(rc.Context, projectID, jobs); err != nil {
		rc.err(err, "Failed to update some jobs")
		return
	}
	rc.ok()
}

// actionTask is used to route actions emitted by job state transitions back
// into Engine (see enqueueActions).
func actionTask(c *router.Context) {
	rc := requestContext(*c)
	body, err := ioutil.ReadAll(rc.Request.Body)
	if err != nil {
		rc.fail(500, "Failed to read request body: %s", err)
		return
	}
	count, _ := strconv.Atoi(rc.Request.Header.Get("X-AppEngine-TaskExecutionCount"))
	err = globalEngine.ExecuteSerializedAction(rc.Context, body, count)
	if err != nil {
		rc.err(err, "Error when executing the action")
		return
	}
	rc.ok()
}
