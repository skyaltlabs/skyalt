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

func VmOp_And(instr *VmInstr, rec *Rec, st *VmST) *Rec {
	is := instr.ExePrm(rec, st, 0).Is()
	if is {
		is = instr.ExePrm(rec, st, 1).Is()
	}
	instr.temp.value.SetBool(is)
	return instr.temp
}

func VmOp_Or(instr *VmInstr, rec *Rec, st *VmST) *Rec {
	is := instr.ExePrm(rec, st, 0).Is()
	if !is {
		is = instr.ExePrm(rec, st, 1).Is()
	}
	instr.temp.value.SetBool(is)
	return instr.temp
}

func VmOp_CmpEq(instr *VmInstr, rec *Rec, st *VmST) *Rec {

	right := instr.ExePrm(rec, st, 1)
	left := instr.ExePrm(rec, st, 0)

	instr.temp.value.SetBool(left.Cmp(right) == 0)
	return instr.temp
}

func VmOp_CmpNeq(instr *VmInstr, rec *Rec, st *VmST) *Rec {

	right := instr.ExePrm(rec, st, 1)
	left := instr.ExePrm(rec, st, 0)

	instr.temp.value.SetBool(left.Cmp(right) != 0)
	return instr.temp
}

func VmOp_CmpL(instr *VmInstr, rec *Rec, st *VmST) *Rec {

	right := instr.ExePrm(rec, st, 1)
	left := instr.ExePrm(rec, st, 0)

	instr.temp.value.SetBool(left.Cmp(right) < 0)
	return instr.temp
}
func VmOp_CmpH(instr *VmInstr, rec *Rec, st *VmST) *Rec {

	right := instr.ExePrm(rec, st, 1)
	left := instr.ExePrm(rec, st, 0)

	instr.temp.value.SetBool(left.Cmp(right) > 0)
	return instr.temp
}

func VmOp_CmpEqL(instr *VmInstr, rec *Rec, st *VmST) *Rec {

	right := instr.ExePrm(rec, st, 1)
	left := instr.ExePrm(rec, st, 0)

	instr.temp.value.SetBool(left.Cmp(right) <= 0)
	return instr.temp
}
func VmOp_CmpEqH(instr *VmInstr, rec *Rec, st *VmST) *Rec {

	right := instr.ExePrm(rec, st, 1)
	left := instr.ExePrm(rec, st, 0)

	instr.temp.value.SetBool(left.Cmp(right) >= 0)
	return instr.temp
}

func VmOp_Mul(instr *VmInstr, rec *Rec, st *VmST) *Rec {

	right := instr.ExePrm(rec, st, 1)
	left := instr.ExePrm(rec, st, 0)

	instr.temp.value.SetNumber(left.value.Number() * right.value.Number())
	return instr.temp
}

func VmOp_Div(instr *VmInstr, rec *Rec, st *VmST) *Rec {

	right := instr.ExePrm(rec, st, 1)
	left := instr.ExePrm(rec, st, 0)

	rV := right.value.Number()
	if rV != 0 {
		instr.temp.value.SetNumber(left.value.Number() / rV)
	} else {
		instr.temp.value.SetNumber(0)
		fmt.Println("Division by zero") //err ...
	}

	return instr.temp
}

func VmOp_Mod(instr *VmInstr, rec *Rec, st *VmST) *Rec {

	right := instr.ExePrm(rec, st, 1)
	left := instr.ExePrm(rec, st, 0)

	rV := int(right.value.Number())
	if rV != 0 {
		instr.temp.value.SetInt(int(left.value.Number()) % rV)
	} else {
		instr.temp.value.SetNumber(0)
		fmt.Println("Modulo by zero") //err ...
	}

	return instr.temp
}

func VmOp_AddNumbers(instr *VmInstr, rec *Rec, st *VmST) *Rec {

	right := instr.ExePrm(rec, st, 1)
	left := instr.ExePrm(rec, st, 0)

	instr.temp.value.SetNumber(left.value.Number() + right.value.Number())
	return instr.temp
}

func VmOp_Sub(instr *VmInstr, rec *Rec, st *VmST) *Rec {

	right := instr.ExePrm(rec, st, 1)
	left := instr.ExePrm(rec, st, 0)

	instr.temp.value.SetNumber(left.value.Number() - right.value.Number())
	return instr.temp
}

func VmOp_AddTexts(instr *VmInstr, rec *Rec, st *VmST) *Rec {

	right := instr.ExePrm(rec, st, 1)
	left := instr.ExePrm(rec, st, 0)

	instr.temp.value.SetString(left.value.String() + right.value.String())
	return instr.temp
}

type VmOp struct {
	prior       int
	leftToRight bool
	name        string
	fn          VmInstr_callbackExecute
}

type VmOps struct {
	ops []VmOp
}

func (ops *VmOps) _Add(item VmOp) {
	ops.ops = append(ops.ops, item)
}

func NewVmOps() *VmOps {
	var ops VmOps

	//must be ordered by len(VmOp.name)
	ops._Add(VmOp{50, true, "&&", VmOp_And})
	ops._Add(VmOp{50, true, "||", VmOp_Or})

	ops._Add(VmOp{40, true, "==", VmOp_CmpEq})
	ops._Add(VmOp{40, true, "!=", VmOp_CmpNeq})
	ops._Add(VmOp{40, true, "<=", VmOp_CmpEqL})
	ops._Add(VmOp{40, true, ">=", VmOp_CmpEqH})

	ops._Add(VmOp{40, true, "<", VmOp_CmpL})
	ops._Add(VmOp{40, true, ">", VmOp_CmpH})

	ops._Add(VmOp{20, true, "+", VmOp_AddNumbers})
	ops._Add(VmOp{25, true, "&", VmOp_AddTexts})
	ops._Add(VmOp{20, true, "-", VmOp_Sub})

	ops._Add(VmOp{10, true, "*", VmOp_Mul})
	ops._Add(VmOp{10, true, "/", VmOp_Div})
	ops._Add(VmOp{10, true, "%", VmOp_Mod})

	return &ops
}

func VmOps_HasPrefix(str string, prefix string) bool {

	nA := len(str)
	nB := len(prefix)

	if nA < nB {
		return false
	}

	return strings.EqualFold(str[:nB], prefix) //can insensitive
}

func (ops *VmOps) SearchForOpStart(line string) *VmOp {

	for _, op := range ops.ops {

		if VmOps_HasPrefix(line, op.name) {
			return &op
		}
	}
	return nil
}

func (ops *VmOps) SearchForOpFull(line string) *VmOp {

	for _, op := range ops.ops {
		if line == op.name {
			return &op
		}
	}
	return nil
}
