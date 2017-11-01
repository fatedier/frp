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

package validation

import (
	"fmt"
	"reflect"
	"regexp"
	"time"
	"unicode/utf8"
)

// MessageTmpls store commond validate template
var MessageTmpls = map[string]string{
	"Required":     "Can not be empty",
	"Min":          "Minimum is %d",
	"Max":          "Maximum is %d",
	"Range":        "Range is %d to %d",
	"MinSize":      "Minimum size is %d",
	"MaxSize":      "Maximum size is %d",
	"Length":       "Required length is %d",
	"Alpha":        "Must be valid alpha characters",
	"Numeric":      "Must be valid numeric characters",
	"AlphaNumeric": "Must be valid alpha or numeric characters",
	"Match":        "Must match %s",
	"NoMatch":      "Must not match %s",
	"AlphaDash":    "Must be valid alpha or numeric or dash(-_) characters",
	"Email":        "Must be a valid email address",
	"IP":           "Must be a valid ip address",
	"Base64":       "Must be valid base64 characters",
	"Mobile":       "Must be valid mobile number",
	"Tel":          "Must be valid telephone number",
	"Phone":        "Must be valid telephone or mobile phone number",
	"ZipCode":      "Must be valid zipcode",
}

// SetDefaultMessage set default messages
// if not set, the default messages are
//  "Required":     "Can not be empty",
//  "Min":          "Minimum is %d",
//  "Max":          "Maximum is %d",
//  "Range":        "Range is %d to %d",
//  "MinSize":      "Minimum size is %d",
//  "MaxSize":      "Maximum size is %d",
//  "Length":       "Required length is %d",
//  "Alpha":        "Must be valid alpha characters",
//  "Numeric":      "Must be valid numeric characters",
//  "AlphaNumeric": "Must be valid alpha or numeric characters",
//  "Match":        "Must match %s",
//  "NoMatch":      "Must not match %s",
//  "AlphaDash":    "Must be valid alpha or numeric or dash(-_) characters",
//  "Email":        "Must be a valid email address",
//  "IP":           "Must be a valid ip address",
//  "Base64":       "Must be valid base64 characters",
//  "Mobile":       "Must be valid mobile number",
//  "Tel":          "Must be valid telephone number",
//  "Phone":        "Must be valid telephone or mobile phone number",
//  "ZipCode":      "Must be valid zipcode",
func SetDefaultMessage(msg map[string]string) {
	if len(msg) == 0 {
		return
	}

	for name := range msg {
		MessageTmpls[name] = msg[name]
	}
}

// Validator interface
type Validator interface {
	IsSatisfied(interface{}) bool
	DefaultMessage() string
	GetKey() string
	GetLimitValue() interface{}
}

// Required struct
type Required struct {
	Key string
}

// IsSatisfied judge whether obj has value
func (r Required) IsSatisfied(obj interface{}) bool {
	if obj == nil {
		return false
	}

	if str, ok := obj.(string); ok {
		return len(str) > 0
	}
	if _, ok := obj.(bool); ok {
		return true
	}
	if i, ok := obj.(int); ok {
		return i != 0
	}
	if i, ok := obj.(uint); ok {
		return i != 0
	}
	if i, ok := obj.(int8); ok {
		return i != 0
	}
	if i, ok := obj.(uint8); ok {
		return i != 0
	}
	if i, ok := obj.(int16); ok {
		return i != 0
	}
	if i, ok := obj.(uint16); ok {
		return i != 0
	}
	if i, ok := obj.(uint32); ok {
		return i != 0
	}
	if i, ok := obj.(int32); ok {
		return i != 0
	}
	if i, ok := obj.(int64); ok {
		return i != 0
	}
	if i, ok := obj.(uint64); ok {
		return i != 0
	}
	if t, ok := obj.(time.Time); ok {
		return !t.IsZero()
	}
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Slice {
		return v.Len() > 0
	}
	return true
}

// DefaultMessage return the default error message
func (r Required) DefaultMessage() string {
	return fmt.Sprint(MessageTmpls["Required"])
}

// GetKey return the r.Key
func (r Required) GetKey() string {
	return r.Key
}

// GetLimitValue return nil now
func (r Required) GetLimitValue() interface{} {
	return nil
}

// Min check struct
type Min struct {
	Min int
	Key string
}

