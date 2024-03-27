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

func (ui *Ui) comp_colorPicker(x, y, w, h int, cd *OsCd, dialogName string, tooltip string, enable bool) bool {
	origCd := *cd
	cd.A = 255

	ui.Div_start(x, y, w, h)
	ui.Div_colMax(0, 100)
	ui.Div_rowMax(0, 100)
	if ui._colorButton(0, 0, 1, 1, "", fmt.Sprintf("%s(RGBA: %d, %d, %d, %d)", tooltip, cd.R, cd.G, cd.B, cd.A), *cd, enable) {
		ui.Dialog_open(dialogName, 1)
	}
	ui.Div_end()

	dialogOpen := ui.Dialog_start(dialogName)
	if dialogOpen {
		ui.comp_colorPalette(cd)
		ui.Dialog_end()
	}

	return !origCd.Cmp(*cd)
}

func (ui *Ui) comp_colorPalette(cd *OsCd) bool {

	orig_cd := *cd

	ui.Div_colMax(0, 7)
	ui.Div_colMax(1, 7)

	//final color
	ui.Div_start(0, 0, 2, 1)
	ui.Paint_rect(0, 0, 1, 1, 0.06, *cd, 0)
	ui.Div_end()

	//RGB
	ui.Div_start(0, 1, 1, 3)
	{
		ui.Div_colMax(0, 100)

		r := float64(cd.R)
		g := float64(cd.G)
		b := float64(cd.B)

		if ui.Comp_slider_desc(ui.trns.RED, 0, 1.5, 0, 0, 1, 1, &r, 0, 255, 1, true) {
			cd.R = uint8(r)
		}
		if ui.Comp_slider_desc(ui.trns.GREEN, 0, 1.5, 0, 1, 1, 1, &g, 0, 255, 1, true) {
			cd.G = uint8(g)
		}
		if ui.Comp_slider_desc(ui.trns.BLUE, 0, 1.5, 0, 2, 1, 1, &b, 0, 255, 1, true) {
			cd.B = uint8(b)
		}
	}
	ui.Div_end()

	//HSL
	ui.Div_start(1, 1, 1, 3)
	{
		ui.Div_colMax(0, 100)

		hsl := cd.RGBtoHSL()
		h := float64(hsl.H)
		s := float64(hsl.S)
		l := float64(hsl.L)
		changed := false

		if ui.Comp_slider_desc(ui.trns.HUE, 0, 2, 0, 0, 1, 1, &h, 0, 359, 1, true) {
			changed = true
		}
		if ui.Comp_slider_desc(ui.trns.SATURATION, 0, 2, 0, 1, 1, 1, &s, 0, 1, 0.01, true) {
			changed = true
		}
		if ui.Comp_slider_desc(ui.trns.LIGHTNESS, 0, 2, 0, 2, 1, 1, &l, 0, 1, 0.01, true) {
			changed = true
		}
		if changed {
			*cd = HSL{H: int(h), S: float64(s), L: float64(l)}.HSLtoRGB()
		}
	}
	ui.Div_end()

	//rainbow
	ui.Div_start(0, 5, 2, 1)
	ui._colorPickerHueRainbow(cd)
	ui.Div_end()

	//pre-build
	ui.Div_start(0, 6, 2, 1)
	{
		for i := 0; i < 12; i++ {
			ui.Div_col(i, 0.25)
			ui.Div_colMax(i, 100)
		}

		//first 8
		for i := 0; i < 8; i++ {
			cdd := HSL{H: int(360 * float64(i) / 8), S: 0.7, L: 0.5}.HSLtoRGB()
			if ui._colorButton(i, 0, 1, 1, "", "", cdd, true) {
				*cd = cdd
			}
		}

		//other 4
		for i := 0; i < 4; i++ {
			cdd := HSL{H: 0, S: 0, L: float64(i) / 4}.HSLtoRGB()
			if ui._colorButton(8+i, 0, 1, 1, "", "", cdd, true) {
				*cd = cdd
			}
		}
	}
	ui.Div_end()

	return !orig_cd.Cmp(*cd)
}

func (ui *Ui) _colorButton(x, y, w, h int, value string, tooltip string, cd OsCd, enable bool) bool {
	var click bool

	ui.Div_start(x, y, w, h)
	ui.Paint_rect(0, 0, 1, 1, 0.06, cd, 0) //background
	ui.Div_end()

	click = ui.Comp_buttonText(x, y, w, h, value, Comp_buttonProp().Enable(enable).Tooltip(tooltip)) > 0 //transparent, so background is seen

	return click
}

func (ui *Ui) _colorPickerHueRainbow(cd *OsCd) bool {

	n := ui.DivInfo_get(SA_DIV_GET_layoutWidth, 0) * 5 //5 lines in 1 cell

	//draw rainbow
	st := 1 / n
	last_i := float64(0)
	for i := st; i < 1+st; i += st {
		//p = i / n
		rgb := HSL{H: int(360 * i), S: 1, L: 0.5}.HSLtoRGB()

		ui.Paint_rect(last_i, 0, (i-last_i)+0.06, 1, 0, rgb, 0)
		last_i = i
	}

	//selected position
	cdd := cd.RGBtoHSL()
	p := float64(cdd.H) / 360
	ui.Paint_line(0, 0, 1, 1, 0, p, 0, p, 1, InitOsCd32(0, 0, 0, 255), 0.06)

	//picker
	changed := false
	if ui.DivInfo_get(SA_DIV_GET_touchInside, 0) > 0 {
		ui.Paint_cursor("hand")

		if ui.DivInfo_get(SA_DIV_GET_touchActive, 0) > 0 {
			x := ui.DivInfo_get(SA_DIV_GET_touchX, 0)
			x = float64(OsClampFloat(float64(x), 0, 1))

			*cd = HSL{H: int(360 * x), S: 0.7, L: 0.5}.HSLtoRGB()
			changed = true
		}
	}
	return changed
}
