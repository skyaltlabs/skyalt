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
	"errors"
	"strconv"
	"strings"
)

func (ui *Ui) Div_startEx(x, y, w, h int, rx, ry, rw, rh float64, name string) {

	lv := ui.GetCall()

	if !lv.call.gridLock {
		// cols/rows resizer
		lv.call.RenderResizeSpliter(ui)
		lv.call.UpdateGrid(ui.win)
		lv.call.lastChild = nil
		lv.call.gridLock = true
	}

	grid := InitOsQuad(x, y, w, h)
	lv.call = lv.call.FindOrCreate(name, grid, ui.GetLastApp())

	ui.renderStart(rx, ry, rw, rh, false)

	//return !st.stack.crop.IsZero()
}

func (ui *Ui) Div_start(x, y, w, h int, name string) {
	ui.Div_startEx(x, y, w, h, 0, 0, 1, 1, name)
}

func (ui *Ui) Div_end() {

	lv := ui.GetCall()

	//if grid is empty
	if !lv.call.gridLock {
		// cols/rows resizer
		lv.call.RenderResizeSpliter(ui)
		lv.call.UpdateGrid(ui.win)
		lv.call.lastChild = nil
		lv.call.gridLock = true
	}

	ui.renderEnd(false)
}

func (ui *Ui) checkGridLock() bool {

	//if root.levels == nil {
	//	return false
	//}

	lv := ui.GetCall()
	if lv == nil {
		return false
	}

	if lv.call.gridLock /*&& (app.debug == nil || app.debug.conn != nil)*/ {
		lv.call.data.app.AddLogErr(errors.New("trying to changed col/row dimension after you already draw div into"))
		return false
	}
	return true
}

func (ui *Ui) Div_col(pos int, val float64) float64 {
	if !ui.checkGridLock() {
		return -1
	}

	lv := ui.GetCall()
	lv.call.GetInputCol(int(pos)).min = float32(val)
	return float64(lv.call.data.cols.GetOutput(int(pos))) / float64(ui.win.Cell())
}

func (ui *Ui) Div_row(pos int, val float64) float64 {
	if !ui.checkGridLock() {
		return -1
	}

	lv := ui.GetCall()
	lv.call.GetInputRow(int(pos)).min = float32(val)
	return float64(lv.call.data.rows.GetOutput(int(pos))) / float64(ui.win.Cell())
}

func (ui *Ui) Div_colMax(pos int, val float64) float64 {
	if !ui.checkGridLock() {
		return -1
	}

	lv := ui.GetCall()
	lv.call.GetInputCol(pos).max = float32(val)
	return float64(lv.call.data.cols.GetOutput(int(pos))) / float64(ui.win.Cell())
}

func (ui *Ui) Div_rowMax(pos int, val float64) float64 {
	if !ui.checkGridLock() {
		return -1
	}

	lv := ui.GetCall()
	lv.call.GetInputRow(int(pos)).max = float32(val)
	return float64(lv.call.data.rows.GetOutput(int(pos))) / float64(ui.win.Cell())
}

func (ui *Ui) Div_colResize(pos int, name string, val float64) float64 {
	if !ui.checkGridLock() {
		return -1
	}
	lv := ui.GetCall()

	//if 'resize' exist in layout than read it from there
	if len(name) == 0 {
		name = strconv.Itoa(int(pos))
	}
	res, found := lv.call.data.cols.FindOrAddResize(name)
	if !found {
		res.value = float32(val)
	}
	lv.call.GetInputCol(int(pos)).resize = res

	return float64(lv.call.data.cols.GetOutput(int(pos))) / float64(ui.win.Cell())
}

func (ui *Ui) Div_rowResize(pos int, name string, val float64) float64 {
	if !ui.checkGridLock() {
		return -1
	}
	lv := ui.GetCall()

	//if 'resize' exist in layout than read it from there
	if len(name) == 0 {
		name = strconv.Itoa(int(pos))
	}
	res, found := lv.call.data.rows.FindOrAddResize(name)
	if !found {
		res.value = float32(val)
	}
	lv.call.GetInputRow(int(pos)).resize = res

	return float64(lv.call.data.rows.GetOutput(int(pos))) / float64(ui.win.Cell())
}

func (ui *Ui) Div_SpacerRow(x, y, w, h int) {

	pl := ui.buff.win.io.GetPalette()

	ui.Div_start(x, y, w, h, "")
	ui.Paint_line(0, 0, 1, 1, 0,
		0, 0.5, 1, 0.5,
		pl.GetGrey(0.3), 0.03)
	ui.Div_end()
}

func (ui *Ui) Div_SpacerCol(x, y, w, h int) {

	pl := ui.buff.win.io.GetPalette()

	ui.Div_start(x, y, w, h, "")
	ui.Paint_line(0, 0, 1, 1, 0,
		0.5, 0, 0.5, 1,
		pl.GetGrey(0.3), 0.03)
	ui.Div_end()
}

