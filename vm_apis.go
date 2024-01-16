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
	"math"
	"strings"
)

func VmApi_If(instr *VmInstr, st *VmST) SAValue {
	o := instr.ExePrm(st, 0)
	return instr.ExePrm(st, OsTrn(o.Is(), 1, 2))
}

func VmApi_Not(instr *VmInstr, st *VmST) SAValue {

	o := instr.ExePrm(st, 0)
	instr.temp.SetBool(!o.Is())
	return instr.temp
}

func VmApi_Min(instr *VmInstr, st *VmST) SAValue {

	a := instr.ExePrmNumber(st, 0)
	b := instr.ExePrmNumber(st, 1)

	instr.temp.SetNumber(OsMinFloat(a, b))
	return instr.temp
}
func VmApi_Max(instr *VmInstr, st *VmST) SAValue {

	a := instr.ExePrmNumber(st, 0)
	b := instr.ExePrmNumber(st, 1)

	instr.temp.SetNumber(OsMaxFloat(a, b))
	return instr.temp
}

func VmApi_Clamp(instr *VmInstr, st *VmST) SAValue {

	v := instr.ExePrmNumber(st, 0)
	mi := instr.ExePrmNumber(st, 1)
	mx := instr.ExePrmNumber(st, 2)

	instr.temp.SetNumber(OsClampFloat(v, mi, mx))
	return instr.temp
}

func VmApi_Sqrt(instr *VmInstr, st *VmST) SAValue {
	v := instr.ExePrmNumber(st, 0)
	if v >= 0 {
		v = math.Sqrt(v)
	} else {
		instr.temp.SetNumber(0)
		v = 0
		fmt.Println("Sqrl from zero") //err ...
	}
	instr.temp.SetNumber(v)
	return instr.temp
}

func VmApi_Pow(instr *VmInstr, st *VmST) SAValue {
	v := instr.ExePrmNumber(st, 0)
	exp := instr.ExePrmNumber(st, 1)

	instr.temp.SetNumber(math.Pow(v, exp))
	return instr.temp
}

func VmApi_Pi(instr *VmInstr, st *VmST) SAValue {

	instr.temp.SetNumber(math.Pi)
	return instr.temp
}
func VmApi_Sin(instr *VmInstr, st *VmST) SAValue {
	v := instr.ExePrmNumber(st, 0)

	instr.temp.SetNumber(math.Sin(v))
	return instr.temp
}
func VmApi_Cos(instr *VmInstr, st *VmST) SAValue {
	v := instr.ExePrmNumber(st, 0)

	instr.temp.SetNumber(math.Cos(v))
	return instr.temp
}

func VmApi_Tan(instr *VmInstr, st *VmST) SAValue {
	v := instr.ExePrmNumber(st, 0)

	instr.temp.SetNumber(math.Tan(v))
	return instr.temp
}
func VmApi_ATan(instr *VmInstr, st *VmST) SAValue {
	v := instr.ExePrmNumber(st, 0)

	instr.temp.SetNumber(math.Atan(v))
	return instr.temp
}
func VmApi_Log(instr *VmInstr, st *VmST) SAValue {
	v := instr.ExePrmNumber(st, 0)

	instr.temp.SetNumber(math.Log(v))
	return instr.temp
}
func VmApi_Exp(instr *VmInstr, st *VmST) SAValue {
	v := instr.ExePrmNumber(st, 0)

	instr.temp.SetNumber(math.Exp(v))
	return instr.temp
}

type VmApi struct {
	prior int
	name  string

	prms int

	fn VmInstr_callbackExecute
}

type VmApis struct {
	apis []VmApi
}

func (apis *VmApis) FindName(line string) *VmApi {

	for _, op := range apis.apis {

		if strings.EqualFold(line, op.name) {
			return &op
		}
	}
	return nil
}

func (apis *VmApis) _Add(item VmApi) {
	apis.apis = append(apis.apis, item)
}

func NewVmApis() *VmApis {
	var apis VmApis

	apis._Add(VmApi{0, "if", 3, VmApi_If})
	apis._Add(VmApi{0, "not", 1, VmApi_Not})

	apis._Add(VmApi{0, "uiCheck", 1, VmApi_UiCheckbox})
	apis._Add(VmApi{0, "uiSwitch", 1, VmApi_UiSwitch})
	apis._Add(VmApi{0, "uiCombo", 2, VmApi_UiCombo})
	apis._Add(VmApi{0, "uiDate", 1, VmApi_UiDate})
	apis._Add(VmApi{0, "uiColor", 1, VmApi_UiColor})
	apis._Add(VmApi{0, "uiBlob", 1, VmApi_UiBlob})

	apis._Add(VmApi{0, "get", 2, VmApi_Get})
	apis._Add(VmApi{0, "array", 2, VmApi_Array})
	apis._Add(VmApi{0, "compress", 1, VmApi_Compress})

	apis._Add(VmApi{0, "min", 2, VmApi_Min})
	apis._Add(VmApi{0, "max", 2, VmApi_Max})
	apis._Add(VmApi{0, "clamp", 3, VmApi_Clamp})

	apis._Add(VmApi{0, "sqrt", 1, VmApi_Sqrt})
	apis._Add(VmApi{0, "pow", 2, VmApi_Pow})

	apis._Add(VmApi{0, "pi", 0, VmApi_Pi})
	apis._Add(VmApi{0, "sin", 1, VmApi_Sin})
	apis._Add(VmApi{0, "cos", 1, VmApi_Cos})
	apis._Add(VmApi{0, "tan", 1, VmApi_Tan})
	apis._Add(VmApi{0, "atan", 1, VmApi_ATan})
	apis._Add(VmApi{0, "log", 1, VmApi_Log})
	apis._Add(VmApi{0, "exp", 1, VmApi_Exp})

	return &apis
}
