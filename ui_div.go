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

func (levels *UiLayoutLevels) Div_startEx(x, y, w, h int, rx, ry, rw, rh float64, name string) {

	lv := levels.GetCall()

	if !lv.call.gridLock {
		// cols/rows resizer
		lv.call.RenderResizeSpliter(levels)
		lv.call.UpdateGrid(levels.win)
		lv.call.lastChild = nil
		lv.call.gridLock = true
	}

	grid := InitOsQuad(x, y, w, h)
	lv.call = lv.call.FindOrCreate(name, grid, levels.app)

	levels.renderStart(rx, ry, rw, rh, false)

	//return !st.stack.crop.IsZero()
}

func (levels *UiLayoutLevels) Div_start(x, y, w, h int, name string) {
	levels.Div_startEx(x, y, w, h, 0, 0, 1, 1, name)
}

func (levels *UiLayoutLevels) Div_end() {

	lv := levels.GetCall()

	//if grid is empty
	if !lv.call.gridLock {
		// cols/rows resizer
		lv.call.RenderResizeSpliter(levels)
		lv.call.UpdateGrid(levels.win)
		lv.call.lastChild = nil
		lv.call.gridLock = true
	}

	levels.renderEnd(false)
}

func (levels *UiLayoutLevels) checkGridLock() bool {

	//if root.levels == nil {
	//	return false
	//}

	lv := levels.GetCall()
	if lv == nil {
		return false
	}

	if lv.call.gridLock /*&& (app.debug == nil || app.debug.conn != nil)*/ {
		lv.call.data.app.AddLogErr(errors.New("trying to changed col/row dimension after you already draw div into"))
		return false
	}
	return true
}

func (levels *UiLayoutLevels) Div_col(pos uint64, val float64) float64 {
	if !levels.checkGridLock() {
		return -1
	}

	lv := levels.GetCall()
	lv.call.GetInputCol(int(pos)).min = float32(val)
	return float64(lv.call.data.cols.GetOutput(int(pos))) / float64(levels.win.Cell())
}

func (levels *UiLayoutLevels) Div_row(pos uint64, val float64) float64 {
	if !levels.checkGridLock() {
		return -1
	}

	lv := levels.GetCall()
	lv.call.GetInputRow(int(pos)).min = float32(val)
	return float64(lv.call.data.rows.GetOutput(int(pos))) / float64(levels.win.Cell())
}

func (levels *UiLayoutLevels) Div_colMax(pos int, val float64) float64 {
	if !levels.checkGridLock() {
		return -1
	}

	lv := levels.GetCall()
	lv.call.GetInputCol(pos).max = float32(val)
	return float64(lv.call.data.cols.GetOutput(int(pos))) / float64(levels.win.Cell())
}

func (levels *UiLayoutLevels) Div_rowMax(pos uint64, val float64) float64 {
	if !levels.checkGridLock() {
		return -1
	}

	lv := levels.GetCall()
	lv.call.GetInputRow(int(pos)).max = float32(val)
	return float64(lv.call.data.rows.GetOutput(int(pos))) / float64(levels.win.Cell())
}

func (levels *UiLayoutLevels) Div_colResize(pos uint64, name string, val float64) float64 {
	if !levels.checkGridLock() {
		return -1
	}
	lv := levels.GetCall()

	//if 'resize' exist in layout than read it from there
	if len(name) == 0 {
		name = strconv.Itoa(int(pos))
	}
	res, found := lv.call.data.cols.FindOrAddResize(name)
	if !found {
		res.value = float32(val)
	}
	lv.call.GetInputCol(int(pos)).resize = res

	return float64(lv.call.data.cols.GetOutput(int(pos))) / float64(levels.win.Cell())
}

func (levels *UiLayoutLevels) Div_rowResize(pos uint64, name string, val float64) float64 {
	if !levels.checkGridLock() {
		return -1
	}
	lv := levels.GetCall()

	//if 'resize' exist in layout than read it from there
	if len(name) == 0 {
		name = strconv.Itoa(int(pos))
	}
	res, found := lv.call.data.rows.FindOrAddResize(name)
	if !found {
		res.value = float32(val)
	}
	lv.call.GetInputRow(int(pos)).resize = res

	return float64(lv.call.data.rows.GetOutput(int(pos))) / float64(levels.win.Cell())
}

func (levels *UiLayoutLevels) Div_drag(groupName string, id uint64) {

	lv := levels.GetCall()

	if lv.call.data.touch_active {
		drag := &levels.drag
		//set
		drag.div = lv.call
		drag.group = groupName
		drag.id = id

		//paint
		levels.Paint_rect(0, 0, 1, 1, 0, OsCd{0, 0, 0, 180}, 0) //fade
	}
}
func (levels *UiLayoutLevels) Div_drop(groupName string, vertical bool, horizontal bool, inside bool) (uint64, int, bool) {

	lv := levels.GetCall()

	id := uint64(0)
	pos := 0
	done := false

	touchPos := levels.win.io.touch.pos
	q := lv.call.crop

	drag := &levels.drag
	if q.Inside(touchPos) && strings.EqualFold(drag.group, groupName) && lv.call != drag.div {

		r := touchPos.Sub(lv.call.crop.Middle())

		if vertical && horizontal {
			arx := float32(OsAbs(r.X)) / float32(lv.call.crop.Size.X)
			ary := float32(OsAbs(r.Y)) / float32(lv.call.crop.Size.Y)
			if arx > ary {
				if r.X < 0 {
					pos = 3 //H_LEFT
				} else {
					pos = 4 //H_RIGHT
				}
			} else {
				if r.Y < 0 {
					pos = 1 //V_LEFT
				} else {
					pos = 2 //V_RIGHT
				}
			}
		} else if vertical {
			if r.Y < 0 {
				pos = 1 //V_LEFT
			} else {
				pos = 2 //V_RIGHT
			}
		} else if horizontal {
			if r.X < 0 {
				pos = 3 //H_LEFT
			} else {
				pos = 4 //H_RIGHT
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
		wx := float64(levels.CellWidth(0.1)) / float64(lv.call.canvas.Size.X)
		wy := float64(levels.CellWidth(0.1)) / float64(lv.call.canvas.Size.Y)
		switch pos {
		case 0: //SA_Drop_INSIDE
			levels.Paint_rect(0, 0, 1, 1, 0, OsCd{0, 0, 0, 180}, 0.03)

		case 1: //SA_Drop_V_LEFT
			levels.Paint_rect(0, 0, 1, wy, 0, OsCd{0, 0, 0, 180}, 0)

		case 2: //SA_Drop_V_RIGHT
			levels.Paint_rect(0, 1-wy, 1, 1, 0, OsCd{0, 0, 0, 180}, 0)

		case 3: //SA_Drop_H_LEFT
			levels.Paint_rect(0, 0, wx, 1, 0, OsCd{0, 0, 0, 180}, 0)

		case 4: //SA_Drop_H_RIGHT
			levels.Paint_rect(1-wx, 0, 1, 1, 0, OsCd{0, 0, 0, 180}, 0)
		}

		id = drag.id
		//if st.stack.data.touch_end {
		if levels.win.io.touch.end {
			done = true
		}

	}

	return id, pos, done
}
