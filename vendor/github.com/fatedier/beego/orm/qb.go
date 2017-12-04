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

import "errors"

// QueryBuilder is the Query builder interface
type QueryBuilder interface {
	Select(fields ...string) QueryBuilder
	ForUpdate() QueryBuilder
	From(tables ...string) QueryBuilder
	InnerJoin(table string) QueryBuilder
	LeftJoin(table string) QueryBuilder
	RightJoin(table string) QueryBuilder
	On(cond string) QueryBuilder
	Where(cond string) QueryBuilder
	And(cond string) QueryBuilder
	Or(cond string) QueryBuilder
	In(vals ...string) QueryBuilder
	OrderBy(fields ...string) QueryBuilder
	Asc() QueryBuilder
	Desc() QueryBuilder
	Limit(limit int) QueryBuilder
	Offset(offset int) QueryBuilder
	GroupBy(fields ...string) QueryBuilder
	Having(cond string) QueryBuilder
	Update(tables ...string) QueryBuilder
	Set(kv ...string) QueryBuilder
	Delete(tables ...string) QueryBuilder
	InsertInto(table string, fields ...string) QueryBuilder
	Values(vals ...string) QueryBuilder
	Subquery(sub string, alias string) string
	String() string
}

// NewQueryBuilder return the QueryBuilder
func NewQueryBuilder(driver string) (qb QueryBuilder, err error) {
	if driver == "mysql" {
		qb = new(MySQLQueryBuilder)
	} else if driver == "tidb" {
		qb = new(TiDBQueryBuilder)
	} else if driver == "postgres" {
		err = errors.New("postgres query builder is not supported yet")
	} else if driver == "sqlite" {
		err = errors.New("sqlite query builder is not supported yet")
	} else {
		err = errors.New("unknown driver for query builder")
	}
	return
}
