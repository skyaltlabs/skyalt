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

import "strings"

type SAGraph struct {
	app *SAApp

	cam_move           bool
	node_move          bool
	node_select        bool
	cam_start          OsV2f
	touch_start        OsV2
	node_move_selected *SAWidget
}

func NewSAGraph(app *SAApp) *SAGraph {
	var gr SAGraph
	gr.app = app
	return &gr
}

func (gr *SAGraph) drawCreateNode(ui *Ui) {

	lvBaseDiv := ui.GetCall().call

	if ui.win.io.keys.tab && lvBaseDiv.IsOver(ui) {
		gr.app.canvas.addWidgetGrid = InitOsV4(0, 0, 1, 1)
		gr.app.canvas.addWidgetPos = gr.app.act.pixelsToNode(ui.buff.win.io.touch.pos, ui, lvBaseDiv)
		ui.Dialog_open("nodes_list", 2)
	}
}

func _SAGraph_drawConnection(start OsV2, end OsV2, active bool, ui *Ui) {
	cd := Node_connectionCd(ui)
	if active {
		cd = SAApp_getYellow()
	}

	mid := start.Aprox(end, 0.5)
	//ui.buff.AddBezier(start, OsV2{mid.X, start.Y}, OsV2{mid.X, end.Y}, end, cd, ui.CellWidth(0.03), false)
	ui.buff.AddBezier(start, OsV2{start.X, mid.Y}, OsV2{end.X, mid.Y}, end, cd, ui.CellWidth(0.03), false)
	//ui.buff.AddLine(start, end, cd, ui.CellWidth(0.03))

	//t := sp / 5
	//ui.buff.AddTringle(end, end.Add(OsV2{-t, t}), end.Add(OsV2{t, t}), pl.GetGrey(0.5)) //arrow
}

func (gr *SAGraph) reorder(onlySelected bool, ui *Ui) {

	//create list
	var nodes []*SAWidget
	for _, n := range gr.app.act.Subs {
		if onlySelected && !n.Selected {
			continue //skip
		}
		nodes = append(nodes, n)
	}

	//...

}

func (gr *SAGraph) autoZoom(onlySelected bool, ui *Ui) {
	//zoom to all
	first := true
	var mn OsV2f
	var mx OsV2f
	for _, n := range gr.app.act.Subs {
		if onlySelected && !n.Selected {
			continue //skip
		}

		if first {
			mn = n.Pos
			mx = n.Pos
			first = false
		}
		if n.Pos.X < mn.X {
			mn.X = n.Pos.X
		}
		if n.Pos.Y < mn.Y {
			mn.Y = n.Pos.Y
		}

		if n.Pos.X > mx.X {
			mx.X = n.Pos.X
		}
		if n.Pos.Y > mx.Y {
			mx.Y = n.Pos.Y
		}
	}
	gr.app.act.Cam_x = float64(mn.X+mx.X) / 2
	gr.app.act.Cam_y = float64(mn.Y+mx.Y) / 2

	canvas := ui.GetCall().call.canvas
	gr.app.act.Cam_z = 1
	for {
		areIn := true
		for _, n := range gr.app.act.Subs {
			if onlySelected && !n.Selected {
				continue //skip
			}

			_, cq := n.nodeToPixelsCoord(ui)
			if !canvas.GetIntersect(cq).Cmp(cq) {
				areIn = false
				break
			}
		}
		if areIn {
			break
		}
		gr.app.act.Cam_z -= 0.05
	}
}

