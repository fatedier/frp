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

package context

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"html/template"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// BeegoOutput does work for sending response header.
type BeegoOutput struct {
	Context    *Context
	Status     int
	EnableGzip bool
}

// NewOutput returns new BeegoOutput.
// it contains nothing now.
func NewOutput() *BeegoOutput {
	return &BeegoOutput{}
}

// Reset init BeegoOutput
func (output *BeegoOutput) Reset(ctx *Context) {
	output.Context = ctx
	output.Status = 0
}

// Header sets response header item string via given key.
func (output *BeegoOutput) Header(key, val string) {
	output.Context.ResponseWriter.Header().Set(key, val)
}

// Body sets response body content.
// if EnableGzip, compress content string.
// it sends out response body directly.
func (output *BeegoOutput) Body(content []byte) error {
	var encoding string
	var buf = &bytes.Buffer{}
	if output.EnableGzip {
		encoding = ParseEncoding(output.Context.Request)
	}
	if b, n, _ := WriteBody(encoding, buf, content); b {
		output.Header("Content-Encoding", n)
		output.Header("Content-Length", strconv.Itoa(buf.Len()))
	} else {
		output.Header("Content-Length", strconv.Itoa(len(content)))
	}
	// Write status code if it has been set manually
	// Set it to 0 afterwards to prevent "multiple response.WriteHeader calls"
	if output.Status != 0 {
		output.Context.ResponseWriter.WriteHeader(output.Status)
		output.Status = 0
	} else {
		output.Context.ResponseWriter.Started = true
	}
	io.Copy(output.Context.ResponseWriter, buf)
	return nil
}

// Cookie sets cookie value via given key.
// others are ordered as cookie's max age time, path,domain, secure and httponly.
func (output *BeegoOutput) Cookie(name string, value string, others ...interface{}) {
	var b bytes.Buffer
	fmt.Fprintf(&b, "%s=%s", sanitizeName(name), sanitizeValue(value))

	//fix cookie not work in IE
	if len(others) > 0 {
		var maxAge int64

		switch v := others[0].(type) {
		case int:
			maxAge = int64(v)
		case int32:
			maxAge = int64(v)
		case int64:
			maxAge = v
		}

		switch {
		case maxAge > 0:
			fmt.Fprintf(&b, "; Expires=%s; Max-Age=%d", time.Now().Add(time.Duration(maxAge)*time.Second).UTC().Format(time.RFC1123), maxAge)
		case maxAge < 0:
			fmt.Fprintf(&b, "; Max-Age=0")
		}
	}

	// the settings below
	// Path, Domain, Secure, HttpOnly
	// can use nil skip set

	// default "/"
	if len(others) > 1 {
		if v, ok := others[1].(string); ok && len(v) > 0 {
			fmt.Fprintf(&b, "; Path=%s", sanitizeValue(v))
		}
	} else {
		fmt.Fprintf(&b, "; Path=%s", "/")
	}

	// default empty
	if len(others) > 2 {
		if v, ok := others[2].(string); ok && len(v) > 0 {
			fmt.Fprintf(&b, "; Domain=%s", sanitizeValue(v))
		}
	}

	// default empty
	if len(others) > 3 {
		var secure bool
		switch v := others[3].(type) {
		case bool:
			secure = v
		default:
			if others[3] != nil {
				secure = true
			}
		}
		if secure {
			fmt.Fprintf(&b, "; Secure")
		}
	}

	// default false. for session cookie default true
	if len(others) > 4 {
		if v, ok := others[4].(bool); ok && v {
			fmt.Fprintf(&b, "; HttpOnly")
		}
	}

	output.Context.ResponseWriter.Header().Add("Set-Cookie", b.String())
}

var cookieNameSanitizer = strings.NewReplacer("\n", "-", "\r", "-")

func sanitizeName(n string) string {
	return cookieNameSanitizer.Replace(n)
}

var cookieValueSanitizer = strings.NewReplacer("\n", " ", "\r", " ", ";", " ")

func sanitizeValue(v string) string {
	return cookieValueSanitizer.Replace(v)
}

// JSON writes json to response body.
// if coding is true, it converts utf-8 to \u0000 type.
func (output *BeegoOutput) JSON(data interface{}, hasIndent bool, coding bool) error {
	output.Header("Content-Type", "application/json; charset=utf-8")
	var content []byte
	var err error
	if hasIndent {
		content, err = json.MarshalIndent(data, "", "  ")
	} else {
		content, err = json.Marshal(data)
	}
	if err != nil {
		http.Error(output.Context.ResponseWriter, err.Error(), http.StatusInternalServerError)
		return err
	}
	if coding {
		content = []byte(stringsToJSON(string(content)))
	}
	return output.Body(content)
}

