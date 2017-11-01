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

package migration

// Table store the tablename and Column
type Table struct {
	TableName string
	Columns   []*Column
}

// Create return the create sql
func (t *Table) Create() string {
	return ""
}

// Drop return the drop sql
func (t *Table) Drop() string {
	return ""
}

// Column define the columns name type and Default
type Column struct {
	Name    string
	Type    string
	Default interface{}
}

// Create return create sql with the provided tbname and columns
func Create(tbname string, columns ...Column) string {
	return ""
}

// Drop return the drop sql with the provided tbname and columns
func Drop(tbname string, columns ...Column) string {
	return ""
}

// TableDDL is still in think
func TableDDL(tbname string, columns ...Column) string {
	return ""
}
