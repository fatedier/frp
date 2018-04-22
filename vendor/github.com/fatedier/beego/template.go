// Copyright 2014 beego Author. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package beego

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/astaxie/beego/logs"
	"github.com/astaxie/beego/utils"
)

var (
	beegoTplFuncMap = make(template.FuncMap)
	beeViewPathTemplateLocked = false
	// beeViewPathTemplates caching map and supported template file extensions per view
	beeViewPathTemplates  = make(map[string]map[string]*template.Template)
	templatesLock sync.RWMutex
	// beeTemplateExt stores the template extension which will build
	beeTemplateExt = []string{"tpl", "html"}
	// beeTemplatePreprocessors stores associations of extension -> preprocessor handler
	beeTemplateEngines = map[string]templatePreProcessor{}
)

// ExecuteTemplate applies the template with name  to the specified data object,
// writing the output to wr.
// A template will be executed safely in parallel.
func ExecuteTemplate(wr io.Writer, name string, data interface{}) error {
	return ExecuteViewPathTemplate(wr,name, BConfig.WebConfig.ViewsPath, data)
}

// ExecuteViewPathTemplate applies the template with name and from specific viewPath to the specified data object,
// writing the output to wr.
// A template will be executed safely in parallel.
func ExecuteViewPathTemplate(wr io.Writer, name string, viewPath string, data interface{}) error {
	if BConfig.RunMode == DEV {
		templatesLock.RLock()
		defer templatesLock.RUnlock()
	}
	if beeTemplates,ok := beeViewPathTemplates[viewPath]; ok {
		if t, ok := beeTemplates[name]; ok {
			var err error
			if t.Lookup(name) != nil {
				err = t.ExecuteTemplate(wr, name, data)
			} else {
				err = t.Execute(wr, data)
			}
			if err != nil {
				logs.Trace("template Execute err:", err)
			}
			return err
		}
		panic("can't find templatefile in the path:" + viewPath + "/" + name)
	}
	panic("Uknown view path:" + viewPath)
}

func init() {
	beegoTplFuncMap["dateformat"] = DateFormat
	beegoTplFuncMap["date"] = Date
	beegoTplFuncMap["compare"] = Compare
	beegoTplFuncMap["compare_not"] = CompareNot
	beegoTplFuncMap["not_nil"] = NotNil
	beegoTplFuncMap["not_null"] = NotNil
	beegoTplFuncMap["substr"] = Substr
	beegoTplFuncMap["html2str"] = HTML2str
	beegoTplFuncMap["str2html"] = Str2html
	beegoTplFuncMap["htmlquote"] = Htmlquote
	beegoTplFuncMap["htmlunquote"] = Htmlunquote
	beegoTplFuncMap["renderform"] = RenderForm
	beegoTplFuncMap["assets_js"] = AssetsJs
	beegoTplFuncMap["assets_css"] = AssetsCSS
	beegoTplFuncMap["config"] = GetConfig
	beegoTplFuncMap["map_get"] = MapGet

	// Comparisons
	beegoTplFuncMap["eq"] = eq // ==
	beegoTplFuncMap["ge"] = ge // >=
	beegoTplFuncMap["gt"] = gt // >
	beegoTplFuncMap["le"] = le // <=
	beegoTplFuncMap["lt"] = lt // <
	beegoTplFuncMap["ne"] = ne // !=

	beegoTplFuncMap["urlfor"] = URLFor // build a URL to match a Controller and it's method
}

// AddFuncMap let user to register a func in the template.
func AddFuncMap(key string, fn interface{}) error {
	beegoTplFuncMap[key] = fn
	return nil
}

type templatePreProcessor func(root, path string, funcs template.FuncMap) (*template.Template, error)

type templateFile struct {
	root  string
	files map[string][]string
}

// visit will make the paths into two part,the first is subDir (without tf.root),the second is full path(without tf.root).
// if tf.root="views" and
// paths is "views/errors/404.html",the subDir will be "errors",the file will be "errors/404.html"
// paths is "views/admin/errors/404.html",the subDir will be "admin/errors",the file will be "admin/errors/404.html"
func (tf *templateFile) visit(paths string, f os.FileInfo, err error) error {
	if f == nil {
		return err
	}
	if f.IsDir() || (f.Mode()&os.ModeSymlink) > 0 {
		return nil
	}
	if !HasTemplateExt(paths) {
		return nil
	}

	replace := strings.NewReplacer("\\", "/")
	file := strings.TrimLeft(replace.Replace(paths[len(tf.root):]), "/")
	subDir := filepath.Dir(file)

	tf.files[subDir] = append(tf.files[subDir], file)
	return nil
}

// HasTemplateExt return this path contains supported template extension of beego or not.
func HasTemplateExt(paths string) bool {
	for _, v := range beeTemplateExt {
		if strings.HasSuffix(paths, "."+v) {
			return true
		}
	}
	return false
}

// AddTemplateExt add new extension for template.
func AddTemplateExt(ext string) {
	for _, v := range beeTemplateExt {
		if v == ext {
			return
		}
	}
	beeTemplateExt = append(beeTemplateExt, ext)
}

// AddViewPath adds a new path to the supported view paths. 
//Can later be used by setting a controller ViewPath to this folder
//will panic if called after beego.Run() 
func AddViewPath(viewPath string) error {
	if beeViewPathTemplateLocked {
		panic("Can not add new view paths after beego.Run()")
	}
	beeViewPathTemplates[viewPath] = make(map[string]*template.Template)
	return BuildTemplate(viewPath)
}

func lockViewPaths() {
	beeViewPathTemplateLocked = true
}

