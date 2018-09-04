// Copyright 2016 beego authors. All Rights Reserved.
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
	"strings"

	"github.com/astaxie/beego/context"
)

// PolicyFunc defines a policy function which is invoked before the controller handler is executed.
type PolicyFunc func(*context.Context)

// FindRouter Find Router info for URL
func (p *ControllerRegister) FindPolicy(cont *context.Context) []PolicyFunc {
	var urlPath = cont.Input.URL()
	if !BConfig.RouterCaseSensitive {
		urlPath = strings.ToLower(urlPath)
	}
	httpMethod := cont.Input.Method()
	isWildcard := false
	// Find policy for current method
	t, ok := p.policies[httpMethod]
	// If not found - find policy for whole controller
	if !ok {
		t, ok = p.policies["*"]
		isWildcard = true
	}
	if ok {
		runObjects := t.Match(urlPath, cont)
		if r, ok := runObjects.([]PolicyFunc); ok {
			return r
		} else if !isWildcard {
			// If no policies found and we checked not for "*" method - try to find it
			t, ok = p.policies["*"]
			if ok {
				runObjects = t.Match(urlPath, cont)
				if r, ok = runObjects.([]PolicyFunc); ok {
					return r
				}
			}
		}
	}
	return nil
}

func (p *ControllerRegister) addToPolicy(method, pattern string, r ...PolicyFunc) {
	method = strings.ToUpper(method)
	p.enablePolicy = true
	if !BConfig.RouterCaseSensitive {
		pattern = strings.ToLower(pattern)
	}
	if t, ok := p.policies[method]; ok {
		t.AddRouter(pattern, r)
	} else {
		t := NewTree()
		t.AddRouter(pattern, r)
		p.policies[method] = t
	}
}

// Register new policy in beego
func Policy(pattern, method string, policy ...PolicyFunc) {
	BeeApp.Handlers.addToPolicy(method, pattern, policy...)
}

// Find policies and execute if were found
func (p *ControllerRegister) execPolicy(cont *context.Context, urlPath string) (started bool) {
	if !p.enablePolicy {
		return false
	}
	// Find Policy for method
	policyList := p.FindPolicy(cont)
	if len(policyList) > 0 {
		// Run policies
		for _, runPolicy := range policyList {
			runPolicy(cont)
			if cont.ResponseWriter.Started {
				return true
			}
		}
		return false
	}
	return false
}
