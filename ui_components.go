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

func (levels *Ui) compIsClicked(enable bool) (int, int, bool, bool, bool) {
	var click, rclick int
	var inside, active, end bool
	if enable {
		lv := levels.GetCall()
		inside = lv.call.data.touch_inside
		active = lv.call.data.touch_active
		end = lv.call.data.touch_end

		end := lv.call.data.touch_end
		force := levels.win.io.touch.rm

		if inside && end {
			click = 1
			rclick = OsTrn(force, 1, 0)
		}

		if click > 0 {
			click = int(levels.win.io.touch.numClicks)
		}
		if rclick > 0 {
			rclick = int(levels.win.io.touch.numClicks)
		}
	}

	return click, rclick, inside, active, end
}

func (levels *Ui) compGetTextImageCoord(coord OsV4, image_width float64, imageAlignH uint8, isImg bool, isText bool) (OsV4, OsV4) {

	lv := levels.GetCall()

	coordImg := coord
	coordText := coord

	//isImg := (len(image_url) > 0)
	//isText := (len(text) > 0 || textEdit)

	if isImg && isText {
		w := float64(levels.win.Cell()) / float64(lv.call.canvas.Size.X) * image_width

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

func (levels *Ui) compDrawShape(coord OsV4, shape uint8, cd OsCd, margin float64, border float64) OsV4 {
	//
	//st := root.levels.GetStack()
	//coord := st.stack.canvas

	b := levels.CellWidth(border)
	m := levels.CellWidth(margin)
	coord = coord.Inner(m, m, m, m)

	if cd.A > 0 {
		switch shape {
		case 0:
			levels.buff.AddRect(coord, cd, b)
		case 1:
			levels.buff.AddCircle(coord, cd, b)
		}
	}
	return coord
}

func (levels *Ui) compDrawImage(coord OsV4, icon string, cd OsCd, style *UiComp) {

	lv := levels.GetCall()

	path, err := InitWinMedia(icon)
	if err != nil {
		lv.call.data.app.AddLogErr(err)
	} else {
		m := levels.CellWidth(style.image_margin)
		coord = coord.Inner(m, m, m, m)

		//style.image_fill = true

		imgRectBackup := levels.buff.AddCrop(lv.call.crop.GetIntersect(coord))
		levels.buff.AddImage(path, coord, cd, int(style.image_alignV), int(style.image_alignH), style.image_fill)
		levels.buff.AddCrop(imgRectBackup)
	}
}

func (levels *Ui) compDrawText(coord OsV4, value string, valueOrigEdit string, cd OsCd, selection bool, editable bool, style *UiComp) {

	lv := levels.GetCall()

	coord = coord.AddSpaceX(levels.CellWidth(0.1))

	// crop
	imgRectBackup := levels.buff.AddCrop(lv.call.crop.GetIntersect(coord))

	//one liner
	active := levels._UiPaint_Text_line(coord, 0, OsV2{utf8.RuneCountInString(value), 0},
		value, valueOrigEdit,
		cd,
		SKYALT_FONT_HEIGHT, 1, 0, 0,
		SKYALT_FONT_PATH, int(style.label_alignH), int(style.label_alignV),
		selection, editable, true, style.label_formating)

	if active {
		levels._UiPaint_resetKeys(false)
	}

	if selection {
		lv := levels.GetCall()
		if lv.call.data.touch_inside {
			levels.Paint_cursor("ibeam")
		}
	}

	// crop back
	levels.buff.AddCrop(imgRectBackup)
}

func (levels *Ui) Comp_button(style *UiComp, value string, icon string, url string, drawBack float64, drawBorder bool) (int, int) {

	lv := levels.GetCall()

	click, rclick, inside, active, _ := levels.compIsClicked(style.enable)
	if click > 0 && len(url) > 0 {
		//SA_DialogStart() warning which open dialog ...
		OsUlit_OpenBrowser(url)
	}

	pl := levels.buff.win.io.GetPalette()
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

	m := levels.CellWidth(0.03)
	coord = coord.Inner(m, m, m, m)

	//background
	if drawBack > 0 {

		//light
		if drawBack <= 0.6 {
			//t := cd
			//cd = onCd
			//onCd = t
			onCd = cd

			cd.A = 30
		}

		levels.compDrawShape(coord, style.shape, cd, 0, 0)
	}

	if drawBorder {
		levels.compDrawShape(coord, style.shape, cd, 0, 0.03)
	}

	if len(icon) > 0 {
		style.image_alignH = 0
	}

	coordImage, coordText := levels.compGetTextImageCoord(coord, 1, style.image_alignH, len(icon) > 0, len(value) > 0)
	if len(icon) > 0 {
		levels.compDrawImage(coordImage, icon, onCd, style)
	}
	if len(value) > 0 {
		levels.compDrawText(coordText, value, "", onCd, false, false, style)
	}

	if style.enable {
		if inside {
			levels.Paint_cursor("hand")
		}

		if len(style.tooltip) > 0 {
			levels.Paint_tooltip(0, 0, 1, 1, style.tooltip)
		} else if len(url) > 0 {
			levels.Paint_tooltip(0, 0, 1, 1, url)
		}
	}

	return click, rclick
}

func (levels *Ui) Comp_text(style *UiComp, value string, icon string) int64 {

	pl := levels.buff.win.io.GetPalette()
	_, onCd := pl.GetCd(style.cd, style.fade, style.enable, false, false)

	//if style.enable_back {
	//levels.buff.AddRect(st.stack.canvas, cd, 0)
	//}
	//if style.enable_border {
	//levels.buff.AddRect(st.stack.canvas, pl.GetGrey(0.8), levels.CellWidth(0.03))
	//}

	if style.enable {
		if len(style.tooltip) > 0 {
			levels.Paint_tooltip(0, 0, 1, 1, style.tooltip)
		}
	}

	levels.Paint_textGrid(InitOsQuad(0, 0, 1, 1), onCd, style, value, "", icon, true, false)

	return 1
}

func (levels *Ui) Comp_edit(style *UiComp, valueIn string, valueInOrig string, icon string, ghost string, highlight bool, tempToValue bool) (string, bool, bool, bool) {

	lv := levels.GetCall()

	lv.call.data.scrollH.narrow = true
	lv.call.data.scrollV.show = false

	pl := levels.buff.win.io.GetPalette()
	cd, onCd := pl.GetCd(style.cd, style.fade, style.enable, false, false)

	if highlight {
		cd, onCd = pl.GetCd(CdPalette_T, false, style.enable, false, false)
	}

	edit := &levels.edit

	inDiv := lv.call.FindOrCreate("", InitOsQuad(0, 0, 1, 1), levels.GetLastApp())
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
	inDiv.data.touch_enabled = style.enable

	coord := lv.call.canvas
	coord = coord.AddSpace(levels.CellWidth(0.03))

	//if style.enable_back || highlight {
	levels.buff.AddRect(coord, cd, 0)
	//}
	//if style.enable_border
	{
		w := levels.CellWidth(0.03)
		if active {
			w *= 2
		}
		levels.buff.AddRect(coord, pl.P, w)
	}

	levels.Paint_textGrid(InitOsQuad(0, 0, 1, 1), onCd, style, value, valueInOrig, icon, true, true)

	//ghost
	if len(edit.last_edit) == 0 && len(ghost) > 0 {
		ghostStyle := style
		ghostStyle.label_alignH = 1 //center
		levels.compDrawText(coord, ghost, "", pl.GetGrey(0.7), false, false, ghostStyle)
	}

	if style.enable {
		if len(style.tooltip) > 0 {
			levels.Paint_tooltip(0, 0, 1, 1, style.tooltip)
		}
	}

	return edit.last_edit, active, (active && value != edit.last_edit), (active && this_uid != edit.uid)
}

func (levels *Ui) Comp_progress(style *UiComp, value float64, prec int) int64 {

	lv := levels.GetCall()

	value = OsClampFloat(value, 0, 1)

	pl := levels.buff.win.io.GetPalette()
	cd, onCd := pl.GetCd(style.cd, style.fade, style.enable, false, false)

	coord := lv.call.canvas

	//border
	levels.compDrawShape(coord, style.shape, cd, 0.03, 0.03)
	//back
	coord = levels.getCoord(0, 0, value, 1, 0, 0, 0)
	levels.compDrawShape(coord, style.shape, cd, 0.09, 0)
	//label
	//levels.Paint_textGrid(coord, onCd, style, strconv.FormatFloat(value*100, 'f', prec, 64)+"%", "", "", true, false)
	levels.compDrawText(coord, strconv.FormatFloat(value*100, 'f', prec, 64)+"%", "", onCd, true, false, style)

	if style.enable {
		if len(style.tooltip) > 0 {
			levels.Paint_tooltip(0, 0, 1, 1, style.tooltip)
		}
	}

	return 1
}

func (levels *Ui) Comp_sliderSimple(value *float64, minValue float64, maxValue float64, jumpValue float64, description string, desc_align int, desc_size int) bool {
	styleSlider := UiComp{enable: true, label_formating: true, cd: CdPalette_P}
	//styleDesc := Comp{enable: true, label_formating: true, cd: CdPalette_B}

	// description ....
	//if description != "" {
	//}

	_, changed, _ := levels.Comp_slider(&styleSlider, value, minValue, maxValue, jumpValue, "", 0)

	return changed
}

func (levels *Ui) Comp_slider(style *UiComp, value *float64, minValue float64, maxValue float64, jumpValue float64, imgPath string, imgMargin float64) (bool, bool, bool) {

	lv := levels.GetCall()

	old_value := *value

	_, _, inside, active, end := levels.compIsClicked(style.enable)

	pl := levels.buff.win.io.GetPalette()
	cd, _ := pl.GetCd(style.cd, style.fade, style.enable, false, false)
	cdThumb, _ := pl.GetCd(style.cd, style.fade, style.enable, inside, active)

	rad := 0.2
	radPx := levels.CellWidth(rad)
	coord := lv.call.canvas
	coord = coord.AddSpaceX(radPx)

	rpos := levels.buff.win.io.touch.pos.Sub(coord.Start)
	touch_x := OsClampFloat(float64(rpos.X)/float64(coord.Size.X), 0, 1)

	if style.enable {
		if active || inside {
			levels.Paint_cursor("hand")
		}

		if active {
			*value = minValue + (maxValue-minValue)*touch_x
		}
		if !active && inside && levels.buff.win.io.touch.wheel != 0 {
			s := maxValue - minValue
			*value += s / 10 * float64(levels.buff.win.io.touch.wheel)
			levels.buff.win.io.touch.wheel = 0 //bug: If slider has canvas which can scroll under, it will scroll and slider is ignored ...
		}

		if len(style.tooltip) > 0 {
			levels.Paint_tooltip(0, 0, 1, 1, style.tooltip)
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
		levels.buff.AddRect(cqA, cd, 0)
		levels.buff.AddRect(cqB, cd.SetAlpha(100), 0)

		//thumb(sphere)
		cqT := InitOsQuadMid(OsV2{cqB.Start.X, coord.Start.Y + coord.Size.Y/2}, OsV2{radPx * 2, radPx * 2})
		levels.buff.AddCircle(cqT, cdThumb, 0)

		//label
		if style.enable && (active || inside) {
			levels.tile.SetForce(cqT.AddSpaceY(-levels.CellWidth(0.2)), true, strconv.FormatFloat(*value, 'f', 2, 64), pl.OnB)
		}
	}

	return active, (active && old_value != *value), end
}

func (levels *Ui) Comp_combo(style *UiComp, value int64, optionsIn string) int64 {

	lv := levels.GetCall()

	var options []string
	if len(optionsIn) > 0 {
		options = strings.Split(optionsIn, "|")
	}
	var valueStr string
	if value >= 0 && value < int64(len(options)) {
		valueStr = options[value]
	}

	nmd := "combo_" + strconv.Itoa(int(lv.call.data.hash))

	click, rclick, inside, active, _ := levels.compIsClicked(style.enable)
	if style.enable {

		if active || inside {
			levels.Paint_cursor("hand")
		}

		if click > 0 || rclick > 0 {
			levels.Dialog_open(nmd, 1)
		}

		if len(style.tooltip) > 0 {
			levels.Paint_tooltip(0, 0, 1, 1, style.tooltip)
		}
	}

	pl := levels.buff.win.io.GetPalette()
	backCd, _ := pl.GetCd(style.cd, true, style.enable, inside, active) //root.GetCdGrey(0.1)
	_, onCd := pl.GetCd(style.cd, style.fade, style.enable, inside, active)

	//back and arrow
	coord := lv.call.canvas
	m := levels.CellWidth(0.03)
	coord = coord.Inner(m, m, m, m)

	levels.buff.AddRect(coord, backCd, 0)                    //background
	levels.buff.AddRect(coord, onCd, levels.CellWidth(0.03)) //border

	//text
	levels.compDrawText(coord, valueStr, "", onCd, false, false, style)
	style.label_alignH = 2
	levels.compDrawText(coord.AddSpace(levels.CellWidth(0.1)), "▼", "", onCd, false, false, style)

	//dialog
	if levels.Dialog_start(nmd) {
		//compute minimum dialog width
		mx := 0
		for _, opt := range options {
			mx = OsMax(mx, len(opt))
		}

		levels.Div_colMax(0, OsMaxFloat(5, SKYALT_FONT_HEIGHT*float64(mx)))

		menuSt := *style
		menuSt.label_alignH = 0
		menuSt.cd = CdPalette_P
		//menuSt.enable_back = true

		for i, opt := range options {
			levels.Div_start(0, i, 1, 1, "")

			//highlight
			/*if value == int64(i) {
				menuSt.cd = CdPalette_P
			} else {
				menuSt.cd = CdPalette_B
			}*/
			lclicks, _ := levels.Comp_button(&menuSt, opt, "", "", OsTrnFloat(value == int64(i), 1, 0), false)
			if lclicks > 0 {
				value = int64(i)
				levels.Dialog_close()
				break
			}

			levels.Div_end()
		}

		levels.Dialog_end()
	}

	return value
}

func (levels *Ui) Comp_checkbox(style *UiComp, value float64, label string) float64 {

	lv := levels.GetCall()

	click, rclick, inside, active, _ := levels.compIsClicked(style.enable)
	if style.enable {

		if active || inside {
			levels.Paint_cursor("hand")
		}

		if click > 0 || rclick > 0 {
			if value != 0 {
				value = 0
			} else {
				value = 1
			}
		}

		if len(style.tooltip) > 0 {
			levels.Paint_tooltip(0, 0, 1, 1, style.tooltip)
		}
	}

	pl := levels.buff.win.io.GetPalette()
	backCd := pl.GetCd2(pl.GetGrey(0.1), false, style.enable, inside, active)
	_, onCd := pl.GetCd(style.cd, style.fade, style.enable, inside, active)

	mn := levels.CellWidth(0.5)
	coord := lv.call.canvas
	if len(label) > 0 {
		coordImg, coordText := levels.compGetTextImageCoord(coord, 1, style.image_alignH, true, true)

		//border
		coordImg = OsV4_centerFull(coordImg, OsV2{mn, mn})
		levels.buff.AddRect(coordImg, backCd, 0)                    //background
		levels.buff.AddRect(coordImg, onCd, levels.CellWidth(0.03)) //border

		//text
		//levels.Paint_textGrid(InitOsQuad(1, 0, 1, 1), pl.GetOnSurface(), style, label, "", "", true, false)
		levels.compDrawText(coordText, label, "", onCd, false, false, style)

		coord = coordImg

	} else {
		//center
		coord = OsV4_centerFull(coord, OsV2{mn, mn})
		levels.buff.AddRect(coord, onCd, levels.CellWidth(0.03))
	}

	coord = coord.AddSpace(levels.CellWidth(0.1))
	if value >= 1 {
		//draw check
		levels.buff.AddLine(coord.GetPos(1.0/3, 0.9), coord.GetPos(0.05, 2.0/3), onCd, levels.CellWidth(0.05))
		levels.buff.AddLine(coord.GetPos(1.0/3, 0.9), coord.GetPos(0.95, 1.0/4), onCd, levels.CellWidth(0.05))

	} else if value > 0 {
		//draw [-]
		levels.buff.AddLine(coord.GetPos(0, 0.5), coord.GetPos(1, 0.5), onCd, levels.CellWidth(0.05))
	}

	return value
}

func (levels *Ui) Comp_switch(style *UiComp, value bool, label string) bool {

	lv := levels.GetCall()

	click, rclick, inside, active, _ := levels.compIsClicked(style.enable)
	if style.enable {
		if active || inside {
			levels.Paint_cursor("hand")
		}

		if click > 0 || rclick > 0 {
			value = !value
		}

		if len(style.tooltip) > 0 {
			levels.Paint_tooltip(0, 0, 1, 1, style.tooltip)
		}
	}

	pl := levels.buff.win.io.GetPalette()
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

	mn := levels.CellWidth(0.6)
	coord := lv.call.canvas
	if len(label) > 0 {
		coordImg, coordText := levels.compGetTextImageCoord(coord, 1.5, style.image_alignH, true, true)

		//border
		coordImg = OsV4_centerFull(coordImg, OsV2{int(float32(mn) * 1.7), mn})

		//text
		//levels.Paint_textGrid(coordText, pl.GetOnSurface(), style, label, "", "", true, false)
		levels.compDrawText(coordText, label, "", labelCd, false, false, style)

		coord = coordImg

	} else {
		//center
		coord = OsV4_centerFull(coord, OsV2{mn * 3 / 2, mn})
	}

	//back
	levels.buff.AddRect(coord, backCd, 0)

	coord = coord.AddSpace(levels.CellWidth(0.1))
	coord.Size.X /= 2
	if !value {
		levels.buff.AddRect(coord, midCd, 0)

		//0
		coord = coord.AddSpace(levels.CellWidth(0.1))
		levels.buff.AddLine(coord.GetPos(0, 0), coord.GetPos(1, 1), backCd, levels.CellWidth(0.05))
		levels.buff.AddLine(coord.GetPos(0, 1), coord.GetPos(1, 0), backCd, levels.CellWidth(0.05))

	} else {
		coord.Start.X += coord.Size.X
		levels.buff.AddRect(coord, midCd, 0)

		//I
		coord = coord.AddSpace(levels.CellWidth(0.1))
		levels.buff.AddLine(coord.GetPos(1.0/3, 0.9), coord.GetPos(0.05, 2.0/3), backCd, levels.CellWidth(0.05))
		levels.buff.AddLine(coord.GetPos(1.0/3, 0.9), coord.GetPos(0.95, 1.0/4), backCd, levels.CellWidth(0.05))
	}

	return value
}

/*func (levels *UiLayoutLevels) paint_textWidth(style *UiComp, value string, cursorPos int64, ratioH float64, fontPath string, enableFormating bool) float64 {

	//sdiv := style.GetDiv(true, app)

	if ratioH <= 0 {
		ratioH = SKYALT_FONT_HEIGHT
	}
	if len(fontPath) == 0 {
		fontPath = SKYALT_FONT_PATH
	}

	font := levels.buff.win.fonts.Get(SKYALT_FONT_PATH)
	textH := levels.CellWidth(ratioH)
	cell := float64(levels.buff.win.Cell())
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
}*/
