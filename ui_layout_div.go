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
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"strings"
)

type UiLayoutDiv struct {
	parent *UiLayoutDiv

	name string

	grid OsV4

	canvas OsV4
	crop   OsV4 //scroll bars NOT included

	data UiLayout

	touch_enabled bool //this is set by user
	enableInput   bool //this is computed

	childs    []*UiLayoutDiv
	lastChild *UiLayoutDiv

	gridLock bool //app must set cols/rows first, then draw div(s)
	use      bool //for maintenance

	touchResizeIndex int
	touchResizeIsCol bool
}

func (div *UiLayoutDiv) Use() {
	div.use = true
}

func (div *UiLayoutDiv) UseAll() {
	div.use = true
	for _, it := range div.childs {
		it.UseAll()
	}
}

func (div *UiLayoutDiv) CropWithScroll(win *Win) OsV4 {
	ret := div.crop

	if div.data.scrollV.Is() {
		ret.Size.X += div.data.scrollV._GetWidth(win)
	}

	if div.data.scrollH.Is() {
		ret.Size.Y += div.data.scrollH._GetWidth(win)
	}

	if div.parent != nil {
		ret = ret.GetIntersect(div.parent.crop)
	}

	return ret
}

func (div *UiLayoutDiv) Print(newLine bool) {
	if div.parent != nil {
		div.parent.Print(false)
	}
	fmt.Printf("[%s : %d, %d, %d, %d](%d, %d, %d, %d)", div.name, div.grid.Start.X, div.grid.Start.Y, div.grid.Size.X, div.grid.Size.Y, div.canvas.Start.X, div.canvas.Start.Y, div.canvas.Size.X, div.canvas.Size.Y)
	if newLine {
		fmt.Printf("\n")
	}
}

func (div *UiLayoutDiv) GetParent(deep int) *UiLayoutDiv {
	act := div
	for deep > 0 && act != nil {
		act = act.parent
		deep--
	}
	return act
}

func (div *UiLayoutDiv) ResetGridLock() {
	div.gridLock = false
	for _, it := range div.childs {
		it.ResetGridLock()
	}
}

func (div *UiLayoutDiv) computeHash() uint64 {
	var tmp [8]byte

	h := sha256.New()
	for div != nil {

		binary.LittleEndian.PutUint64(tmp[:], uint64(div.grid.Start.X))
		h.Write(tmp[:])
		binary.LittleEndian.PutUint64(tmp[:], uint64(div.grid.Start.Y))
		h.Write(tmp[:])
		binary.LittleEndian.PutUint64(tmp[:], uint64(div.grid.Size.X))
		h.Write(tmp[:])
		binary.LittleEndian.PutUint64(tmp[:], uint64(div.grid.Size.Y))
		h.Write(tmp[:])

		h.Write([]byte(div.name))

		div = div.parent
	}
	return binary.LittleEndian.Uint64(h.Sum(nil))
}

func NewUiLayoutPack(parent *UiLayoutDiv, name string, grid OsV4, app *UiLayoutApp) *UiLayoutDiv {
	var div UiLayoutDiv

	div.name = name
	div.parent = parent
	div.grid = grid

	div.touch_enabled = true
	div.data.Init(div.computeHash(), app)

	div.Use()

	return &div
}

func (div *UiLayoutDiv) ClearChilds() {
	for _, it := range div.childs {
		it.Destroy()
	}
	div.childs = []*UiLayoutDiv{}
}

func (div *UiLayoutDiv) Destroy() {
	div.ClearChilds()
	div.data.Close()
}

func (div *UiLayoutDiv) FindOrCreate(name string, grid OsV4, app *UiLayoutApp) *UiLayoutDiv {
	//finds
	for _, it := range div.childs {
		if strings.EqualFold(it.name, name) && it.grid.Cmp(grid) && it.data.app == app {
			div.lastChild = it
			it.Use()
			return it
		}
	}

	// creates
	l := NewUiLayoutPack(div, name, grid, app)
	div.childs = append(div.childs, l)
	div.lastChild = l
	return l
}

