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

	accessAttr *SANodeAttr

	next *VmInstr

	pos_attr *SANodeAttr
	pos      OsV2
}

func NewVmInstr(exe VmInstr_callbackExecute, lexer *VmLexer, pos_attr *SANodeAttr) *VmInstr {
	var instr VmInstr

	instr.fn = exe
	instr.temp = InitSAValue()

	instr.pos_attr = pos_attr
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

func (instr *VmInstr) _lineReplace(line *string, value string) {
	if instr == nil {
		return
	}
	*line = (*line)[:instr.pos.X] + value + (*line)[instr.pos.Y:]
}

func (instr *VmInstr) LineReplace(value string) {
	if instr == nil {
		return
	}

	if value != "" {
		_, err := strconv.ParseFloat(value, 64)
		if err != nil {
			value = "\"" + value + "\""
		}
	}

	instr.pos_attr.Value = instr.pos_attr.Value[:instr.pos.X] + value + instr.pos_attr.Value[instr.pos.Y:] //replace

	instr.pos_attr.ParseExpresion()
	instr.pos_attr.ExecuteExpression()
}

func (instr *VmInstr) LineExtract(line string, value string) string {
	return line[instr.pos.X:instr.pos.Y]
}

func (instr *VmInstr) IsFnGui() *VmInstr {
	if VmCallback_Cmp(instr.fn, VmApi_GuiBool) ||
		VmCallback_Cmp(instr.fn, VmApi_GuiBool2) ||
		VmCallback_Cmp(instr.fn, VmApi_GuiCombo) ||
		VmCallback_Cmp(instr.fn, VmApi_GuiDate) ||
		VmCallback_Cmp(instr.fn, VmApi_GuiColor) {

		if len(instr.prms) >= 1 {
			return instr.prms[0].instr
		}
	}
	return nil
}

func (instr *VmInstr) isFnAccess() *VmInstr {

	if VmCallback_Cmp(instr.fn, VmBasic_Access) { //access(!)
		if instr.accessAttr != nil {
			return instr.accessAttr.instr
		}
	}

	if VmCallback_Cmp(instr.fn, VmApi_AccessArray) { //array(!)
		if len(instr.prms) >= 2 {
			acc := instr.prms[0].instr
			ind := instr.prms[1].instr

			if VmCallback_Cmp(acc.fn, VmBasic_Access) && acc.accessAttr != nil && VmCallback_Cmp(ind.fn, VmBasic_Constant) {
				arr_instr := acc.accessAttr.instr
				if arr_instr != nil {
					arr_i := int(ind.temp.Number())
					if arr_i < len(arr_instr.prms) {
						return arr_instr.prms[arr_i].instr
					}
				}
			}
		}
	}
	return nil
}

func (instr *VmInstr) GetConst() *VmInstr {
	if instr == nil {
		return nil
	}

	if VmCallback_Cmp(instr.fn, VmBasic_Constant) { //const(!)
		return instr
	}
	acc := instr.isFnAccess()
	if acc != nil {
		return acc.GetConst()
	}
	gui := instr.IsFnGui()
	if gui != nil {
		return gui.GetConst()
	}
	return nil
}
func (instr *VmInstr) GetConstArray() *VmInstr {
	if instr == nil {
		return nil
	}

	if VmCallback_Cmp(instr.fn, VmBasic_ConstArray) { //const(!)
		return instr
	}
	acc := instr.isFnAccess()
	if acc != nil {
		return acc.GetConstArray()
	}
	gui := instr.IsFnGui()
	if gui != nil {
		return gui.GetConstArray()
	}
	return nil
}
func (instr *VmInstr) GetConstTable() *VmInstr {
	if instr == nil {
		return nil
	}

	if VmCallback_Cmp(instr.fn, VmBasic_ConstTable) { //const(!)
		return instr
	}
	acc := instr.isFnAccess()
	if acc != nil {
		return acc.GetConstTable()
	}
	gui := instr.IsFnGui()
	if gui != nil {
		return gui.GetConstTable()
	}
	return nil
}

func (instr *VmInstr) GetConstArrayPrm(i int) *VmInstr {
	if instr == nil {
		return nil
	}

	instr = instr.GetConstArray()
	if instr != nil && i < len(instr.prms) {
		return instr.prms[i].instr.GetConst()
	}
	return nil
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
	instr.temp = instr.accessAttr.finalValue
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
	return instr.ExePrm(st, 0)
}

func VmApi_AccessArray(instr *VmInstr, st *VmST) SAValue {

	item := instr.ExePrm(st, 0)
	index := instr.ExePrm(st, 1)

	return *item.Array().Get(int(index.Number()))
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
			r = tb.AddRow()
		} else {
			tb.Get(c, r).value = v.value

			c++
			if c >= len(tb.names) {
				c = 0
				r = tb.AddRow()
			}
		}
	}

	instr.temp.SetTable(tb)
	return instr.temp
}
