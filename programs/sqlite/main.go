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
	"encoding/json"
	"fmt"
	"os"

	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	var sa SkyAltClient
	file := sa.AddAttr("file", "")
	query := sa.AddAttr("query", "")
	outRows := sa.AddAttrOut("rows", "[]")

	if len(os.Args) < 3 {
		fmt.Printf("Need 2 arguments: uid, port\n")
		return
	}

	sa.Start(os.Args[1], os.Args[2])
	defer sa.Destroy()

	for sa.Get() {
		if file.Value == "" {
			file.Error = "empty"
			sa.Finalize()
			continue
		}
		if query.Value == "" {
			query.Error = "empty"
			sa.Finalize()
			continue
		}

		_, err := os.Stat(file.Value)
		if os.IsNotExist(err) {
			file.Error = fmt.Sprintf("file(%s) doesn't exist", file.Value)
			sa.Finalize()
			continue
		}

		db, err := sql.Open("sqlite3", "file:"+file.Value+"?&_journal_mode=WAL")
		if err != nil {
			file.Error = fmt.Sprintf("Open(%s) failed: %v", file.Value, err)
			sa.Finalize()
			continue
		}

		rows, err := db.Query(query.Value)
		if err != nil {
			query.Error = fmt.Sprintf("Query() failed: %v", err)
			sa.Finalize()
			db.Close()
			continue
		}

		columnTypes, err := rows.ColumnTypes()
		if err != nil {
			sa.Error(fmt.Sprintf("ColumnTypes() failed: %v", err))
			db.Close()
			continue
		}
		scanArgs := make([]interface{}, len(columnTypes))
		for i, v := range columnTypes {
			switch v.DatabaseTypeName() {
			case "VARCHAR", "TEXT", "UUID", "TIMESTAMP":
				scanArgs[i] = new(sql.NullString)

			case "BOOL":
				scanArgs[i] = new(sql.NullBool)

			case "INT4":
				scanArgs[i] = new(sql.NullInt64)

			case "REAL":
				scanArgs[i] = new(sql.NullFloat64)

			default:
				scanArgs[i] = new(sql.NullString)
			}
		}

		finalRows := []interface{}{}
		ok := true
		for rows.Next() {
			err := rows.Scan(scanArgs...)
			if err != nil {
				sa.Error(fmt.Sprintf("Scan() failed: %v", err))
				ok = false
				break
			}
			masterData := map[string]interface{}{}
			for i, v := range columnTypes {
				if z, ok := (scanArgs[i]).(*sql.NullBool); ok {
					masterData[v.Name()] = z.Bool
					continue
				}
				if z, ok := (scanArgs[i]).(*sql.NullString); ok {
					masterData[v.Name()] = z.String
					continue
				}
				if z, ok := (scanArgs[i]).(*sql.NullInt64); ok {
					masterData[v.Name()] = z.Int64
					continue
				}
				if z, ok := (scanArgs[i]).(*sql.NullFloat64); ok {
					masterData[v.Name()] = z.Float64
					continue
				}
				if z, ok := (scanArgs[i]).(*sql.NullInt32); ok {
					masterData[v.Name()] = z.Int32
					continue
				}
				masterData[v.Name()] = scanArgs[i]
			}
			finalRows = append(finalRows, masterData)
		}

		if !ok {
			db.Close()
			continue
		}

		//result
		js, err := json.Marshal(finalRows)
		if err != nil {
			sa.Error(fmt.Sprintf("Marshal() failed: %v", err))
			db.Close()
			continue
		}

		outRows.Value = string(js)
		db.Close()
		if !sa.Finalize() {
			break
		}
	}
}
