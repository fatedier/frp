package util

import (
	"io/ioutil"
)

func GenerateConfigFile(path string, content string) error {
	return ioutil.WriteFile(path, []byte(content), 0666)
}
