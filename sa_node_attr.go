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
	"errors"
	"fmt"
	"strconv"
	"strings"
)

//{"fn": "switch"}
//{"fn": "combo", "prm": "a;b;c", "prm2": "0;1;2"}
//{"fn": "map", "map": {"column": {"fn":"switch"}, "value": {"fn":"switch"}}}

var SAAttrUi_SWITCH = SAAttrUiValue{Fn: "switch"}
var SAAttrUi_CHECKBOX = SAAttrUiValue{Fn: "checkbox"}
var SAAttrUi_DATE = SAAttrUiValue{Fn: "date"}
var SAAttrUi_COLOR = SAAttrUiValue{Fn: "color"}
var SAAttrUi_BLOB = SAAttrUiValue{Fn: "blob"}

func SAAttrUi_COMBO(names string, values string) SAAttrUiValue {
	return SAAttrUiValue{Fn: "combo", Prm: names, Prm2: values}
}

type SAAttrUiValue struct {
	Fn         string                   `json:",omitempty"`
	Prm        string                   `json:",omitempty"` //can be SAAttrUiMap
	Prm2       string                   `json:",omitempty"`
	Map        map[string]SAAttrUiValue `json:",omitempty"`
	HideAddDel bool
}

type SANodeAttr struct {
	node *SANode

	Name    string //output attr start with '_'
	Value   string `json:",omitempty"` //every(!) value is expression
	ShowExp bool
	Ui      SAAttrUiValue `json:",omitempty"`

	defaultValue string

	instr   *VmInstr
	depends []*SANodeAttr

	errExp error
	errExe error

	exeMark bool
}

func (attr *SANodeAttr) IsOutput() bool {
	return strings.HasPrefix(attr.Name, "_")
}

func (attr *SANodeAttr) CheckUniqueName() {
	if attr.Name == "" {
		attr.Name = "attr"
	}
	attr.Name = strings.ReplaceAll(attr.Name, ".", "") //remove all '.'

	for attr.node.NumAttrNames(attr.Name) >= 2 {
		attr.Name += "1"
	}
}

func (attr *SANodeAttr) SetErrorExe(err string) {
	attr.errExe = errors.New(err)
}

func (attr *SANodeAttr) SetExpString(value string, mapOrArray bool) {
	instr := attr.instr.GetConst()
	if instr != nil { //editable
		instr.LineReplace(value, mapOrArray)
	}
}
func (attr *SANodeAttr) SetExpInt(value int) {
	attr.SetExpString(strconv.Itoa(value), false)
}
func (attr *SANodeAttr) SetExpBool(value bool) {
	attr.SetExpString(OsTrnString(value, "1", "0"), false)
}
func (attr *SANodeAttr) SetExpFloat(value float64) {
	attr.SetExpString(strconv.FormatFloat(value, 'f', -1, 64), false)
}

func (attr *SANodeAttr) GetResult() *SAValue {
	if attr.instr == nil {
		attr.instr = NewVmInstr(VmBasic_Constant, nil, attr)
	}
	return &attr.instr.temp
}

func (attr *SANodeAttr) GetString() string {
	return attr.GetResult().String()
}
func (attr *SANodeAttr) GetInt() int {
	return int(attr.GetResult().Number())
}
func (attr *SANodeAttr) GetInt64() int64 {
	return int64(attr.GetResult().Number())
}
func (attr *SANodeAttr) GetFloat() float64 {
	return attr.GetResult().Number()
}
func (attr *SANodeAttr) GetBool() bool {
	return attr.GetInt() != 0
}
func (attr *SANodeAttr) GetBlob() OsBlob {
	return attr.GetResult().Blob()
}
func (attr *SANodeAttr) IsText() bool {
	return attr.GetResult().IsText()
}
func (attr *SANodeAttr) IsNumber() bool {
	return attr.GetResult().IsNumber()
}
func (attr *SANodeAttr) IsBlob() bool {
	return attr.GetResult().IsBlob()
}

func (attr *SANodeAttr) GetCd() OsCd {
	return attr.GetResult().Cd()
}
func (attr *SANodeAttr) GetV2() OsV2 {
	return attr.GetResult().V2()
}
func (attr *SANodeAttr) GetV2f() OsV2f {
	return attr.GetResult().V2f()
}
func (attr *SANodeAttr) GetV4() OsV4 {
	return attr.GetResult().V4()
}

func (attr *SANodeAttr) NumMapItems() int {
	return attr.GetResult().NumMapItems()
}
func (attr *SANodeAttr) NumArrayItems() int {
	return attr.GetResult().NumArrayItems()
}

