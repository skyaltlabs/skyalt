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
	"strings"
)

func (app *SAApp) drawCreateNode(ui *Ui) {

	lvBaseDiv := ui.GetCall().call

	if ui.win.io.keys.tab && lvBaseDiv.data.over {
		app.tab_touchPos = ui.buff.win.io.touch.pos
		ui.Dialog_open("nodes_list", 2)
	}

	if ui.Dialog_start("nodes_list") {
		ui.Div_colMax(0, 5)

		y := 0
		var search string
		ui.Comp_editbox(0, 0, 1, 1, &search, 0, "", app.parent.trns.SAVE, search != "", true, true)
		y++

		keys := &ui.buff.win.io.keys
		for _, fn := range app.app.fns {
			if search == "" || strings.Contains(fn.name, search) {
				if keys.enter || ui.Comp_buttonMenu(0, y, 1, 1, fn.name, "", true, false) > 0 {
					n := app.app.root.AddNode("", fn)
					n.Pos = app.app.root.pixelsToNode(app.tab_touchPos, ui, lvBaseDiv)
					ui.Dialog_close()
					break
				}
				y++
			}
		}

		ui.Dialog_end()
	}

}

func (node *Node) pixelsToNode(touchPos OsV2, ui *Ui, lvDiv *UiLayoutDiv) OsV2f {

	cell := ui.win.Cell()

	p := touchPos.Sub(lvDiv.canvas.Start).Sub(lvDiv.canvas.Size.MulV(0.5))

	var r OsV2f
	r.X = float32(p.X) / node.Cam_z / float32(cell)
	r.Y = float32(p.Y) / node.Cam_z / float32(cell)

	r = r.Add(OsV2f{node.Cam_x, node.Cam_y})

	return r
}

func (node *Node) nodeToPixels(p OsV2f, ui *Ui) OsV2 {

	lv := ui.GetCall()
	cell := ui.win.Cell()

	p = p.Sub(OsV2f{node.Cam_x, node.Cam_y})

	var r OsV2
	r.X = int(p.X * float32(cell) * node.Cam_z)
	r.Y = int(p.Y * float32(cell) * node.Cam_z)

	return r.Add(lv.call.canvas.Start).Add(lv.call.canvas.Size.MulV(0.5))
}

func (node *Node) nodeToPixelsCoord(ui *Ui) OsV4 {

	coordS := node.parent.nodeToPixels(node.Pos, ui) //.parent, because it has Cam
	cellr := int(float32(ui.win.Cell()) * node.parent.Cam_z * 0.8)

	return InitOsV4Mid(coordS, OsV2{cellr * 3, cellr})
}

func (node *Node) getIORad(ui *Ui) int {
	return int(float32(ui.win.Cell()) / 4 * node.parent.Cam_z)
}

func (node *Node) getInputPos(pos int, ui *Ui) (OsV2, OsV4, OsV4) {

	coord := node.nodeToPixelsCoord(ui)

	fn := node.FindFn(node.FnName)
	if fn != nil {
		rad := node.getIORad(ui)
		n := len(fn.ins)
		sp := coord.Size.X / (n + 1)
		p := OsV2{coord.Start.X + (sp * (pos + 1)), coord.Start.Y}
		p.Y -= rad

		coord := InitOsV4Mid(p, OsV2{rad, rad})
		return p, coord, coord.AddSpace(-rad / 2)
	}
	return OsV2{}, OsV4{}, OsV4{}
}
func (node *Node) getOutputPos(pos int, ui *Ui) (OsV2, OsV4, OsV4) {

	coord := node.nodeToPixelsCoord(ui)

	fn := node.FindFn(node.FnName)
	if fn != nil {
		rad := node.getIORad(ui)
		n := len(fn.outs)
		sp := coord.Size.X / (n + 1)
		p := OsV2{coord.Start.X + (sp * (pos + 1)), coord.Start.Y + coord.Size.Y}
		p.Y += rad

		coord := InitOsV4Mid(p, OsV2{rad, rad})
		return p, coord, coord.AddSpace(-rad / 2)
	}
	return OsV2{}, OsV4{}, OsV4{}
}

