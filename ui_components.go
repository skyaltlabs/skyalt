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
	"slices"
	"strconv"
)

type UiComp struct {
	enable          bool
	shape           uint8 //0=rect, 1=sphere
	cd              uint8 //back
	cd_border       uint8
	fade            bool
	label_align     OsV2
	label_formating bool
	image_alignV    uint8
	image_alignH    uint8
	image_fill      bool
	image_margin    float64
	tooltip         string
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
	b := ui.CellWidth(border)
	m := ui.CellWidth(margin)
	coord = coord.Inner(m, m, m, m)

	if cd.A > 0 {
		switch shape {
		case 0:
			ui.buff.AddRect(coord, cd, b)
		case 1:
			mn := OsMin(coord.Size.X, coord.Size.Y)
			coord = InitOsV4Mid(coord.Middle(), OsV2{mn, mn})
			ui.buff.AddCircle(coord, cd, b)
		}
	}
	return coord
}

func (ui *Ui) _compDrawImage(coord OsV4, icon WinMedia, cd OsCd, margin float64, align OsV2, fill bool) {
	lv := ui.GetCall()

	m := ui.CellWidth(margin)
	coord = coord.Inner(m, m, m, m)

	imgRectBackup := ui.buff.AddCrop(lv.call.crop.GetIntersect(coord))
	ui.buff.AddImage(icon, coord, cd, align, fill, false)
	ui.buff.AddCrop(imgRectBackup)
}

