package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

const (
	GO_EXECUTABLE     = "go"
	NONE_EXECUTABLE   = ""
	PROTOC_EXECUTABLE = "protoc"
	FLATC_EXECUTABLE  = "flatc"
)

// Get GO environmental variable by running "go env ..." command.
// Return empty string if not found.
//
// Accept GO executable path and environment variable name.
// Return environment variable value and boolean flag, whether variable was found.
func lookupGoEnv(goExecuteble, key string) (string, bool) {
	cmd := exec.Command(goExecuteble, "env", key)
	output, err := cmd.Output()
	if err != nil || (len(output) == 1 && output[0] == '\n') {
		return "", false
	}

	return strings.TrimSuffix(string(output), "\n"), true
}

// Find GO executable, either locally or by provided path.
// Verify the executable exists.
//
// Accept custom GO executable path envieonment variable (or empty string if none).
// Return verified GO executable pointer and error.
func getGoExecutable(key string) (*string, error) {
	var executable string

	if value, ok := os.LookupEnv(key); ok {
		executable = value
	} else {
		executable = getExecutableName(GO_EXECUTABLE)
	}

	logrus.Debugf("Looking up for GO executable: %s", executable)
	_, err := exec.LookPath(executable)
	if err != nil {
		return nil, fmt.Errorf("go executable couldn't be found: %v", err)
	} else {
		logrus.Debug("GO executable found!")
	}

	return &executable, nil
}

// Find GO binary directory location.
// Just like "go install ..." [documentation] suggests, all possible binary locations are searched.
//
// Accept GO executable path.
// Return GO binary directory path pointer and error.
//
// [documentation]: https://pkg.go.dev/cmd/go#hdr-Compile_and_install_packages_and_dependencies
func getGoBinaryLocation(goExecuteble string) (*string, error) {
	var binary string

	if value, ok := lookupGoEnv(goExecuteble, "GOBIN"); ok {
		binary = value
	} else if value, ok := lookupGoEnv(goExecuteble, "GOPATH"); ok {
		binary = filepath.Join(value, "bin")
	} else {
		userHome, err := os.UserHomeDir()
		if err != nil {
			return nil, errors.New("user home directory couldn't be resolved")
		}
		binary = filepath.Join(userHome, "go", "bin")
	}

	return &binary, nil
}

// Get "protogo" package cache directory.
// Is either specified by environmental variable or placed into [default cache directory].
// Create the directory if it doesn't exist.
//
// Accept custom cache directory environment variable (or empty string if none).
// Return cache directory path pointer and error.
//
// [default cache directory]: https://pkg.go.dev/os#UserCacheDir
func getProtogoCacheDir(key string) (*string, error) {
	var cacheDir string

	if value, ok := os.LookupEnv(key); ok {
		cacheDir = value
	} else {
		userCache, err := os.UserCacheDir()
		if err != nil {
			return nil, errors.New("user cache directory couldn't be resolved")
		}
		cacheDir = filepath.Join(userCache, "protogo")
	}

	logrus.Debugf("Creating cache dir: %s", cacheDir)
	err := os.MkdirAll(cacheDir, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("could not create cache directory in '%s' root: %v", cacheDir, err)
	} else {
		logrus.Debug("Cache dir created!")
	}

	return &cacheDir, nil
}

