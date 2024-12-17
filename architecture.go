package main

import (
	"fmt"
	"runtime"
)

const (
	LINUX_AMD64    = "linux-x86_64"
	LINUX_AMD32    = "linux-x86_32"
	LINUX_390_64   = "linux-s390_64"
	LINUX_PPCLE_64 = "linux-ppcle_64"
	LINUX_ARM64    = "linux-aarch_64"
	OSX_UNIVERSAL  = "osx-universal_binary"
	WIN32          = "win32"
	WIN64          = "win64"
)

// Return a string identifying protoc release binary.
// Return an error if there is no pre-compiled protoc binary for current OS and architecture.
//
// NB! Help needed! Maybe some other binaries are suitable for some other platforms - and maybe not!
// I could only check on GitHub Actions hosted runners.
//
// Check out [protobuf releases] for the list of supported version.
//
// Check out [GO documentation] for possible GOOS and GOARCH values.
//
// [protobuf releases]: https://github.com/protocolbuffers/protobuf/releases
// [GO documentation]: https://go.dev/doc/install/source#environment
func getProtocOSandArch() (*string, error) {
	var platform string
	undefinedOS := false
	undefinedArchitecture := false

	if runtime.GOOS == "linux" {
		if runtime.GOARCH == "amd64" {
			platform = LINUX_AMD64
		} else if runtime.GOARCH == "386" {
			platform = LINUX_AMD32
		} else if runtime.GOARCH == "s390x" {
			platform = LINUX_390_64
		} else if runtime.GOARCH == "ppc64le" {
			platform = LINUX_PPCLE_64
		} else if runtime.GOARCH == "arm64" {
			platform = LINUX_ARM64
		} else {
			undefinedArchitecture = true
		}
	} else if runtime.GOOS == "darwin" {
		if runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64" {
			platform = OSX_UNIVERSAL
		} else {
			undefinedArchitecture = true
		}
	} else if runtime.GOOS == "windows" {
		if runtime.GOARCH == "386" || runtime.GOARCH == "arm" {
			platform = WIN32
		} else if runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64" {
			platform = WIN64
		} else {
			undefinedArchitecture = true
		}
	} else {
		undefinedOS = true
	}

	if undefinedOS {
		return nil, fmt.Errorf("the OS '%s' is either not supported by protogo or there are no protobuf binaries distributed for it", runtime.GOOS)
	} else if undefinedArchitecture {
		return nil, fmt.Errorf("the architecture '%s' is either not supported by protogo or there are no protobuf binaries distributed for it", runtime.GOARCH)
	}

	return &platform, nil
}
