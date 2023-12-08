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
	"fmt"
	"strconv"
)

func (node *Node) nodeToPixelsCoord(ui *Ui, view *NodeView) OsV4 {

	coordS := view.nodeToPixels(node.Pos, ui) //.parent, because it has Cam

	w := float64(0)
	for _, attr := range node.Attrs {
		w = OsMaxFloat(w, ui.Paint_textWidth(attr.Name, -1, 0, "", true))
	}
	for _, in := range node.Inputs {
		w = OsMaxFloat(w, ui.Paint_textWidth(in.Name, -1, 0, "", true))
	}
	w += 3 //+value
	for _, out := range node.outputs {
		w = OsMaxFloat(w, ui.Paint_textWidth(out.Name, -1, 0, "", true))
	}

	w = OsMaxFloat(w, 5) //5 is minimum

	h := 1 + OsTrn(node.err != nil, 1, 0) + len(node.Attrs) + len(node.Inputs) + len(node.outputs)

	cellr := view.cellZoom(ui)
	cq := InitOsV4Mid(coordS, OsV2{int(w * float64(cellr)), h * cellr})
	cq.Size.Y += cellr / 5 //extra bottom space, because round connors
	return cq
}

func Node_getYellow() OsCd {
	return OsCd{204, 204, 0, 255} //...
}

func Node_drawComp(tp string, value *string, options string, enable bool, ui *Ui) {
	switch tp {
	case "combo":
		ui.Comp_combo(1, 0, 1, 1, value, options, "", enable, false)

	case "checkbox":
		//ui.Comp_checkbox()		//...
	case "switch":
		//ui.Comp_switch(1, 0, 1, 1, )		//...
	case "slider":
		//...
	default:
		_, _, _, _, div := ui.Comp_editbox(1, 0, 1, 1, value, 3, "", "", false, false, enable)

		if *value != "" {
			ui.Paint_tooltipDiv(div, 0, 0, 1, 1, *value)
		}

	}
}

