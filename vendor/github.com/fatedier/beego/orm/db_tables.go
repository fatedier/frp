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
	"time"
)

// table info struct.
type dbTable struct {
	id    int
	index string
	name  string
	names []string
	sel   bool
	inner bool
	mi    *modelInfo
	fi    *fieldInfo
	jtl   *dbTable
}

// tables collection struct, contains some tables.
type dbTables struct {
	tablesM map[string]*dbTable
	tables  []*dbTable
	mi      *modelInfo
	base    dbBaser
	skipEnd bool
}

// set table info to collection.
// if not exist, create new.
func (t *dbTables) set(names []string, mi *modelInfo, fi *fieldInfo, inner bool) *dbTable {
	name := strings.Join(names, ExprSep)
	if j, ok := t.tablesM[name]; ok {
		j.name = name
		j.mi = mi
		j.fi = fi
		j.inner = inner
	} else {
		i := len(t.tables) + 1
		jt := &dbTable{i, fmt.Sprintf("T%d", i), name, names, false, inner, mi, fi, nil}
		t.tablesM[name] = jt
		t.tables = append(t.tables, jt)
	}
	return t.tablesM[name]
}

// add table info to collection.
func (t *dbTables) add(names []string, mi *modelInfo, fi *fieldInfo, inner bool) (*dbTable, bool) {
	name := strings.Join(names, ExprSep)
	if _, ok := t.tablesM[name]; ok == false {
		i := len(t.tables) + 1
		jt := &dbTable{i, fmt.Sprintf("T%d", i), name, names, false, inner, mi, fi, nil}
		t.tablesM[name] = jt
		t.tables = append(t.tables, jt)
		return jt, true
	}
	return t.tablesM[name], false
}

// get table info in collection.
func (t *dbTables) get(name string) (*dbTable, bool) {
	j, ok := t.tablesM[name]
	return j, ok
}

// get related fields info in recursive depth loop.
// loop once, depth decreases one.
func (t *dbTables) loopDepth(depth int, prefix string, fi *fieldInfo, related []string) []string {
	if depth < 0 || fi.fieldType == RelManyToMany {
		return related
	}

	if prefix == "" {
		prefix = fi.name
	} else {
		prefix = prefix + ExprSep + fi.name
	}
	related = append(related, prefix)

	depth--
	for _, fi := range fi.relModelInfo.fields.fieldsRel {
		related = t.loopDepth(depth, prefix, fi, related)
	}

	return related
}

// parse related fields.
func (t *dbTables) parseRelated(rels []string, depth int) {

	relsNum := len(rels)
	related := make([]string, relsNum)
	copy(related, rels)

	relDepth := depth

	if relsNum != 0 {
		relDepth = 0
	}

	relDepth--
	for _, fi := range t.mi.fields.fieldsRel {
		related = t.loopDepth(relDepth, "", fi, related)
	}

	for i, s := range related {
		var (
			exs    = strings.Split(s, ExprSep)
			names  = make([]string, 0, len(exs))
			mmi    = t.mi
			cancel = true
			jtl    *dbTable
		)

		inner := true

		for _, ex := range exs {
			if fi, ok := mmi.fields.GetByAny(ex); ok && fi.rel && fi.fieldType != RelManyToMany {
				names = append(names, fi.name)
				mmi = fi.relModelInfo

				if fi.null || t.skipEnd {
					inner = false
				}

				jt := t.set(names, mmi, fi, inner)
				jt.jtl = jtl

				if fi.reverse {
					cancel = false
				}

				if cancel {
					jt.sel = depth > 0

					if i < relsNum {
						jt.sel = true
					}
				}

				jtl = jt

			} else {
				panic(fmt.Errorf("unknown model/table name `%s`", ex))
			}
		}
	}
}

