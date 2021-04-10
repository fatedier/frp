// Copyright 2016 fatedier, fatedier@gmail.com
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package assets

import (
	"io/fs"
	"io/ioutil"
	"net/http"
	"os"
	"path"
)

var (
	// store static files in memory by go embed
	FileSystem http.FileSystem

	// if prefix is not empty, we get file content from disk
	prefixPath string

	StaticFiles fs.FS
)

// if path is empty, load assets in memory
// or set FileSystem using disk files
func Load(path string) (err error) {
	prefixPath = path
	if prefixPath != "" {
		FileSystem = http.Dir(prefixPath)
		return nil
	} else {
		FileSystem = http.FS(StaticFiles)
	}
	return err
}

func ReadFile(file string) (content string, err error) {
	if prefixPath == "" {
		file, err := FileSystem.Open(path.Join("/", file))
		if err != nil {
			return content, err
		}
		defer file.Close()
		buf, err := ioutil.ReadAll(file)
		if err != nil {
			return content, err
		}
		content = string(buf)
	} else {
		file, err := os.Open(path.Join(prefixPath, file))
		if err != nil {
			return content, err
		}
		defer file.Close()
		buf, err := ioutil.ReadAll(file)
		if err != nil {
			return content, err
		}
		content = string(buf)
	}
	return content, err
}
