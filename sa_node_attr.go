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
	Value   string `json:",omitempty"` //every value is expression
	ShowExp bool

	finalValue SAValue
	instr      *VmInstr
	depends    []*SANodeAttr

	errExp error
	errExe error
}

func (attr *SANodeAttr) SetErrorExe(err string) {
	attr.errExe = errors.New(err)
}

func (attr *SANodeAttr) getDirectLink_inner(orig *SANodeAttr) (*SANodeAttr, *VmInstr) {

	if attr.instr != nil {
		link := attr.instr.GetDirectDirectAccess()
		if link != nil && link.attr != nil {
			if link.attr == orig {
				fmt.Println("Warning: infinite loop")
				return attr, nil //avoid infinite loop
			}
			return link.attr.getDirectLink_inner(orig) //go to source
		}
	}

	if attr.instr != nil {
		return attr, attr.instr.GetConst()
	}

	return attr, nil //err
}

func (attr *SANodeAttr) GetDirectLink() (*SANodeAttr, *VmInstr) {
	return attr.getDirectLink_inner(attr)
}

func (attr *SANodeAttr) GetArrayDirectLink(i int) (*SANodeAttr, *VmInstr) {

	if attr.instr != nil && i < len(attr.instr.prms) {
		prm := attr.instr.prms[i]
		link := prm.instr.GetDirectDirectAccess()
		if link != nil && link.attr != nil {
			if link.attr == attr {
				fmt.Println("Warning: infinite loop")
				return attr, nil //avoid infinite loop
			}
			return link.attr.getDirectLink_inner(attr) //go to source
		}

		if prm.instr != nil {
			return attr, prm.instr.GetConst()
		}
	}
	return attr, nil //err
}

func (attr *SANodeAttr) SetExpString(value string) {
	a, instr := attr.GetDirectLink()
	if instr != nil { //editable
		a.LineReplace(instr, value)
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

func (attr *SANodeAttr) GetString() string {
	a, _ := attr.GetDirectLink()
	return a.finalValue.String()
}
func (attr *SANodeAttr) GetInt() int {
	a, _ := attr.GetDirectLink()
	return int(a.finalValue.Number())
}
func (attr *SANodeAttr) GetInt64() int64 {
	a, _ := attr.GetDirectLink()
	return int64(a.finalValue.Number())
}
func (attr *SANodeAttr) GetFloat() float64 {
	a, _ := attr.GetDirectLink()
	return a.finalValue.Number()
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
		ln, err := InitVmLine(attr.Value, app.ops, app.apis, app.prior, attr.node)
		if err == nil {
			attr.instr = ln.Parse()
			attr.depends = ln.depends
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
		rec := attr.instr.Exe(nil, &st)
		attr.finalValue = rec.value
	} else {
		attr.finalValue.SetString(attr.Value)
	}
}

func (a *SANodeAttr) Cmp(b *SANodeAttr) bool {
	return a.Name == b.Name && a.Value == b.Value
}

func (a *SANodeAttr) GetCd() OsCd {
	return a.finalValue.Array().GetCd()
}
func (a *SANodeAttr) SetCd(cd OsCd) {
	a.finalValue.Array().SetCd(cd)
}

func (a *SANodeAttr) LineReplace(instr *VmInstr, value string) {
	if value != "" {
		_, err := strconv.ParseFloat(value, 64)
		if err != nil {
			value = "\"" + value + "\""
		}
	}

	instr.LineReplace(&a.Value, value)
	a.ParseExpresion()
	a.ExecuteExpression()
}
