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

// Package contains an HTTP file system that works with zip contents.
package fs

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"sync"
)

var zipData string

type statikFS struct {
	files map[string]*zip.File
}

// Registers zip contents data, later used to initialize
// the statik file system.
func Register(data string) {
	zipData = data
}

// Creates a new file system with the registered zip contents data.
func New() (http.FileSystem, error) {
	if zipData == "" {
		return nil, errors.New("statik/fs: No zip data registered.")
	}
	zipReader, err := zip.NewReader(strings.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, err
	}
	files := make(map[string]*zip.File)
	for _, file := range zipReader.File {
		files["/"+file.Name] = file
	}
	return &statikFS{files: files}, nil
}

// Opens a file, unzip the contents and initializes
// readers. Returns os.ErrNotExists if file is not
// found in the archive.
func (fs *statikFS) Open(name string) (http.File, error) {
	name = strings.Replace(name, "//", "/", -1)
	f, ok := fs.files[name]

	// The file doesn't match, but maybe it's a directory,
	// thus we should look for index.html
	if !ok {
		indexName := strings.Replace(name+"/index.html", "//", "/", -1)
		f, ok = fs.files[indexName]

		if !ok {
			return nil, os.ErrNotExist
		}

		return newFile(f, true)
	}
	return newFile(f, false)
}

var nopCloser = ioutil.NopCloser(nil)

func newFile(zf *zip.File, isDir bool) (*file, error) {
	rc, err := zf.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	all, err := ioutil.ReadAll(rc)
	if err != nil {
		return nil, err
	}
	return &file{
		FileInfo: zf.FileInfo(),
		data:     all,
		readerAt: bytes.NewReader(all),
		Closer:   nopCloser,
		isDir:    isDir,
	}, nil
}

// Represents an HTTP file, acts as a bridge between
// zip.File and http.File.
type file struct {
	os.FileInfo
	io.Closer

	data     []byte // non-nil if regular file
	reader   *io.SectionReader
	readerAt io.ReaderAt // over data
	isDir    bool

	once sync.Once
}

func (f *file) newReader() {
	f.reader = io.NewSectionReader(f.readerAt, 0, f.FileInfo.Size())
}

// Reads bytes into p, returns the number of read bytes.
func (f *file) Read(p []byte) (n int, err error) {
	f.once.Do(f.newReader)
	return f.reader.Read(p)
}

// Seeks to the offset.
func (f *file) Seek(offset int64, whence int) (ret int64, err error) {
	f.once.Do(f.newReader)
	return f.reader.Seek(offset, whence)
}

// Stats the file.
func (f *file) Stat() (os.FileInfo, error) {
	return f, nil
}

// IsDir returns true if the file location represents a directory.
func (f *file) IsDir() bool {
	return f.isDir
}

// Returns an empty slice of files, directory
// listing is disabled.
func (f *file) Readdir(count int) ([]os.FileInfo, error) {
	// directory listing is disabled.
	return make([]os.FileInfo, 0), nil
}
