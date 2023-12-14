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

type Rec struct {
	value interface{}
}

func NewRec() *Rec {
	var rec Rec
	rec.value = ""
	return &rec
}

func (rec *Rec) Is() bool {

	if rec == nil {
		return false
	}

	switch val := rec.value.(type) {
	case float64:
		return val != 0
	case string:
		return len(val) > 0
	}

	return false
}

func (rec *Rec) IsNumber() bool {
	switch rec.value.(type) {
	case float64:
		return true
	}
	return false
}

func (rec *Rec) IsText() bool {
	switch rec.value.(type) {
	case string:
		return true
	}
	return false
}

func (rec *Rec) SetNumber(value float64) {
	rec.value = value
}
func (rec *Rec) SetInt(value int) {
	rec.SetNumber(float64(value))
}
func (rec *Rec) SetBool(value bool) {
	rec.SetNumber(OsTrnFloat(value, 1, 0))
}
func (rec *Rec) SetString(value string) {
	rec.value = value
}

func (rec *Rec) GetNumber() float64 {

	switch val := rec.value.(type) {
	case float64:
		return val
	case string:
		v := float64(0)
		if len(val) != 0 {
			var err error
			v, err = strconv.ParseFloat(val, 64)
			if err != nil {
				//fmt.Println("ERROR: Converting string . float")
				return 0
			}
		}
		return v
	}

	return 0
}
func (rec *Rec) GetInt() int {
	return int(rec.GetNumber())
}
func (rec *Rec) GetBool() bool {
	return rec.GetNumber() != 0
}
func (rec *Rec) GetString() string {

	switch val := rec.value.(type) {
	case float64:
		if val == float64(int(val)) {
			return fmt.Sprintf("%d", int(val))
		}
		return strconv.FormatFloat(val, 'f', -1, 64)
	case string:
		return string(val)
	}

	return ""
}

func (A *Rec) Cmp(B *Rec) int {
	if A != nil && B != nil {
		if reflect.TypeOf(A.value) == reflect.TypeOf(B.value) {
			switch val := A.value.(type) {
			case float64:
				ta := val
				tb := B.GetNumber()
				return OsTrn(ta > tb, 1, 0) - OsTrn(ta < tb, 1, 0)
			case string:
				return strings.Compare(val, B.GetString())
			}
		} else {
			if A.IsNumber() || B.IsNumber() {
				ta := A.GetNumber()
				tb := B.GetNumber()
				return OsTrn(ta > tb, 1, 0) - OsTrn(ta < tb, 1, 0)
			}

			if A.IsText() && B.IsText() {
				return strings.Compare(A.GetString(), B.GetString())
			}

			return OsTrn(!A.Is() && !B.Is(), 0, 1) // both null = 0
		}
	}

	return 1 // different
}
