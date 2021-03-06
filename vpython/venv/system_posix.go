// Copyright 2017 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// +build darwin linux freebsd netbsd openbsd android

package venv

import (
	"os"
	"path/filepath"
	"syscall"

	"github.com/luci/luci-go/common/errors"
)

// longestGeneratedScriptPath returns the path of the longest generated script
// given a VirtualEnv root.
func longestGeneratedScriptPath(baseDir string) string {
	return filepath.Join(venvBinDir(baseDir), "python-config")
}

// venvBinDir resolves the path where VirtualEnv binaries are installed.
func venvBinDir(root string) string {
	return filepath.Join(root, "bin")
}

func checkProcessRunning(pid int) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return errors.Annotate(err, "failed to find process").Err()
	}

	if err := proc.Signal(os.Signal(syscall.Signal(0))); err != nil {
		return errors.Annotate(err, "failed to signal process").Err()
	}
	return nil
}
