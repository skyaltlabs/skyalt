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

import "fmt"

func Node_connectionCd(ui *Ui) OsCd {
	pl := ui.win.io.GetPalette()
	return pl.GetGrey(0.2)
}

func (node *SANode) KeyProgessSelection(keys *WinKeys) bool {

	if keys.shift {
		if node.selected_cover {
			return true
		}
		return node.Selected
	} else if keys.ctrl {
		if node.selected_cover {
			return false
		}
		return node.Selected
	}

	return node.selected_cover
}

func (attr *SANodeAttr) IsVisible() bool {

	if attr.IsOutput() {
		return false //outputs has special place
	}

	if attr.Ui.Visible {
		return true
	}

	if attr.isRead {
		return true
	}

	for _, d := range attr.depends {
		if d.node != attr.node { //avoid self
			return true
		}
	}

	return false
}
func (node *SANode) NumVisibleAndCheck() int {
	n := 0
	for _, attr := range node.Attrs {
		if attr.IsVisible() || attr.IsOutput() || attr.isRead {
			//if i != n {
			//Div_DropMoveElement(&node.Attrs, &node.Attrs, i, n, SA_Drop_V_LEFT)	//reorder
			//}
			n++
		}

	}
	return n
}
func (node *SANode) VisiblePos(attr *SANodeAttr) int {

	y := 1
	for _, at := range node.Attrs {
		if at.IsVisible() {
			if attr == at {
				return y
			}
			y++
		}
	}

	for _, at := range node.Attrs {
		if at.IsOutput() {
			if attr == at {
				return y
			}
			y++
		}
	}
	return -1
}

func (w *SANode) cellZoom(ui *Ui) float32 {
	return float32(ui.win.Cell()) * float32(w.app.Cam_z) * 1
}

func (w *SANode) pixelsToNode(touchPos OsV2, ui *Ui, lvDiv *UiLayoutDiv) OsV2f {

	cell := ui.win.Cell()

	p := touchPos.Sub(lvDiv.canvas.Start).Sub(lvDiv.canvas.Size.MulV(0.5))

	var r OsV2f
	r.X = float32(p.X) / float32(w.app.Cam_z) / float32(cell)
	r.Y = float32(p.Y) / float32(w.app.Cam_z) / float32(cell)

	r.X += float32(w.app.Cam_x)
	r.Y += float32(w.app.Cam_y)

	return r
}

func (node *SANode) nodeToPixels(p OsV2f, canvas OsV4, ui *Ui) OsV2 {

	node = node.GetAbsoluteRoot()

	cell := ui.win.Cell()

	p.X -= float32(node.app.Cam_x)
	p.Y -= float32(node.app.Cam_y)

	var r OsV2
	r.X = int(p.X * float32(cell) * float32(node.app.Cam_z))
	r.Y = int(p.Y * float32(cell) * float32(node.app.Cam_z))

	return r.Add(canvas.Start).Add(canvas.Size.MulV(0.5))
}

func (node *SANode) nodeToPixelsCoord(canvas OsV4, ui *Ui) (OsV4, OsV4, OsV4) {
	var cq OsV4
	var cq_sel OsV4
	cellr := node.cellZoom(ui)

	mid := node.nodeToPixels(node.Pos, canvas, ui) //.parent, because it has Cam

	w := 4
	h := 1 + node.NumVisibleAndCheck()

	if SAGroups_HasNodeSub(node.Exe) {
		//compute bound
		bound := InitOsV4Mid(mid, OsV2{int(float32(w) * cellr), int(float32(h) * cellr)})
		for i, nd := range node.Subs {
			coord, _, _ := nd.nodeToPixelsCoord(canvas, ui)
			if i == 0 {
				bound = coord
			} else {
				bound = bound.Extend(coord)
			}
		}

		//add 1cell around and 1cell header
		header_h := int(1 * cellr)
		cq = bound.AddSpace(int(-1.0 * float64(cellr)))
		cq = InitOsV4(cq.Start.X, cq.Start.Y-header_h, cq.Size.X, cq.Size.Y+header_h)

		cq_sel = cq
		cq_sel.Size.Y = header_h
	} else {
		cq = InitOsV4Mid(mid, OsV2{int(float32(w) * cellr), int(float32(h) * cellr)})
		cq_sel = cq //same
	}

	return cq, cq.AddSpace(int(-0.15 * float64(cellr))), cq_sel
}

