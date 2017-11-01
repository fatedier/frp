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
)

// single model info
type modelInfo struct {
	pkg       string
	name      string
	fullName  string
	table     string
	model     interface{}
	fields    *fields
	manual    bool
	addrField reflect.Value //store the original struct value
	uniques   []string
	isThrough bool
}

// new model info
func newModelInfo(val reflect.Value) (mi *modelInfo) {
	mi = &modelInfo{}
	mi.fields = newFields()
	ind := reflect.Indirect(val)
	mi.addrField = val
	mi.name = ind.Type().Name()
	mi.fullName = getFullName(ind.Type())
	addModelFields(mi, ind, "", []int{})
	return
}

// index: FieldByIndex returns the nested field corresponding to index
func addModelFields(mi *modelInfo, ind reflect.Value, mName string, index []int) {
	var (
		err error
		fi  *fieldInfo
		sf  reflect.StructField
	)

	for i := 0; i < ind.NumField(); i++ {
		field := ind.Field(i)
		sf = ind.Type().Field(i)
		// if the field is unexported skip
		if sf.PkgPath != "" {
			continue
		}
		// add anonymous struct fields
		if sf.Anonymous {
			addModelFields(mi, field, mName+"."+sf.Name, append(index, i))
			continue
		}

		fi, err = newFieldInfo(mi, field, sf, mName)
		if err == errSkipField {
			err = nil
			continue
		} else if err != nil {
			break
		}
		//record current field index
		fi.fieldIndex = append(index, i)
		fi.mi = mi
		fi.inModel = true
		if mi.fields.Add(fi) == false {
			err = fmt.Errorf("duplicate column name: %s", fi.column)
			break
		}
		if fi.pk {
			if mi.fields.pk != nil {
				err = fmt.Errorf("one model must have one pk field only")
				break
			} else {
				mi.fields.pk = fi
			}
		}
	}

	if err != nil {
		fmt.Println(fmt.Errorf("field: %s.%s, %s", ind.Type(), sf.Name, err))
		os.Exit(2)
	}
}

// combine related model info to new model info.
// prepare for relation models query.
func newM2MModelInfo(m1, m2 *modelInfo) (mi *modelInfo) {
	mi = new(modelInfo)
	mi.fields = newFields()
	mi.table = m1.table + "_" + m2.table + "s"
	mi.name = camelString(mi.table)
	mi.fullName = m1.pkg + "." + mi.name

	fa := new(fieldInfo) // pk
	f1 := new(fieldInfo) // m1 table RelForeignKey
	f2 := new(fieldInfo) // m2 table RelForeignKey
	fa.fieldType = TypeBigIntegerField
	fa.auto = true
	fa.pk = true
	fa.dbcol = true
	fa.name = "Id"
	fa.column = "id"
	fa.fullName = mi.fullName + "." + fa.name

	f1.dbcol = true
	f2.dbcol = true
	f1.fieldType = RelForeignKey
	f2.fieldType = RelForeignKey
	f1.name = camelString(m1.table)
	f2.name = camelString(m2.table)
	f1.fullName = mi.fullName + "." + f1.name
	f2.fullName = mi.fullName + "." + f2.name
	f1.column = m1.table + "_id"
	f2.column = m2.table + "_id"
	f1.rel = true
	f2.rel = true
	f1.relTable = m1.table
	f2.relTable = m2.table
	f1.relModelInfo = m1
	f2.relModelInfo = m2
	f1.mi = mi
	f2.mi = mi

	mi.fields.Add(fa)
	mi.fields.Add(f1)
	mi.fields.Add(f2)
	mi.fields.pk = fa

	mi.uniques = []string{f1.column, f2.column}
	return
}