func Node_getYellow() OsCd {
	return OsCd{204, 204, 0, 255} //...
}

func (node *Node) drawNode(ui *Ui, someNodeIsDraged bool) bool {

	lv := ui.GetCall()
	touch := &ui.buff.win.io.touch
	pl := ui.buff.win.io.GetPalette()

	coord := node.nodeToPixelsCoord(ui)

	backCd := pl.P
	labelCd := pl.OnB
	if node.Bypass {
		backCd.A = 100
		labelCd.A = 100
	}

	ui.buff.AddRect(coord, backCd, 0)

	//select
	if (someNodeIsDraged && node.KeyProgessSelection(&ui.buff.win.io.keys)) || (!someNodeIsDraged && node.Selected) {
		cq := coord.AddSpace(ui.CellWidth(-0.1))
		ui.buff.AddRect(cq, Node_getYellow(), ui.CellWidth(0.06)) //selected
	}

	//label
	{
		cq := coord
		cq.Start.X += cq.Size.X + ui.CellWidth(0.1)
		cq.Size.X = 1000
		ui._compDrawText(cq, node.Name, "", labelCd, SKYALT_FONT_HEIGHT*float64(node.parent.Cam_z), false, false, 0, 1, true)
	}

	inside := coord.GetIntersect(lv.call.crop).Inside(touch.pos)

	//inputs/outputs
	fn := node.FindFn(node.FnName)
	if fn != nil {
		//inputs
		for i := range fn.ins {
			_, cq, cqH := node.getInputPos(i, ui)

			cd := pl.GetGrey(0.5)
			if lv.call.data.over && cqH.Inside(touch.pos) {
				cd = pl.S
			}

			ui.buff.AddCircle(cq, cd, 0)
		}

		//outputs
		for i := range fn.outs {
			_, cq, cqH := node.getOutputPos(i, ui)

			cd := pl.GetGrey(0.5)
			if lv.call.data.over && cqH.Inside(touch.pos) {
				cd = pl.S
			}

			ui.buff.AddCircle(cq, cd, 0)
		}
	}

	return inside
}

func Node_drawConnection(start OsV2, end OsV2, ui *Ui) {
	//sp := ui.win.Cell() / 2

	pl := ui.buff.win.io.GetPalette()
	cd := pl.GetGrey(0.5)

	mid := start.Aprox(end, 0.5)
	ui.buff.AddBezier(start, OsV2{start.X, mid.Y}, OsV2{end.X, mid.Y}, end, cd, ui.CellWidth(0.03), false)
	//ui.buff.AddLine(start, end, cd, ui.CellWidth(0.03))

	//t := sp / 5
	//ui.buff.AddTringle(end, end.Add(OsV2{-t, t}), end.Add(OsV2{t, t}), pl.GetGrey(0.5)) //arrow
}

func Node_drawNodeConnection(outNode *Node, inNode *Node, outPos int, inPos int, ui *Ui) {

	out, _, _ := outNode.getOutputPos(outPos, ui)
	in, _, _ := inNode.getInputPos(inPos, ui)

	Node_drawConnection(out, in, ui)
}