// generate join string.
func (t *dbTables) getJoinSQL() (join string) {
	Q := t.base.TableQuote()

	for _, jt := range t.tables {
		if jt.inner {
			join += "INNER JOIN "
		} else {
			join += "LEFT OUTER JOIN "
		}
		var (
			table  string
			t1, t2 string
			c1, c2 string
		)
		t1 = "T0"
		if jt.jtl != nil {
			t1 = jt.jtl.index
		}
		t2 = jt.index
		table = jt.mi.table

		switch {
		case jt.fi.fieldType == RelManyToMany || jt.fi.fieldType == RelReverseMany || jt.fi.reverse && jt.fi.reverseFieldInfo.fieldType == RelManyToMany:
			c1 = jt.fi.mi.fields.pk.column
			for _, ffi := range jt.mi.fields.fieldsRel {
				if jt.fi.mi == ffi.relModelInfo {
					c2 = ffi.column
					break
				}
			}
		default:
			c1 = jt.fi.column
			c2 = jt.fi.relModelInfo.fields.pk.column

			if jt.fi.reverse {
				c1 = jt.mi.fields.pk.column
				c2 = jt.fi.reverseFieldInfo.column
			}
		}

		join += fmt.Sprintf("%s%s%s %s ON %s.%s%s%s = %s.%s%s%s ", Q, table, Q, t2,
			t2, Q, c2, Q, t1, Q, c1, Q)
	}
	return
}

// parse orm model struct field tag expression.
func (t *dbTables) parseExprs(mi *modelInfo, exprs []string) (index, name string, info *fieldInfo, success bool) {
	var (
		jtl *dbTable
		fi  *fieldInfo
		fiN *fieldInfo
		mmi = mi
	)

	num := len(exprs) - 1
	var names []string

	inner := true

loopFor:
	for i, ex := range exprs {

		var ok, okN bool

		if fiN != nil {
			fi = fiN
			ok = true
			fiN = nil
		}

		if i == 0 {
			fi, ok = mmi.fields.GetByAny(ex)
		}

		_ = okN

		if ok {

			isRel := fi.rel || fi.reverse

			names = append(names, fi.name)

			switch {
			case fi.rel:
				mmi = fi.relModelInfo
				if fi.fieldType == RelManyToMany {
					mmi = fi.relThroughModelInfo
				}
			case fi.reverse:
				mmi = fi.reverseFieldInfo.mi
			}

			if i < num {
				fiN, okN = mmi.fields.GetByAny(exprs[i+1])
			}

			if isRel && (fi.mi.isThrough == false || num != i) {
				if fi.null || t.skipEnd {
					inner = false
				}

				if t.skipEnd && okN || !t.skipEnd {
					if t.skipEnd && okN && fiN.pk {
						goto loopEnd
					}

					jt, _ := t.add(names, mmi, fi, inner)
					jt.jtl = jtl
					jtl = jt
				}

			}

			if num != i {
				continue
			}

		loopEnd:

			if i == 0 || jtl == nil {
				index = "T0"
			} else {
				index = jtl.index
			}

			info = fi

			if jtl == nil {
				name = fi.name
			} else {
				name = jtl.name + ExprSep + fi.name
			}

			switch {
			case fi.rel:

			case fi.reverse:
				switch fi.reverseFieldInfo.fieldType {
				case RelOneToOne, RelForeignKey:
					index = jtl.index
					info = fi.reverseFieldInfo.mi.fields.pk
					name = info.name
				}
			}

			break loopFor

		} else {
			index = ""
			name = ""
			info = nil
			success = false
			return
		}
	}

	success = index != "" && info != nil
	return
}

