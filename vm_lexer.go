package main

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	VmLexerBracketRound  = 0
	VmLexerBracketSquare = 1
	VmLexerBracketCurly  = 2
	VmLexerWord          = 3
	VmLexerOp            = 4
	VmLexerNumber        = 5
	VmLexerQuote         = 6
	VmLexerDot           = 7
	VmLexerComma         = 8
	VmLexerDiv           = 9
)

// use for root and brackets
type VmLexer struct {
	tp         int
	start, end int
	subs       []*VmLexer
	parent     *VmLexer
}

func NewVmLexer(tp, start, end int) *VmLexer {
	var lexer VmLexer
	lexer.tp = tp
	lexer.start = start
	lexer.end = end
	return &lexer
}

func (lexer *VmLexer) GetStart(line string) byte {
	return line[lexer.start]
}
func (lexer *VmLexer) GetString(line string) string {

	st := lexer.start
	en := lexer.end
	if lexer.tp == VmLexerQuote {
		st++
		en--
		if en < st {
			en = st
		}
	}

	return line[st:en]
}

func (lexer *VmLexer) GetRange(line string) OsV2 {

	start := 0
	end := 0

	//convert byte_pos -> rune_pos
	pos := 0
	for i := 0; i < len(line); i++ {
		if i == lexer.start {
			start = pos
		}
		if i < lexer.start {
			start = pos + 1
		}

		if i == lexer.end {
			end = pos
			break
		}
		if i < lexer.end {
			end = pos + 1
		}
		pos++
	}

	return OsV2{start, end}
}

func (lexer *VmLexer) FindOp(line string, ops *VmOps) *VmOp {
	return ops.SearchForOpFull(lexer.GetString(line))
}

func (lexer *VmLexer) Cmp(line string, find string) bool {
	return strings.EqualFold(lexer.GetString(line), find) //can insensitive
}

func (lexer *VmLexer) Find(tp int) int {
	for i, it := range lexer.subs {
		if it.tp == tp {
			return i
		}
	}
	return -1
}

func (lexer *VmLexer) Extract(st, en int) *VmLexer {
	if st == en {
		return nil
	}

	if en < 0 {
		en = len(lexer.subs)
	}

	var ret VmLexer
	ret.tp = VmLexerBracketRound
	ret.parent = lexer

	if st >= len(lexer.subs) {
		return &ret
	}

	ret.start = lexer.subs[st].start
	ret.end = lexer.subs[en-1].end
	ret.subs = lexer.subs[st:en]

	return &ret
}

func (lexer *VmLexer) ExtractParam(prm_i int) *VmLexer {

	comma_st := 0
	comma_en := len(lexer.subs)

	for i, it := range lexer.subs {
		if it.tp == VmLexerComma {
			if prm_i == 1 {
				comma_st = i + 1
			}
			if prm_i == 0 {
				comma_en = i
				break
			}
			prm_i-- //decrement!
		}
	}

	if prm_i > 0 {
		return nil
	}

	return lexer.Extract(comma_st, comma_en)
}

func (lexer *VmLexer) IsQuote(chStart byte, line string) bool {
	return lexer.tp == VmLexerQuote && lexer.GetStart(line) == chStart
}

