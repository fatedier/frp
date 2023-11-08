// Copyright 2023 The frp Authors
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

package util

import (
	"encoding/json"
	"fmt"
	"github.com/invopop/jsonschema"
	"reflect"
)

const packageName = "github.com/fatedier/frp"

// DumpJSONSchema dumps the JSON schema of the given type to stdout.
// If reflection fails, it panics.
func DumpJSONSchema(typ reflect.Type) {
	r := new(jsonschema.Reflector)
	r.BaseSchemaID = packageName
	err := r.AddGoComments(packageName, "./")
	if err != nil {
		// Do nothing – we don't have the source code, probably
	}

	scm := r.ReflectFromType(typ)
	b, err := json.MarshalIndent(scm, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))
}
