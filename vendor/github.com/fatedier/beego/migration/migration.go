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

// Package migration is used for migration
//
// The table structure is as follow:
//
//	CREATE TABLE `migrations` (
//		`id_migration` int(10) unsigned NOT NULL AUTO_INCREMENT COMMENT 'surrogate key',
//		`name` varchar(255) DEFAULT NULL COMMENT 'migration name, unique',
//		`created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'date migrated or rolled back',
//		`statements` longtext COMMENT 'SQL statements for this migration',
//		`rollback_statements` longtext,
//		`status` enum('update','rollback') DEFAULT NULL COMMENT 'update indicates it is a normal migration while rollback means this migration is rolled back',
//		PRIMARY KEY (`id_migration`)
//	) ENGINE=InnoDB DEFAULT CHARSET=utf8;
package migration

import (
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/astaxie/beego/logs"
	"github.com/astaxie/beego/orm"
)

// const the data format for the bee generate migration datatype
const (
	DateFormat   = "20060102_150405"
	DBDateFormat = "2006-01-02 15:04:05"
)

// Migrationer is an interface for all Migration struct
type Migrationer interface {
	Up()
	Down()
	Reset()
	Exec(name, status string) error
	GetCreated() int64
}

var (
	migrationMap map[string]Migrationer
)

func init() {
	migrationMap = make(map[string]Migrationer)
}

// Migration the basic type which will implement the basic type
type Migration struct {
	sqls    []string
	Created string
}

// Up implement in the Inheritance struct for upgrade
func (m *Migration) Up() {

}

// Down implement in the Inheritance struct for down
func (m *Migration) Down() {

}

// SQL add sql want to execute
func (m *Migration) SQL(sql string) {
	m.sqls = append(m.sqls, sql)
}

// Reset the sqls
func (m *Migration) Reset() {
	m.sqls = make([]string, 0)
}

// Exec execute the sql already add in the sql
func (m *Migration) Exec(name, status string) error {
	o := orm.NewOrm()
	for _, s := range m.sqls {
		logs.Info("exec sql:", s)
		r := o.Raw(s)
		_, err := r.Exec()
		if err != nil {
			return err
		}
	}
	return m.addOrUpdateRecord(name, status)
}

func (m *Migration) addOrUpdateRecord(name, status string) error {
	o := orm.NewOrm()
	if status == "down" {
		status = "rollback"
		p, err := o.Raw("update migrations set status = ?, rollback_statements = ?, created_at = ? where name = ?").Prepare()
		if err != nil {
			return nil
		}
		_, err = p.Exec(status, strings.Join(m.sqls, "; "), time.Now().Format(DBDateFormat), name)
		return err
	}
	status = "update"
	p, err := o.Raw("insert into migrations(name, created_at, statements, status) values(?,?,?,?)").Prepare()
	if err != nil {
		return err
	}
	_, err = p.Exec(name, time.Now().Format(DBDateFormat), strings.Join(m.sqls, "; "), status)
	return err
}

// GetCreated get the unixtime from the Created
func (m *Migration) GetCreated() int64 {
	t, err := time.Parse(DateFormat, m.Created)
	if err != nil {
		return 0
	}
	return t.Unix()
}

// Register register the Migration in the map
func Register(name string, m Migrationer) error {
	if _, ok := migrationMap[name]; ok {
		return errors.New("already exist name:" + name)
	}
	migrationMap[name] = m
	return nil
}

// Upgrade upgrate the migration from lasttime
func Upgrade(lasttime int64) error {
	sm := sortMap(migrationMap)
	i := 0
	for _, v := range sm {
		if v.created > lasttime {
			logs.Info("start upgrade", v.name)
			v.m.Reset()
			v.m.Up()
			err := v.m.Exec(v.name, "up")
			if err != nil {
				logs.Error("execute error:", err)
				time.Sleep(2 * time.Second)
				return err
			}
			logs.Info("end upgrade:", v.name)
			i++
		}
	}
	logs.Info("total success upgrade:", i, " migration")
	time.Sleep(2 * time.Second)
	return nil
}

// Rollback rollback the migration by the name
func Rollback(name string) error {
	if v, ok := migrationMap[name]; ok {
		logs.Info("start rollback")
		v.Reset()
		v.Down()
		err := v.Exec(name, "down")
		if err != nil {
			logs.Error("execute error:", err)
			time.Sleep(2 * time.Second)
			return err
		}
		logs.Info("end rollback")
		time.Sleep(2 * time.Second)
		return nil
	}
	logs.Error("not exist the migrationMap name:" + name)
	time.Sleep(2 * time.Second)
	return errors.New("not exist the migrationMap name:" + name)
}

// Reset reset all migration
// run all migration's down function
func Reset() error {
	sm := sortMap(migrationMap)
	i := 0
	for j := len(sm) - 1; j >= 0; j-- {
		v := sm[j]
		if isRollBack(v.name) {
			logs.Info("skip the", v.name)
			time.Sleep(1 * time.Second)
			continue
		}
		logs.Info("start reset:", v.name)
		v.m.Reset()
		v.m.Down()
		err := v.m.Exec(v.name, "down")
		if err != nil {
			logs.Error("execute error:", err)
			time.Sleep(2 * time.Second)
			return err
		}
		i++
		logs.Info("end reset:", v.name)
	}
	logs.Info("total success reset:", i, " migration")
	time.Sleep(2 * time.Second)
	return nil
}

// Refresh first Reset, then Upgrade
func Refresh() error {
	err := Reset()
	if err != nil {
		logs.Error("execute error:", err)
		time.Sleep(2 * time.Second)
		return err
	}
	err = Upgrade(0)
	return err
}

type dataSlice []data

type data struct {
	created int64
	name    string
	m       Migrationer
}

// Len is part of sort.Interface.
func (d dataSlice) Len() int {
	return len(d)
}

// Swap is part of sort.Interface.
func (d dataSlice) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}

// Less is part of sort.Interface. We use count as the value to sort by
func (d dataSlice) Less(i, j int) bool {
	return d[i].created < d[j].created
}

func sortMap(m map[string]Migrationer) dataSlice {
	s := make(dataSlice, 0, len(m))
	for k, v := range m {
		d := data{}
		d.created = v.GetCreated()
		d.name = k
		d.m = v
		s = append(s, d)
	}
	sort.Sort(s)
	return s
}

func isRollBack(name string) bool {
	o := orm.NewOrm()
	var maps []orm.Params
	num, err := o.Raw("select * from migrations where `name` = ? order by id_migration desc", name).Values(&maps)
	if err != nil {
		logs.Info("get name has error", err)
		return false
	}
	if num <= 0 {
		return false
	}
	if maps[0]["status"] == "rollback" {
		return true
	}
	return false
}
