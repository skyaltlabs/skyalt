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
	"math"
	"strconv"
	"strings"
	"unicode/utf8"
)

type UiComp struct {
	enable bool
	shape  uint8 //0=rect, 1=sphere
	cd     uint8
	fade   bool
	//enable_back     bool
	//enable_border   bool
	label_alignV    uint8
	label_alignH    uint8
	label_formating bool
	image_alignV    uint8
	image_alignH    uint8
	image_fill      bool
	image_margin    float64
	tooltip         string
}

func (ui *Ui) _compIsClicked(enable bool) (int, int, bool, bool, bool) {
	var click, rclick int
	var inside, active, end bool
	if enable {
		lv := ui.GetCall()
		inside = lv.call.IsTouchInside(ui)
		active = lv.call.IsTouchActive(ui)
		end = lv.call.IsTouchEnd(ui)

		force := ui.win.io.touch.rm

		if inside && end {
			click = 1
			rclick = OsTrn(force, 1, 0)
		}

		if click > 0 {
			click = int(ui.win.io.touch.numClicks)
		}
		if rclick > 0 {
			rclick = int(ui.win.io.touch.numClicks)
		}
	}

	return click, rclick, inside, active, end
}

func (ui *Ui) _compGetTextImageCoord(coord OsV4, image_width float64, imageAlignH uint8, isImg bool, isText bool) (OsV4, OsV4) {

	lv := ui.GetCall()

	coordImg := coord
	coordText := coord

	//isImg := (len(image_url) > 0)
	//isText := (len(text) > 0 || textEdit)

	if isImg && isText {
		w := float64(ui.win.Cell()) / float64(lv.call.canvas.Size.X) * image_width

		switch imageAlignH {
		case 0: //left
			coordImg = coord.Cut(0, 0, w, 1)
			coordText = coord.Cut(w, 0, 1-w, 1)
		case 1: //center
			coordImg = coord
			coordText = coord
		default: //right
			coordImg = coord.Cut(1-w, 0, w, 1)
			coordText = coord.Cut(0, 0, 1-w, 1)
		}
	}

	return coordImg, coordText
}

func (ui *Ui) _compDrawShape(coord OsV4, shape uint8, cd OsCd, margin float64, border float64) OsV4 {
	//
	//st := root.ui.GetStack()
	//coord := st.stack.canvas

	b := ui.CellWidth(border)
	m := ui.CellWidth(margin)
	coord = coord.Inner(m, m, m, m)

	if cd.A > 0 {
		switch shape {
		case 0:
			ui.buff.AddRect(coord, cd, b)
		case 1:
			ui.buff.AddCircle(coord, cd, b)
		}
	}
	return coord
}

func (ui *Ui) _compDrawImage(coord OsV4, icon string, cd OsCd, style *UiComp) {

	lv := ui.GetCall()

	path, err := InitWinMedia(icon)
	if err != nil {
		lv.call.data.app.AddLogErr(err)
	} else {
		m := ui.CellWidth(style.image_margin)
		coord = coord.Inner(m, m, m, m)

		//style.image_fill = true

		imgRectBackup := ui.buff.AddCrop(lv.call.crop.GetIntersect(coord))
		ui.buff.AddImage(path, coord, cd, int(style.image_alignV), int(style.image_alignH), style.image_fill)
		ui.buff.AddCrop(imgRectBackup)
	}
}

func (ui *Ui) _compDrawText(coord OsV4, value string, valueOrigEdit string, cd OsCd, textH float64, selection bool, editable bool, alignH int, alignV int, formating bool) {

	lv := ui.GetCall()

	coord = coord.AddSpaceX(ui.CellWidth(0.1))

	// crop
	imgRectBackup := ui.buff.AddCrop(lv.call.crop.GetIntersect(coord))

	//one liner
	active := ui._UiPaint_Text_line(coord, 0, OsV2{utf8.RuneCountInString(value), 0},
		value, valueOrigEdit,
		cd,
		textH, 1, 0, 0,
		SKYALT_FONT_PATH, alignH, alignV,
		selection, editable, true, formating)

	if active {
		ui._UiPaint_resetKeys(false)
	}

	if selection {
		lv := ui.GetCall()
		if lv.call.IsTouchInside(ui) {
			ui.Paint_cursor("ibeam")
		}
	}

	// crop back
	ui.buff.AddCrop(imgRectBackup)
}

func (ui *Ui) _buttonBasicStyle(enable bool, tooltip string) UiComp {
	var style UiComp
	style.tooltip = tooltip
	style.enable = enable
	style.cd = CdPalette_P
	style.label_alignH = 1
	style.label_alignV = 1
	style.image_alignH = 0
	style.image_alignV = 1
	return style
}

