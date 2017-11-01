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

package orm

import (
	"fmt"
	"os"
	"reflect"
	"strings"
)

// register models.
// PrefixOrSuffix means table name prefix or suffix.
// isPrefix whether the prefix is prefix or suffix
func registerModel(PrefixOrSuffix string, model interface{}, isPrefix bool) {
	val := reflect.ValueOf(model)
	typ := reflect.Indirect(val).Type()

	if val.Kind() != reflect.Ptr {
		panic(fmt.Errorf("<orm.RegisterModel> cannot use non-ptr model struct `%s`", getFullName(typ)))
	}
	// For this case:
	// u := &User{}
	// registerModel(&u)
	if typ.Kind() == reflect.Ptr {
		panic(fmt.Errorf("<orm.RegisterModel> only allow ptr model struct, it looks you use two reference to the struct `%s`", typ))
	}

	table := getTableName(val)

	if PrefixOrSuffix != "" {
		if isPrefix {
			table = PrefixOrSuffix + table
		} else {
			table = table + PrefixOrSuffix
		}
	}
	// models's fullname is pkgpath + struct name
	name := getFullName(typ)
	if _, ok := modelCache.getByFullName(name); ok {
		fmt.Printf("<orm.RegisterModel> model `%s` repeat register, must be unique\n", name)
		os.Exit(2)
	}

	if _, ok := modelCache.get(table); ok {
		fmt.Printf("<orm.RegisterModel> table name `%s` repeat register, must be unique\n", table)
		os.Exit(2)
	}

	mi := newModelInfo(val)
	if mi.fields.pk == nil {
	outFor:
		for _, fi := range mi.fields.fieldsDB {
			if strings.ToLower(fi.name) == "id" {
				switch fi.addrValue.Elem().Kind() {
				case reflect.Int, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint32, reflect.Uint64:
					fi.auto = true
					fi.pk = true
					mi.fields.pk = fi
					break outFor
				}
			}
		}

		if mi.fields.pk == nil {
			fmt.Printf("<orm.RegisterModel> `%s` need a primary key field, default use 'id' if not set\n", name)
			os.Exit(2)
		}

	}

	mi.table = table
	mi.pkg = typ.PkgPath()
	mi.model = model
	mi.manual = true

	modelCache.set(table, mi)
}

