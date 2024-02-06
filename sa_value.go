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
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

type SAValue struct {
	value interface{}
}

func InitSAValue() SAValue {
	return SAValue{value: ""}
}

func InitSAValueInteface(v interface{}) *SAValue {
	val := &SAValue{}

	switch vv := v.(type) {
	case string:
		val.value = vv
	case float64:
		val.value = v
	case int:
		val.value = float64(vv)
	case []byte:
		val.value = InitOsBlob(vv)
	case OsBlob:
		val.value = vv
	}

	return val
}

func (v *SAValue) SetNumber(val float64) {
	v.value = val
}
func (v *SAValue) SetString(val string) {
	v.value = val
}

func (v *SAValue) SetBlob(val []byte) {
	v.value = InitOsBlob(val)
}

func (v *SAValue) SetBool(val bool) {
	v.SetNumber(OsTrnFloat(val, 1, 0))
}

func (v *SAValue) SetInt(val int) {
	v.SetNumber(float64(val))
}

func (v *SAValue) StringWithQuotes() string {

	switch vv := v.value.(type) {
	case string:
		return "\"" + OsText_PrintToRaw(vv) + "\""
	case float64:
		return strconv.FormatFloat(vv, 'f', -1, 64)
	case OsBlob:
		if len(vv.data) > 0 && (vv.data[0] == '{' || vv.data[0] == '[') {
			return string(vv.data) //map or array
		}
		return "\"" + string(vv.data) + "\"" //binary/hex

	default:
		fmt.Println("Warning: Unknown SAValue conversion into String")
	}
	return "\"\""
}

func (v *SAValue) String() string {
	switch vv := v.value.(type) {
	case string:
		return vv
	case float64:
		return strconv.FormatFloat(vv, 'f', -1, 64)
	case OsBlob:
		return string(vv.data)

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

func (v *SAValue) Blob() OsBlob {
	switch vv := v.value.(type) {
	case string:
		return InitOsBlob([]byte(vv))
	case OsBlob:
		return vv
	}
	return OsBlob{}
}

func (v *SAValue) Is() bool {
	switch vv := v.value.(type) {
	case string:
		return vv != ""
	case float64:
		return vv != 0
	case OsBlob:
		return len(vv.data) > 0
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
	case OsBlob:
		return true
	}
	return false
}

func (a *SAValue) Cd() OsCd {
	var ret OsCd

	var v []byte
	err := json.Unmarshal(a.Blob().data, &v)
	if err == nil {
		if len(v) > 0 {
			ret.R = v[0]
		}
		if len(v) > 1 {
			ret.G = v[1]
		}
		if len(v) > 2 {
			ret.B = v[2]
		}
		if len(v) > 3 {
			ret.A = v[3]
		}
	}

	return ret
}

func (a *SAValue) V2() OsV2 {
	var ret OsV2

	var v []int
	err := json.Unmarshal(a.Blob().data, &v)
	if err == nil {
		if len(v) > 0 {
			ret.X = v[0]
		}
		if len(v) > 1 {
			ret.Y = v[1]
		}
	}
	return ret
}
func (a *SAValue) V2f() OsV2f {
	var ret OsV2f

	var v []float32
	err := json.Unmarshal(a.Blob().data, &v)
	if err == nil {
		if len(v) > 0 {
			ret.X = v[0]
		}
		if len(v) > 1 {
			ret.Y = v[1]
		}
	}
	return ret
}

func (a *SAValue) V4() OsV4 {
	var ret OsV4

	var v []int
	err := json.Unmarshal(a.Blob().data, &v)
	if err == nil {
		if len(v) > 0 {
			ret.Start.X = v[0]
		}
		if len(v) > 1 {
			ret.Start.Y = v[1]
		}

		if len(v) > 2 {
			ret.Size.X = v[2]
		}
		if len(v) > 3 {
			ret.Size.Y = v[3]
		}
	}

	return ret
}

func (v *SAValue) NumArrayItems() int {

	blob := v.Blob()
	if len(blob.data) == 0 {
		return 0
	}

	var arr []interface{}
	err := json.Unmarshal(blob.data, &arr)
	if err != nil {
		//fmt.Printf("Warning: Array Unmarshal(%s) failed: %v", string(blob.data), err)
		return 0
	}
	return len(arr)
}

func (v *SAValue) NumMapItems() int {

	blob := v.Blob()
	if len(blob.data) == 0 {
		return 0
	}

	var arr map[string]interface{}
	err := json.Unmarshal(blob.data, &arr)
	if err != nil {
		//fmt.Printf("Warning: Map Unmarshal(%s) failed: %v", string(blob.data), err)
		return 0
	}
	return len(arr)
}

func (v *SAValue) GetArrayItem(i int) *SAValue {
	var arr []interface{}
	err := json.Unmarshal(v.Blob().data, &arr)
	if err != nil {
		return nil
	}

	if i >= len(arr) {
		return nil
	}

	return InitSAValueInteface(arr[i])
}

func (v *SAValue) GetMapItem(i int) (string, *SAValue) {
	var arr map[string]interface{}
	err := json.Unmarshal(v.Blob().data, &arr)
	if err != nil {
		return "", nil
	}

	if i >= len(arr) {
		return "", nil
	}

	keys := make([]string, 0, len(arr))
	for k := range arr {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	ii := 0
	for _, key := range keys {
		if i == ii {
			return key, InitSAValueInteface(arr[key])
		}
		ii++
	}

	return "", nil
}

func (v *SAValue) GetMapKey(key string) *SAValue {

	var arr map[string]interface{}
	err := json.Unmarshal(v.Blob().data, &arr)
	if err != nil {
		return nil
	}

	item, found := arr[key]
	if !found {
		return nil
	}

	return InitSAValueInteface(item)
}

func (A *SAValue) Cmp(B *SAValue, sameType bool) int {
	if reflect.TypeOf(A.value) == reflect.TypeOf(B.value) {
		switch val := A.value.(type) {
		case float64:
			ta := val
			tb := B.Number()
			return OsTrn(ta > tb, 1, 0) - OsTrn(ta < tb, 1, 0)
		case string:
			return strings.Compare(val, B.String())
		case OsBlob:
			return bytes.Compare(val.data, B.Blob().data)
		}
	} else {
		if sameType {
			return 1 //different
		}

		if A.IsNumber() || B.IsNumber() {
			ta := A.Number()
			tb := B.Number()
			return OsTrn(ta > tb, 1, 0) - OsTrn(ta < tb, 1, 0)
		}

		if A.IsText() || B.IsText() {
			return strings.Compare(A.String(), B.String())
		}

		return OsTrn(!A.Is() && !B.Is(), 0, 1) // both empty = 0(aka same)
	}

	return 1 // different
}
