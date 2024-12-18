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
)

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

func getLatestProtocRelease() (*string, error) {
	logrus.Debugf("Downloading latest protoc release info: %s", LATEST_PROTOC_RELEASE)
	resp, err := makeGETRequestToGitHubAPI(LATEST_PROTOC_RELEASE, false)
	if err != nil {
		return nil, fmt.Errorf("reading latest protobuf release error: %v", err)
	} else {
		defer resp.Body.Close()
	}

	logrus.Debug("Decoding latest protoc release JSON...")
	var responseJSON map[string]interface{}
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

	protocExec := filepath.Join(cacheDir, "bin", PROTOC_EXECUTABLE)
	return &protocExec, nil
}
