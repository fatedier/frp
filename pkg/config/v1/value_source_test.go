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
	"os"
	"path/filepath"
	"testing"
)

func TestValueSource_Validate(t *testing.T) {
	tests := []struct {
		name    string
		vs      *ValueSource
		wantErr bool
	}{
		{
			name:    "nil valueSource",
			vs:      nil,
			wantErr: true,
		},
		{
			name: "unsupported type",
			vs: &ValueSource{
				Type: "unsupported",
			},
			wantErr: true,
		},
		{
			name: "file type without file config",
			vs: &ValueSource{
				Type: "file",
				File: nil,
			},
			wantErr: true,
		},
		{
			name: "valid file type with absolute path",
			vs: &ValueSource{
				Type: "file",
				File: &FileSource{
					Path: "/tmp/test",
				},
			},
			wantErr: false,
		},
		{
			name: "valid file type with relative path",
			vs: &ValueSource{
				Type: "file",
				File: &FileSource{
					Path: "configs/token",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.vs.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValueSource.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFileSource_Validate(t *testing.T) {
	tests := []struct {
		name    string
		fs      *FileSource
		wantErr bool
	}{
		{
			name:    "nil fileSource",
			fs:      nil,
			wantErr: true,
		},
		{
			name: "empty path",
			fs: &FileSource{
				Path: "",
			},
			wantErr: true,
		},
		{
			name: "relative path (allowed)",
			fs: &FileSource{
				Path: "relative/path",
			},
			wantErr: false,
		},
		{
			name: "absolute path",
			fs: &FileSource{
				Path: "/absolute/path",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fs.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("FileSource.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFileSource_Resolve(t *testing.T) {
	// Create a temporary file for testing
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test_token")
	testContent := "test-token-value\n\t "
	expectedContent := "test-token-value"

	err := os.WriteFile(testFile, []byte(testContent), 0o600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	tests := []struct {
		name    string
		fs      *FileSource
		want    string
		wantErr bool
	}{
		{
			name: "valid file path",
			fs: &FileSource{
				Path: testFile,
			},
			want:    expectedContent,
			wantErr: false,
		},
		{
			name: "non-existent file",
			fs: &FileSource{
				Path: "/non/existent/file",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "path traversal attempt (should fail validation)",
			fs: &FileSource{
				Path: "../../../etc/passwd",
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.fs.Resolve(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("FileSource.Resolve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("FileSource.Resolve() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValueSource_Resolve(t *testing.T) {
	// Create a temporary file for testing
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test_token")
	testContent := "test-token-value"

	err := os.WriteFile(testFile, []byte(testContent), 0o600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	tests := []struct {
		name    string
		vs      *ValueSource
		want    string
		wantErr bool
	}{
		{
			name: "valid file type",
			vs: &ValueSource{
				Type: "file",
				File: &FileSource{
					Path: testFile,
				},
			},
			want:    testContent,
			wantErr: false,
		},
		{
			name: "unsupported type",
			vs: &ValueSource{
				Type: "unsupported",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "file type with path traversal",
			vs: &ValueSource{
				Type: "file",
				File: &FileSource{
					Path: "../../../etc/passwd",
				},
			},
			want:    "",
			wantErr: true,
		},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.vs.Resolve(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValueSource.Resolve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ValueSource.Resolve() = %v, want %v", got, tt.want)
			}
		})
	}
}
