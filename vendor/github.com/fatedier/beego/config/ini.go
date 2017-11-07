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

package config

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

var (
	defaultSection = "default"   // default section means if some ini items not in a section, make them in default section,
	bNumComment    = []byte{'#'} // number signal
	bSemComment    = []byte{';'} // semicolon signal
	bEmpty         = []byte{}
	bEqual         = []byte{'='} // equal signal
	bDQuote        = []byte{'"'} // quote signal
	sectionStart   = []byte{'['} // section start signal
	sectionEnd     = []byte{']'} // section end signal
	lineBreak      = "\n"
)

// IniConfig implements Config to parse ini file.
type IniConfig struct {
}

// Parse creates a new Config and parses the file configuration from the named file.
func (ini *IniConfig) Parse(name string) (Configer, error) {
	return ini.parseFile(name)
}

func (ini *IniConfig) parseFile(name string) (*IniConfigContainer, error) {
	data, err := ioutil.ReadFile(name)
	if err != nil {
		return nil, err
	}

	return ini.parseData(filepath.Dir(name), data)
}

func (ini *IniConfig) parseData(dir string, data []byte) (*IniConfigContainer, error) {
	cfg := &IniConfigContainer{
		data:           make(map[string]map[string]string),
		sectionComment: make(map[string]string),
		keyComment:     make(map[string]string),
		RWMutex:        sync.RWMutex{},
	}
	cfg.Lock()
	defer cfg.Unlock()

	var comment bytes.Buffer
	buf := bufio.NewReader(bytes.NewBuffer(data))
	// check the BOM
	head, err := buf.Peek(3)
	if err == nil && head[0] == 239 && head[1] == 187 && head[2] == 191 {
		for i := 1; i <= 3; i++ {
			buf.ReadByte()
		}
	}
	section := defaultSection
	for {
		line, _, err := buf.ReadLine()
		if err == io.EOF {
			break
		}
		//It might be a good idea to throw a error on all unknonw errors?
		if _, ok := err.(*os.PathError); ok {
			return nil, err
		}
		line = bytes.TrimSpace(line)
		if bytes.Equal(line, bEmpty) {
			continue
		}
		var bComment []byte
		switch {
		case bytes.HasPrefix(line, bNumComment):
			bComment = bNumComment
		case bytes.HasPrefix(line, bSemComment):
			bComment = bSemComment
		}
		if bComment != nil {
			line = bytes.TrimLeft(line, string(bComment))
			// Need append to a new line if multi-line comments.
			if comment.Len() > 0 {
				comment.WriteByte('\n')
			}
			comment.Write(line)
			continue
		}

		if bytes.HasPrefix(line, sectionStart) && bytes.HasSuffix(line, sectionEnd) {
			section = strings.ToLower(string(line[1 : len(line)-1])) // section name case insensitive
			if comment.Len() > 0 {
				cfg.sectionComment[section] = comment.String()
				comment.Reset()
			}
			if _, ok := cfg.data[section]; !ok {
				cfg.data[section] = make(map[string]string)
			}
			continue
		}

		if _, ok := cfg.data[section]; !ok {
			cfg.data[section] = make(map[string]string)
		}
		keyValue := bytes.SplitN(line, bEqual, 2)

		key := string(bytes.TrimSpace(keyValue[0])) // key name case insensitive
		key = strings.ToLower(key)

		// handle include "other.conf"
		if len(keyValue) == 1 && strings.HasPrefix(key, "include") {

			includefiles := strings.Fields(key)
			if includefiles[0] == "include" && len(includefiles) == 2 {

				otherfile := strings.Trim(includefiles[1], "\"")
				if !filepath.IsAbs(otherfile) {
					otherfile = filepath.Join(dir, otherfile)
				}

				i, err := ini.parseFile(otherfile)
				if err != nil {
					return nil, err
				}

				for sec, dt := range i.data {
					if _, ok := cfg.data[sec]; !ok {
						cfg.data[sec] = make(map[string]string)
					}
					for k, v := range dt {
						cfg.data[sec][k] = v
					}
				}

				for sec, comm := range i.sectionComment {
					cfg.sectionComment[sec] = comm
				}

				for k, comm := range i.keyComment {
					cfg.keyComment[k] = comm
				}

				continue
			}
		}

		if len(keyValue) != 2 {
			return nil, errors.New("read the content error: \"" + string(line) + "\", should key = val")
		}
		val := bytes.TrimSpace(keyValue[1])
		if bytes.HasPrefix(val, bDQuote) {
			val = bytes.Trim(val, `"`)
		}

		cfg.data[section][key] = ExpandValueEnv(string(val))
		if comment.Len() > 0 {
			cfg.keyComment[section+"."+key] = comment.String()
			comment.Reset()
		}

	}
	return cfg, nil
}