// BuildTemplate will build all template files in a directory.
// it makes beego can render any template file in view directory.
func BuildTemplate(dir string, files ...string) error {
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return errors.New("dir open err")
	}
	beeTemplates,ok := beeViewPathTemplates[dir];
	if !ok {
		panic("Unknown view path: " + dir)
	}
	self := &templateFile{
		root:  dir,
		files: make(map[string][]string),
	}
	err := filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		return self.visit(path, f, err)
	})
	if err != nil {
		fmt.Printf("filepath.Walk() returned %v\n", err)
		return err
	}
	buildAllFiles := len(files) == 0
	for _, v := range self.files {
		for _, file := range v {
			if buildAllFiles || utils.InSlice(file, files) {
				templatesLock.Lock()
				ext := filepath.Ext(file)
				var t *template.Template
				if len(ext) == 0 {
					t, err = getTemplate(self.root, file, v...)
				} else if fn, ok := beeTemplateEngines[ext[1:]]; ok {
					t, err = fn(self.root, file, beegoTplFuncMap)
				} else {
					t, err = getTemplate(self.root, file, v...)
				}
				if err != nil {
					logs.Trace("parse template err:", file, err)
				} else {
					beeTemplates[file] = t
				}
				templatesLock.Unlock()
			}
		}
	}
	return nil
}

func getTplDeep(root, file, parent string, t *template.Template) (*template.Template, [][]string, error) {
	var fileAbsPath string
	if filepath.HasPrefix(file, "../") {
		fileAbsPath = filepath.Join(root, filepath.Dir(parent), file)
	} else {
		fileAbsPath = filepath.Join(root, file)
	}
	if e := utils.FileExists(fileAbsPath); !e {
		panic("can't find template file:" + file)
	}
	data, err := ioutil.ReadFile(fileAbsPath)
	if err != nil {
		return nil, [][]string{}, err
	}
	t, err = t.New(file).Parse(string(data))
	if err != nil {
		return nil, [][]string{}, err
	}
	reg := regexp.MustCompile(BConfig.WebConfig.TemplateLeft + "[ ]*template[ ]+\"([^\"]+)\"")
	allSub := reg.FindAllStringSubmatch(string(data), -1)
	for _, m := range allSub {
		if len(m) == 2 {
			tl := t.Lookup(m[1])
			if tl != nil {
				continue
			}
			if !HasTemplateExt(m[1]) {
				continue
			}
			_, _, err = getTplDeep(root, m[1], file, t)
			if err != nil {
				return nil, [][]string{}, err
			}
		}
	}
	return t, allSub, nil
}

func getTemplate(root, file string, others ...string) (t *template.Template, err error) {
	t = template.New(file).Delims(BConfig.WebConfig.TemplateLeft, BConfig.WebConfig.TemplateRight).Funcs(beegoTplFuncMap)
	var subMods [][]string
	t, subMods, err = getTplDeep(root, file, "", t)
	if err != nil {
		return nil, err
	}
	t, err = _getTemplate(t, root, subMods, others...)

	if err != nil {
		return nil, err
	}
	return
}

func _getTemplate(t0 *template.Template, root string, subMods [][]string, others ...string) (t *template.Template, err error) {
	t = t0
	for _, m := range subMods {
		if len(m) == 2 {
			tpl := t.Lookup(m[1])
			if tpl != nil {
				continue
			}
			//first check filename
			for _, otherFile := range others {
				if otherFile == m[1] {
					var subMods1 [][]string
					t, subMods1, err = getTplDeep(root, otherFile, "", t)
					if err != nil {
						logs.Trace("template parse file err:", err)
					} else if subMods1 != nil && len(subMods1) > 0 {
						t, err = _getTemplate(t, root, subMods1, others...)
					}
					break
				}
			}
			//second check define
			for _, otherFile := range others {
				fileAbsPath := filepath.Join(root, otherFile)
				data, err := ioutil.ReadFile(fileAbsPath)
				if err != nil {
					continue
				}
				reg := regexp.MustCompile(BConfig.WebConfig.TemplateLeft + "[ ]*define[ ]+\"([^\"]+)\"")
				allSub := reg.FindAllStringSubmatch(string(data), -1)
				for _, sub := range allSub {
					if len(sub) == 2 && sub[1] == m[1] {
						var subMods1 [][]string
						t, subMods1, err = getTplDeep(root, otherFile, "", t)
						if err != nil {
							logs.Trace("template parse file err:", err)
						} else if subMods1 != nil && len(subMods1) > 0 {
							t, err = _getTemplate(t, root, subMods1, others...)
						}
						break
					}
				}
			}
		}

	}
	return
}

// SetViewsPath sets view directory path in beego application.
func SetViewsPath(path string) *App {
	BConfig.WebConfig.ViewsPath = path
	return BeeApp
}

// SetStaticPath sets static directory path and proper url pattern in beego application.
// if beego.SetStaticPath("static","public"), visit /static/* to load static file in folder "public".
func SetStaticPath(url string, path string) *App {
	if !strings.HasPrefix(url, "/") {
		url = "/" + url
	}
	if url != "/" {
		url = strings.TrimRight(url, "/")
	}
	BConfig.WebConfig.StaticDir[url] = path
	return BeeApp
}

// DelStaticPath removes the static folder setting in this url pattern in beego application.
func DelStaticPath(url string) *App {
	if !strings.HasPrefix(url, "/") {
		url = "/" + url
	}
	if url != "/" {
		url = strings.TrimRight(url, "/")
	}
	delete(BConfig.WebConfig.StaticDir, url)
	return BeeApp
}

func AddTemplateEngine(extension string, fn templatePreProcessor) *App {
	AddTemplateExt(extension)
	beeTemplateEngines[extension] = fn
	return BeeApp
}
