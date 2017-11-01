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
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
)

const (
	formatTime     = "15:04:05"
	formatDate     = "2006-01-02"
	formatDateTime = "2006-01-02 15:04:05"
)

var (
	// ErrMissPK missing pk error
	ErrMissPK = errors.New("missed pk value")
)

var (
	operators = map[string]bool{
		"exact":     true,
		"iexact":    true,
		"contains":  true,
		"icontains": true,
		// "regex":       true,
		// "iregex":      true,
		"gt":          true,
		"gte":         true,
		"lt":          true,
		"lte":         true,
		"eq":          true,
		"nq":          true,
		"ne":	       true,
		"startswith":  true,
		"endswith":    true,
		"istartswith": true,
		"iendswith":   true,
		"in":          true,
		"between":     true,
		// "year":        true,
		// "month":       true,
		// "day":         true,
		// "week_day":    true,
		"isnull": true,
		// "search":      true,
	}
)

// an instance of dbBaser interface/
type dbBase struct {
	ins dbBaser
}

// check dbBase implements dbBaser interface.
var _ dbBaser = new(dbBase)

// get struct columns values as interface slice.
func (d *dbBase) collectValues(mi *modelInfo, ind reflect.Value, cols []string, skipAuto bool, insert bool, names *[]string, tz *time.Location) (values []interface{}, autoFields []string, err error) {
	if names == nil {
		ns := make([]string, 0, len(cols))
		names = &ns
	}
	values = make([]interface{}, 0, len(cols))

	for _, column := range cols {
		var fi *fieldInfo
		if fi, _ = mi.fields.GetByAny(column); fi != nil {
			column = fi.column
		} else {
			panic(fmt.Errorf("wrong db field/column name `%s` for model `%s`", column, mi.fullName))
		}
		if fi.dbcol == false || fi.auto && skipAuto {
			continue
		}
		value, err := d.collectFieldValue(mi, fi, ind, insert, tz)
		if err != nil {
			return nil, nil, err
		}

		// ignore empty value auto field
		if insert && fi.auto {
			if fi.fieldType&IsPositiveIntegerField > 0 {
				if vu, ok := value.(uint64); !ok || vu == 0 {
					continue
				}
			} else {
				if vu, ok := value.(int64); !ok || vu == 0 {
					continue
				}
			}
			autoFields = append(autoFields, fi.column)
		}

		*names, values = append(*names, column), append(values, value)
	}

	return
}

// get one field value in struct column as interface.
func (d *dbBase) collectFieldValue(mi *modelInfo, fi *fieldInfo, ind reflect.Value, insert bool, tz *time.Location) (interface{}, error) {
	var value interface{}
	if fi.pk {
		_, value, _ = getExistPk(mi, ind)
	} else {
		field := ind.FieldByIndex(fi.fieldIndex)
		if fi.isFielder {
			f := field.Addr().Interface().(Fielder)
			value = f.RawValue()
		} else {
			switch fi.fieldType {
			case TypeBooleanField:
				if nb, ok := field.Interface().(sql.NullBool); ok {
					value = nil
					if nb.Valid {
						value = nb.Bool
					}
				} else if field.Kind() == reflect.Ptr {
					if field.IsNil() {
						value = nil
					} else {
						value = field.Elem().Bool()
					}
				} else {
					value = field.Bool()
				}
			case TypeCharField, TypeTextField, TypeJSONField, TypeJsonbField:
				if ns, ok := field.Interface().(sql.NullString); ok {
					value = nil
					if ns.Valid {
						value = ns.String
					}
				} else if field.Kind() == reflect.Ptr {
					if field.IsNil() {
						value = nil
					} else {
						value = field.Elem().String()
					}
				} else {
					value = field.String()
				}
			case TypeFloatField, TypeDecimalField:
				if nf, ok := field.Interface().(sql.NullFloat64); ok {
					value = nil
					if nf.Valid {
						value = nf.Float64
					}
				} else if field.Kind() == reflect.Ptr {
					if field.IsNil() {
						value = nil
					} else {
						value = field.Elem().Float()
					}
				} else {
					vu := field.Interface()
					if _, ok := vu.(float32); ok {
						value, _ = StrTo(ToStr(vu)).Float64()
					} else {
						value = field.Float()
					}
				}
			case TypeTimeField, TypeDateField, TypeDateTimeField:
				value = field.Interface()
				if t, ok := value.(time.Time); ok {
					d.ins.TimeToDB(&t, tz)
					if t.IsZero() {
						value = nil
					} else {
						value = t
					}
				}
			default:
				switch {
				case fi.fieldType&IsPositiveIntegerField > 0:
					if field.Kind() == reflect.Ptr {
						if field.IsNil() {
							value = nil
						} else {
							value = field.Elem().Uint()
						}
					} else {
						value = field.Uint()
					}
				case fi.fieldType&IsIntegerField > 0:
					if ni, ok := field.Interface().(sql.NullInt64); ok {
						value = nil
						if ni.Valid {
							value = ni.Int64
						}
					} else if field.Kind() == reflect.Ptr {
						if field.IsNil() {
							value = nil
						} else {
							value = field.Elem().Int()
						}
					} else {
						value = field.Int()
					}
				case fi.fieldType&IsRelField > 0:
					if field.IsNil() {
						value = nil
					} else {
						if _, vu, ok := getExistPk(fi.relModelInfo, reflect.Indirect(field)); ok {
							value = vu
						} else {
							value = nil
						}
					}
					if fi.null == false && value == nil {
						return nil, fmt.Errorf("field `%s` cannot be NULL", fi.fullName)
					}
				}
			}
		}
		switch fi.fieldType {
		case TypeTimeField, TypeDateField, TypeDateTimeField:
			if fi.autoNow || fi.autoNowAdd && insert {
				if insert {
					if t, ok := value.(time.Time); ok && !t.IsZero() {
						break
					}
				}
				tnow := time.Now()
				d.ins.TimeToDB(&tnow, tz)
				value = tnow
				if fi.isFielder {
					f := field.Addr().Interface().(Fielder)
					f.SetRaw(tnow.In(DefaultTimeLoc))
				} else if field.Kind() == reflect.Ptr {
					v := tnow.In(DefaultTimeLoc)
					field.Set(reflect.ValueOf(&v))
				} else {
					field.Set(reflect.ValueOf(tnow.In(DefaultTimeLoc)))
				}
			}
		case TypeJSONField, TypeJsonbField:
			if s, ok := value.(string); (ok && len(s) == 0) || value == nil {
				if fi.colDefault && fi.initial.Exist() {
					value = fi.initial.String()
				} else {
					value = nil
				}
			}
		}
	}
	return value, nil
}

