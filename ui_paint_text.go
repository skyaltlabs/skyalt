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
	"unicode/utf8"
)

func (ui *Ui) Paint_textWidth(style *UiComp, value string, cursorPos int64, ratioH float64, fontPath string, enableFormating bool) float64 {

	//sdiv := style.GetDiv(true, app)

	if ratioH <= 0 {
		ratioH = SKYALT_FONT_HEIGHT
	}
	if len(fontPath) == 0 {
		fontPath = SKYALT_FONT_PATH
	}

	font := ui.win.fonts.Get(SKYALT_FONT_PATH)
	textH := ui.CellWidth(ratioH)
	cell := float64(ui.win.Cell())
	if cursorPos < 0 {
		size, err := font.GetTextSize(value, g_WinFont_DEFAULT_Weight, textH, 0, enableFormating)
		if err == nil {
			return float64(size.X) / cell // pixels for the whole string
		}
	} else {
		px, err := font.GetPxPos(value, g_WinFont_DEFAULT_Weight, textH, int(cursorPos), enableFormating)
		if err == nil {
			return float64(px) / cell // pixels to cursor
		}
	}
	return -1
}

func (ui *Ui) Paint_textGrid(grid OsV4, cd OsCd, style *UiComp, value string, valueOrigEdit string, icon string, selection bool, editable bool) {

	lv := ui.GetCall()
	if lv.call == nil /*|| lv.call.crop.IsZero()*/ {
		return
	}

	if !style.enable {
		lv.call.data.touch_enabled = false
	}
	lv.call.data.scrollH.narrow = true
	lv.call.data.scrollV.show = false

	ui.Div_col(grid.Start.X, OsMaxFloat(ui.DivInfo_get(SA_DIV_GET_layoutWidth, 0), ui.Paint_textWidth(style, value, -1, 0, "", style.label_formating))) //+marginX*4+margin*2
	ui.Div_row(grid.Start.Y, ui.DivInfo_get(SA_DIV_GET_layoutHeight, 0))
	ui.Div_start(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y)
	//style.Paint(st.stack.canvas, value, valueOrigEdit, selection, edit, icon, icon_margin, enable, app)

	coordImage, coordText := ui._compGetTextImageCoord(lv.call.canvas, 1, style.image_alignH, len(icon) > 0, len(value) > 0)
	if len(icon) > 0 {
		ui._compDrawImage(coordImage, icon, cd, style)
	}
	if editable || len(value) > 0 {
		ui._compDrawText(coordText, value, valueOrigEdit, cd, SKYALT_FONT_HEIGHT, selection, editable, int(style.label_alignH), int(style.label_alignV), style.label_formating)
	}

	ui.Div_end()
}

func _UiPaint_WordPos(str string, mid int) (int, int) {
	start := 0
	end := 0

	str = strings.ToLower(str)

	p := 0
	for _, ch := range str {

		if OsIsTextWord(ch) {
			end = p + 1
		} else {
			if p < mid {
				start = p + 1
			} else {
				break
			}
		}
		p++
	}
	if end < start {
		end = start
	}

	return start, end
}

func _UiPaint_RemoveFormating(str string) string {

	str = strings.ReplaceAll(str, "***", "")
	str = strings.ReplaceAll(str, "___", "")
	str = strings.ReplaceAll(str, "###", "")

	str = strings.ReplaceAll(str, "**", "")
	str = strings.ReplaceAll(str, "__", "")
	str = strings.ReplaceAll(str, "##", "")

	return str
}

func _UiPaint_HashFormatingPreSuf_fix(str string, fn func(a, b string) bool) int {

	if fn(str, "***") || fn(str, "___") || fn(str, "###") {
		return 3
	}
	if fn(str, "**") || fn(str, "__") || fn(str, "##") {
		return 2
	}
	return 0
}

