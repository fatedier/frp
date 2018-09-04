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

import "reflect"

// model to model struct
type queryM2M struct {
	md  interface{}
	mi  *modelInfo
	fi  *fieldInfo
	qs  *querySet
	ind reflect.Value
}

// add models to origin models when creating queryM2M.
// example:
// 	m2m := orm.QueryM2M(post,"Tag")
// 	m2m.Add(&Tag1{},&Tag2{})
//  for _,tag := range post.Tags{}
//
// make sure the relation is defined in post model struct tag.
func (o *queryM2M) Add(mds ...interface{}) (int64, error) {
	fi := o.fi
	mi := fi.relThroughModelInfo
	mfi := fi.reverseFieldInfo
	rfi := fi.reverseFieldInfoTwo

	orm := o.qs.orm
	dbase := orm.alias.DbBaser

	var models []interface{}
	var otherValues []interface{}
	var otherNames []string

	for _, colname := range mi.fields.dbcols {
		if colname != mfi.column && colname != rfi.column && colname != fi.mi.fields.pk.column &&
			mi.fields.columns[colname] != mi.fields.pk {
			otherNames = append(otherNames, colname)
		}
	}
	for i, md := range mds {
		if reflect.Indirect(reflect.ValueOf(md)).Kind() != reflect.Struct && i > 0 {
			otherValues = append(otherValues, md)
			mds = append(mds[:i], mds[i+1:]...)
		}
	}
	for _, md := range mds {
		val := reflect.ValueOf(md)
		if val.Kind() == reflect.Slice || val.Kind() == reflect.Array {
			for i := 0; i < val.Len(); i++ {
				v := val.Index(i)
				if v.CanInterface() {
					models = append(models, v.Interface())
				}
			}
		} else {
			models = append(models, md)
		}
	}

	_, v1, exist := getExistPk(o.mi, o.ind)
	if exist == false {
		panic(ErrMissPK)
	}

	names := []string{mfi.column, rfi.column}

	values := make([]interface{}, 0, len(models)*2)
	for _, md := range models {

		ind := reflect.Indirect(reflect.ValueOf(md))
		var v2 interface{}
		if ind.Kind() != reflect.Struct {
			v2 = ind.Interface()
		} else {
			_, v2, exist = getExistPk(fi.relModelInfo, ind)
			if exist == false {
				panic(ErrMissPK)
			}
		}
		values = append(values, v1, v2)

	}
	names = append(names, otherNames...)
	values = append(values, otherValues...)
	return dbase.InsertValue(orm.db, mi, true, names, values)
}

// remove models following the origin model relationship
func (o *queryM2M) Remove(mds ...interface{}) (int64, error) {
	fi := o.fi
	qs := o.qs.Filter(fi.reverseFieldInfo.name, o.md)

	nums, err := qs.Filter(fi.reverseFieldInfoTwo.name+ExprSep+"in", mds).Delete()
	if err != nil {
		return nums, err
	}
	return nums, nil
}

// check model is existed in relationship of origin model
func (o *queryM2M) Exist(md interface{}) bool {
	fi := o.fi
	return o.qs.Filter(fi.reverseFieldInfo.name, o.md).
		Filter(fi.reverseFieldInfoTwo.name, md).Exist()
}

// clean all models in related of origin model
func (o *queryM2M) Clear() (int64, error) {
	fi := o.fi
	return o.qs.Filter(fi.reverseFieldInfo.name, o.md).Delete()
}

// count all related models of origin model
func (o *queryM2M) Count() (int64, error) {
	fi := o.fi
	return o.qs.Filter(fi.reverseFieldInfo.name, o.md).Count()
}

var _ QueryM2Mer = new(queryM2M)

// create new M2M queryer.
func newQueryM2M(md interface{}, o *orm, mi *modelInfo, fi *fieldInfo, ind reflect.Value) QueryM2Mer {
	qm2m := new(queryM2M)
	qm2m.md = md
	qm2m.mi = mi
	qm2m.fi = fi
	qm2m.ind = ind
	qm2m.qs = newQuerySet(o, fi.relThroughModelInfo).(*querySet)
	return qm2m
}