func (div *UiLayoutDiv) FindFromGridPos(gridPos OsV2) *UiLayoutDiv {

	var last *UiLayoutDiv
	for _, it := range div.childs {
		if it.grid.Inside(gridPos) {
			last = it
		}
	}
	return last
}

func (div *UiLayoutDiv) FindFromCanvasPos(canvasPos OsV2) *UiLayoutDiv {

	for _, it := range div.childs {
		if it.crop.Inside(canvasPos) {
			return it
		}
	}
	return nil
}

func (div *UiLayoutDiv) GetGridMax(minSize OsV2) OsV2 {
	mx := minSize
	for _, it := range div.childs {
		mx = mx.Max(it.grid.End())
	}

	mx = mx.Max(OsV2{div.data.cols.NumInputs(), div.data.rows.NumInputs()})

	return mx
}

func (div *UiLayoutDiv) GetLevelSize(winRect OsV4, win *Win) OsV4 {

	cell := win.Cell()
	q := OsV4{OsV2{}, div.GetGridMax(OsV2{1, 1})}

	q.Size = div.data.ConvertMax(cell, q).Size
	q.Start = winRect.Start

	q = q.GetIntersect(winRect)
	return q
}

func (div *UiLayoutDiv) Maintenance() {

	div.use = false

	for i := len(div.childs) - 1; i >= 0; i-- {
		it := div.childs[i]
		if !it.use {
			it.Destroy()
			div.childs = append(div.childs[:i], div.childs[i+1:]...) //remove
		} else {
			it.Maintenance()
		}
	}
}

func (div *UiLayoutDiv) updateGridAndScroll(screen *OsV2, gridMax OsV2, makeSmallerX *bool, makeSmallerY *bool, win *Win) bool {

	// update cols/rows
	div.data.UpdateArray(win.Cell(), *screen, gridMax)

	// get max
	data := div.data.Convert(win.Cell(), OsV4{OsV2{}, gridMax}).Size

	// make canvas smaller
	hasScrollV := OsTrnBool(*makeSmallerX, data.Y > screen.Y, false)
	hasScrollH := OsTrnBool(*makeSmallerY, data.X > screen.X, false)
	if hasScrollV {
		screen.X -= div.data.scrollV._GetWidth(win)
		*makeSmallerX = false
	}
	if hasScrollH {
		screen.Y -= div.data.scrollH._GetWidth(win)
		*makeSmallerY = false
	}

	// save to scroll
	div.data.scrollV.data_height = data.Y
	div.data.scrollV.screen_height = screen.Y

	div.data.scrollH.data_height = data.X
	div.data.scrollH.screen_height = screen.X

	return hasScrollV || hasScrollH
}

func (div *UiLayoutDiv) UpdateGrid(win *Win) {

	makeSmallerX := div.data.scrollV.show
	makeSmallerY := div.data.scrollH.show
	gridMax := div.GetGridMax(OsV2{})

	screen := div.canvas.Size
	for div.updateGridAndScroll(&screen, gridMax, &makeSmallerX, &makeSmallerY, win) {
	}
}

func (div *UiLayoutDiv) UpdateCoord(rx, ry, rw, rh float64, win *Win) {

	parent := div.parent
	//backup := laypack.canvas

	//old_canvas := div.canvas
	//old_crop := div.crop

	if parent != nil {
		div.canvas = parent.data.Convert(win.Cell(), div.grid)
		div.canvas.Start = parent.canvas.Start.Add(div.canvas.Start)

		div.canvas.Start.X += int(float64(div.canvas.Size.X) * rx)
		div.canvas.Start.Y += int(float64(div.canvas.Size.Y) * ry)
		div.canvas.Size.X = int(float64(div.canvas.Size.X) * rw)
		div.canvas.Size.Y = int(float64(div.canvas.Size.Y) * rh)

		// move start by scroll
		div.canvas.Start.Y -= parent.data.scrollV.GetWheel()
		div.canvas.Start.X -= parent.data.scrollH.GetWheel()
	}

	if parent != nil {
		div.crop = div.canvas.GetIntersect(parent.crop)
	}

	// cut 'crop' by scrollbars space
	if div.data.scrollH.Is() {
		div.crop.Size.Y = OsMax(0, div.crop.Size.Y-div.data.scrollH._GetWidth(win))
	}
	if div.data.scrollV.Is() {
		div.crop.Size.X = OsMax(0, div.crop.Size.X-div.data.scrollV._GetWidth(win))
	}

	//if !old_canvas.Cmp(div.canvas) /*|| !old_crop.Cmp(div.crop)*/ {
	//	div.changed_size = true
	//}

	//div.changed_size = !old_canvas.Cmp(div.canvas) || !old_crop.Cmp(div.crop)

}