func (node *Node) findInputOver(ui *Ui, emptySlotOverNode bool) (*Node, int) {

	lv := ui.GetCall()
	touch := &ui.buff.win.io.touch

	var node_found *Node
	node_pos := -1

	for _, n := range node.Subs {
		fn := node.FindFn(n.FnName)
		if fn != nil {
			//inputs
			for i, in := range fn.ins {
				_, _, cq := n.getInputPos(i, ui)

				if lv.call.data.over && cq.Inside(touch.pos) {
					node_found = n
					node_pos = i

					ui.tile.Set(touch.pos, cq, false, in.name, OsCd{0, 0, 0, 255})
				}
			}
		}
	}

	if node_found == nil && emptySlotOverNode {
		for _, n := range node.Subs {
			cq := n.nodeToPixelsCoord(ui)
			if lv.call.data.over && cq.Inside(touch.pos) {
				//find empty slot
				for i, in := range n.Inputs {
					if in.Node == "" {
						node_found = n
						node_pos = i
					}
				}

				//no empty -> use first one
				if node_found == nil {
					node_found = n
					node_pos = 0
				}
			}
		}
	}

	return node_found, node_pos
}
func (node *Node) findOutputOver(ui *Ui, emptySlotOverNode bool) (*Node, int) {

	lv := ui.GetCall()
	touch := &ui.buff.win.io.touch

	var node_found *Node
	node_pos := -1

	for _, n := range node.Subs {
		fn := node.FindFn(n.FnName)
		if fn != nil {
			//outputs
			for i, out := range fn.outs {
				_, _, cq := n.getOutputPos(i, ui)

				if lv.call.data.over && cq.Inside(touch.pos) {
					node_found = n
					node_pos = i

					ui.tile.Set(touch.pos, cq, false, out.name, OsCd{0, 0, 0, 255})
				}
			}
		}
	}

	/*if node_found == nil && emptySlotOverNode {
		for _, n := range node.Subs {
			cq := n.nodeToPixelsCoord(ui)
			if lv.call.data.over && cq.Inside(touch.pos) {
				//find empty slot

				//must go through all nodes which can connect to this one ...
			}
		}
	}*/

	return node_found, node_pos
}

