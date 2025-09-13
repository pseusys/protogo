// Package provides automated protoc compiler downloading and preparing environment pipeline, for use of GO with protobuf and gRPC.
//
// The description of the pipeline being automated can be found on [official gRPC GO quickstart website].
//
// [official gRPC GO quickstart website]: https://grpc.io/docs/languages/go/quickstart/#prerequisites
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"

	"github.com/sirupsen/logrus"
)

const (
	PROTOC_GEN_GO_PACKAGE      = "protoc-gen-go"
	PROTOC_GEN_GO_PREFIX       = "google.golang.org/protobuf/cmd"
	PROTOC_GEN_GO_GRPC_PACKAGE = "protoc-gen-go-grpc"
	PROTOC_GEN_GO_GRPC_PREFIX  = "google.golang.org/grpc/cmd"
)

// `protogo` package help string.
const HELP_TEXT = `    'protogo' is an automatization tool for Go + protobuf / flatbuffers + gRPC builds!
You can run it with the same arguments as 'go' executable, followed by '--' flag and then compiler name ('protoc' or 'flatc') and its arguments.
Protogo will handle everything else, including compiler binaries installation, installing required packages, etc.
Use official gRPC installation guide as reference for protobuf: https://grpc.io/docs/languages/go/quickstart/#prerequisites.
Use official gRPC installation guide as reference for flatbuffers: https://flatbuffers.dev/languages/go/.
Inspired by similar projects for other languages, including https://pypi.org/project/protoc-exe/ and https://crates.io/crates/protoc-prebuilt/.
You can additionally control it with the following environment variables:
  - PROTOGO_GO_EXECUTABLE: define 'go' executable to use, default: go
  - PROTOGO_PROTOC_VERSION: defing 'protoc' version to use, should match protobuf release tags, default: latest
      NB! If 'local' is specified as 'protoc' version, local installation will be used
  - PROTOGO_FLATC_VERSION: defing 'flatc' version to use, should match protobuf release tags, default: latest
      NB! If 'local' is specified as 'flatc' version, local installation will be used
  - PROTOGO_FLATC_DISTRO: select distribution of 'flatc' for linux (can be either 'g++' or 'clang', default 'g++')
  - PROTOGO_CACHE: define cache directory, where 'protobuf' executables will be stored, default: ~/.cache/protogo
  - PROTOGO_GITHUB_BEARER_TOKEN: GitHub authentication token for API requests (release assets retrieval)
  - PROTOGO_LOG_LEVEL: define logging level, the levels match 'logrus' ones`

func init() {
	var unparsedLevel string

	if value, ok := os.LookupEnv("PROTOGO_LOG_LEVEL"); ok {
		unparsedLevel = value
	} else {
		unparsedLevel = "WARN"
	}

	level, err := logrus.ParseLevel(unparsedLevel)
	if err != nil {
		logrus.Fatalf("Error parsing log level environmental variable: %v", unparsedLevel)
	}

	logrus.SetLevel(level)
}