// create insert sql preparation statement object.
func (d *dbBase) PrepareInsert(q dbQuerier, mi *modelInfo) (stmtQuerier, string, error) {
	Q := d.ins.TableQuote()

	dbcols := make([]string, 0, len(mi.fields.dbcols))
	marks := make([]string, 0, len(mi.fields.dbcols))
	for _, fi := range mi.fields.fieldsDB {
		if fi.auto == false {
			dbcols = append(dbcols, fi.column)
			marks = append(marks, "?")
		}
	}
	qmarks := strings.Join(marks, ", ")
	sep := fmt.Sprintf("%s, %s", Q, Q)
	columns := strings.Join(dbcols, sep)

	query := fmt.Sprintf("INSERT INTO %s%s%s (%s%s%s) VALUES (%s)", Q, mi.table, Q, Q, columns, Q, qmarks)

	d.ins.ReplaceMarks(&query)

	d.ins.HasReturningID(mi, &query)

	stmt, err := q.Prepare(query)
	return stmt, query, err
}

// insert struct with prepared statement and given struct reflect value.
func (d *dbBase) InsertStmt(stmt stmtQuerier, mi *modelInfo, ind reflect.Value, tz *time.Location) (int64, error) {
	values, _, err := d.collectValues(mi, ind, mi.fields.dbcols, true, true, nil, tz)
	if err != nil {
		return 0, err
	}

	if d.ins.HasReturningID(mi, nil) {
		row := stmt.QueryRow(values...)
		var id int64
		err := row.Scan(&id)
		return id, err
	}
	res, err := stmt.Exec(values...)
	if err == nil {
		return res.LastInsertId()
	}
	return 0, err
}

// query sql ,read records and persist in dbBaser.
func (d *dbBase) Read(q dbQuerier, mi *modelInfo, ind reflect.Value, tz *time.Location, cols []string, isForUpdate bool) error {
	var whereCols []string
	var args []interface{}

	// if specify cols length > 0, then use it for where condition.
	if len(cols) > 0 {
		var err error
		whereCols = make([]string, 0, len(cols))
		args, _, err = d.collectValues(mi, ind, cols, false, false, &whereCols, tz)
		if err != nil {
			return err
		}
	} else {
		// default use pk value as where condtion.
		pkColumn, pkValue, ok := getExistPk(mi, ind)
		if ok == false {
			return ErrMissPK
		}
		whereCols = []string{pkColumn}
		args = append(args, pkValue)
	}

	Q := d.ins.TableQuote()

	sep := fmt.Sprintf("%s, %s", Q, Q)
	sels := strings.Join(mi.fields.dbcols, sep)
	colsNum := len(mi.fields.dbcols)

	sep = fmt.Sprintf("%s = ? AND %s", Q, Q)
	wheres := strings.Join(whereCols, sep)

	forUpdate := ""
	if isForUpdate {
		forUpdate = "FOR UPDATE"
	}

	query := fmt.Sprintf("SELECT %s%s%s FROM %s%s%s WHERE %s%s%s = ? %s", Q, sels, Q, Q, mi.table, Q, Q, wheres, Q, forUpdate)

	refs := make([]interface{}, colsNum)
	for i := range refs {
		var ref interface{}
		refs[i] = &ref
	}

	d.ins.ReplaceMarks(&query)

	row := q.QueryRow(query, args...)
	if err := row.Scan(refs...); err != nil {
		if err == sql.ErrNoRows {
			return ErrNoRows
		}
		return err
	}
	elm := reflect.New(mi.addrField.Elem().Type())
	mind := reflect.Indirect(elm)
	d.setColsValues(mi, &mind, mi.fields.dbcols, refs, tz)
	ind.Set(mind)
	return nil
}

// execute insert sql dbQuerier with given struct reflect.Value.
func (d *dbBase) Insert(q dbQuerier, mi *modelInfo, ind reflect.Value, tz *time.Location) (int64, error) {
	names := make([]string, 0, len(mi.fields.dbcols))
	values, autoFields, err := d.collectValues(mi, ind, mi.fields.dbcols, false, true, &names, tz)
	if err != nil {
		return 0, err
	}

	id, err := d.InsertValue(q, mi, false, names, values)
	if err != nil {
		return 0, err
	}

	if len(autoFields) > 0 {
		err = d.ins.setval(q, mi, autoFields)
	}
	return id, err
}

// multi-insert sql with given slice struct reflect.Value.
func (d *dbBase) InsertMulti(q dbQuerier, mi *modelInfo, sind reflect.Value, bulk int, tz *time.Location) (int64, error) {
	var (
		cnt    int64
		nums   int
		values []interface{}
		names  []string
	)

	// typ := reflect.Indirect(mi.addrField).Type()

	length, autoFields := sind.Len(), make([]string, 0, 1)

	for i := 1; i <= length; i++ {

		ind := reflect.Indirect(sind.Index(i - 1))

		// Is this needed ?
		// if !ind.Type().AssignableTo(typ) {
		// 	return cnt, ErrArgs
		// }

		if i == 1 {
			var (
				vus []interface{}
				err error
			)
			vus, autoFields, err = d.collectValues(mi, ind, mi.fields.dbcols, false, true, &names, tz)
			if err != nil {
				return cnt, err
			}
			values = make([]interface{}, bulk*len(vus))
			nums += copy(values, vus)
		} else {
			vus, _, err := d.collectValues(mi, ind, mi.fields.dbcols, false, true, nil, tz)
			if err != nil {
				return cnt, err
			}

			if len(vus) != len(names) {
				return cnt, ErrArgs
			}

			nums += copy(values[nums:], vus)
		}

		if i > 1 && i%bulk == 0 || length == i {
			num, err := d.InsertValue(q, mi, true, names, values[:nums])
			if err != nil {
				return cnt, err
			}
			cnt += num
			nums = 0
		}
	}

	var err error
	if len(autoFields) > 0 {
		err = d.ins.setval(q, mi, autoFields)
	}

	return cnt, err
}

// execute insert sql with given struct and given values.
// insert the given values, not the field values in struct.
func (d *dbBase) InsertValue(q dbQuerier, mi *modelInfo, isMulti bool, names []string, values []interface{}) (int64, error) {
	Q := d.ins.TableQuote()

	marks := make([]string, len(names))
	for i := range marks {
		marks[i] = "?"
	}

	sep := fmt.Sprintf("%s, %s", Q, Q)
	qmarks := strings.Join(marks, ", ")
	columns := strings.Join(names, sep)

	multi := len(values) / len(names)

	if isMulti {
		qmarks = strings.Repeat(qmarks+"), (", multi-1) + qmarks
	}

	query := fmt.Sprintf("INSERT INTO %s%s%s (%s%s%s) VALUES (%s)", Q, mi.table, Q, Q, columns, Q, qmarks)

	d.ins.ReplaceMarks(&query)

	if isMulti || !d.ins.HasReturningID(mi, &query) {
		res, err := q.Exec(query, values...)
		if err == nil {
			if isMulti {
				return res.RowsAffected()
			}
			return res.LastInsertId()
		}
		return 0, err
	}
	row := q.QueryRow(query, values...)
	var id int64
	err := row.Scan(&id)
	return id, err
}