func (ui *Ui) Comp_button(x, y, w, h int, label string, tooltip string, enable bool) int {
	ui.Div_start(x, y, w, h)

	style := ui._buttonBasicStyle(enable, tooltip)
	click, rclick := ui.Comp_button_s(&style, label, "", "", 1, false)

	ui.Div_end()
	if rclick > 0 {
		return 2
	} else if click > 0 {
		return 1
	}
	return 0
}

func (ui *Ui) Comp_buttonIcon(x, y, w, h int, icon string, icon_margin float64, tooltip string, enable bool) int {
	ui.Div_start(x, y, w, h)

	style := ui._buttonBasicStyle(enable, tooltip)
	style.image_alignH = 1
	style.image_margin = icon_margin
	click, rclick := ui.Comp_button_s(&style, "", icon, "", 0, false)

	ui.Div_end()
	if rclick > 0 {
		return 2
	} else if click > 0 {
		return 1
	}
	return 0
}

func (ui *Ui) Comp_buttonLight(x, y, w, h int, label string, tooltip string, enable bool) int {
	ui.Div_start(x, y, w, h)

	style := ui._buttonBasicStyle(enable, tooltip)
	click, rclick := ui.Comp_button_s(&style, label, "", "", 0.5, false)

	ui.Div_end()
	if rclick > 0 {
		return 2
	} else if click > 0 {
		return 1
	}
	return 0
}

func (ui *Ui) Comp_buttonText(x, y, w, h int, label string, url string, tooltip string, enable bool, selected bool) int {
	ui.Div_start(x, y, w, h)

	style := ui._buttonBasicStyle(enable, tooltip)
	click, rclick := ui.Comp_button_s(&style, label, "", url, OsTrnFloat(selected, 1, 0), false)

	ui.Div_end()
	if rclick > 0 {
		return 2
	} else if click > 0 {
		return 1
	}
	return 0
}

func (ui *Ui) Comp_buttonTextFade(x, y, w, h int, label string, url string, tooltip string, enable bool, selected bool, fade bool) int {
	ui.Div_start(x, y, w, h)

	style := ui._buttonBasicStyle(enable, tooltip)
	style.fade = fade
	click, rclick := ui.Comp_button_s(&style, label, "", url, OsTrnFloat(selected, 1, 0), false)

	ui.Div_end()
	if rclick > 0 {
		return 2
	} else if click > 0 {
		return 1
	}
	return 0
}

func (ui *Ui) Comp_buttonOutlined(x, y, w, h int, label string, tooltip string, enable bool, selected bool) int {
	ui.Div_start(x, y, w, h)

	style := ui._buttonBasicStyle(enable, tooltip)
	click, rclick := ui.Comp_button_s(&style, label, "", "", OsTrnFloat(selected, 1, 0), true)

	ui.Div_end()
	if rclick > 0 {
		return 2
	} else if click > 0 {
		return 1
	}
	return 0
}

func (ui *Ui) Comp_buttonOutlinedFade(x, y, w, h int, label string, tooltip string, enable bool, selected bool, fade bool) int {
	ui.Div_start(x, y, w, h)

	style := ui._buttonBasicStyle(enable, tooltip)
	style.fade = fade
	click, rclick := ui.Comp_button_s(&style, label, "", "", OsTrnFloat(selected, 1, 0), true)

	ui.Div_end()
	if rclick > 0 {
		return 2
	} else if click > 0 {
		return 1
	}
	return 0
}

func (ui *Ui) Comp_buttonMenuIcon(x, y, w, h int, label string, icon string, iconMargin float64, tooltip string, enable bool, selected bool) int {
	ui.Div_start(x, y, w, h)

	style := ui._buttonBasicStyle(enable, tooltip)
	style.label_alignH = 0
	style.image_margin = iconMargin
	if !selected {
		style.cd = CdPalette_B
	}

	click, rclick := ui.Comp_button_s(&style, label, icon, "", OsTrnFloat(selected, 1, 0), false)

	ui.Div_end()
	if rclick > 0 {
		return 2
	} else if click > 0 {
		return 1
	}
	return 0
}

func (ui *Ui) Comp_buttonMenu(x, y, w, h int, label string, tooltip string, enable bool, selected bool) int {
	return ui.Comp_buttonMenuIcon(x, y, w, h, label, "", 0, tooltip, enable, selected)
}

