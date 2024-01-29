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

func (ui *Ui) getCoord(x, y, w, h float64, margin float64, marginX float64, marginY float64) OsV4 {

	return ui.GetCall().call.canvas.CutEx(x, y, w, h, ui.CellWidth(margin), ui.CellWidth(marginX), ui.CellWidth(marginY))
}

func (ui *Ui) Paint_rect(x, y, w, h float64, margin float64, cd OsCd, borderWidth float64) bool {

	lv := ui.GetCall()
	if lv.call == nil || lv.call.crop.IsZero() {
		return false
	}
	ui.buff.AddRect(ui.getCoord(x, y, w, h, margin, 0, 0), cd, ui.CellWidth(borderWidth))
	return true
}

func (ui *Ui) Paint_rect_round(x, y, w, h float64, margin float64, cd OsCd, rad float64, borderWidth float64) bool {

	lv := ui.GetCall()
	if lv.call == nil || lv.call.crop.IsZero() {
		return false
	}
	ui.buff.AddRectRound(ui.getCoord(x, y, w, h, margin, 0, 0), ui.CellWidth(rad), cd, ui.CellWidth(borderWidth))
	return true
}

func (ui *Ui) Paint_line(x, y, w, h float64, margin float64, sx, sy, ex, ey float64, cd OsCd, width float64) bool {

	lv := ui.GetCall()
	if lv.call == nil || lv.call.crop.IsZero() {
		return false
	}

	coord := ui.getCoord(x, y, w, h, margin, 0, 0)
	var start OsV2
	start.X = coord.Start.X + int(float64(coord.Size.X)*sx)
	start.Y = coord.Start.Y + int(float64(coord.Size.Y)*sy)

	var end OsV2
	end.X = coord.Start.X + int(float64(coord.Size.X)*ex)
	end.Y = coord.Start.Y + int(float64(coord.Size.Y)*ey)

	ui.buff.AddLine(start, end, cd, ui.CellWidth(width))
	return true
}

func (ui *Ui) Paint_circle(x, y, w, h float64, margin float64, sx, sy, rad float64, cd OsCd, borderWidth float64) bool {

	lv := ui.GetCall()
	if lv.call == nil || lv.call.crop.IsZero() {
		return false
	}

	coord := ui.getCoord(x, y, w, h, margin, 0, 0)
	var s OsV2
	s.X = coord.Start.X + int(float64(coord.Size.X)*sx)
	s.Y = coord.Start.Y + int(float64(coord.Size.Y)*sy)
	rr := ui.CellWidth(rad)
	cq := InitOsV4Mid(s, OsV2{rr * 2, rr * 2})

	ui.buff.AddCircle(cq, cd, ui.CellWidth(borderWidth))
	return true
}

func (ui *Ui) Paint_file(x, y, w, h float64, margin float64, path WinMedia, cd OsCd, alignV, alignH int, fill bool, background bool) bool {

	lv := ui.GetCall()
	if lv.call == nil || lv.call.crop.IsZero() {
		return false
	}

	coord := ui.getCoord(x, y, w, h, margin, 0, 0)

	ui.buff.AddImage(path, coord, cd, alignV, alignH, fill, background)

	return true
}

func (ui *Ui) Paint_tooltipDiv(div *UiLayoutDiv, x, y, w, h float64, text string) bool {

	if div == nil || div.crop.IsZero() || ui.touch.IsAnyActive() {
		return false
	}

	if div.enableInput {
		coord := ui.getCoord(x, y, w, h, 0, 0, 0)

		if coord.HasIntersect(div.crop) {
			ui.tile.Set(ui.win.io.touch.pos, coord, false, text, ui.win.io.GetPalette().OnB)
		}
	}
	return true
}

func (ui *Ui) Paint_tooltip(x, y, w, h float64, text string) bool {
	return ui.Paint_tooltipDiv(ui.GetCall().call, x, y, w, h, text)

}

func (ui *Ui) Paint_cursor(name string) bool {

	lv := ui.GetCall()
	if lv.call == nil || lv.call.crop.IsZero() {
		return false
	}

	if lv.call.enableInput {
		err := ui.win.PaintCursor(name)
		if err != nil {
			lv.call.data.app.AddLogErr(err)
			return false
		}
	}
	return true
}