// InsertOrUpdate a row
// If your primary key or unique column conflict will update
// If no will insert
func (d *dbBase) InsertOrUpdate(q dbQuerier, mi *modelInfo, ind reflect.Value, a *alias, args ...string) (int64, error) {
	args0 := ""
	iouStr := ""
	argsMap := map[string]string{}
	switch a.Driver {
	case DRMySQL:
		iouStr = "ON DUPLICATE KEY UPDATE"
	case DRPostgres:
		if len(args) == 0 {
			return 0, fmt.Errorf("`%s` use InsertOrUpdate must have a conflict column", a.DriverName)
		} else {
			args0 = strings.ToLower(args[0])
			iouStr = fmt.Sprintf("ON CONFLICT (%s) DO UPDATE SET", args0)
		}
	default:
		return 0, fmt.Errorf("`%s` nonsupport InsertOrUpdate in beego", a.DriverName)
	}

	//Get on the key-value pairs
	for _, v := range args {
		kv := strings.Split(v, "=")
		if len(kv) == 2 {
			argsMap[strings.ToLower(kv[0])] = kv[1]
		}
	}

	isMulti := false
	names := make([]string, 0, len(mi.fields.dbcols)-1)
	Q := d.ins.TableQuote()
	values, _, err := d.collectValues(mi, ind, mi.fields.dbcols, true, true, &names, a.TZ)

	if err != nil {
		return 0, err
	}

	marks := make([]string, len(names))
	updateValues := make([]interface{}, 0)
	updates := make([]string, len(names))
	var conflitValue interface{}
	for i, v := range names {
		marks[i] = "?"
		valueStr := argsMap[strings.ToLower(v)]
		if v == args0 {
			conflitValue = values[i]
		}
		if valueStr != "" {
			switch a.Driver {
			case DRMySQL:
				updates[i] = v + "=" + valueStr
			case DRPostgres:
				if conflitValue != nil {
					//postgres ON CONFLICT DO UPDATE SET can`t use colu=colu+values
					updates[i] = fmt.Sprintf("%s=(select %s from %s where %s = ? )", v, valueStr, mi.table, args0)
					updateValues = append(updateValues, conflitValue)
				} else {
					return 0, fmt.Errorf("`%s` must be in front of `%s` in your struct", args0, v)
				}
			}
		} else {
			updates[i] = v + "=?"
			updateValues = append(updateValues, values[i])
		}
	}

	values = append(values, updateValues...)

	sep := fmt.Sprintf("%s, %s", Q, Q)
	qmarks := strings.Join(marks, ", ")
	qupdates := strings.Join(updates, ", ")
	columns := strings.Join(names, sep)

	multi := len(values) / len(names)

	if isMulti {
		qmarks = strings.Repeat(qmarks+"), (", multi-1) + qmarks
	}
	//conflitValue maybe is a int,can`t use fmt.Sprintf
	query := fmt.Sprintf("INSERT INTO %s%s%s (%s%s%s) VALUES (%s) %s "+qupdates, Q, mi.table, Q, Q, columns, Q, qmarks, iouStr)

	d.ins.ReplaceMarks(&query)

	if isMulti || !d.ins.HasReturningID(mi, &query) {
		res, err := q.Exec(query, values...)
		if err == nil {
			if isMulti {
				return res.RowsAffected()
			}
			return res.LastInsertId()
		}
		return 0, err
	}

	row := q.QueryRow(query, values...)
	var id int64
	err = row.Scan(&id)
	if err.Error() == `pq: syntax error at or near "ON"` {
		err = fmt.Errorf("postgres version must 9.5 or higher")
	}
	return id, err
}

// execute update sql dbQuerier with given struct reflect.Value.
func (d *dbBase) Update(q dbQuerier, mi *modelInfo, ind reflect.Value, tz *time.Location, cols []string) (int64, error) {
	pkName, pkValue, ok := getExistPk(mi, ind)
	if ok == false {
		return 0, ErrMissPK
	}

	var setNames []string

	// if specify cols length is zero, then commit all columns.
	if len(cols) == 0 {
		cols = mi.fields.dbcols
		setNames = make([]string, 0, len(mi.fields.dbcols)-1)
	} else {
		setNames = make([]string, 0, len(cols))
	}

	setValues, _, err := d.collectValues(mi, ind, cols, true, false, &setNames, tz)
	if err != nil {
		return 0, err
	}

	setValues = append(setValues, pkValue)

	Q := d.ins.TableQuote()

	sep := fmt.Sprintf("%s = ?, %s", Q, Q)
	setColumns := strings.Join(setNames, sep)

	query := fmt.Sprintf("UPDATE %s%s%s SET %s%s%s = ? WHERE %s%s%s = ?", Q, mi.table, Q, Q, setColumns, Q, Q, pkName, Q)

	d.ins.ReplaceMarks(&query)

	res, err := q.Exec(query, setValues...)
	if err == nil {
		return res.RowsAffected()
	}
	return 0, err
}

// execute delete sql dbQuerier with given struct reflect.Value.
// delete index is pk.
func (d *dbBase) Delete(q dbQuerier, mi *modelInfo, ind reflect.Value, tz *time.Location, cols []string) (int64, error) {
	var whereCols []string
	var args []interface{}
	// if specify cols length > 0, then use it for where condition.
	if len(cols) > 0 {
		var err error
		whereCols = make([]string, 0, len(cols))
		args, _, err = d.collectValues(mi, ind, cols, false, false, &whereCols, tz)
		if err != nil {
			return 0, err
		}
	} else {
		// default use pk value as where condtion.
		pkColumn, pkValue, ok := getExistPk(mi, ind)
		if ok == false {
			return 0, ErrMissPK
		}
		whereCols = []string{pkColumn}
		args = append(args, pkValue)
	}

	Q := d.ins.TableQuote()

	sep := fmt.Sprintf("%s = ? AND %s", Q, Q)
	wheres := strings.Join(whereCols, sep)

	query := fmt.Sprintf("DELETE FROM %s%s%s WHERE %s%s%s = ?", Q, mi.table, Q, Q, wheres, Q)

	d.ins.ReplaceMarks(&query)
	res, err := q.Exec(query, args...)
	if err == nil {
		num, err := res.RowsAffected()
		if err != nil {
			return 0, err
		}
		if num > 0 {
			if mi.fields.pk.auto {
				if mi.fields.pk.fieldType&IsPositiveIntegerField > 0 {
					ind.FieldByIndex(mi.fields.pk.fieldIndex).SetUint(0)
				} else {
					ind.FieldByIndex(mi.fields.pk.fieldIndex).SetInt(0)
				}
			}
			err := d.deleteRels(q, mi, args, tz)
			if err != nil {
				return num, err
			}
		}
		return num, err
	}
	return 0, err
}