// IsSatisfied judge whether obj is valid
func (m Min) IsSatisfied(obj interface{}) bool {
	num, ok := obj.(int)
	if ok {
		return num >= m.Min
	}
	return false
}

// DefaultMessage return the default min error message
func (m Min) DefaultMessage() string {
	return fmt.Sprintf(MessageTmpls["Min"], m.Min)
}

// GetKey return the m.Key
func (m Min) GetKey() string {
	return m.Key
}

// GetLimitValue return the limit value, Min
func (m Min) GetLimitValue() interface{} {
	return m.Min
}

// Max validate struct
type Max struct {
	Max int
	Key string
}

// IsSatisfied judge whether obj is valid
func (m Max) IsSatisfied(obj interface{}) bool {
	num, ok := obj.(int)
	if ok {
		return num <= m.Max
	}
	return false
}

// DefaultMessage return the default max error message
func (m Max) DefaultMessage() string {
	return fmt.Sprintf(MessageTmpls["Max"], m.Max)
}

// GetKey return the m.Key
func (m Max) GetKey() string {
	return m.Key
}

// GetLimitValue return the limit value, Max
func (m Max) GetLimitValue() interface{} {
	return m.Max
}

// Range Requires an integer to be within Min, Max inclusive.
type Range struct {
	Min
	Max
	Key string
}

// IsSatisfied judge whether obj is valid
func (r Range) IsSatisfied(obj interface{}) bool {
	return r.Min.IsSatisfied(obj) && r.Max.IsSatisfied(obj)
}

// DefaultMessage return the default Range error message
func (r Range) DefaultMessage() string {
	return fmt.Sprintf(MessageTmpls["Range"], r.Min.Min, r.Max.Max)
}

// GetKey return the m.Key
func (r Range) GetKey() string {
	return r.Key
}

// GetLimitValue return the limit value, Max
func (r Range) GetLimitValue() interface{} {
	return []int{r.Min.Min, r.Max.Max}
}

// MinSize Requires an array or string to be at least a given length.
type MinSize struct {
	Min int
	Key string
}

// IsSatisfied judge whether obj is valid
func (m MinSize) IsSatisfied(obj interface{}) bool {
	if str, ok := obj.(string); ok {
		return utf8.RuneCountInString(str) >= m.Min
	}
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Slice {
		return v.Len() >= m.Min
	}
	return false
}

// DefaultMessage return the default MinSize error message
func (m MinSize) DefaultMessage() string {
	return fmt.Sprintf(MessageTmpls["MinSize"], m.Min)
}

// GetKey return the m.Key
func (m MinSize) GetKey() string {
	return m.Key
}

// GetLimitValue return the limit value
func (m MinSize) GetLimitValue() interface{} {
	return m.Min
}

// MaxSize Requires an array or string to be at most a given length.
type MaxSize struct {
	Max int
	Key string
}

// IsSatisfied judge whether obj is valid
func (m MaxSize) IsSatisfied(obj interface{}) bool {
	if str, ok := obj.(string); ok {
		return utf8.RuneCountInString(str) <= m.Max
	}
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Slice {
		return v.Len() <= m.Max
	}
	return false
}

// DefaultMessage return the default MaxSize error message
func (m MaxSize) DefaultMessage() string {
	return fmt.Sprintf(MessageTmpls["MaxSize"], m.Max)
}

// GetKey return the m.Key
func (m MaxSize) GetKey() string {
	return m.Key
}

// GetLimitValue return the limit value
func (m MaxSize) GetLimitValue() interface{} {
	return m.Max
}

// Length Requires an array or string to be exactly a given length.
type Length struct {
	N   int
	Key string
}

// IsSatisfied judge whether obj is valid
func (l Length) IsSatisfied(obj interface{}) bool {
	if str, ok := obj.(string); ok {
		return utf8.RuneCountInString(str) == l.N
	}
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Slice {
		return v.Len() == l.N
	}
	return false
}

// DefaultMessage return the default Length error message
func (l Length) DefaultMessage() string {
	return fmt.Sprintf(MessageTmpls["Length"], l.N)
}

// GetKey return the m.Key
func (l Length) GetKey() string {
	return l.Key
}

// GetLimitValue return the limit value
func (l Length) GetLimitValue() interface{} {
	return l.N
}