func (attr *SANodeAttr) GetArrayItem(i int) *SAValue {
	return attr.GetResult().GetArrayItem(i)
}

func (attr *SANodeAttr) GetMapItem(i int) (string, *SAValue) {
	return attr.GetResult().GetMapItem(i)
}

func (attr *SANodeAttr) SetOutBlob(blob []byte) {
	attr.GetResult().SetBlob(blob)
}

func (attr *SANodeAttr) CheckForLoopAttr(find *SANodeAttr) {
	for _, dep := range attr.depends {
		if dep == find {
			dep.errExp = fmt.Errorf("Loop")
			continue //avoid infinite recursion
		}
		dep.CheckForLoopAttr(find)
	}

}

func (attr *SANodeAttr) ParseExpresion() {

	attr.instr = nil
	attr.depends = nil
	attr.errExp = nil

	app := attr.node.app
	if attr.Value != "" {
		ln, err := InitVmLine(attr.Value, app.ops, app.apis, app.prior, attr)
		if err == nil {
			attr.instr = ln.Parse()
			//attr.depends = ln.depends
			if len(ln.errs) > 0 {
				attr.errExp = errors.New(ln.errs[0])
			}
		} else {
			attr.errExp = err
		}
	}

}

func (attr *SANodeAttr) ExecuteExpression() {

	if attr.IsOutput() {
		return
	}

	if attr.errExp != nil {
		return
	}

	for _, dep := range attr.depends {
		if dep.node == attr.node {
			dep.ExecuteExpression() //self
		}
	}

	st := InitVmST()
	attr.instr.Exe(&st)
}

func (a *SANodeAttr) Cmp(b *SANodeAttr) bool {
	return a.Name == b.Name && a.Value == b.Value
}

func (a *SANodeAttr) ReplaceArrayItemValue(prm_i int, value string) {
	if a == nil {
		return
	}
	instr := a.instr.GetConstArrayPrm(prm_i)
	if instr != nil {
		instr.LineReplace(value, false)
	}
}
func (a *SANodeAttr) ReplaceArrayItemValueInt(prm_i int, value int) {
	a.ReplaceArrayItemValue(prm_i, strconv.Itoa(value))
}

func (a *SANodeAttr) ReplaceMapItemValue(prm_i int, value string) {
	if a == nil {
		return
	}
	_, instr := a.instr.GetConstMapPrm(prm_i)
	if instr != nil {
		instr.LineReplace(value, false)
	}
}
func (a *SANodeAttr) ReplaceMapItemKey(prm_i int, key string) {
	if a == nil {
		return
	}
	key_instr, _ := a.instr.GetConstMapPrm(prm_i)
	if key_instr != nil {
		key_instr.LineReplace(key, false)
	}
}

func (a *SANodeAttr) ReplaceCd(cd OsCd) {
	if a == nil {
		return
	}

	oldCd := a.GetCd()

	if oldCd.A != cd.A {
		a.ReplaceArrayItemValueInt(3, int(cd.A))
	}
	if oldCd.B != cd.B {
		a.ReplaceArrayItemValueInt(2, int(cd.B))
	}
	if oldCd.G != cd.G {
		a.ReplaceArrayItemValueInt(1, int(cd.G))
	}
	if oldCd.R != cd.R {
		a.ReplaceArrayItemValueInt(0, int(cd.R))
	}
}

func (a *SANodeAttr) AddParamsItem(itemVal string, isMap bool) {
	n := 0
	if isMap {
		n = a.NumMapItems()
	} else {
		n = a.NumArrayItems()
	}

	if n > 0 {
		pe := a.instr.prms[n-1].value.pos.Y
		a.Value = a.Value[:pe] + ", " + itemVal + a.Value[pe:]
	} else {
		a.Value = OsTrnString(isMap, "{"+itemVal+"}", "["+itemVal+"]")
	}
}

func (a *SANodeAttr) RemoveParamsItem(isMap bool) {
	n := 0
	if isMap {
		n = a.NumMapItems()
	} else {
		n = a.NumArrayItems()
	}

	if n > 1 {
		p := a.instr.prms[n-1].value.pos

		stValue := a.Value[:p.X]
		comma := strings.LastIndexByte(stValue, ',')
		if comma >= 0 {
			stValue = stValue[:comma]
		}
		a.Value = stValue + a.Value[p.Y:]
	} else {
		a.Value = OsTrnString(isMap, "{}", "[]")
	}
}
