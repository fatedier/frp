// Copyright 2025 The frp Authors
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

package v1

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
)

// ValueSource provides a way to dynamically resolve configuration values
// from various sources like files, environment variables, or external services.
type ValueSource struct {
	Type string      `json:"type"`
	File *FileSource `json:"file,omitempty"`
}

// FileSource specifies how to load a value from a file.
type FileSource struct {
	Path string `json:"path"`
}

// Validate validates the ValueSource configuration.
func (v *ValueSource) Validate() error {
	if v == nil {
		return errors.New("valueSource cannot be nil")
	}

	switch v.Type {
	case "file":
		if v.File == nil {
			return errors.New("file configuration is required when type is 'file'")
		}
		return v.File.Validate()
	default:
		return fmt.Errorf("unsupported value source type: %s (only 'file' is supported)", v.Type)
	}
}

// Resolve resolves the value from the configured source.
func (v *ValueSource) Resolve(ctx context.Context) (string, error) {
	if err := v.Validate(); err != nil {
		return "", err
	}

	switch v.Type {
	case "file":
		return v.File.Resolve(ctx)
	default:
		return "", fmt.Errorf("unsupported value source type: %s", v.Type)
	}
}

// Validate validates the FileSource configuration.
func (f *FileSource) Validate() error {
	if f == nil {
		return errors.New("fileSource cannot be nil")
	}

	if f.Path == "" {
		return errors.New("file path cannot be empty")
	}
	return nil
}

// Resolve reads and returns the content from the specified file.
func (f *FileSource) Resolve(_ context.Context) (string, error) {
	if err := f.Validate(); err != nil {
		return "", err
	}

	content, err := os.ReadFile(f.Path)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %v", f.Path, err)
	}

	// Trim whitespace, which is important for file-based tokens
	return strings.TrimSpace(string(content)), nil
}