func _UiPaint_CheckSelectionExplode(str string, start *int, end *int) {
	if *start < *end {
		*start -= _UiPaint_HashFormatingPreSuf_fix(str[:*start], strings.HasSuffix)
		*end += _UiPaint_HashFormatingPreSuf_fix(str[*end:], strings.HasPrefix)
	}
	if *end < *start {
		*end -= _UiPaint_HashFormatingPreSuf_fix(str[:*end], strings.HasSuffix)
		*start += _UiPaint_HashFormatingPreSuf_fix(str[*start:], strings.HasPrefix)
	}
}

/*func _UiPaint_CheckSelectionImplode(str string, start *int, end *int) {
	if *start <= *end {
		*start += _UiPaint_HashFormatingPreSuf_fix(str[*start:], strings.HasPrefix)
		*end -= _UiPaint_HashFormatingPreSuf_fix(str[:*end], strings.HasSuffix)

	}
	if *end < *start {
		*end += _UiPaint_HashFormatingPreSuf_fix(str[*end:], strings.HasPrefix)
		*start -= _UiPaint_HashFormatingPreSuf_fix(str[:*start], strings.HasSuffix)
	}
}*/

func _UiPaint_CursorPos(str string, curr int, move int, enableFormating bool) int {

	n := utf8.RuneCountInString(str)

	//raw string
	if enableFormating {
		if move < 0 {
			curr -= _UiPaint_HashFormatingPreSuf_fix(str[:curr], strings.HasSuffix)
		}

		if move > 0 {
			curr += _UiPaint_HashFormatingPreSuf_fix(str[curr:], strings.HasPrefix)
		}
	}
	curr += move

	//check
	curr = OsMax(curr, 0) //min
	curr = OsMin(curr, n) //max

	return curr
}

func (ui *Ui) _UiPaint_resetKeys(editable bool) {

	keys := &ui.win.io.keys

	//copy/cut/paste
	keys.copy = false
	keys.cut = false
	keys.paste = false

	//arrows
	keys.arrowL = false
	keys.arrowR = false
	keys.home = false
	keys.end = false

	if editable {
		keys.text = ""
		keys.delete = false
		keys.backspace = false

		keys.esc = false
	}
}

func (ui *Ui) _UiPaint_Text_VScrollInto(cursor OsV2, lineH int) {

	lv := ui.GetCall()
	if lv.call.parent == nil {
		return
	}

	v_pos := cursor.Y * lineH

	v_st := lv.call.parent.data.scrollV.GetWheel()
	v_sz := lv.call.crop.Size.Y - lineH
	v_en := v_st + v_sz

	if v_pos <= v_st {
		lv.call.parent.data.scrollV.SetWheel(OsMax(0, v_pos))
	} else if v_pos >= v_en {
		lv.call.parent.data.scrollV.wheel = OsMax(0, v_pos-v_sz) //SetWheel() has boundary check, which is not good here
	}
}
func (ui *Ui) _UiPaint_Text_HScrollInto(str string, cursor OsV2, font *WinFont, textH int, margin float64, marginX float64, enableFormating bool) error {

	lv := ui.GetCall()
	if lv.call.parent == nil {
		return nil
	}

	h_pos, err := font.GetPxPos(str, g_WinFont_DEFAULT_Weight, textH, cursor.X, enableFormating)
	if err != nil {
		return err
	}

	h_align := ui.CellWidth(margin + marginX) //margin + marginX

	h_st := lv.call.parent.data.scrollH.GetWheel()
	h_sz := lv.call.crop.Size.X - 3*h_align
	h_en := h_st + h_sz

	if h_pos <= h_st {
		lv.call.parent.data.scrollH.SetWheel(OsMax(0, h_pos))
	} else if h_pos >= h_en {
		lv.call.parent.data.scrollH.wheel = OsMax(0, h_pos-h_sz) //SetWheel() has boundary check, which is not good here
	}
	return nil
}

