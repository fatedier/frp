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

// Package orm provide ORM for MySQL/PostgreSQL/sqlite
// Simple Usage
//
//	package main
//
//	import (
//		"fmt"
//		"github.com/astaxie/beego/orm"
//		_ "github.com/go-sql-driver/mysql" // import your used driver
//	)
//
//	// Model Struct
//	type User struct {
//		Id   int    `orm:"auto"`
//		Name string `orm:"size(100)"`
//	}
//
//	func init() {
//		orm.RegisterDataBase("default", "mysql", "root:root@/my_db?charset=utf8", 30)
//	}
//
//	func main() {
//		o := orm.NewOrm()
//		user := User{Name: "slene"}
//		// insert
//		id, err := o.Insert(&user)
//		// update
//		user.Name = "astaxie"
//		num, err := o.Update(&user)
//		// read one
//		u := User{Id: user.Id}
//		err = o.Read(&u)
//		// delete
//		num, err = o.Delete(&u)
//	}
//
// more docs: http://beego.me/docs/mvc/model/overview.md
package orm

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"reflect"
	"time"
)

// DebugQueries define the debug
const (
	DebugQueries = iota
)

// Define common vars
var (
	Debug            = false
	DebugLog         = NewLog(os.Stdout)
	DefaultRowsLimit = 1000
	DefaultRelsDepth = 2
	DefaultTimeLoc   = time.Local
	ErrTxHasBegan    = errors.New("<Ormer.Begin> transaction already begin")
	ErrTxDone        = errors.New("<Ormer.Commit/Rollback> transaction not begin")
	ErrMultiRows     = errors.New("<QuerySeter> return multi rows")
	ErrNoRows        = errors.New("<QuerySeter> no row found")
	ErrStmtClosed    = errors.New("<QuerySeter> stmt already closed")
	ErrArgs          = errors.New("<Ormer> args error may be empty")
	ErrNotImplement  = errors.New("have not implement")
)

// Params stores the Params
type Params map[string]interface{}

// ParamsList stores paramslist
type ParamsList []interface{}

type orm struct {
	alias *alias
	db    dbQuerier
	isTx  bool
}

var _ Ormer = new(orm)

// get model info and model reflect value
func (o *orm) getMiInd(md interface{}, needPtr bool) (mi *modelInfo, ind reflect.Value) {
	val := reflect.ValueOf(md)
	ind = reflect.Indirect(val)
	typ := ind.Type()
	if needPtr && val.Kind() != reflect.Ptr {
		panic(fmt.Errorf("<Ormer> cannot use non-ptr model struct `%s`", getFullName(typ)))
	}
	name := getFullName(typ)
	if mi, ok := modelCache.getByFullName(name); ok {
		return mi, ind
	}
	panic(fmt.Errorf("<Ormer> table: `%s` not found, maybe not RegisterModel", name))
}

// get field info from model info by given field name
func (o *orm) getFieldInfo(mi *modelInfo, name string) *fieldInfo {
	fi, ok := mi.fields.GetByAny(name)
	if !ok {
		panic(fmt.Errorf("<Ormer> cannot find field `%s` for model `%s`", name, mi.fullName))
	}
	return fi
}

// read data to model
func (o *orm) Read(md interface{}, cols ...string) error {
	mi, ind := o.getMiInd(md, true)
	err := o.alias.DbBaser.Read(o.db, mi, ind, o.alias.TZ, cols, false)
	if err != nil {
		return err
	}
	return nil
}

// read data to model, like Read(), but use "SELECT FOR UPDATE" form
func (o *orm) ReadForUpdate(md interface{}, cols ...string) error {
	mi, ind := o.getMiInd(md, true)
	err := o.alias.DbBaser.Read(o.db, mi, ind, o.alias.TZ, cols, true)
	if err != nil {
		return err
	}
	return nil
}

