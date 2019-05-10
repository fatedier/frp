package service

import (
	"errors"
	"strconv"
	"strings"
)

// versionAtMost will return true if the provided version is less than or equal to max
func versionAtMost(version, max []int) (bool, error) {
	if comp, err := versionCompare(version, max); err != nil {
		return false, err
	} else if comp == 1 {
		return false, nil
	}
	return true, nil
}

// versionCompare take to versions split into integer arrays and attempts to compare them
// An error will be returned if there is an array length mismatch.
// Return values are as follows
// -1 - v1 is less than v2
// 0  - v1 is equal to v2
// 1  - v1 is greater than v2
func versionCompare(v1, v2 []int) (int, error) {
	if len(v1) != len(v2) {
		return 0, errors.New("version length mismatch")
	}

	for idx, v2S := range v2 {
		v1S := v1[idx]
		if v1S > v2S {
			return 1, nil
		}

		if v1S < v2S {
			return -1, nil
		}
	}
	return 0, nil
}

// parseVersion will parse any integer type version seperated by periods.
// This does not fully support semver style versions.
func parseVersion(v string) []int {
	version := make([]int, 3)

	for idx, vStr := range strings.Split(v, ".") {
		vS, err := strconv.Atoi(vStr)
		if err != nil {
			return nil
		}
		version[idx] = vS
	}

	return version
}