// update table-related record by querySet.
// need querySet not struct reflect.Value to update related records.
func (d *dbBase) UpdateBatch(q dbQuerier, qs *querySet, mi *modelInfo, cond *Condition, params Params, tz *time.Location) (int64, error) {
	columns := make([]string, 0, len(params))
	values := make([]interface{}, 0, len(params))
	for col, val := range params {
		if fi, ok := mi.fields.GetByAny(col); ok == false || fi.dbcol == false {
			panic(fmt.Errorf("wrong field/column name `%s`", col))
		} else {
			columns = append(columns, fi.column)
			values = append(values, val)
		}
	}

	if len(columns) == 0 {
		panic(fmt.Errorf("update params cannot empty"))
	}

	tables := newDbTables(mi, d.ins)
	if qs != nil {
		tables.parseRelated(qs.related, qs.relDepth)
	}

	where, args := tables.getCondSQL(cond, false, tz)

	values = append(values, args...)

	join := tables.getJoinSQL()

	var query, T string

	Q := d.ins.TableQuote()

	if d.ins.SupportUpdateJoin() {
		T = "T0."
	}

	cols := make([]string, 0, len(columns))

	for i, v := range columns {
		col := fmt.Sprintf("%s%s%s%s", T, Q, v, Q)
		if c, ok := values[i].(colValue); ok {
			switch c.opt {
			case ColAdd:
				cols = append(cols, col+" = "+col+" + ?")
			case ColMinus:
				cols = append(cols, col+" = "+col+" - ?")
			case ColMultiply:
				cols = append(cols, col+" = "+col+" * ?")
			case ColExcept:
				cols = append(cols, col+" = "+col+" / ?")
			}
			values[i] = c.value
		} else {
			cols = append(cols, col+" = ?")
		}
	}

	sets := strings.Join(cols, ", ") + " "

	if d.ins.SupportUpdateJoin() {
		query = fmt.Sprintf("UPDATE %s%s%s T0 %sSET %s%s", Q, mi.table, Q, join, sets, where)
	} else {
		supQuery := fmt.Sprintf("SELECT T0.%s%s%s FROM %s%s%s T0 %s%s", Q, mi.fields.pk.column, Q, Q, mi.table, Q, join, where)
		query = fmt.Sprintf("UPDATE %s%s%s SET %sWHERE %s%s%s IN ( %s )", Q, mi.table, Q, sets, Q, mi.fields.pk.column, Q, supQuery)
	}

	d.ins.ReplaceMarks(&query)
	res, err := q.Exec(query, values...)
	if err == nil {
		return res.RowsAffected()
	}
	return 0, err
}

// delete related records.
// do UpdateBanch or DeleteBanch by condition of tables' relationship.
func (d *dbBase) deleteRels(q dbQuerier, mi *modelInfo, args []interface{}, tz *time.Location) error {
	for _, fi := range mi.fields.fieldsReverse {
		fi = fi.reverseFieldInfo
		switch fi.onDelete {
		case odCascade:
			cond := NewCondition().And(fmt.Sprintf("%s__in", fi.name), args...)
			_, err := d.DeleteBatch(q, nil, fi.mi, cond, tz)
			if err != nil {
				return err
			}
		case odSetDefault, odSetNULL:
			cond := NewCondition().And(fmt.Sprintf("%s__in", fi.name), args...)
			params := Params{fi.column: nil}
			if fi.onDelete == odSetDefault {
				params[fi.column] = fi.initial.String()
			}
			_, err := d.UpdateBatch(q, nil, fi.mi, cond, params, tz)
			if err != nil {
				return err
			}
		case odDoNothing:
		}
	}
	return nil
}

// delete table-related records.
func (d *dbBase) DeleteBatch(q dbQuerier, qs *querySet, mi *modelInfo, cond *Condition, tz *time.Location) (int64, error) {
	tables := newDbTables(mi, d.ins)
	tables.skipEnd = true

	if qs != nil {
		tables.parseRelated(qs.related, qs.relDepth)
	}

	if cond == nil || cond.IsEmpty() {
		panic(fmt.Errorf("delete operation cannot execute without condition"))
	}

	Q := d.ins.TableQuote()

	where, args := tables.getCondSQL(cond, false, tz)
	join := tables.getJoinSQL()

	cols := fmt.Sprintf("T0.%s%s%s", Q, mi.fields.pk.column, Q)
	query := fmt.Sprintf("SELECT %s FROM %s%s%s T0 %s%s", cols, Q, mi.table, Q, join, where)

	d.ins.ReplaceMarks(&query)

	var rs *sql.Rows
	r, err := q.Query(query, args...)
	if err != nil {
		return 0, err
	}
	rs = r
	defer rs.Close()

	var ref interface{}
	args = make([]interface{}, 0)
	cnt := 0
	for rs.Next() {
		if err := rs.Scan(&ref); err != nil {
			return 0, err
		}
		args = append(args, reflect.ValueOf(ref).Interface())
		cnt++
	}

	if cnt == 0 {
		return 0, nil
	}

	marks := make([]string, len(args))
	for i := range marks {
		marks[i] = "?"
	}
	sql := fmt.Sprintf("IN (%s)", strings.Join(marks, ", "))
	query = fmt.Sprintf("DELETE FROM %s%s%s WHERE %s%s%s %s", Q, mi.table, Q, Q, mi.fields.pk.column, Q, sql)

	d.ins.ReplaceMarks(&query)
	res, err := q.Exec(query, args...)
	if err == nil {
		num, err := res.RowsAffected()
		if err != nil {
			return 0, err
		}
		if num > 0 {
			err := d.deleteRels(q, mi, args, tz)
			if err != nil {
				return num, err
			}
		}
		return num, nil
	}
	return 0, err
}