// ParseData parse ini the data
// When include other.conf,other.conf is either absolute directory
// or under beego in default temporary directory(/tmp/beego).
func (ini *IniConfig) ParseData(data []byte) (Configer, error) {
	dir := filepath.Join(os.TempDir(), "beego")
	os.MkdirAll(dir, os.ModePerm)

	return ini.parseData(dir, data)
}

// IniConfigContainer A Config represents the ini configuration.
// When set and get value, support key as section:name type.
type IniConfigContainer struct {
	data           map[string]map[string]string // section=> key:val
	sectionComment map[string]string            // section : comment
	keyComment     map[string]string            // id: []{comment, key...}; id 1 is for main comment.
	sync.RWMutex
}

// Bool returns the boolean value for a given key.
func (c *IniConfigContainer) Bool(key string) (bool, error) {
	return ParseBool(c.getdata(key))
}

// DefaultBool returns the boolean value for a given key.
// if err != nil return defaltval
func (c *IniConfigContainer) DefaultBool(key string, defaultval bool) bool {
	v, err := c.Bool(key)
	if err != nil {
		return defaultval
	}
	return v
}

// Int returns the integer value for a given key.
func (c *IniConfigContainer) Int(key string) (int, error) {
	return strconv.Atoi(c.getdata(key))
}

// DefaultInt returns the integer value for a given key.
// if err != nil return defaltval
func (c *IniConfigContainer) DefaultInt(key string, defaultval int) int {
	v, err := c.Int(key)
	if err != nil {
		return defaultval
	}
	return v
}

// Int64 returns the int64 value for a given key.
func (c *IniConfigContainer) Int64(key string) (int64, error) {
	return strconv.ParseInt(c.getdata(key), 10, 64)
}

// DefaultInt64 returns the int64 value for a given key.
// if err != nil return defaltval
func (c *IniConfigContainer) DefaultInt64(key string, defaultval int64) int64 {
	v, err := c.Int64(key)
	if err != nil {
		return defaultval
	}
	return v
}

// Float returns the float value for a given key.
func (c *IniConfigContainer) Float(key string) (float64, error) {
	return strconv.ParseFloat(c.getdata(key), 64)
}

// DefaultFloat returns the float64 value for a given key.
// if err != nil return defaltval
func (c *IniConfigContainer) DefaultFloat(key string, defaultval float64) float64 {
	v, err := c.Float(key)
	if err != nil {
		return defaultval
	}
	return v
}

// String returns the string value for a given key.
func (c *IniConfigContainer) String(key string) string {
	return c.getdata(key)
}

// DefaultString returns the string value for a given key.
// if err != nil return defaltval
func (c *IniConfigContainer) DefaultString(key string, defaultval string) string {
	v := c.String(key)
	if v == "" {
		return defaultval
	}
	return v
}

// Strings returns the []string value for a given key.
// Return nil if config value does not exist or is empty.
func (c *IniConfigContainer) Strings(key string) []string {
	v := c.String(key)
	if v == "" {
		return nil
	}
	return strings.Split(v, ";")
}

// DefaultStrings returns the []string value for a given key.
// if err != nil return defaltval
func (c *IniConfigContainer) DefaultStrings(key string, defaultval []string) []string {
	v := c.Strings(key)
	if v == nil {
		return defaultval
	}
	return v
}

// GetSection returns map for the given section
func (c *IniConfigContainer) GetSection(section string) (map[string]string, error) {
	if v, ok := c.data[section]; ok {
		return v, nil
	}
	return nil, errors.New("not exist section")
}