// JSONP writes jsonp to response body.
func (output *BeegoOutput) JSONP(data interface{}, hasIndent bool) error {
	output.Header("Content-Type", "application/javascript; charset=utf-8")
	var content []byte
	var err error
	if hasIndent {
		content, err = json.MarshalIndent(data, "", "  ")
	} else {
		content, err = json.Marshal(data)
	}
	if err != nil {
		http.Error(output.Context.ResponseWriter, err.Error(), http.StatusInternalServerError)
		return err
	}
	callback := output.Context.Input.Query("callback")
	if callback == "" {
		return errors.New(`"callback" parameter required`)
	}
	callback = template.JSEscapeString(callback)
	callbackContent := bytes.NewBufferString(" if(window." + callback + ")" + callback)
	callbackContent.WriteString("(")
	callbackContent.Write(content)
	callbackContent.WriteString(");\r\n")
	return output.Body(callbackContent.Bytes())
}

// XML writes xml string to response body.
func (output *BeegoOutput) XML(data interface{}, hasIndent bool) error {
	output.Header("Content-Type", "application/xml; charset=utf-8")
	var content []byte
	var err error
	if hasIndent {
		content, err = xml.MarshalIndent(data, "", "  ")
	} else {
		content, err = xml.Marshal(data)
	}
	if err != nil {
		http.Error(output.Context.ResponseWriter, err.Error(), http.StatusInternalServerError)
		return err
	}
	return output.Body(content)
}

// Download forces response for download file.
// it prepares the download response header automatically.
func (output *BeegoOutput) Download(file string, filename ...string) {
	// check get file error, file not found or other error.
	if _, err := os.Stat(file); err != nil {
		http.ServeFile(output.Context.ResponseWriter, output.Context.Request, file)
		return
	}

	var fName string
	if len(filename) > 0 && filename[0] != "" {
		fName = filename[0]
	} else {
		fName = filepath.Base(file)
	}
	output.Header("Content-Disposition", "attachment; filename="+url.QueryEscape(fName))
	output.Header("Content-Description", "File Transfer")
	output.Header("Content-Type", "application/octet-stream")
	output.Header("Content-Transfer-Encoding", "binary")
	output.Header("Expires", "0")
	output.Header("Cache-Control", "must-revalidate")
	output.Header("Pragma", "public")
	http.ServeFile(output.Context.ResponseWriter, output.Context.Request, file)
}

// ContentType sets the content type from ext string.
// MIME type is given in mime package.
func (output *BeegoOutput) ContentType(ext string) {
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	ctype := mime.TypeByExtension(ext)
	if ctype != "" {
		output.Header("Content-Type", ctype)
	}
}

// SetStatus sets response status code.
// It writes response header directly.
func (output *BeegoOutput) SetStatus(status int) {
	output.Status = status
}

// IsCachable returns boolean of this request is cached.
// HTTP 304 means cached.
func (output *BeegoOutput) IsCachable() bool {
	return output.Status >= 200 && output.Status < 300 || output.Status == 304
}

// IsEmpty returns boolean of this request is empty.
// HTTP 201，204 and 304 means empty.
func (output *BeegoOutput) IsEmpty() bool {
	return output.Status == 201 || output.Status == 204 || output.Status == 304
}

// IsOk returns boolean of this request runs well.
// HTTP 200 means ok.
func (output *BeegoOutput) IsOk() bool {
	return output.Status == 200
}

// IsSuccessful returns boolean of this request runs successfully.
// HTTP 2xx means ok.
func (output *BeegoOutput) IsSuccessful() bool {
	return output.Status >= 200 && output.Status < 300
}

// IsRedirect returns boolean of this request is redirection header.
// HTTP 301,302,307 means redirection.
func (output *BeegoOutput) IsRedirect() bool {
	return output.Status == 301 || output.Status == 302 || output.Status == 303 || output.Status == 307
}

// IsForbidden returns boolean of this request is forbidden.
// HTTP 403 means forbidden.
func (output *BeegoOutput) IsForbidden() bool {
	return output.Status == 403
}

// IsNotFound returns boolean of this request is not found.
// HTTP 404 means forbidden.
func (output *BeegoOutput) IsNotFound() bool {
	return output.Status == 404
}

// IsClientError returns boolean of this request client sends error data.
// HTTP 4xx means forbidden.
func (output *BeegoOutput) IsClientError() bool {
	return output.Status >= 400 && output.Status < 500
}

// IsServerError returns boolean of this server handler errors.
// HTTP 5xx means server internal error.
func (output *BeegoOutput) IsServerError() bool {
	return output.Status >= 500 && output.Status < 600
}

func stringsToJSON(str string) string {
	rs := []rune(str)
	var jsons bytes.Buffer
	for _, r := range rs {
		rint := int(r)
		if rint < 128 {
			jsons.WriteRune(r)
		} else {
			jsons.WriteString("\\u")
			jsons.WriteString(strconv.FormatInt(int64(rint), 16))
		}
	}
	return jsons.String()
}

// Session sets session item value with given key.
func (output *BeegoOutput) Session(name interface{}, value interface{}) {
	output.Context.Input.CruSession.Set(name, value)
}
