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

type VmInstr_callbackExecute func(self *VmInstr, rec *Rec, st *VmST) *Rec

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

	attr *SANodeAttr
	temp *Rec

	next *VmInstr

	pos OsV2
}

func NewVmInstr(exe VmInstr_callbackExecute, lexer *VmLexer) *VmInstr {
	var instr VmInstr

	instr.fn = exe
	instr.temp = NewRec()

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

func (instr *VmInstr) GetDirectDirectAccess() *VmInstr {

	if VmCallback_Cmp(instr.fn, VmBasic_Access) { //access(!)
		return instr
	}

	if VmCallback_Cmp(instr.fn, VmApi_GuiBool) ||
		VmCallback_Cmp(instr.fn, VmApi_GuiBool2) ||
		VmCallback_Cmp(instr.fn, VmApi_GuiCombo) ||
		VmCallback_Cmp(instr.fn, VmApi_GuiDate) ||
		VmCallback_Cmp(instr.fn, VmApi_GuiColor) {
		return instr.prms[0].instr.GetDirectDirectAccess()
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

func (instr *VmInstr) Exe(rec *Rec, st *VmST) *Rec {

	ret := rec

	for instr != nil {
		ret = instr.fn(instr, rec, st)
		instr = instr.next
	}

	if ret == nil {
		fmt.Println("This should never happen")
	}

	return ret
}

func (instr *VmInstr) ExePrm(rec *Rec, st *VmST, prm_i int) *Rec {
	return instr.prms[prm_i].instr.Exe(rec, st)
}

func (instr *VmInstr) ExePrmString(rec *Rec, st *VmST, prm_i int) string {

	rec = instr.ExePrm(rec, st, prm_i)
	return rec.value.String()
}
func (instr *VmInstr) ExePrmNumber(rec *Rec, st *VmST, prm_i int) float64 {

	rec = instr.ExePrm(rec, st, prm_i)
	return rec.value.Number()
}
func (instr *VmInstr) ExePrmInt(rec *Rec, st *VmST, prm_i int) int {
	return int(instr.ExePrmNumber(rec, st, prm_i))
}

func VmBasic_Constant(instr *VmInstr, rec *Rec, st *VmST) *Rec {
	return instr.temp
}
func VmBasic_Bracket(instr *VmInstr, rec *Rec, st *VmST) *Rec {
	return instr.ExePrm(rec, st, 0)
}

func VmBasic_Access(instr *VmInstr, rec *Rec, st *VmST) *Rec {
	instr.temp.value = instr.attr.finalValue
	return instr.temp
}

func VmApi_GuiBool(instr *VmInstr, rec *Rec, st *VmST) *Rec {
	return instr.ExePrm(rec, st, 0)
}
func VmApi_GuiBool2(instr *VmInstr, rec *Rec, st *VmST) *Rec {
	return instr.ExePrm(rec, st, 0)
}

func VmApi_GuiCombo(instr *VmInstr, rec *Rec, st *VmST) *Rec {
	ret := instr.ExePrm(rec, st, 0)
	instr.temp = instr.ExePrm(rec, st, 1) //save options into temp
	return ret
}
func VmApi_GuiDate(instr *VmInstr, rec *Rec, st *VmST) *Rec {
	return instr.ExePrm(rec, st, 0)
}

func VmApi_GuiColor(instr *VmInstr, rec *Rec, st *VmST) *Rec {
	return VmBasic_ConstArray(instr, rec, st)
}

func VmBasic_ConstArray(instr *VmInstr, rec *Rec, st *VmST) *Rec {
	var arr SAValueArray
	arr.Resize(len(instr.prms))
	for i := range instr.prms {
		arr.Get(i).value = instr.ExePrm(rec, st, i).value.value
	}
	instr.temp.value.SetArray(&arr)
	return instr.temp
}

func VmBasic_ConstTable(instr *VmInstr, rec *Rec, st *VmST) *Rec {
	tb := NewSAValueTable(nil)
	c := 0
	r := 0
	for i := range instr.prms {
		v := instr.ExePrm(rec, st, i).value
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

	instr.temp.value.SetTable(tb)
	return instr.temp
}
