// Copyright 2026 The frp Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package validation

import (
	"fmt"
	"regexp"
	"strings"
)

// Annotation keys follow the Kubernetes "qualified name" (label key) format.
// The constants below mirror k8s.io/apimachinery/pkg/util/validation's
// qualified-name validation (v0.28.8 semantics and messages) so that frp does
// not need to import that package; newer versions of the package link klog.
const (
	qualifiedNameCharFmt    = "[A-Za-z0-9]"
	qualifiedNameExtCharFmt = "[-A-Za-z0-9_.]"
	qualifiedNameFmt        = "(" + qualifiedNameCharFmt + qualifiedNameExtCharFmt + "*)?" + qualifiedNameCharFmt
	qualifiedNameErrMsg     = "must consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character"
	qualifiedNameMaxLength  = 63

	dns1123LabelFmt          = "[a-z0-9]([-a-z0-9]*[a-z0-9])?"
	dns1123SubdomainFmt      = dns1123LabelFmt + `(\.` + dns1123LabelFmt + ")*"
	dns1123SubdomainErrorMsg = "a lowercase RFC 1123 subdomain must consist of lower case " +
		"alphanumeric characters, '-' or '.', and must start and end with an alphanumeric character"
	dns1123SubdomainMaxLength = 253
)

var (
	qualifiedNameRegexp    = regexp.MustCompile("^" + qualifiedNameFmt + "$")
	dns1123SubdomainRegexp = regexp.MustCompile("^" + dns1123SubdomainFmt + "$")
)

// IsQualifiedName reports whether value is a valid Kubernetes qualified name
// (label key), the format used by frp for proxy annotation keys. If value is
// invalid, it returns one message per violation; an empty result means valid.
//
// The function does not normalize case: callers that accept case-insensitive
// keys must lowercase value themselves (frp's annotation validation passes
// strings.ToLower(key)).
func IsQualifiedName(value string) []string {
	var errs []string
	parts := strings.Split(value, "/")
	var name string
	switch len(parts) {
	case 1:
		name = parts[0]
	case 2:
		var prefix string
		prefix, name = parts[0], parts[1]
		if len(prefix) == 0 {
			errs = append(errs, "prefix part "+emptyError())
		} else if msgs := dns1123SubdomainErrors(prefix); len(msgs) != 0 {
			errs = append(errs, prefixEach(msgs, "prefix part ")...)
		}
	default:
		return append(errs, "a qualified name "+regexError(qualifiedNameErrMsg, qualifiedNameFmt, "MyName", "my.name", "123-abc")+
			" with an optional DNS subdomain prefix and '/' (e.g. 'example.com/MyName')")
	}

	if len(name) == 0 {
		errs = append(errs, "name part "+emptyError())
	} else if len(name) > qualifiedNameMaxLength {
		errs = append(errs, "name part "+maxLenError(qualifiedNameMaxLength))
	}
	if !qualifiedNameRegexp.MatchString(name) {
		errs = append(errs, "name part "+regexError(qualifiedNameErrMsg, qualifiedNameFmt, "MyName", "my.name", "123-abc"))
	}
	return errs
}

func dns1123SubdomainErrors(value string) []string {
	var errs []string
	if len(value) > dns1123SubdomainMaxLength {
		errs = append(errs, maxLenError(dns1123SubdomainMaxLength))
	}
	if !dns1123SubdomainRegexp.MatchString(value) {
		errs = append(errs, regexError(dns1123SubdomainErrorMsg, dns1123SubdomainFmt, "example.com"))
	}
	return errs
}

func maxLenError(length int) string {
	return fmt.Sprintf("must be no more than %d characters", length)
}

func emptyError() string {
	return "must be non-empty"
}

func regexError(msg string, re string, examples ...string) string {
	if len(examples) == 0 {
		return msg + " (regex used for validation is '" + re + "')"
	}
	msg += " (e.g. "
	for i := range examples {
		if i > 0 {
			msg += " or "
		}
		msg += "'" + examples[i] + "', "
	}
	msg += "regex used for validation is '" + re + "')"
	return msg
}

func prefixEach(msgs []string, prefix string) []string {
	for i := range msgs {
		msgs[i] = prefix + msgs[i]
	}
	return msgs
}
