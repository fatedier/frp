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
	"database/sql"
	"fmt"
	"reflect"
	"time"
)

// raw sql string prepared statement
type rawPrepare struct {
	rs     *rawSet
	stmt   stmtQuerier
	closed bool
}

func (o *rawPrepare) Exec(args ...interface{}) (sql.Result, error) {
	if o.closed {
		return nil, ErrStmtClosed
	}
	return o.stmt.Exec(args...)
}

func (o *rawPrepare) Close() error {
	o.closed = true
	return o.stmt.Close()
}

func newRawPreparer(rs *rawSet) (RawPreparer, error) {
	o := new(rawPrepare)
	o.rs = rs

	query := rs.query
	rs.orm.alias.DbBaser.ReplaceMarks(&query)

	st, err := rs.orm.db.Prepare(query)
	if err != nil {
		return nil, err
	}
	if Debug {
		o.stmt = newStmtQueryLog(rs.orm.alias, st, query)
	} else {
		o.stmt = st
	}
	return o, nil
}

// raw query seter
type rawSet struct {
	query string
	args  []interface{}
	orm   *orm
}

var _ RawSeter = new(rawSet)

// set args for every query
func (o rawSet) SetArgs(args ...interface{}) RawSeter {
	o.args = args
	return &o
}

// execute raw sql and return sql.Result
func (o *rawSet) Exec() (sql.Result, error) {
	query := o.query
	o.orm.alias.DbBaser.ReplaceMarks(&query)

	args := getFlatParams(nil, o.args, o.orm.alias.TZ)
	return o.orm.db.Exec(query, args...)
}

// set field value to row container
func (o *rawSet) setFieldValue(ind reflect.Value, value interface{}) {
	switch ind.Kind() {
	case reflect.Bool:
		if value == nil {
			ind.SetBool(false)
		} else if v, ok := value.(bool); ok {
			ind.SetBool(v)
		} else {
			v, _ := StrTo(ToStr(value)).Bool()
			ind.SetBool(v)
		}

	case reflect.String:
		if value == nil {
			ind.SetString("")
		} else {
			ind.SetString(ToStr(value))
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if value == nil {
			ind.SetInt(0)
		} else {
			val := reflect.ValueOf(value)
			switch val.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				ind.SetInt(val.Int())
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				ind.SetInt(int64(val.Uint()))
			default:
				v, _ := StrTo(ToStr(value)).Int64()
				ind.SetInt(v)
			}
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if value == nil {
			ind.SetUint(0)
		} else {
			val := reflect.ValueOf(value)
			switch val.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				ind.SetUint(uint64(val.Int()))
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				ind.SetUint(val.Uint())
			default:
				v, _ := StrTo(ToStr(value)).Uint64()
				ind.SetUint(v)
			}
		}
	case reflect.Float64, reflect.Float32:
		if value == nil {
			ind.SetFloat(0)
		} else {
			val := reflect.ValueOf(value)
			switch val.Kind() {
			case reflect.Float64:
				ind.SetFloat(val.Float())
			default:
				v, _ := StrTo(ToStr(value)).Float64()
				ind.SetFloat(v)
			}
		}

	case reflect.Struct:
		if value == nil {
			ind.Set(reflect.Zero(ind.Type()))

		} else if _, ok := ind.Interface().(time.Time); ok {
			var str string
			switch d := value.(type) {
			case time.Time:
				o.orm.alias.DbBaser.TimeFromDB(&d, o.orm.alias.TZ)
				ind.Set(reflect.ValueOf(d))
			case []byte:
				str = string(d)
			case string:
				str = d
			}
			if str != "" {
				if len(str) >= 19 {
					str = str[:19]
					t, err := time.ParseInLocation(formatDateTime, str, o.orm.alias.TZ)
					if err == nil {
						t = t.In(DefaultTimeLoc)
						ind.Set(reflect.ValueOf(t))
					}
				} else if len(str) >= 10 {
					str = str[:10]
					t, err := time.ParseInLocation(formatDate, str, DefaultTimeLoc)
					if err == nil {
						ind.Set(reflect.ValueOf(t))
					}
				}
			}
		}
	}
}

