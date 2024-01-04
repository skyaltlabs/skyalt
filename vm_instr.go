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
	"strings"
)

type VmInstr_callbackExecute func(instr *VmInstr, st *VmST) SAValue

func VmCallback_Cmp(a VmInstr_callbackExecute, b VmInstr_callbackExecute) bool {
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

type VmST struct {
	running bool
}

func InitVmST() VmST {
	var st VmST
	st.running = true
	return st
}

type VmInstrPrm struct {
	instr *VmInstr
}

type VmInstr struct {
	parent *VmInstr
	fn     VmInstr_callbackExecute

	prms []VmInstrPrm

	temp SAValue

	attr *SANodeAttr

	next *VmInstr

	pos OsV2
}

func NewVmInstr(exe VmInstr_callbackExecute, lexer *VmLexer) *VmInstr {
	var instr VmInstr

	instr.fn = exe
	instr.temp = InitSAValue()

	instr.pos = OsV2{lexer.start, lexer.end}

	return &instr
}

func (instr *VmInstr) RenameAccessNode(line string, oldName, newName string) string {
	if VmCallback_Cmp(instr.fn, VmBasic_Access) {
		spl := strings.Split(line[instr.pos.X:instr.pos.Y], ".")
		if len(spl) > 0 && spl[0] == oldName {
			line = line[:instr.pos.X] + newName + line[instr.pos.X+len(spl[0]):]
		}
	}

	for _, prm := range instr.prms {
		if prm.instr != nil {
			line = prm.instr.RenameAccessNode(line, oldName, newName)
		}
	}

	return line
}

func (instr *VmInstr) IsRunning(st *VmST) bool {
	return st.running
}

func (instr *VmInstr) LineReplace(line *string, value string) {
	if instr == nil {
		return
	}
	*line = (*line)[:instr.pos.X] + value + (*line)[instr.pos.Y:]
}
func (instr *VmInstr) LineExtract(line string, value string) string {
	return line[instr.pos.X:instr.pos.Y]
}

func (instr *VmInstr) GetConst() *VmInstr {

	if VmCallback_Cmp(instr.fn, VmBasic_Constant) { //const(!)
		return instr
	}

	if VmCallback_Cmp(instr.fn, VmApi_GuiBool) ||
		VmCallback_Cmp(instr.fn, VmApi_GuiBool2) ||
		VmCallback_Cmp(instr.fn, VmApi_GuiCombo) ||
		VmCallback_Cmp(instr.fn, VmApi_GuiDate) ||
		VmCallback_Cmp(instr.fn, VmApi_GuiColor) {
		return instr.prms[0].instr.GetConst()
	}

	return nil
}

func (instr *VmInstr) GetDirectDirectAccess() (*VmInstr, *VmInstr) {

	if VmCallback_Cmp(instr.fn, VmBasic_Access) { //access(!)
		if instr.attr != nil {
			return instr, nil
		}
	}

	if VmCallback_Cmp(instr.fn, VmApi_AccessArray) { //array(!)
		if len(instr.prms) >= 2 {
			acc := instr.prms[0].instr
			ind := instr.prms[1].instr

			if VmCallback_Cmp(acc.fn, VmBasic_Access) && acc.attr != nil && VmCallback_Cmp(ind.fn, VmBasic_Constant) {
				return nil, instr
			}
		}

	}

	if VmCallback_Cmp(instr.fn, VmApi_GuiBool) ||
		VmCallback_Cmp(instr.fn, VmApi_GuiBool2) ||
		VmCallback_Cmp(instr.fn, VmApi_GuiCombo) ||
		VmCallback_Cmp(instr.fn, VmApi_GuiDate) ||
		VmCallback_Cmp(instr.fn, VmApi_GuiColor) {
		return instr.prms[0].instr.GetDirectDirectAccess()
	}

	return nil, nil
}

func (instr *VmInstr) NumPrms() int {
	return len(instr.prms)
}

func (instr *VmInstr) AddPropInstr(add *VmInstr) int {

	add.parent = instr

	var t VmInstrPrm
	t.instr = add

	instr.prms = append(instr.prms, t)

	return instr.NumPrms() - 1
}

func (instr *VmInstr) Exe(st *VmST) SAValue {

	var ret SAValue

	for instr != nil {
		ret = instr.fn(instr, st)
		instr = instr.next
	}

	return ret
}

func (instr *VmInstr) ExePrm(st *VmST, prm_i int) SAValue {
	return instr.prms[prm_i].instr.Exe(st)
}

func (instr *VmInstr) ExePrmString(st *VmST, prm_i int) string {
	rec := instr.ExePrm(st, prm_i)
	return rec.String()
}
func (instr *VmInstr) ExePrmNumber(st *VmST, prm_i int) float64 {

	rec := instr.ExePrm(st, prm_i)
	return rec.Number()
}
func (instr *VmInstr) ExePrmInt(st *VmST, prm_i int) int {
	return int(instr.ExePrmNumber(st, prm_i))
}

func VmBasic_Constant(instr *VmInstr, st *VmST) SAValue {
	return instr.temp
}
func VmBasic_Bracket(instr *VmInstr, st *VmST) SAValue {
	return instr.ExePrm(st, 0)
}

func VmBasic_Access(instr *VmInstr, st *VmST) SAValue {
	instr.temp = instr.attr.finalValue
	return instr.temp
}

func VmApi_GuiBool(instr *VmInstr, st *VmST) SAValue {
	return instr.ExePrm(st, 0)
}
func VmApi_GuiBool2(instr *VmInstr, st *VmST) SAValue {
	return instr.ExePrm(st, 0)
}

func VmApi_GuiCombo(instr *VmInstr, st *VmST) SAValue {
	ret := instr.ExePrm(st, 0)
	instr.temp = instr.ExePrm(st, 1) //save options into temp
	return ret
}
func VmApi_GuiDate(instr *VmInstr, st *VmST) SAValue {
	return instr.ExePrm(st, 0)
}
func VmApi_GuiColor(instr *VmInstr, st *VmST) SAValue {
	return VmBasic_ConstArray(instr, st)
}

func VmApi_AccessArray(instr *VmInstr, st *VmST) SAValue {

	item := instr.ExePrm(st, 0)
	index := instr.ExePrm(st, 1)

	return *item.Array().Get(int(index.Number()))

	//instr.attr_item = int(index.Number())
	//return instr.temp
}

func VmBasic_ConstArray(instr *VmInstr, st *VmST) SAValue {
	var arr SAValueArray
	arr.Resize(len(instr.prms))
	for i := range instr.prms {
		arr.Get(i).value = instr.ExePrm(st, i).value
	}
	instr.temp.SetArray(&arr)
	return instr.temp
}

func VmBasic_ConstTable(instr *VmInstr, st *VmST) SAValue {
	tb := NewSAValueTable(nil)
	c := 0
	r := 0
	for i := range instr.prms {
		v := instr.ExePrm(st, i)
		if i == 0 {
			tb = NewSAValueTable(strings.Split(v.String(), ";"))
		} else {
			tb.Get(c, r).value = v.value

			c++
			if c >= len(tb.names) {
				c = 0
				r++
			}
		}
	}

	instr.temp.SetTable(tb)
	return instr.temp
}
