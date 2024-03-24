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

func Node_connectionCd(active bool, ui *Ui) OsCd {
	pl := ui.win.io.GetPalette()

	cd := pl.GetGrey(0.2)

	if active {
		cd = SAApp_getYellow()
	}

	return cd
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

func (w *SANode) cellZoom(ui *Ui) float32 {
	return float32(ui.win.Cell()) * float32(w.app.Cam_z) * 1
}

func (node *SANode) pixelsToNode(touchPos OsV2, lvDiv *UiLayoutDiv) OsV2f {

	ui := node.app.base.ui
	cell := ui.win.Cell()

	p := touchPos.Sub(lvDiv.canvas.Start).Sub(lvDiv.canvas.Size.MulV(0.5))

	var r OsV2f
	r.X = float32(p.X) / float32(node.app.Cam_z) / float32(cell)
	r.Y = float32(p.Y) / float32(node.app.Cam_z) / float32(cell)

	r.X += float32(node.app.Cam_x)
	r.Y += float32(node.app.Cam_y)

	return r
}

func (node *SANode) nodeToPixels(p OsV2f, canvas OsV4) OsV2 {
	ui := node.app.base.ui

	node = node.GetAbsoluteRoot()

	cell := ui.win.Cell()

	p.X -= float32(node.app.Cam_x)
	p.Y -= float32(node.app.Cam_y)

	var r OsV2
	r.X = int(p.X * float32(cell) * float32(node.app.Cam_z))
	r.Y = int(p.Y * float32(cell) * float32(node.app.Cam_z))

	return r.Add(canvas.Start).Add(canvas.Size.MulV(0.5))
}

func (node *SANode) GetNodeLabel() string {
	return node.Name + "(" + node.Exe + ")"
}

func (node *SANode) nodeToPixelsCoord(canvas OsV4) (OsV4, OsV4, OsV4) {
	ui := node.app.base.ui

	var cq OsV4
	var cq_sel OsV4
	cellr := node.cellZoom(ui)

	mid := node.nodeToPixels(node.Pos, canvas) //.parent, because it has Cam

	w := float32(ui.win.GetTextSize(-1, node.GetNodeLabel(), InitWinFontPropsDef(ui.win)).X) / float32(ui.win.Cell())
	w = OsMaxFloat32(w+1, 4) //1=extra space, minimum 4
	if node.IsTypeCode() {
		w += 1 //chat icon
	}
	h := float32(1) //+ node.NumVisibleAndCheck()

	if node.HasNodeSubs() {
		//compute bound
		bound := InitOsV4Mid(mid, OsV2{int(w * cellr), int(h * cellr)})
		for i, nd := range node.Subs {
			coord, _, _ := nd.nodeToPixelsCoord(canvas)
			if i == 0 {
				bound = coord
			} else {
				bound = bound.Extend(coord)
			}
		}

		//add 1cell around and 1cell header
		header_h := int(float32(h) * cellr)
		cq = bound.AddSpace(int(-1.0 * float64(cellr)))
		cq = InitOsV4(cq.Start.X, cq.Start.Y-header_h, cq.Size.X, cq.Size.Y+header_h)

		cq_sel = cq
		cq_sel.Size.Y = header_h
	} else {
		cq = InitOsV4Mid(mid, OsV2{int(float32(w) * cellr), int(float32(h) * cellr)})
		cq_sel = cq //same
	}

	return cq, cq.AddSpace(int(-0.25 * float64(cellr))), cq_sel
}

func (node *SANode) FindInsideParent(touchPos OsV2, canvas OsV4) *SANode {
	var found *SANode

	if node.HasNodeSubs() {
		coord, _, _ := node.nodeToPixelsCoord(canvas)
		if coord.Inside(touchPos) {
			found = node
		}
	}

	for _, nd := range node.Subs {
		ff := nd.FindInsideParent(touchPos, canvas)
		if ff != nil {
			found = ff
		}
	}

	return found
}

func (node *SANode) drawShadow(coord OsV4, roundc float64) {
	ui := node.app.base.ui

	rc := ui.CellWidth(roundc)
	sh := coord
	sh = sh.AddSpace((-rc * 1))
	sh.Start = sh.Start.Add(OsV2{rc / 3, rc / 3})
	ui.buff.AddRectRoundGrad(sh, rc*3, InitOsCdBlack().SetAlpha(100), 0) //smooth
}

func (node *SANode) drawHeader() bool {
	ui := node.app.base.ui

	ui.DivInfo_set(SA_DIV_SET_scrollHshow, 0, 0)
	ui.DivInfo_set(SA_DIV_SET_scrollVshow, 0, 0)

	ui.Div_col(0, 0.5)
	ui.Div_colMax(1, 100)
	ui.Div_col(2, 0.5)
	if node.IsTypeCode() {
		//chat icon
		ui.Div_col(2, 1)
		ui.Div_col(3, 0.5)

	} else {
		ui.Div_col(2, 0.5)
	}

	inside := false

	ui.Div_start(1, 0, 1, 1)
	{
		ui.Div_colMax(0, 100)

		//name
		div := ui.Comp_textSelect(0, 0, 1, 1, node.GetNodeLabel(), OsV2{1, 1}, false, false)
		if div.IsTouchEndSubs(ui) && ui.win.io.touch.rm {
			ui.win.io.keys.clipboard = "'" + node.GetPathSplit() + "'"
		}

		//ui.Paint_tooltip(0, 0, 1, 1, "Type: "+node.Exe)
	}
	ui.Div_end()

	//chat icon
	if node.IsTypeCode() {
		cd := CdPalette_B
		if node.ShowCodeChat {
			cd = CdPalette_P
		}
		if ui.Comp_buttonIcon(2, 0, 1, 1, InitWinMedia_url("file:apps/base/resources/chat.png"), 0.2, "Code Chat", cd, true, false) > 0 {
			node.ShowCodeChat = !node.ShowCodeChat
		}
	}

	//attributes
	/*circleCd := CdPalette_S
	if node.app.base.node_groups.IsUI(node.Exe) {
		circleCd = CdPalette_P
	}

	//make connection - dialog attr list
	{
		if node.IsTypeCode() {
			if ui.Comp_buttonCircle(0, 0, 1, 1, "", "", CdPalette_B, circleCd, node.app.graph.connect_in == nil) > 0 {
				node.app.graph.SetConnectIn(node)
			}
		}

		if node.IsTypeTrigger() {
			if ui.Comp_buttonCircle(2, 0, 1, 1, "", "", CdPalette_B, circleCd, node.app.graph.connect_out == nil) > 0 {
				node.app.graph.SetConnectOut(node)
			}
		}
	}*/

	//inside & double-click
	if ui.GetCall().call.enableInput {
		inside = ui.GetCall().call.crop.Inside(ui.win.io.touch.pos)
		/*if inside && ui.win.io.touch.end && ui.win.io.touch.numClicks >= 2 && !node.app.graph.isConnecting() {
			node.SelectOnlyThis()
			ui.Dialog_open("attributes", 0)
			inside = false
		}*/
	}

	return inside
}
func (node *SANode) drawRectNode(someNodeIsDraged bool, app *SAApp) bool {
	ui := app.base.ui
	lv := ui.GetCall()
	pl := ui.win.io.GetPalette()
	roundc := 0.2

	coord, selCoord, headerCoord := node.nodeToPixelsCoord(lv.call.canvas)

	bck := ui.win.io.ini.Dpi
	ui.win.io.ini.Dpi = int(float32(ui.win.io.ini.Dpi) * float32(node.app.Cam_z))

	//shadow
	node.drawShadow(coord, roundc)

	//background
	backCd := pl.GetGrey(0.8)
	ui.buff.AddRectRound(coord, ui.CellWidth(roundc), backCd, 0)

	//header background
	backkCd := pl.GetGrey(1)
	ui.buff.AddRectRound(headerCoord, ui.CellWidth(roundc), backkCd, 0)

	//border
	ui.buff.AddRectRound(coord, ui.CellWidth(roundc), backkCd, ui.CellWidth((0.03)))

	//header
	ui.Div_startCoord(0, 0, 1, 1, headerCoord.AddSpaceX(-ui.CellWidth(0.25)), node.Name)
	inside := node.drawHeader()
	ui.Div_end()

	//select rect
	selectRad := ui.CellWidth(roundc * 1.3)
	if (someNodeIsDraged && node.KeyProgessSelection(&ui.win.io.keys)) || (!someNodeIsDraged && node.Selected) {
		ui.buff.AddRectRound(selCoord, selectRad, SAApp_getYellow(), ui.CellWidth(0.06)) //selected
	}

	ui.win.io.ini.Dpi = bck

	return inside
}

func (node *SANode) drawNode(someNodeIsDraged bool) bool {

	ui := node.app.base.ui
	lv := ui.GetCall()
	pl := ui.win.io.GetPalette()
	roundc := 0.2

	coord, selCoord, _ := node.nodeToPixelsCoord(lv.call.canvas)

	bck := ui.win.io.ini.Dpi
	ui.win.io.ini.Dpi = int(float32(ui.win.io.ini.Dpi) * float32(node.app.Cam_z))

	//back
	{
		backCd := pl.GetGrey(1) //code

		if node.CanBeRenderOnCanvas() {
			backCd = pl.P.Aprox(InitOsCdWhite(), 0.3)
		} else if !node.IsTypeCode() {
			backCd = pl.S.Aprox(InitOsCdWhite(), 0.3)
		}

		if node.HasError() {
			backCd = pl.OnE
		}

		// shadow
		node.drawShadow(coord, roundc)

		//background
		ui.buff.AddRectRound(coord, ui.CellWidth(roundc), backCd, 0)
	}

	ui.Div_startCoord(0, 0, 1, 1, coord.AddSpaceX(-ui.CellWidth(0.25)), node.Name)
	inside := node.drawHeader()
	ui.Div_end()

	//show bypass
	if node.IsBypassed() {
		ui.buff.AddRectRound(coord, ui.CellWidth(roundc), InitOsCdWhite().SetAlpha(150), 0)
	}

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

	return inside
}
