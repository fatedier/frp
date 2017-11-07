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

package cache

import (
	"bytes"
	"crypto/md5"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"time"
)

// FileCacheItem is basic unit of file cache adapter.
// it contains data and expire time.
type FileCacheItem struct {
	Data       interface{}
	Lastaccess time.Time
	Expired    time.Time
}

// FileCache Config
var (
	FileCachePath           = "cache"     // cache directory
	FileCacheFileSuffix     = ".bin"      // cache file suffix
	FileCacheDirectoryLevel = 2           // cache file deep level if auto generated cache files.
	FileCacheEmbedExpiry    time.Duration // cache expire time, default is no expire forever.
)

// FileCache is cache adapter for file storage.
type FileCache struct {
	CachePath      string
	FileSuffix     string
	DirectoryLevel int
	EmbedExpiry    int
}

// NewFileCache Create new file cache with no config.
// the level and expiry need set in method StartAndGC as config string.
func NewFileCache() Cache {
	//    return &FileCache{CachePath:FileCachePath, FileSuffix:FileCacheFileSuffix}
	return &FileCache{}
}

// StartAndGC will start and begin gc for file cache.
// the config need to be like {CachePath:"/cache","FileSuffix":".bin","DirectoryLevel":2,"EmbedExpiry":0}
func (fc *FileCache) StartAndGC(config string) error {

	var cfg map[string]string
	json.Unmarshal([]byte(config), &cfg)
	if _, ok := cfg["CachePath"]; !ok {
		cfg["CachePath"] = FileCachePath
	}
	if _, ok := cfg["FileSuffix"]; !ok {
		cfg["FileSuffix"] = FileCacheFileSuffix
	}
	if _, ok := cfg["DirectoryLevel"]; !ok {
		cfg["DirectoryLevel"] = strconv.Itoa(FileCacheDirectoryLevel)
	}
	if _, ok := cfg["EmbedExpiry"]; !ok {
		cfg["EmbedExpiry"] = strconv.FormatInt(int64(FileCacheEmbedExpiry.Seconds()), 10)
	}
	fc.CachePath = cfg["CachePath"]
	fc.FileSuffix = cfg["FileSuffix"]
	fc.DirectoryLevel, _ = strconv.Atoi(cfg["DirectoryLevel"])
	fc.EmbedExpiry, _ = strconv.Atoi(cfg["EmbedExpiry"])

	fc.Init()
	return nil
}

// Init will make new dir for file cache if not exist.
func (fc *FileCache) Init() {
	if ok, _ := exists(fc.CachePath); !ok { // todo : error handle
		_ = os.MkdirAll(fc.CachePath, os.ModePerm) // todo : error handle
	}
}

// get cached file name. it's md5 encoded.
func (fc *FileCache) getCacheFileName(key string) string {
	m := md5.New()
	io.WriteString(m, key)
	keyMd5 := hex.EncodeToString(m.Sum(nil))
	cachePath := fc.CachePath
	switch fc.DirectoryLevel {
	case 2:
		cachePath = filepath.Join(cachePath, keyMd5[0:2], keyMd5[2:4])
	case 1:
		cachePath = filepath.Join(cachePath, keyMd5[0:2])
	}

	if ok, _ := exists(cachePath); !ok { // todo : error handle
		_ = os.MkdirAll(cachePath, os.ModePerm) // todo : error handle
	}

	return filepath.Join(cachePath, fmt.Sprintf("%s%s", keyMd5, fc.FileSuffix))
}

// Get value from file cache.
// if non-exist or expired, return empty string.
func (fc *FileCache) Get(key string) interface{} {
	fileData, err := FileGetContents(fc.getCacheFileName(key))
	if err != nil {
		return ""
	}
	var to FileCacheItem
	GobDecode(fileData, &to)
	if to.Expired.Before(time.Now()) {
		return ""
	}
	return to.Data
}

// GetMulti gets values from file cache.
// if non-exist or expired, return empty string.
func (fc *FileCache) GetMulti(keys []string) []interface{} {
	var rc []interface{}
	for _, key := range keys {
		rc = append(rc, fc.Get(key))
	}
	return rc
}

// Put value into file cache.
// timeout means how long to keep this file, unit of ms.
// if timeout equals FileCacheEmbedExpiry(default is 0), cache this item forever.
func (fc *FileCache) Put(key string, val interface{}, timeout time.Duration) error {
	gob.Register(val)

	item := FileCacheItem{Data: val}
	if timeout == FileCacheEmbedExpiry {
		item.Expired = time.Now().Add((86400 * 365 * 10) * time.Second) // ten years
	} else {
		item.Expired = time.Now().Add(timeout)
	}
	item.Lastaccess = time.Now()
	data, err := GobEncode(item)
	if err != nil {
		return err
	}
	return FilePutContents(fc.getCacheFileName(key), data)
}

// Delete file cache value.
func (fc *FileCache) Delete(key string) error {
	filename := fc.getCacheFileName(key)
	if ok, _ := exists(filename); ok {
		return os.Remove(filename)
	}
	return nil
}

// Incr will increase cached int value.
// fc value is saving forever unless Delete.
func (fc *FileCache) Incr(key string) error {
	data := fc.Get(key)
	var incr int
	if reflect.TypeOf(data).Name() != "int" {
		incr = 0
	} else {
		incr = data.(int) + 1
	}
	fc.Put(key, incr, FileCacheEmbedExpiry)
	return nil
}

// Decr will decrease cached int value.
func (fc *FileCache) Decr(key string) error {
	data := fc.Get(key)
	var decr int
	if reflect.TypeOf(data).Name() != "int" || data.(int)-1 <= 0 {
		decr = 0
	} else {
		decr = data.(int) - 1
	}
	fc.Put(key, decr, FileCacheEmbedExpiry)
	return nil
}

// IsExist check value is exist.
func (fc *FileCache) IsExist(key string) bool {
	ret, _ := exists(fc.getCacheFileName(key))
	return ret
}

// ClearAll will clean cached files.
// not implemented.
func (fc *FileCache) ClearAll() error {
	return nil
}

// check file exist.
func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// FileGetContents Get bytes to file.
// if non-exist, create this file.
func FileGetContents(filename string) (data []byte, e error) {
	return ioutil.ReadFile(filename)
}

// FilePutContents Put bytes to file.
// if non-exist, create this file.
func FilePutContents(filename string, content []byte) error {
	return ioutil.WriteFile(filename, content, os.ModePerm)
}

// GobEncode Gob encodes file cache item.
func GobEncode(data interface{}) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(data)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), err
}

// GobDecode Gob decodes file cache item.
func GobDecode(data []byte, to *FileCacheItem) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	return dec.Decode(&to)
}

func init() {
	Register("file", NewFileCache)
}
