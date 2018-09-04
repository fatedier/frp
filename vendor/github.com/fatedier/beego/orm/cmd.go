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
	"flag"
	"fmt"
	"os"
	"strings"
)

type commander interface {
	Parse([]string)
	Run() error
}

var (
	commands = make(map[string]commander)
)

// print help.
func printHelp(errs ...string) {
	content := `orm command usage:

    syncdb     - auto create tables
    sqlall     - print sql of create tables
    help       - print this help
`

	if len(errs) > 0 {
		fmt.Println(errs[0])
	}
	fmt.Println(content)
	os.Exit(2)
}

// RunCommand listen for orm command and then run it if command arguments passed.
func RunCommand() {
	if len(os.Args) < 2 || os.Args[1] != "orm" {
		return
	}

	BootStrap()

	args := argString(os.Args[2:])
	name := args.Get(0)

	if name == "help" {
		printHelp()
	}

	if cmd, ok := commands[name]; ok {
		cmd.Parse(os.Args[3:])
		cmd.Run()
		os.Exit(0)
	} else {
		if name == "" {
			printHelp()
		} else {
			printHelp(fmt.Sprintf("unknown command %s", name))
		}
	}
}

// sync database struct command interface.
type commandSyncDb struct {
	al        *alias
	force     bool
	verbose   bool
	noInfo    bool
	rtOnError bool
}

// parse orm command line arguments.
func (d *commandSyncDb) Parse(args []string) {
	var name string

	flagSet := flag.NewFlagSet("orm command: syncdb", flag.ExitOnError)
	flagSet.StringVar(&name, "db", "default", "DataBase alias name")
	flagSet.BoolVar(&d.force, "force", false, "drop tables before create")
	flagSet.BoolVar(&d.verbose, "v", false, "verbose info")
	flagSet.Parse(args)

	d.al = getDbAlias(name)
}

// run orm line command.
func (d *commandSyncDb) Run() error {
	var drops []string
	if d.force {
		drops = getDbDropSQL(d.al)
	}

	db := d.al.DB

	if d.force {
		for i, mi := range modelCache.allOrdered() {
			query := drops[i]
			if !d.noInfo {
				fmt.Printf("drop table `%s`\n", mi.table)
			}
			_, err := db.Exec(query)
			if d.verbose {
				fmt.Printf("    %s\n\n", query)
			}
			if err != nil {
				if d.rtOnError {
					return err
				}
				fmt.Printf("    %s\n", err.Error())
			}
		}
	}

	sqls, indexes := getDbCreateSQL(d.al)

	tables, err := d.al.DbBaser.GetTables(db)
	if err != nil {
		if d.rtOnError {
			return err
		}
		fmt.Printf("    %s\n", err.Error())
	}

	for i, mi := range modelCache.allOrdered() {
		if tables[mi.table] {
			if !d.noInfo {
				fmt.Printf("table `%s` already exists, skip\n", mi.table)
			}

			var fields []*fieldInfo
			columns, err := d.al.DbBaser.GetColumns(db, mi.table)
			if err != nil {
				if d.rtOnError {
					return err
				}
				fmt.Printf("    %s\n", err.Error())
			}

			for _, fi := range mi.fields.fieldsDB {
				if _, ok := columns[fi.column]; ok == false {
					fields = append(fields, fi)
				}
			}

			for _, fi := range fields {
				query := getColumnAddQuery(d.al, fi)

				if !d.noInfo {
					fmt.Printf("add column `%s` for table `%s`\n", fi.fullName, mi.table)
				}

				_, err := db.Exec(query)
				if d.verbose {
					fmt.Printf("    %s\n", query)
				}
				if err != nil {
					if d.rtOnError {
						return err
					}
					fmt.Printf("    %s\n", err.Error())
				}
			}

			for _, idx := range indexes[mi.table] {
				if d.al.DbBaser.IndexExists(db, idx.Table, idx.Name) == false {
					if !d.noInfo {
						fmt.Printf("create index `%s` for table `%s`\n", idx.Name, idx.Table)
					}

					query := idx.SQL
					_, err := db.Exec(query)
					if d.verbose {
						fmt.Printf("    %s\n", query)
					}
					if err != nil {
						if d.rtOnError {
							return err
						}
						fmt.Printf("    %s\n", err.Error())
					}
				}
			}

			continue
		}

		if !d.noInfo {
			fmt.Printf("create table `%s` \n", mi.table)
		}

		queries := []string{sqls[i]}
		for _, idx := range indexes[mi.table] {
			queries = append(queries, idx.SQL)
		}

		for _, query := range queries {
			_, err := db.Exec(query)
			if d.verbose {
				query = "    " + strings.Join(strings.Split(query, "\n"), "\n    ")
				fmt.Println(query)
			}
			if err != nil {
				if d.rtOnError {
					return err
				}
				fmt.Printf("    %s\n", err.Error())
			}
		}
		if d.verbose {
			fmt.Println("")
		}
	}

	return nil
}

// database creation commander interface implement.
type commandSQLAll struct {
	al *alias
}

// parse orm command line arguments.
func (d *commandSQLAll) Parse(args []string) {
	var name string

	flagSet := flag.NewFlagSet("orm command: sqlall", flag.ExitOnError)
	flagSet.StringVar(&name, "db", "default", "DataBase alias name")
	flagSet.Parse(args)

	d.al = getDbAlias(name)
}

// run orm line command.
func (d *commandSQLAll) Run() error {
	sqls, indexes := getDbCreateSQL(d.al)
	var all []string
	for i, mi := range modelCache.allOrdered() {
		queries := []string{sqls[i]}
		for _, idx := range indexes[mi.table] {
			queries = append(queries, idx.SQL)
		}
		sql := strings.Join(queries, "\n")
		all = append(all, sql)
	}
	fmt.Println(strings.Join(all, "\n\n"))

	return nil
}

func init() {
	commands["syncdb"] = new(commandSyncDb)
	commands["sqlall"] = new(commandSQLAll)
}

// RunSyncdb run syncdb command line.
// name means table's alias name. default is "default".
// force means run next sql if the current is error.
// verbose means show all info when running command or not.
func RunSyncdb(name string, force bool, verbose bool) error {
	BootStrap()

	al := getDbAlias(name)
	cmd := new(commandSyncDb)
	cmd.al = al
	cmd.force = force
	cmd.noInfo = !verbose
	cmd.verbose = verbose
	cmd.rtOnError = true
	return cmd.Run()
}