func (ui *Ui) Div_drag(groupName string, id int) {

	lv := ui.GetCall()

	if lv.call.data.touch_active {
		drag := &ui.drag
		//set
		drag.div = lv.call
		drag.group = groupName
		drag.id = id

		//paint
		ui.Paint_rect(0, 0, 1, 1, 0, OsCd{0, 0, 0, 180}, 0) //fade
	}
}

type SA_Drop_POS int

const (
	SA_Drop_INSIDE  SA_Drop_POS = 0
	SA_Drop_V_LEFT  SA_Drop_POS = 1
	SA_Drop_V_RIGHT SA_Drop_POS = 2
	SA_Drop_H_LEFT  SA_Drop_POS = 3
	SA_Drop_H_RIGHT SA_Drop_POS = 4
)

func (ui *Ui) Div_drop(groupName string, vertical bool, horizontal bool, inside bool) (int, SA_Drop_POS, bool) {

	lv := ui.GetCall()

	id := 0
	pos := SA_Drop_INSIDE
	done := false

	touchPos := ui.win.io.touch.pos
	q := lv.call.crop

	drag := &ui.drag
	if q.Inside(touchPos) && strings.EqualFold(drag.group, groupName) && lv.call != drag.div {

		r := touchPos.Sub(lv.call.crop.Middle())

		if vertical && horizontal {
			arx := float32(OsAbs(r.X)) / float32(lv.call.crop.Size.X)
			ary := float32(OsAbs(r.Y)) / float32(lv.call.crop.Size.Y)
			if arx > ary {
				if r.X < 0 {
					pos = SA_Drop_H_LEFT
				} else {
					pos = SA_Drop_H_RIGHT
				}
			} else {
				if r.Y < 0 {
					pos = SA_Drop_V_LEFT
				} else {
					pos = SA_Drop_V_RIGHT
				}
			}
		} else if vertical {
			if r.Y < 0 {
				pos = SA_Drop_V_LEFT
			} else {
				pos = SA_Drop_V_RIGHT
			}
		} else if horizontal {
			if r.X < 0 {
				pos = SA_Drop_H_LEFT
			} else {
				pos = SA_Drop_H_RIGHT
			}
		}

		if inside {
			if vertical {
				q = q.AddSpaceY(lv.call.crop.Size.Y / 3)
			}
			if horizontal {
				q = q.AddSpaceX(lv.call.crop.Size.X / 3)
			}

			if !vertical && !horizontal {
				pos = 0
			} else if q.Inside(touchPos) {
				pos = 0
			}
		}

		//paint
		wx := float64(ui.CellWidth(0.1)) / float64(lv.call.canvas.Size.X)
		wy := float64(ui.CellWidth(0.1)) / float64(lv.call.canvas.Size.Y)
		switch pos {
		case SA_Drop_INSIDE:
			ui.Paint_rect(0, 0, 1, 1, 0, OsCd{0, 0, 0, 180}, 0.03)

		case SA_Drop_V_LEFT:
			ui.Paint_rect(0, 0, 1, wy, 0, OsCd{0, 0, 0, 180}, 0)

		case SA_Drop_V_RIGHT:
			ui.Paint_rect(0, 1-wy, 1, 1, 0, OsCd{0, 0, 0, 180}, 0)

		case SA_Drop_H_LEFT:
			ui.Paint_rect(0, 0, wx, 1, 0, OsCd{0, 0, 0, 180}, 0)

		case SA_Drop_H_RIGHT:
			ui.Paint_rect(1-wx, 0, 1, 1, 0, OsCd{0, 0, 0, 180}, 0)
		}

		id = drag.id
		//if st.stack.data.touch_end {
		if ui.win.io.touch.end {
			done = true
		}

	}

	return id, pos, done
}

func Div_DropMoveElement[T any](array_src *[]T, array_dst *[]T, src int, dst int, pos SA_Drop_POS) {

	//check
	if src < dst && (pos == SA_Drop_V_LEFT || pos == SA_Drop_H_LEFT) {
		dst--
	}
	if src > dst && (pos == SA_Drop_V_RIGHT || pos == SA_Drop_H_RIGHT) {
		dst++
	}

	//move(by swap one-by-one)
	if array_src == array_dst {
		for i := src; i < dst; i++ {
			(*array_dst)[i], (*array_dst)[i+1] = (*array_dst)[i+1], (*array_dst)[i]
		}
		for i := src; i > dst; i-- {
			(*array_dst)[i], (*array_dst)[i-1] = (*array_dst)[i-1], (*array_dst)[i]
		}
	} else {

		backup := (*array_src)[src]

		//remove
		*array_src = append((*array_src)[:src], (*array_src)[src+1:]...)

		//insert
		if dst < len(*array_dst) {
			*array_dst = append((*array_dst)[:dst+1], (*array_dst)[dst:]...)
			(*array_dst)[dst] = backup
		} else {
			*array_dst = append(*array_dst, backup)
		}
	}
}