func (ui *Ui) Comp_button_s(style *UiComp, value string, icon string, url string, drawBack float64, drawBorder bool) (int, int) {

	lv := ui.GetCall()

	click, rclick, inside, active, _ := ui._compIsClicked(style.enable)
	if click > 0 && len(url) > 0 {
		//SA_DialogStart() warning which open dialog ...
		OsUlit_OpenBrowser(url)
	}

	pl := ui.buff.win.io.GetPalette()
	cd, onCd := pl.GetCd(style.cd, style.fade, style.enable, inside, active)

	//coord := st.stack.data.Convert(root.ui.Cell(), grid)
	//coord.Start = st.stack.canvas.Start.Add(coord.Start)
	coord := lv.call.canvas

	//if selectedMenu {
	//	drawBorder = true
	//}

	if drawBack == 0 {
		if style.cd != CdPalette_B {
			onCd = cd
		}

		if style.enable && len(value) > 0 { //no background for icons. For text yes
			if inside || active {
				//not same color as background
				if style.cd == CdPalette_B {
					cd = pl.GetGrey(0.3)
				}
				cd.A = 100
				drawBack = 1
			}
		}
	}

	m := ui.CellWidth(0.03)
	coord = coord.Inner(m, m, m, m)

	//background
	if drawBack > 0 {
		//light
		if drawBack <= 0.6 {
			onCd = cd
			cd.A = 30
		}

		margin := OsTrnFloat(drawBorder, 0.06, 0)
		ui._compDrawShape(coord, style.shape, cd, margin, 0)
	}

	if drawBorder {
		ui._compDrawShape(coord, style.shape, cd, 0, 0.03)
	}

	if len(icon) > 0 {
		style.image_alignH = 0
	}

	coordImage, coordText := ui._compGetTextImageCoord(coord, 1, style.image_alignH, len(icon) > 0, len(value) > 0)
	if len(icon) > 0 {
		ui._compDrawImage(coordImage, icon, onCd, style)
	}
	if len(value) > 0 {
		ui._compDrawText(coordText, value, "", onCd, SKYALT_FONT_HEIGHT, false, false, int(style.label_alignH), int(style.label_alignV), style.label_formating)
	}

	if style.enable {
		if inside {
			ui.Paint_cursor("hand")
		}

		if len(style.tooltip) > 0 {
			ui.Paint_tooltip(0, 0, 1, 1, style.tooltip)
		} else if len(url) > 0 {
			ui.Paint_tooltip(0, 0, 1, 1, url)
		}
	}

	return click, rclick
}

func (ui *Ui) Comp_textIcon(x, y, w, h int, label string, icon string, iconMargin float64) {
	ui.Div_start(x, y, w, h)

	var style UiComp
	style.enable = true
	style.cd = CdPalette_B
	style.label_alignV = 1
	style.image_alignV = 1
	style.image_margin = iconMargin

	ui.Comp_text_s(&style, label, icon, false)

	ui.Div_end()
}

func (ui *Ui) Comp_text(x, y, w, h int, label string, alignH int) *UiLayoutDiv {
	ui.Div_start(x, y, w, h)
	div := ui.GetCall().call

	var style UiComp
	style.enable = true
	style.cd = CdPalette_B
	style.label_alignV = 1
	style.label_alignH = uint8(alignH)

	ui.Comp_text_s(&style, label, "", true)

	ui.Div_end()
	return div
}

func (ui *Ui) Comp_textSelect(x, y, w, h int, label string, alignH int, selection bool) *UiLayoutDiv {
	ui.Div_start(x, y, w, h)
	div := ui.GetCall().call

	var style UiComp
	style.enable = true
	style.cd = CdPalette_B
	style.label_alignV = 1
	style.label_alignH = uint8(alignH)

	ui.Comp_text_s(&style, label, "", selection)

	ui.Div_end()
	return div
}

func (ui *Ui) Comp_text_s(style *UiComp, value string, icon string, selection bool) {

	pl := ui.buff.win.io.GetPalette()
	_, onCd := pl.GetCd(style.cd, style.fade, style.enable, false, false)

	//if style.enable_back {
	//ui.buff.AddRect(st.stack.canvas, cd, 0)
	//}
	//if style.enable_border {
	//ui.buff.AddRect(st.stack.canvas, pl.GetGrey(0.8), ui.CellWidth(0.03))
	//}

	if style.enable {
		if len(style.tooltip) > 0 {
			ui.Paint_tooltip(0, 0, 1, 1, style.tooltip)
		}
	}

	ui.Paint_textGrid(InitOsV4(0, 0, 1, 1), onCd, style, value, "", icon, selection, false)
}

