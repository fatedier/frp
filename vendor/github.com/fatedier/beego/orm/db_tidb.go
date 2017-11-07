// Copyright 2015 TiDB Author. All Rights Reserved.
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

// mysql dbBaser implementation.
type dbBaseTidb struct {
	dbBase
}

var _ dbBaser = new(dbBaseTidb)

// get mysql operator.
func (d *dbBaseTidb) OperatorSQL(operator string) string {
	return mysqlOperators[operator]
}

// get mysql table field types.
func (d *dbBaseTidb) DbTypes() map[string]string {
	return mysqlTypes
}

// show table sql for mysql.
func (d *dbBaseTidb) ShowTablesQuery() string {
	return "SELECT table_name FROM information_schema.tables WHERE table_type = 'BASE TABLE' AND table_schema = DATABASE()"
}

// show columns sql of table for mysql.
func (d *dbBaseTidb) ShowColumnsQuery(table string) string {
	return fmt.Sprintf("SELECT COLUMN_NAME, COLUMN_TYPE, IS_NULLABLE FROM information_schema.columns "+
		"WHERE table_schema = DATABASE() AND table_name = '%s'", table)
}

// execute sql to check index exist.
func (d *dbBaseTidb) IndexExists(db dbQuerier, table string, name string) bool {
	row := db.QueryRow("SELECT count(*) FROM information_schema.statistics "+
		"WHERE table_schema = DATABASE() AND table_name = ? AND index_name = ?", table, name)
	var cnt int
	row.Scan(&cnt)
	return cnt > 0
}

// create new mysql dbBaser.
func newdbBaseTidb() dbBaser {
	b := new(dbBaseTidb)
	b.ins = b
	return b
}
