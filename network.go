package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

const (
	GET_HTTP              = "GET"
	GITHUB_API_VERSION    = "2022-11-28"
	GITHUB_AGENT_NAME     = "Protogo App"
	PROTOC_ZIP_NAME       = "protoc-%s-%s.zip"
	LATEST_PROTOC_RELEASE = "https://api.github.com/repos/protocolbuffers/protobuf/releases/latest"
	PROTOC_BINARY_URL     = "https://github.com/protocolbuffers/protobuf/releases/download/v%s/%s"
	FLATC_ZIP_NAME        = "%s.flatc.binary%s.zip"
	LATEST_FLATC_RELEASE  = "https://api.github.com/repos/google/flatbuffers/releases/latest"
	FLATC_BINARY_URL      = "https://github.com/google/flatbuffers/releases/download/v%s/%s"
)

// Make GET HTTP request to GitHub API.
// Add "Authorization: Bearer ..." header if "PROTOGO_GITHUB_BEARER_TOKEN" environmental variable is found.
// Add "User-Agent" header for app authentication and "X-GitHub-Api-Version" for ensuring GitHub API version.
// Add "Accept" header with either "application/octet-stream" or "application/vnd.github+json" depending on the "binary" argument vaule.
// Send the request using the default HTTP client.
//
// Accept URL to make request to and boolean flag, whether binary or JSON response is expected.
// Return HTTP response pointer and error.
func makeGETRequestToGitHubAPI(url string, binary bool) (*http.Response, error) {
	req, err := http.NewRequest(GET_HTTP, url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http GET request to %s: %v", url, err)
	}

	if value, ok := os.LookupEnv("PROTOGO_GITHUB_BEARER_TOKEN"); ok {
		logrus.Debug("GitHub API authorization token set!")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", value))
	} else {
		logrus.Debug("GitHub API authorization token not set!")
	}

	req.Header.Set("X-GitHub-Api-Version", GITHUB_API_VERSION)
	req.Header.Set("User-Agent", GITHUB_AGENT_NAME)

	if binary {
		req.Header.Set("Accept", "application/octet-stream")
	} else {
		req.Header.Set("Accept", "application/vnd.github+json")
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing http GET request to %s: %v", url, err)
	}

	return res, nil
}

// Get latest protoc release tag, making GitHub API request.
// Decode JSON response and extract "tag_name" value from it.
//
// Return latest tag string pointer and error.
func getLatestProtocReleaseTag() (*string, error) {
	logrus.Debugf("Downloading latest protoc release info: %s", LATEST_PROTOC_RELEASE)
	resp, err := makeGETRequestToGitHubAPI(LATEST_PROTOC_RELEASE, false)
	if err != nil {
		return nil, fmt.Errorf("reading latest protobuf release error: %v", err)
	} else {
		defer resp.Body.Close()
	}

	logrus.Debug("Decoding latest protoc release JSON...")
	var responseJSON map[string]any
	err = json.NewDecoder(resp.Body).Decode(&responseJSON)
	if err != nil {
		return nil, fmt.Errorf("latest protobuf release info parsing error: %v", err)
	}

	logrus.Debug("Decoding latest protoc release version...")
	tag, ok := responseJSON["tag_name"]
	if !ok {
		return nil, fmt.Errorf("latest protobuf release info 'tag_name' not found in: %s", responseJSON)
	}

	logrus.Debug("Extracting version string...")
	if tagName, ok := tag.(string); ok {
		return &tagName, nil
	} else {
		return nil, fmt.Errorf("latest protobuf release info 'tag_name' field is not string, but: %v", tag)
	}
}

// Get latest flatc release tag, making GitHub API request.
// Decode JSON response and extract "tag_name" value from it.
//
// Return latest tag string pointer and error.
func getLatestFlatcReleaseTag() (*string, error) {
	logrus.Debugf("Downloading latest flatc release info: %s", LATEST_FLATC_RELEASE)
	resp, err := makeGETRequestToGitHubAPI(LATEST_FLATC_RELEASE, false)
	if err != nil {
		return nil, fmt.Errorf("reading latest flatbuffers release error: %v", err)
	} else {
		defer resp.Body.Close()
	}

	logrus.Debug("Decoding latest flatc release JSON...")
	var responseJSON map[string]any
	err = json.NewDecoder(resp.Body).Decode(&responseJSON)
	if err != nil {
		return nil, fmt.Errorf("latest flatbuffers release info parsing error: %v", err)
	}

	logrus.Debug("Decoding latest flatc release version...")
	tag, ok := responseJSON["tag_name"]
	if !ok {
		return nil, fmt.Errorf("latest flatbuffers release info 'tag_name' not found in: %s", responseJSON)
	}

	logrus.Debug("Extracting version string...")
	if tagName, ok := tag.(string); ok {
		return &tagName, nil
	} else {
		return nil, fmt.Errorf("latest flatbuffers release info 'tag_name' field is not string, but: %v", tag)
	}
}

