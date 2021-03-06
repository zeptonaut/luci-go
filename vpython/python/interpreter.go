// Copyright 2017 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package python

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/luci/luci-go/common/errors"
	"github.com/luci/luci-go/common/system/filesystem"

	"golang.org/x/net/context"
)

// Interpreter represents a system Python interpreter. It exposes the ability
// to use common functionality of that interpreter.
type Interpreter struct {
	// Python is the path to the system Python interpreter.
	Python string

	// cachedVersion is the cached Version for this interpreter. It is populated
	// on the first GetVersion call.
	cachedVersion   *Version
	cachedVersionMu sync.Mutex

	// cachedHash is the cached SHA256 hash string of the interpreter's binary
	// contents. It is populated once, protected by cachedHashOnce.
	cachedHash     string
	cachedHashErr  error
	cachedHashOnce sync.Once

	// testCommandHook, if not nil, is called on generated Command results prior
	// to returning them.
	testCommandHook func(*exec.Cmd)
}

// Normalize normalizes the Interpreter configuration by resolving relative
// paths into absolute paths and evaluating symlnks.
func (i *Interpreter) Normalize() error {
	if err := filesystem.AbsPath(&i.Python); err != nil {
		return err
	}
	resolved, err := filepath.EvalSymlinks(i.Python)
	if err != nil {
		return errors.Annotate(err, "could not evaluate symlinks for: %q", i.Python).Err()
	}
	i.Python = resolved
	return nil
}

// IsolatedCommand returns a configurable exec.Cmd structure bound to this
// Interpreter.
//
// The supplied arguments have several Python isolation flags prepended to them
// to remove environmental factors such as:
//	- The user's "site.py".
//	- The current PYTHONPATH environment variable.
//	- Compiled ".pyc/.pyo" files.
func (i *Interpreter) IsolatedCommand(c context.Context, args ...string) *exec.Cmd {
	// Isolate the supplied arguments.
	args = append([]string{
		"-B", // Don't compile "pyo" binaries.
		"-E", // Don't use PYTHON* environment variables.
		"-s", // Don't use user 'site.py'.
	}, args...)

	cmd := exec.CommandContext(c, i.Python, args...)
	if i.testCommandHook != nil {
		i.testCommandHook(cmd)
	}
	return cmd
}

// GetVersion runs the specified Python interpreter with the "--version"
// flag and maps it to a known specification verison.
func (i *Interpreter) GetVersion(c context.Context) (v Version, err error) {
	i.cachedVersionMu.Lock()
	defer i.cachedVersionMu.Unlock()

	// Check again, under write-lock.
	if i.cachedVersion != nil {
		v = *i.cachedVersion
		return
	}

	// We use CombinedOutput here becuase Python2 writes the version to STDERR,
	// while Python3+ writes it to STDOUT.
	cmd := i.IsolatedCommand(c, "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		err = errors.Annotate(err, "").Err()
		return
	}

	if v, err = ParseVersionOutput(string(out)); err != nil {
		return
	}

	i.cachedVersion = &v
	return
}

// Hash returns the SHA256 hash string of this interpreter.
//
// The hash value is cached; if called multiple times, the cached value will
// be returned.
func (i *Interpreter) Hash() (string, error) {
	hashInterpreter := func(path string) (string, error) {
		fd, err := os.Open(i.Python)
		if err != nil {
			return "", errors.Annotate(err, "failed to open interpreter").Err()
		}
		defer fd.Close()

		hash := sha256.New()
		if _, err := io.Copy(hash, fd); err != nil {
			return "", errors.Annotate(err, "failed to read [%s] for hashing", path).Err()
		}

		return hex.EncodeToString(hash.Sum(nil)), nil
	}

	i.cachedHashOnce.Do(func() {
		i.cachedHash, i.cachedHashErr = hashInterpreter(i.Python)
	})
	return i.cachedHash, i.cachedHashErr
}

// ParseVersionOutput parses a Version out of the output of a "--version" Python
// invocation.
func ParseVersionOutput(output string) (Version, error) {
	s := strings.TrimSpace(string(output))

	// Expected output:
	// Python X.Y.Z
	parts := strings.SplitN(s, " ", 2)
	if len(parts) != 2 || parts[0] != "Python" {
		return Version{}, errors.Reason("unknown version output").
			InternalReason("output(%q)", s).Err()
	}

	v, err := ParseVersion(parts[1])
	if err != nil {
		err = errors.Annotate(err, "failed to parse version from: %q", parts[1]).Err()
		return v, err
	}
	return v, nil
}
