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

func (w *SANode) pixelsToNode(touchPos OsV2, ui *Ui, lvDiv *UiLayoutDiv) OsV2f {

	cell := ui.win.Cell()

	p := touchPos.Sub(lvDiv.canvas.Start).Sub(lvDiv.canvas.Size.MulV(0.5))

	var r OsV2f
	r.X = float32(p.X) / float32(w.Cam_z) / float32(cell)
	r.Y = float32(p.Y) / float32(w.Cam_z) / float32(cell)

	r.X += float32(w.Cam_x)
	r.Y += float32(w.Cam_y)

	return r
}

func (w *SANode) nodeToPixels(p OsV2f, canvas OsV4, ui *Ui) OsV2 {

	cell := ui.win.Cell()

	p.X -= float32(w.Cam_x)
	p.Y -= float32(w.Cam_y)

	var r OsV2
	r.X = int(p.X * float32(cell) * float32(w.Cam_z))
	r.Y = int(p.Y * float32(cell) * float32(w.Cam_z))

	return r.Add(canvas.Start).Add(canvas.Size.MulV(0.5))
}

func (w *SANode) cellZoom(ui *Ui) float32 {
	return float32(ui.win.Cell()) * float32(w.Cam_z) * 1
}

func (node *SANode) nodeToPixelsCoord(canvas OsV4, ui *Ui) (OsV4, OsV4) {

	coord := node.parent.nodeToPixels(node.Pos, canvas, ui) //.parent, because it has Cam

	w := 4
	h := 1

	cellr := node.parent.cellZoom(ui)
	cq := InitOsV4Mid(coord, OsV2{int(float32(w) * cellr), int(float32(h) * cellr)})

	return cq, cq.AddSpace(int(-0.15 * float64(cellr)))
}

func (node *SANode) drawNode(someNodeIsDraged bool, app *SAApp) bool {

	ui := app.base.ui

	lv := ui.GetCall()
	touch := &ui.win.io.touch
	pl := ui.win.io.GetPalette()

	coord, selCoord := node.nodeToPixelsCoord(lv.call.canvas, ui)

	roundc := 0.2

	bck := ui.win.io.ini.Dpi
	ui.win.io.ini.Dpi = int(float32(ui.win.io.ini.Dpi) * float32(node.parent.Cam_z))

	ui.Div_startCoord(0, 0, 1, 1, coord, node.Name)
	ui.DivInfo_set(SA_DIV_SET_scrollHshow, 0, 0)
	ui.DivInfo_set(SA_DIV_SET_scrollVshow, 0, 0)
	inside := coord.GetIntersect(lv.call.crop).Inside(touch.pos)
	ui.Div_end()

	//back
	{
		if node.state.Load() == SANode_STATE_DONE {
			backCd := pl.GetGrey(1)

			if node.CanBeRenderOnCanvas() {
				backCd = pl.P //InitOsCd32(50, 50, 180, 255)
				backCd.A = 150
			}

			if node.HasError() {
				backCd = pl.E
			}
			if node.Bypass {
				backCd.A = 20
			}

			ui.buff.AddRectRound(coord, ui.CellWidth(roundc), backCd, 0)
		} else {
			cq := coord
			cq.Size.X = int(float64(cq.Size.X) * OsClampFloat(node.progress, 0, 1))
			ui.buff.AddRectRound(cq, ui.CellWidth(roundc), pl.S, 0)
		}

		//shadow
		shadowCd := pl.GetGrey(0.4)
		if node.Bypass {
			shadowCd.A = 20
		}
		ui.buff.AddRectRound(coord, ui.CellWidth(roundc), shadowCd, ui.CellWidth(0.03)) //smooth
	}

	ui.Div_startCoord(0, 0, 1, 1, coord, node.Name)
	{
		ui.Div_colMax(0, 1)
		ui.Div_colMax(1, 100)

		nm := node.Name
		if node.state.Load() != SANode_STATE_DONE {
			nm += fmt.Sprintf("(%.0f%%)", node.progress*100)

			ui.Paint_tooltip(0, 0, 1, 1, node.progress_desc)
		} else {
			ui.Paint_tooltip(0, 0, 1, 1, fmt.Sprintf("Type: %s, Executed in %.3f", node.Exe, node.exeTimeSec))
		}

		//ui.Comp_image(0, 0, 1, 1, node.app.base.node_groups.FindNodeGroupIcon(node.Exe), InitOsCdWhite(), 0.3, 1, 1, false)	//icon
		ui.Comp_textSelect(0, 0, 2, 1, nm, OsV2{1, 1}, false, false)
	}
	ui.Div_end()

	selectRad := ui.CellWidth(roundc * 1.3)
	ui.win.io.ini.Dpi = bck

	//select
	if (someNodeIsDraged && node.KeyProgessSelection(&ui.win.io.keys)) || (!someNodeIsDraged && node.Selected) {
		ui.buff.AddRectRound(selCoord, selectRad, SAApp_getYellow(), ui.CellWidth(0.06)) //selected
	}

	//go inside sub
	if node.IsGuiLayout() && inside && touch.end && touch.numClicks > 1 {
		app.act = node
	}

	return inside
}