// set field value in loop for slice container
func (o *rawSet) loopSetRefs(refs []interface{}, sInds []reflect.Value, nIndsPtr *[]reflect.Value, eTyps []reflect.Type, init bool) {
	nInds := *nIndsPtr

	cur := 0
	for i := 0; i < len(sInds); i++ {
		sInd := sInds[i]
		eTyp := eTyps[i]

		typ := eTyp
		isPtr := false
		if typ.Kind() == reflect.Ptr {
			isPtr = true
			typ = typ.Elem()
		}
		if typ.Kind() == reflect.Ptr {
			isPtr = true
			typ = typ.Elem()
		}

		var nInd reflect.Value
		if init {
			nInd = reflect.New(sInd.Type()).Elem()
		} else {
			nInd = nInds[i]
		}

		val := reflect.New(typ)
		ind := val.Elem()

		tpName := ind.Type().String()

		if ind.Kind() == reflect.Struct {
			if tpName == "time.Time" {
				value := reflect.ValueOf(refs[cur]).Elem().Interface()
				if isPtr && value == nil {
					val = reflect.New(val.Type()).Elem()
				} else {
					o.setFieldValue(ind, value)
				}
				cur++
			}

		} else {
			value := reflect.ValueOf(refs[cur]).Elem().Interface()
			if isPtr && value == nil {
				val = reflect.New(val.Type()).Elem()
			} else {
				o.setFieldValue(ind, value)
			}
			cur++
		}

		if nInd.Kind() == reflect.Slice {
			if isPtr {
				nInd = reflect.Append(nInd, val)
			} else {
				nInd = reflect.Append(nInd, ind)
			}
		} else {
			if isPtr {
				nInd.Set(val)
			} else {
				nInd.Set(ind)
			}
		}

		nInds[i] = nInd
	}
}

// query data and map to container
func (o *rawSet) QueryRow(containers ...interface{}) error {
	var (
		refs  = make([]interface{}, 0, len(containers))
		sInds []reflect.Value
		eTyps []reflect.Type
		sMi   *modelInfo
	)
	structMode := false
	for _, container := range containers {
		val := reflect.ValueOf(container)
		ind := reflect.Indirect(val)

		if val.Kind() != reflect.Ptr {
			panic(fmt.Errorf("<RawSeter.QueryRow> all args must be use ptr"))
		}

		etyp := ind.Type()
		typ := etyp
		if typ.Kind() == reflect.Ptr {
			typ = typ.Elem()
		}

		sInds = append(sInds, ind)
		eTyps = append(eTyps, etyp)

		if typ.Kind() == reflect.Struct && typ.String() != "time.Time" {
			if len(containers) > 1 {
				panic(fmt.Errorf("<RawSeter.QueryRow> now support one struct only. see #384"))
			}

			structMode = true
			fn := getFullName(typ)
			if mi, ok := modelCache.getByFullName(fn); ok {
				sMi = mi
			}
		} else {
			var ref interface{}
			refs = append(refs, &ref)
		}
	}

	query := o.query
	o.orm.alias.DbBaser.ReplaceMarks(&query)

	args := getFlatParams(nil, o.args, o.orm.alias.TZ)
	rows, err := o.orm.db.Query(query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrNoRows
		}
		return err
	}

	defer rows.Close()

	if rows.Next() {
		if structMode {
			columns, err := rows.Columns()
			if err != nil {
				return err
			}

			columnsMp := make(map[string]interface{}, len(columns))

			refs = make([]interface{}, 0, len(columns))
			for _, col := range columns {
				var ref interface{}
				columnsMp[col] = &ref
				refs = append(refs, &ref)
			}

			if err := rows.Scan(refs...); err != nil {
				return err
			}

			ind := sInds[0]

			if ind.Kind() == reflect.Ptr {
				if ind.IsNil() || !ind.IsValid() {
					ind.Set(reflect.New(eTyps[0].Elem()))
				}
				ind = ind.Elem()
			}

			if sMi != nil {
				for _, col := range columns {
					if fi := sMi.fields.GetByColumn(col); fi != nil {
						value := reflect.ValueOf(columnsMp[col]).Elem().Interface()
						field := ind.FieldByIndex(fi.fieldIndex)
						if fi.fieldType&IsRelField > 0 {
							mf := reflect.New(fi.relModelInfo.addrField.Elem().Type())
							field.Set(mf)
							field = mf.Elem().FieldByIndex(fi.relModelInfo.fields.pk.fieldIndex)
						}
						o.setFieldValue(field, value)
					}
				}
			} else {
				for i := 0; i < ind.NumField(); i++ {
					f := ind.Field(i)
					fe := ind.Type().Field(i)
					_, tags := parseStructTag(fe.Tag.Get(defaultStructTagName))
					var col string
					if col = tags["column"]; col == "" {
						col = snakeString(fe.Name)
					}
					if v, ok := columnsMp[col]; ok {
						value := reflect.ValueOf(v).Elem().Interface()
						o.setFieldValue(f, value)
					}
				}
			}

		} else {
			if err := rows.Scan(refs...); err != nil {
				return err
			}

			nInds := make([]reflect.Value, len(sInds))
			o.loopSetRefs(refs, sInds, &nInds, eTyps, true)
			for i, sInd := range sInds {
				nInd := nInds[i]
				sInd.Set(nInd)
			}
		}

	} else {
		return ErrNoRows
	}

	return nil
}

