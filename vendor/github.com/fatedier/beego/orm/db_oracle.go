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
	"strings"
)

// oracle operators.
var oracleOperators = map[string]string{
	"exact":       "= ?",
	"gt":          "> ?",
	"gte":         ">= ?",
	"lt":          "< ?",
	"lte":         "<= ?",
	"//iendswith": "LIKE ?",
}

// oracle column field types.
var oracleTypes = map[string]string{
	"pk":              "NOT NULL PRIMARY KEY",
	"bool":            "bool",
	"string":          "VARCHAR2(%d)",
	"string-text":     "VARCHAR2(%d)",
	"time.Time-date":  "DATE",
	"time.Time":       "TIMESTAMP",
	"int8":            "INTEGER",
	"int16":           "INTEGER",
	"int32":           "INTEGER",
	"int64":           "INTEGER",
	"uint8":           "INTEGER",
	"uint16":          "INTEGER",
	"uint32":          "INTEGER",
	"uint64":          "INTEGER",
	"float64":         "NUMBER",
	"float64-decimal": "NUMBER(%d, %d)",
}

// oracle dbBaser
type dbBaseOracle struct {
	dbBase
}

var _ dbBaser = new(dbBaseOracle)

// create oracle dbBaser.
func newdbBaseOracle() dbBaser {
	b := new(dbBaseOracle)
	b.ins = b
	return b
}

// OperatorSQL get oracle operator.
func (d *dbBaseOracle) OperatorSQL(operator string) string {
	return oracleOperators[operator]
}

// DbTypes get oracle table field types.
func (d *dbBaseOracle) DbTypes() map[string]string {
	return oracleTypes
}

//ShowTablesQuery show all the tables in database
func (d *dbBaseOracle) ShowTablesQuery() string {
	return "SELECT TABLE_NAME FROM USER_TABLES"
}

// Oracle
func (d *dbBaseOracle) ShowColumnsQuery(table string) string {
	return fmt.Sprintf("SELECT COLUMN_NAME FROM ALL_TAB_COLUMNS "+
		"WHERE TABLE_NAME ='%s'", strings.ToUpper(table))
}

// check index is exist
func (d *dbBaseOracle) IndexExists(db dbQuerier, table string, name string) bool {
	row := db.QueryRow("SELECT COUNT(*) FROM USER_IND_COLUMNS, USER_INDEXES "+
		"WHERE USER_IND_COLUMNS.INDEX_NAME = USER_INDEXES.INDEX_NAME "+
		"AND  USER_IND_COLUMNS.TABLE_NAME = ? AND USER_IND_COLUMNS.INDEX_NAME = ?", strings.ToUpper(table), strings.ToUpper(name))

	var cnt int
	row.Scan(&cnt)
	return cnt > 0
}
