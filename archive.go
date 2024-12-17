package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func extractFile(reader io.ReadCloser, file *zip.File, path string) error {
	var fdir string
	if lastIndex := strings.LastIndex(path, string(os.PathSeparator)); lastIndex > -1 {
		fdir = path[:lastIndex]
	}

	err := os.MkdirAll(fdir, file.Mode())
	if err != nil {
		return fmt.Errorf("error making directory %s: %v", fdir, err)
	}

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
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

func extractItem(file *zip.File, dest string) error {
	rc, err := file.Open()
	if err != nil {
		return fmt.Errorf("error opening object %s: %v", file.Name, err)
	} else {
		defer rc.Close()
	}

	fpath := filepath.Join(dest, file.Name)

	if file.FileInfo().IsDir() {
		err = os.MkdirAll(fpath, file.Mode())
		if err != nil {
			return fmt.Errorf("error making directory %s: %v", fpath, err)
		}
	} else {
		err = extractFile(rc, file, fpath)
		if err != nil {
			return fmt.Errorf("error extracting file %s: %v", fpath, err)
		}
	}

	return nil
}

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