// generate condition sql.
func (t *dbTables) getCondSQL(cond *Condition, sub bool, tz *time.Location) (where string, params []interface{}) {
	if cond == nil || cond.IsEmpty() {
		return
	}

	Q := t.base.TableQuote()

	mi := t.mi

	for i, p := range cond.params {
		if i > 0 {
			if p.isOr {
				where += "OR "
			} else {
				where += "AND "
			}
		}
		if p.isNot {
			where += "NOT "
		}
		if p.isCond {
			w, ps := t.getCondSQL(p.cond, true, tz)
			if w != "" {
				w = fmt.Sprintf("( %s) ", w)
			}
			where += w
			params = append(params, ps...)
		} else {
			exprs := p.exprs

			num := len(exprs) - 1
			operator := ""
			if operators[exprs[num]] {
				operator = exprs[num]
				exprs = exprs[:num]
			}

			index, _, fi, suc := t.parseExprs(mi, exprs)
			if suc == false {
				panic(fmt.Errorf("unknown field/column name `%s`", strings.Join(p.exprs, ExprSep)))
			}

			if operator == "" {
				operator = "exact"
			}

			operSQL, args := t.base.GenerateOperatorSQL(mi, fi, operator, p.args, tz)

			leftCol := fmt.Sprintf("%s.%s%s%s", index, Q, fi.column, Q)
			t.base.GenerateOperatorLeftCol(fi, operator, &leftCol)

			where += fmt.Sprintf("%s %s ", leftCol, operSQL)
			params = append(params, args...)

		}
	}

	if sub == false && where != "" {
		where = "WHERE " + where
	}

	return
}

// generate group sql.
func (t *dbTables) getGroupSQL(groups []string) (groupSQL string) {
	if len(groups) == 0 {
		return
	}

	Q := t.base.TableQuote()

	groupSqls := make([]string, 0, len(groups))
	for _, group := range groups {
		exprs := strings.Split(group, ExprSep)

		index, _, fi, suc := t.parseExprs(t.mi, exprs)
		if suc == false {
			panic(fmt.Errorf("unknown field/column name `%s`", strings.Join(exprs, ExprSep)))
		}

		groupSqls = append(groupSqls, fmt.Sprintf("%s.%s%s%s", index, Q, fi.column, Q))
	}

	groupSQL = fmt.Sprintf("GROUP BY %s ", strings.Join(groupSqls, ", "))
	return
}

// generate order sql.
func (t *dbTables) getOrderSQL(orders []string) (orderSQL string) {
	if len(orders) == 0 {
		return
	}

	Q := t.base.TableQuote()

	orderSqls := make([]string, 0, len(orders))
	for _, order := range orders {
		asc := "ASC"
		if order[0] == '-' {
			asc = "DESC"
			order = order[1:]
		}
		exprs := strings.Split(order, ExprSep)

		index, _, fi, suc := t.parseExprs(t.mi, exprs)
		if suc == false {
			panic(fmt.Errorf("unknown field/column name `%s`", strings.Join(exprs, ExprSep)))
		}

		orderSqls = append(orderSqls, fmt.Sprintf("%s.%s%s%s %s", index, Q, fi.column, Q, asc))
	}

	orderSQL = fmt.Sprintf("ORDER BY %s ", strings.Join(orderSqls, ", "))
	return
}

// generate limit sql.
func (t *dbTables) getLimitSQL(mi *modelInfo, offset int64, limit int64) (limits string) {
	if limit == 0 {
		limit = int64(DefaultRowsLimit)
	}
	if limit < 0 {
		// no limit
		if offset > 0 {
			maxLimit := t.base.MaxLimit()
			if maxLimit == 0 {
				limits = fmt.Sprintf("OFFSET %d", offset)
			} else {
				limits = fmt.Sprintf("LIMIT %d OFFSET %d", maxLimit, offset)
			}
		}
	} else if offset <= 0 {
		limits = fmt.Sprintf("LIMIT %d", limit)
	} else {
		limits = fmt.Sprintf("LIMIT %d OFFSET %d", limit, offset)
	}
	return
}

// crete new tables collection.
func newDbTables(mi *modelInfo, base dbBaser) *dbTables {
	tables := &dbTables{}
	tables.tablesM = make(map[string]*dbTable)
	tables.mi = mi
	tables.base = base
	return tables
}