// query data rows and map to container
func (o *rawSet) QueryRows(containers ...interface{}) (int64, error) {
	var (
		refs  = make([]interface{}, 0, len(containers))
		sInds []reflect.Value
		eTyps []reflect.Type
		sMi   *modelInfo
	)
	structMode := false
	for _, container := range containers {
		val := reflect.ValueOf(container)
		sInd := reflect.Indirect(val)
		if val.Kind() != reflect.Ptr || sInd.Kind() != reflect.Slice {
			panic(fmt.Errorf("<RawSeter.QueryRows> all args must be use ptr slice"))
		}

		etyp := sInd.Type().Elem()
		typ := etyp
		if typ.Kind() == reflect.Ptr {
			typ = typ.Elem()
		}

		sInds = append(sInds, sInd)
		eTyps = append(eTyps, etyp)

		if typ.Kind() == reflect.Struct && typ.String() != "time.Time" {
			if len(containers) > 1 {
				panic(fmt.Errorf("<RawSeter.QueryRow> now support one struct only. see #384"))
			}

			structMode = true
			fn := getFullName(typ)
			if mi, ok := modelCache.getByFullName(fn); ok {
				sMi = mi
			}
		} else {
			var ref interface{}
			refs = append(refs, &ref)
		}
	}

	query := o.query
	o.orm.alias.DbBaser.ReplaceMarks(&query)

	args := getFlatParams(nil, o.args, o.orm.alias.TZ)
	rows, err := o.orm.db.Query(query, args...)
	if err != nil {
		return 0, err
	}

	defer rows.Close()

	var cnt int64
	nInds := make([]reflect.Value, len(sInds))
	sInd := sInds[0]

	for rows.Next() {

		if structMode {
			columns, err := rows.Columns()
			if err != nil {
				return 0, err
			}

			columnsMp := make(map[string]interface{}, len(columns))

			refs = make([]interface{}, 0, len(columns))
			for _, col := range columns {
				var ref interface{}
				columnsMp[col] = &ref
				refs = append(refs, &ref)
			}

			if err := rows.Scan(refs...); err != nil {
				return 0, err
			}

			if cnt == 0 && !sInd.IsNil() {
				sInd.Set(reflect.New(sInd.Type()).Elem())
			}

			var ind reflect.Value
			if eTyps[0].Kind() == reflect.Ptr {
				ind = reflect.New(eTyps[0].Elem())
			} else {
				ind = reflect.New(eTyps[0])
			}

			if ind.Kind() == reflect.Ptr {
				ind = ind.Elem()
			}

			if sMi != nil {
				for _, col := range columns {
					if fi := sMi.fields.GetByColumn(col); fi != nil {
						value := reflect.ValueOf(columnsMp[col]).Elem().Interface()
						field := ind.FieldByIndex(fi.fieldIndex)
						if fi.fieldType&IsRelField > 0 {
							mf := reflect.New(fi.relModelInfo.addrField.Elem().Type())
							field.Set(mf)
							field = mf.Elem().FieldByIndex(fi.relModelInfo.fields.pk.fieldIndex)
						}
						o.setFieldValue(field, value)
					}
				}
			} else {
				for i := 0; i < ind.NumField(); i++ {
					f := ind.Field(i)
					fe := ind.Type().Field(i)
					_, tags := parseStructTag(fe.Tag.Get(defaultStructTagName))
					var col string
					if col = tags["column"]; col == "" {
						col = snakeString(fe.Name)
					}
					if v, ok := columnsMp[col]; ok {
						value := reflect.ValueOf(v).Elem().Interface()
						o.setFieldValue(f, value)
					}
				}
			}

			if eTyps[0].Kind() == reflect.Ptr {
				ind = ind.Addr()
			}

			sInd = reflect.Append(sInd, ind)

		} else {
			if err := rows.Scan(refs...); err != nil {
				return 0, err
			}

			o.loopSetRefs(refs, sInds, &nInds, eTyps, cnt == 0)
		}

		cnt++
	}

	if cnt > 0 {

		if structMode {
			sInds[0].Set(sInd)
		} else {
			for i, sInd := range sInds {
				nInd := nInds[i]
				sInd.Set(nInd)
			}
		}
	}

	return cnt, nil
}