func (ui *Ui) Comp_editbox_desc(description string, description_alignH int, width float64, x, y, w, h int, valueIn interface{}, value_precision int, icon string, ghost string, highlight bool, tempToValue bool, enable bool) (string, bool, bool, bool, *UiLayoutDiv) {

	ui.Div_start(x, y, w, h)

	xx := 0

	if width > 0 {
		//1 row
		ui.Div_col(0, width)
		ui.Div_colMax(1, 100)
		ui.Comp_text(0, 0, 1, 1, description, description_alignH)
		xx = 1
	} else {
		//2 rows
		ui.Div_colMax(0, 100)
		ui.Comp_text(0, 0, 1, 1, description, description_alignH)
	}

	editedValue, active, changed, finished, div := ui.Comp_editbox(xx, 0, 1, 1, valueIn, value_precision, icon, ghost, highlight, tempToValue, enable)

	ui.Div_end()

	return editedValue, active, changed, finished, div
}

func (ui *Ui) Comp_editbox(x, y, w, h int, valueIn interface{}, value_precision int, icon string, ghost string, highlight bool, tempToValue bool, enable bool) (string, bool, bool, bool, *UiLayoutDiv) {

	ui.Div_start(x, y, w, h)
	div := ui.GetCall().call

	value := ""
	switch v := valueIn.(type) {
	case *float32:
		value = strconv.FormatFloat(float64(*v), 'f', value_precision, 64)
	case *float64:
		value = strconv.FormatFloat(*v, 'f', value_precision, 64)
	case *int:
		value = strconv.Itoa(*v)
	case *string:
		value = *v
		//int8/16/32, uint8, byte, etc ...
	}

	var style UiComp
	style.enable = enable
	style.cd = CdPalette_B
	style.label_alignH = 0
	style.label_alignV = 1

	editedValue, active, changed, finished := ui.Comp_edit_s(&style, value, value, icon, ghost, highlight, tempToValue)

	if finished || tempToValue {
		switch v := valueIn.(type) {
		case *float32:
			vv, _ := strconv.ParseFloat(editedValue, 64)
			*v = float32(vv)
		case *float64:
			*v, _ = strconv.ParseFloat(editedValue, 64)
		case *int:
			*v, _ = strconv.Atoi(editedValue)
		case *string:
			*v = editedValue
			//int8/16/32, uint8, byte, etc ...
		}
	}

	ui.Div_end()

	return editedValue, active, changed, finished, div
}
func (ui *Ui) Comp_edit_s(style *UiComp, valueIn string, valueInOrig string, icon string, ghost string, highlight bool, tempToValue bool) (string, bool, bool, bool) {

	lv := ui.GetCall()

	lv.call.data.scrollH.narrow = true
	lv.call.data.scrollV.show = false

	pl := ui.buff.win.io.GetPalette()
	cd, onCd := pl.GetCd(style.cd, style.fade, style.enable, false, false)

	if highlight {
		cd, onCd = pl.GetCd(CdPalette_T, false, style.enable, false, false)
	}

	edit := &ui.edit

	inDiv := lv.call.FindOrCreate("", InitOsV4(0, 0, 1, 1), ui.GetLastApp())
	this_uid := inDiv //.Hash()
	edit_uid := edit.uid
	active := (edit_uid != nil && edit_uid == this_uid)

	if active {
		edit.tempToValue = tempToValue
	}

	var value string
	if active {
		value = edit.temp
	} else {
		value = valueIn
	}
	inDiv.touch_enabled = style.enable

	coord := lv.call.canvas
	coord = coord.AddSpace(ui.CellWidth(0.03))

	//if style.enable_back || highlight {
	ui.buff.AddRect(coord, cd, 0)
	//}
	//if style.enable_border
	{
		w := ui.CellWidth(0.03)
		if active {
			w *= 2
		}
		ui.buff.AddRect(coord, pl.P, w)
	}

	ui.Paint_textGrid(InitOsV4(0, 0, 1, 1), onCd, style, value, valueInOrig, icon, true, true)

	//ghost
	if len(edit.last_edit) == 0 && len(ghost) > 0 {
		//ghostStyle := style
		//ghostStyle.label_alignH = 1 //center
		ui._compDrawText(coord, ghost, "", pl.GetGrey(0.7), SKYALT_FONT_HEIGHT, false, false, 1, int(style.label_alignV), style.label_formating)
	}

	if style.enable {
		if len(style.tooltip) > 0 {
			ui.Paint_tooltip(0, 0, 1, 1, style.tooltip)
		}
	}

	return edit.last_edit, active, (active && value != edit.last_edit), (active && this_uid != edit.uid)
}