// Try to read a row from the database, or insert one if it doesn't exist
func (o *orm) ReadOrCreate(md interface{}, col1 string, cols ...string) (bool, int64, error) {
	cols = append([]string{col1}, cols...)
	mi, ind := o.getMiInd(md, true)
	err := o.alias.DbBaser.Read(o.db, mi, ind, o.alias.TZ, cols, false)
	if err == ErrNoRows {
		// Create
		id, err := o.Insert(md)
		return (err == nil), id, err
	}

	id, vid := int64(0), ind.FieldByIndex(mi.fields.pk.fieldIndex)
	if mi.fields.pk.fieldType&IsPositiveIntegerField > 0 {
		id = int64(vid.Uint())
	} else if mi.fields.pk.rel {
		return o.ReadOrCreate(vid.Interface(), mi.fields.pk.relModelInfo.fields.pk.name)
	} else {
		id = vid.Int()
	}

	return false, id, err
}

// insert model data to database
func (o *orm) Insert(md interface{}) (int64, error) {
	mi, ind := o.getMiInd(md, true)
	id, err := o.alias.DbBaser.Insert(o.db, mi, ind, o.alias.TZ)
	if err != nil {
		return id, err
	}

	o.setPk(mi, ind, id)

	return id, nil
}

// set auto pk field
func (o *orm) setPk(mi *modelInfo, ind reflect.Value, id int64) {
	if mi.fields.pk.auto {
		if mi.fields.pk.fieldType&IsPositiveIntegerField > 0 {
			ind.FieldByIndex(mi.fields.pk.fieldIndex).SetUint(uint64(id))
		} else {
			ind.FieldByIndex(mi.fields.pk.fieldIndex).SetInt(id)
		}
	}
}

// insert some models to database
func (o *orm) InsertMulti(bulk int, mds interface{}) (int64, error) {
	var cnt int64

	sind := reflect.Indirect(reflect.ValueOf(mds))

	switch sind.Kind() {
	case reflect.Array, reflect.Slice:
		if sind.Len() == 0 {
			return cnt, ErrArgs
		}
	default:
		return cnt, ErrArgs
	}

	if bulk <= 1 {
		for i := 0; i < sind.Len(); i++ {
			ind := reflect.Indirect(sind.Index(i))
			mi, _ := o.getMiInd(ind.Interface(), false)
			id, err := o.alias.DbBaser.Insert(o.db, mi, ind, o.alias.TZ)
			if err != nil {
				return cnt, err
			}

			o.setPk(mi, ind, id)

			cnt++
		}
	} else {
		mi, _ := o.getMiInd(sind.Index(0).Interface(), false)
		return o.alias.DbBaser.InsertMulti(o.db, mi, sind, bulk, o.alias.TZ)
	}
	return cnt, nil
}

// InsertOrUpdate data to database
func (o *orm) InsertOrUpdate(md interface{}, colConflitAndArgs ...string) (int64, error) {
	mi, ind := o.getMiInd(md, true)
	id, err := o.alias.DbBaser.InsertOrUpdate(o.db, mi, ind, o.alias, colConflitAndArgs...)
	if err != nil {
		return id, err
	}

	o.setPk(mi, ind, id)

	return id, nil
}

// update model to database.
// cols set the columns those want to update.
func (o *orm) Update(md interface{}, cols ...string) (int64, error) {
	mi, ind := o.getMiInd(md, true)
	num, err := o.alias.DbBaser.Update(o.db, mi, ind, o.alias.TZ, cols)
	if err != nil {
		return num, err
	}
	return num, nil
}

// delete model in database
// cols shows the delete conditions values read from. deafult is pk
func (o *orm) Delete(md interface{}, cols ...string) (int64, error) {
	mi, ind := o.getMiInd(md, true)
	num, err := o.alias.DbBaser.Delete(o.db, mi, ind, o.alias.TZ, cols)
	if err != nil {
		return num, err
	}
	if num > 0 {
		o.setPk(mi, ind, 0)
	}
	return num, nil
}

// create a models to models queryer
func (o *orm) QueryM2M(md interface{}, name string) QueryM2Mer {
	mi, ind := o.getMiInd(md, true)
	fi := o.getFieldInfo(mi, name)

	switch {
	case fi.fieldType == RelManyToMany:
	case fi.fieldType == RelReverseMany && fi.reverseFieldInfo.mi.isThrough:
	default:
		panic(fmt.Errorf("<Ormer.QueryM2M> model `%s` . name `%s` is not a m2m field", fi.name, mi.fullName))
	}

	return newQueryM2M(md, o, mi, fi, ind)
}