func (node *Node) drawNode(someNodeIsDraged bool, ui *Ui, view *NodeView) (bool, bool) {

	lv := ui.GetCall()
	touch := &ui.buff.win.io.touch
	pl := ui.buff.win.io.GetPalette()

	coord := node.nodeToPixelsCoord(ui, view)

	roundc := 0.3

	bck := ui.win.io.ini.Dpi
	ui.win.io.ini.Dpi = int(float32(ui.win.io.ini.Dpi) * view.GetAttrCamZ())

	ui.Div_startCoord(0, 0, 1, 1, coord, strconv.Itoa(node.Id))
	innCoord := lv.call.canvas.AddSpaceX(ui.CellWidth(0.5))
	inside := coord.GetIntersect(lv.call.crop).Inside(touch.pos)
	ui.Div_end()

	//back
	{
		backCd := pl.GetGrey(1)
		labelCd := pl.OnP
		if node.Bypass {
			backCd = pl.GetGrey(0.5)
			labelCd.A = 100
		} else if node.err != nil {
			backCd = pl.E
			labelCd = pl.OnE
		}

		//shadow
		shadowCd := pl.GetGrey(0.4)
		ui.buff.AddRectRound(innCoord, ui.CellWidth(roundc), backCd, 0)
		ui.buff.AddRectRound(innCoord, ui.CellWidth(roundc), shadowCd, ui.CellWidth(0.03)) //smooth
	}

	insideMove := false

	ui.Div_startCoord(0, 0, 1, 1, coord, strconv.Itoa(node.Id))
	{
		ui.Div_col(0, 0.5)
		ui.Div_colMax(1, 100)
		ui.Div_col(2, 0.5)
		//ui.Div_rowMax(0, 100)

		y := 0

		//Function name + context menu
		{
			ui.Div_start(1, y, 1, 1)
			{
				ui.Div_colMax(0, 100)
				div := ui.Comp_textSelect(0, 0, 1, 1, node.FnName, 1, false) //cd ...
				if !insideMove {
					insideMove = div.crop.Inside(touch.pos)
				}

				//minimalize/maximalize ...

				//menu
				nm := fmt.Sprintf("node_menu_%d", node.Id)
				if ui.Comp_buttonIcon(1, 0, 1, 1, "file:apps/base/resources/context.png", 0.3, "", true) > 0 {
					ui.Dialog_open(nm, 1)
				}
				if ui.Dialog_start(nm) {
					//add comment / duplicate / delete ...
					ui.Dialog_end()
				}
			}
			ui.Div_end()
			y++
		}

		if node.err != nil {
			ui.Comp_text(1, y, 1, 1, "Error: "+node.err.Error(), 1) //cd(red) ...
			y++
		}

		//attrs
		for _, attr := range node.Attrs {

			//name + value
			ui.Div_start(1, y, 1, 1)
			{
				ui.Div_colMax(0, 2)
				ui.Div_colMax(1, 100)

				//name
				div := ui.Comp_textSelect(0, 0, 1, 1, attr.Name, 0, false)
				if !insideMove {
					insideMove = div.crop.Inside(touch.pos)
				}

				//value
				old_value := attr.Value
				Node_drawComp(attr.Gui_type, &attr.Value, attr.Gui_options, true, ui)
				if old_value != attr.Value {
					node.changed = true
				}

				attr.coordLabel = ui.GetCall().call.canvas
			}
			ui.Div_end()

			//dot
			ui.Div_start(2, y, 1, 1)
			{
				attr.coordDot = lv.call.canvas

				cd := Node_connectionCd(ui)
				if lv.call.data.over && attr.coordLabel.Inside(touch.pos) {
					cd = Node_getYellow()
				}
				ui.Paint_circle(0, 0, 1, 1, 0, 0.5, 0.5, 0.1, cd, 0)

			}
			ui.Div_end()

			y++
		}

		//inputs
		for _, in := range node.Inputs {

			//name + value
			ui.Div_start(1, y, 1, 1)
			{
				ui.Div_colMax(0, 2)
				ui.Div_colMax(1, 100)

				div := ui.Comp_textSelect(0, 0, 1, 1, in.Name, 0, false)
				if !insideMove {
					insideMove = div.crop.Inside(touch.pos)
				}

				out := in.FindWireOut()

				value := &in.Value
				if out != nil {
					value = &out.Value
				}

				old_value := *value
				Node_drawComp(in.Gui_type, value, in.Gui_options, out == nil, ui)
				if out == nil && old_value != *value {
					node.changed = true
				}

				in.coordLabel = ui.GetCall().call.canvas
			}
			ui.Div_end()

			//dot
			ui.Div_start(0, y, 1, 1)
			{
				in.coordDot = lv.call.canvas

				cd := Node_connectionCd(ui)
				if lv.call.data.over && in.coordLabel.Inside(touch.pos) {
					cd = Node_getYellow()
				}
				ui.Paint_circle(0, 0, 1, 1, 0, 0.5, 0.5, 0.1, cd, 0)

			}
			ui.Div_end()

			y++
		}

		//output names
		for _, out := range node.outputs {

			//name
			div := ui.Comp_textSelect(1, y, 1, 1, out.Name, 2, false)
			if !insideMove {
				insideMove = div.crop.Inside(touch.pos)
			}
			out.coordLabel = div.canvas

			//dot
			ui.Div_start(2, y, 1, 1)
			{
				cd := Node_connectionCd(ui)
				if lv.call.data.over {
					cd = Node_getYellow()
				}
				ui.Paint_circle(0, 0, 1, 1, 0, 0.5, 0.5, 0.1, cd, 0)
				out.coordDot = lv.call.canvas
			}
			ui.Div_end()

			y++
		}

	}

	selectCoord := innCoord.AddSpace(ui.CellWidth(-0.1))
	selectRad := ui.CellWidth(roundc * 1.3)

	ui.win.io.ini.Dpi = bck
	ui.Div_end()

	//select
	if (someNodeIsDraged && node.KeyProgessSelection(&ui.buff.win.io.keys)) || (!someNodeIsDraged && node.Selected) {
		ui.buff.AddRectRound(selectCoord, selectRad, Node_getYellow(), ui.CellWidth(0.06)) //selected
	}

	//go inside sub
	if node.IsGuiSub() && insideMove && touch.numClicks >= 2 {
		view.act = node
	}

	return inside, insideMove

}

func Node_connectionCd(ui *Ui) OsCd {
	pl := ui.buff.win.io.GetPalette()
	return pl.GetGrey(0.2)
}

func (node *Node) buildSubsList(listPathes *string, listNodes *[]*Node) {
	nm := node.getPath()
	if len(nm) > 2 {
		nm = nm[:len(nm)-1] //cut last '/'
	}

	*listPathes += nm + "|"
	*listNodes = append(*listNodes, node)

	for _, n := range node.Subs {
		if n.IsGuiSub() {
			n.buildSubsList(listPathes, listNodes)
		}
	}
}