// read related records.
func (d *dbBase) ReadBatch(q dbQuerier, qs *querySet, mi *modelInfo, cond *Condition, container interface{}, tz *time.Location, cols []string) (int64, error) {

	val := reflect.ValueOf(container)
	ind := reflect.Indirect(val)

	errTyp := true
	one := true
	isPtr := true

	if val.Kind() == reflect.Ptr {
		fn := ""
		if ind.Kind() == reflect.Slice {
			one = false
			typ := ind.Type().Elem()
			switch typ.Kind() {
			case reflect.Ptr:
				fn = getFullName(typ.Elem())
			case reflect.Struct:
				isPtr = false
				fn = getFullName(typ)
			}
		} else {
			fn = getFullName(ind.Type())
		}
		errTyp = fn != mi.fullName
	}

	if errTyp {
		if one {
			panic(fmt.Errorf("wrong object type `%s` for rows scan, need *%s", val.Type(), mi.fullName))
		} else {
			panic(fmt.Errorf("wrong object type `%s` for rows scan, need *[]*%s or *[]%s", val.Type(), mi.fullName, mi.fullName))
		}
	}

	rlimit := qs.limit
	offset := qs.offset

	Q := d.ins.TableQuote()

	var tCols []string
	if len(cols) > 0 {
		hasRel := len(qs.related) > 0 || qs.relDepth > 0
		tCols = make([]string, 0, len(cols))
		var maps map[string]bool
		if hasRel {
			maps = make(map[string]bool)
		}
		for _, col := range cols {
			if fi, ok := mi.fields.GetByAny(col); ok {
				tCols = append(tCols, fi.column)
				if hasRel {
					maps[fi.column] = true
				}
			} else {
				panic(fmt.Errorf("wrong field/column name `%s`", col))
			}
		}
		if hasRel {
			for _, fi := range mi.fields.fieldsDB {
				if fi.fieldType&IsRelField > 0 {
					if maps[fi.column] == false {
						tCols = append(tCols, fi.column)
					}
				}
			}
		}
	} else {
		tCols = mi.fields.dbcols
	}

	colsNum := len(tCols)
	sep := fmt.Sprintf("%s, T0.%s", Q, Q)
	sels := fmt.Sprintf("T0.%s%s%s", Q, strings.Join(tCols, sep), Q)

	tables := newDbTables(mi, d.ins)
	tables.parseRelated(qs.related, qs.relDepth)

	where, args := tables.getCondSQL(cond, false, tz)
	groupBy := tables.getGroupSQL(qs.groups)
	orderBy := tables.getOrderSQL(qs.orders)
	limit := tables.getLimitSQL(mi, offset, rlimit)
	join := tables.getJoinSQL()

	for _, tbl := range tables.tables {
		if tbl.sel {
			colsNum += len(tbl.mi.fields.dbcols)
			sep := fmt.Sprintf("%s, %s.%s", Q, tbl.index, Q)
			sels += fmt.Sprintf(", %s.%s%s%s", tbl.index, Q, strings.Join(tbl.mi.fields.dbcols, sep), Q)
		}
	}

	sqlSelect := "SELECT"
	if qs.distinct {
		sqlSelect += " DISTINCT"
	}
	query := fmt.Sprintf("%s %s FROM %s%s%s T0 %s%s%s%s%s", sqlSelect, sels, Q, mi.table, Q, join, where, groupBy, orderBy, limit)

	d.ins.ReplaceMarks(&query)

	var rs *sql.Rows
	r, err := q.Query(query, args...)
	if err != nil {
		return 0, err
	}
	rs = r

	refs := make([]interface{}, colsNum)
	for i := range refs {
		var ref interface{}
		refs[i] = &ref
	}

	defer rs.Close()

	slice := ind

	var cnt int64
	for rs.Next() {
		if one && cnt == 0 || one == false {
			if err := rs.Scan(refs...); err != nil {
				return 0, err
			}

			elm := reflect.New(mi.addrField.Elem().Type())
			mind := reflect.Indirect(elm)

			cacheV := make(map[string]*reflect.Value)
			cacheM := make(map[string]*modelInfo)
			trefs := refs

			d.setColsValues(mi, &mind, tCols, refs[:len(tCols)], tz)
			trefs = refs[len(tCols):]

			for _, tbl := range tables.tables {
				// loop selected tables
				if tbl.sel {
					last := mind
					names := ""
					mmi := mi
					// loop cascade models
					for _, name := range tbl.names {
						names += name
						if val, ok := cacheV[names]; ok {
							last = *val
							mmi = cacheM[names]
						} else {
							fi := mmi.fields.GetByName(name)
							lastm := mmi
							mmi = fi.relModelInfo
							field := last
							if last.Kind() != reflect.Invalid {
								field = reflect.Indirect(last.FieldByIndex(fi.fieldIndex))
								if field.IsValid() {
									d.setColsValues(mmi, &field, mmi.fields.dbcols, trefs[:len(mmi.fields.dbcols)], tz)
									for _, fi := range mmi.fields.fieldsReverse {
										if fi.inModel && fi.reverseFieldInfo.mi == lastm {
											if fi.reverseFieldInfo != nil {
												f := field.FieldByIndex(fi.fieldIndex)
												if f.Kind() == reflect.Ptr {
													f.Set(last.Addr())
												}
											}
										}
									}
									last = field
								}
							}
							cacheV[names] = &field
							cacheM[names] = mmi
						}
					}
					trefs = trefs[len(mmi.fields.dbcols):]
				}
			}

			if one {
				ind.Set(mind)
			} else {
				if cnt == 0 {
					// you can use a empty & caped container list
					// orm will not replace it
					if ind.Len() != 0 {
						// if container is not empty
						// create a new one
						slice = reflect.New(ind.Type()).Elem()
					}
				}

				if isPtr {
					slice = reflect.Append(slice, mind.Addr())
				} else {
					slice = reflect.Append(slice, mind)
				}
			}
		}
		cnt++
	}

	if one == false {
		if cnt > 0 {
			ind.Set(slice)
		} else {
			// when a result is empty and container is nil
			// to set a empty container
			if ind.IsNil() {
				ind.Set(reflect.MakeSlice(ind.Type(), 0, 0))
			}
		}
	}

	return cnt, nil
}

// excute count sql and return count result int64.
func (d *dbBase) Count(q dbQuerier, qs *querySet, mi *modelInfo, cond *Condition, tz *time.Location) (cnt int64, err error) {
	tables := newDbTables(mi, d.ins)
	tables.parseRelated(qs.related, qs.relDepth)

	where, args := tables.getCondSQL(cond, false, tz)
	groupBy := tables.getGroupSQL(qs.groups)
	tables.getOrderSQL(qs.orders)
	join := tables.getJoinSQL()

	Q := d.ins.TableQuote()

	query := fmt.Sprintf("SELECT COUNT(*) FROM %s%s%s T0 %s%s%s", Q, mi.table, Q, join, where, groupBy)

	if groupBy != "" {
		query = fmt.Sprintf("SELECT COUNT(*) FROM (%s) AS T", query)
	}

	d.ins.ReplaceMarks(&query)

	row := q.QueryRow(query, args...)

	err = row.Scan(&cnt)
	return
}