func (ui *Ui) Comp_progress(style *UiComp, value float64, prec int) int64 {

	lv := ui.GetCall()

	value = OsClampFloat(value, 0, 1)

	pl := ui.buff.win.io.GetPalette()
	cd, onCd := pl.GetCd(style.cd, style.fade, style.enable, false, false)

	coord := lv.call.canvas

	//border
	ui._compDrawShape(coord, style.shape, cd, 0.03, 0.03)
	//back
	coord = ui.getCoord(0, 0, value, 1, 0, 0, 0)
	ui._compDrawShape(coord, style.shape, cd, 0.09, 0)
	//label
	//ui.Paint_textGrid(coord, onCd, style, strconv.FormatFloat(value*100, 'f', prec, 64)+"%", "", "", true, false)
	ui._compDrawText(coord, strconv.FormatFloat(value*100, 'f', prec, 64)+"%", "", onCd, SKYALT_FONT_HEIGHT, true, false, int(style.label_alignH), int(style.label_alignV), style.label_formating)

	if style.enable {
		if len(style.tooltip) > 0 {
			ui.Paint_tooltip(0, 0, 1, 1, style.tooltip)
		}
	}

	return 1
}

func (ui *Ui) Comp_slider(x, y, w, h int, value *float64, minValue float64, maxValue float64, jumpValue float64) bool {

	ui.Div_start(x, y, w, h)

	styleSlider := UiComp{enable: true, label_formating: true, cd: CdPalette_P}
	_, changed, _ := ui.Comp_slider_s(&styleSlider, value, minValue, maxValue, jumpValue, "", 0)

	ui.Div_end()

	return changed
}

func (ui *Ui) Comp_slider_desc(description string, description_alignH int, width float64, x, y, w, h int, value *float64, minValue float64, maxValue float64, jumpValue float64) bool {

	ui.Div_start(x, y, w, h)

	xx := 0

	if width > 0 {
		//1 row
		ui.Div_col(0, width)
		ui.Div_colMax(1, 100)
		ui.Comp_text(0, 0, 1, 1, description, description_alignH)
		xx = 1
	} else {
		//2 rows
		ui.Div_colMax(0, 100)
		ui.Comp_text(0, 0, 1, 1, description, description_alignH)
	}

	changed := ui.Comp_slider(xx, 0, 1, 1, value, minValue, maxValue, jumpValue)

	ui.Div_end()

	return changed
}

func (ui *Ui) Comp_slider_s(style *UiComp, value *float64, minValue float64, maxValue float64, jumpValue float64, imgPath string, imgMargin float64) (bool, bool, bool) {

	lv := ui.GetCall()

	old_value := *value

	_, _, inside, active, end := ui._compIsClicked(style.enable)

	pl := ui.buff.win.io.GetPalette()
	cd, _ := pl.GetCd(style.cd, style.fade, style.enable, false, false)
	cdThumb, _ := pl.GetCd(style.cd, style.fade, style.enable, inside, active)

	rad := 0.2
	radPx := ui.CellWidth(rad)
	coord := lv.call.canvas
	coord = coord.AddSpaceX(radPx)

	rpos := ui.buff.win.io.touch.pos.Sub(coord.Start)
	touch_x := OsClampFloat(float64(rpos.X)/float64(coord.Size.X), 0, 1)

	if style.enable {
		if active || inside {
			ui.Paint_cursor("hand")
		}

		if active {
			*value = minValue + (maxValue-minValue)*touch_x
		}
		if !active && inside && ui.buff.win.io.touch.wheel != 0 {
			s := maxValue - minValue
			*value += s / 10 * float64(ui.buff.win.io.touch.wheel)
			ui.buff.win.io.touch.wheel = 0 //bug: If slider has canvas which can scroll under, it will scroll and slider is ignored ...
		}

		if len(style.tooltip) > 0 {
			ui.Paint_tooltip(0, 0, 1, 1, style.tooltip)
		}
	}
	//check & round
	{
		t := math.Round((*value - minValue) / jumpValue)
		*value = minValue + t*jumpValue
		*value = OsClampFloat(*value, minValue, maxValue)
	}

	t := (*value - minValue) / (maxValue - minValue)

	//draw
	if imgPath != "" {
		//imgMargin ...
		//'rating' ......

	} else {
		cqA := coord
		cqB := coord
		cqA.Size.X = int(float64(cqA.Size.X) * t)
		cqB.Start.X += cqA.Size.X
		cqB.Size.X -= cqA.Size.X
		cqA = cqA.AddSpaceY((cqA.Size.Y - radPx/2) / 2)
		cqB = cqB.AddSpaceY((cqB.Size.Y - radPx/2) / 2)

		//track(2x lines)
		ui.buff.AddRect(cqA, cd, 0)
		ui.buff.AddRect(cqB, cd.SetAlpha(100), 0)

		//thumb(sphere)
		cqT := InitOsV4Mid(OsV2{cqB.Start.X, coord.Start.Y + coord.Size.Y/2}, OsV2{radPx * 2, radPx * 2})
		ui.buff.AddCircle(cqT, cdThumb, 0)

		//label
		if style.enable && (active || inside) {
			ui.tile.SetForce(cqT.AddSpaceY(-ui.CellWidth(0.2)), true, strconv.FormatFloat(*value, 'f', 2, 64), pl.OnB)
		}
	}

	return active, (active && old_value != *value), end
}

