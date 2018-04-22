package cmd

import (
	"testing"
)

func TestFindExistingPackage(t *testing.T) {
	path := findPackage("github.com/spf13/cobra")
	if path == "" {
		t.Fatal("findPackage didn't find the existing package")
	}
	if !hasGoPathPrefix(path) {
		t.Fatalf("%q is not in GOPATH, but must be", path)
	}
}

func hasGoPathPrefix(path string) bool {
	for _, srcPath := range srcPaths {
		if filepathHasPrefix(path, srcPath) {
			return true
		}
	}
	return false
}