func (node *SANode) FindInsideParent(touchPos OsV2, canvas OsV4, ui *Ui) *SANode {
	var found *SANode

	if SAGroups_HasNodeSub(node.Exe) {
		coord, _, _ := node.nodeToPixelsCoord(canvas, ui)
		if coord.Inside(touchPos) {
			found = node
		}
	}

	for _, nd := range node.Subs {
		ff := nd.FindInsideParent(touchPos, canvas, ui)
		if ff != nil {
			found = ff
		}
	}

	return found
}

func (node *SANode) drawRectNode(someNodeIsDraged bool, app *SAApp) bool {
	ui := app.base.ui
	lv := ui.GetCall()
	touch := &ui.win.io.touch
	pl := ui.win.io.GetPalette()
	roundc := 0.2

	coord, selCoord, headerCoord := node.nodeToPixelsCoord(lv.call.canvas, ui)

	bck := ui.win.io.ini.Dpi
	ui.win.io.ini.Dpi = int(float32(ui.win.io.ini.Dpi) * float32(node.app.Cam_z))

	backCd := pl.GetGrey(1)

	backCd.A = 255
	ui.buff.AddRectRound(headerCoord, ui.CellWidth(roundc), backCd, 0)

	inside := false
	ui.Div_startCoord(0, 0, 1, 1, headerCoord, node.Name)
	{
		ui.DivInfo_set(SA_DIV_SET_scrollHshow, 0, 0)
		ui.DivInfo_set(SA_DIV_SET_scrollVshow, 0, 0)
		ui.Div_colMax(0, 1)
		ui.Div_colMax(1, 100)
		ui.Comp_textSelect(0, 0, 2, 1, node.Name, OsV2{1, 1}, false, false)
		inside = headerCoord.GetIntersect(lv.call.crop).Inside(touch.pos)
	}
	ui.Div_end()

	backCd.A = 50
	ui.buff.AddRectRound(coord, ui.CellWidth(roundc), backCd, 0)

	backCd.A = 255
	ui.buff.AddRectRound(coord, ui.CellWidth(roundc), backCd, ui.CellWidth((0.03)))

	//select rect
	selectRad := ui.CellWidth(roundc * 1.3)
	if (someNodeIsDraged && node.KeyProgessSelection(&ui.win.io.keys)) || (!someNodeIsDraged && node.Selected) {
		ui.buff.AddRectRound(selCoord, selectRad, SAApp_getYellow(), ui.CellWidth(0.06)) //selected
	}

	ui.win.io.ini.Dpi = bck

	return inside
}

