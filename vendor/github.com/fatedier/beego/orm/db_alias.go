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
	"fmt"
	"reflect"
	"sync"
	"time"
)

// DriverType database driver constant int.
type DriverType int

// Enum the Database driver
const (
	_          DriverType = iota // int enum type
	DRMySQL                      // mysql
	DRSqlite                     // sqlite
	DROracle                     // oracle
	DRPostgres                   // pgsql
	DRTiDB                       // TiDB
)

// database driver string.
type driver string

// get type constant int of current driver..
func (d driver) Type() DriverType {
	a, _ := dataBaseCache.get(string(d))
	return a.Driver
}

// get name of current driver
func (d driver) Name() string {
	return string(d)
}

// check driver iis implemented Driver interface or not.
var _ Driver = new(driver)

var (
	dataBaseCache = &_dbCache{cache: make(map[string]*alias)}
	drivers       = map[string]DriverType{
		"mysql":    DRMySQL,
		"postgres": DRPostgres,
		"sqlite3":  DRSqlite,
		"tidb":     DRTiDB,
		"oracle":   DROracle,
	}
	dbBasers = map[DriverType]dbBaser{
		DRMySQL:    newdbBaseMysql(),
		DRSqlite:   newdbBaseSqlite(),
		DROracle:   newdbBaseOracle(),
		DRPostgres: newdbBasePostgres(),
		DRTiDB:     newdbBaseTidb(),
	}
)

// database alias cacher.
type _dbCache struct {
	mux   sync.RWMutex
	cache map[string]*alias
}

// add database alias with original name.
func (ac *_dbCache) add(name string, al *alias) (added bool) {
	ac.mux.Lock()
	defer ac.mux.Unlock()
	if _, ok := ac.cache[name]; !ok {
		ac.cache[name] = al
		added = true
	}
	return
}

// get database alias if cached.
func (ac *_dbCache) get(name string) (al *alias, ok bool) {
	ac.mux.RLock()
	defer ac.mux.RUnlock()
	al, ok = ac.cache[name]
	return
}

// get default alias.
func (ac *_dbCache) getDefault() (al *alias) {
	al, _ = ac.get("default")
	return
}

type alias struct {
	Name         string
	Driver       DriverType
	DriverName   string
	DataSource   string
	MaxIdleConns int
	MaxOpenConns int
	DB           *sql.DB
	DbBaser      dbBaser
	TZ           *time.Location
	Engine       string
}

func detectTZ(al *alias) {
	// orm timezone system match database
	// default use Local
	al.TZ = time.Local

	if al.DriverName == "sphinx" {
		return
	}

	switch al.Driver {
	case DRMySQL:
		row := al.DB.QueryRow("SELECT TIMEDIFF(NOW(), UTC_TIMESTAMP)")
		var tz string
		row.Scan(&tz)
		if len(tz) >= 8 {
			if tz[0] != '-' {
				tz = "+" + tz
			}
			t, err := time.Parse("-07:00:00", tz)
			if err == nil {
				al.TZ = t.Location()
			} else {
				DebugLog.Printf("Detect DB timezone: %s %s\n", tz, err.Error())
			}
		}

		// get default engine from current database
		row = al.DB.QueryRow("SELECT ENGINE, TRANSACTIONS FROM information_schema.engines WHERE SUPPORT = 'DEFAULT'")
		var engine string
		var tx bool
		row.Scan(&engine, &tx)

		if engine != "" {
			al.Engine = engine
		} else {
			al.Engine = "INNODB"
		}

	case DRSqlite, DROracle:
		al.TZ = time.UTC

	case DRPostgres:
		row := al.DB.QueryRow("SELECT current_setting('TIMEZONE')")
		var tz string
		row.Scan(&tz)
		loc, err := time.LoadLocation(tz)
		if err == nil {
			al.TZ = loc
		} else {
			DebugLog.Printf("Detect DB timezone: %s %s\n", tz, err.Error())
		}
	}
}

