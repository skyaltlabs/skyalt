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

func (levels *Ui) getCoord(x, y, w, h float64, margin float64, marginX float64, marginY float64) OsV4 {

	return levels.GetCall().call.canvas.CutEx(x, y, w, h, levels.CellWidth(margin), levels.CellWidth(marginX), levels.CellWidth(marginY))
}

func (levels *Ui) Paint_rect(x, y, w, h float64, margin float64, cd OsCd, borderWidth float64) bool {

	lv := levels.GetCall()
	if lv.call == nil || lv.call.crop.IsZero() {
		return false
	}
	levels.buff.AddRect(levels.getCoord(x, y, w, h, margin, 0, 0), cd, levels.CellWidth(borderWidth))
	return true
}

func (levels *Ui) Paint_line(x, y, w, h float64, margin float64, sx, sy, ex, ey float64, cd OsCd, width float64) bool {

	lv := levels.GetCall()
	if lv.call == nil || lv.call.crop.IsZero() {
		return false
	}

	coord := levels.getCoord(x, y, w, h, margin, 0, 0)
	var start OsV2
	start.X = coord.Start.X + int(float64(coord.Size.X)*sx)
	start.Y = coord.Start.Y + int(float64(coord.Size.Y)*sy)

	var end OsV2
	end.X = coord.Start.X + int(float64(coord.Size.X)*ex)
	end.Y = coord.Start.Y + int(float64(coord.Size.Y)*ey)

	levels.buff.AddLine(start, end, cd, levels.CellWidth(width))
	return true
}

func (levels *Ui) Paint_circle(x, y, w, h float64, margin float64, sx, sy, rad float64, cd OsCd, borderWidth float64) bool {

	lv := levels.GetCall()
	if lv.call == nil || lv.call.crop.IsZero() {
		return false
	}

	coord := levels.getCoord(x, y, w, h, margin, 0, 0)
	var s OsV2
	s.X = coord.Start.X + int(float64(coord.Size.X)*sx)
	s.Y = coord.Start.Y + int(float64(coord.Size.Y)*sy)
	rr := levels.CellWidth(rad)
	cq := InitOsQuadMid(s, OsV2{rr * 2, rr * 2})

	levels.buff.AddCircle(cq, cd, levels.CellWidth(borderWidth))
	return true
}

func (levels *Ui) Paint_file(x, y, w, h float64, margin float64, url string, cd OsCd, alignV, alignH int, fill bool) bool {

	lv := levels.GetCall()
	if lv.call == nil || lv.call.crop.IsZero() {
		return false
	}

	coord := levels.getCoord(x, y, w, h, margin, 0, 0)

	path, err := InitWinMedia(url)
	if err != nil {
		lv.call.data.app.AddLogErr(err)
		return false
	}

	levels.buff.AddImage(path, coord, cd, alignV, alignH, fill)

	return true
}

func (levels *Ui) Paint_tooltip(x, y, w, h float64, text string) bool {

	lv := levels.GetCall()
	if lv.call == nil || lv.call.crop.IsZero() {
		return false
	}

	if lv.call.enableInput {
		coord := levels.getCoord(x, y, w, h, 0, 0, 0)

		if coord.HasIntersect(lv.call.crop) {
			levels.tile.Set(levels.win.io.touch.pos, coord, false, text, levels.win.io.GetPalette().OnB)
		}
	}
	return true
}

func (levels *Ui) Paint_cursor(name string) bool {

	lv := levels.GetCall()
	if lv.call == nil || lv.call.crop.IsZero() {
		return false
	}

	if lv.call.enableInput {
		err := levels.win.PaintCursor(name)
		if err != nil {
			lv.call.data.app.AddLogErr(err)
			return false
		}
	}
	return true
}