func (node *SANode) drawNode(someNodeIsDraged bool, app *SAApp) bool {

	ui := app.base.ui
	lv := ui.GetCall()
	touch := &ui.win.io.touch
	pl := ui.win.io.GetPalette()
	roundc := 0.2

	coord, selCoord, _ := node.nodeToPixelsCoord(lv.call.canvas, ui)

	bck := ui.win.io.ini.Dpi
	ui.win.io.ini.Dpi = int(float32(ui.win.io.ini.Dpi) * float32(node.app.Cam_z))

	inside := false

	//back
	{
		backCd := pl.GetGrey(1)

		if node.CanBeRenderOnCanvas() {
			backCd = pl.P //InitOsCd32(50, 50, 180, 255)
			backCd.A = 150
		}

		if node.HasError() {
			backCd = pl.E
		}

		ui.buff.AddRectRound(coord, ui.CellWidth(roundc), backCd, 0)

		//shadow
		ui.buff.AddRectRound(coord, ui.CellWidth(roundc), pl.GetGrey(0.4), ui.CellWidth(0.03)) //smooth
	}

	ui.Div_startCoord(0, 0, 1, 1, coord, node.Name)
	{
		ui.DivInfo_set(SA_DIV_SET_scrollHshow, 0, 0)
		ui.DivInfo_set(SA_DIV_SET_scrollVshow, 0, 0)

		ui.Div_colMax(0, 100)

		//name
		div := ui.Comp_textSelect(0, 0, 1, 1, node.Name, OsV2{1, 1}, false, false) //double click? ........
		inside = div.crop.Inside(touch.pos)
		//inside = lv.call.crop.Inside(touch.pos)

		//open settings
		if ui.Comp_buttonIcon(1, 0, 1, 1, InitWinMedia_url("file:apps/base/resources/fullscreen_mode.png"), 0.3, "Show everything", CdPalette_B, true, false) > 0 {
			node.SelectOnlyThis()
			ui.Dialog_open("attributes", 0)
			inside = false
		}

		//attributes
		y := 1
		for _, attr := range node.Attrs {
			if attr.IsVisible() {
				ui.Comp_textSelect(0, y, 1, 1, attr.Name, OsV2{0, 1}, false, false) //center
				y++
			}
		}
		for _, attr := range node.Attrs {
			if attr.IsOutput() {
				ui.Comp_textSelect(0, y, 2, 1, attr.Name, OsV2{2, 1}, false, false) //right
				y++
			}
		}

		//nepotřebuji visibility - zobrazuji jenom jména(aka data-flow) ............
		//chci vytvořit spojení a není viditelný - dolů dát extra connect, který zobrazí dialog se všemy možnostmi ........

	}
	ui.Div_end()

	//draw progress text
	isJobActive := node.progress > 0
	if isJobActive {
		if node.progress > 0 {
			cellr := node.cellZoom(ui)
			cq := coord.AddSpace(int(-0.3 * float64(cellr)))
			cq.Start.Y -= int(cellr) //cq.End().X
			cq.Size.Y = int(cellr)
			ui.Div_startCoord(0, 0, 1, 1, cq, node.Name)
			{
				ui.Div_colMax(0, 100)
				ui.Div_rowMax(0, 100)
				str := fmt.Sprintf("%.0f%%", node.progress*100)
				if node.progress_desc != "" {
					str += "(" + node.progress_desc + ")"
				}
				ui.Comp_textSelect(0, 0, 1, 1, str, OsV2{0, 1}, false, false)
			}
			ui.Div_end()
		}
	}

	//select rect
	selectRad := ui.CellWidth(roundc * 1.3)
	if (someNodeIsDraged && node.KeyProgessSelection(&ui.win.io.keys)) || (!someNodeIsDraged && node.Selected) {
		ui.buff.AddRectRound(selCoord, selectRad, SAApp_getYellow(), ui.CellWidth(0.06)) //selected
	}

	//exe rect
	selectRad = ui.CellWidth(roundc * 1.7)
	if isJobActive {
		pl := ui.win.io.GetPalette()
		cd := pl.P

		cellr := node.cellZoom(ui)
		cq := coord.AddSpace(int(-0.3 * float64(cellr)))

		ui.buff.AddRectRound(cq, selectRad, cd, ui.CellWidth(0.06)) //selected
	}

	ui.win.io.ini.Dpi = bck

	if ui.Dialog_start("attributes") {
		sel := app.root.FindSelected()
		if sel != nil {
			ui.Div_colMax(0, 20)
			ui.Div_rowMax(0, 20)
			ui.Div_start(0, 0, 1, 1)
			sel.RenderAttrs()
			ui.Div_end()
		} else {
			ui.Dialog_close()
		}

		ui.Dialog_end()
	}

	return inside
}
