// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.
package e2e

import (
	"io/ioutil"
	"os"
	"testing"
)

var testenv = struct {
	CmdPath   string `split_words:"true"` // singularity program
	ImagePath string `split_words:"true"` // base image for tests
}{}

func EnsureImage(t *testing.T) {
	LoadEnv(t, &testenv)

	switch _, err := os.Stat(testenv.ImagePath); {
	case err == nil:
		// OK: file exists, return
		return

	case os.IsNotExist(err):
		// OK: file does not exist, continue

	default:
		// FATAL: something else is wrong
		t.Fatalf("Failed when checking image %q: %+v\n",
			testenv.ImagePath,
			err)
	}

	opts := BuildOpts{
		Force:   true,
		Sandbox: false,
	}

	b, err := ImageBuild(
		testenv.CmdPath,
		opts,
		testenv.ImagePath,
		"./testdata/Singularity")

	if err != nil {
		t.Logf("Failed to build image %q.\nOutput:\n%s\n",
			testenv.ImagePath,
			b)
		t.Fatalf("Unexpected failure: %+v", err)
	}
}

// MakeTmpDir will make a tmp dir and return a string of the path
func MakeTmpDir(t *testing.T) string {
	name, err := ioutil.TempDir("", "stest.")
	if err != nil {
		t.Fatalf("failed to create temporary directory: %v", err)
	}
	//defer os.RemoveAll(name)
	//defer name
	if err := os.Chmod(name, 0777); err != nil {
		t.Fatalf("failed to chmod temporary directory: %v", err)
	}
	return name
}
