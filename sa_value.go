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

type SAValueBlob struct {
	data []byte
	hash OsHash
}

func NewSAValueBlob(in []byte) *SAValueBlob {
	var bl SAValueBlob
	bl.data = make([]byte, len(in))
	copy(bl.data, in)

	bl.hash, _ = InitOsHash(in) //err ...
	return &bl
}

type SAValueArray struct {
	items []SAValue
}

func (va *SAValueArray) GetV2() OsV2 {
	v := OsV2{}
	if len(va.items) >= 1 {
		v.X = int(va.items[0].Number())
	}
	if len(va.items) >= 2 {
		v.Y = int(va.items[1].Number())
	}
	return v
}
func (va *SAValueArray) GetV4() OsV4 {
	v := OsV4{}
	if len(va.items) >= 1 {
		v.Start.X = int(va.items[0].Number())
	}
	if len(va.items) >= 2 {
		v.Start.Y = int(va.items[1].Number())
	}
	if len(va.items) >= 3 {
		v.Size.X = int(va.items[2].Number())
	}
	if len(va.items) >= 4 {
		v.Size.Y = int(va.items[3].Number())
	}
	return v
}

func (va *SAValueArray) GetCd() OsCd {
	v := OsCd{}
	if len(va.items) >= 1 {
		v.R = byte(va.items[0].Number())
	}
	if len(va.items) >= 2 {
		v.G = byte(va.items[1].Number())
	}
	if len(va.items) >= 3 {
		v.B = byte(va.items[2].Number())
	}
	if len(va.items) >= 4 {
		v.A = byte(va.items[3].Number())
	}
	return v
}

func (va *SAValueArray) SetV4(v OsV4) {
	va.Resize(4)
	va.Get(0).SetInt(v.Start.X)
	va.Get(1).SetInt(v.Start.Y)
	va.Get(2).SetInt(v.Size.X)
	va.Get(3).SetInt(v.Size.Y)
}
func (va *SAValueArray) SetCd(v OsCd) {
	va.Resize(4)
	va.Get(0).SetInt(int(v.R))
	va.Get(1).SetInt(int(v.G))
	va.Get(2).SetInt(int(v.B))
	va.Get(3).SetInt(int(v.A))
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
	items []*SAValueArray
}

func NewSAValueTable(names []string) *SAValueTable {
	var vt SAValueTable
	vt.names = names
	vt.items = make([]*SAValueArray, len(names))

	for i := range vt.items {
		vt.items[i] = &SAValueArray{}
	}

	return &vt
}

func (vt *SAValueTable) NumRows() int {
	if len(vt.items) > 0 {
		return vt.items[0].Num()
	}
	return 0
}

func (vt *SAValueTable) FindName(name string) int {
	for i, nm := range vt.names {
		if strings.EqualFold(nm, name) {
			return i
		}
	}
	return -1
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
func (v *SAValue) SetBlobCopy(in []byte) {
	v.value = NewSAValueBlob(in)
}
func (v *SAValue) SetArray(val *SAValueArray) {
	v.value = val
}
func (v *SAValue) SetTable(val *SAValueTable) {
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
	case *SAValueBlob:
		return string(vv.data) //?

	case *SAValueArray:
		return vv.GetString()

	case *SAValueTable:
		return vv.GetString()

	default:
		fmt.Println("Warning: Unknown SAValue conversion into String")
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
	default:
		fmt.Println("Warning: Unknown SAValue conversion into Number")
	}
	return 0
}

func (v *SAValue) Blob() *SAValueBlob {
	switch vv := v.value.(type) {
	case *SAValueBlob:
		return vv
	default:
		fmt.Println("Warning: Unknown SAValue conversion into Blob")
	}
	return nil
}

func (v *SAValue) Array() *SAValueArray {
	switch vv := v.value.(type) {
	case *SAValueArray:
		return vv
	}
	return &SAValueArray{}
}
func (v *SAValue) Table() *SAValueTable {
	switch vv := v.value.(type) {
	case *SAValueTable:
		return vv
	}
	return &SAValueTable{}
}

func (v *SAValue) Is() bool {
	switch vv := v.value.(type) {
	case string:
		return vv != ""
	case float64:
		return vv != 0
	case *SAValueBlob:
		return len(vv.data) > 0
	case *SAValueArray:
		return len(vv.items) > 0
	case *SAValueTable:
		return len(vv.items) > 0 && len(vv.names) > 0
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
func (v *SAValue) IsBlob() bool {
	switch v.value.(type) {
	case *SAValueBlob:
		return true
	}
	return false
}
func (v *SAValue) IsTable() bool {
	switch v.value.(type) {
	case *SAValueTable:
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
