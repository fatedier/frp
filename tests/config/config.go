package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

func GenerateConfigFile(path string, content string) (realPath string, err error) {
	realPath = filepath.Join(os.TempDir(), path)
	err = ioutil.WriteFile(realPath, []byte(content), 0666)
	return realPath, err
}