func (ui *Ui) _UiPaint_TextSelectTouch(str string, strEditOrig string, touchPos OsV2, lineEnd OsV2, editable bool, font *WinFont, textH int, lineH int, margin float64, marginX float64, enableFormating bool) {

	lv := ui.GetCall()

	//dict := stt.dict
	edit := &ui.edit
	keys := &ui.win.io.keys
	touch := &ui.win.io.touch

	this_uid := lv.call //.Hash()
	edit_uid := edit.uid
	next_uid := edit.next_uid

	active := (edit_uid != nil && edit_uid == this_uid)
	activate_next_uid := (this_uid == next_uid)

	if lv.call.enableInput && ((editable && edit.setFirstEditbox) || (editable && edit.tab) || activate_next_uid) {
		//setFirstEditbox or Tab
		edit.uid = this_uid

		if !active {
			edit.temp = strEditOrig
			edit.orig = strEditOrig
		}

		if !activate_next_uid {
			//select all
			edit.start = OsV2{}
			edit.end = lineEnd
		}

		edit.setFirstEditbox = false
		edit.next_uid = nil
		edit.tab = false

		ui.win.SetTextCursorMove()
	} else if lv.call.data.touch_inside && touch.start {
		//click inside
		if !active {
			edit.next_uid = this_uid //set next_uid
		}

		//set end
		edit.end = touchPos

		if !active || !keys.shift {
			//set start
			edit.start = touchPos
		}

		switch touch.numClicks {
		case 2:
			first, last := _UiPaint_WordPos(str, touchPos.X)
			edit.start = OsV2{first, touchPos.Y} //set start
			edit.end = OsV2{last, touchPos.Y}    //set end
		case 3:
			edit.start = OsV2{0, touchPos.Y}                         //set start
			edit.end = OsV2{utf8.RuneCountInString(str), touchPos.Y} //set end
		}
	}

	//keep selecting
	if active && lv.call.data.touch_active && (touch.numClicks != 2 && touch.numClicks != 3) {
		edit.end = touchPos //set end

		//scroll
		ui._UiPaint_Text_VScrollInto(touchPos, lineH)
		ui._UiPaint_Text_HScrollInto(str, touchPos, font, textH, margin, marginX, enableFormating)

		//root.buff.ResetHost() //SetNoSleep()
	}
}

func subString(s string, rune_start int, rune_end int) (int, int) {

	st := len(s)
	en := len(s)

	p := 0
	//convert rune_pos -> byte_pos
	for i := range s {
		if p == rune_start {
			st = i
		}
		if p == rune_end {
			en = i
			break
		}
		p++
	}
	return st, en
}

func _UiPaint_getStringSubBytePosEx(str string, sx int, ex int) (int, int) {
	//swap
	if sx > ex {
		t := sx
		sx = ex
		ex = t
	}
	return subString(str, int(sx), int(ex))
}
func (ui *Ui) _UiPaint_getStringSubBytePos(str string) (int, int, int, int) {
	edit := &ui.edit

	sx := edit.start.X
	ex := edit.end.X

	selFirst := sx
	selLast := ex
	if ex < sx {
		selFirst = ex
		selLast = sx
	}

	x, y := _UiPaint_getStringSubBytePosEx(str, int(sx), int(ex))
	return x, y, selFirst, selLast
}