// Download protoc compiler from GitHub releases, unpack it and save to the specified cache directory.
// Use current package GOOS and GOARCH values for exact binary location.
// Save downloaded archive to a temporary directory, remove it after unpacking.
//
// Accept protobuf compiler version (without "v" prefix) and cache directory to store compiler binaries.
// Return compiler executable path pointer and error.
func downloadProtocVersion(version, cacheDir string) (*string, error) {
	platform, err := getProtocOSandArch()
	if err != nil {
		return nil, fmt.Errorf("error parsing current OS and architecture: %v", err)
	} else {
		logrus.Debugf("Current protoc architecture: %s", *platform)
	}

	protocZip := fmt.Sprintf(PROTOC_ZIP_NAME, version, *platform)
	protocDownloadUrl := fmt.Sprintf(PROTOC_BINARY_URL, version, protocZip)

	logrus.Debugf("Downloading protoc release: %s", protocDownloadUrl)
	resp, err := makeGETRequestToGitHubAPI(protocDownloadUrl, true)
	if err != nil {
		return nil, fmt.Errorf("accessing URL '%s' error: %v", protocDownloadUrl, err)
	} else {
		defer resp.Body.Close()
	}

	protocArchive := filepath.Join(os.TempDir(), protocZip)

	logrus.Debugf("Creating protoc archive: %s", protocArchive)
	out, err := os.Create(protocArchive)
	if err != nil {
		return nil, fmt.Errorf("creating file '%s' error: %v", protocArchive, err)
	} else {
		defer out.Close()
		defer os.Remove(protocArchive)
	}

	logrus.Debugf("Populating protoc archive: %s", protocArchive)
	n, err := io.Copy(out, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("response copying error: %v", err)
	} else {
		logrus.Debugf("Downloaded file '%s' %d bytes successfully!", protocZip, n)
	}

	logrus.Debugf("Unzipping protoc archive: %s", protocArchive)
	err = unzip(protocArchive, cacheDir)
	if err != nil {
		return nil, fmt.Errorf("protoc archive unzipping error: %v", err)
	} else {
		logrus.Debugf("Protoc archive extracted successfully to: %s", cacheDir)
	}

	protocExec := filepath.Join(cacheDir, "bin", getExecutableName(PROTOC_EXECUTABLE))
	return &protocExec, nil
}

// Download flatc compiler from GitHub releases, unpack it and save to the specified cache directory.
// Use current package GOOS and GOARCH values for exact binary location.
// Save downloaded archive to a temporary directory, remove it after unpacking.
//
// Accept flatbuffers compiler version (without "v" prefix) and cache directory to store compiler binaries.
// Return compiler executable path pointer and error.
func downloadFlatcVersion(version, cacheDir string) (*string, error) {
	system, addition, err := getFlatcOSandAddition()
	if err != nil {
		return nil, fmt.Errorf("error parsing current OS and architecture: %v", err)
	} else {
		logrus.Debugf("Current flatc architecture: %s (%s)", *system, addition)
	}

	flatcZip := fmt.Sprintf(FLATC_ZIP_NAME, *system, addition)
	flatcDownloadUrl := fmt.Sprintf(FLATC_BINARY_URL, version, flatcZip)

	logrus.Debugf("Downloading flatc release: %s", flatcDownloadUrl)
	resp, err := makeGETRequestToGitHubAPI(flatcDownloadUrl, true)
	if err != nil {
		return nil, fmt.Errorf("accessing URL '%s' error: %v", flatcDownloadUrl, err)
	} else {
		defer resp.Body.Close()
	}

	flatcArchive := filepath.Join(os.TempDir(), flatcZip)

	logrus.Debugf("Creating flatc archive: %s", flatcArchive)
	out, err := os.Create(flatcArchive)
	if err != nil {
		return nil, fmt.Errorf("creating file '%s' error: %v", flatcArchive, err)
	} else {
		defer out.Close()
		defer os.Remove(flatcArchive)
	}

	logrus.Debugf("Populating flatc archive: %s", flatcArchive)
	n, err := io.Copy(out, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("response copying error: %v", err)
	} else {
		logrus.Debugf("Downloaded file '%s' %d bytes successfully!", flatcZip, n)
	}

	logrus.Debugf("Unzipping flatc archive: %s", flatcArchive)
	err = unzip(flatcArchive, cacheDir)
	if err != nil {
		return nil, fmt.Errorf("flatc archive unzipping error: %v", err)
	} else {
		logrus.Debugf("Flatc archive extracted successfully to: %s", cacheDir)
	}

	flatcExec := filepath.Join(cacheDir, getExecutableName(FLATC_EXECUTABLE))
	return &flatcExec, nil
}
