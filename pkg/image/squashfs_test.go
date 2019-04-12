// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package image

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// createSquashfs creates a small but valid squashfs file that can be used
// with an image.
func createSquashfs(t *testing.T) string {
	dir, dirErr := ioutil.TempDir("", "squashfsHdrTesting-")
	if dirErr != nil {
		t.Fatalf("impossible to create temporary directory: %s\n", dirErr)
	}
	defer os.RemoveAll(dir)

	sqshFile, fileErr := ioutil.TempFile("", "")
	if fileErr != nil {
		t.Fatalf("impossible to create temporary file: %s\n", fileErr)
	}
	defer sqshFile.Close()

	sqshFilePath, err := filepath.Abs(sqshFile.Name())
	if err != nil {
		t.Fatalf("cannot get path of temporary file: %s\n", err)
	}
	// close and delete the temp file since we will be used with the
	// mksquashfs command. We still use TempFile to have a clean way
	// to get a valid path for a temporary file
	sqshFile.Close()
	os.Remove(sqshFilePath)

	cmdBin := "/usr/bin/mksquashfs"
	if _, statErr := os.Stat(cmdBin); os.IsNotExist(statErr) {
		t.Skipf("%s is not  available, skipping the test...", cmdBin)
	}

	cmd := exec.Command(cmdBin, dir, sqshFilePath)
	cmdErr := cmd.Run()
	if cmdErr != nil {
		t.Fatalf("cannot create squashfs volume: %s\n", cmdErr)
	}

	return sqshFilePath
}

func TestCheckSquashfsHeader(t *testing.T) {
	sqshFilePath := createSquashfs(t)
	defer os.Remove(sqshFilePath)

	img, imgErr := os.Open(sqshFilePath)
	if imgErr != nil {
		t.Fatalf("cannot open file: %s\n", imgErr)
	}
	b := make([]byte, bufferSize)
	n, readErr := img.Read(b)
	if readErr != nil || n != bufferSize {
		t.Fatalf("cannot read the first %d bytes of the image file\n", bufferSize)
	}

	_, err := CheckSquashfsHeader(b)
	if err != nil {
		t.Fatalf("cannot check squashfs header of a valid image")
	}
}

func TestSquashfsInitializer(t *testing.T) {
	// Valid image test
	sqshFilePath := createSquashfs(t)
	defer os.Remove(sqshFilePath)

	var squashfsfmt squashfsFormat
	var err error
	mode := squashfsfmt.openMode(true)

	img := &Image{
		Path: sqshFilePath,
		Name: "test",
	}
	img.Writable = true
	img.File, err = os.OpenFile(sqshFilePath, mode, 0)
	if err != nil {
		t.Fatalf("cannot open image's file: %s\n", err)
	}
	fileinfo, err := img.File.Stat()
	if err != nil {
		img.File.Close()
		t.Fatalf("cannot stat the image file: %s\n", err)
	}

	err = squashfsfmt.initializer(img, fileinfo)
	if err != nil {
		img.File.Close()
		t.Fatalf("initializer failed: %s\n", err)
	}
	img.File.Close()

	// Invalid image
	invalidPath, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("impossible to create temporary directory: %s\n", err)
	}
	img.File, err = os.Open(invalidPath)
	if err != nil {
		t.Fatalf("open() failed: %s\n", err)
	}
	defer img.File.Close()
	fileinfo, err = img.File.Stat()
	if err != nil {
		t.Fatalf("cannot stat file pointer: %s\n", err)
	}

	err = squashfsfmt.initializer(img, fileinfo)
	if err == nil {
		t.Fatal("squashfs succeeded with a directory while expected to fail")
	}
}

func TestSFSOpenMode(t *testing.T) {
	var squashfsfmt squashfsFormat

	// Yes, openMode() for squashfs always returns os.O_RDONLY
	if squashfsfmt.openMode(true) != os.O_RDONLY {
		t.Fatal("openMode(true) returned the wrong value")
	}
	if squashfsfmt.openMode(false) != os.O_RDONLY {
		t.Fatal("openMode(false) returned the wrong value")
	}
}