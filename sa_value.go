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
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type SAValueArray struct {
	items []SAValue
}

func (va *SAValueArray) GetString() string {
	s := "["

	for _, it := range va.items {
		quotes := !it.IsNumber()
		if quotes {
			s += "\""
		}
		s += it.String()
		if quotes {
			s += "\""
		}
		s += ","
	}

	s, _ = strings.CutSuffix(s, ",")
	s += "]"
	return s
}

func (vt *SAValueArray) Get(i int) *SAValue {
	return &vt.items[i]
}
func (vt *SAValueArray) Num() int {
	return len(vt.items)
}

func (va *SAValueArray) Resize(n int) {
	//realloc
	v := make([]SAValue, n)
	copy(v, va.items)
	//reset
	for i := len(va.items); i < n; i++ {
		v[i].value = 0.0
	}
	va.items = v
}

type SAValueTable struct {
	names []string
	items []SAValueArray
}

func InitSAValueTable(names []string) SAValueTable {
	var vt SAValueTable
	vt.names = names
	vt.items = make([]SAValueArray, len(names))
	return vt
}

func (vt *SAValueTable) NumRows() int {
	if len(vt.items) > 0 {
		return vt.items[0].Num()
	}
	return 0
}
func (vt *SAValueTable) Get(col int, row int) *SAValue {
	return vt.items[col].Get(row)
}
func (vt *SAValueTable) Resize(n int) {
	for i := range vt.names {
		vt.items[i].Resize(n)
	}
}
func (vt *SAValueTable) AddRow() int {
	n := vt.NumRows()
	vt.Resize(n + 1)
	return n
}

func (vt *SAValueTable) GetString() string {
	s := "{"

	s += "\"" + strings.Join(vt.names, ";") + "\""
	s += ","

	for r := 0; r < vt.NumRows(); r++ {
		for c := range vt.names {
			cell := vt.Get(c, r)
			quotes := !cell.IsNumber()
			if quotes {
				s += "\""
			}
			s += cell.String()
			if quotes {
				s += "\""
			}
			s += ","
		}
	}

	s, _ = strings.CutSuffix(s, ",")
	s += "}"
	return s
}

// can save only Number/String/Blob into JSON, because Table would become something else ...
type SAValue struct {
	value interface{}
}

func InitSAValue() SAValue {
	return SAValue{value: ""}
}

func (v *SAValue) SetNumber(val float64) {
	v.value = val
}
func (v *SAValue) SetString(val string) {
	v.value = val
}
func (v *SAValue) SetBlobCopy(val []byte) {
	vv := make([]byte, len(val))
	copy(vv, val)
	v.value = v
}
func (v *SAValue) SetArray(val SAValueArray) {
	v.value = val
}
func (v *SAValue) SetTable(val SAValueTable) {
	v.value = val
}

func (v *SAValue) SetBool(val bool) {
	v.SetNumber(OsTrnFloat(val, 1, 0))
}

func (v *SAValue) SetInt(val int) {
	v.SetNumber(float64(val))
}

func (v *SAValue) String() string {
	switch vv := v.value.(type) {
	case string:
		return vv
	case float64:
		return strconv.FormatFloat(vv, 'f', -1, 64)
	case []byte:
		return string(vv) //?

	case SAValueArray:
		return vv.GetString()

	case SAValueTable:
		return vv.GetString()

	default:
		fmt.Println("Warning: Unknown SAValue conversion")
	}
	return ""
}
func (v *SAValue) Number() float64 {
	switch vv := v.value.(type) {
	case string:
		ret, _ := strconv.ParseFloat(vv, 64)
		return ret
	case float64:
		return vv
	case []byte:
		ret, _ := strconv.ParseFloat(string(vv), 64)
		return ret
	default:
		fmt.Println("Warning: Unknown SAValue conversion")
	}
	return 0
}

func (v *SAValue) Blob() []byte {
	switch vv := v.value.(type) {
	case string:
		return []byte(vv)
	case float64:
		return nil
	case []byte:
		return vv
	default:
		fmt.Println("Warning: Unknown SAValue conversion")
	}
	return nil
}

func (v *SAValue) Table() SAValueTable {
	switch vv := v.value.(type) {
	case SAValueTable:
		return vv
	}
	return SAValueTable{}
}

func (v *SAValue) Is() bool {
	switch vv := v.value.(type) {
	case string:
		return vv != ""
	case float64:
		return vv != 0
	case []byte:
		return len(vv) > 0
	}
	return false
}

func (v *SAValue) IsText() bool {
	switch v.value.(type) {
	case string:
		return true
	}
	return false
}
func (v *SAValue) IsNumber() bool {
	switch v.value.(type) {
	case float64:
		return true
	}
	return false
}
func (v *SAValue) IsBlobb() bool {
	switch v.value.(type) {
	case []byte:
		return true
	}
	return false
}
func (v *SAValue) IsTable() bool {
	switch v.value.(type) {
	case SAValueTable:
		return true
	}
	return false
}

func (A *SAValue) Cmp(B *SAValue) int {
	if reflect.TypeOf(A.value) == reflect.TypeOf(B.value) {
		switch val := A.value.(type) {
		case float64:
			ta := val
			tb := B.Number()
			return OsTrn(ta > tb, 1, 0) - OsTrn(ta < tb, 1, 0)
		case string:
			return strings.Compare(val, B.String())
		}
	} else {
		if A.IsNumber() || B.IsNumber() {
			ta := A.Number()
			tb := B.Number()
			return OsTrn(ta > tb, 1, 0) - OsTrn(ta < tb, 1, 0)
		}

		if A.IsText() && B.IsText() {
			return strings.Compare(A.String(), B.String())
		}

		return OsTrn(!A.Is() && !B.Is(), 0, 1) // both null = 0
	}

	return 1 // different
}
