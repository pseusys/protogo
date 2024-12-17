package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

const (
	PROTOC_GEN_GO_PACKAGE      = "protoc-gen-go"
	PROTOC_GEN_GO_PREFIX       = "google.golang.org/protobuf/cmd"
	PROTOC_GEN_GO_GRPC_PACKAGE = "protoc-gen-go-grpc"
	PROTOC_GEN_GO_GRPC_PREFIX  = "google.golang.org/grpc/cmd"
)

const HELP_TEXT = `    'protogo' is an automatization tool for Go + protobuf + gRPC builds!
You can run it with the same arguments as 'go' executable, followed by '--' flag and then 'protoc' arguments.
Protogo will handle everything else, including 'protoc' binaries installation, installing required packages, etc.
Use official gRPC installation guide as reference: https://grpc.io/docs/languages/go/quickstart/#prerequisites.
Inspired by similar projects for other languages, including https://pypi.org/project/protoc-exe/ and https://crates.io/crates/protoc-prebuilt/.
You can additionally control it with the following environment variables:
  - PROTOGO_GO_EXECUTABLE: define 'go' executable to use, default: go
  - PROTOGO_PROTOC_VERSION: defing 'protoc' version to use, should match protobuf release tags, default: latest
      NB! If 'local' is specified as 'protoc' version, local installation will be used
  - PROTOGO_CACHE: define cache directory, where 'protobuf' executables will be stored, default: ~/.cache/protogo
  - PROTOGO_LOG_LEVEL: define logging level, the levels match 'logrus' ones.`

func init() {
	var unparsedLevel string

	if value, ok := os.LookupEnv("PROTOGO_LOG_LEVEL"); ok {
		unparsedLevel = value
	} else {
		unparsedLevel = "INFO"
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

	var protocArgs []string
	protocArgStart := argsDelim + 1
	if protocArgStart < argLen {
		protocArgs = os.Args[protocArgStart:argLen]
		logrus.Debugf("Protoc command arguments parsed: %v", protocArgs)
	}

	logrus.Debug("Checking cache directory location...")
	protogoCache, err := getEnvCacheDir("PROTOGO_CACHE")
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

	logrus.Debug("Extracting required protoc version...")
	protocTag, protocCache, shouldDownload, err := getProtocCache("PROTOGO_PROTOC_VERSION", *protogoCache)
	if err != nil {
		logrus.Fatalf("Could not find or load protoc executable: %v", err)
	} else if protocCache != nil {
		logrus.Debugf("Protoc version requested: %s, cache location: %s, will be downloaded: %t", *protocTag, *protocCache, shouldDownload)
	} else {
		logrus.Debugf("Protoc version requested: %s, system default, will be downloaded: %t", *protocTag, shouldDownload)
	}

	var protocExecutable string
	if shouldDownload {
		logrus.Debug("Downloading protoc executable...")
		protocExec, err := downloadProtocVersion(*protocTag, *protocCache)
		if err != nil {
			logrus.Fatalf("Could not download or extract protoc: %v", err)
		}
		protocExecutable = *protocExec
		logrus.Debugf("Protoc executable downloaded to: %s", protocExecutable)
	} else if protocCache != nil {
		protocExecutable = filepath.Join(*protocCache, "bin", PROTOC_EXECUTABLE)
		logrus.Debugf("Protoc executable found at: %s", protocExecutable)
	} else {
		protocExecutable = PROTOC_EXECUTABLE
		logrus.Debugf("Protoc executable found at: %s", protocExecutable)
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

	protocPath := fmt.Sprintf("PATH=%s:%s", os.Getenv("PATH"), *goBin)
	logrus.Debugf("Protoc will be executed with following PATH: %s", protocPath)

	if len(protocArgs) > 0 {
		logrus.Debugf("Running protoc command: %s %v", protocExecutable, protocArgs)
		protocCmd := exec.Command(protocExecutable, protocArgs...)
		protocCmd.Env = append(protocCmd.Env, protocPath)
		protocCmd.Stderr = os.Stderr
		protocCmd.Stdout = os.Stdout
		err = protocCmd.Run()
		if err != nil {
			logrus.Fatalf("Protoc execution failed: %v", err)
		}
	} else {
		logrus.Debugf("No protoc arguments were supplied, skipping protoc execution!")
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
		logrus.Debugf("No GO arguments were supplied, skipping GO execution!")
	}
}