// generate sql with replacing operator string placeholders and replaced values.
func (d *dbBase) GenerateOperatorSQL(mi *modelInfo, fi *fieldInfo, operator string, args []interface{}, tz *time.Location) (string, []interface{}) {
	sql := ""
	params := getFlatParams(fi, args, tz)

	if len(params) == 0 {
		panic(fmt.Errorf("operator `%s` need at least one args", operator))
	}
	arg := params[0]

	switch operator {
	case "in":
		marks := make([]string, len(params))
		for i := range marks {
			marks[i] = "?"
		}
		sql = fmt.Sprintf("IN (%s)", strings.Join(marks, ", "))
	case "between":
		if len(params) != 2 {
			panic(fmt.Errorf("operator `%s` need 2 args not %d", operator, len(params)))
		}
		sql = "BETWEEN ? AND ?"
	default:
		if len(params) > 1 {
			panic(fmt.Errorf("operator `%s` need 1 args not %d", operator, len(params)))
		}
		sql = d.ins.OperatorSQL(operator)
		switch operator {
		case "exact":
			if arg == nil {
				params[0] = "IS NULL"
			}
		case "iexact", "contains", "icontains", "startswith", "endswith", "istartswith", "iendswith":
			param := strings.Replace(ToStr(arg), `%`, `\%`, -1)
			switch operator {
			case "iexact":
			case "contains", "icontains":
				param = fmt.Sprintf("%%%s%%", param)
			case "startswith", "istartswith":
				param = fmt.Sprintf("%s%%", param)
			case "endswith", "iendswith":
				param = fmt.Sprintf("%%%s", param)
			}
			params[0] = param
		case "isnull":
			if b, ok := arg.(bool); ok {
				if b {
					sql = "IS NULL"
				} else {
					sql = "IS NOT NULL"
				}
				params = nil
			} else {
				panic(fmt.Errorf("operator `%s` need a bool value not `%T`", operator, arg))
			}
		}
	}
	return sql, params
}

// gernerate sql string with inner function, such as UPPER(text).
func (d *dbBase) GenerateOperatorLeftCol(*fieldInfo, string, *string) {
	// default not use
}

// set values to struct column.
func (d *dbBase) setColsValues(mi *modelInfo, ind *reflect.Value, cols []string, values []interface{}, tz *time.Location) {
	for i, column := range cols {
		val := reflect.Indirect(reflect.ValueOf(values[i])).Interface()

		fi := mi.fields.GetByColumn(column)

		field := ind.FieldByIndex(fi.fieldIndex)

		value, err := d.convertValueFromDB(fi, val, tz)
		if err != nil {
			panic(fmt.Errorf("Raw value: `%v` %s", val, err.Error()))
		}

		_, err = d.setFieldValue(fi, value, field)

		if err != nil {
			panic(fmt.Errorf("Raw value: `%v` %s", val, err.Error()))
		}
	}
}

// convert value from database result to value following in field type.
func (d *dbBase) convertValueFromDB(fi *fieldInfo, val interface{}, tz *time.Location) (interface{}, error) {
	if val == nil {
		return nil, nil
	}

	var value interface{}
	var tErr error

	var str *StrTo
	switch v := val.(type) {
	case []byte:
		s := StrTo(string(v))
		str = &s
	case string:
		s := StrTo(v)
		str = &s
	}

	fieldType := fi.fieldType

setValue:
	switch {
	case fieldType == TypeBooleanField:
		if str == nil {
			switch v := val.(type) {
			case int64:
				b := v == 1
				value = b
			default:
				s := StrTo(ToStr(v))
				str = &s
			}
		}
		if str != nil {
			b, err := str.Bool()
			if err != nil {
				tErr = err
				goto end
			}
			value = b
		}
	case fieldType == TypeCharField || fieldType == TypeTextField || fieldType == TypeJSONField || fieldType == TypeJsonbField:
		if str == nil {
			value = ToStr(val)
		} else {
			value = str.String()
		}
	case fieldType == TypeTimeField || fieldType == TypeDateField || fieldType == TypeDateTimeField:
		if str == nil {
			switch t := val.(type) {
			case time.Time:
				d.ins.TimeFromDB(&t, tz)
				value = t
			default:
				s := StrTo(ToStr(t))
				str = &s
			}
		}
		if str != nil {
			s := str.String()
			var (
				t   time.Time
				err error
			)
			if len(s) >= 19 {
				s = s[:19]
				t, err = time.ParseInLocation(formatDateTime, s, tz)
			} else if len(s) >= 10 {
				if len(s) > 10 {
					s = s[:10]
				}
				t, err = time.ParseInLocation(formatDate, s, tz)
			} else if len(s) >= 8 {
				if len(s) > 8 {
					s = s[:8]
				}
				t, err = time.ParseInLocation(formatTime, s, tz)
			}
			t = t.In(DefaultTimeLoc)

			if err != nil && s != "00:00:00" && s != "0000-00-00" && s != "0000-00-00 00:00:00" {
				tErr = err
				goto end
			}
			value = t
		}
	case fieldType&IsIntegerField > 0:
		if str == nil {
			s := StrTo(ToStr(val))
			str = &s
		}
		if str != nil {
			var err error
			switch fieldType {
			case TypeBitField:
				_, err = str.Int8()
			case TypeSmallIntegerField:
				_, err = str.Int16()
			case TypeIntegerField:
				_, err = str.Int32()
			case TypeBigIntegerField:
				_, err = str.Int64()
			case TypePositiveBitField:
				_, err = str.Uint8()
			case TypePositiveSmallIntegerField:
				_, err = str.Uint16()
			case TypePositiveIntegerField:
				_, err = str.Uint32()
			case TypePositiveBigIntegerField:
				_, err = str.Uint64()
			}
			if err != nil {
				tErr = err
				goto end
			}
			if fieldType&IsPositiveIntegerField > 0 {
				v, _ := str.Uint64()
				value = v
			} else {
				v, _ := str.Int64()
				value = v
			}
		}
	case fieldType == TypeFloatField || fieldType == TypeDecimalField:
		if str == nil {
			switch v := val.(type) {
			case float64:
				value = v
			default:
				s := StrTo(ToStr(v))
				str = &s
			}
		}
		if str != nil {
			v, err := str.Float64()
			if err != nil {
				tErr = err
				goto end
			}
			value = v
		}
	case fieldType&IsRelField > 0:
		fi = fi.relModelInfo.fields.pk
		fieldType = fi.fieldType
		goto setValue
	}

end:
	if tErr != nil {
		err := fmt.Errorf("convert to `%s` failed, field: %s err: %s", fi.addrValue.Type(), fi.fullName, tErr)
		return nil, err
	}

	return value, nil

}

