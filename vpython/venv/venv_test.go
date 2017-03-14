// Copyright 2017 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package venv

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/luci/luci-go/vpython/api/vpython"
	"github.com/luci/luci-go/vpython/python"

	"github.com/luci/luci-go/common/errors"
	"github.com/luci/luci-go/common/system/filesystem"
	"github.com/luci/luci-go/common/testing/testfs"

	"golang.org/x/net/context"

	. "github.com/luci/luci-go/common/testing/assertions"
	. "github.com/smartystreets/goconvey/convey"
)

type resolvedInterpreter struct {
	py      *python.Interpreter
	version python.Version
}

func resolveFromPath(vers python.Version) *resolvedInterpreter {
	c := context.Background()
	py, err := python.Find(c, vers)
	if err != nil {
		return nil
	}
	if err := filesystem.AbsPath(&py.Python); err != nil {
		panic(err)
	}

	ri := resolvedInterpreter{
		py: py,
	}
	if ri.version, err = ri.py.GetVersion(c); err != nil {
		panic(err)
	}
	return &ri
}

var (
	pythonGeneric = resolveFromPath(python.Version{})
	python27      = resolveFromPath(python.Version{2, 7, 0})
	python3       = resolveFromPath(python.Version{3, 0, 0})
)

func TestResolvePythonInterpreter(t *testing.T) {
	t.Parallel()

	Convey(`Resolving a Python interpreter`, t, func() {
		c := context.Background()
		cfg := Config{
			Spec: &vpython.Spec{},
		}

		// Tests to run if we have Python 2.7 installed.
		if python27 != nil {
			Convey(`When Python 2.7 is requested, it gets resolved.`, func() {
				cfg.Spec.PythonVersion = "2.7"
				So(cfg.resolvePythonInterpreter(c), ShouldBeNil)
				So(cfg.Python, ShouldEqual, python27.py.Python)

				vers, err := python.ParseVersion(cfg.Spec.PythonVersion)
				So(err, ShouldBeNil)
				So(vers.IsSatisfiedBy(python27.version), ShouldBeTrue)
			})

			Convey(`Fails when Python 9999 is requested, but a Python 2 interpreter is forced.`, func() {
				cfg.Python = python27.py.Python
				cfg.Spec.PythonVersion = "9999"
				So(cfg.resolvePythonInterpreter(c), ShouldErrLike, "doesn't match specification")
			})
		}

		// Tests to run if we have Python 2.7 and a generic Python installed.
		if pythonGeneric != nil && python27 != nil {
			// Our generic Python resolves to a known version, so we can proceed.
			Convey(`When no Python version is specified, spec resolves to generic.`, func() {
				So(cfg.resolvePythonInterpreter(c), ShouldBeNil)
				So(cfg.Python, ShouldEqual, pythonGeneric.py.Python)

				vers, err := python.ParseVersion(cfg.Spec.PythonVersion)
				So(err, ShouldBeNil)
				So(vers.IsSatisfiedBy(pythonGeneric.version), ShouldBeTrue)
			})
		}

		// Tests to run if we have Python 3 installed.
		if python3 != nil {
			Convey(`When Python 3 is requested, it gets resolved.`, func() {
				cfg.Spec.PythonVersion = "3"
				So(cfg.resolvePythonInterpreter(c), ShouldBeNil)
				So(cfg.Python, ShouldEqual, python3.py.Python)

				vers, err := python.ParseVersion(cfg.Spec.PythonVersion)
				So(err, ShouldBeNil)
				So(vers.IsSatisfiedBy(python3.version), ShouldBeTrue)
			})

			Convey(`Fails when Python 9999 is requested, but a Python 3 interpreter is forced.`, func() {
				cfg.Python = python3.py.Python
				cfg.Spec.PythonVersion = "9999"
				So(cfg.resolvePythonInterpreter(c), ShouldErrLike, "doesn't match specification")
			})
		}
	})
}

type setupCheckManifest struct {
	Interpreter string `json:"interpreter"`
	Pants       string `json:"pants"`
	Shirt       string `json:"shirt"`
}

func testVirtualEnvWith(t *testing.T, ri *resolvedInterpreter) {
	t.Parallel()

	if ri == nil {
		t.Skipf("No python interpreter found.")
	}

	tl, err := loadTestEnvironment(context.Background(), t)
	if err != nil {
		t.Fatalf("could not set up test loader for %q: %s", ri.py.Python, err)
	}

	Convey(`Testing Setup`, t, testfs.MustWithTempDir(t, "vpython", func(tdir string) {
		c := context.Background()
		config := Config{
			BaseDir:    tdir,
			MaxHashLen: 4,
			Package: vpython.Spec_Package{
				Name:    "foo/bar/virtualenv",
				Version: "unresolved",
			},
			Python: ri.py.Python,
			Spec: &vpython.Spec{
				Wheel: []*vpython.Spec_Package{
					{Name: "foo/bar/shirt", Version: "unresolved"},
					{Name: "foo/bar/pants", Version: "unresolved"},
				},
			},
			Loader: tl,
		}

		// Load the bootstrap wheels for the next part of the test.
		So(tl.ensureWheels(c, t, ri.py, tdir), ShouldBeNil)

		err := With(c, config, false, func(c context.Context, v *Env) error {
			testScriptPath := filepath.Join(testDataDir, "setup_check.py")
			checkOut := filepath.Join(tdir, "output.json")
			i := v.InterpreterCommand()
			So(i.Run(c, testScriptPath, "--json-output", checkOut), ShouldBeNil)

			var m setupCheckManifest
			So(loadJSON(checkOut, &m), ShouldBeNil)
			So(m.Interpreter, ShouldStartWith, v.Root)
			So(m.Pants, ShouldStartWith, v.Root)
			So(m.Shirt, ShouldStartWith, v.Root)

			// We should be able to delete it.
			So(v.Delete(c), ShouldBeNil)
			return nil
		})
		So(err, ShouldBeNil)
	}))
}

func TestVirtualEnv(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		ri   *resolvedInterpreter
	}{
		{"python27", python27},
		{"python3", python3},
	} {
		tc := tc

		t.Run(fmt.Sprintf(`Testing Virtualenv for: %s`, tc.name), func(t *testing.T) {
			testVirtualEnvWith(t, tc.ri)
		})
	}
}

func loadJSON(path string, dst interface{}) error {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return errors.Annotate(err).Reason("failed to open file").Err()
	}
	if err := json.Unmarshal(content, dst); err != nil {
		return errors.Annotate(err).Reason("failed to unmarshal JSON").Err()
	}
	return nil
}