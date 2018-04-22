// Copyright © 2015 Steve Francia <spf@spf13.com>.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

var srcPaths []string

func init() {
	// Initialize srcPaths.
	envGoPath := os.Getenv("GOPATH")
	goPaths := filepath.SplitList(envGoPath)
	if len(goPaths) == 0 {
		// Adapted from https://github.com/Masterminds/glide/pull/798/files.
		// As of Go 1.8 the GOPATH is no longer required to be set. Instead there
		// is a default value. If there is no GOPATH check for the default value.
		// Note, checking the GOPATH first to avoid invoking the go toolchain if
		// possible.

		goExecutable := os.Getenv("COBRA_GO_EXECUTABLE")
		if len(goExecutable) <= 0 {
			goExecutable = "go"
		}

		out, err := exec.Command(goExecutable, "env", "GOPATH").Output()
		if err != nil {
			er(err)
		}

		toolchainGoPath := strings.TrimSpace(string(out))
		goPaths = filepath.SplitList(toolchainGoPath)
		if len(goPaths) == 0 {
			er("$GOPATH is not set")
		}
	}
	srcPaths = make([]string, 0, len(goPaths))
	for _, goPath := range goPaths {
		srcPaths = append(srcPaths, filepath.Join(goPath, "src"))
	}
}

func er(msg interface{}) {
	fmt.Println("Error:", msg)
	os.Exit(1)
}

// isEmpty checks if a given path is empty.
// Hidden files in path are ignored.
func isEmpty(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		er(err)
	}

	if !fi.IsDir() {
		return fi.Size() == 0
	}

	f, err := os.Open(path)
	if err != nil {
		er(err)
	}
	defer f.Close()

	names, err := f.Readdirnames(-1)
	if err != nil && err != io.EOF {
		er(err)
	}

	for _, name := range names {
		if len(name) > 0 && name[0] != '.' {
			return false
		}
	}
	return true
}

// exists checks if a file or directory exists.
func exists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if !os.IsNotExist(err) {
		er(err)
	}
	return false
}

func executeTemplate(tmplStr string, data interface{}) (string, error) {
	tmpl, err := template.New("").Funcs(template.FuncMap{"comment": commentifyString}).Parse(tmplStr)
	if err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)
	err = tmpl.Execute(buf, data)
	return buf.String(), err
}

func writeStringToFile(path string, s string) error {
	return writeToFile(path, strings.NewReader(s))
}

// writeToFile writes r to file with path only
// if file/directory on given path doesn't exist.
func writeToFile(path string, r io.Reader) error {
	if exists(path) {
		return fmt.Errorf("%v already exists", path)
	}

	dir := filepath.Dir(path)
	if dir != "" {
		if err := os.MkdirAll(dir, 0777); err != nil {
			return err
		}
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, r)
	return err
}

// commentfyString comments every line of in.
func commentifyString(in string) string {
	var newlines []string
	lines := strings.Split(in, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "//") {
			newlines = append(newlines, line)
		} else {
			if line == "" {
				newlines = append(newlines, "//")
			} else {
				newlines = append(newlines, "// "+line)
			}
		}
	}
	return strings.Join(newlines, "\n")
}