// set one value to struct column field.
func (d *dbBase) setFieldValue(fi *fieldInfo, value interface{}, field reflect.Value) (interface{}, error) {

	fieldType := fi.fieldType
	isNative := fi.isFielder == false

setValue:
	switch {
	case fieldType == TypeBooleanField:
		if isNative {
			if nb, ok := field.Interface().(sql.NullBool); ok {
				if value == nil {
					nb.Valid = false
				} else {
					nb.Bool = value.(bool)
					nb.Valid = true
				}
				field.Set(reflect.ValueOf(nb))
			} else if field.Kind() == reflect.Ptr {
				if value != nil {
					v := value.(bool)
					field.Set(reflect.ValueOf(&v))
				}
			} else {
				if value == nil {
					value = false
				}
				field.SetBool(value.(bool))
			}
		}
	case fieldType == TypeCharField || fieldType == TypeTextField || fieldType == TypeJSONField || fieldType == TypeJsonbField:
		if isNative {
			if ns, ok := field.Interface().(sql.NullString); ok {
				if value == nil {
					ns.Valid = false
				} else {
					ns.String = value.(string)
					ns.Valid = true
				}
				field.Set(reflect.ValueOf(ns))
			} else if field.Kind() == reflect.Ptr {
				if value != nil {
					v := value.(string)
					field.Set(reflect.ValueOf(&v))
				}
			} else {
				if value == nil {
					value = ""
				}
				field.SetString(value.(string))
			}
		}
	case fieldType == TypeTimeField || fieldType == TypeDateField || fieldType == TypeDateTimeField:
		if isNative {
			if value == nil {
				value = time.Time{}
			} else if field.Kind() == reflect.Ptr {
				if value != nil {
					v := value.(time.Time)
					field.Set(reflect.ValueOf(&v))
				}
			} else {
				field.Set(reflect.ValueOf(value))
			}
		}
	case fieldType == TypePositiveBitField && field.Kind() == reflect.Ptr:
		if value != nil {
			v := uint8(value.(uint64))
			field.Set(reflect.ValueOf(&v))
		}
	case fieldType == TypePositiveSmallIntegerField && field.Kind() == reflect.Ptr:
		if value != nil {
			v := uint16(value.(uint64))
			field.Set(reflect.ValueOf(&v))
		}
	case fieldType == TypePositiveIntegerField && field.Kind() == reflect.Ptr:
		if value != nil {
			if field.Type() == reflect.TypeOf(new(uint)) {
				v := uint(value.(uint64))
				field.Set(reflect.ValueOf(&v))
			} else {
				v := uint32(value.(uint64))
				field.Set(reflect.ValueOf(&v))
			}
		}
	case fieldType == TypePositiveBigIntegerField && field.Kind() == reflect.Ptr:
		if value != nil {
			v := value.(uint64)
			field.Set(reflect.ValueOf(&v))
		}
	case fieldType == TypeBitField && field.Kind() == reflect.Ptr:
		if value != nil {
			v := int8(value.(int64))
			field.Set(reflect.ValueOf(&v))
		}
	case fieldType == TypeSmallIntegerField && field.Kind() == reflect.Ptr:
		if value != nil {
			v := int16(value.(int64))
			field.Set(reflect.ValueOf(&v))
		}
	case fieldType == TypeIntegerField && field.Kind() == reflect.Ptr:
		if value != nil {
			if field.Type() == reflect.TypeOf(new(int)) {
				v := int(value.(int64))
				field.Set(reflect.ValueOf(&v))
			} else {
				v := int32(value.(int64))
				field.Set(reflect.ValueOf(&v))
			}
		}
	case fieldType == TypeBigIntegerField && field.Kind() == reflect.Ptr:
		if value != nil {
			v := value.(int64)
			field.Set(reflect.ValueOf(&v))
		}
	case fieldType&IsIntegerField > 0:
		if fieldType&IsPositiveIntegerField > 0 {
			if isNative {
				if value == nil {
					value = uint64(0)
				}
				field.SetUint(value.(uint64))
			}
		} else {
			if isNative {
				if ni, ok := field.Interface().(sql.NullInt64); ok {
					if value == nil {
						ni.Valid = false
					} else {
						ni.Int64 = value.(int64)
						ni.Valid = true
					}
					field.Set(reflect.ValueOf(ni))
				} else {
					if value == nil {
						value = int64(0)
					}
					field.SetInt(value.(int64))
				}
			}
		}
	case fieldType == TypeFloatField || fieldType == TypeDecimalField:
		if isNative {
			if nf, ok := field.Interface().(sql.NullFloat64); ok {
				if value == nil {
					nf.Valid = false
				} else {
					nf.Float64 = value.(float64)
					nf.Valid = true
				}
				field.Set(reflect.ValueOf(nf))
			} else if field.Kind() == reflect.Ptr {
				if value != nil {
					if field.Type() == reflect.TypeOf(new(float32)) {
						v := float32(value.(float64))
						field.Set(reflect.ValueOf(&v))
					} else {
						v := value.(float64)
						field.Set(reflect.ValueOf(&v))
					}
				}
			} else {

				if value == nil {
					value = float64(0)
				}
				field.SetFloat(value.(float64))
			}
		}
	case fieldType&IsRelField > 0:
		if value != nil {
			fieldType = fi.relModelInfo.fields.pk.fieldType
			mf := reflect.New(fi.relModelInfo.addrField.Elem().Type())
			field.Set(mf)
			f := mf.Elem().FieldByIndex(fi.relModelInfo.fields.pk.fieldIndex)
			field = f
			goto setValue
		}
	}

	if isNative == false {
		fd := field.Addr().Interface().(Fielder)
		err := fd.SetRaw(value)
		if err != nil {
			err = fmt.Errorf("converted value `%v` set to Fielder `%s` failed, err: %s", value, fi.fullName, err)
			return nil, err
		}
	}

	return value, nil
}

