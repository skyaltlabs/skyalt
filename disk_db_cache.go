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
	"encoding/binary"
	"fmt"
	"math"
)

const TpI64 = byte(0x7e)
const TpF32 = byte(0x7d)
const TpF64 = byte(0x7c)
const TpBytes = byte(0x7b)
const TpString = byte(0x7a)

type DiskDbCache struct {
	query_hash  int64
	query       string
	result_rows [][]byte
}

func NewDiskDbCache(query string, db *sql.DB) (*DiskDbCache, error) {
	var cache DiskDbCache

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("Query(%s) failed: %w", query, err)
	}
	defer rows.Close()

	cache.query = query
	{
		h, err := InitOsHash([]byte(query))
		if err != nil {
			return nil, fmt.Errorf("InitOsHash() failed: %w", err)
		}
		cache.query_hash = h.GetInt64()
	}

	// out fields
	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, fmt.Errorf("ColumnTypes() failed: %w", err)
	}

	values := make([]interface{}, len(colTypes))
	scanCallArgs := make([]interface{}, len(colTypes))
	for i := range colTypes {
		scanCallArgs[i] = &values[i]
	}

	for rows.Next() {
		//reset
		for i := range values {
			values[i] = nil
		}
		err := rows.Scan(scanCallArgs...)
		if err != nil {
			return nil, fmt.Errorf("Scan() failed: %w", err)
		}

		var row []byte
		for _, v := range values {
			row = _DiskDb_argsToArray(row, v)
		}
		cache.result_rows = append(cache.result_rows, row)
	}

	return &cache, nil
}

// 100% copied from sdk.go
func _DiskDb_argsToArray(data []byte, arg interface{}) []byte {

	switch v := arg.(type) {

	case bool:
		data = append(data, TpI64)
		if v {
			data = binary.LittleEndian.AppendUint64(data, 1)
		} else {
			data = binary.LittleEndian.AppendUint64(data, 0)
		}
	case byte:
		data = append(data, TpI64)
		data = binary.LittleEndian.AppendUint64(data, uint64(v))
	case int:
		data = append(data, TpI64)
		data = binary.LittleEndian.AppendUint64(data, uint64(v))
	case uint:
		data = append(data, TpI64)
		data = binary.LittleEndian.AppendUint64(data, uint64(v))

	case int16:
		data = append(data, TpI64)
		data = binary.LittleEndian.AppendUint64(data, uint64(v))
	case uint16:
		data = append(data, TpI64)
		data = binary.LittleEndian.AppendUint64(data, uint64(v))

	case int32:
		data = append(data, TpI64)
		data = binary.LittleEndian.AppendUint64(data, uint64(v))
	case int64:
		data = append(data, TpI64)
		data = binary.LittleEndian.AppendUint64(data, uint64(v))

	case uint32:
		data = append(data, TpI64)
		data = binary.LittleEndian.AppendUint64(data, uint64(v))
	case uint64:
		data = append(data, TpI64)
		data = binary.LittleEndian.AppendUint64(data, uint64(v))

	case float32:
		data = append(data, TpF32)
		data = binary.LittleEndian.AppendUint64(data, uint64(math.Float32bits(v)))

	case float64:
		data = append(data, TpF64)
		data = binary.LittleEndian.AppendUint64(data, uint64(math.Float64bits(v)))

	case []byte:
		data = append(data, TpBytes)
		data = binary.LittleEndian.AppendUint64(data, uint64(len(v)))
		data = append(data, v...)
	case string:
		data = append(data, TpBytes)
		data = binary.LittleEndian.AppendUint64(data, uint64(len(v)))
		data = append(data, v...)

	case nil:
		data = append(data, TpBytes)
		data = binary.LittleEndian.AppendUint64(data, 0)
	}

	return data
}