func (div *UiLayoutDiv) RenderResizeDraw(layoutScreen OsV4, i int, cd OsCd, vertical bool, buff *WinPaintBuff) {

	cell := buff.win.Cell()

	rspace := LayoutArray_resizerSize(cell)
	if vertical {
		layoutScreen.Start.X -= div.data.scrollH.GetWheel()

		layoutScreen.Start.X += div.data.cols.GetResizerPos(i, cell)
		layoutScreen.Size.X = rspace
	} else {
		layoutScreen.Start.Y -= div.data.scrollV.GetWheel()

		layoutScreen.Start.Y += div.data.rows.GetResizerPos(i, cell)
		layoutScreen.Size.Y = rspace
	}

	//fmt.Println(layoutScreen.Start.X, layoutScreen.Size.X, cd.R, cd.G, cd.B, cd.A, buff.noDraw, buff.crop.Start.X, buff.crop.Size.X)
	buff.AddRect(layoutScreen.AddSpace(4), cd, 0)
}

func (div *UiLayoutDiv) RenderResizeSpliter(ui *Ui) {
	enableInput := div.enableInput

	cell := ui.win.Cell()
	tpos := div.GetRelativePos(ui.win.io.touch.pos)

	vHighlight := false
	hHighlight := false
	col := -1
	row := -1
	if enableInput && div.crop.Inside(ui.win.io.touch.pos) {
		col = div.data.cols.IsResizerTouch((tpos.X), cell)
		row = div.data.rows.IsResizerTouch((tpos.Y), cell)

		vHighlight = (col >= 0)
		hHighlight = (row >= 0)

		// start
		if ui.win.io.touch.start && (vHighlight || hHighlight) {
			if vHighlight || hHighlight {
				ui.touch.Set(nil, nil, nil, div)
			}

			if vHighlight {
				div.touchResizeIndex = col
				div.touchResizeIsCol = true
			}
			if hHighlight {
				div.touchResizeIndex = row
				div.touchResizeIsCol = false
			}
		}

		if ui.touch.IsAnyActive() {
			vHighlight = false
			hHighlight = false
			//active = true
		}
	}

	// resize
	if ui.touch.IsFnMove(nil, nil, nil, div) {

		r := 1.0
		if div.touchResizeIsCol {
			col = div.touchResizeIndex
			vHighlight = true

			if div.data.cols.IsLastResizeValid() && int(col) == div.data.cols.NumInputs()-2 {
				r = float64(div.canvas.Size.X - tpos.X) // last
			} else {
				r = float64(tpos.X - div.data.cols.GetResizerPos(int(col)-1, cell))
			}

			div.SetResizer(int(col), r, true, ui.win)
		} else {
			row = div.touchResizeIndex
			hHighlight = true

			if div.data.rows.IsLastResizeValid() && int(row) == div.data.rows.NumInputs()-2 {
				r = float64(div.canvas.Size.Y - tpos.Y) // last
			} else {
				r = float64(tpos.Y - (div.data.rows.GetResizerPos(int(row)-1, cell)))
			}

			div.SetResizer(int(row), r, false, ui.win)
		}
	}

	// draw all(+active)
	{
		activeCd, _ := ui.win.io.GetPalette().GetCd(CdPalette_P, false, true, false, false)
		activeCd.A = 150

		defaultCd := ui.win.io.GetPalette().GetGrey(0.5)

		for i := 0; i < div.data.cols.NumInputs(); i++ {
			if div.data.cols.GetResizeIndex(i) >= 0 {
				if vHighlight && i == int(col) {
					div.RenderResizeDraw(div.canvas, i, activeCd, true, ui.buff)
				} else {
					div.RenderResizeDraw(div.canvas, i, defaultCd, true, ui.buff)
				}
			}
		}

		for i := 0; i < div.data.rows.NumInputs(); i++ {
			if div.data.rows.GetResizeIndex(i) >= 0 {
				if hHighlight && i == int(row) {
					div.RenderResizeDraw(div.canvas, i, activeCd, false, ui.buff)
				} else {
					div.RenderResizeDraw(div.canvas, i, defaultCd, false, ui.buff)
				}
			}
		}
	}

	// cursor
	if enableInput {
		if vHighlight {
			ui.win.PaintCursor("res_col")
		}
		if hHighlight {
			ui.win.PaintCursor("res_row")
		}
	}
}