func (ui *Ui) Comp_combo_desc(description string, description_alignH int, width float64, x, y, w, h int, value interface{}, optionsIn string, tooltip string, enable bool, search bool) bool {
	ui.Div_start(x, y, w, h)

	xx := 0

	if width > 0 {
		//1 row
		ui.Div_col(0, width)
		ui.Div_colMax(1, 100)
		ui.Comp_text(0, 0, 1, 1, description, description_alignH)
		xx = 1
	} else {
		//2 rows
		ui.Div_colMax(0, 100)
		ui.Comp_text(0, 0, 1, 1, description, description_alignH)
	}

	ret := ui.Comp_combo(xx, 0, 1, 1, value, optionsIn, tooltip, enable, search)

	ui.Div_end()

	return ret
}

func (ui *Ui) Comp_combo(x, y, w, h int, valueIn interface{}, optionsIn string, tooltip string, enable bool, search bool) bool {

	//search ...

	ui.Div_start(x, y, w, h)

	var value int
	switch v := valueIn.(type) {
	case *float32:
		value = int(*v)
	case *float64:
		value = int(*v)
	case *int:
		value = *v
	case *string:
		value, _ = strconv.Atoi(*v)
		//int8/16/32, uint8, byte, etc ...
	}

	var style UiComp
	style.cd = CdPalette_B
	style.label_alignV = 1
	style.enable = enable
	style.tooltip = tooltip

	ret := ui.Comp_combo_s(&style, value, optionsIn)
	changed := ret != value
	if changed {
		switch v := valueIn.(type) {
		case *float32:
			*v = float32(ret)
		case *float64:
			*v = float64(ret)
		case *int:
			*v = ret
		case *string:
			*v = strconv.Itoa(ret)
			//int8/16/32, uint8, byte, etc ...
		}
	}

	ui.Div_end()
	return changed
}

func (ui *Ui) Comp_combo_s(style *UiComp, value int, optionsIn string) int {

	lv := ui.GetCall()

	var options []string
	if len(optionsIn) > 0 {
		options = strings.Split(optionsIn, ";")
	}
	var valueStr string
	if value >= 0 && value < len(options) {
		valueStr = options[value]
	}

	nmd := "combo_" + strconv.Itoa(int(lv.call.data.hash))

	click, rclick, inside, active, _ := ui._compIsClicked(style.enable)
	if style.enable {

		if active || inside {
			ui.Paint_cursor("hand")
		}

		if click > 0 || rclick > 0 {
			ui.Dialog_open(nmd, 1)
		}

		if len(style.tooltip) > 0 {
			ui.Paint_tooltip(0, 0, 1, 1, style.tooltip)
		}
	}

	pl := ui.buff.win.io.GetPalette()
	backCd, _ := pl.GetCd(style.cd, true, style.enable, inside, active) //root.GetCdGrey(0.1)
	_, onCd := pl.GetCd(style.cd, style.fade, style.enable, inside, active)

	//back and arrow
	coord := lv.call.canvas
	m := ui.CellWidth(0.03)
	coord = coord.Inner(m, m, m, m)

	ui.buff.AddRect(coord, backCd, 0)                //background
	ui.buff.AddRect(coord, onCd, ui.CellWidth(0.03)) //border

	//text
	ui._compDrawText(coord, valueStr, "", onCd, SKYALT_FONT_HEIGHT, false, false, int(style.label_alignH), int(style.label_alignV), style.label_formating)
	style.label_alignH = 2
	ui._compDrawText(coord.AddSpace(ui.CellWidth(0.1)), "▼", "", onCd, SKYALT_FONT_HEIGHT, false, false, int(style.label_alignH), int(style.label_alignV), style.label_formating)

	//dialog
	if ui.Dialog_start(nmd) {
		//compute minimum dialog width
		mx := 0
		for _, opt := range options {
			mx = OsMax(mx, len(opt))
		}

		ui.Div_colMax(0, OsMaxFloat(5, SKYALT_FONT_HEIGHT*float64(mx)))

		for i, opt := range options {
			//highlight
			/*if value == int64(i) {
				menuSt.cd = CdPalette_P
			} else {
				menuSt.cd = CdPalette_B
			}*/
			if ui.Comp_buttonMenu(0, i, 1, 1, opt, "", true, value == i) > 0 {
				value = i
				ui.Dialog_close()
				break
			}
		}

		ui.Dialog_end()
	}

	return value
}