func (node *Node) drawNetwork(app *SAApp, ui *Ui) {

	pl := ui.buff.win.io.GetPalette()

	ui.Div_colMax(0, 100)
	ui.Div_rowMax(0, 100)

	//fade "press tab" centered in background
	lv := ui.GetCall()
	ui._compDrawText(lv.call.canvas, "press tab", "", pl.GetGrey(0.7), SKYALT_FONT_HEIGHT, false, false, 1, 1, true)

	//+
	app.drawCreateNode(ui)

	//draw connections
	for _, nodeIn := range node.Subs {
		for i, in := range nodeIn.Inputs {
			nodeOut := node.FindNode(in.Node)
			if nodeOut != nil {
				Node_drawNodeConnection(nodeOut, nodeIn, in.Pos, i, ui)
			}
		}
	}

	//draw node bodies
	var clickedNode *Node
	for _, n := range node.Subs {
		if n.drawNode(ui, app.node_select) {
			clickedNode = n
		}
	}

	//keys actions
	keys := &ui.buff.win.io.keys
	if lv.call.data.over {

		//delete
		if keys.delete {
			for i := len(node.Subs) - 1; i >= 0; i-- {
				if node.Subs[i].Selected {
					node.Subs = append(node.Subs[:i], node.Subs[i+1:]...)
				}
			}

		}

		//bypass
		if strings.EqualFold(keys.text, "b") {
			for _, n := range node.Subs {
				if n.Selected {
					n.Bypass = !n.Bypass
				}
			}
		}
		//H - zoom out to all ...
		//G - zoom to selected ...
	}

	//Node context menu: copy / cut / paste / delete ...

	//Node rename - double click? ...

	//touch actions
	touch := &ui.buff.win.io.touch
	if lv.call.data.over {

		//delete node connection ...

		//connect with 2x click ...
		//disconnect with 2x click ...

		//connect lines have different colors(db, json) ...

		//nodes: inputs/outputs
		{
			node_in, node_in_pos := node.findInputOver(ui, false)
			node_out, node_out_pos := node.findOutputOver(ui, false)
			if touch.start {
				if node_in != nil && node_in_pos >= 0 {
					app.node_connect = node_in
					app.node_connect_in = node_in_pos
					app.node_connect_out = -1
				}
				if node_out != nil && node_out_pos >= 0 {
					app.node_connect = node_out
					app.node_connect_in = -1
					app.node_connect_out = node_out_pos
				}
			}
		}
		if app.node_connect != nil {
			if app.node_connect_in >= 0 {
				p, _, _ := app.node_connect.getInputPos(app.node_connect_in, ui)
				Node_drawConnection(touch.pos, p, ui)
			}

			if app.node_connect_out >= 0 {
				p, _, _ := app.node_connect.getOutputPos(app.node_connect_out, ui)
				Node_drawConnection(p, touch.pos, ui)
			}
		}

		//nodes
		if clickedNode != nil && touch.start && !keys.shift && !keys.ctrl && app.node_connect == nil {
			app.node_move = true
			app.touch_start = touch.pos
			for _, n := range node.Subs {
				n.pos_start = n.Pos
			}
			app.node_move_selected = clickedNode

			//click on un-selected => de-select all & select only current
			if !clickedNode.Selected {
				for _, n := range node.Subs {
					n.Selected = false
				}
				clickedNode.Selected = true
			}
		}
		if app.node_move {
			cell := ui.win.Cell()
			p := touch.pos.Sub(app.touch_start)
			var r OsV2f
			r.X = float32(p.X) / node.Cam_z / float32(cell)
			r.Y = float32(p.Y) / node.Cam_z / float32(cell)

			for _, n := range node.Subs {
				if n.Selected {
					n.Pos = n.pos_start.Add(r)
				}
			}
		}

		//zoom
		if touch.wheel != 0 {
			node.Cam_z += float32(touch.wheel) * -0.1
			if node.Cam_z <= 0.1 {
				node.Cam_z = 0.1
			}
			if node.Cam_z >= 2.5 {
				node.Cam_z = 2.5
			}

			touch.wheel = 0
		}

		//cam
		if (clickedNode == nil || keys.shift || keys.ctrl) && touch.start && app.node_connect == nil {
			if keys.space || touch.rm {
				//start camera move
				app.cam_move = true
				app.touch_start = touch.pos
				app.cam_start = OsV2f{node.Cam_x, node.Cam_y}
			} else {
				//start selection
				app.node_select = true
				app.touch_start = touch.pos
			}
		}
		if app.cam_move {
			cell := ui.win.Cell()
			p := touch.pos.Sub(app.touch_start)
			var r OsV2f
			r.X = float32(p.X) / node.Cam_z / float32(cell)
			r.Y = float32(p.Y) / node.Cam_z / float32(cell)

			node.Cam_x = app.cam_start.Sub(r).X
			node.Cam_y = app.cam_start.Sub(r).Y
		}

		if app.node_select {
			coord := InitOsV4ab(app.touch_start, touch.pos)
			coord.Size.X++
			coord.Size.Y++

			ui.buff.AddRect(coord, pl.P.SetAlpha(50), 0)     //back
			ui.buff.AddRect(coord, pl.P, ui.CellWidth(0.03)) //border

			//update
			for _, n := range node.Subs {
				n.selected_cover = coord.HasCover(n.nodeToPixelsCoord(ui))
			}
		}

	}

	if touch.end {
		//when it's clicked on selected node, but it's not moved => select only this node
		if app.node_move && app.node_move_selected != nil {
			if app.touch_start.Distance(touch.pos) < float32(ui.win.Cell())/5 {
				for _, n := range node.Subs {
					n.Selected = false
				}
				app.node_move_selected.Selected = true
			}
		}

		if app.node_select {
			for _, n := range node.Subs {
				n.Selected = n.KeyProgessSelection(keys)
			}
		}

		if app.node_connect != nil {
			if app.node_connect_in >= 0 {
				node_out, node_out_pos := node.findOutputOver(ui, true)

				if node_out != nil {
					app.node_connect.SetInput(app.node_connect_in, node_out, node_out_pos)
				}
			}

			if app.node_connect_out >= 0 {
				node_in, node_in_pos := node.findInputOver(ui, true)

				if node_in != nil {

					node_in.SetInput(node_in_pos, app.node_connect, app.node_connect_out)

				}
			}

			//click on node and find first free slot ...

		}

		app.cam_move = false
		app.node_move = false
		app.node_select = false
		app.node_connect = nil
	}
}