func (o *rawSet) readValues(container interface{}, needCols []string) (int64, error) {
	var (
		maps  []Params
		lists []ParamsList
		list  ParamsList
	)

	typ := 0
	switch container.(type) {
	case *[]Params:
		typ = 1
	case *[]ParamsList:
		typ = 2
	case *ParamsList:
		typ = 3
	default:
		panic(fmt.Errorf("<RawSeter> unsupport read values type `%T`", container))
	}

	query := o.query
	o.orm.alias.DbBaser.ReplaceMarks(&query)

	args := getFlatParams(nil, o.args, o.orm.alias.TZ)

	var rs *sql.Rows
	rs, err := o.orm.db.Query(query, args...)
	if err != nil {
		return 0, err
	}

	defer rs.Close()

	var (
		refs   []interface{}
		cnt    int64
		cols   []string
		indexs []int
	)

	for rs.Next() {
		if cnt == 0 {
			columns, err := rs.Columns()
			if err != nil {
				return 0, err
			}
			if len(needCols) > 0 {
				indexs = make([]int, 0, len(needCols))
			} else {
				indexs = make([]int, 0, len(columns))
			}

			cols = columns
			refs = make([]interface{}, len(cols))
			for i := range refs {
				var ref sql.NullString
				refs[i] = &ref

				if len(needCols) > 0 {
					for _, c := range needCols {
						if c == cols[i] {
							indexs = append(indexs, i)
						}
					}
				} else {
					indexs = append(indexs, i)
				}
			}
		}

		if err := rs.Scan(refs...); err != nil {
			return 0, err
		}

		switch typ {
		case 1:
			params := make(Params, len(cols))
			for _, i := range indexs {
				ref := refs[i]
				value := reflect.Indirect(reflect.ValueOf(ref)).Interface().(sql.NullString)
				if value.Valid {
					params[cols[i]] = value.String
				} else {
					params[cols[i]] = nil
				}
			}
			maps = append(maps, params)
		case 2:
			params := make(ParamsList, 0, len(cols))
			for _, i := range indexs {
				ref := refs[i]
				value := reflect.Indirect(reflect.ValueOf(ref)).Interface().(sql.NullString)
				if value.Valid {
					params = append(params, value.String)
				} else {
					params = append(params, nil)
				}
			}
			lists = append(lists, params)
		case 3:
			for _, i := range indexs {
				ref := refs[i]
				value := reflect.Indirect(reflect.ValueOf(ref)).Interface().(sql.NullString)
				if value.Valid {
					list = append(list, value.String)
				} else {
					list = append(list, nil)
				}
			}
		}

		cnt++
	}

	switch v := container.(type) {
	case *[]Params:
		*v = maps
	case *[]ParamsList:
		*v = lists
	case *ParamsList:
		*v = list
	}

	return cnt, nil
}

