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
	PROTOC_EXECUTABLE = "protoc"
	FULL_PERMISSIONS  = 0775
)

func lookupGoEnv(goExecuteble, key string) (string, bool) {
	cmd := exec.Command(goExecuteble, "env", key)
	output, err := cmd.Output()
	if err != nil || (len(output) == 1 && output[0] == '\n') {
		return "", false
	}

	return strings.TrimSuffix(string(output), "\n"), true
}

func getGoExecutable(key string) (*string, error) {
	var executable string

	if value, ok := os.LookupEnv(key); ok {
		executable = value
	} else {
		executable = GO_EXECUTABLE
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

func getEnvCacheDir(key string) (*string, error) {
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
	err := os.MkdirAll(cacheDir, FULL_PERMISSIONS)
	if err != nil {
		return nil, fmt.Errorf("could not create cache directory in '~/.cache' root: %v", err)
	} else {
		logrus.Debug("Cache dir created!")
	}

	return &cacheDir, nil
}

func getProtocCache(key, cacheDir string) (*string, *string, bool, error) {
	var versionTag string

	if value, ok := os.LookupEnv(key); ok {
		versionTag = value
	} else {
		versionTag = "latest"
	}

	logrus.Debugf("Requested version tag is: %s", versionTag)
	if versionTag == "latest" {
		latestTag, err := getLatestProtocRelease()
		if err != nil {
			return nil, nil, false, fmt.Errorf("latest protoc version tag couldn't be resolved: %v", err)
		}
		versionTag = *latestTag
	} else if versionTag == "local" {
		_, err := exec.LookPath(PROTOC_EXECUTABLE)
		if err != nil {
			return nil, nil, false, fmt.Errorf("protoc executable couldn't be found: %v", err)
		} else {
			return &versionTag, nil, false, nil
		}
	}

	versionTag = strings.TrimPrefix(versionTag, "v")
	protoCache := filepath.Join(cacheDir, fmt.Sprintf("protoc-%s", versionTag))

	dir, err := os.Stat(protoCache)
	if err != nil || !dir.IsDir() {
		return &versionTag, &protoCache, true, nil
	} else {
		return &versionTag, &protoCache, false, nil
	}
}

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
