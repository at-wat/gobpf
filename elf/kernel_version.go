//+build linux

// Copyright 2016-2017 Kinvolk
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package elf

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"
	"syscall"
)

func utsnameStr(in []int8) string {
	out := make([]byte, len(in))
	for i := 0; i < len(in); i++ {
		if in[i] == 0 {
			break
		}
		out = append(out, byte(in[i]))
	}
	return string(out)
}

var versionRegex = regexp.MustCompile(`^(\d+)\.(\d+).(\d+).*$`)

func kernelVersionFromReleaseString(releaseString string) (int, error) {
	versionParts := versionRegex.FindStringSubmatch(releaseString)
	if len(versionParts) != 4 {
		return -1, fmt.Errorf("got invalid release version %q (expected format '4.3.2-1')", releaseString)
	}
	major, err := strconv.Atoi(versionParts[1])
	if err != nil {
		return -1, err
	}

	minor, err := strconv.Atoi(versionParts[2])
	if err != nil {
		return -1, err
	}

	patch, err := strconv.Atoi(versionParts[3])
	if err != nil {
		return -1, err
	}
	out := major*256*256 + minor*256 + patch
	return out, nil
}

func currentVersionUname() (int, error) {
	var buf syscall.Utsname
	if err := syscall.Uname(&buf); err != nil {
		return -1, err
	}
	releaseString := strings.Trim(utsnameStr(buf.Release[:]), "\x00")
	return kernelVersionFromReleaseString(releaseString)
}

func currentVersionUbuntu() (int, error) {
	procVersion, err := ioutil.ReadFile("/proc/version_signature")
	if err != nil {
		return -1, err
	}
	var u1, u2, releaseString string
	_, err = fmt.Sscanf(string(procVersion), "%s %s %s", &u1, &u2, &releaseString)
	if err != nil {
		return -1, err
	}
	return kernelVersionFromReleaseString(releaseString)
}

var debianVersionRegex = regexp.MustCompile(`.* SMP Debian (\d+\.\d+.\d+-\d+) .*`)

func currentVersionDebian() (int, error) {
	procVersion, err := ioutil.ReadFile("/proc/version")
	if err != nil {
		return -1, err
	}
	match := debianVersionRegex.FindStringSubmatch(string(procVersion))
	if len(match) != 2 {
		return -1, fmt.Errorf("failed to get kernel version from /proc/version: %s", procVersion)
	}
	return kernelVersionFromReleaseString(match[1])
}

func currentVersion() (int, error) {
	// We need extra checks for Debian and Ubuntu as they modify
	// the kernel version patch number for compatibilty with
	// out-of-tree modules. Linux perf tools do the same for Ubuntu
	// systems: https://github.com/torvalds/linux/commit/d18acd15c
	//
	// See also:
	// https://kernel-handbook.alioth.debian.org/ch-versions.html
	// https://wiki.ubuntu.com/Kernel/FAQ
	version, err := currentVersionUbuntu()
	if err == nil {
		return version, nil
	}
	version, err = currentVersionDebian()
	if err == nil {
		return version, nil
	}
	return currentVersionUname()
}