// load related models to md model.
// args are limit, offset int and order string.
//
// example:
// 	orm.LoadRelated(post,"Tags")
// 	for _,tag := range post.Tags{...}
//
// make sure the relation is defined in model struct tags.
func (o *orm) LoadRelated(md interface{}, name string, args ...interface{}) (int64, error) {
	_, fi, ind, qseter := o.queryRelated(md, name)

	qs := qseter.(*querySet)

	var relDepth int
	var limit, offset int64
	var order string
	for i, arg := range args {
		switch i {
		case 0:
			if v, ok := arg.(bool); ok {
				if v {
					relDepth = DefaultRelsDepth
				}
			} else if v, ok := arg.(int); ok {
				relDepth = v
			}
		case 1:
			limit = ToInt64(arg)
		case 2:
			offset = ToInt64(arg)
		case 3:
			order, _ = arg.(string)
		}
	}

	switch fi.fieldType {
	case RelOneToOne, RelForeignKey, RelReverseOne:
		limit = 1
		offset = 0
	}

	qs.limit = limit
	qs.offset = offset
	qs.relDepth = relDepth

	if len(order) > 0 {
		qs.orders = []string{order}
	}

	find := ind.FieldByIndex(fi.fieldIndex)

	var nums int64
	var err error
	switch fi.fieldType {
	case RelOneToOne, RelForeignKey, RelReverseOne:
		val := reflect.New(find.Type().Elem())
		container := val.Interface()
		err = qs.One(container)
		if err == nil {
			find.Set(val)
			nums = 1
		}
	default:
		nums, err = qs.All(find.Addr().Interface())
	}

	return nums, err
}

// return a QuerySeter for related models to md model.
// it can do all, update, delete in QuerySeter.
// example:
// 	qs := orm.QueryRelated(post,"Tag")
//  qs.All(&[]*Tag{})
//
func (o *orm) QueryRelated(md interface{}, name string) QuerySeter {
	// is this api needed ?
	_, _, _, qs := o.queryRelated(md, name)
	return qs
}

// get QuerySeter for related models to md model
func (o *orm) queryRelated(md interface{}, name string) (*modelInfo, *fieldInfo, reflect.Value, QuerySeter) {
	mi, ind := o.getMiInd(md, true)
	fi := o.getFieldInfo(mi, name)

	_, _, exist := getExistPk(mi, ind)
	if exist == false {
		panic(ErrMissPK)
	}

	var qs *querySet

	switch fi.fieldType {
	case RelOneToOne, RelForeignKey, RelManyToMany:
		if !fi.inModel {
			break
		}
		qs = o.getRelQs(md, mi, fi)
	case RelReverseOne, RelReverseMany:
		if !fi.inModel {
			break
		}
		qs = o.getReverseQs(md, mi, fi)
	}

	if qs == nil {
		panic(fmt.Errorf("<Ormer> name `%s` for model `%s` is not an available rel/reverse field", md, name))
	}

	return mi, fi, ind, qs
}

// get reverse relation QuerySeter
func (o *orm) getReverseQs(md interface{}, mi *modelInfo, fi *fieldInfo) *querySet {
	switch fi.fieldType {
	case RelReverseOne, RelReverseMany:
	default:
		panic(fmt.Errorf("<Ormer> name `%s` for model `%s` is not an available reverse field", fi.name, mi.fullName))
	}

	var q *querySet

	if fi.fieldType == RelReverseMany && fi.reverseFieldInfo.mi.isThrough {
		q = newQuerySet(o, fi.relModelInfo).(*querySet)
		q.cond = NewCondition().And(fi.reverseFieldInfoM2M.column+ExprSep+fi.reverseFieldInfo.column, md)
	} else {
		q = newQuerySet(o, fi.reverseFieldInfo.mi).(*querySet)
		q.cond = NewCondition().And(fi.reverseFieldInfo.column, md)
	}

	return q
}