// Get cached protobuf compiler by version.
// Resolve requested protobuf version, find out the exact version name for "latest".
// Verify "protoc" is installed locally, if "local" is specified as version.
// Search for the required version directory in cache otherwise.
//
// Accept protobuf compiler version environment variable (with or without "v" prefix, empty string if none) and cache root path.
// Return version tag string pointer, cache directory for the given version (or nil for "local"), boolean flag, whether protoc binary should be downloaded, and error.
func getProtocCache(key, cacheDir string) (*string, *string, bool, error) {
	var versionTag string

	if value, ok := os.LookupEnv(key); ok {
		versionTag = value
	} else {
		versionTag = "latest"
	}

	logrus.Debugf("Requested version tag is: %s", versionTag)
	switch versionTag {
	case "latest":
		latestTag, err := getLatestProtocReleaseTag()
		if err != nil {
			return nil, nil, false, fmt.Errorf("latest protoc version tag couldn't be resolved: %v", err)
		}
		versionTag = *latestTag
	case "local":
		_, err := exec.LookPath(PROTOC_EXECUTABLE)
		if err != nil {
			return nil, nil, false, fmt.Errorf("protoc executable couldn't be found: %v", err)
		} else {
			return &versionTag, nil, false, nil
		}
	}

	versionTag = strings.TrimPrefix(versionTag, "v")
	protocCache := filepath.Join(cacheDir, fmt.Sprintf("protoc-%s", versionTag))
	protocExec := filepath.Join(protocCache, "bin", getExecutableName(PROTOC_EXECUTABLE))

	_, err := os.Stat(protocExec)
	if err != nil {
		return &versionTag, &protocCache, true, nil
	} else {
		return &versionTag, &protocCache, false, nil
	}
}

// Get cached flatbuffers compiler by version.
// Resolve requested flatbuffers version, find out the exact version name for "latest".
// Verify "flatc" is installed locally, if "local" is specified as version.
// Search for the required version directory in cache otherwise.
//
// Accept flatbuffers compiler version environment variable (with or without "v" prefix, empty string if none) and cache root path.
// Return version tag string pointer, cache directory for the given version (or nil for "local"), boolean flag, whether flatc binary should be downloaded, and error.
func getFlatcCache(key, cacheDir string) (*string, *string, bool, error) {
	var versionTag string

	if value, ok := os.LookupEnv(key); ok {
		versionTag = value
	} else {
		versionTag = "latest"
	}

	logrus.Debugf("Requested version tag is: %s", versionTag)
	switch versionTag {
	case "latest":
		latestTag, err := getLatestFlatcReleaseTag()
		if err != nil {
			return nil, nil, false, fmt.Errorf("latest flatc version tag couldn't be resolved: %v", err)
		}
		versionTag = *latestTag
	case "local":
		_, err := exec.LookPath(FLATC_EXECUTABLE)
		if err != nil {
			return nil, nil, false, fmt.Errorf("flatc executable couldn't be found: %v", err)
		} else {
			return &versionTag, nil, false, nil
		}
	}

	versionTag = strings.TrimPrefix(versionTag, "v")
	flatcCache := filepath.Join(cacheDir, fmt.Sprintf("flatc-%s", versionTag))
	flatcExec := filepath.Join(flatcCache, getExecutableName(FLATC_EXECUTABLE))

	dir, err := os.Stat(flatcExec)
	if err != nil || !dir.IsDir() {
		return &versionTag, &flatcCache, true, nil
	} else {
		return &versionTag, &flatcCache, false, nil
	}
}

// Ensure GO binary (command) is installed locally.
// Search for the package in the GO binary directory.
// Install the package if it is not found (ensure correct GOOS and GOARCH during installation).
// Search for the package in the GO binary directory again.
//
// Accept GO executable path, GO binary directory path, package prefix (without name) and package (command) name.
// Return error.
func ensureGoPackageInstalled(goExecutable, goBin, packagePrefix, packageName string) error {
	packageExecutable := filepath.Join(goBin, packageName)

	_, err := exec.LookPath(packageExecutable)
	if err == nil {
		return nil
	}

	packageUrl := fmt.Sprintf("%s/%s@latest", packagePrefix, packageName)
	logrus.Debugf("Package %s is not installed, installing latest version from: %s", packageName, packageUrl)
	cmd := exec.Command(goExecutable, "install", packageUrl)
	cmd.Env = append(cmd.Environ(), fmt.Sprintf("GOOS=%s", runtime.GOOS), fmt.Sprintf("GOARCH=%s", runtime.GOARCH))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error installing package %s: %v\n%s", packageName, err, string(output))
	}

	_, err = exec.LookPath(packageExecutable)
	if err != nil {
		return fmt.Errorf("after installation, still could not find package %s: %v", packageName, err)
	}

	return nil
}