func IsWordLetter(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

func (lexer *VmLexer) IsSyntaxKeyword(line string) bool {

	if lexer.tp != VmLexerWord {
		return false
	}
	str := lexer.GetString(line)
	return strings.EqualFold(str, "return")
}

func ParseLine(line string, skipN int, ops *VmOps) (*VmLexer, error) {

	lexer := NewVmLexer(VmLexerBracketRound, 0, len(line))
	if lexer.end == 0 {
		return lexer, nil
	}

	//skipN := 0
	for i, ch := range line {

		if skipN > 0 {
			skipN--
			continue
		}

		if ch == '"' {
			//quote
			ok := false
			lastCh2 := rune(0)
			for ii, ch2 := range line[i+1:] {
				if lastCh2 != '\\' { //if previous was '\' than ignore
					if ch == '"' && ch2 == '"' {
						//done
						lexer.subs = append(lexer.subs, NewVmLexer(VmLexerQuote, i, i+1+ii+1))
						skipN = utf8.RuneCountInString(line[i+1 : i+1+ii+1]) //ii + 1
						ok = true
						break
					}
				}
				if lastCh2 == '\\' && ch2 == '\\' { //for case where there are two \\
					ch2 = 0
				}
				lastCh2 = ch2
			}
			if !ok {
				return nil, fmt.Errorf("quote %c not closed", line[i])
			}
		} else if ch == '(' {
			//bracket(open)
			lex := NewVmLexer(VmLexerBracketRound, i, -1)
			lex.parent = lexer
			lexer.subs = append(lexer.subs, lex)
			lexer = lex //go deeper
		} else if ch == '[' {
			//bracket(open)
			lex := NewVmLexer(VmLexerBracketSquare, i, -1)
			lex.parent = lexer
			lexer.subs = append(lexer.subs, lex)
			lexer = lex //go deeper
		} else if ch == '{' {
			//bracket(open)
			lex := NewVmLexer(VmLexerBracketCurly, i, -1)
			lex.parent = lexer
			lexer.subs = append(lexer.subs, lex)
			lexer = lex //go deeper

		} else if ch == ')' || ch == ']' || ch == '}' {
			//bracket(close)
			lexer.end = i + 1
			lexer = lexer.parent //go back
			if lexer == nil {
				return nil, fmt.Errorf("lexer.parent == nil")
			}
		} else if ch == '.' {
			//dot
			lexer.subs = append(lexer.subs, NewVmLexer(VmLexerDot, i, i+1))
		} else if ch == ',' {
			//comma
			lexer.subs = append(lexer.subs, NewVmLexer(VmLexerComma, i, i+1))
		} else if ch == ':' {
			//div
			lexer.subs = append(lexer.subs, NewVmLexer(VmLexerDiv, i, i+1))
		} else if IsWordLetter(ch) || ch == '$' || (ch == '#' && (i+1 >= len(line) || line[i+1] != '=')) {
			//word
			var lex *VmLexer
			for ii, ch2 := range line[i+1:] {
				if !(IsWordLetter(ch2) || (ch2 >= '0' && ch2 <= '9')) {
					//done
					lex = NewVmLexer(VmLexerWord, i, i+ii+1)
					skipN = ii
					break
				}
			}
			if lex == nil {
				//done
				lex = NewVmLexer(VmLexerWord, i, len(line))
				skipN = len(line) - i
			}
			lexer.subs = append(lexer.subs, lex)

		} else if ch >= '0' && ch <= '9' {
			//number
			var lex *VmLexer
			dotUse := false
			for ii, ch2 := range line[i+1:] {
				dot := (ch2 == '.')
				digits := (ch2 >= '0' && ch2 <= '9')

				if (dotUse || !dot) && !digits {
					//done
					lex = NewVmLexer(VmLexerNumber, i, i+ii+1)
					skipN = ii
					break
				}
				if dot {
					dotUse = true
				}
			}

			if lex == nil {
				//done
				lex = NewVmLexer(VmLexerNumber, i, len(line))
				skipN = len(line) - i
			}

			//cover cases as: a = -1; a = a + -2; etc.
			{
				n := len(lexer.subs)
				pre_op_plus_minus := false
				pre_pre_op := false

				if n > 0 {
					//pre op
					if lexer.subs[n-1].tp == VmLexerOp {
						str := lexer.subs[n-1].GetString(line)
						if str == "+" || str == "-" {
							if lexer.subs[n-1].start+1 == i { //no space between +/- and digit
								pre_op_plus_minus = true
							}
						}
					}
				}
				if n > 1 {
					//pre pre op
					if lexer.subs[n-2].tp == VmLexerOp {
						pre_pre_op = true
					}

					if lexer.subs[n-2].IsSyntaxKeyword(line) {
						pre_pre_op = true
					}

				}
				if pre_op_plus_minus {

					last_n := 0 //number of subs between comma(or start) and end
					for j := len(lexer.subs) - 1; j >= 0 && (lexer.subs[j].tp != VmLexerComma); j-- {
						last_n++
					}

					if pre_pre_op || last_n < 2 {

						lexer.subs = lexer.subs[:n-1] //removes last
						lex.start--                   //number start one letter back
					}
				}
			}

			lexer.subs = append(lexer.subs, lex)
		} else if ch == ' ' || ch == '\t' {
			//nothing, just empty space
		} else {
			//operator
			op := ops.SearchForOpStart(line[i:])
			if op != nil {
				lexer.subs = append(lexer.subs, NewVmLexer(VmLexerOp, i, i+len(op.name)))
				skipN = len(op.name) - 1
			} else {
				return nil, fmt.Errorf("unknown syntax")
			}
		}
	}

	if lexer.parent != nil || lexer.end == 0 {
		return nil, fmt.Errorf("bracket %c not closed", line[lexer.start])
	}

	return lexer, nil
}