func (gr *SAGraph) drawGraph(ui *Ui) {
	pl := ui.buff.win.io.GetPalette()

	ui.Div_colMax(0, 100)
	ui.Div_rowMax(0, 100)

	ui.Paint_rect(0, 0, 1, 1, 0, pl.GetGrey(0.8), 0)

	//fade "press tab" centered in background
	lv := ui.GetCall()
	ui._compDrawText(lv.call.canvas, "press tab", "", pl.GetGrey(1), SKYALT_FONT_HEIGHT, false, false, 1, 1, true)

	//+
	gr.drawCreateNode(ui)

	//draw connections
	for _, node := range gr.app.act.Subs {
		for _, in := range node.Attrs {
			for _, out := range in.depends {
				coordOut, selCoordOut := out.widget.nodeToPixelsCoord(ui)
				coordIn, selCoordIn := in.widget.nodeToPixelsCoord(ui)

				if out.widget.Selected {
					coordOut = selCoordOut
				}
				if in.widget.Selected {
					coordIn = selCoordIn
				}
				_SAGraph_drawConnection(OsV2{coordOut.Middle().X, coordOut.End().Y}, OsV2{coordIn.Middle().X, coordIn.Start.Y}, node.Selected || out.widget.Selected, ui)
			}
		}
	}

	//draw node bodies
	var touchInsideNode *SAWidget
	for _, n := range gr.app.act.Subs {
		inside := n.drawNode(gr.node_select, gr.app)
		if inside {
			touchInsideNode = n
		}
	}

	touch := &ui.buff.win.io.touch
	keys := &ui.buff.win.io.keys
	over := lv.call.IsOver(ui)

	if touch.rm {
		touchInsideNode = nil
	}

	//keys actions
	if over && ui.edit.uid == nil {

		//delete
		if keys.delete {
			gr.app.act.RemoveSelectedNodes()
		}

		if keys.copy {
			for _, n := range gr.app.act.Subs {
				if n.Selected {
					keys.clipboard = n.Name //copy name into clipboard
					break
				}
			}
		}

		if strings.EqualFold(keys.text, "h") {
			gr.autoZoom(false, ui) //zoom to all
		}
		if strings.EqualFold(keys.text, "g") {
			gr.autoZoom(true, ui) //zoom to selected
		}

		if keys.text == "l" {
			gr.reorder(false, ui) //reorder nodes
		}
		if keys.text == "L" {
			gr.reorder(true, ui) //reorder only selected nodes
		}

	}

	//search node editbox ...

	//touch actions
	{
		//nodes
		if touchInsideNode != nil && over && touch.start && !keys.shift && !keys.ctrl {
			gr.node_move = true
			gr.touch_start = touch.pos
			for _, n := range gr.app.act.Subs {
				n.pos_start = n.Pos
			}
			gr.node_move_selected = touchInsideNode

			//click on un-selected => de-select all & select only current
			if !touchInsideNode.Selected {
				for _, n := range gr.app.act.Subs {
					n.Selected = false
				}
				touchInsideNode.Selected = true
			}
		}
		if gr.node_move {
			cell := ui.win.Cell()
			p := touch.pos.Sub(gr.touch_start)
			var r OsV2f
			r.X = float32(p.X) / float32(gr.app.act.Cam_z) / float32(cell)
			r.Y = float32(p.Y) / float32(gr.app.act.Cam_z) / float32(cell)

			for _, n := range gr.app.act.Subs {
				if n.Selected {
					n.Pos = n.pos_start.Add(r)
				}
			}
		}

		//zoom
		if over && touch.wheel != 0 {
			zoom := OsClampFloat(float64(gr.app.act.Cam_z)+float64(touch.wheel)*-0.1, 0.2, 1.0) //zoom += wheel
			gr.app.act.Cam_z = zoom
			gr.app.saveIt = true

			touch.wheel = 0
		}

		//cam & selection
		if (touchInsideNode == nil || keys.shift || keys.ctrl) && over && touch.start {
			if touch.rm {
				//start camera move
				gr.cam_move = true
				gr.touch_start = touch.pos
				gr.cam_start = OsV2f{float32(gr.app.act.Cam_x), float32(gr.app.act.Cam_y)}
			} else if touchInsideNode == nil || keys.shift || keys.ctrl {
				//start selection
				gr.node_select = true
				gr.touch_start = touch.pos
			}
		}

		if gr.cam_move {
			cell := ui.win.Cell()
			p := touch.pos.Sub(gr.touch_start)
			var r OsV2f
			r.X = float32(p.X) / float32(gr.app.act.Cam_z) / float32(cell)
			r.Y = float32(p.Y) / float32(gr.app.act.Cam_z) / float32(cell)

			gr.app.act.Cam_x = float64(gr.cam_start.Sub(r).X)
			gr.app.act.Cam_y = float64(gr.cam_start.Sub(r).Y)
			gr.app.saveIt = true
		}

		if gr.node_select {
			coord := InitOsV4ab(gr.touch_start, touch.pos)
			coord.Size.X++
			coord.Size.Y++

			ui.buff.AddRect(coord, pl.P.SetAlpha(50), 0)     //back
			ui.buff.AddRect(coord, pl.P, ui.CellWidth(0.03)) //border

			//update
			for _, n := range gr.app.act.Subs {
				cq, _ := n.nodeToPixelsCoord(ui)
				n.selected_cover = coord.HasCover(cq)
			}
		}

	}

	if touch.end {
		//when it's clicked on selected node, but it's not moved => select only this node
		if gr.node_move && gr.node_move_selected != nil {
			if gr.touch_start.Distance(touch.pos) < float32(ui.win.Cell())/5 {
				for _, n := range gr.app.act.Subs {
					n.Selected = false
				}
				gr.node_move_selected.Selected = true
			}
		}

		if gr.node_select {
			for _, n := range gr.app.act.Subs {
				n.Selected = n.KeyProgessSelection(keys)
			}
		}

		gr.cam_move = false
		gr.node_move = false
		gr.node_select = false
	}
}
