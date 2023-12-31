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
	"strconv"
	"strings"
)

type SANodeAttr struct {
	node *SANode

	Name    string
	Value   string `json:",omitempty"`
	ShowExp bool

	finalValue   string
	instr        *VmInstr
	depends      []*SANodeAttr
	isDirectLink bool
	errExp       error
	errExe       error

	Gui_type     string `json:",omitempty"`
	Gui_options  string `json:",omitempty"`
	Gui_ReadOnly bool   `json:",omitempty"` //output

}

func (attr *SANodeAttr) IsExpression() bool {
	var found bool
	for found { //loop!
		attr.Value, found = strings.CutPrefix(attr.Value, " ")
	}

	return strings.HasPrefix(attr.Value, "=")
}

func (attr *SANodeAttr) getDirectLink_inner(orig *SANodeAttr) (*string, bool) {

	if attr.isDirectLink {
		if attr.depends[0] == orig {
			fmt.Println("Warning: infinite loop")
			return &attr.Value, true //avoid infinite loop
		}
		return attr.depends[0].getDirectLink_inner(orig) //go to source
	}

	if len(attr.depends) > 0 {
		return &attr.finalValue, false //expression. oldValue = result
	}

	return &attr.Value, true //this
}

func (attr *SANodeAttr) GetDirectLink() (*string, bool) {
	return attr.getDirectLink_inner(attr)
}
func (attr *SANodeAttr) SetString(value string) {
	val, editable := attr.GetDirectLink()
	if editable {
		*val = value
	}
}
func (attr *SANodeAttr) SetInt(value int) {
	val, editable := attr.GetDirectLink()
	if editable {
		*val = strconv.Itoa(value)
	}
}
func (attr *SANodeAttr) SetFloat(value float64) {
	val, editable := attr.GetDirectLink()
	if editable {
		*val = strconv.FormatFloat(value, 'f', -1, 64)
	}
}

func (attr *SANodeAttr) GetString() string {
	val, _ := attr.GetDirectLink()
	return *val
}
func (attr *SANodeAttr) GetInt() int {
	v, _ := strconv.Atoi(attr.GetString())
	return v
}
func (attr *SANodeAttr) GetInt64() int64 {
	v, _ := strconv.Atoi(attr.GetString())
	return int64(v)
}
func (attr *SANodeAttr) GetFloat() float64 {
	v, _ := strconv.ParseFloat(attr.GetString(), 64)
	return v
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

func (attr *SANodeAttr) ExecuteExpression() {
	if attr.errExp != nil {
		return
	}

	for _, dep := range attr.depends {
		if dep.node == attr.node {
			dep.ExecuteExpression() //self
		}
	}

	var val string
	if attr.instr != nil && !attr.isDirectLink {
		st := InitVmST()
		rec := attr.instr.Exe(nil, &st)
		val = rec.GetString()
	} else {
		value, _ := attr.GetDirectLink()
		val = *value
	}

	attr.finalValue = val
}

func (a *SANodeAttr) Cmp(b *SANodeAttr) bool {
	return a.Name == b.Name && a.Value == b.Value && a.Gui_type == b.Gui_type && a.Gui_options == b.Gui_options && a.Gui_ReadOnly == b.Gui_ReadOnly
}
