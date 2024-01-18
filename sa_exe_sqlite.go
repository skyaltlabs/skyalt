/*
Copyright 2023 Milan Suk

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this db except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func (node *SANode) _SAExe_Sqlite_open(fileAttr *SANodeAttr) *sql.DB {

	file := fileAttr.GetString()

	if file == "" {
		fileAttr.SetErrorExe("empty")
		return nil
	}

	_, err := os.Stat(file)
	if os.IsNotExist(err) {
		fileAttr.SetErrorExe(fmt.Sprintf("file(%s) doesn't exist", file))
		return nil
	}

	db, err := sql.Open("sqlite3", "file:"+file+"?&_journal_mode=WAL")
	if err != nil {
		fileAttr.SetErrorExe(fmt.Sprintf("Open(%s) failed: %v", file, err))
		return nil
	}
	return db
}

func (node *SANode) SAExe_Sqlite_insert() bool {

	triggerAttr := node.GetAttrUi("trigger", "0", SAAttrUi_SWITCH)
	fileAttr := node.GetAttr("file", "")
	tableAttr := node.GetAttr("table", "") //combo ...

	if triggerAttr.GetBool() {

		db := node._SAExe_Sqlite_open(fileAttr)
		if db == nil {
			return false
		}
		defer db.Close()

		table := tableAttr.GetString()
		if table == "" {
			tableAttr.SetErrorExe("empty")
			return false
		}

		query := "INSERT INTO" + table
		//...........
		//INSERT INTO table_name (column1, column2, column3, ...) VALUES (value1, value2, value3, ...);

		query += ";"
		db.Exec(query)

		triggerAttr.SetExpBool(false)
	}

	return true
}

func (node *SANode) SAExe_Sqlite_select() bool {

	fileAttr := node.GetAttr("file", "")
	queryAttr := node.GetAttr("query", "")
	rowsAttr := node.GetAttr("_rows", "[]")
	rowsAttr.result.SetBlob(nil) //reset

	colsAttr := node.GetAttr("_columns", "[]")
	colsAttr.result.SetBlob(nil) //reset

	db := node._SAExe_Sqlite_open(fileAttr)
	if db == nil {
		return false
	}
	defer db.Close()

	query := queryAttr.GetString()
	if query == "" {
		queryAttr.SetErrorExe("empty")
		return false
	}

	rows, err := db.Query(query)
	if err != nil {
		queryAttr.SetErrorExe(fmt.Sprintf("Query() failed: %v", err))
		return false
	}
	defer rows.Close()

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		node.SetError(fmt.Sprintf("ColumnTypes() failed: %v", err))
		return false
	}
	scanArgs := make([]interface{}, len(columnTypes))
	for i, v := range columnTypes {
		switch v.DatabaseTypeName() {

		case "BLOB":
			scanArgs[i] = new(sql.RawBytes)

		case "BOOL":
			scanArgs[i] = new(sql.NullFloat64)

		case "INT4":
			scanArgs[i] = new(sql.NullFloat64)

		case "REAL":
			scanArgs[i] = new(sql.NullFloat64)

		case "VARCHAR", "TEXT", "UUID", "TIMESTAMP":
			scanArgs[i] = new(sql.NullString)

		default:
			scanArgs[i] = new(sql.NullString)
		}
	}

	{
		columnNames, err := rows.Columns()
		if err != nil {
			node.SetError(fmt.Sprintf("Columns() failed: %v", err))
			return false
		}

		cols := "["
		for _, c := range columnNames {
			cols += "\"" + c + "\","
		}
		cols, _ = strings.CutSuffix(cols, ",")
		cols += "]"

		colsAttr.result.SetBlob([]byte(cols))
	}

	rws := "["

	//rows
	for rows.Next() {
		err := rows.Scan(scanArgs...)
		if err != nil {
			node.SetError(fmt.Sprintf("Scan() failed: %v", err))
			return false
		}

		//columns
		rws += "["
		for i := range columnTypes {
			if z, ok := (scanArgs[i]).(*sql.RawBytes); ok {
				rws += "\"" + base64.StdEncoding.EncodeToString(*z) + "\","
				continue
			}

			if z, ok := (scanArgs[i]).(*sql.NullString); ok {
				rws += "\"" + z.String + "\","
				continue
			}
			if z, ok := (scanArgs[i]).(*sql.NullFloat64); ok {
				rws += strconv.FormatFloat(z.Float64, 'f', -1, 64) + ","
				continue
			}

			fmt.Printf("Unknown type\n")
		}
		rws, _ = strings.CutSuffix(rws, ",")
		rws += "],"
	}

	rws, _ = strings.CutSuffix(rws, ",")
	rws += "]"

	rowsAttr.result.SetBlob([]byte(rws))
	return true
}
