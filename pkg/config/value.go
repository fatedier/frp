package config

import (
	"bytes"
	"io/ioutil"
	"os"
	"strings"
	"text/template"
)

var (
	glbEnvs map[string]string
)

func init() {
	glbEnvs = make(map[string]string)
	envs := os.Environ()
	for _, env := range envs {
		kv := strings.Split(env, "=")
		if len(kv) != 2 {
			continue
		}
		glbEnvs[kv[0]] = kv[1]
	}
}

type Values struct {
	Envs map[string]string // environment vars
}

func GetValues() *Values {
	return &Values{
		Envs: glbEnvs,
	}
}

func RenderContent(in string) (out string, err error) {
	tmpl, errRet := template.New("frp").Parse(in)
	if errRet != nil {
		err = errRet
		return
	}

	buffer := bytes.NewBufferString("")
	v := GetValues()
	err = tmpl.Execute(buffer, v)
	if err != nil {
		return
	}
	out = buffer.String()
	return
}

func GetRenderedConfFromFile(path string) (out string, err error) {
	var b []byte
	b, err = ioutil.ReadFile(path)
	if err != nil {
		return
	}
	content := string(b)

	out, err = RenderContent(content)
	return
}