// Alpha check the alpha
type Alpha struct {
	Key string
}

// IsSatisfied judge whether obj is valid
func (a Alpha) IsSatisfied(obj interface{}) bool {
	if str, ok := obj.(string); ok {
		for _, v := range str {
			if ('Z' < v || v < 'A') && ('z' < v || v < 'a') {
				return false
			}
		}
		return true
	}
	return false
}

// DefaultMessage return the default Length error message
func (a Alpha) DefaultMessage() string {
	return fmt.Sprint(MessageTmpls["Alpha"])
}

// GetKey return the m.Key
func (a Alpha) GetKey() string {
	return a.Key
}

// GetLimitValue return the limit value
func (a Alpha) GetLimitValue() interface{} {
	return nil
}

// Numeric check number
type Numeric struct {
	Key string
}

// IsSatisfied judge whether obj is valid
func (n Numeric) IsSatisfied(obj interface{}) bool {
	if str, ok := obj.(string); ok {
		for _, v := range str {
			if '9' < v || v < '0' {
				return false
			}
		}
		return true
	}
	return false
}

// DefaultMessage return the default Length error message
func (n Numeric) DefaultMessage() string {
	return fmt.Sprint(MessageTmpls["Numeric"])
}

// GetKey return the n.Key
func (n Numeric) GetKey() string {
	return n.Key
}

// GetLimitValue return the limit value
func (n Numeric) GetLimitValue() interface{} {
	return nil
}

// AlphaNumeric check alpha and number
type AlphaNumeric struct {
	Key string
}

// IsSatisfied judge whether obj is valid
func (a AlphaNumeric) IsSatisfied(obj interface{}) bool {
	if str, ok := obj.(string); ok {
		for _, v := range str {
			if ('Z' < v || v < 'A') && ('z' < v || v < 'a') && ('9' < v || v < '0') {
				return false
			}
		}
		return true
	}
	return false
}

// DefaultMessage return the default Length error message
func (a AlphaNumeric) DefaultMessage() string {
	return fmt.Sprint(MessageTmpls["AlphaNumeric"])
}

// GetKey return the a.Key
func (a AlphaNumeric) GetKey() string {
	return a.Key
}

// GetLimitValue return the limit value
func (a AlphaNumeric) GetLimitValue() interface{} {
	return nil
}

// Match Requires a string to match a given regex.
type Match struct {
	Regexp *regexp.Regexp
	Key    string
}

// IsSatisfied judge whether obj is valid
func (m Match) IsSatisfied(obj interface{}) bool {
	return m.Regexp.MatchString(fmt.Sprintf("%v", obj))
}

// DefaultMessage return the default Match error message
func (m Match) DefaultMessage() string {
	return fmt.Sprintf(MessageTmpls["Match"], m.Regexp.String())
}

// GetKey return the m.Key
func (m Match) GetKey() string {
	return m.Key
}

// GetLimitValue return the limit value
func (m Match) GetLimitValue() interface{} {
	return m.Regexp.String()
}

// NoMatch Requires a string to not match a given regex.
type NoMatch struct {
	Match
	Key string
}

// IsSatisfied judge whether obj is valid
func (n NoMatch) IsSatisfied(obj interface{}) bool {
	return !n.Match.IsSatisfied(obj)
}

// DefaultMessage return the default NoMatch error message
func (n NoMatch) DefaultMessage() string {
	return fmt.Sprintf(MessageTmpls["NoMatch"], n.Regexp.String())
}

// GetKey return the n.Key
func (n NoMatch) GetKey() string {
	return n.Key
}

// GetLimitValue return the limit value
func (n NoMatch) GetLimitValue() interface{} {
	return n.Regexp.String()
}

var alphaDashPattern = regexp.MustCompile("[^\\d\\w-_]")

// AlphaDash check not Alpha
type AlphaDash struct {
	NoMatch
	Key string
}

// DefaultMessage return the default AlphaDash error message
func (a AlphaDash) DefaultMessage() string {
	return fmt.Sprint(MessageTmpls["AlphaDash"])
}

// GetKey return the n.Key
func (a AlphaDash) GetKey() string {
	return a.Key
}

// GetLimitValue return the limit value
func (a AlphaDash) GetLimitValue() interface{} {
	return nil
}