// get relation QuerySeter
func (o *orm) getRelQs(md interface{}, mi *modelInfo, fi *fieldInfo) *querySet {
	switch fi.fieldType {
	case RelOneToOne, RelForeignKey, RelManyToMany:
	default:
		panic(fmt.Errorf("<Ormer> name `%s` for model `%s` is not an available rel field", fi.name, mi.fullName))
	}

	q := newQuerySet(o, fi.relModelInfo).(*querySet)
	q.cond = NewCondition()

	if fi.fieldType == RelManyToMany {
		q.cond = q.cond.And(fi.reverseFieldInfoM2M.column+ExprSep+fi.reverseFieldInfo.column, md)
	} else {
		q.cond = q.cond.And(fi.reverseFieldInfo.column, md)
	}

	return q
}

// return a QuerySeter for table operations.
// table name can be string or struct.
// e.g. QueryTable("user"), QueryTable(&user{}) or QueryTable((*User)(nil)),
func (o *orm) QueryTable(ptrStructOrTableName interface{}) (qs QuerySeter) {
	name := ""
	if table, ok := ptrStructOrTableName.(string); ok {
		name = snakeString(table)
		if mi, ok := modelCache.get(name); ok {
			qs = newQuerySet(o, mi)
		}
	} else {
		name = getFullName(indirectType(reflect.TypeOf(ptrStructOrTableName)))
		if mi, ok := modelCache.getByFullName(name); ok {
			qs = newQuerySet(o, mi)
		}
	}
	if qs == nil {
		panic(fmt.Errorf("<Ormer.QueryTable> table name: `%s` not exists", name))
	}
	return
}

// switch to another registered database driver by given name.
func (o *orm) Using(name string) error {
	if o.isTx {
		panic(fmt.Errorf("<Ormer.Using> transaction has been start, cannot change db"))
	}
	if al, ok := dataBaseCache.get(name); ok {
		o.alias = al
		if Debug {
			o.db = newDbQueryLog(al, al.DB)
		} else {
			o.db = al.DB
		}
	} else {
		return fmt.Errorf("<Ormer.Using> unknown db alias name `%s`", name)
	}
	return nil
}

// begin transaction
func (o *orm) Begin() error {
	if o.isTx {
		return ErrTxHasBegan
	}
	var tx *sql.Tx
	tx, err := o.db.(txer).Begin()
	if err != nil {
		return err
	}
	o.isTx = true
	if Debug {
		o.db.(*dbQueryLog).SetDB(tx)
	} else {
		o.db = tx
	}
	return nil
}

// commit transaction
func (o *orm) Commit() error {
	if o.isTx == false {
		return ErrTxDone
	}
	err := o.db.(txEnder).Commit()
	if err == nil {
		o.isTx = false
		o.Using(o.alias.Name)
	} else if err == sql.ErrTxDone {
		return ErrTxDone
	}
	return err
}

// rollback transaction
func (o *orm) Rollback() error {
	if o.isTx == false {
		return ErrTxDone
	}
	err := o.db.(txEnder).Rollback()
	if err == nil {
		o.isTx = false
		o.Using(o.alias.Name)
	} else if err == sql.ErrTxDone {
		return ErrTxDone
	}
	return err
}

// return a raw query seter for raw sql string.
func (o *orm) Raw(query string, args ...interface{}) RawSeter {
	return newRawSet(o, query, args)
}

// return current using database Driver
func (o *orm) Driver() Driver {
	return driver(o.alias.Name)
}

// NewOrm create new orm
func NewOrm() Ormer {
	BootStrap() // execute only once

	o := new(orm)
	err := o.Using("default")
	if err != nil {
		panic(err)
	}
	return o
}

// NewOrmWithDB create a new ormer object with specify *sql.DB for query
func NewOrmWithDB(driverName, aliasName string, db *sql.DB) (Ormer, error) {
	var al *alias

	if dr, ok := drivers[driverName]; ok {
		al = new(alias)
		al.DbBaser = dbBasers[dr]
		al.Driver = dr
	} else {
		return nil, fmt.Errorf("driver name `%s` have not registered", driverName)
	}

	al.Name = aliasName
	al.DriverName = driverName

	o := new(orm)
	o.alias = al

	if Debug {
		o.db = newDbQueryLog(o.alias, db)
	} else {
		o.db = db
	}

	return o, nil
}
