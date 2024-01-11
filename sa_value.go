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
	"reflect"
	"sort"
	"strconv"
	"strings"
)

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

func (v *SAValue) SetBlob(val []byte) {
	v.value = val
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
	case []byte:
		if len(vv) > 0 && (vv[0] == '{' || vv[0] == '[') {
			return string(vv) //map or array
		}
		return "\"" + string(vv) + "\"" //binary/hex

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
	case []byte:
		return string(vv)

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

func (v *SAValue) Blob() []byte {
	switch vv := v.value.(type) {
	case string:
		return []byte(vv)
	case []byte:
		return vv
	}
	return nil
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

func (v *SAValue) IsBlob() bool {
	switch v.value.(type) {
	case []byte:
		return true
	}
	return false
}

func (a *SAValue) GetCd() OsCd {
	var v []int                  //change? ....................
	json.Unmarshal(a.Blob(), &v) //err? ...
	return InitOsCd32(uint32(v[0]), uint32(v[1]), uint32(v[2]), uint32(v[3]))
}

func (a *SAValue) GetV4() OsV4 {
	//type Xywh struct {
	//	X, Y, W, H int
	//}
	var v []int                  //change? ....................
	json.Unmarshal(a.Blob(), &v) //err? ...
	return InitOsV4(v[0], v[1], v[2], v[3])
	//return InitOsV4(v.X, v.Y, v.W, v.H)
}

func (v *SAValue) NumArrayItems() int {

	blob := v.Blob()
	if len(blob) == 0 {
		return 0
	}

	var arr []interface{}
	err := json.Unmarshal(blob, &arr)
	if err != nil {
		fmt.Printf("Warning: Array Unmarshal(%s) failed: %v", string(blob), err)
		return 0
	}
	return len(arr)
}

func (v *SAValue) NumMapItems() int {

	blob := v.Blob()
	if len(blob) == 0 {
		return 0
	}

	var arr map[string]interface{}
	err := json.Unmarshal(blob, &arr)
	if err != nil {
		fmt.Printf("Warning: Map Unmarshal(%s) failed: %v", string(blob), err)
		return 0
	}
	return len(arr)
}

func (v *SAValue) GetArrayItem(i int) *SAValue {
	var arr []interface{}
	err := json.Unmarshal(v.Blob(), &arr)
	if err != nil {
		return nil
	}

	if i >= len(arr) {
		return nil
	}

	return &SAValue{value: arr[i]}
}

func (v *SAValue) GetMapItem(i int) (string, *SAValue) {
	var arr map[string]interface{}
	err := json.Unmarshal(v.Blob(), &arr)
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
			return key, &SAValue{value: arr[key]}
		}
		ii++
	}

	return "", nil
}

func (v *SAValue) GetMapKey(key string) *SAValue {

	var arr map[string]interface{}
	err := json.Unmarshal(v.Blob(), &arr)
	if err != nil {
		return nil
	}

	item, found := arr[key]
	if found {
		return nil
	}

	ret, err := json.Marshal(item)
	if err != nil {
		return nil
	}

	return &SAValue{value: ret}
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

		//is blob? .......

		return OsTrn(!A.Is() && !B.Is(), 0, 1) // both null = 0
	}

	return 1 // different
}