func (ui *Ui) Comp_checkbox(x, y, w, h int, valueIn interface{}, reverseValue bool, label string, tooltip string, enable bool) bool {

	ui.Div_start(x, y, w, h)

	var value float64
	switch v := valueIn.(type) {
	case *bool:
		value = OsTrnFloat(*v, 1, 0)
	case *float32:
		value = float64(*v)
	case *float64:
		value = float64(*v)
	case *int:
		value = float64(*v)
	case *string:
		vv, _ := strconv.Atoi(*v)
		value = float64(vv)
		//int8/16/32, uint8, byte, etc ...
	}

	var style UiComp
	style.cd = CdPalette_B
	style.label_alignV = 1
	style.enable = enable
	style.tooltip = tooltip

	ret := ui.Comp_checkbox_s(&style, value, label)
	changed := (ret != value)
	if changed {
		switch v := valueIn.(type) {
		case *bool:
			*v = ret != 0
		case *float32:
			*v = float32(ret)
		case *float64:
			*v = float64(ret)
		case *int:
			*v = int(ret)
		case *string:
			*v = strconv.FormatFloat(ret, 'f', -1, 64)
			//int8/16/32, uint8, byte, etc ...
		}
	}

	ui.Div_end()
	return changed
}

func (ui *Ui) Comp_checkbox_s(style *UiComp, value float64, label string) float64 {

	lv := ui.GetCall()

	click, rclick, inside, active, _ := ui._compIsClicked(style.enable)
	if style.enable {

		if active || inside {
			ui.Paint_cursor("hand")
		}

		if click > 0 || rclick > 0 {
			if value != 0 {
				value = 0
			} else {
				value = 1
			}
		}

		if len(style.tooltip) > 0 {
			ui.Paint_tooltip(0, 0, 1, 1, style.tooltip)
		}
	}

	pl := ui.buff.win.io.GetPalette()
	//backCd := pl.GetCd2(pl.GetGrey(0.1), false, style.enable, inside, active)
	_, onCd := pl.GetCd(style.cd, style.fade, style.enable, inside, active)

	mn := ui.CellWidth(0.5)
	coord := lv.call.canvas
	if len(label) > 0 {
		coordImg, coordText := ui._compGetTextImageCoord(coord, 1, style.image_alignH, true, true)

		//border
		coordImg = OsV4_centerFull(coordImg, OsV2{mn, mn})
		//ui.buff.AddRect(coordImg, backCd, 0)                //background
		ui.buff.AddRect(coordImg, onCd, ui.CellWidth(0.03)) //border

		//text
		//ui.Paint_textGrid(InitOsQuad(1, 0, 1, 1), pl.GetOnSurface(), style, label, "", "", true, false)
		ui._compDrawText(coordText, label, "", onCd, SKYALT_FONT_HEIGHT, false, false, int(style.label_alignH), int(style.label_alignV), style.label_formating)

		coord = coordImg

	} else {
		//center
		coord = OsV4_centerFull(coord, OsV2{mn, mn})
		ui.buff.AddRect(coord, onCd, ui.CellWidth(0.03))
	}

	coord = coord.AddSpace(ui.CellWidth(0.1))
	if value >= 1 {
		//draw check
		ui.buff.AddLine(coord.GetPos(1.0/3, 0.9), coord.GetPos(0.05, 2.0/3), onCd, ui.CellWidth(0.05))
		ui.buff.AddLine(coord.GetPos(1.0/3, 0.9), coord.GetPos(0.95, 1.0/4), onCd, ui.CellWidth(0.05))

	} else if value > 0 {
		//draw [-]
		ui.buff.AddLine(coord.GetPos(0, 0.5), coord.GetPos(1, 0.5), onCd, ui.CellWidth(0.05))
	}

	return value
}

