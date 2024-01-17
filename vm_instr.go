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
	key   *VmInstr
	value *VmInstr
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
		if prm.value != nil {
			line = prm.value.RenameAccessNode(line, oldName, newName)
		}
	}

	return line
}

func (instr *VmInstr) IsRunning(st *VmST) bool {
	return st.running
}

func (instr *VmInstr) LineReplace(value string) {
	if instr == nil {
		return
	}

	if value != "" {
		value = OsText_PrintToRaw(value)

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

func (instr *VmInstr) isFnAccess() *VmInstr {

	if VmCallback_Cmp(instr.fn, VmBasic_Access) { //access(!)
		if instr.accessAttr != nil {
			return instr.accessAttr.instr
		}
	}

	if VmCallback_Cmp(instr.fn, VmApi_Get) { //map or array(!)
		if len(instr.prms) >= 2 {
			acc := instr.prms[0].value
			key := instr.prms[1].value //index or key

			if VmCallback_Cmp(acc.fn, VmBasic_Access) && acc.accessAttr != nil && VmCallback_Cmp(key.fn, VmBasic_Constant) {
				arr_instr := acc.accessAttr.instr
				if arr_instr != nil {

					//map
					if key.temp.IsText() {
						arr_key := key.temp.String()
						for i, prm := range arr_instr.prms {
							if prm.key != nil && prm.key.temp.String() == arr_key {
								return arr_instr.prms[i].value
							}
						}
					}

					//array
					if key.temp.IsNumber() {
						arr_i := int(key.temp.Number())
						if arr_i < len(arr_instr.prms) {
							return arr_instr.prms[arr_i].value
						}
					}
				}
			}
		}
	}

	/*if VmCallback_Cmp(instr.fn, VmApi_AccessMap) { //map(!)
		if len(instr.prms) >= 2 {
			acc := instr.prms[0].value
			key := instr.prms[1].value

			if VmCallback_Cmp(acc.fn, VmBasic_Access) && acc.accessAttr != nil && VmCallback_Cmp(key.fn, VmBasic_Constant) {
				arr_instr := acc.accessAttr.instr
				if arr_instr != nil {
					arr_key := key.temp.String()
					for i, prm := range arr_instr.prms {
						if prm.key.temp.String() == arr_key {
							return arr_instr.prms[i].value
						}
					}
				}
			}
		}
	}*/

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

	return nil
}
func (instr *VmInstr) GetConstArray() *VmInstr {
	if instr == nil {
		return nil
	}

	if VmCallback_Cmp(instr.fn, VmBasic_InitArray) { //const(!)
		return instr
	}
	acc := instr.isFnAccess()
	if acc != nil {
		return acc.GetConstArray()
	}

	return nil
}
func (instr *VmInstr) GetConstMap() *VmInstr {
	if instr == nil {
		return nil
	}

	if VmCallback_Cmp(instr.fn, VmBasic_InitMap) { //const(!)
		return instr
	}
	acc := instr.isFnAccess()
	if acc != nil {
		return acc.GetConstMap()
	}

	return nil
}

func (instr *VmInstr) GetConstArrayPrm(i int) *VmInstr {
	if instr == nil {
		return nil
	}

	instr = instr.GetConstArray()
	if instr != nil && i < len(instr.prms) {
		return instr.prms[i].value.GetConst()
	}
	return nil
}
func (instr *VmInstr) GetConstMapPrm(i int) (*VmInstr, *VmInstr) {
	if instr == nil {
		return nil, nil
	}

	instr = instr.GetConstMap()
	if instr != nil && i < len(instr.prms) {
		return instr.prms[i].key, instr.prms[i].value.GetConst()
	}
	return nil, nil
}

func (instr *VmInstr) NumPrms() int {
	return len(instr.prms)
}

func (instr *VmInstr) AddPrm_key(key *VmInstr, add *VmInstr) int {
	if key != nil {
		key.parent = instr
	}
	if add != nil {
		add.parent = instr
	}

	var t VmInstrPrm
	t.key = key
	t.value = add

	instr.prms = append(instr.prms, t)

	return instr.NumPrms() - 1
}

func (instr *VmInstr) AddPrm_instr(add *VmInstr) int {
	return instr.AddPrm_key(nil, add)
}

func (instr *VmInstr) Exe(st *VmST) SAValue {

	var ret SAValue

	for instr != nil {
		ret = instr.fn(instr, st)
		instr = instr.next
	}

	return ret
}

func (instr *VmInstr) ExePrmKey(st *VmST, prm_i int) SAValue {
	return instr.prms[prm_i].key.Exe(st)
}

func (instr *VmInstr) ExePrm(st *VmST, prm_i int) SAValue {
	return instr.prms[prm_i].value.Exe(st)
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
	instr.temp = instr.accessAttr.result
	return instr.temp
}

func VmBasic_InitArray(instr *VmInstr, st *VmST) SAValue {
	str := "["
	for i := range instr.prms {
		prm := instr.ExePrm(st, i)
		str += prm.StringWithQuotes()
		str += ","
	}
	str, _ = strings.CutSuffix(str, ",")
	str += "]"

	instr.temp.SetBlob([]byte(str))
	return instr.temp
}

func VmBasic_InitMap(instr *VmInstr, st *VmST) SAValue {
	str := "{"
	for i := range instr.prms {

		k := instr.ExePrmKey(st, i)
		v := instr.ExePrm(st, i)

		key := k.StringWithQuotes()
		value := v.StringWithQuotes()

		str += fmt.Sprintf("%s: %s,", key, value)
	}
	str, _ = strings.CutSuffix(str, ",")
	str += "}"

	instr.temp.SetBlob([]byte(str))
	return instr.temp
}

func VmApi_Get(instr *VmInstr, st *VmST) SAValue {
	item := instr.ExePrm(st, 0)
	key := instr.ExePrm(st, 1) //index or key

	//array
	{
		arr := item.GetArrayItem(int(key.Number()))
		if arr != nil {
			instr.temp = *arr
			return instr.temp
		}
	}

	//map
	{
		mp := item.GetMapKey(key.String())
		if mp != nil {
			instr.temp = *mp
			return instr.temp
		}
	}

	instr.pos_attr.SetErrorExe("source is not Array [] or Map {} or map key isn't found")
	instr.temp = InitSAValue()
	return instr.temp
}

func VmApi_Array(instr *VmInstr, st *VmST) SAValue {

	a := instr.ExePrm(st, 0)
	b := instr.ExePrm(st, 1)

	//result [a, b] or [a] or [b], where a,b can be number, string, array, map

	str := "["

	aStr := a.String()
	bStr := b.String()

	if aStr != "" {
		str += aStr //a
	}

	if aStr != "" && bStr != "" {
		str += "," //comma
	}

	if bStr != "" {
		str += bStr //b
	}

	str += "]"

	instr.temp.SetBlob([]byte(str))
	return instr.temp
}

func VmApi_Compress(instr *VmInstr, st *VmST) SAValue {
	src := instr.ExePrm(st, 0)

	if src.IsBlob() {
		js := src.String()
		js = strings.ReplaceAll(js, " ", "")
		js = strings.ReplaceAll(js, "\t", "")

		instr.temp.SetBlob([]byte(js))
	} else {
		instr.temp = src //same
	}
	return instr.temp
}