func main() {
	var err error

	argsDelim := -1
	argLen := len(os.Args)
	for i := 1; i < argLen; i++ {
		if os.Args[i] == "--" {
			argsDelim = i
			break
		}
	}

	logrus.Debugf("Running protogo (delim: %d) with arguments: %v", argsDelim, os.Args)
	if argsDelim == -1 {
		fmt.Println(HELP_TEXT)
		os.Exit(0)
	}

	var goArgs []string
	if argsDelim > 0 {
		goArgs = os.Args[1:argsDelim]
		logrus.Debugf("GO command arguments parsed: %v", goArgs)
	}

	var compiler string
	compilerNameArg := argsDelim + 1
	if compilerNameArg < argLen {
		compiler = os.Args[compilerNameArg]
		logrus.Debugf("Compiler command parsed: %v", compiler)
	} else {
		logrus.Debug("Compiler command not found, so will be ignored!")
	}

	var compilerArgs []string
	compilerArgStart := compilerNameArg + 1
	if compilerArgStart < argLen {
		compilerArgs = os.Args[compilerArgStart:argLen]
		logrus.Debugf("Compiler command arguments parsed: %v", compilerArgs)
	}

	logrus.Debug("Checking compiler name...")
	if !slices.Contains([]string{PROTOC_EXECUTABLE, FLATC_EXECUTABLE, NONE_EXECUTABLE}, compiler) {
		logrus.Errorf("Unknown compiler requested: %s", compiler)
		os.Exit(1)
	}

	logrus.Debug("Checking cache directory location...")
	protogoCache, err := getProtogoCacheDir("PROTOGO_CACHE")
	if err != nil {
		logrus.Fatalf("Could not find or create cache directory: %v", err)
	} else {
		logrus.Debugf("Cache directory found: %s", *protogoCache)
	}

	logrus.Debug("Checking GO executable...")
	goExec, err := getGoExecutable("PROTOGO_GO_EXECUTABLE")
	if err != nil {
		logrus.Fatalf("Could not find go executable: %v", err)
	} else {
		logrus.Debugf("GO executable found: %s", *goExec)
	}

	logrus.Debug("Checking GO binary location...")
	goBin, err := getGoBinaryLocation(*goExec)
	if err != nil {
		logrus.Fatalf("Could not find go binary location: %v", err)
	} else {
		logrus.Debugf("GO binary location found: %s", *goBin)
	}

	var compilerExecutable string
	switch compiler {
	case PROTOC_EXECUTABLE:
		logrus.Debug("Extracting required compiler version...")
		protocTag, protocCache, shouldDownload, err := getProtocCache("PROTOGO_PROTOC_VERSION", *protogoCache)
		if err != nil {
			logrus.Fatalf("Could not find or load protoc executable: %v", err)
		} else if protocCache != nil {
			logrus.Debugf("Protoc version requested: %s, cache location: %s, will be downloaded: %t", *protocTag, *protocCache, shouldDownload)
		} else {
			logrus.Debugf("Protoc version requested: %s, system default, will be downloaded: %t", *protocTag, shouldDownload)
		}

		if shouldDownload {
			logrus.Debug("Downloading protoc executable...")
			protocExec, err := downloadProtocVersion(*protocTag, *protocCache)
			if err != nil {
				logrus.Fatalf("Could not download or extract protoc: %v", err)
			}
			compilerExecutable = *protocExec
			logrus.Debugf("Protoc executable downloaded to: %s", compilerExecutable)
		} else if protocCache != nil {
			compilerExecutable = filepath.Join(*protocCache, "bin", getExecutableName(PROTOC_EXECUTABLE))
			logrus.Debugf("Protoc executable found at: %s", compilerExecutable)
		} else {
			compilerExecutable = PROTOC_EXECUTABLE
			logrus.Debugf("Protoc executable found at: %s", compilerExecutable)
		}

		err = ensureGoPackageInstalled(*goExec, *goBin, PROTOC_GEN_GO_PREFIX, PROTOC_GEN_GO_PACKAGE)
		if err != nil {
			logrus.Fatalf("Could not find or install package %s: %v", PROTOC_GEN_GO_PACKAGE, err)
		} else {
			logrus.Debugf("Package %s found or installed successfully!", PROTOC_GEN_GO_PACKAGE)
		}

		err = ensureGoPackageInstalled(*goExec, *goBin, PROTOC_GEN_GO_GRPC_PREFIX, PROTOC_GEN_GO_GRPC_PACKAGE)
		if err != nil {
			logrus.Fatalf("Could not find or install package %s: %v", PROTOC_GEN_GO_GRPC_PACKAGE, err)
		} else {
			logrus.Debugf("Package %s found or installed successfully!", PROTOC_GEN_GO_GRPC_PACKAGE)
		}

	case FLATC_EXECUTABLE:
		logrus.Debug("Extracting required compiler version...")
		flatcTag, flatcCache, shouldDownload, err := getFlatcCache("PROTOGO_FLATC_VERSION", *protogoCache)
		if err != nil {
			logrus.Fatalf("Could not find or load flatc executable: %v", err)
		} else if flatcCache != nil {
			logrus.Debugf("Flatc version requested: %s, cache location: %s, will be downloaded: %t", *flatcTag, *flatcCache, shouldDownload)
		} else {
			logrus.Debugf("Flatc version requested: %s, system default, will be downloaded: %t", *flatcTag, shouldDownload)
		}

		if shouldDownload {
			logrus.Debug("Downloading flatc executable...")
			flatcExec, err := downloadFlatcVersion(*flatcTag, *flatcCache)
			if err != nil {
				logrus.Fatalf("Could not download or extract flatc: %v", err)
			}
			compilerExecutable = *flatcExec
			logrus.Debugf("Flatc executable downloaded to: %s", compilerExecutable)
		} else if flatcCache != nil {
			compilerExecutable = filepath.Join(*flatcCache, getExecutableName(FLATC_EXECUTABLE))
			logrus.Debugf("Flatc executable found at: %s", compilerExecutable)
		} else {
			compilerExecutable = FLATC_EXECUTABLE
			logrus.Debugf("Flatc executable found at: %s", compilerExecutable)
		}

	default:
		logrus.Debug("No compiler supplied, so installation skipped!")
	}

	if len(compilerArgs) > 0 {
		compilerPath := fmt.Sprintf("PATH=%s%c%s", os.Getenv("PATH"), os.PathListSeparator, *goBin)
		logrus.Debugf("Compiler will be executed with following PATH: %s", compilerPath)

		logrus.Debugf("Running compiler command: %s %v", compilerExecutable, compilerArgs)
		compilerCmd := exec.Command(compilerExecutable, compilerArgs...)
		compilerCmd.Env = append(compilerCmd.Environ(), compilerPath)
		compilerCmd.Stderr = os.Stderr
		compilerCmd.Stdout = os.Stdout
		err = compilerCmd.Run()
		if err != nil {
			logrus.Fatalf("Compiler execution failed: %v", err)
		}
	} else {
		logrus.Debug("No compiler arguments were supplied, skipping compiler execution!")
	}

	if len(goArgs) > 0 {
		logrus.Debugf("Running GO command: %s %v", *goExec, goArgs)
		goCmd := exec.Command(*goExec, goArgs...)
		goCmd.Stderr = os.Stderr
		goCmd.Stdout = os.Stdout
		err = goCmd.Run()
		if err != nil {
			logrus.Fatalf("GO execution failed: %v", err)
		}
	} else {
		logrus.Debug("No GO arguments were supplied, skipping GO execution!")
	}
}
