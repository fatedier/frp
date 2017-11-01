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
)

type colValue struct {
	value int64
	opt   operator
}

type operator int

// define Col operations
const (
	ColAdd operator = iota
	ColMinus
	ColMultiply
	ColExcept
)

// ColValue do the field raw changes. e.g Nums = Nums + 10. usage:
// 	Params{
// 		"Nums": ColValue(Col_Add, 10),
// 	}
func ColValue(opt operator, value interface{}) interface{} {
	switch opt {
	case ColAdd, ColMinus, ColMultiply, ColExcept:
	default:
		panic(fmt.Errorf("orm.ColValue wrong operator"))
	}
	v, err := StrTo(ToStr(value)).Int64()
	if err != nil {
		panic(fmt.Errorf("orm.ColValue doesn't support non string/numeric type, %s", err))
	}
	var val colValue
	val.value = v
	val.opt = opt
	return val
}

// real query struct
type querySet struct {
	mi       *modelInfo
	cond     *Condition
	related  []string
	relDepth int
	limit    int64
	offset   int64
	groups   []string
	orders   []string
	distinct bool
	orm      *orm
}

var _ QuerySeter = new(querySet)

// add condition expression to QuerySeter.
func (o querySet) Filter(expr string, args ...interface{}) QuerySeter {
	if o.cond == nil {
		o.cond = NewCondition()
	}
	o.cond = o.cond.And(expr, args...)
	return &o
}

// add NOT condition to querySeter.
func (o querySet) Exclude(expr string, args ...interface{}) QuerySeter {
	if o.cond == nil {
		o.cond = NewCondition()
	}
	o.cond = o.cond.AndNot(expr, args...)
	return &o
}

// set offset number
func (o *querySet) setOffset(num interface{}) {
	o.offset = ToInt64(num)
}

// add LIMIT value.
// args[0] means offset, e.g. LIMIT num,offset.
func (o querySet) Limit(limit interface{}, args ...interface{}) QuerySeter {
	o.limit = ToInt64(limit)
	if len(args) > 0 {
		o.setOffset(args[0])
	}
	return &o
}

// add OFFSET value
func (o querySet) Offset(offset interface{}) QuerySeter {
	o.setOffset(offset)
	return &o
}

// add GROUP expression
func (o querySet) GroupBy(exprs ...string) QuerySeter {
	o.groups = exprs
	return &o
}

// add ORDER expression.
// "column" means ASC, "-column" means DESC.
func (o querySet) OrderBy(exprs ...string) QuerySeter {
	o.orders = exprs
	return &o
}

// add DISTINCT to SELECT
func (o querySet) Distinct() QuerySeter {
	o.distinct = true
	return &o
}

// set relation model to query together.
// it will query relation models and assign to parent model.
func (o querySet) RelatedSel(params ...interface{}) QuerySeter {
	if len(params) == 0 {
		o.relDepth = DefaultRelsDepth
	} else {
		for _, p := range params {
			switch val := p.(type) {
			case string:
				o.related = append(o.related, val)
			case int:
				o.relDepth = val
			default:
				panic(fmt.Errorf("<QuerySeter.RelatedSel> wrong param kind: %v", val))
			}
		}
	}
	return &o
}

// set condition to QuerySeter.
func (o querySet) SetCond(cond *Condition) QuerySeter {
	o.cond = cond
	return &o
}

// get condition from QuerySeter
func (o querySet) GetCond() *Condition {
	return o.cond
}

// return QuerySeter execution result number
func (o *querySet) Count() (int64, error) {
	return o.orm.alias.DbBaser.Count(o.orm.db, o, o.mi, o.cond, o.orm.alias.TZ)
}

// check result empty or not after QuerySeter executed
func (o *querySet) Exist() bool {
	cnt, _ := o.orm.alias.DbBaser.Count(o.orm.db, o, o.mi, o.cond, o.orm.alias.TZ)
	return cnt > 0
}

// execute update with parameters
func (o *querySet) Update(values Params) (int64, error) {
	return o.orm.alias.DbBaser.UpdateBatch(o.orm.db, o, o.mi, o.cond, values, o.orm.alias.TZ)
}

// execute delete
func (o *querySet) Delete() (int64, error) {
	return o.orm.alias.DbBaser.DeleteBatch(o.orm.db, o, o.mi, o.cond, o.orm.alias.TZ)
}

// return a insert queryer.
// it can be used in times.
// example:
// 	i,err := sq.PrepareInsert()
// 	i.Add(&user1{},&user2{})
func (o *querySet) PrepareInsert() (Inserter, error) {
	return newInsertSet(o.orm, o.mi)
}

// query all data and map to containers.
// cols means the columns when querying.
func (o *querySet) All(container interface{}, cols ...string) (int64, error) {
	return o.orm.alias.DbBaser.ReadBatch(o.orm.db, o, o.mi, o.cond, container, o.orm.alias.TZ, cols)
}

// query one row data and map to containers.
// cols means the columns when querying.
func (o *querySet) One(container interface{}, cols ...string) error {
	o.limit = 1
	num, err := o.orm.alias.DbBaser.ReadBatch(o.orm.db, o, o.mi, o.cond, container, o.orm.alias.TZ, cols)
	if err != nil {
		return err
	}
	if num == 0 {
		return ErrNoRows
	}

	if num > 1 {
		return ErrMultiRows
	}
	return nil
}

// query all data and map to []map[string]interface.
// expres means condition expression.
// it converts data to []map[column]value.
func (o *querySet) Values(results *[]Params, exprs ...string) (int64, error) {
	return o.orm.alias.DbBaser.ReadValues(o.orm.db, o, o.mi, o.cond, exprs, results, o.orm.alias.TZ)
}

// query all data and map to [][]interface
// it converts data to [][column_index]value
func (o *querySet) ValuesList(results *[]ParamsList, exprs ...string) (int64, error) {
	return o.orm.alias.DbBaser.ReadValues(o.orm.db, o, o.mi, o.cond, exprs, results, o.orm.alias.TZ)
}

// query all data and map to []interface.
// it's designed for one row record set, auto change to []value, not [][column]value.
func (o *querySet) ValuesFlat(result *ParamsList, expr string) (int64, error) {
	return o.orm.alias.DbBaser.ReadValues(o.orm.db, o, o.mi, o.cond, []string{expr}, result, o.orm.alias.TZ)
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
func (o *querySet) RowsToMap(result *Params, keyCol, valueCol string) (int64, error) {
	panic(ErrNotImplement)
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
func (o *querySet) RowsToStruct(ptrStruct interface{}, keyCol, valueCol string) (int64, error) {
	panic(ErrNotImplement)
}

// create new QuerySeter.
func newQuerySet(orm *orm, mi *modelInfo) QuerySeter {
	o := new(querySet)
	o.mi = mi
	o.orm = orm
	return o
}