func addAliasWthDB(aliasName, driverName string, db *sql.DB) (*alias, error) {
	al := new(alias)
	al.Name = aliasName
	al.DriverName = driverName
	al.DB = db

	if dr, ok := drivers[driverName]; ok {
		al.DbBaser = dbBasers[dr]
		al.Driver = dr
	} else {
		return nil, fmt.Errorf("driver name `%s` have not registered", driverName)
	}

	err := db.Ping()
	if err != nil {
		return nil, fmt.Errorf("register db Ping `%s`, %s", aliasName, err.Error())
	}

	if dataBaseCache.add(aliasName, al) == false {
		return nil, fmt.Errorf("DataBase alias name `%s` already registered, cannot reuse", aliasName)
	}

	return al, nil
}

// AddAliasWthDB add a aliasName for the drivename
func AddAliasWthDB(aliasName, driverName string, db *sql.DB) error {
	_, err := addAliasWthDB(aliasName, driverName, db)
	return err
}

// RegisterDataBase Setting the database connect params. Use the database driver self dataSource args.
func RegisterDataBase(aliasName, driverName, dataSource string, params ...int) error {
	var (
		err error
		db  *sql.DB
		al  *alias
	)

	db, err = sql.Open(driverName, dataSource)
	if err != nil {
		err = fmt.Errorf("register db `%s`, %s", aliasName, err.Error())
		goto end
	}

	al, err = addAliasWthDB(aliasName, driverName, db)
	if err != nil {
		goto end
	}

	al.DataSource = dataSource

	detectTZ(al)

	for i, v := range params {
		switch i {
		case 0:
			SetMaxIdleConns(al.Name, v)
		case 1:
			SetMaxOpenConns(al.Name, v)
		}
	}

end:
	if err != nil {
		if db != nil {
			db.Close()
		}
		DebugLog.Println(err.Error())
	}

	return err
}

// RegisterDriver Register a database driver use specify driver name, this can be definition the driver is which database type.
func RegisterDriver(driverName string, typ DriverType) error {
	if t, ok := drivers[driverName]; ok == false {
		drivers[driverName] = typ
	} else {
		if t != typ {
			return fmt.Errorf("driverName `%s` db driver already registered and is other type\n", driverName)
		}
	}
	return nil
}

// SetDataBaseTZ Change the database default used timezone
func SetDataBaseTZ(aliasName string, tz *time.Location) error {
	if al, ok := dataBaseCache.get(aliasName); ok {
		al.TZ = tz
	} else {
		return fmt.Errorf("DataBase alias name `%s` not registered\n", aliasName)
	}
	return nil
}

// SetMaxIdleConns Change the max idle conns for *sql.DB, use specify database alias name
func SetMaxIdleConns(aliasName string, maxIdleConns int) {
	al := getDbAlias(aliasName)
	al.MaxIdleConns = maxIdleConns
	al.DB.SetMaxIdleConns(maxIdleConns)
}

// SetMaxOpenConns Change the max open conns for *sql.DB, use specify database alias name
func SetMaxOpenConns(aliasName string, maxOpenConns int) {
	al := getDbAlias(aliasName)
	al.MaxOpenConns = maxOpenConns
	// for tip go 1.2
	if fun := reflect.ValueOf(al.DB).MethodByName("SetMaxOpenConns"); fun.IsValid() {
		fun.Call([]reflect.Value{reflect.ValueOf(maxOpenConns)})
	}
}

// GetDB Get *sql.DB from registered database by db alias name.
// Use "default" as alias name if you not set.
func GetDB(aliasNames ...string) (*sql.DB, error) {
	var name string
	if len(aliasNames) > 0 {
		name = aliasNames[0]
	} else {
		name = "default"
	}
	al, ok := dataBaseCache.get(name)
	if ok {
		return al.DB, nil
	}
	return nil, fmt.Errorf("DataBase of alias name `%s` not found\n", name)
}