func (ui *Ui) _UiPaint_TextSelectKeys(str string, lineY int, lineEnd OsV2, editable bool, font *WinFont, textH int, lineH int, margin float64, marginX float64, enableFormating bool) {

	keys := &ui.win.io.keys
	edit := &ui.edit

	s := &edit.start
	e := &edit.end

	old := *e

	if editable {
		str = edit.temp
	}
	st, en, _, _ := ui._UiPaint_getStringSubBytePos(str)

	//select all
	if keys.selectAll {
		*s = OsV2{}
		*e = lineEnd
	}

	//copy, cut
	if keys.copy || keys.cut {
		if keys.shift {
			keys.clipboard = _UiPaint_RemoveFormating(str)
		} else {
			keys.clipboard = str[st:en]
		}
	}

	//shift
	if keys.shift {

		//ctrl
		ex := e.X
		if keys.ctrl {
			if keys.arrowL {
				p := _UiPaint_CursorPos(str, ex, -1, enableFormating)
				first, _ := _UiPaint_WordPos(str, p)
				if first == p && p > 0 {
					first, _ = _UiPaint_WordPos(str, p-1)
				}
				e.X = first
			}
			if keys.arrowR {
				p := _UiPaint_CursorPos(str, ex, +1, enableFormating)
				_, last := _UiPaint_WordPos(str, p)
				if last == p && p+1 < utf8.RuneCountInString(str) {
					_, last = _UiPaint_WordPos(str, p+1)
				}
				e.X = last
			}
		} else {
			if keys.arrowL {
				p := _UiPaint_CursorPos(str, ex, -1, enableFormating)
				e.X = p
			}
			if keys.arrowR {
				p := _UiPaint_CursorPos(str, ex, +1, enableFormating)
				e.X = p
			}
		}

		//home & end
		//scroll whole layout ............
		if keys.home {
			e.X = 0
		}
		if keys.end {
			e.X = utf8.RuneCountInString(str)
		}
	}

	//scroll
	newPos := *e
	if old.Y != newPos.Y {
		ui._UiPaint_Text_VScrollInto(newPos, lineH)
	}
	if old.X != newPos.X {
		ui._UiPaint_Text_HScrollInto(str, newPos, font, textH, margin, marginX, enableFormating)
	}
}

func (ui *Ui) _UiPaint_TextEditKeys(tabIsChar bool, font *WinFont, textH int, lineH int, margin float64, marginX float64, enableFormating bool) string {

	edit := &ui.edit
	keys := &ui.win.io.keys

	shiftKey := keys.shift

	uid := edit.uid

	s := &edit.start
	e := &edit.end

	old := *e

	//tempRec := &edit.temp
	str := edit.temp
	st, en, selFirst, selLast := ui._UiPaint_getStringSubBytePos(str)

	//cut/paste(copy() is in selectKeys)
	if keys.cut {
		//remove
		str = str[:st] + str[en:]
		edit.temp = str

		//select
		s.X = selFirst
		e.X = selFirst
	} else if keys.paste {
		//remove old selection
		if st != en {
			str = str[:st] + str[en:]
		}

		//insert
		cb := keys.clipboard
		str = str[:st] + cb + str[st:]
		edit.temp = str

		p := selFirst + utf8.RuneCountInString(cb)
		s.X = p
		e.X = p
	}

	//insert text
	txt := keys.text
	if tabIsChar && keys.tab {
		txt += "\t"
	}
	if len(txt) > 0 {
		//remove old selection
		if st != en {
			str = str[:st] + str[en:]
		}

		//insert
		str = str[:st] + txt + str[st:]
		edit.temp = str

		//cursor
		p := selFirst + utf8.RuneCountInString(txt)
		s.X = p
		e.X = p

		//reset
		keys.text = ""
	}

	//delete/backspace
	if st != en {
		if keys.delete || keys.backspace {
			//remove
			str = str[:st] + str[en:]
			edit.temp = str

			//cursor
			s.X = selFirst
			e.X = selFirst
		}
	} else {
		if keys.delete {
			//remove
			if st < len(str) {
				//removes one letter
				st2, en2 := _UiPaint_getStringSubBytePosEx(str, s.X, s.X+1)
				str = str[:st2] + str[en2:]
				edit.temp = str
			}
		} else if keys.backspace {
			//remove
			if st > 0 {
				//removes one letter
				st2, en2 := _UiPaint_getStringSubBytePosEx(str, s.X-1, s.X)
				str = str[:st2] + str[en2:]
				edit.temp = str

				//select
				p := OsMax(0, selFirst-1)
				s.X = p
				e.X = p
			}
		}
	}

	if !shiftKey {
		//arrows
		if st != en {
			if keys.arrowL {
				//from select -> single start
				s.X = selFirst
				e.X = selFirst
			} else if keys.arrowR {
				//from select -> single end
				s.X = selLast
				e.X = selLast
			}
		} else {
			if keys.ctrl {
				if keys.arrowL {
					p := OsMax(e.X-1, 0)
					first, _ := _UiPaint_WordPos(str, p)
					if first == p && p > 0 {
						first, _ = _UiPaint_WordPos(str, p-1)
					}
					s.X = first
					e.X = first
				}
				if keys.arrowR {
					p := OsMin(e.X+1, utf8.RuneCountInString(str))
					_, last := _UiPaint_WordPos(str, p)
					if last == p && p+1 < utf8.RuneCountInString(str) {
						_, last = _UiPaint_WordPos(str, p+1)
					}
					s.X = last
					e.X = last
				}
			} else {
				if keys.arrowL {
					p := OsMax(0, e.X-1)
					s.X = p
					e.X = p
				} else if keys.arrowR {
					p := OsMin(e.X+1, utf8.RuneCountInString(str))
					s.X = p
					e.X = p
				}
			}
		}

		//home/end
		if keys.home {
			s.X = 0
			e.X = 0
		} else if keys.end {
			p := utf8.RuneCountInString(str)
			s.X = p
			e.X = p
		}
	}

	//history
	{
		//app := stt.GetApp()
		his := UiPaintTextHistoryItem{str: str, cur: e.X}

		ui.edit_history.FindOrAdd(uid, his).AddWithTimeOut(his)

		if keys.backward {
			his = ui.edit_history.FindOrAdd(uid, his).Backward(his)
			edit.temp = his.str
			s.X = his.cur
			e.X = his.cur
		}
		if keys.forward {
			his = ui.edit_history.FindOrAdd(uid, his).Forward()
			edit.temp = his.str
			s.X = his.cur
			e.Y = his.cur
		}
	}

	//scroll
	newPos := *e
	if old.Y != newPos.Y {
		ui._UiPaint_Text_VScrollInto(newPos, lineH)
	}
	if old.X != newPos.X {
		ui._UiPaint_Text_HScrollInto(str, newPos, font, textH, margin, marginX, enableFormating)
	}

	return edit.temp
}

