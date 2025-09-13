package main

import (
	"fmt"
	"os"
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

	LINUX_ANY      = "Linux"
	MAC            = "Mac"
	MAC_INTEL      = "MacIntel"
	WINDOWS        = "Windows"
	ADDITION_CLANG = ".clang++-18"
	ADDITION_GCC   = ".g++-13"
)

func getExecutableName(executable string) string {
	switch runtime.GOOS {
	case "windows":
		return fmt.Sprintf("%s.exe", executable)
	default:
		return executable
	}
}

// Determine the string identifying protoc release binary.
//
// NB! Help needed! Maybe some other binaries are suitable for some other platforms - and maybe not!
// I could only check on GitHub Actions hosted runners.
//
// Check out [protobuf releases] for the list of supported version.
// Check out [GO documentation] for possible GOOS and GOARCH values.
//
// Return the platform string and error.
//
// [protobuf releases]: https://github.com/protocolbuffers/protobuf/releases
// [GO documentation]: https://go.dev/doc/install/source#environment
func getProtocOSandArch() (*string, error) {
	var platform string
	undefinedOS := false
	undefinedArchitecture := false

	switch runtime.GOOS {
	case "linux":
		switch runtime.GOARCH {
		case "amd64":
			platform = LINUX_AMD64
		case "386":
			platform = LINUX_AMD32
		case "s390x":
			platform = LINUX_390_64
		case "ppc64le":
			platform = LINUX_PPCLE_64
		case "arm64":
			platform = LINUX_ARM64
		default:
			undefinedArchitecture = true
		}
	case "darwin":
		switch runtime.GOARCH {
		case "amd64", "arm64":
			platform = OSX_UNIVERSAL
		default:
			undefinedArchitecture = true
		}
	case "windows":
		switch runtime.GOARCH {
		case "386", "arm":
			platform = WIN32
		case "amd64", "arm64":
			platform = WIN64
		default:
			undefinedArchitecture = true
		}
	default:
		undefinedOS = true
	}

	if undefinedOS {
		return nil, fmt.Errorf("the OS '%s' is either not supported by protogo or there are no protobuf binaries distributed for it", runtime.GOOS)
	} else if undefinedArchitecture {
		return nil, fmt.Errorf("the architecture '%s' is either not supported by protogo or there are no protobuf binaries distributed for it", runtime.GOARCH)
	}

	return &platform, nil
}

func getFlatcOSandAddition() (*string, string, error) {
	var system string
	undefinedOS := false
	undefinedArchitecture := false

	addition := ""
	switch runtime.GOOS {
	case "linux":
		system = LINUX_ANY
		if value, ok := os.LookupEnv("PROTOGO_FLATC_DISTRO"); ok {
			switch value {
			case "g++":
				addition = ADDITION_GCC
			case "clang":
				addition = ADDITION_CLANG
			}
		} else {
			addition = ADDITION_GCC
		}
	case "darwin":
		switch runtime.GOARCH {
		case "amd64":
			system = MAC_INTEL
		case "arm64":
			system = MAC
		default:
			undefinedArchitecture = true
		}
	case "windows":
		system = WINDOWS
	default:
		undefinedOS = true
	}

	if undefinedOS {
		return nil, addition, fmt.Errorf("the OS '%s' is either not supported by protogo or there are no protobuf binaries distributed for it", runtime.GOOS)
	} else if undefinedArchitecture {
		return nil, addition, fmt.Errorf("the architecture '%s' is either not supported by protogo or there are no protobuf binaries distributed for it", runtime.GOARCH)
	}

	return &system, addition, nil
}