// query sql, read values , save to *[]ParamList.
func (d *dbBase) ReadValues(q dbQuerier, qs *querySet, mi *modelInfo, cond *Condition, exprs []string, container interface{}, tz *time.Location) (int64, error) {

	var (
		maps  []Params
		lists []ParamsList
		list  ParamsList
	)

	typ := 0
	switch v := container.(type) {
	case *[]Params:
		d := *v
		if len(d) == 0 {
			maps = d
		}
		typ = 1
	case *[]ParamsList:
		d := *v
		if len(d) == 0 {
			lists = d
		}
		typ = 2
	case *ParamsList:
		d := *v
		if len(d) == 0 {
			list = d
		}
		typ = 3
	default:
		panic(fmt.Errorf("unsupport read values type `%T`", container))
	}

	tables := newDbTables(mi, d.ins)

	var (
		cols  []string
		infos []*fieldInfo
	)

	hasExprs := len(exprs) > 0

	Q := d.ins.TableQuote()

	if hasExprs {
		cols = make([]string, 0, len(exprs))
		infos = make([]*fieldInfo, 0, len(exprs))
		for _, ex := range exprs {
			index, name, fi, suc := tables.parseExprs(mi, strings.Split(ex, ExprSep))
			if suc == false {
				panic(fmt.Errorf("unknown field/column name `%s`", ex))
			}
			cols = append(cols, fmt.Sprintf("%s.%s%s%s %s%s%s", index, Q, fi.column, Q, Q, name, Q))
			infos = append(infos, fi)
		}
	} else {
		cols = make([]string, 0, len(mi.fields.dbcols))
		infos = make([]*fieldInfo, 0, len(exprs))
		for _, fi := range mi.fields.fieldsDB {
			cols = append(cols, fmt.Sprintf("T0.%s%s%s %s%s%s", Q, fi.column, Q, Q, fi.name, Q))
			infos = append(infos, fi)
		}
	}

	where, args := tables.getCondSQL(cond, false, tz)
	groupBy := tables.getGroupSQL(qs.groups)
	orderBy := tables.getOrderSQL(qs.orders)
	limit := tables.getLimitSQL(mi, qs.offset, qs.limit)
	join := tables.getJoinSQL()

	sels := strings.Join(cols, ", ")

	sqlSelect := "SELECT"
	if qs.distinct {
		sqlSelect += " DISTINCT"
	}
	query := fmt.Sprintf("%s %s FROM %s%s%s T0 %s%s%s%s%s", sqlSelect, sels, Q, mi.table, Q, join, where, groupBy, orderBy, limit)

	d.ins.ReplaceMarks(&query)

	rs, err := q.Query(query, args...)
	if err != nil {
		return 0, err
	}
	refs := make([]interface{}, len(cols))
	for i := range refs {
		var ref interface{}
		refs[i] = &ref
	}

	defer rs.Close()

	var (
		cnt     int64
		columns []string
	)
	for rs.Next() {
		if cnt == 0 {
			cols, err := rs.Columns()
			if err != nil {
				return 0, err
			}
			columns = cols
		}

		if err := rs.Scan(refs...); err != nil {
			return 0, err
		}

		switch typ {
		case 1:
			params := make(Params, len(cols))
			for i, ref := range refs {
				fi := infos[i]

				val := reflect.Indirect(reflect.ValueOf(ref)).Interface()

				value, err := d.convertValueFromDB(fi, val, tz)
				if err != nil {
					panic(fmt.Errorf("db value convert failed `%v` %s", val, err.Error()))
				}

				params[columns[i]] = value
			}
			maps = append(maps, params)
		case 2:
			params := make(ParamsList, 0, len(cols))
			for i, ref := range refs {
				fi := infos[i]

				val := reflect.Indirect(reflect.ValueOf(ref)).Interface()

				value, err := d.convertValueFromDB(fi, val, tz)
				if err != nil {
					panic(fmt.Errorf("db value convert failed `%v` %s", val, err.Error()))
				}

				params = append(params, value)
			}
			lists = append(lists, params)
		case 3:
			for i, ref := range refs {
				fi := infos[i]

				val := reflect.Indirect(reflect.ValueOf(ref)).Interface()

				value, err := d.convertValueFromDB(fi, val, tz)
				if err != nil {
					panic(fmt.Errorf("db value convert failed `%v` %s", val, err.Error()))
				}

				list = append(list, value)
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

func (d *dbBase) RowsTo(dbQuerier, *querySet, *modelInfo, *Condition, interface{}, string, string, *time.Location) (int64, error) {
	return 0, nil
}

// flag of update joined record.
func (d *dbBase) SupportUpdateJoin() bool {
	return true
}

func (d *dbBase) MaxLimit() uint64 {
	return 18446744073709551615
}

// return quote.
func (d *dbBase) TableQuote() string {
	return "`"
}

// replace value placeholer in parametered sql string.
func (d *dbBase) ReplaceMarks(query *string) {
	// default use `?` as mark, do nothing
}

// flag of RETURNING sql.
func (d *dbBase) HasReturningID(*modelInfo, *string) bool {
	return false
}

// sync auto key
func (d *dbBase) setval(db dbQuerier, mi *modelInfo, autoFields []string) error {
	return nil
}

// convert time from db.
func (d *dbBase) TimeFromDB(t *time.Time, tz *time.Location) {
	*t = t.In(tz)
}

// convert time to db.
func (d *dbBase) TimeToDB(t *time.Time, tz *time.Location) {
	*t = t.In(tz)
}

// get database types.
func (d *dbBase) DbTypes() map[string]string {
	return nil
}

// gt all tables.
func (d *dbBase) GetTables(db dbQuerier) (map[string]bool, error) {
	tables := make(map[string]bool)
	query := d.ins.ShowTablesQuery()
	rows, err := db.Query(query)
	if err != nil {
		return tables, err
	}

	defer rows.Close()

	for rows.Next() {
		var table string
		err := rows.Scan(&table)
		if err != nil {
			return tables, err
		}
		if table != "" {
			tables[table] = true
		}
	}

	return tables, nil
}

// get all cloumns in table.
func (d *dbBase) GetColumns(db dbQuerier, table string) (map[string][3]string, error) {
	columns := make(map[string][3]string)
	query := d.ins.ShowColumnsQuery(table)
	rows, err := db.Query(query)
	if err != nil {
		return columns, err
	}

	defer rows.Close()

	for rows.Next() {
		var (
			name string
			typ  string
			null string
		)
		err := rows.Scan(&name, &typ, &null)
		if err != nil {
			return columns, err
		}
		columns[name] = [3]string{name, typ, null}
	}

	return columns, nil
}

// not implement.
func (d *dbBase) OperatorSQL(operator string) string {
	panic(ErrNotImplement)
}

// not implement.
func (d *dbBase) ShowTablesQuery() string {
	panic(ErrNotImplement)
}

// not implement.
func (d *dbBase) ShowColumnsQuery(table string) string {
	panic(ErrNotImplement)
}

// not implement.
func (d *dbBase) IndexExists(dbQuerier, string, string) bool {
	panic(ErrNotImplement)
}
