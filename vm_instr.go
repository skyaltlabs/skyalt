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

	lineX OsV2
}

type VmInstr struct {
	fn VmInstr_callbackExecute

	prms []VmInstrPrm

	attr *SANodeAttr
	temp *Rec

	next *VmInstr
}

func NewVmInstr(exe VmInstr_callbackExecute) *VmInstr {
	var instr VmInstr

	instr.fn = exe
	instr.temp = NewRec()

	return &instr
}

func (instr *VmInstr) IsRunning(st *VmST) bool {
	return st.running
}

func (instr *VmInstr) IsDirectLink() bool {
	return VmCallback_Cmp(instr.fn, VmBasic_Access)
}

func (instr *VmInstr) NumPrms() int {
	return len(instr.prms)
}

func (instr *VmInstr) AddPropInstr(add *VmInstr) int {
	var t VmInstrPrm
	t.instr = add
	t.lineX = OsV2{-1, -1}

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

func (instr *VmInstr) ExePrmNumber(rec *Rec, st *VmST, prm_i int) float64 {

	rec = instr.ExePrm(rec, st, prm_i)
	return rec.GetNumber()
}
func (instr *VmInstr) ExePrmInt(rec *Rec, st *VmST, prm_i int) int {

	rec = instr.ExePrm(rec, st, prm_i)
	return rec.GetInt()
}

func (instr *VmInstr) ExePrmString(rec *Rec, st *VmST, prm_i int) string {

	rec = instr.ExePrm(rec, st, prm_i)
	return rec.GetString()
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
