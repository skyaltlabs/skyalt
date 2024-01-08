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
)

type SANodeAttr struct {
	node *SANode

	Name    string
	Value   string `json:",omitempty"` //every(!) value is expression
	ShowExp bool

	result  SAValue
	instr   *VmInstr
	depends []*SANodeAttr

	errExp error
	errExe error

	useMark bool
}

func (attr *SANodeAttr) SetErrorExe(err string) {
	attr.errExe = errors.New(err)
}

func (attr *SANodeAttr) SetExpString(value string) {
	instr := attr.instr.GetConst()
	if instr != nil { //editable
		instr.LineReplace(value)
	}
}
func (attr *SANodeAttr) SetExpInt(value int) {
	attr.SetExpString(strconv.Itoa(value))
}
func (attr *SANodeAttr) SetExpBool(value bool) {
	attr.SetExpString(OsTrnString(value, "1", "0"))
}
func (attr *SANodeAttr) SetExpFloat(value float64) {
	attr.SetExpString(strconv.FormatFloat(value, 'f', -1, 64))
}

func (attr *SANodeAttr) _getFinalValue() SAValue {
	if attr.instr != nil {
		instr := attr.instr.GetConst()
		if instr != nil {
			return instr.pos_attr.result
		}
	}

	return SAValue{}
}

func (attr *SANodeAttr) GetString() string {
	v := attr._getFinalValue()
	return v.String()
}
func (attr *SANodeAttr) GetInt() int {
	v := attr._getFinalValue()
	return int(v.Number())
}
func (attr *SANodeAttr) GetInt64() int64 {
	v := attr._getFinalValue()
	return int64(v.Number())
}
func (attr *SANodeAttr) GetFloat() float64 {
	v := attr._getFinalValue()
	return v.Number()
}
func (attr *SANodeAttr) GetBool() bool {
	return attr.GetInt() != 0
}
func (attr *SANodeAttr) GetByte() byte {
	return byte(attr.GetInt())
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
	if attr.errExp != nil {
		return
	}

	for _, dep := range attr.depends {
		if dep.node == attr.node {
			dep.ExecuteExpression() //self
		}
	}

	if attr.instr != nil {
		st := InitVmST()
		attr.result = attr.instr.Exe(&st)
	} else {
		attr.result.SetString(attr.Value)
	}
}

func (a *SANodeAttr) Cmp(b *SANodeAttr) bool {
	return a.Name == b.Name && a.Value == b.Value
}

func (a *SANodeAttr) GetCd() OsCd {
	return a.result.Array().GetCd()
}
func (a *SANodeAttr) SetCd(cd OsCd) {
	a.result.Array().SetCd(cd)
}

func (a *SANodeAttr) ReplaceArrayItem(prm_i int, value string) {
	if a == nil {
		return
	}
	instr := a.instr.GetConstArrayPrm(prm_i)
	if instr != nil {
		instr.LineReplace(value)
	}
}
func (a *SANodeAttr) ReplaceArrayItemInt(prm_i int, value int) {
	a.ReplaceArrayItem(prm_i, strconv.Itoa(value))
}

func (a *SANodeAttr) ReplaceCd(cd OsCd) {
	if a == nil {
		return
	}

	oldCd := a.GetCd()

	if oldCd.A != cd.A {
		a.ReplaceArrayItemInt(3, int(cd.A))
	}
	if oldCd.B != cd.B {
		a.ReplaceArrayItemInt(2, int(cd.B))
	}
	if oldCd.G != cd.G {
		a.ReplaceArrayItemInt(1, int(cd.G))
	}
	if oldCd.R != cd.R {
		a.ReplaceArrayItemInt(0, int(cd.R))
	}
}
