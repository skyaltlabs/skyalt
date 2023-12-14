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
)

func VmColor_text() OsCd    { return OsCd{50, 170, 160, 255} }
func VmColor_api() OsCd     { return OsCd{150, 80, 200, 255} }
func VmColor_apiDraw() OsCd { return OsCd{100, 40, 30, 255} }

type VmLine struct {
	widget *SAWidget
	line   string

	lexer *VmLexer

	ops   *VmOps
	apis  *VmApis
	prior int

	depends []*SAWidgetAttr

	errs []string
}

func InitVmLine(ln string, ops *VmOps, apis *VmApis, prior int, widget *SAWidget) (*VmLine, error) {
	var line VmLine
	line.widget = widget
	line.line = ln
	line.ops = ops
	line.apis = apis
	line.prior = prior

	var err error
	line.lexer, err = ParseLine(line.line, ops)
	if err != nil {
		return &line, err
	}

	return &line, nil
}

func (line *VmLine) addSyntax_text(lexer *VmLexer, cd OsCd) {
	//...
}

func (line *VmLine) addSyntax_back(lexer *VmLexer, cd OsCd) {
	//...
}

func (line *VmLine) addSyntax_label(lexer *VmLexer, cd OsCd, label string) {
	//...
}

func (line *VmLine) addError(lexer *VmLexer, err string) bool {
	line.errs = append(line.errs, err)
	return false
}

func (line *VmLine) findOp(lexer *VmLexer) (*VmOp, int) {

	var ret_op *VmOp
	var ret_i = -1

	for i, it := range lexer.subs {
		if it.tp == VmLexerOp {
			op := it.FindOp(line.line, line.ops)
			if op != nil {
				if ret_op == nil || (len(op.name) >= len(ret_op.name) && OsTrnBool(op.leftToRight, (op.prior >= ret_op.prior), (op.prior > ret_op.prior))) {
					ret_op = op
					ret_i = i
				}
			}
		}
	}

	return ret_op, ret_i
}

func (line *VmLine) setParams(lexer *VmLexer, instr *VmInstr) int {

	if len(lexer.subs) == 0 {
		return 0
	}

	prm_i := 0
	var param *VmLexer
	for {
		param = lexer.ExtractParam(prm_i)
		if param == nil {
			break
		}

		if len(param.subs) == 0 {
			line.addError(param, "Empty parameter")
			return -1
		}

		instr.AddPropInstr(line.getExp(param))

		prm_i++
	}

	return prm_i
}

func (line *VmLine) getConstant(lexer *VmLexer) (bool, *VmInstr) {
	if len(lexer.subs) != 1 {
		return false, nil
	}

	lexer = lexer.subs[0]

	if lexer.tp == VmLexerNumber {
		instr := NewVmInstr(VmBasic_Constant)
		value, err := strconv.ParseFloat(lexer.GetString(line.line), 64)
		if err != nil {
			line.addError(lexer, "Converting string . number failed")
			return true, nil
		}
		instr.temp.SetNumber(value)
		return true, instr

	} else if lexer.IsQuote('"', line.line) {
		line.addSyntax_text(lexer, VmColor_text())
		ch := line.line[lexer.start]
		if ch == '"' {
			instr := NewVmInstr(VmBasic_Constant)
			instr.temp.SetString(lexer.GetStringReplaceDivs(line.line))
			return true, instr
		}

	}

	return false, nil
}

func (line *VmLine) getExp(lexer *VmLexer) *VmInstr {

	if len(lexer.subs) == 0 {
		line.addError(lexer, "Empty expression")
		return nil
	}

	// operator
	opp, op_i := line.findOp(lexer)

	if opp != nil && opp.fn != nil {
		if op_i == 0 {
			line.addError(lexer, "Missing left side")
			return nil
		}

		op := NewVmInstr(opp.fn)

		// right
		rightLex := lexer.Extract(op_i+1, -1)
		opRight := line.getExp(rightLex)

		// left
		leftLex := lexer.Extract(0, op_i)
		opLeft := line.getExp(leftLex)

		if opLeft != nil && opRight != nil {
			op.AddPropInstr(opLeft)
			op.AddPropInstr(opRight)
		} else {
			if opLeft == nil {
				line.addError(leftLex, "Left side is Missing")
			}
			if opRight == nil {
				line.addError(rightLex, "Right side is Missing")
			}
		}
		return op
	}

	// brackets
	if len(lexer.subs) == 1 && lexer.subs[0].tp == VmLexerBracket {
		instr := NewVmInstr(VmBasic_Bracket)
		instr.AddPropInstr(line.getExp(lexer.subs[0]))
		return instr
	}

	// api()
	if len(lexer.subs) >= 2 &&
		lexer.subs[0].tp == VmLexerWord &&
		lexer.subs[1].tp == VmLexerBracket {

		firstLex := lexer.subs[0]

		var instr *VmInstr
		api := line.apis.FindName(firstLex.GetString(line.line))
		if api != nil {

			if api.prior > line.prior {
				line.addError(firstLex, "Asset can't use this high prior API")
			}

			if api.prms >= 0 {
				line.addSyntax_text(firstLex, VmColor_api())
			} else if api.prms < 0 {
				line.addSyntax_text(firstLex, VmColor_apiDraw())
			}

			instr = NewVmInstr(api.fn)
			prmsLex := lexer.subs[1]
			line.setParams(prmsLex, instr)

			if api.prms != instr.NumPrms() {
				line.addError(lexer, "Need "+strconv.Itoa(api.prms)+" parameters")
			}

			return instr
		}

		return instr
	}

	// Constant
	{
		ok, instr := line.getConstant(lexer)
		if ok {
			return instr
		}
	}

	// Access
	if lexer.subs[0].tp == VmLexerWord {

		instr := NewVmInstr(VmBasic_Access)

		if len(lexer.subs) >= 2 && lexer.subs[1].tp == VmLexerDot {
			if len(lexer.subs) >= 3 && lexer.subs[2].tp == VmLexerWord {

				if len(lexer.subs) == 3 {

					widgetName := lexer.subs[0].GetString(line.line)
					attrName := lexer.subs[2].GetString(line.line)

					ww := line.widget.parent.FindNode(widgetName)
					if ww != nil {
						vv := ww.findAttr(attrName)
						if vv != nil {
							instr.attr = vv
							line.depends = append(line.depends, vv) //add
						} else {
							line.addError(lexer, fmt.Sprintf("Attribute(%s) not found", attrName))
						}
					} else {
						line.addError(lexer, fmt.Sprintf("Widget(%s) not found", widgetName))
					}
				} else {
					line.addError(lexer, "Access must be in form of <widget>.<attribute>")
				}
			} else {
				line.addError(lexer, "Missing attribute")
			}
		} else {
			line.addError(lexer, "Missing '.'")
		}

		return instr
	}

	line.addError(lexer, "Unrecognize syntax")
	return nil
}

func (line *VmLine) Parse() *VmInstr {
	return line.getExp(line.lexer)
}