func (ui *Ui) _compDrawText(coord OsV4,
	value string, valueOrigEdit string,
	frontCd OsCd, prop WinFontProps,
	selection bool, editable bool, align OsV2,
	multi_line, multi_line_enter_finish, line_wrapping bool) {

	lv := ui.GetCall()

	coord = coord.AddSpace(ui.CellWidth(0.1))

	crop := lv.call.crop
	if editable {
		crop = crop.AddSpace(ui.CellWidth(0.1)) //crop rect_border
	}

	// crop
	imgRectBackup := ui.buff.AddCrop(crop.GetIntersect(coord))

	active := ui._UiPaint_Text_line(coord,
		value, valueOrigEdit,
		frontCd,
		prop,
		align,
		selection, editable, true,
		multi_line, multi_line_enter_finish, line_wrapping)

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

type Comp_buttonP struct {
	shape uint8 //0=rect, 1=sphere

	tooltip   string
	enable    bool
	formating bool

	cd        uint8
	cd_border uint8
	cd_fade   bool

	align      OsV2
	img_align  OsV2
	img_fill   bool
	img_margin float64

	icon *WinMedia

	draw_back   float32
	draw_border bool

	url              string
	confirmation     string
	confirmation_dnm string
}

func Comp_buttonProp() *Comp_buttonP {
	var p Comp_buttonP

	p.enable = true
	p.formating = true
	p.cd = CdPalette_P
	p.cd_border = CdPalette_P
	p.align = OsV2{1, 1}
	p.img_align = OsV2{0, 1}

	p.draw_back = 4

	return &p
}

func (p *Comp_buttonP) Shape(v uint8) *Comp_buttonP {
	p.shape = v
	return p
}
func (p *Comp_buttonP) Tooltip(v string) *Comp_buttonP {
	p.tooltip = v
	return p
}
func (p *Comp_buttonP) Enable(v bool) *Comp_buttonP {
	p.enable = v
	return p
}
func (p *Comp_buttonP) Formating(v bool) *Comp_buttonP {
	p.formating = v
	return p
}

func (p *Comp_buttonP) Cd(v uint8) *Comp_buttonP {
	p.cd = v
	return p
}
func (p *Comp_buttonP) CdBorder(v uint8) *Comp_buttonP {
	p.cd_border = v
	return p
}
func (p *Comp_buttonP) CdFade(v bool) *Comp_buttonP {
	p.cd_fade = v
	return p
}

func (p *Comp_buttonP) Align(h, v int) *Comp_buttonP {
	p.align = OsV2{h, v}
	return p
}
func (p *Comp_buttonP) ImgAlign(h, v int) *Comp_buttonP {
	p.img_align = OsV2{h, v}
	return p
}
func (p *Comp_buttonP) ImgMargin(v float64) *Comp_buttonP {
	p.img_margin = v
	return p
}
func (p *Comp_buttonP) ImgFill(v bool) *Comp_buttonP {
	p.img_fill = v
	return p
}

func (p *Comp_buttonP) Icon(v *WinMedia) *Comp_buttonP {
	p.icon = v
	return p
}

func (p *Comp_buttonP) DrawBack(v bool) *Comp_buttonP {
	p.draw_back = float32(OsTrnFloat(v, 1, 0))
	return p
}
func (p *Comp_buttonP) DrawBackLight(v bool) *Comp_buttonP {
	p.draw_back = float32(OsTrnFloat(v, 0.5, 0))
	return p
}

func (p *Comp_buttonP) DrawBorder(v bool) *Comp_buttonP {
	p.draw_border = v
	return p
}

func (p *Comp_buttonP) Confirmation(questionStr string, dialog_name string) *Comp_buttonP {
	p.confirmation = questionStr
	p.confirmation_dnm = dialog_name
	return p
}
func (p *Comp_buttonP) Url(v string) *Comp_buttonP {
	p.url = v
	return p
}

func (p *Comp_buttonP) SetError(v bool) *Comp_buttonP {
	if v {
		p.cd = CdPalette_E
		p.cd_border = CdPalette_E
	}
	return p
}

func (ui *Ui) Comp_button(x, y, w, h int, label string, prop *Comp_buttonP) int {
	ui.Div_start(x, y, w, h)

	click, rclick := ui.Comp_button_s(label, prop)

	ui.Div_end()
	if rclick > 0 {
		return 2
	} else if click > 0 {
		return 1
	}
	return 0
}

func (ui *Ui) Comp_buttonLight(x, y, w, h int, label string, prop *Comp_buttonP) int {
	ui.Div_start(x, y, w, h)

	prop.draw_back = 0.5
	click, rclick := ui.Comp_button_s(label, prop)

	ui.Div_end()
	if rclick > 0 {
		return 2
	} else if click > 0 {
		return 1
	}
	return 0
}

func (ui *Ui) Comp_buttonText(x, y, w, h int, label string, prop *Comp_buttonP) int {
	ui.Div_start(x, y, w, h)

	prop.draw_back = 0
	click, rclick := ui.Comp_button_s(label, prop)

	ui.Div_end()
	if rclick > 0 {
		return 2
	} else if click > 0 {
		return 1
	}
	return 0
}

func (ui *Ui) Comp_buttonOutlined(x, y, w, h int, label string, prop *Comp_buttonP) int {
	ui.Div_start(x, y, w, h)

	prop.draw_back = 0
	prop.draw_border = true
	click, rclick := ui.Comp_button_s(label, prop)

	ui.Div_end()
	if rclick > 0 {
		return 2
	} else if click > 0 {
		return 1
	}
	return 0
}

func (ui *Ui) Comp_buttonIcon(x, y, w, h int, icon WinMedia, icon_margin float64, tooltip string, prop *Comp_buttonP) int {
	ui.Div_start(x, y, w, h)

	prop.icon = &icon
	prop.img_margin = icon_margin
	prop.tooltip = tooltip
	prop.draw_back = 0
	click, rclick := ui.Comp_button_s("", prop)

	ui.Div_end()
	if rclick > 0 {
		return 2
	} else if click > 0 {
		return 1
	}
	return 0
}

func (ui *Ui) Comp_buttonMenu(x, y, w, h int, label string, selected bool, prop *Comp_buttonP) int {
	ui.Div_start(x, y, w, h)

	prop.draw_back = float32(OsTrn(selected, 1, 0))
	prop.align.X = 0
	if !selected && prop.cd != CdPalette_E {
		prop.cd = CdPalette_B
		prop.cd_border = CdPalette_B
	}

	click, rclick := ui.Comp_button_s(label, prop)

	ui.Div_end()
	if rclick > 0 {
		return 2
	} else if click > 0 {
		return 1
	}
	return 0
}

func (ui *Ui) Comp_buttonMenuIcon(x, y, w, h int, label string, icon WinMedia, icon_margin float64, selected bool, prop *Comp_buttonP) int {
	ui.Div_start(x, y, w, h)

	prop.icon = &icon
	prop.img_margin = icon_margin
	prop.draw_back = float32(OsTrn(selected, 1, 0))
	prop.align.X = 0
	if !selected && prop.cd != CdPalette_E {
		prop.cd = CdPalette_B
		prop.cd_border = CdPalette_B
	}

	click, rclick := ui.Comp_button_s(label, prop)

	ui.Div_end()
	if rclick > 0 {
		return 2
	} else if click > 0 {
		return 1
	}
	return 0
}

func (ui *Ui) Comp_button_s(label string, prop *Comp_buttonP) (int, int) {

	lv := ui.GetCall()

	click, rclick, inside, active, _ := lv.call.IsClicked(prop.enable, ui)
	if click > 0 && len(prop.url) > 0 {
		//SA_DialogStart() warning which open dialog ...
		OsUlit_OpenBrowser(prop.url)
	}

	pl := ui.win.io.GetPalette()
	cd, onCd := pl.GetCd(prop.cd, prop.cd_fade, prop.enable, inside, active)
	cdBorder, _ := pl.GetCd(prop.cd_border, prop.cd_fade, prop.enable, inside, active)

	coord := lv.call.canvas

	drawBack := prop.draw_back
	if drawBack == 0 {
		if prop.cd != CdPalette_B {
			onCd = cd
		}

		if prop.enable && len(label) > 0 { //no background for icons. For text yes
			if inside || active {
				//not same color as background
				if prop.cd == CdPalette_B {
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

		margin := OsTrnFloat(prop.draw_border, 0.06, 0)
		ui._compDrawShape(coord, prop.shape, cd, margin, 0)
	}

	if prop.draw_border {
		ui._compDrawShape(coord, prop.shape, cdBorder, 0, 0.03)
	}

	if prop.icon != nil && label != "" {
		prop.img_align.X = 0
	}

	coordImage, coordText := ui._compGetTextImageCoord(coord, 1, uint8(prop.img_align.X), prop.icon != nil, label != "")
	if prop.icon != nil {
		ui._compDrawImage(coordImage, *prop.icon, onCd, prop.img_margin, prop.align, prop.img_fill)
	}
	if len(label) > 0 {
		pr := InitWinFontPropsDef(ui.win)
		pr.formating = prop.formating
		ui._compDrawText(coordText, label, "", onCd, pr, false, false, prop.align, true, false, false)
	}

	if prop.enable {
		if inside {
			ui.Paint_cursor("hand")
		}

		if len(prop.tooltip) > 0 {
			ui.Paint_tooltip(0, 0, 1, 1, prop.tooltip)
		} else if len(prop.url) > 0 {
			ui.Paint_tooltip(0, 0, 1, 1, prop.url)
		}
	}

	if (click > 0 || rclick > 0) && prop.confirmation != "" {
		ui.Dialog_open(prop.confirmation_dnm, 1)
		click = 0
		rclick = 0
	}
	if ui.Dialog_start(prop.confirmation_dnm) {

		sz := OsMaxFloat(4, float64(1+ui.win.GetTextSize(-1, prop.confirmation, InitWinFontPropsDef(ui.win)).X/ui.win.Cell()))

		ui.Div_colMax(0, sz/2)
		ui.Div_colMax(1, sz/2)
		ui.Comp_text(0, 0, 2, 1, prop.confirmation, 1)
		if ui.Comp_button(0, 1, 1, 1, "Yes", Comp_buttonProp().SetError(true)) > 0 {
			click = 1
			ui.Dialog_close()
		}
		if ui.Comp_button(1, 1, 1, 1, "No", Comp_buttonProp()) > 0 {
			ui.Dialog_close()
		}
		ui.Dialog_end()
	}

	return click, rclick
}

func (ui *Ui) Comp_textIcon(x, y, w, h int, label string, icon WinMedia, iconMargin float64) {
	ui.Div_start(x, y, w, h)

	var style UiComp
	style.enable = true
	style.cd = CdPalette_B
	style.label_align = OsV2{0, 1}
	style.label_formating = true
	style.image_alignV = 1
	style.image_margin = iconMargin

	ui.Comp_text_s(&style, label, &icon, false, false, false, false)

	ui.Div_end()
}

func (ui *Ui) Comp_text(x, y, w, h int, label string, alignH int) *UiLayoutDiv {
	ui.Div_start(x, y, w, h)
	div := ui.GetCall().call

	var style UiComp
	style.enable = true
	style.cd = CdPalette_B
	style.label_align = OsV2{alignH, 1}
	style.label_formating = true

	ui.Comp_text_s(&style, label, nil, true, false, false, false)

	ui.Div_end()
	return div
}

func (ui *Ui) Comp_textCd(x, y, w, h int, label string, alignH int, cd uint8) *UiLayoutDiv {
	ui.Div_start(x, y, w, h)
	div := ui.GetCall().call

	var style UiComp
	style.enable = true
	style.cd = cd
	style.label_align = OsV2{alignH, 1}
	style.label_formating = true

	//background
	//backCd, _ := ui.win.io.GetPalette().GetCd(cd, false, true, false, false)
	//ui.Paint_rect(0, 0, 1, 1, 0, backCd, 0)

	//text
	ui.Comp_text_s(&style, label, nil, true, false, false, false)

	ui.Div_end()
	return div
}

func (ui *Ui) Comp_textSelect(x, y, w, h int, label string, align OsV2, selection bool, drawBorder bool) *UiLayoutDiv {
	ui.Div_start(x, y, w, h)
	div := ui.GetCall().call

	var style UiComp
	style.enable = true
	style.cd = CdPalette_B
	style.label_align = align
	style.label_formating = true

	ui.Comp_text_s(&style, label, nil, selection, false, false, false)

	if drawBorder {
		pl := ui.win.io.GetPalette()
		ui.Paint_rect(0, 0, 1, 1, 0, pl.OnB, 0.03)
	}

	ui.Div_end()
	return div
}
func (ui *Ui) Comp_textSelectMulti(x, y, w, h int, label string, align OsV2, selection bool, drawBorder bool, formating bool, line_wrapping bool) *UiLayoutDiv {
	ui.Div_start(x, y, w, h)
	div := ui.GetCall().call

	var style UiComp
	style.enable = true
	style.cd = CdPalette_B
	style.label_align = align
	style.label_formating = true
	style.label_formating = formating

	ui.Comp_text_s(&style, label, nil, selection, true, false, line_wrapping)

	if drawBorder {
		pl := ui.win.io.GetPalette()
		ui.Paint_rect(0, 0, 1, 1, 0, pl.OnB, 0.03)
	}

	ui.Div_end()
	return div
}

func (ui *Ui) Comp_text_s(style *UiComp, value string, icon *WinMedia, selection bool, multi_line bool, multi_line_enter_finish, line_wrapping bool) {

	pl := ui.win.io.GetPalette()
	cd, onCd := pl.GetCd(style.cd, style.fade, style.enable, false, false)

	if style.cd == CdPalette_E {
		onCd = cd //swap
	}

	if style.enable {
		if len(style.tooltip) > 0 {
			ui.Paint_tooltip(0, 0, 1, 1, style.tooltip)
		}
	}

	prop := InitWinFontPropsDef(ui.win)
	prop.formating = style.label_formating

	ui.Paint_textGrid(onCd, style, value, "", prop, icon, selection, false, multi_line, multi_line_enter_finish, line_wrapping)
}

type Comp_editboxP struct {
	value_precision         int
	align                   OsV2
	icon                    *WinMedia
	ghost                   string
	highlight               bool
	tempToValue             bool
	multi_line              bool
	multi_line_enter_finish bool
	line_wrapping           bool
	enable                  bool
	formating               bool
}

func Comp_editboxProp() *Comp_editboxP {
	var p Comp_editboxP

	p.value_precision = -1
	p.align = OsV2{0, 1}
	p.enable = true
	p.formating = true
	return &p
}

func (p *Comp_editboxP) Precision(v int) *Comp_editboxP {
	p.value_precision = v
	return p
}
func (p *Comp_editboxP) Align(h, v int) *Comp_editboxP {
	p.align = OsV2{h, v}
	return p
}
func (p *Comp_editboxP) Ghost(v string) *Comp_editboxP {
	p.ghost = v
	return p
}
func (p *Comp_editboxP) Icon(v *WinMedia) *Comp_editboxP {
	p.icon = v
	return p
}
func (p *Comp_editboxP) Highlight(v bool) *Comp_editboxP {
	p.highlight = v
	return p
}
func (p *Comp_editboxP) TempToValue(v bool) *Comp_editboxP {
	p.tempToValue = v
	return p
}
func (p *Comp_editboxP) MultiLine(enable, line_wrapping bool) *Comp_editboxP {
	p.multi_line = enable
	p.line_wrapping = line_wrapping
	return p
}
func (p *Comp_editboxP) MultiLineEnterFinish(v bool) *Comp_editboxP {
	p.multi_line_enter_finish = v
	return p
}
func (p *Comp_editboxP) Enable(v bool) *Comp_editboxP {
	p.enable = v
	return p
}
func (p *Comp_editboxP) Formating(v bool) *Comp_editboxP {
	p.formating = v
	return p
}

func (ui *Ui) Comp_editbox_desc(description string, description_alignH int, width float64, x, y, w, h int, valueIn interface{}, prop *Comp_editboxP) (string, bool, bool, bool, *UiLayoutDiv) {
	ui.Div_start(x, y, w, h)

	xx := 0
	if width > 0 {
		//1 row
		ui.Div_colMax(0, width)
		ui.Div_colMax(1, 100)
		ui.Comp_text(0, 0, 1, 1, description, description_alignH)
		xx = 1
	} else {
		//2 rows
		ui.Div_colMax(0, 100)
		ui.Comp_text(0, 0, 1, 1, description, description_alignH)
	}

	editedValue, active, changed, finished, div := ui.Comp_editbox(xx, 0, 1, 1, valueIn, prop)

	ui.Div_end()

	return editedValue, active, changed, finished, div
}

func (ui *Ui) Comp_editbox(x, y, w, h int, valueIn interface{}, prop *Comp_editboxP) (string, bool, bool, bool, *UiLayoutDiv) {

	ui.Div_start(x, y, w, h)
	div := ui.GetCall().call

	value := ""
	switch v := valueIn.(type) {
	case *float32:
		if v != nil {
			value = strconv.FormatFloat(float64(*v), 'f', prop.value_precision, 64)
		}
	case *float64:
		if v != nil {
			value = strconv.FormatFloat(*v, 'f', prop.value_precision, 64)
		}
	case *int:
		if v != nil {
			value = strconv.Itoa(*v)
		}
	case *[]byte:
		if v != nil {
			value = string(*v)
		}
	case *string:
		if v != nil {
			value = *v
		}
		//int8/16/32, uint8, byte, etc ...
	}

	var style UiComp
	style.enable = prop.enable
	style.cd = CdPalette_B
	style.label_align = prop.align
	style.label_formating = prop.formating

	editedValue, active, changed, finished := ui.Comp_edit_s(&style, value, value, prop.icon, prop.ghost, prop.highlight, prop.tempToValue, prop.multi_line, prop.multi_line_enter_finish, prop.line_wrapping)

	if finished || prop.tempToValue {
		switch v := valueIn.(type) {
		case *float32:
			if v != nil {
				vv, _ := strconv.ParseFloat(editedValue, 64)
				*v = float32(vv)
			}
		case *float64:
			if v != nil {
				*v, _ = strconv.ParseFloat(editedValue, 64)
			}
		case *int:
			if v != nil {
				*v, _ = strconv.Atoi(editedValue)
			}
		case *[]byte:
			if v != nil {
				*v = []byte(editedValue)
			}
		case *string:
			if v != nil {
				*v = editedValue
			}
			//int8/16/32, uint8, byte, etc ...
		}
	}

	ui.Div_end()

	return editedValue, active, changed, finished, div
}
func (ui *Ui) Comp_edit_s(style *UiComp, valueIn string, valueInOrig string, icon *WinMedia, ghost string, highlight bool, tempToValue bool, multi_line bool, multi_line_enter_finish, line_wrapping bool) (string, bool, bool, bool) {

	lv := ui.GetCall()

	pl := ui.win.io.GetPalette()
	cd, onCd := pl.GetCd(style.cd, style.fade, style.enable, false, false)

	if highlight {
		cd, onCd = pl.GetCd(CdPalette_T, false, style.enable, false, false)
	}

	edit := &ui.edit

	inDiv := lv.call.FindOrCreate("", InitOsV4(0, 0, 1, 1), ui.GetLastApp())
	this_uid := inDiv
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
	ui.buff.AddRect(coord, cd, 0)

	{
		w := ui.CellWidth(0.03)
		if active {
			w *= 2
		}
		ui.buff.AddRect(coord, pl.P, w)
	}

	prop := InitWinFontPropsDef(ui.win)
	prop.formating = style.label_formating
	ui.Paint_textGrid(onCd, style, value, valueInOrig, prop, icon, true, true, multi_line, multi_line_enter_finish, line_wrapping)

	//ghost
	if len(edit.last_edit) == 0 && len(ghost) > 0 {
		ui._compDrawText(coord, ghost, "", pl.GetGrey(0.7), prop, false, false, OsV2{1, 1}, false, false, false)
	}

	if style.enable {
		if len(style.tooltip) > 0 {
			ui.Paint_tooltip(0, 0, 1, 1, style.tooltip)
		}
	}

	return edit.last_edit, active, (active && value != edit.last_edit), (active && this_uid != edit.uid)
}

func (ui *Ui) Comp_progress(x, y, w, h int, value float64, prec int, tooltip string, enable bool) {

	ui.Div_start(x, y, w, h)

	style := UiComp{enable: enable, label_formating: true, cd: CdPalette_P, tooltip: tooltip, label_align: OsV2{2, 1}}
	ui.Comp_progress_s(&style, value, prec)

	ui.Div_end()
}

func (ui *Ui) Comp_progress_s(style *UiComp, value float64, prec int) {

	lv := ui.GetCall()

	value = OsClampFloat(value, 0, 1)

	pl := ui.win.io.GetPalette()
	cd, onCd := pl.GetCd(style.cd, style.fade, style.enable, false, false)

	coord := lv.call.canvas

	//border
	ui._compDrawShape(coord, style.shape, cd, 0.03, 0.03)
	//back
	coord = ui.getCoord(0, 0, value, 1, 0, 0, 0)
	ui._compDrawShape(coord, style.shape, cd, 0.09, 0)
	//label
	//ui.Paint_textGrid(coord, onCd, style, strconv.FormatFloat(value*100, 'f', prec, 64)+"%", "", "", true, false)
	prop := InitWinFontPropsDef(ui.win)
	prop.formating = style.label_formating
	ui._compDrawText(coord, strconv.FormatFloat(value*100, 'f', prec, 64)+"%", "", onCd, prop, true, false, style.label_align, false, false, false)

	if style.enable {
		if len(style.tooltip) > 0 {
			ui.Paint_tooltip(0, 0, 1, 1, style.tooltip)
		}
	}
}

func (ui *Ui) Comp_slider(x, y, w, h int, valueIn interface{}, minValue float64, maxValue float64, stepValue float64, enable bool) bool {

	var value float64
	switch v := valueIn.(type) {
	case *bool:
		if v != nil {
			value = OsTrnFloat(*v, 1, 0)
		}
	case *float32:
		if v != nil {
			value = float64(*v)
		}
	case *float64:
		if v != nil {
			value = float64(*v)
		}
	case *int:
		if v != nil {
			value = float64(*v)
		}
	case *string:
		if v != nil {
			vv, _ := strconv.ParseFloat(*v, 64)
			value = vv
		}
		//int8/16/32, uint8, byte, etc ...
	}

	ui.Div_start(x, y, w, h)

	styleSlider := UiComp{enable: enable, label_formating: true, cd: CdPalette_P}
	active, changed, end := ui.Comp_slider_s(&styleSlider, &value, minValue, maxValue, stepValue, "", 0)

	ui.Div_end()

	if changed {
		switch v := valueIn.(type) {
		case *bool:
			if v != nil {
				*v = value != 0
			}
		case *float32:
			if v != nil {
				*v = float32(value)
			}
		case *float64:
			if v != nil {
				*v = value
			}
		case *int:
			if v != nil {
				*v = int(value)
			}
		case *string:
			if v != nil {
				*v = strconv.FormatFloat(value, 'f', -1, 64)
			}
			//int8/16/32, uint8, byte, etc ...
		}
	}

	return (active || end) && changed //change can be true alone, because of stepValue "align", no click neede
}

func (ui *Ui) Comp_slider_desc(description string, description_alignH int, width float64, x, y, w, h int, valueIn interface{}, minValue float64, maxValue float64, stepValue float64, enable bool) bool {
	ui.Div_start(x, y, w, h)

	xx := 0
	if width > 0 {
		//1 row
		ui.Div_colMax(0, width)
		ui.Div_colMax(1, 100)
		ui.Comp_text(0, 0, 1, 1, description, description_alignH)
		xx = 1
	} else {
		//2 rows
		ui.Div_colMax(0, 100)
		ui.Comp_text(0, 0, 1, 1, description, description_alignH)
	}

	changed := ui.Comp_slider(xx, 0, 1, 1, valueIn, minValue, maxValue, stepValue, enable)

	ui.Div_end()

	return changed
}

func (ui *Ui) Comp_slider_s(style *UiComp, value *float64, minValue float64, maxValue float64, stepValue float64, imgPath string, imgMargin float64) (bool, bool, bool) {

	if stepValue == 0 {
		stepValue = 0.001 //avoid division by zero
	}
	if minValue == maxValue {
		maxValue = minValue + 0.001 //avoid division by zero
	}

	lv := ui.GetCall()

	old_value := *value

	_, _, inside, active, end := lv.call.IsClicked(style.enable, ui)

	pl := ui.win.io.GetPalette()
	cd, _ := pl.GetCd(style.cd, style.fade, style.enable, false, false)
	cdThumb, _ := pl.GetCd(style.cd, style.fade, style.enable, inside, active)

	rad := 0.2
	radPx := ui.CellWidth(rad)
	coord := lv.call.canvas
	coord = coord.AddSpaceX(radPx)

	rpos := ui.win.io.touch.pos.Sub(coord.Start)
	touch_x := OsClampFloat(float64(rpos.X)/float64(coord.Size.X), 0, 1)

	if style.enable {
		if active || inside {
			ui.Paint_cursor("hand")
		}

		if active {
			*value = minValue + (maxValue-minValue)*touch_x
		}
		if !active && inside && ui.win.io.touch.wheel != 0 {
			s := maxValue - minValue
			*value += s / 10 * float64(ui.win.io.touch.wheel)
			ui.win.io.touch.wheel = 0 //bug: If slider has canvas which can scroll under, it will scroll and slider is ignored ...
			end = true

			ui.touch.scrollWheel = lv.call.GetHash()
		}

		if len(style.tooltip) > 0 {
			ui.Paint_tooltip(0, 0, 1, 1, style.tooltip)
		}
	}

	//check & round
	{
		t := math.Round((*value - minValue) / stepValue)
		*value = minValue + t*stepValue
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

	return active, (old_value != *value), (ui.win.io.touch.wheel != 0 || end)
}

func (ui *Ui) Comp_combo_desc(description string, description_alignH int, width float64, x, y, w, h int, value *string, options_names []string, options_values []string, tooltip string, enable bool, search bool) bool {
	ui.Div_start(x, y, w, h)

	xx := 0
	if width > 0 {
		//1 row
		ui.Div_colMax(0, width)
		ui.Div_colMax(1, 100)
		ui.Comp_text(0, 0, 1, 1, description, description_alignH)
		xx = 1
	} else {
		//2 rows
		ui.Div_colMax(0, 100)
		ui.Comp_text(0, 0, 1, 1, description, description_alignH)
	}

	ret := ui.Comp_combo(xx, 0, 1, 1, value, options_names, options_values, tooltip, enable, search)

	ui.Div_end()

	return ret
}

func (ui *Ui) Comp_combo(x, y, w, h int, valueIn *string, options_names []string, options_values []string, tooltip string, enable bool, search bool) bool {

	//search ...

	ui.Div_start(x, y, w, h)

	var style UiComp
	style.cd = CdPalette_B
	style.label_align = OsV2{0, 1}
	style.enable = enable
	style.tooltip = tooltip

	ret := ui.Comp_combo_s(&style, *valueIn, options_names, options_values)
	changed := (ret != *valueIn)
	*valueIn = ret

	ui.Div_end()
	return changed
}

func (ui *Ui) Comp_combo_s(style *UiComp, value string, options_names []string, options_values []string) string {

	//must have same size
	if len(options_values) == 1 && options_values[0] == "" {
		options_values = nil
	}

	//must have same size
	if len(options_values) > 0 {
		n := OsMin(len(options_names), len(options_values))
		options_names = options_names[:n]
		options_values = options_values[:n]
	}

	lv := ui.GetCall()

	nmd := "combo_" + strconv.Itoa(int(lv.call.GetHash()))

	click, rclick, inside, active, _ := lv.call.IsClicked(style.enable, ui)
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

	pl := ui.win.io.GetPalette()
	backCd, _ := pl.GetCd(style.cd, true, style.enable, inside, active) //root.GetCdGrey(0.1)
	_, onCd := pl.GetCd(style.cd, style.fade, style.enable, inside, active)

	//back and arrow
	coord := lv.call.canvas
	m := ui.CellWidth(0.03)
	coord = coord.Inner(m, m, m, m)

	ui.buff.AddRect(coord, backCd, 0)                //background
	ui.buff.AddRect(coord, onCd, ui.CellWidth(0.03)) //border

	//text
	{
		label := value
		if len(options_values) > 0 {
			pos := slices.Index(options_values, value)
			if pos >= 0 {
				label = options_names[pos]
			}
		} else {
			pos, err := strconv.Atoi(value)
			if err == nil && pos >= 0 && pos < len(options_names) {
				label = options_names[pos]
			}
		}
		prop := InitWinFontPropsDef(ui.win)
		prop.formating = style.label_formating
		ui._compDrawText(coord, label, "", onCd, prop, false, false, style.label_align, false, false, false)
		style.label_align.X = 2
		prop.formating = true
		ui._compDrawText(coord.AddSpaceX(ui.CellWidth(0.1)), "###▼###", "", onCd, prop, false, false, style.label_align, false, false, false) //### aka smaller
	}

	//dialog
	if ui.Dialog_start(nmd) {
		//compute minimum dialog width
		mx := float64(0)
		for _, opt := range options_names {
			sz := ui.win.GetTextSize(-1, opt, InitWinFontPropsDef(ui.win))
			mx = OsMaxFloat(mx, float64(sz.X)/float64(ui.win.Cell())+0.5)
		}
		ui.Div_colMax(0, OsMaxFloat(3, mx))

		for i, opt := range options_names {
			var highlight bool
			if len(options_values) > 0 {
				highlight = (value == options_values[i])
			} else {
				highlight = (value == strconv.Itoa(i))
			}

			if ui.Comp_buttonMenu(0, i, 1, 1, opt, highlight, Comp_buttonProp()) > 0 {
				if len(options_values) > 0 {
					value = options_values[i]
				} else {
					value = strconv.Itoa(i)
				}

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
		if v != nil {
			value = OsTrnFloat(*v, 1, 0)
		}
	case *float32:
		if v != nil {
			value = float64(*v)
		}
	case *float64:
		if v != nil {
			value = float64(*v)
		}
	case *int:
		if v != nil {
			value = float64(*v)
		}
	case *string:
		if v != nil {
			vv, _ := strconv.ParseFloat(*v, 64)
			value = vv
		}
		//int8/16/32, uint8, byte, etc ...
	}

	var style UiComp
	style.cd = CdPalette_B
	style.label_align = OsV2{0, 1}
	style.enable = enable
	style.tooltip = tooltip

	ret := ui.Comp_checkbox_s(&style, value, label)
	changed := (ret != value)
	if changed {
		switch v := valueIn.(type) {
		case *bool:
			if v != nil {
				*v = ret != 0
			}
		case *float32:
			if v != nil {
				*v = float32(ret)
			}
		case *float64:
			if v != nil {
				*v = float64(ret)
			}
		case *int:
			if v != nil {
				*v = int(ret)
			}
		case *string:
			if v != nil {
				*v = strconv.FormatFloat(ret, 'f', -1, 64)
			}
			//int8/16/32, uint8, byte, etc ...
		}
	}

	ui.Div_end()
	return changed
}

func (ui *Ui) Comp_checkbox_s(style *UiComp, value float64, label string) float64 {

	lv := ui.GetCall()

	click, rclick, inside, active, _ := lv.call.IsClicked(style.enable, ui)
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

	pl := ui.win.io.GetPalette()
	_, onCd := pl.GetCd(style.cd, style.fade, style.enable, inside, active)

	mn := OsMin(int(float64(lv.call.canvas.Size.Y)*0.9), ui.CellWidth(0.5))
	coord := lv.call.canvas
	if len(label) > 0 {
		coordImg, coordText := ui._compGetTextImageCoord(coord, 1, style.image_alignH, true, true)

		//border
		coordImg = OsV4_centerFull(coordImg, OsV2{mn, mn})
		//ui.buff.AddRect(coordImg, backCd, 0)                //background
		ui.buff.AddRect(coordImg, onCd, ui.CellWidth(0.03)) //border

		//text
		prop := InitWinFontPropsDef(ui.win)
		prop.formating = style.label_formating
		ui._compDrawText(coordText, label, "", onCd, prop, false, false, style.label_align, true, false, true)

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
		if v != nil {
			value = *v
		}
	case *float32:
		if v != nil {
			value = *v != 0
		}
	case *float64:
		if v != nil {
			value = *v != 0
		}
	case *int:
		if v != nil {
			value = *v != 0
		}
	case *string:
		if v != nil {
			vv, _ := strconv.Atoi(*v)
			value = vv != 0
		}
		//int8/16/32, uint8, byte, etc ...
	}

	var style UiComp
	style.cd = CdPalette_P
	style.label_align = OsV2{0, 1}
	style.enable = enable
	style.tooltip = tooltip

	orig := OsTrnBool(reverseValue, !value, value)

	ret := ui.Comp_switch_s(&style, orig, label)
	changed := (ret != orig)
	ret = OsTrnBool(reverseValue, !ret, ret)
	if changed {
		switch v := valueIn.(type) {
		case *bool:
			if v != nil {
				*v = ret
			}
		case *float32:
			if v != nil {
				*v = float32(OsTrn(ret, 1, 0))
			}
		case *float64:
			if v != nil {
				*v = float64(OsTrn(ret, 1, 0))
			}
		case *int:
			if v != nil {
				*v = OsTrn(ret, 1, 0)
			}
		case *string:
			if v != nil {
				*v = strconv.Itoa(OsTrn(ret, 1, 0))
			}
			//int8/16/32, uint8, byte, etc ...
		}
	}

	ui.Div_end()
	return changed
}

func (ui *Ui) Comp_switch_s(style *UiComp, value bool, label string) bool {

	lv := ui.GetCall()

	click, rclick, inside, active, _ := lv.call.IsClicked(style.enable, ui)
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

	pl := ui.win.io.GetPalette()
	if !value {
		style.cd = CdPalette_B
	}

	backCd, _ := pl.GetCd(style.cd, style.fade, style.enable, inside, active)
	if !value {
		backCd = pl.GetCd2(pl.GetGrey(0.3), false, style.enable, inside, active)
	}

	midCd, _ := pl.GetCd(CdPalette_B, style.fade, style.enable, inside, active)

	style.cd = CdPalette_B
	_, labelOnCd := pl.GetCd(style.cd, style.fade, style.enable, inside, active)

	mn := OsMin(int(float64(lv.call.canvas.Size.Y)*0.9), ui.CellWidth(0.6))
	coord := lv.call.canvas
	if len(label) > 0 {
		coordImg, coordText := ui._compGetTextImageCoord(coord, 1.5, style.image_alignH, true, true)

		//border
		coordImg = OsV4_centerFull(coordImg, OsV2{int(float32(mn) * 1.7), mn})

		//text
		prop := InitWinFontPropsDef(ui.win)
		prop.formating = style.label_formating
		ui._compDrawText(coordText, label, "", labelOnCd, prop, false, false, style.label_align, true, false, true)

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

func (ui *Ui) Comp_image(x, y, w, h int, image WinMedia, cd OsCd, margin float64, align OsV2, fill bool) {

	ui.Div_start(x, y, w, h)

	lv := ui.GetCall()
	ui._compDrawImage(lv.call.canvas, image, cd, margin, align, fill)

	ui.Div_end()
}