// SaveConfigFile save the config into file.
//
// BUG(env): The environment variable config item will be saved with real value in SaveConfigFile Funcation.
func (c *IniConfigContainer) SaveConfigFile(filename string) (err error) {
	// Write configuration file by filename.
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	// Get section or key comments. Fixed #1607
	getCommentStr := func(section, key string) string {
		comment, ok := "", false
		if len(key) == 0 {
			comment, ok = c.sectionComment[section]
		} else {
			comment, ok = c.keyComment[section+"."+key]
		}

		if ok {
			// Empty comment
			if len(comment) == 0 || len(strings.TrimSpace(comment)) == 0 {
				return string(bNumComment)
			}
			prefix := string(bNumComment)
			// Add the line head character "#"
			return prefix + strings.Replace(comment, lineBreak, lineBreak+prefix, -1)
		}
		return ""
	}

	buf := bytes.NewBuffer(nil)
	// Save default section at first place
	if dt, ok := c.data[defaultSection]; ok {
		for key, val := range dt {
			if key != " " {
				// Write key comments.
				if v := getCommentStr(defaultSection, key); len(v) > 0 {
					if _, err = buf.WriteString(v + lineBreak); err != nil {
						return err
					}
				}

				// Write key and value.
				if _, err = buf.WriteString(key + string(bEqual) + val + lineBreak); err != nil {
					return err
				}
			}
		}

		// Put a line between sections.
		if _, err = buf.WriteString(lineBreak); err != nil {
			return err
		}
	}
	// Save named sections
	for section, dt := range c.data {
		if section != defaultSection {
			// Write section comments.
			if v := getCommentStr(section, ""); len(v) > 0 {
				if _, err = buf.WriteString(v + lineBreak); err != nil {
					return err
				}
			}

			// Write section name.
			if _, err = buf.WriteString(string(sectionStart) + section + string(sectionEnd) + lineBreak); err != nil {
				return err
			}

			for key, val := range dt {
				if key != " " {
					// Write key comments.
					if v := getCommentStr(section, key); len(v) > 0 {
						if _, err = buf.WriteString(v + lineBreak); err != nil {
							return err
						}
					}

					// Write key and value.
					if _, err = buf.WriteString(key + string(bEqual) + val + lineBreak); err != nil {
						return err
					}
				}
			}

			// Put a line between sections.
			if _, err = buf.WriteString(lineBreak); err != nil {
				return err
			}
		}
	}

	if _, err = buf.WriteTo(f); err != nil {
		return err
	}
	return nil
}

// Set writes a new value for key.
// if write to one section, the key need be "section::key".
// if the section is not existed, it panics.
func (c *IniConfigContainer) Set(key, value string) error {
	c.Lock()
	defer c.Unlock()
	if len(key) == 0 {
		return errors.New("key is empty")
	}

	var (
		section, k string
		sectionKey = strings.Split(key, "::")
	)

	if len(sectionKey) >= 2 {
		section = sectionKey[0]
		k = sectionKey[1]
	} else {
		section = defaultSection
		k = sectionKey[0]
	}

	if _, ok := c.data[section]; !ok {
		c.data[section] = make(map[string]string)
	}
	c.data[section][k] = value
	return nil
}

// DIY returns the raw value by a given key.
func (c *IniConfigContainer) DIY(key string) (v interface{}, err error) {
	if v, ok := c.data[strings.ToLower(key)]; ok {
		return v, nil
	}
	return v, errors.New("key not find")
}

// section.key or key
func (c *IniConfigContainer) getdata(key string) string {
	if len(key) == 0 {
		return ""
	}
	c.RLock()
	defer c.RUnlock()

	var (
		section, k string
		sectionKey = strings.Split(strings.ToLower(key), "::")
	)
	if len(sectionKey) >= 2 {
		section = sectionKey[0]
		k = sectionKey[1]
	} else {
		section = defaultSection
		k = sectionKey[0]
	}
	if v, ok := c.data[section]; ok {
		if vv, ok := v[k]; ok {
			return vv
		}
	}
	return ""
}

func init() {
	Register("ini", &IniConfig{})
}