func (div *UiLayoutDiv) Save() {
	div.data.Save()

	for _, it := range div.childs {
		it.Save()
	}
}

func (div *UiLayoutDiv) SetResizer(i int, value float64, isCol bool, win *Win) {
	value = math.Max(0.3, value/float64(win.Cell()))

	var ind int
	var arr *UiLayoutArray
	if isCol {
		ind = div.data.cols.GetResizeIndex(i)
		arr = &div.data.cols
	} else {
		ind = div.data.rows.GetResizeIndex(i)
		arr = &div.data.rows
	}

	if ind >= 0 {
		arr.inputs[ind].resize.value = float32(value)
	}
}

func (div *UiLayoutDiv) GetInputCol(pos int) *UiLayoutArrayItem {
	return div.data.cols.findOrAdd(pos)
}
func (div *UiLayoutDiv) GetInputRow(pos int) *UiLayoutArrayItem {
	return div.data.rows.findOrAdd(pos)
}

func (div *UiLayoutDiv) GetRelativePos(abs_pos OsV2) OsV2 {
	rpos := abs_pos.Sub(div.canvas.Start)
	rpos.Y += div.data.scrollV.GetWheel()
	rpos.X += div.data.scrollH.GetWheel()
	return rpos
}

func (div *UiLayoutDiv) FindBaseApp() *UiLayoutDiv {

	app := div.data.app
	ret := div
	for div != nil && div.data.app == app {
		ret = div //previous
		div = div.parent
	}
	return ret
}

func (div *UiLayoutDiv) FindHash(hash uint64) *UiLayoutDiv {
	if div.data.hash == hash {
		return div
	}

	for _, it := range div.childs {
		ret := it.FindHash(hash)
		if ret != nil {
			return ret
		}
	}
	return nil
}

func (div *UiLayoutDiv) FindUid(uid float64) *UiLayoutDiv {
	if uid == 0 {
		return div
	}
	base := div.FindBaseApp()
	return base.FindHash(math.Float64bits(uid))
}

func (div *UiLayoutDiv) GetCloseCell(touchPos OsV2) OsV4 {
	rpos := div.GetRelativePos(touchPos)

	var grid OsV4
	grid.Start.X = div.data.cols.GetCloseCell(rpos.X)
	grid.Start.Y = div.data.rows.GetCloseCell(rpos.Y)
	grid.Size.X = 1
	grid.Size.Y = 1
	return grid
}

func (div *UiLayoutDiv) IsOver(ui *Ui) bool {
	return div.enableInput && div.crop.Inside(ui.win.io.touch.pos) && !ui.touch.IsResizeActive()
}

func (div *UiLayoutDiv) IsOverScroll(ui *Ui) bool {
	insideScrollV, insideScrollH := ui._render_touchScrollEnabled(div)
	return div.enableInput && (insideScrollV || insideScrollH)
}

func (div *UiLayoutDiv) IsTouchActive(ui *Ui) bool {
	return ui.touch.IsFnMove(div, nil, nil, nil)
}

func (div *UiLayoutDiv) IsTouchInside(ui *Ui) bool {
	inside := div.IsOver(ui)

	if !div.IsTouchActive(ui) && div.enableInput && ui.touch.IsAnyActive() { // when click and move, other Buttons, etc. are disabled
		inside = false
	}

	return inside
}

func (div *UiLayoutDiv) IsTouchEnd(ui *Ui) bool {
	return div.enableInput && ui.win.io.touch.end && div.IsTouchActive(ui) //doesn't have to be inside!
}
