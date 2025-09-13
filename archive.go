// This whole file is based on the awesome [stackoverflow answer] by swtdrgn.
//
// [stackoverflow answer]: https://stackoverflow.com/a/24430720/9124072

package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Extract a file from ZIP archive.
// Make all the parent directories, if needed.
//
// Accept ZIP file and destination path.
// Return error.
func extractFile(file *zip.File, path string) error {
	fdir := filepath.Dir(path)

	reader, err := file.Open()
	if err != nil {
		return fmt.Errorf("error opening object %s: %v", file.Name, err)
	} else {
		defer reader.Close()
	}

	err = os.MkdirAll(fdir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("error making directory %s: %v", fdir, err)
	}

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return fmt.Errorf("error opening file %s: %v", path, err)
	} else {
		defer f.Close()
	}

	_, err = io.Copy(f, reader)
	if err != nil {
		return fmt.Errorf("error copying file contents %s: %v", file.Name, err)
	}

	return nil
}

// Extract any item from ZIP archive.
// If it is a directory, create corresponding directory in the target location.
// If it is a file, extract this file.
//
// Accept ZIP file and destination path.
// Return error.
func extractItem(file *zip.File, dest string) error {
	fpath, err := filepath.Abs(filepath.Join(dest, file.Name))
	if err != nil {
		return fmt.Errorf("error resolving path: %s", fpath)
	} else if !strings.Contains(fpath, dest) {
		return fmt.Errorf("error extracting path: %s (%v)", fpath, dest)
	}

	if file.FileInfo().IsDir() {
		err := os.MkdirAll(fpath, os.ModePerm)
		if err != nil {
			return fmt.Errorf("error making directory %s: %v", fpath, err)
		}
	} else {
		err := extractFile(file, fpath)
		if err != nil {
			return fmt.Errorf("error extracting file %s: %v", fpath, err)
		}
	}

	return nil
}

// Extract ZIP archive.
// Set current user permissions to all the extracted files and directories.
// Replace any existing files, if they are found.
//
// Accept source ZIP archive path and destination extraction directory path.
// Return error.
func unzip(src, dest string) error {
	reader, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("error opening archive reader %s: %v", src, err)
	} else {
		defer reader.Close()
	}

	for _, f := range reader.File {
		err = extractItem(f, dest)
		if err != nil {
			return fmt.Errorf("error extracting file %s: %v", f.Name, err)
		}
	}

	return nil
}
