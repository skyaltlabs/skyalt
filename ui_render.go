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
	"fmt"
)

func (ui *Ui) _render_touchScrollEnabled(packLayout *UiLayoutDiv) (bool, bool) {

	//root := app.root

	insideScrollV := false
	insideScrollH := false
	if packLayout.data.scrollV.Is() {
		scrollQuad := packLayout.data.scrollV.GetScrollBackCoordV(packLayout.crop, ui.win)
		insideScrollV = scrollQuad.Inside(ui.win.io.touch.pos)
	}

	if packLayout.data.scrollH.Is() {
		scrollQuad := packLayout.data.scrollH.GetScrollBackCoordH(packLayout.crop, ui.win)
		insideScrollH = scrollQuad.Inside(ui.win.io.touch.pos)
	}
	return insideScrollV, insideScrollH
}

func (ui *Ui) _render_touchScroll(packLayout *UiLayoutDiv, enableInput bool) {

	hasScrollV := packLayout.data.scrollV.IsPure()
	hasScrollH := packLayout.data.scrollH.IsPure()

	if hasScrollV {
		if enableInput {
			packLayout.data.scrollV.TouchV(packLayout, ui)
		}
	}

	if hasScrollH {
		if enableInput {
			packLayout.data.scrollH.TouchH(hasScrollV, packLayout, ui)
		}
	}
}

func (ui *Ui) _renderScroll(packLayout *UiLayoutDiv, showBackground bool) {

	if packLayout.data.scrollV.Is() {
		scrollQuad := packLayout.data.scrollV.GetScrollBackCoordV(packLayout.crop, ui.win)
		packLayout.data.scrollV.DrawV(scrollQuad, showBackground, ui.buff)
	}

	if packLayout.data.scrollH.Is() {
		scrollQuad := packLayout.data.scrollH.GetScrollBackCoordH(packLayout.crop, ui.win)
		packLayout.data.scrollH.DrawH(scrollQuad, showBackground, ui.buff)
	}
}

func (ui *Ui) renderStart(rx, ry, rw, rh float64, drawBack bool) {

	lv := ui.GetCall()

	//st.stack.changed_size = false
	//st.stack.changed_touch = false
	//st.stack.changed_key = false

	lv.call.data.Reset() //here because after *dialog* needs to know old size
	lv.call.UpdateCoord(rx, ry, rw, rh, ui.win)

	enableInput := lv.call.data.touch_enabled
	if lv.call.parent == nil {
		enableInput = ui.IsStackTop()
	} else {
		enableInput = enableInput && lv.call.parent.enableInput
	}
	ui._render_touchScroll(lv.call, enableInput)

	// scroll touch
	insideScrollV, insideScrollH := ui._render_touchScrollEnabled(lv.call)
	overScroll := enableInput && (insideScrollV || insideScrollH)
	enableInput = enableInput && !insideScrollV && !insideScrollH //can NOT click through

	startTouch := enableInput && ui.win.io.touch.start && !ui.win.io.keys.alt
	endTouch := enableInput && ui.win.io.touch.end
	over := enableInput && lv.call.crop.Inside(ui.win.io.touch.pos)
	inside := over
	if inside && startTouch && enableInput {
		if !ui.touch.IsScrollOrResizeActive() { //if lower resize or scroll is activated than don't rewrite it with higher canvas
			ui.touch.Set(lv.call, nil, nil, nil)
		}
	}
	touchActiveMove := ui.touch.IsFnMove(lv.call, nil, nil, nil)

	if !touchActiveMove && enableInput && ui.touch.IsAnyActive() { // when click and move, other Buttons, etc. are disabled
		inside = false
	}

	touch_end := (endTouch && touchActiveMove)

	lv.call.enableInput = enableInput

	//jump between inside/outside or action/noaction
	//st.stack.changed_touch = (st.stack.data.over != over || st.stack.data.overScroll != overScroll || st.stack.data.touch_inside != inside || st.stack.data.touch_active != touchActiveMove || st.stack.data.touch_end != touch_end)

	lv.call.data.over = over
	lv.call.data.overScroll = overScroll
	lv.call.data.touch_inside = inside
	lv.call.data.touch_active = touchActiveMove
	lv.call.data.touch_end = touch_end //&& inside

	//there is action inside
	/*if st.stack.data.over || st.stack.data.overScroll || st.stack.data.touch_inside || st.stack.data.touch_active || st.stack.data.touch_end {
		st.stack.changed_touch = true
	}*/

	/*if st.stack.changed_touch {
		p := st.stack.parent
		for p != nil {
			p.changed_touch = true
			p = p.parent
		}
	}*/

	//root.buff.Reset(st.stack.crop, root, drawBack)
	ui.buff.AddCrop(lv.call.crop)

	if ui.win.io.ini.Grid {
		ui.renderGrid()
	}

	lv.call.data.touch_enabled = true //reset for next tick
}

func (ui *Ui) renderGrid() {

	lv := ui.GetCall()

	start := lv.call.canvas.Start
	size := lv.call.canvas.Size

	cd := OsCd{200, 100, 80, 200}

	//cols
	px := start.X
	for _, col := range lv.call.data.cols.outputs {
		px += int(col)
		ui.buff.AddLine(OsV2{px, start.Y}, OsV2{px, start.Y + size.Y}, cd, ui.CellWidth(0.03))
	}

	//rows
	py := start.Y
	for _, row := range lv.call.data.rows.outputs {
		py += int(row)
		ui.buff.AddLine(OsV2{start.X, py}, OsV2{start.X + size.X, py}, cd, ui.CellWidth(0.03))
	}

	px = start.X
	for x, col := range lv.call.data.cols.outputs {

		py = start.Y
		for y, row := range lv.call.data.rows.outputs {
			ui.buff.AddText(fmt.Sprintf("[%d, %d]", x, y), OsV4{OsV2{px, py}, OsV2{int(col), int(row)}}, ui.win.fonts.Get(SKYALT_FONT_PATH), cd, ui.win.io.GetDPI()/8, OsV2{1, 1}, nil, true)
			py += int(row)
		}

		px += int(col)
	}

}

func (ui *Ui) renderEnd(baseDiv bool) {

	lv := ui.GetCall()

	lv.call.gridLock = false

	// show scroll
	ui.buff.AddCrop(lv.call.CropWithScroll(ui.win))
	ui._renderScroll(lv.call, lv.call.data.scrollOnScreen)

	if lv.call.parent != nil {
		lv.call = lv.call.parent
		ui.buff.AddCrop(lv.call.crop)
	} else {
		if !baseDiv /*&& (app.debug == nil || app.debug.conn != nil)*/ {
			lv.call.data.app.AddLogErr(errors.New("div pair corrupted2. Check if every SA_DivStart() has SA_DivEnd(). Check return/continue/break in the middle"))
		}
	}
}
