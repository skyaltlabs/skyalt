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
	"fmt"
	"os"
)

func (node *SANode) Sqlite_select() bool {

	fileAttr := node.GetAttr("file", "")
	queryAttr := node.GetAttr("query", "")
	resultAttr := node.GetAttr("result", "{}")

	file := fileAttr.GetString()
	query := queryAttr.GetString()

	if file == "" {
		fileAttr.SetErrorExe("empty")
		return false
	}
	if query == "" {
		queryAttr.SetErrorExe("empty")
		return false
	}

	_, err := os.Stat(file)
	if os.IsNotExist(err) {
		fileAttr.SetErrorExe(fmt.Sprintf("file(%s) doesn't exist", file))
		return false
	}

	db, err := sql.Open("sqlite3", "file:"+file+"?&_journal_mode=WAL")
	if err != nil {
		fileAttr.SetErrorExe(fmt.Sprintf("Open(%s) failed: %v", file, err))
		return false
	}
	defer db.Close()

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

	columnNames, err := rows.Columns()
	if err != nil {
		node.SetError(fmt.Sprintf("Columns() failed: %v", err))
		return false
	}
	tb := NewSAValueTable(columnNames)

	for rows.Next() {
		err := rows.Scan(scanArgs...)
		if err != nil {
			node.SetError(fmt.Sprintf("Scan() failed: %v", err))
			return false
		}

		r := tb.AddRow()

		for i := range columnTypes {
			if z, ok := (scanArgs[i]).(*sql.RawBytes); ok {
				tb.Get(i, r).SetBlobCopy(*z)
				continue
			}

			if z, ok := (scanArgs[i]).(*sql.NullString); ok {
				tb.Get(i, r).SetString(z.String)
				continue
			}
			if z, ok := (scanArgs[i]).(*sql.NullFloat64); ok {
				tb.Get(i, r).SetNumber(z.Float64)
				continue
			}
		}
	}

	resultAttr.Value = ""
	resultAttr.finalValue.SetTable(tb)

	return true
}