func (ui *Ui) Comp_switch(x, y, w, h int, valueIn interface{}, reverseValue bool, label string, tooltip string, enable bool) bool {

	ui.Div_start(x, y, w, h)

	var value bool
	switch v := valueIn.(type) {
	case *bool:
		value = *v
	case *float32:
		value = *v != 0
	case *float64:
		value = *v != 0
	case *int:
		value = *v != 0
	case *string:
		vv, _ := strconv.Atoi(*v)
		value = vv != 0
		//int8/16/32, uint8, byte, etc ...
	}

	var style UiComp
	style.cd = CdPalette_P
	style.label_alignV = 1
	style.enable = enable
	style.tooltip = tooltip

	orig := OsTrnBool(reverseValue, !value, value)

	ret := ui.Comp_switch_s(&style, orig, label)
	changed := (ret != orig)
	ret = OsTrnBool(reverseValue, !ret, ret)
	if changed {
		switch v := valueIn.(type) {
		case *bool:
			*v = ret
		case *float32:
			*v = float32(OsTrn(ret, 1, 0))
		case *float64:
			*v = float64(OsTrn(ret, 1, 0))
		case *int:
			*v = OsTrn(ret, 1, 0)
		case *string:
			*v = strconv.Itoa(OsTrn(ret, 1, 0))
			//int8/16/32, uint8, byte, etc ...
		}

	}

	ui.Div_end()
	return changed
}

func (ui *Ui) Comp_switch_s(style *UiComp, value bool, label string) bool {

	lv := ui.GetCall()

	click, rclick, inside, active, _ := ui._compIsClicked(style.enable)
	if style.enable {
		if active || inside {
			ui.Paint_cursor("hand")
		}

		if click > 0 || rclick > 0 {
			value = !value
		}

		if len(style.tooltip) > 0 {
			ui.Paint_tooltip(0, 0, 1, 1, style.tooltip)
		}
	}

	pl := ui.buff.win.io.GetPalette()
	//backCd := pl.GetCd(style.cd, false)
	//surfCd := pl.GetSurface()
	//onSurfCd := pl.GetOnSurface()
	//onCd := pl.GetCd(style.cd, true)

	if !value {
		style.cd = CdPalette_B
	}

	backCd, _ := pl.GetCd(style.cd, style.fade, style.enable, inside, active)
	if !value {
		//backCd = OsCd_Aprox(backCd, pl.GetSurface(), 0.3)
		backCd = pl.GetCd2(pl.GetGrey(0.3), false, style.enable, inside, active)
	}

	midCd, _ := pl.GetCd(CdPalette_B, style.fade, style.enable, inside, active)

	style.cd = CdPalette_B
	_, labelCd := pl.GetCd(style.cd, style.fade, style.enable, inside, active)

	mn := ui.CellWidth(0.6)
	coord := lv.call.canvas
	if len(label) > 0 {
		coordImg, coordText := ui._compGetTextImageCoord(coord, 1.5, style.image_alignH, true, true)

		//border
		coordImg = OsV4_centerFull(coordImg, OsV2{int(float32(mn) * 1.7), mn})

		//text
		//ui.Paint_textGrid(coordText, pl.GetOnSurface(), style, label, "", "", true, false)
		ui._compDrawText(coordText, label, "", labelCd, SKYALT_FONT_HEIGHT, false, false, int(style.label_alignH), int(style.label_alignV), style.label_formating)

		coord = coordImg

	} else {
		//center
		coord = OsV4_centerFull(coord, OsV2{mn * 3 / 2, mn})
	}

	//back
	ui.buff.AddRect(coord, backCd, 0)

	coord = coord.AddSpace(ui.CellWidth(0.1))
	coord.Size.X /= 2
	if !value {
		ui.buff.AddRect(coord, midCd, 0)

		//0
		coord = coord.AddSpace(ui.CellWidth(0.1))
		ui.buff.AddLine(coord.GetPos(0, 0), coord.GetPos(1, 1), backCd, ui.CellWidth(0.05))
		ui.buff.AddLine(coord.GetPos(0, 1), coord.GetPos(1, 0), backCd, ui.CellWidth(0.05))

	} else {
		coord.Start.X += coord.Size.X
		ui.buff.AddRect(coord, midCd, 0)

		//I
		coord = coord.AddSpace(ui.CellWidth(0.1))
		ui.buff.AddLine(coord.GetPos(1.0/3, 0.9), coord.GetPos(0.05, 2.0/3), backCd, ui.CellWidth(0.05))
		ui.buff.AddLine(coord.GetPos(1.0/3, 0.9), coord.GetPos(0.95, 1.0/4), backCd, ui.CellWidth(0.05))
	}

	return value
}

func (ui *Ui) Comp_image(x, y, w, h int, path string, cd OsCd, margin float64, alignH, alignV int, fill bool) {

	ui.Div_start(x, y, w, h)

	var style UiComp
	style.enable = true
	style.image_alignH = uint8(alignH)
	style.image_alignV = uint8(alignV)
	style.image_margin = margin
	style.image_fill = fill

	lv := ui.GetCall()
	ui._compDrawImage(lv.call.canvas, path, cd, &style)

	ui.Div_end()
}