func (ui *Ui) _UiPaint_Text_line(coord OsV4, lineY int, lineEnd OsV2,
	value string, valueOrigEdit string,
	cd OsCd,
	textHeight, lineHeight, margin, marginX float64,
	font_path string,
	alignH, alignV int,
	selection, editable, tabIsChar, enableFormating bool) bool {

	lv := ui.GetCall()

	if textHeight <= 0 {
		textHeight = SKYALT_FONT_HEIGHT
	}
	textH := ui.CellWidth(textHeight)
	align := OsV2{int(alignH), int(alignV)}
	lineH := coord.Size.Y
	font := ui.win.fonts.Get(font_path)
	edit := &ui.edit
	keys := &ui.win.io.keys
	touch := &ui.win.io.touch

	active := false
	oldCursorPos := edit.end
	cursorPos := OsV2{-1, -1}

	if selection || editable {

		this_uid := lv.call
		edit_uid := edit.uid
		active = (edit_uid != nil && edit_uid == this_uid)
		enableFormating = !editable || !active

		touchPos, err := font.GetTouchPos(ui.win.io.touch.pos, value, coord, g_WinFont_DEFAULT_Weight, textH, align, enableFormating)
		if err != nil {
			fmt.Println("Error: VmDraw_Text.GetTextPos() failed: %w", err)
			return false
		}

		if (lv.call.data.over || lv.call.data.touch_active) || edit.setFirstEditbox {
			ui._UiPaint_TextSelectTouch(value, valueOrigEdit, OsV2{touchPos, lineY}, lineEnd, editable, font, textH, lineH, margin, marginX, enableFormating)
		}

		//this_uid = st.stack
		//edit_uid = edit.uid
		//active = (edit_uid != nil && edit_uid == this_uid)

		edit.last_edit = value
		if active {
			if lineY == edit.end.Y {
				ui._UiPaint_TextSelectKeys(value, lineY, lineEnd, editable, font, textH, lineH, margin, marginX, enableFormating)
			}

			if editable {
				value = ui._UiPaint_TextEditKeys(tabIsChar, font, textH, lineH, margin, marginX, enableFormating) //rewrite 'str' with temp value

				//enter or Tab(key) or outside => save
				isOutside := false
				if touch.start && !lv.call.data.touch_inside {
					uid := edit.uid
					isOutside = (uid != nil && uid == lv.call)
				}
				isEnter := keys.enter
				isEsc := keys.esc
				isTab := !tabIsChar && keys.tab

				if isTab || isEnter || isOutside || isEsc {

					if isEsc {
						//recover
						value = edit.orig
					} //else {
					//save
					//}

					//reset
					edit.uid = nil
					edit.temp = ""
				}
				if isTab {
					edit.tab = true //edit
				}

				//end
				cursorPos = edit.end

				edit.last_edit = value
			}

			//draw selection rectangle
			{
				s := edit.start
				e := edit.end

				if s.Y > e.Y {
					s, e = e, s //swap
				}

				var sx, ex int
				if s.Y != e.Y {
					//multi line
					sx = s.X
					ex = e.X
					if lineY == s.Y {
						ex = utf8.RuneCountInString(value)
					} else if lineY == e.Y {
						sx = 0
					} else if lineY > s.Y && lineY < e.Y {
						sx = 0
						ex = utf8.RuneCountInString(value)
					} else {
						sx = 0
						ex = 0
					}
				} else if lineY == s.Y {
					//one line
					sx = OsMin(s.X, e.X)
					ex = OsMax(s.X, e.X)
				}

				ui.buff.AddTextBack(OsV2{sx, ex}, value, coord, font, ui.buff.win.io.GetPalette().GetGrey(0.5), textH, align, false, enableFormating)
			}

			if enableFormating {
				_UiPaint_CheckSelectionExplode(value, &edit.start.X, &edit.end.X)
			}
		}
	}

	/*if syntaxtBack != nil {
		for _, it := range syntaxtBack.subs {
			root.ui.PaintTextBack(it, str, coord, font, it.GetColor(), textH, align, false, true)
		}
	}

	if syntaxtUnderline != nil {
		for _, it := range syntaxtUnderline.subs {
			root.ui.PaintTextBack(it, str, coord, font, it.GetColor(), textH, align, true, true)
		}
	}

	if syntaxtLabel != nil {
		for _, it := range syntaxtLabel.subs {
			root.ui.PaintTextTile(str, it, it, coord, font, it.GetColor(), textH, align)
		}
	}*/

	var cds []OsCd
	/*if syntaxtText != nil {
		strN := len(str)
		cds = root.ui.AllocColors(strN, cd)

		for _, it := range syntaxtText.subs {

			cdIt := it.GetColor()
			rng := it
			rng.Sort()
			for j := rng.X; j < strN && j < rng.Y; j++ {
				cds[j] = cdIt
			}
		}
	}*/

	// draw
	ui.buff.AddText(value, coord, font, cd, textH, align, cds, enableFormating)

	if cursorPos.X >= 0 {
		//cursor moved
		if !edit.end.Cmp(oldCursorPos) {
			ui.win.SetTextCursorMove()
		}

		var err error
		_ /*cCursorQuad*/, err = ui.buff.AddTextCursor(value, coord, font, cd, textH, align, cursorPos.X, ui.win.Cell(), enableFormating)
		if err != nil {
			fmt.Println("Error: VmDraw_Text.PaintTextCursor() failed: %w", err)
			return false
		}
	}

	return active
}
