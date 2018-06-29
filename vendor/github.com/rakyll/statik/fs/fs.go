// Copyright 2014 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package fs contains an HTTP file system that works with zip contents.
package fs

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

var zipData string

// file holds unzipped read-only file contents and file metadata.
type file struct {
	os.FileInfo
	data []byte
}

type statikFS struct {
	files map[string]file
}

// Register registers zip contents data, later used to initialize
// the statik file system.
func Register(data string) {
	zipData = data
}

// New creates a new file system with the registered zip contents data.
// It unzips all files and stores them in an in-memory map.
func New() (http.FileSystem, error) {
	if zipData == "" {
		return nil, errors.New("statik/fs: no zip data registered")
	}
	zipReader, err := zip.NewReader(strings.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, err
	}
	files := make(map[string]file)
	for _, zipFile := range zipReader.File {
		unzipped, err := unzip(zipFile)
		if err != nil {
			return nil, fmt.Errorf("statik/fs: error unzipping file %q: %s", zipFile.Name, err)
		}
		files["/"+zipFile.Name] = file{
			FileInfo: zipFile.FileInfo(),
			data:     unzipped,
		}
	}
	return &statikFS{files: files}, nil
}

func unzip(zf *zip.File) ([]byte, error) {
	rc, err := zf.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return ioutil.ReadAll(rc)
}

// Open returns a file matching the given file name, or os.ErrNotExists if
// no file matching the given file name is found in the archive.
// If a directory is requested, Open returns the file named "index.html"
// in the requested directory, if that file exists.
func (fs *statikFS) Open(name string) (http.File, error) {
	name = strings.Replace(name, "//", "/", -1)
	f, ok := fs.files[name]
	if ok {
		return newHTTPFile(f, false), nil
	}
	// The file doesn't match, but maybe it's a directory,
	// thus we should look for index.html
	indexName := strings.Replace(name+"/index.html", "//", "/", -1)
	f, ok = fs.files[indexName]
	if !ok {
		return nil, os.ErrNotExist
	}
	return newHTTPFile(f, true), nil
}

func newHTTPFile(file file, isDir bool) *httpFile {
	return &httpFile{
		file:   file,
		reader: bytes.NewReader(file.data),
		isDir:  isDir,
	}
}

// httpFile represents an HTTP file and acts as a bridge
// between file and http.File.
type httpFile struct {
	file

	reader *bytes.Reader
	isDir  bool
}

// Read reads bytes into p, returns the number of read bytes.
func (f *httpFile) Read(p []byte) (n int, err error) {
	return f.reader.Read(p)
}

// Seek seeks to the offset.
func (f *httpFile) Seek(offset int64, whence int) (ret int64, err error) {
	return f.reader.Seek(offset, whence)
}

// Stat stats the file.
func (f *httpFile) Stat() (os.FileInfo, error) {
	return f, nil
}

// IsDir returns true if the file location represents a directory.
func (f *httpFile) IsDir() bool {
	return f.isDir
}

// Readdir returns an empty slice of files, directory
// listing is disabled.
func (f *httpFile) Readdir(count int) ([]os.FileInfo, error) {
	// directory listing is disabled.
	return make([]os.FileInfo, 0), nil
}

func (f *httpFile) Close() error {
	return nil
}
