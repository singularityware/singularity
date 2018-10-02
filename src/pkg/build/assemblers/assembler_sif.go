// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package assemblers

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"syscall"

	"github.com/satori/go.uuid"
	"github.com/sylabs/sif/pkg/sif"
	"github.com/sylabs/singularity/src/pkg/build/types"
	"github.com/sylabs/singularity/src/pkg/build/types/parser"
	"github.com/sylabs/singularity/src/pkg/sylog"
)

// SIFAssembler doesnt store anything
type SIFAssembler struct {
}

func createSIF(path string, definition []byte, squashfile string) (err error) {
	// general info for the new SIF file creation
	cinfo := sif.CreateInfo{
		Pathname:   path,
		Launchstr:  sif.HdrLaunch,
		Sifversion: sif.HdrVersion,
		ID:         uuid.NewV4(),
	}

	// data we need to create a definition file descriptor
	definput := sif.DescriptorInput{
		Datatype: sif.DataDeffile,
		Groupid:  sif.DescrDefaultGroup,
		Link:     sif.DescrUnusedLink,
		Data:     definition,
	}
	definput.Size = int64(binary.Size(definput.Data))

	// add this descriptor input element to creation descriptor slice
	cinfo.InputDescr = append(cinfo.InputDescr, definput)

	// data we need to create a system partition descriptor
	parinput := sif.DescriptorInput{
		Datatype: sif.DataPartition,
		Groupid:  sif.DescrDefaultGroup,
		Link:     sif.DescrUnusedLink,
		Fname:    squashfile,
	}
	// open up the data object file for this descriptor
	if parinput.Fp, err = os.Open(parinput.Fname); err != nil {
		return fmt.Errorf("while opening partition file: %s", err)
	}
	defer parinput.Fp.Close()
	fi, err := parinput.Fp.Stat()
	if err != nil {
		return fmt.Errorf("while calling start on partition file: %s", err)
	}
	parinput.Size = fi.Size()

	err = parinput.SetPartExtra(sif.FsSquash, sif.PartPrimSys, sif.GetSIFArch(runtime.GOARCH))
	if err != nil {
		return
	}

	// add this descriptor input element to the list
	cinfo.InputDescr = append(cinfo.InputDescr, parinput)

	// remove anything that may exist at the build destination at last moment
	os.RemoveAll(path)

	// test container creation with two partition input descriptors
	if _, err := sif.CreateContainer(cinfo); err != nil {
		return fmt.Errorf("while creating container: %s", err)
	}

	// chown the sif file to the calling user
	if uid, gid, ok := changeOwner(); ok {
		os.Chown(path, uid, gid)
	}

	return nil
}

// Assemble creates a SIF image from a Bundle
func (a *SIFAssembler) Assemble(b *types.Bundle, path string) (err error) {
	defer os.RemoveAll(b.Path)

	sylog.Infof("Creating SIF file...")

	// convert definition to plain text
	var buf bytes.Buffer
	parser.WriteDefinitionFile(&(b.Recipe), &buf)
	def := buf.Bytes()

	// make system partition image
	mksquashfs, err := exec.LookPath("mksquashfs")
	if err != nil {
		sylog.Errorf("mksquashfs is not installed on this system")
		return
	}

	f, err := ioutil.TempFile(b.Path, "squashfs-")
	squashfsPath := f.Name() + ".img"
	f.Close()
	os.Remove(f.Name())
	os.Remove(squashfsPath)

	args := []string{b.Rootfs(), squashfsPath, "-noappend"}

	// build squashfs with all-root flag when building as a user
	if syscall.Getuid() != 0 {
		args = append(args, "-all-root")
	}

	mksquashfsCmd := exec.Command(mksquashfs, args...)
	err = mksquashfsCmd.Run()
	defer os.Remove(squashfsPath)
	if err != nil {
		return
	}

	err = createSIF(path, def, squashfsPath)
	if err != nil {
		return
	}

	return
}

// changeOwner check the command being called with sudo with the environment
// variable SUDO_COMMAND. Pattern match that for the singularity bin
func changeOwner() (int, int, bool) {
	var uid, gid int
	var chown bool
	var err error

	r := regexp.MustCompile("(singularity)")
	sudoCmd := os.Getenv("SUDO_COMMAND")
	if r.MatchString(sudoCmd) {
		if syscall.Getuid() == 0 && os.Getenv("SUDO_USER") != "root" {
			_uid := os.Getenv("SUDO_UID")
			_gid := os.Getenv("SUDO_GID")
			if _uid == "" || _gid == "" {
				sylog.Errorf("Env vars SUDO_UID or SUDO_GID are not set, won't call chown over built SIF")

				return 0, 0, false
			}

			uid, err = strconv.Atoi(_uid)
			if err != nil {
				sylog.Errorf("Error while calling strconv: %v", err)

				return 0, 0, false
			}
			gid, err = strconv.Atoi(_gid)
			if err != nil {
				sylog.Errorf("Error while calling strconv : %v", err)

				return 0, 0, false
			}
			chown = true
		}
	}

	return uid, gid, chown
}
