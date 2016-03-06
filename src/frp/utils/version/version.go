package version

import (
	"strconv"
	"strings"
)

var version string = "0.2.0"

func Full() string {
	return version
}

func Proto(v string) int64 {
	arr := strings.Split(v, ".")
	if len(arr) < 2 {
		return 0
	}
	res, _ := strconv.ParseInt(arr[0], 10, 64)
	return res
}

func Major(v string) int64 {
	arr := strings.Split(v, ".")
	if len(arr) < 2 {
		return 0
	}
	res, _ := strconv.ParseInt(arr[1], 10, 64)
	return res
}

func Minor(v string) int64 {
	arr := strings.Split(v, ".")
	if len(arr) < 2 {
		return 0
	}
	res, _ := strconv.ParseInt(arr[2], 10, 64)
	return res
}

// add every case there if server will not accept client's protocol and return false
func Compat(client string, server string) bool {
	return true
}
