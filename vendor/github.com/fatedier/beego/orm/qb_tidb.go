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
	"strconv"
	"strings"
)

// TiDBQueryBuilder is the SQL build
type TiDBQueryBuilder struct {
	Tokens []string
}

// Select will join the fields
func (qb *TiDBQueryBuilder) Select(fields ...string) QueryBuilder {
	qb.Tokens = append(qb.Tokens, "SELECT", strings.Join(fields, CommaSpace))
	return qb
}

// ForUpdate add the FOR UPDATE clause
func (qb *TiDBQueryBuilder) ForUpdate() QueryBuilder {
	qb.Tokens = append(qb.Tokens, "FOR UPDATE")
	return qb
}

// From join the tables
func (qb *TiDBQueryBuilder) From(tables ...string) QueryBuilder {
	qb.Tokens = append(qb.Tokens, "FROM", strings.Join(tables, CommaSpace))
	return qb
}

// InnerJoin INNER JOIN the table
func (qb *TiDBQueryBuilder) InnerJoin(table string) QueryBuilder {
	qb.Tokens = append(qb.Tokens, "INNER JOIN", table)
	return qb
}

// LeftJoin LEFT JOIN the table
func (qb *TiDBQueryBuilder) LeftJoin(table string) QueryBuilder {
	qb.Tokens = append(qb.Tokens, "LEFT JOIN", table)
	return qb
}

// RightJoin RIGHT JOIN the table
func (qb *TiDBQueryBuilder) RightJoin(table string) QueryBuilder {
	qb.Tokens = append(qb.Tokens, "RIGHT JOIN", table)
	return qb
}

// On join with on cond
func (qb *TiDBQueryBuilder) On(cond string) QueryBuilder {
	qb.Tokens = append(qb.Tokens, "ON", cond)
	return qb
}

// Where join the Where cond
func (qb *TiDBQueryBuilder) Where(cond string) QueryBuilder {
	qb.Tokens = append(qb.Tokens, "WHERE", cond)
	return qb
}

// And join the and cond
func (qb *TiDBQueryBuilder) And(cond string) QueryBuilder {
	qb.Tokens = append(qb.Tokens, "AND", cond)
	return qb
}

// Or join the or cond
func (qb *TiDBQueryBuilder) Or(cond string) QueryBuilder {
	qb.Tokens = append(qb.Tokens, "OR", cond)
	return qb
}

// In join the IN (vals)
func (qb *TiDBQueryBuilder) In(vals ...string) QueryBuilder {
	qb.Tokens = append(qb.Tokens, "IN", "(", strings.Join(vals, CommaSpace), ")")
	return qb
}

// OrderBy join the Order by fields
func (qb *TiDBQueryBuilder) OrderBy(fields ...string) QueryBuilder {
	qb.Tokens = append(qb.Tokens, "ORDER BY", strings.Join(fields, CommaSpace))
	return qb
}

// Asc join the asc
func (qb *TiDBQueryBuilder) Asc() QueryBuilder {
	qb.Tokens = append(qb.Tokens, "ASC")
	return qb
}

// Desc join the desc
func (qb *TiDBQueryBuilder) Desc() QueryBuilder {
	qb.Tokens = append(qb.Tokens, "DESC")
	return qb
}

// Limit join the limit num
func (qb *TiDBQueryBuilder) Limit(limit int) QueryBuilder {
	qb.Tokens = append(qb.Tokens, "LIMIT", strconv.Itoa(limit))
	return qb
}

// Offset join the offset num
func (qb *TiDBQueryBuilder) Offset(offset int) QueryBuilder {
	qb.Tokens = append(qb.Tokens, "OFFSET", strconv.Itoa(offset))
	return qb
}

// GroupBy join the Group by fields
func (qb *TiDBQueryBuilder) GroupBy(fields ...string) QueryBuilder {
	qb.Tokens = append(qb.Tokens, "GROUP BY", strings.Join(fields, CommaSpace))
	return qb
}

// Having join the Having cond
func (qb *TiDBQueryBuilder) Having(cond string) QueryBuilder {
	qb.Tokens = append(qb.Tokens, "HAVING", cond)
	return qb
}

// Update join the update table
func (qb *TiDBQueryBuilder) Update(tables ...string) QueryBuilder {
	qb.Tokens = append(qb.Tokens, "UPDATE", strings.Join(tables, CommaSpace))
	return qb
}

// Set join the set kv
func (qb *TiDBQueryBuilder) Set(kv ...string) QueryBuilder {
	qb.Tokens = append(qb.Tokens, "SET", strings.Join(kv, CommaSpace))
	return qb
}

// Delete join the Delete tables
func (qb *TiDBQueryBuilder) Delete(tables ...string) QueryBuilder {
	qb.Tokens = append(qb.Tokens, "DELETE")
	if len(tables) != 0 {
		qb.Tokens = append(qb.Tokens, strings.Join(tables, CommaSpace))
	}
	return qb
}

// InsertInto join the insert SQL
func (qb *TiDBQueryBuilder) InsertInto(table string, fields ...string) QueryBuilder {
	qb.Tokens = append(qb.Tokens, "INSERT INTO", table)
	if len(fields) != 0 {
		fieldsStr := strings.Join(fields, CommaSpace)
		qb.Tokens = append(qb.Tokens, "(", fieldsStr, ")")
	}
	return qb
}

// Values join the Values(vals)
func (qb *TiDBQueryBuilder) Values(vals ...string) QueryBuilder {
	valsStr := strings.Join(vals, CommaSpace)
	qb.Tokens = append(qb.Tokens, "VALUES", "(", valsStr, ")")
	return qb
}

// Subquery join the sub as alias
func (qb *TiDBQueryBuilder) Subquery(sub string, alias string) string {
	return fmt.Sprintf("(%s) AS %s", sub, alias)
}

// String join all Tokens
func (qb *TiDBQueryBuilder) String() string {
	return strings.Join(qb.Tokens, " ")
}