// boostrap models
func bootStrap() {
	if modelCache.done {
		return
	}
	var (
		err    error
		models map[string]*modelInfo
	)
	if dataBaseCache.getDefault() == nil {
		err = fmt.Errorf("must have one register DataBase alias named `default`")
		goto end
	}

	// set rel and reverse model
	// RelManyToMany set the relTable
	models = modelCache.all()
	for _, mi := range models {
		for _, fi := range mi.fields.columns {
			if fi.rel || fi.reverse {
				elm := fi.addrValue.Type().Elem()
				if fi.fieldType == RelReverseMany || fi.fieldType == RelManyToMany {
					elm = elm.Elem()
				}
				// check the rel or reverse model already register
				name := getFullName(elm)
				mii, ok := modelCache.getByFullName(name)
				if !ok || mii.pkg != elm.PkgPath() {
					err = fmt.Errorf("can not find rel in field `%s`, `%s` may be miss register", fi.fullName, elm.String())
					goto end
				}
				fi.relModelInfo = mii

				switch fi.fieldType {
				case RelManyToMany:
					if fi.relThrough != "" {
						if i := strings.LastIndex(fi.relThrough, "."); i != -1 && len(fi.relThrough) > (i+1) {
							pn := fi.relThrough[:i]
							rmi, ok := modelCache.getByFullName(fi.relThrough)
							if ok == false || pn != rmi.pkg {
								err = fmt.Errorf("field `%s` wrong rel_through value `%s` cannot find table", fi.fullName, fi.relThrough)
								goto end
							}
							fi.relThroughModelInfo = rmi
							fi.relTable = rmi.table
						} else {
							err = fmt.Errorf("field `%s` wrong rel_through value `%s`", fi.fullName, fi.relThrough)
							goto end
						}
					} else {
						i := newM2MModelInfo(mi, mii)
						if fi.relTable != "" {
							i.table = fi.relTable
						}
						if v := modelCache.set(i.table, i); v != nil {
							err = fmt.Errorf("the rel table name `%s` already registered, cannot be use, please change one", fi.relTable)
							goto end
						}
						fi.relTable = i.table
						fi.relThroughModelInfo = i
					}

					fi.relThroughModelInfo.isThrough = true
				}
			}
		}
	}

	// check the rel filed while the relModelInfo also has filed point to current model
	// if not exist, add a new field to the relModelInfo
	models = modelCache.all()
	for _, mi := range models {
		for _, fi := range mi.fields.fieldsRel {
			switch fi.fieldType {
			case RelForeignKey, RelOneToOne, RelManyToMany:
				inModel := false
				for _, ffi := range fi.relModelInfo.fields.fieldsReverse {
					if ffi.relModelInfo == mi {
						inModel = true
						break
					}
				}
				if inModel == false {
					rmi := fi.relModelInfo
					ffi := new(fieldInfo)
					ffi.name = mi.name
					ffi.column = ffi.name
					ffi.fullName = rmi.fullName + "." + ffi.name
					ffi.reverse = true
					ffi.relModelInfo = mi
					ffi.mi = rmi
					if fi.fieldType == RelOneToOne {
						ffi.fieldType = RelReverseOne
					} else {
						ffi.fieldType = RelReverseMany
					}
					if rmi.fields.Add(ffi) == false {
						added := false
						for cnt := 0; cnt < 5; cnt++ {
							ffi.name = fmt.Sprintf("%s%d", mi.name, cnt)
							ffi.column = ffi.name
							ffi.fullName = rmi.fullName + "." + ffi.name
							if added = rmi.fields.Add(ffi); added {
								break
							}
						}
						if added == false {
							panic(fmt.Errorf("cannot generate auto reverse field info `%s` to `%s`", fi.fullName, ffi.fullName))
						}
					}
				}
			}
		}
	}

	models = modelCache.all()
	for _, mi := range models {
		for _, fi := range mi.fields.fieldsRel {
			switch fi.fieldType {
			case RelManyToMany:
				for _, ffi := range fi.relThroughModelInfo.fields.fieldsRel {
					switch ffi.fieldType {
					case RelOneToOne, RelForeignKey:
						if ffi.relModelInfo == fi.relModelInfo {
							fi.reverseFieldInfoTwo = ffi
						}
						if ffi.relModelInfo == mi {
							fi.reverseField = ffi.name
							fi.reverseFieldInfo = ffi
						}
					}
				}
				if fi.reverseFieldInfoTwo == nil {
					err = fmt.Errorf("can not find m2m field for m2m model `%s`, ensure your m2m model defined correct",
						fi.relThroughModelInfo.fullName)
					goto end
				}
			}
		}
	}

	models = modelCache.all()
	for _, mi := range models {
		for _, fi := range mi.fields.fieldsReverse {
			switch fi.fieldType {
			case RelReverseOne:
				found := false
			mForA:
				for _, ffi := range fi.relModelInfo.fields.fieldsByType[RelOneToOne] {
					if ffi.relModelInfo == mi {
						found = true
						fi.reverseField = ffi.name
						fi.reverseFieldInfo = ffi

						ffi.reverseField = fi.name
						ffi.reverseFieldInfo = fi
						break mForA
					}
				}
				if found == false {
					err = fmt.Errorf("reverse field `%s` not found in model `%s`", fi.fullName, fi.relModelInfo.fullName)
					goto end
				}
			case RelReverseMany:
				found := false
			mForB:
				for _, ffi := range fi.relModelInfo.fields.fieldsByType[RelForeignKey] {
					if ffi.relModelInfo == mi {
						found = true
						fi.reverseField = ffi.name
						fi.reverseFieldInfo = ffi

						ffi.reverseField = fi.name
						ffi.reverseFieldInfo = fi

						break mForB
					}
				}
				if found == false {
				mForC:
					for _, ffi := range fi.relModelInfo.fields.fieldsByType[RelManyToMany] {
						conditions := fi.relThrough != "" && fi.relThrough == ffi.relThrough ||
							fi.relTable != "" && fi.relTable == ffi.relTable ||
							fi.relThrough == "" && fi.relTable == ""
						if ffi.relModelInfo == mi && conditions {
							found = true

							fi.reverseField = ffi.reverseFieldInfoTwo.name
							fi.reverseFieldInfo = ffi.reverseFieldInfoTwo
							fi.relThroughModelInfo = ffi.relThroughModelInfo
							fi.reverseFieldInfoTwo = ffi.reverseFieldInfo
							fi.reverseFieldInfoM2M = ffi
							ffi.reverseFieldInfoM2M = fi

							break mForC
						}
					}
				}
				if found == false {
					err = fmt.Errorf("reverse field for `%s` not found in model `%s`", fi.fullName, fi.relModelInfo.fullName)
					goto end
				}
			}
		}
	}

end:
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
}

// RegisterModel register models
func RegisterModel(models ...interface{}) {
	if modelCache.done {
		panic(fmt.Errorf("RegisterModel must be run before BootStrap"))
	}
	RegisterModelWithPrefix("", models...)
}

// RegisterModelWithPrefix register models with a prefix
func RegisterModelWithPrefix(prefix string, models ...interface{}) {
	if modelCache.done {
		panic(fmt.Errorf("RegisterModelWithPrefix must be run before BootStrap"))
	}

	for _, model := range models {
		registerModel(prefix, model, true)
	}
}

// RegisterModelWithSuffix register models with a suffix
func RegisterModelWithSuffix(suffix string, models ...interface{}) {
	if modelCache.done {
		panic(fmt.Errorf("RegisterModelWithSuffix must be run before BootStrap"))
	}

	for _, model := range models {
		registerModel(suffix, model, false)
	}
}

// BootStrap bootrap models.
// make all model parsed and can not add more models
func BootStrap() {
	if modelCache.done {
		return
	}
	modelCache.Lock()
	defer modelCache.Unlock()
	bootStrap()
	modelCache.done = true
}
