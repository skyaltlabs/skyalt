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
	"strconv"
	"strings"
)

func _SAExe_Sqlite_open(node *SANode, fileAttr *SANodeAttr) *DiskDb {

	file := fileAttr.GetString()

	if file == "" {
		fileAttr.SetErrorStr("empty")
		return nil
	}

	if !OsFileExists(file) {
		fileAttr.SetError(fmt.Errorf("file(%s) doesn't exist", file))
		return nil
	}

	db, _, err := node.app.base.ui.win.disk.OpenDb(file)
	if err != nil {
		fileAttr.SetError(err)
		return nil
	}
	return db
}

func SAExe_Sqlite_info(node *SANode) bool {
	fileAttr := node.GetAttrUi("file", "", SAAttrUi_FILE)

	db := _SAExe_Sqlite_open(node, fileAttr)
	if db == nil {
		return false
	}

	_infoAttr := node.GetAttr("_info", []byte("[]"))

	type TableInfo struct {
		Tables []*DiskDbIndexTable
	}
	var tinf TableInfo

	var err error
	tinf.Tables, err = db.GetTableInfo()
	if err != nil {
		node.SetError(err)
	}

	js, err := json.MarshalIndent(tinf, "", "")
	if err != nil {
		node.SetError(err)
	}
	_infoAttr.SetOutBlob([]byte(js))

	return true
}

func SAExe_Sqlite_insert(node *SANode) bool {

	fileAttr := node.GetAttrUi("file", "", SAAttrUi_FILE)

	db := _SAExe_Sqlite_open(node, fileAttr)
	if db == nil {
		return false
	}

	tbls, err := db.GetTableInfo()
	if err != nil {
		fileAttr.SetError(err)
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
		tableAttr.SetErrorStr("empty")
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
		`[{"column": "a", "value": "1"}, {"column": "b", "value": "2"}]`,
		SAAttrUiValue{Fn: "map", HideAddDel: true, Map: map[string]SAAttrUiValue{"column": SAAttrUi_COMBO(columnList, columnList), "value": {}}})

	type Values struct {
		Column string
		Value  interface{}
	}
	var vals []Values
	err = json.Unmarshal([]byte(valuesAttr.GetString()), &vals)
	if err != nil {
		valuesAttr.SetError(err)
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

	node.GetAttr("_query", "").SetOutBlob([]byte(query)) //show final query

	//write
	_, err = db.Write(query, valValuesArr...)
	if err != nil {
		node.SetError(err)
		return false
	}

	return true
}

func SAExe_Sqlite_select(node *SANode) bool {

	fileAttr := node.GetAttrUi("file", "", SAAttrUi_FILE)
	queryAttr := node.GetAttr("query", "")
	rowsAttr := node.GetAttr("_rows", []byte("[]"))
	colsAttr := node.GetAttr("_columns", []byte("[]"))

	db := _SAExe_Sqlite_open(node, fileAttr)
	if db == nil {
		return false
	}

	query := queryAttr.GetString()
	if query == "" {
		queryAttr.SetErrorStr("empty")
		return false
	}

	db.Read_Lock()
	defer db.Read_Unlock()
	rows, err := db.Read_unsafe(query)
	if err != nil {
		queryAttr.SetError(fmt.Errorf("Query() failed: %w", err))
		return false
	}
	defer rows.Close()

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		node.SetError(fmt.Errorf("ColumnTypes() failed: %w", err))
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
			node.SetError(fmt.Errorf("Columns() failed: %w", err))
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
			node.SetError(fmt.Errorf("Scan() failed: %w", err))
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

				rws += OsText_RAWtoJSON(z.String) + ","
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