var emailPattern = regexp.MustCompile("[\\w!#$%&'*+/=?^_`{|}~-]+(?:\\.[\\w!#$%&'*+/=?^_`{|}~-]+)*@(?:[\\w](?:[\\w-]*[\\w])?\\.)+[a-zA-Z0-9](?:[\\w-]*[\\w])?")

// Email check struct
type Email struct {
	Match
	Key string
}

// DefaultMessage return the default Email error message
func (e Email) DefaultMessage() string {
	return fmt.Sprint(MessageTmpls["Email"])
}

// GetKey return the n.Key
func (e Email) GetKey() string {
	return e.Key
}

// GetLimitValue return the limit value
func (e Email) GetLimitValue() interface{} {
	return nil
}

var ipPattern = regexp.MustCompile("^((2[0-4]\\d|25[0-5]|[01]?\\d\\d?)\\.){3}(2[0-4]\\d|25[0-5]|[01]?\\d\\d?)$")

// IP check struct
type IP struct {
	Match
	Key string
}

// DefaultMessage return the default IP error message
func (i IP) DefaultMessage() string {
	return fmt.Sprint(MessageTmpls["IP"])
}

// GetKey return the i.Key
func (i IP) GetKey() string {
	return i.Key
}

// GetLimitValue return the limit value
func (i IP) GetLimitValue() interface{} {
	return nil
}

var base64Pattern = regexp.MustCompile("^(?:[A-Za-z0-99+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$")

// Base64 check struct
type Base64 struct {
	Match
	Key string
}

// DefaultMessage return the default Base64 error message
func (b Base64) DefaultMessage() string {
	return fmt.Sprint(MessageTmpls["Base64"])
}

// GetKey return the b.Key
func (b Base64) GetKey() string {
	return b.Key
}

// GetLimitValue return the limit value
func (b Base64) GetLimitValue() interface{} {
	return nil
}

// just for chinese mobile phone number
var mobilePattern = regexp.MustCompile("^((\\+86)|(86))?(1(([35][0-9])|[8][0-9]|[7][06789]|[4][579]))\\d{8}$")

// Mobile check struct
type Mobile struct {
	Match
	Key string
}

// DefaultMessage return the default Mobile error message
func (m Mobile) DefaultMessage() string {
	return fmt.Sprint(MessageTmpls["Mobile"])
}

// GetKey return the m.Key
func (m Mobile) GetKey() string {
	return m.Key
}

// GetLimitValue return the limit value
func (m Mobile) GetLimitValue() interface{} {
	return nil
}

// just for chinese telephone number
var telPattern = regexp.MustCompile("^(0\\d{2,3}(\\-)?)?\\d{7,8}$")

// Tel check telephone struct
type Tel struct {
	Match
	Key string
}

// DefaultMessage return the default Tel error message
func (t Tel) DefaultMessage() string {
	return fmt.Sprint(MessageTmpls["Tel"])
}

// GetKey return the t.Key
func (t Tel) GetKey() string {
	return t.Key
}

// GetLimitValue return the limit value
func (t Tel) GetLimitValue() interface{} {
	return nil
}

// Phone just for chinese telephone or mobile phone number
type Phone struct {
	Mobile
	Tel
	Key string
}

// IsSatisfied judge whether obj is valid
func (p Phone) IsSatisfied(obj interface{}) bool {
	return p.Mobile.IsSatisfied(obj) || p.Tel.IsSatisfied(obj)
}

// DefaultMessage return the default Phone error message
func (p Phone) DefaultMessage() string {
	return fmt.Sprint(MessageTmpls["Phone"])
}

// GetKey return the p.Key
func (p Phone) GetKey() string {
	return p.Key
}

// GetLimitValue return the limit value
func (p Phone) GetLimitValue() interface{} {
	return nil
}

// just for chinese zipcode
var zipCodePattern = regexp.MustCompile("^[1-9]\\d{5}$")

// ZipCode check the zip struct
type ZipCode struct {
	Match
	Key string
}

// DefaultMessage return the default Zip error message
func (z ZipCode) DefaultMessage() string {
	return fmt.Sprint(MessageTmpls["ZipCode"])
}

// GetKey return the z.Key
func (z ZipCode) GetKey() string {
	return z.Key
}

// GetLimitValue return the limit value
func (z ZipCode) GetLimitValue() interface{} {
	return nil
}
