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
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func _SAExe_Sqlite_open(node *SANode, fileAttr *SANodeAttr) *DiskDb {

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

	/*db, err := sql.Open("sqlite3", "file:"+file+"?&_journal_mode=WAL")
	if err != nil {
		fileAttr.SetErrorExe(fmt.Sprintf("Open(%s) failed: %v", file, err))
		return nil
	}
	return db*/

	db, err := NewDiskDb(file, false, nil)
	if err != nil {
		fileAttr.SetErrorExe(err.Error())
		return nil
	}
	return db
}

func SAExe_Sqlite_insert(node *SANode) bool {

	triggerAttr := node.GetAttrUi("trigger", "0", SAAttrUi_SWITCH)

	fileAttr := node.GetAttr("file", "")

	db := _SAExe_Sqlite_open(node, fileAttr)
	if db == nil {
		return false
	}
	defer db.Destroy()

	tbls, err := db.GetTableInfo()
	if err != nil {
		fileAttr.SetErrorExe(err.Error())
		return false
	}

	var tableList string
	{
		for _, tb := range tbls {
			tableList += tb.Name + ";"
		}
		tableList, _ = strings.CutSuffix(tableList, ";")
	}
	tableAttr := node.GetAttrUi("table", "", SAAttrUi_COMBO(tableList, tableList))

	table := tableAttr.GetString()
	if table == "" {
		tableAttr.SetErrorExe("empty")
		return false
	}

	var columnList string
	{
		table := tableAttr.GetString()
		for _, tb := range tbls {
			if tb.Name == table {
				for _, cl := range tb.Columns {
					columnList += cl.Name + ";"
				}
				columnList, _ = strings.CutSuffix(columnList, ";")
			}
		}
	}

	valuesAttr := node.GetAttrUi(
		"values",
		"[{\"column\": \"a\", \"value\": \"1\"}, {\"column\": \"b\", \"value\": \"2\"}]",
		SAAttrUiValue{Fn: "map", Map: map[string]SAAttrUiValue{"column": SAAttrUi_COMBO(columnList, columnList), "value": SAAttrUiValue{}}})

	if triggerAttr.GetBool() {

		type Values struct {
			Column string
			Value  string
		}
		var vals []Values
		err := json.Unmarshal([]byte(valuesAttr.GetString()), &vals)
		if err != nil {
			valuesAttr.SetErrorExe(err.Error())
			return false
		}

		var valColumns string
		var valValues string
		var valValuesArr []interface{}
		for _, v := range vals {
			valColumns += v.Column + ","
			valValues += "?,"
			valValuesArr = append(valValuesArr, v.Value)
		}
		valColumns, _ = strings.CutSuffix(valColumns, ",")
		valValues, _ = strings.CutSuffix(valValues, ",")

		query := fmt.Sprintf("INSERT INTO %s(%s) VALUES(%s);", table, valColumns, valValues)
		_, err = db.Write(query, valValuesArr...)
		if err != nil {
			node.SetError(err.Error())
			return false
		}

		triggerAttr.SetExpBool(false)
	}

	return true
}

func SAExe_Sqlite_select(node *SANode) bool {

	fileAttr := node.GetAttr("file", "")
	queryAttr := node.GetAttr("query", "")
	rowsAttr := node.GetAttr("_rows", "[]")
	rowsAttr.SetOutBlob([]byte("[]")) //reset

	colsAttr := node.GetAttr("_columns", "[]")
	colsAttr.SetOutBlob([]byte("[]")) //reset

	db := _SAExe_Sqlite_open(node, fileAttr)
	if db == nil {
		return false
	}
	defer db.Destroy()

	query := queryAttr.GetString()
	if query == "" {
		queryAttr.SetErrorExe("empty")
		return false
	}

	rows, err := db.Read(query)
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

		colsAttr.SetOutBlob([]byte(cols))
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

	rowsAttr.SetOutBlob([]byte(rws))
	return true
}