func (o *rawSet) queryRowsTo(container interface{}, keyCol, valueCol string) (int64, error) {
	var (
		maps Params
		ind  *reflect.Value
	)

	typ := 0
	switch container.(type) {
	case *Params:
		typ = 1
	default:
		typ = 2
		vl := reflect.ValueOf(container)
		id := reflect.Indirect(vl)
		if vl.Kind() != reflect.Ptr || id.Kind() != reflect.Struct {
			panic(fmt.Errorf("<RawSeter> RowsTo unsupport type `%T` need ptr struct", container))
		}

		ind = &id
	}

	query := o.query
	o.orm.alias.DbBaser.ReplaceMarks(&query)

	args := getFlatParams(nil, o.args, o.orm.alias.TZ)

	rs, err := o.orm.db.Query(query, args...)
	if err != nil {
		return 0, err
	}

	defer rs.Close()

	var (
		refs []interface{}
		cnt  int64
		cols []string
	)

	var (
		keyIndex   = -1
		valueIndex = -1
	)

	for rs.Next() {
		if cnt == 0 {
			columns, err := rs.Columns()
			if err != nil {
				return 0, err
			}
			cols = columns
			refs = make([]interface{}, len(cols))
			for i := range refs {
				if keyCol == cols[i] {
					keyIndex = i
				}
				if typ == 1 || keyIndex == i {
					var ref sql.NullString
					refs[i] = &ref
				} else {
					var ref interface{}
					refs[i] = &ref
				}
				if valueCol == cols[i] {
					valueIndex = i
				}
			}
			if keyIndex == -1 || valueIndex == -1 {
				panic(fmt.Errorf("<RawSeter> RowsTo unknown key, value column name `%s: %s`", keyCol, valueCol))
			}
		}

		if err := rs.Scan(refs...); err != nil {
			return 0, err
		}

		if cnt == 0 {
			switch typ {
			case 1:
				maps = make(Params)
			}
		}

		key := reflect.Indirect(reflect.ValueOf(refs[keyIndex])).Interface().(sql.NullString).String

		switch typ {
		case 1:
			value := reflect.Indirect(reflect.ValueOf(refs[valueIndex])).Interface().(sql.NullString)
			if value.Valid {
				maps[key] = value.String
			} else {
				maps[key] = nil
			}

		default:
			if id := ind.FieldByName(camelString(key)); id.IsValid() {
				o.setFieldValue(id, reflect.ValueOf(refs[valueIndex]).Elem().Interface())
			}
		}

		cnt++
	}

	if typ == 1 {
		v, _ := container.(*Params)
		*v = maps
	}

	return cnt, nil
}

// query data to []map[string]interface
func (o *rawSet) Values(container *[]Params, cols ...string) (int64, error) {
	return o.readValues(container, cols)
}

// query data to [][]interface
func (o *rawSet) ValuesList(container *[]ParamsList, cols ...string) (int64, error) {
	return o.readValues(container, cols)
}

// query data to []interface
func (o *rawSet) ValuesFlat(container *ParamsList, cols ...string) (int64, error) {
	return o.readValues(container, cols)
}

// query all rows into map[string]interface with specify key and value column name.
// keyCol = "name", valueCol = "value"
// table data
// name  | value
// total | 100
// found | 200
// to map[string]interface{}{
// 	"total": 100,
// 	"found": 200,
// }
func (o *rawSet) RowsToMap(result *Params, keyCol, valueCol string) (int64, error) {
	return o.queryRowsTo(result, keyCol, valueCol)
}

// query all rows into struct with specify key and value column name.
// keyCol = "name", valueCol = "value"
// table data
// name  | value
// total | 100
// found | 200
// to struct {
// 	Total int
// 	Found int
// }
func (o *rawSet) RowsToStruct(ptrStruct interface{}, keyCol, valueCol string) (int64, error) {
	return o.queryRowsTo(ptrStruct, keyCol, valueCol)
}

// return prepared raw statement for used in times.
func (o *rawSet) Prepare() (RawPreparer, error) {
	return newRawPreparer(o)
}

func newRawSet(orm *orm, query string, args []interface{}) RawSeter {
	o := new(rawSet)
	o.query = query
	o.args = args
	o.orm = orm
	return o
}
