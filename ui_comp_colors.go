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
	"encoding/binary"
	"fmt"
	"strconv"
)

func (ui *Ui) comp_colorPicker(cd *OsCd, dialogName string, enable bool) bool {
	origCd := *cd
	cd.A = 255

	ui.Div_colMax(0, 100)
	ui.Div_rowMax(0, 100)

	if ui._colorButton(0, 0, 1, 1, "", "", *cd, enable) {
		ui.Dialog_open(dialogName, 1)
	}

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

		if ui.Comp_slider_desc(ui.trns.RED, 0, 1.5, 0, 0, 1, 1, &r, 0, 255, 1) {
			cd.R = uint8(r)
		}
		if ui.Comp_slider_desc(ui.trns.GREEN, 0, 1.5, 0, 1, 1, 1, &g, 0, 255, 1) {
			cd.G = uint8(g)
		}
		if ui.Comp_slider_desc(ui.trns.BLUE, 0, 1.5, 0, 2, 1, 1, &b, 0, 255, 1) {
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

		if ui.Comp_slider_desc(ui.trns.HUE, 0, 2, 0, 0, 1, 1, &h, 0, 359, 1) {
			changed = true
		}
		if ui.Comp_slider_desc(ui.trns.SATURATION, 0, 2, 0, 1, 1, 1, &s, 0, 1, 0.01) {
			changed = true
		}
		if ui.Comp_slider_desc(ui.trns.LIGHTNESS, 0, 2, 0, 2, 1, 1, &l, 0, 1, 0.01) {
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

	click = ui.Comp_buttonText(x, y, w, h, value, "", tooltip, enable, false) > 0 //transparent, so background is seen

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

type HSL struct {
	H int
	S float64
	L float64
}

func _hueToRGB(v1, v2, vH float64) float64 {
	if vH < 0 {
		vH++
	}
	if vH > 1 {
		vH--
	}

	if (6 * vH) < 1 {
		return v1 + (v2-v1)*6*vH
	} else if (2 * vH) < 1 {
		return v2
	} else if (3 * vH) < 2 {
		return v1 + (v2-v1)*((2.0/3)-vH)*6
	}

	return v1
}

func INTtoRGB(v uint32) OsCd {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], v)

	return OsCd{R: b[0], G: b[1], B: b[2], A: b[3]}
}
func (rgb OsCd) RGBtoINT() uint32 {
	b := []byte{rgb.R, rgb.G, rgb.B, rgb.A}
	return binary.LittleEndian.Uint32(b)
}

func (hsl HSL) HSLtoRGB() OsCd {
	cd := OsCd{A: 255}

	if hsl.S == 0 {
		ll := hsl.L * 255
		ll = float64(OsClampFloat(float64(ll), 0, 255))

		cd.R = uint8(ll)
		cd.G = uint8(ll)
		cd.B = uint8(ll)
	} else {
		var v2 float64
		if hsl.L < 0.5 {
			v2 = (hsl.L * (1 + hsl.S))
		} else {
			v2 = ((hsl.L + hsl.S) - (hsl.L * hsl.S))
		}
		v1 := 2*hsl.L - v2

		hue := float64(hsl.H) / 360
		cd.R = uint8(255 * _hueToRGB(v1, v2, hue+(1.0/3)))
		cd.G = uint8(255 * _hueToRGB(v1, v2, hue))
		cd.B = uint8(255 * _hueToRGB(v1, v2, hue-(1.0/3)))
	}

	return cd
}

func (cd OsCd) RGBtoHSL() HSL {
	var hsl HSL

	r := float64(cd.R) / 255
	g := float64(cd.G) / 255
	b := float64(cd.B) / 255

	min := OsMinFloat(OsMinFloat(r, g), b)
	max := OsMaxFloat(OsMaxFloat(r, g), b)
	delta := max - min

	hsl.L = (max + min) / 2

	if delta == 0 {
		hsl.H = 0
		hsl.S = 0
	} else {
		if hsl.L <= 0.5 {
			hsl.S = delta / (max + min)
		} else {
			hsl.S = delta / (2 - max - min)
		}

		var hue float64
		if r == max {
			hue = ((g - b) / 6) / delta
		} else if g == max {
			hue = (1.0 / 3) + ((b-r)/6)/delta
		} else {
			hue = (2.0 / 3) + ((r-g)/6)/delta
		}

		if hue < 0 {
			hue += 1
		}
		if hue > 1 {
			hue -= 1
		}

		hsl.H = int(hue * 360)
	}

	return hsl
}

func HEXtoRGBwithCheck(hex string, defaultCd OsCd) OsCd {
	if len(hex) == 6 || (len(hex) == 7 && hex[0] == '#') {
		return HEXtoRGB(hex)
	}
	return defaultCd
}

func HEXtoRGB(hex string) OsCd {
	cd := OsCd{A: 255}

	if len(hex) == 0 {
		return cd
	}

	if hex[0] == '#' {
		hex = hex[1:] //skip
	}

	if len(hex) < 2 {
		return cd
	}
	r, _ := strconv.ParseInt(hex[:2], 16, 16)
	cd.R = uint8(r)
	hex = hex[2:]

	if len(hex) < 2 {
		return cd
	}
	g, _ := strconv.ParseInt(hex[:2], 16, 16)
	cd.G = uint8(g)
	hex = hex[2:]

	if len(hex) < 2 {
		return cd
	}
	b, _ := strconv.ParseInt(hex[:2], 16, 16)
	cd.B = uint8(b)

	return cd
}

func (cd OsCd) RGBtoHEX() string {
	return fmt.Sprintf("#%02x%02x%02x", cd.R, cd.G, cd.B)
}
