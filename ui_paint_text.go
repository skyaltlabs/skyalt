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
	"strconv"
	"strings"
	"unicode/utf8"
)

func (ui *Ui) Paint_textGrid(
	frontCd OsCd, style *UiComp,
	value string, valueOrigEdit string,
	prop WinFontProps, icon *WinMedia,
	selection bool, editable bool,
	multi_line bool, multi_line_enter_finish bool) {

	lv := ui.GetCall()
	if lv.call == nil /*|| lv.call.crop.IsZero()*/ {
		return
	}

	if !style.enable {
		lv.call.touch_enabled = false
	}

	if !multi_line {
		lv.call.data.scrollH.narrow = true
		lv.call.data.scrollV.show = false
	} else {
		lv.call.data.scrollH.narrow = false
		lv.call.data.scrollV.show = true
	}

	var size OsV2f
	{
		var sizePx OsV2

		var mx, my int
		if multi_line {
			lines := strings.Split(value, "\n")
			mx = 0
			for _, ln := range lines {
				mx = OsMax(mx, ui.win.GetTextSize(-1, ln, prop).X)
			}
			my = OsMax(1, strings.Count(value, "\n")+1)
		} else {
			mx = ui.win.GetTextSize(-1, value, prop).X
			my = 1
		}
		sizePx = OsV2{mx, my * prop.lineH}

		size = sizePx.toV2f().DivV(float32((ui.win.Cell()))) //conver into cell
		size.X += 2 * 0.1                                    //because rect_border
		size.Y += 2 * 0.1
	}

	if !multi_line {
		size.Y -= 0.5 //make space for narrow h-scroll
	}

	size.X = OsMaxFloat32(size.X, float32(lv.call.canvas.Size.X)/float32((ui.win.Cell()))) //minimum sizeX is over whole div(user can click at the end)

	ui.Div_col(0, float64(size.X))
	ui.Div_row(0, float64(size.Y))
	ui.Div_rowMax(0, 100)

	ui.Div_start(0, 0, 1, 1)
	{
		coordImage, coordText := ui._compGetTextImageCoord(lv.call.canvas, 1, style.image_alignH, icon != nil, len(value) > 0)
		if icon != nil {
			ui._compDrawImage(coordImage, *icon, frontCd, style)
		}
		if editable || len(value) > 0 {
			ui._compDrawText(coordText, value, valueOrigEdit, frontCd, prop, selection, editable, style.label_align, multi_line, multi_line_enter_finish)
		}
	}
	ui.Div_end()
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

func (ui *Ui) _UiPaint_resetKeys(editable bool) {

	keys := &ui.win.io.keys

	//copy/cut/paste
	keys.copy = false
	keys.cut = false
	keys.paste = false

	//arrows
	keys.arrowU = false
	keys.arrowD = false
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

func _UiPaint_CursorMoveLR(text string, cursor int, move int, prop WinFontProps) int {

	//skip formating
	if prop.enableFormating {
		if move < 0 {
			cursor -= _UiPaint_HashFormatingPreSuf_fix(text[:cursor], strings.HasSuffix)
		}

		if move > 0 {
			cursor += _UiPaint_HashFormatingPreSuf_fix(text[cursor:], strings.HasPrefix)
		}
	}

	//shift rune
	if move < 0 {
		_, l := utf8.DecodeLastRuneInString(text[:cursor])
		cursor -= l
	}
	if move > 0 {
		_, l := utf8.DecodeRuneInString(text[cursor:])
		cursor += l
	}

	//check
	cursor = OsClamp(cursor, 0, len(text))

	return cursor
}

func _UiPaint_CursorMoveU(text string, lines []int, cursor int) int {
	y := _UiPaint_CursorLineY(lines, cursor)
	if y > 0 {
		_, pos := _UiPaint_CursorLine(text, lines, cursor)

		st, en := _UiPaint_PosLineRange(lines, y-1) //up line
		cursor = st + OsMin(pos, en-st)
	}
	return cursor
}
func _UiPaint_CursorMoveD(text string, lines []int, cursor int) int {
	y := _UiPaint_CursorLineY(lines, cursor)
	if y+1 < len(lines) {
		_, pos := _UiPaint_CursorLine(text, lines, cursor)

		st, en := _UiPaint_PosLineRange(lines, y+1) //down line
		cursor = st + OsMin(pos, en-st)
	}
	return cursor
}

func _UiPaint_CursorWordRange(text string, cursor int) (int, int) {
	start := 0
	end := 0

	text = strings.ToLower(text)

	p := 0
	for _, ch := range text {

		if OsIsTextWord(ch) {
			end = p + 1
		} else {
			if p < cursor {
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

func _UiPaint_Split(text string) []int {
	var ret []int
	for p, ch := range text {
		if ch == '\n' {
			ret = append(ret, p)
		}
	}
	ret = append(ret, len(text)) //last

	return ret
}
func _UiPaint_CursorLineY(lines []int, cursor int) int {
	for i, p := range lines {
		if cursor <= p {
			return i
		}
	}
	return len(lines) - 1
}

func _UiPaint_PosLineRange(lines []int, i int) (int, int) {
	var st, en int
	if i == 0 {
		st = 0
		en = lines[i]
	} else {
		st = lines[i-1] + 1 //+1 - after \n
		en = lines[i]
	}
	return st, en
}

func _UiPaint_CursorLineRange(lines []int, cursor int) (int, int) {
	i := _UiPaint_CursorLineY(lines, cursor)
	return _UiPaint_PosLineRange(lines, i)
}

func _UiPaint_CursorLine(text string, lines []int, cursor int) (string, int) {
	st, en := _UiPaint_CursorLineRange(lines, cursor)
	return text[st:en], cursor - st
}

func _UiPaint_GetLineYCrop(startY int, num_lines int, coord OsV4, crop OsV4, prop WinFontProps) (int, int) {

	sy := (crop.Start.Y - startY) / prop.lineH
	ey := OsRoundUp(float64(crop.End().Y-startY) / float64(prop.lineH))

	//check
	sy = OsClamp(sy, 0, num_lines-1)
	ey = OsClamp(ey, 0, num_lines)

	return sy, ey
}

func (ui *Ui) _UiPaint_Text_VScrollInto(lines []int, cursor int, prop WinFontProps) {

	lv := ui.GetCall()
	if lv.call.parent == nil {
		return
	}
	v_pos := _UiPaint_CursorLineY(lines, cursor) * prop.lineH

	v_st := lv.call.parent.data.scrollV.GetWheel()
	v_sz := lv.call.crop.Size.Y - prop.lineH - ui.CellWidth(2*0.1)
	v_en := v_st + v_sz

	if v_pos <= v_st {
		lv.call.parent.data.scrollV.SetWheel(OsMax(0, v_pos))
	} else if v_pos >= v_en {
		lv.call.parent.data.scrollV.wheel = OsMax(0, v_pos-v_sz) //SetWheel() has boundary check, which is not good here
	}
}

func (ui *Ui) _UiPaint_Text_HScrollInto(text string, lines []int, cursor int, prop WinFontProps) error {

	lv := ui.GetCall()
	if lv.call.parent == nil {
		return nil
	}

	ln, curr := _UiPaint_CursorLine(text, lines, cursor)
	h_pos := ui.win.GetTextSize(curr, ln, prop).X

	h_st := lv.call.parent.data.scrollH.GetWheel()
	h_sz := lv.call.crop.Size.X - ui.CellWidth(2*0.1) //text is shifted 0.1 to left
	h_en := h_st + h_sz

	if h_pos <= h_st {
		lv.call.parent.data.scrollH.SetWheel(OsMax(0, h_pos))
	} else if h_pos >= h_en {
		lv.call.parent.data.scrollH.wheel = OsMax(0, h_pos-h_sz) //SetWheel() has boundary check, which is not good here
	}
	return nil
}

func (ui *Ui) _UiPaint_TextSelectTouch(text string, lines []int, strEditOrig string, cursor int, editable bool, prop WinFontProps) {

	lv := ui.GetCall()
	if !lv.call.enableInput {
		return
	}

	edit := &ui.edit
	keys := &ui.win.io.keys
	touch := &ui.win.io.touch

	this_uid := lv.call
	edit_uid := edit.uid
	next_uid := edit.next_uid

	active := (edit_uid != nil && edit_uid == this_uid)
	activate_next_uid := (this_uid == next_uid)

	if touch.rm && lv.call.IsTouchInside(ui) && active && cursor >= OsMin(edit.start, edit.end) && cursor < OsMax(edit.start, edit.end) {
		return
	}

	if !ui.touch.IsScrollOrResizeActive() && ((editable && edit.setFirstEditbox) || (editable && edit.tab) || activate_next_uid) {
		//setFirstEditbox or Tab
		edit.uid = this_uid

		if !active {
			edit.temp = strEditOrig
			edit.orig = strEditOrig
		}

		if !activate_next_uid {
			//select all
			edit.start = 0
			edit.end = len(text)
		}

		edit.setFirstEditbox = false
		edit.next_uid = nil
		edit.tab = false

		ui.win.SetTextCursorMove()
	} else if lv.call.IsTouchInside(ui) && touch.start {
		//click inside
		if !active {
			edit.next_uid = this_uid //set next_uid
		}
		//set end
		edit.end = cursor

		if !active || !keys.shift {
			//set start
			edit.start = cursor
		}

		switch touch.numClicks {
		case 2:
			st, en := _UiPaint_CursorWordRange(text, cursor)
			edit.start = st //set start
			edit.end = en   //set end
		case 3:
			st, en := _UiPaint_CursorLineRange(lines, cursor)
			edit.start = st //set start
			edit.end = en   //set end
		}
	}

	//keep selecting
	if active && lv.call.IsTouchActive(ui) && (touch.numClicks != 2 && touch.numClicks != 3) {
		edit.end = cursor //set end

		//scroll
		ui._UiPaint_Text_VScrollInto(lines, cursor, prop)
		ui._UiPaint_Text_HScrollInto(text, lines, cursor, prop)

		//root.buff.ResetHost() //SetNoSleep()
	}
}

func (ui *Ui) _UiPaint_TextSelectKeys(text string, lines []int, editable bool, prop WinFontProps, multi_line bool) {

	touch := &ui.win.io.touch
	keys := &ui.win.io.keys
	edit := &ui.edit

	s := &edit.start
	e := &edit.end

	old := *e

	if editable {
		text = edit.temp
	}

	//context dialog
	{
		lv := ui.GetCall()
		dnm := strconv.Itoa(int(lv.call.data.hash))
		if lv.call.IsTouchInside(ui) && touch.end && touch.rm {
			ui.Dialog_open(dnm, 2)
		}
		if ui.Dialog_start(dnm) {
			ui.Div_col(0, 5)
			if ui.Comp_buttonMenu(0, 0, 1, 1, "Copy", "", true, false) > 0 {
				keys.copy = true
				ui.Dialog_close()
			}
			if ui.Comp_buttonMenu(0, 1, 1, 1, "Cut", "", editable, false) > 0 {
				keys.cut = true
				ui.Dialog_close()
			}
			if ui.Comp_buttonMenu(0, 2, 1, 1, "Paste", "", editable, false) > 0 {
				keys.paste = true
				ui.win.RefreshClipboard()
				ui.Dialog_close()
			}
			ui.Dialog_end()
		}
	}

	//select all
	if keys.selectAll {
		*s = 0
		*e = len(text)
	}

	//copy, cut
	if keys.copy || keys.cut {
		if keys.shift {
			keys.clipboard = _UiPaint_RemoveFormating(text)
		} else {
			firstCur := OsTrn(*s < *e, *s, *e)
			lastCur := OsTrn(*s > *e, *s, *e)

			keys.clipboard = text[firstCur:lastCur]
		}
	}

	//shift
	if keys.shift {
		//ctrl
		if keys.ctrl {
			if keys.arrowL {
				p := _UiPaint_CursorMoveLR(text, *e, -1, prop)
				first, _ := _UiPaint_CursorWordRange(text, p)
				if first == p && p > 0 {
					first, _ = _UiPaint_CursorWordRange(text, p-1)
				}
				*e = first
			}
			if keys.arrowR {
				p := _UiPaint_CursorMoveLR(text, *e, +1, prop)
				_, last := _UiPaint_CursorWordRange(text, p)
				if last == p && p+1 < len(text) {
					_, last = _UiPaint_CursorWordRange(text, p+1)
				}
				*e = last
			}
		} else {
			if multi_line {
				if keys.arrowU {
					*e = _UiPaint_CursorMoveU(text, lines, *e)
				}
				if keys.arrowD {
					*e = _UiPaint_CursorMoveD(text, lines, *e)
				}
			}

			if keys.arrowL {
				p := _UiPaint_CursorMoveLR(text, *e, -1, prop)
				*e = p
			}
			if keys.arrowR {
				p := _UiPaint_CursorMoveLR(text, *e, +1, prop)
				*e = p
			}
		}

		//home & end
		if keys.home {
			*e, _ = _UiPaint_CursorLineRange(lines, *e) //line start
		}
		if keys.end {
			_, *e = _UiPaint_CursorLineRange(lines, *e) //line end
		}
	}

	//scroll
	newPos := *e
	if old != newPos {
		ui._UiPaint_Text_VScrollInto(lines, newPos, prop)
	}
	if old != newPos {
		ui._UiPaint_Text_HScrollInto(text, lines, newPos, prop)
	}
}

func (ui *Ui) _UiPaint_TextEditKeys(text string, lines []int, tabIsChar bool, prop WinFontProps, multi_line bool, multi_line_enter_finish bool) (string, bool) {
	edit := &ui.edit
	keys := &ui.win.io.keys

	shiftKey := keys.shift

	uid := edit.uid

	s := &edit.start
	e := &edit.end
	old := *e

	firstCur := OsTrn(*s < *e, *s, *e)
	lastCur := OsTrn(*s > *e, *s, *e)

	//cut/paste(copy() is in selectKeys)
	if keys.cut {
		//remove
		text = text[:firstCur] + text[lastCur:]
		edit.temp = text

		//select
		*s = firstCur
		*e = firstCur
	} else if keys.paste {
		//remove old selection
		if *s != *e {
			text = text[:firstCur] + text[lastCur:]
		}

		//insert
		cb := keys.clipboard
		text = text[:firstCur] + cb + text[firstCur:]
		edit.temp = text

		firstCur += len(cb)
		*s = firstCur
		*e = firstCur
	}

	//when dialog is active, don't edit
	lv := ui.GetCall()
	if !lv.call.enableInput {
		return edit.temp, old != *e
	}

	//insert text
	txt := keys.text
	if tabIsChar && keys.tab {
		txt += "\t"
	}

	if keys.enter && multi_line && ((multi_line_enter_finish && keys.ctrl) || (!multi_line_enter_finish && !keys.ctrl)) {
		txt = "\n"
	}

	if len(txt) > 0 {
		//remove old selection
		if *s != *e {
			text = text[:firstCur] + text[lastCur:]
			*e = *s
		}

		//insert
		text = text[:firstCur] + txt + text[firstCur:]
		edit.temp = text

		//cursor
		firstCur += len(txt)
		*s = firstCur
		*e = firstCur

		//reset
		keys.text = ""
	}

	//delete/backspace
	if *s != *e {
		if keys.delete || keys.backspace {

			//remove
			text = text[:firstCur] + text[lastCur:]
			edit.temp = text

			//cursor
			*s = firstCur
			*e = firstCur
		}
	} else {
		if keys.backspace {
			//remove
			if *s > 0 {
				//removes one letter
				p := _UiPaint_CursorMoveLR(text, firstCur, -1, prop)
				text = text[:p] + text[firstCur:]
				edit.temp = text

				//cursor
				firstCur = p
				*s = firstCur
				*e = firstCur
			}
		} else if keys.delete {
			//remove
			if *s < len(text) {
				//removes one letter
				p := _UiPaint_CursorMoveLR(text, firstCur, +1, prop)
				text = text[:firstCur] + text[p:]
				edit.temp = text
			}
		}
	}

	if !shiftKey {
		//arrows
		if *s != *e {
			if multi_line {
				if keys.arrowU {
					firstCur = _UiPaint_CursorMoveU(text, lines, *e)
					*s = firstCur
					*e = firstCur
				}
				if keys.arrowD {
					firstCur = _UiPaint_CursorMoveD(text, lines, *e)
					*s = firstCur
					*e = firstCur
				}
			}

			if keys.arrowL {
				//from select -> single start
				*s = firstCur
				*e = firstCur
			} else if keys.arrowR {
				//from select -> single end
				*s = lastCur
				*e = lastCur
			}
		} else {
			if keys.ctrl {
				if keys.arrowL {
					p := _UiPaint_CursorMoveLR(text, *s, -1, prop)
					first, _ := _UiPaint_CursorWordRange(text, p)
					if first == p && p > 0 {
						first, _ = _UiPaint_CursorWordRange(text, p-1)
					}
					*s = first
					*e = first
				}
				if keys.arrowR {
					p := _UiPaint_CursorMoveLR(text, *s, +1, prop)
					_, last := _UiPaint_CursorWordRange(text, p)
					if last == p && p+1 < len(text) {
						_, last = _UiPaint_CursorWordRange(text, p+1)
					}
					*s = last
					*e = last
				}
			} else {
				if multi_line {
					if keys.arrowU {
						p := _UiPaint_CursorMoveU(text, lines, *e)
						*s = p
						*e = p
					}
					if keys.arrowD {
						p := _UiPaint_CursorMoveD(text, lines, *e)
						*s = p
						*e = p
					}
				}

				if keys.arrowL {
					p := _UiPaint_CursorMoveLR(text, *s, -1, prop)
					*s = p
					*e = p
				} else if keys.arrowR {
					p := _UiPaint_CursorMoveLR(text, *s, +1, prop)
					*s = p
					*e = p
				}
			}
		}

		//home/end
		if keys.home {
			if multi_line {
				firstCur, _ = _UiPaint_CursorLineRange(lines, *e) //line start
			} else {
				firstCur = 0
			}
			*s = firstCur
			*e = firstCur
		} else if keys.end {
			if multi_line {
				_, firstCur = _UiPaint_CursorLineRange(lines, *e) //line start
			} else {
				firstCur = len(text)
			}

			*s = firstCur
			*e = firstCur
		}
	}

	//history
	{
		//app := stt.GetApp()
		his := UiPaintTextHistoryItem{str: text, cur: *e}

		ui.edit_history.FindOrAdd(uid, his).AddWithTimeOut(his)

		if keys.backward {
			his = ui.edit_history.FindOrAdd(uid, his).Backward(his)
			edit.temp = his.str
			*s = his.cur
			*e = his.cur
		}
		if keys.forward {
			his = ui.edit_history.FindOrAdd(uid, his).Forward()
			edit.temp = his.str
			*s = his.cur
			*e = his.cur
		}
	}

	return edit.temp, old != *e
}

func (ui *Ui) _UiPaint_Text_line(coord OsV4,
	value string, valueOrigEdit string,
	frontCd OsCd,
	prop WinFontProps,
	font_path string,
	align OsV2,
	selection, editable, tabIsChar bool,
	multi_line bool, multi_line_enter_finish bool) bool {

	lv := ui.GetCall()

	edit := &ui.edit
	keys := &ui.win.io.keys
	touch := &ui.win.io.touch

	active := false
	oldCursor := edit.end
	cursorPos := -1

	lines := _UiPaint_Split(value)
	startY := ui.win.GetTextStart(value, prop, coord, align, len(lines)).Y

	if selection || editable {
		this_uid := lv.call
		edit_uid := edit.uid
		active = (edit_uid != nil && edit_uid == this_uid)
		prop.enableFormating = !editable || !active

		//touch
		if (lv.call.IsOver(ui) || lv.call.IsTouchActive(ui)) || edit.setFirstEditbox {

			var touchCursor int
			if multi_line {
				y := (ui.win.io.touch.pos.Y - startY) / prop.lineH
				y = OsClamp(y, 0, len(lines)-1)

				st, en := _UiPaint_PosLineRange(lines, y)
				touchCursor = st + ui.win.GetTextPos(ui.win.io.touch.pos.X, value[st:en], prop, coord, align)
			} else {
				touchCursor = ui.win.GetTextPos(ui.win.io.touch.pos.X, value, prop, coord, align)
			}

			ui._UiPaint_TextSelectTouch(value, lines, valueOrigEdit, touchCursor, editable, prop)
		}

		edit.last_edit = value
		if active {
			ui._UiPaint_TextSelectKeys(value, lines, editable, prop, multi_line)

			if editable {
				var tryMoveScroll bool
				value, tryMoveScroll = ui._UiPaint_TextEditKeys(edit.temp, lines, tabIsChar, prop, multi_line, multi_line_enter_finish) //rewrite 'str' with temp value

				lines = _UiPaint_Split(value) //refresh

				if tryMoveScroll {
					ui._UiPaint_Text_VScrollInto(lines, edit.end, prop)
					ui._UiPaint_Text_HScrollInto(value, lines, edit.end, prop)
				}

				isTab := !tabIsChar && keys.tab
				if isTab {
					edit.tab = true //edit
				}

				//end
				cursorPos = edit.end

				edit.last_edit = value
			}

			//enter or Tab(key) or outside => save
			isOutside := false
			if touch.start && lv.call.enableInput && !lv.call.IsTouchInside(ui) {
				uid := edit.uid
				isOutside = (uid != nil && uid == lv.call)
			}
			isEnter := keys.enter && (!multi_line || (multi_line_enter_finish && !keys.ctrl) || (!multi_line_enter_finish && keys.ctrl))
			isEsc := keys.esc
			isTab := !tabIsChar && keys.tab

			if isTab || isEnter || isOutside || isEsc {
				if isEsc {
					value = edit.orig
				}

				//reset
				edit.uid = nil
				edit.temp = ""
			}

			//draw selection
			curr_sx := OsMin(edit.start, edit.end)
			curr_ex := OsMax(edit.start, edit.end)

			if multi_line {
				curr_sy := _UiPaint_CursorLineY(lines, curr_sx)
				curr_ey := _UiPaint_CursorLineY(lines, curr_ex)
				if curr_sy > curr_ey {
					curr_sy, curr_ey = curr_ey, curr_sy //swap
				}

				crop_sy, crop_ey := _UiPaint_GetLineYCrop(startY, len(lines), coord, lv.call.crop, prop) //only rows which are on screen

				yst := OsMax(curr_sy, crop_sy)
				yen := OsMin(curr_ey, crop_ey)

				for y := yst; y <= yen; y++ { //less or equal!

					st, en := _UiPaint_PosLineRange(lines, y)
					ln := value[st:en]

					sx, ex := 0, len(ln) //whole line
					if y == curr_sy {    //first line
						_, sx = _UiPaint_CursorLine(value, lines, curr_sx)
					}
					if y == curr_ey { //last line
						_, ex = _UiPaint_CursorLine(value, lines, curr_ex)
					}

					ui.buff.AddTextBack(OsV2{sx, ex}, ln, prop, coord, ui.win.io.GetPalette().GetGrey(0.5), align, false, y, len(lines))
				}
			} else {
				ui.buff.AddTextBack(OsV2{curr_sx, curr_ex}, value, prop, coord, ui.win.io.GetPalette().GetGrey(0.5), align, false, 0, 1)
			}

			if prop.enableFormating {
				_UiPaint_CheckSelectionExplode(value, &edit.start, &edit.end)
			}
		}
	}

	// draw
	if multi_line {
		sy, ey := _UiPaint_GetLineYCrop(startY, len(lines), coord, lv.call.crop, prop) //only rows which are on screen
		for y := sy; y < ey; y++ {
			st, en := _UiPaint_PosLineRange(lines, y)
			ui.buff.AddText(value[st:en], prop, coord, frontCd, align, y, len(lines))
		}

	} else {
		ui.buff.AddText(value, prop, coord, frontCd, align, 0, 1)
	}

	// draw cursor
	if cursorPos >= 0 {
		//cursor moved
		if edit.end != oldCursor {
			ui.win.SetTextCursorMove()
		}

		if multi_line {
			y := _UiPaint_CursorLineY(lines, cursorPos)
			ln, ln_cursorPos := _UiPaint_CursorLine(value, lines, cursorPos)

			ui.buff.AddTextCursor(ln, prop, coord, frontCd, align, ln_cursorPos, y, len(lines))
		} else {
			ui.buff.AddTextCursor(value, prop, coord, frontCd, align, cursorPos, 0, 1)
		}
	}

	return active
}
