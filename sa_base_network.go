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

func (base *SABase) drawCreateNode(app *SAApp, ui *Ui) {

	lvBaseDiv := ui.GetCall().call

	if ui.win.io.keys.tab && lvBaseDiv.data.over {
		app.tab_touchPos = ui.buff.win.io.touch.pos
		ui.Dialog_open("nodes_list", 2)
	}

	if ui.Dialog_start("nodes_list") {
		ui.Div_colMax(0, 5)

		y := 0
		var search string
		ui.Comp_editbox(0, 0, 1, 1, &search, 0, "", base.trns.SAVE, search != "", true)
		y++

		keys := &ui.buff.win.io.keys
		for _, fn := range app.nodes.fns {
			if search == "" || strings.Contains(fn.name, search) {

				if keys.enter || ui.Comp_buttonMenu(0, y, 1, 1, fn.name, "", true, false) > 0 {
					n := app.nodes.AddNode("", fn)
					n.Pos = base.pixelsToNode(app.tab_touchPos, app, ui, lvBaseDiv)
					ui.Dialog_close()
				}
				y++

			}
		}

		ui.Dialog_end()
	}

}

func (base *SABase) pixelsToNode(touchPos OsV2, app *SAApp, ui *Ui, lvDiv *UiLayoutDiv) OsV2f {

	cell := ui.win.Cell()

	p := touchPos.Sub(lvDiv.canvas.Start).Sub(lvDiv.canvas.Size.MulV(0.5))

	var r OsV2f
	r.X = float32(p.X) / float32(app.Cam_zoom) / float32(cell)
	r.Y = float32(p.Y) / float32(app.Cam_zoom) / float32(cell)

	r = r.Add(app.Cam)

	return r
}

func (base *SABase) nodeToPixels(p OsV2f, app *SAApp, ui *Ui) OsV2 {

	lv := ui.GetCall()
	cell := ui.win.Cell()

	p = p.Sub(app.Cam)

	var r OsV2
	r.X = int(p.X * float32(cell) * float32(app.Cam_zoom))
	r.Y = int(p.Y * float32(cell) * float32(app.Cam_zoom))

	return r.Add(lv.call.canvas.Start).Add(lv.call.canvas.Size.MulV(0.5))
}

func (base *SABase) nodeToPixelsCoord(node *Node, app *SAApp, ui *Ui) OsV4 {

	coordS := base.nodeToPixels(node.Pos, app, ui)
	cellr := int(float32(ui.win.Cell()) * app.Cam_zoom * 0.8)

	return InitOsV4Mid(coordS, OsV2{cellr * 3, cellr})
}

func (base *SABase) getIORad(app *SAApp, ui *Ui) int {
	return int(float32(ui.win.Cell()) / 4 * app.Cam_zoom)
}

func (base *SABase) getInputPos(pos int, node *Node, app *SAApp, ui *Ui) (OsV2, OsV4, OsV4) {

	coord := base.nodeToPixelsCoord(node, app, ui)

	fn := app.nodes.FindFn(node.FnName)
	if fn != nil {
		rad := base.getIORad(app, ui)
		n := len(fn.ins)
		sp := coord.Size.X / (n + 1)
		p := OsV2{coord.Start.X + (sp * (pos + 1)), coord.Start.Y}
		p.Y -= rad

		coord := InitOsV4Mid(p, OsV2{rad, rad})
		return p, coord, coord.AddSpace(-rad / 2)
	}
	return OsV2{}, OsV4{}, OsV4{}
}
func (base *SABase) getOutputPos(pos int, node *Node, app *SAApp, ui *Ui) (OsV2, OsV4, OsV4) {

	coord := base.nodeToPixelsCoord(node, app, ui)

	fn := app.nodes.FindFn(node.FnName)
	if fn != nil {
		rad := base.getIORad(app, ui)
		n := len(fn.outs)
		sp := coord.Size.X / (n + 1)
		p := OsV2{coord.Start.X + (sp * (pos + 1)), coord.Start.Y + coord.Size.Y}
		p.Y += rad

		coord := InitOsV4Mid(p, OsV2{rad, rad})
		return p, coord, coord.AddSpace(-rad / 2)
	}
	return OsV2{}, OsV4{}, OsV4{}
}

func (base *SABase) drawNode(node *Node, app *SAApp, ui *Ui) bool {

	lv := ui.GetCall()
	touch := &ui.buff.win.io.touch
	pl := ui.buff.win.io.GetPalette()

	coord := base.nodeToPixelsCoord(node, app, ui)

	ui.buff.AddRect(coord, pl.P, 0)

	//body
	if (app.node_select && node.KeyProgessSelection(&ui.buff.win.io.keys)) || (!app.node_select && node.selected) {
		cq := coord.AddSpace(ui.CellWidth(-0.1))
		ui.buff.AddRect(cq, pl.S, ui.CellWidth(0.05)) //selected
	}

	//label
	{
		cq := coord
		cq.Start.X += cq.Size.X + ui.CellWidth(0.1)
		cq.Size.X = 1000
		ui._compDrawText(cq, node.Name, "", pl.OnB, SKYALT_FONT_HEIGHT*float64(app.Cam_zoom), false, false, 0, 1, true)
	}

	inside := coord.GetIntersect(lv.call.crop).Inside(touch.pos)

	//inputs/outputs
	fn := app.nodes.FindFn(node.FnName)
	if fn != nil {
		//inputs
		for i := range fn.ins {
			_, cq, cqH := base.getInputPos(i, node, app, ui)

			cd := pl.GetGrey(0.5)
			if lv.call.data.over && cqH.Inside(touch.pos) {
				cd = pl.S
			}

			ui.buff.AddCircle(cq, cd, 0)
		}

		//outputs
		for i := range fn.outs {
			_, cq, cqH := base.getOutputPos(i, node, app, ui)

			cd := pl.GetGrey(0.5)
			if lv.call.data.over && cqH.Inside(touch.pos) {
				cd = pl.S
			}

			ui.buff.AddCircle(cq, cd, 0)
		}
	}

	return inside
}

func (base *SABase) drawConnection(start OsV2, end OsV2, ui *Ui) {
	sp := ui.win.Cell() / 2

	pl := ui.buff.win.io.GetPalette()
	cd := pl.GetGrey(0.5)

	mid := start.Aprox(end, 0.5)
	ui.buff.AddBezier(start, OsV2{start.X, mid.Y}, OsV2{end.X, mid.Y}, end, cd, ui.CellWidth(0.03), false)
	//ui.buff.AddLine(start, end, cd, ui.CellWidth(0.03))

	t := sp / 5
	ui.buff.AddTringle(end, end.Add(OsV2{-t, t}), end.Add(OsV2{t, t}), pl.GetGrey(0.5)) //arrow
}

func (base *SABase) drawNodeConnection(outNode *Node, inNode *Node, outPos int, inPos int, app *SAApp, ui *Ui) {

	out, _, _ := base.getOutputPos(outPos, outNode, app, ui)
	in, _, _ := base.getInputPos(inPos, inNode, app, ui)

	base.drawConnection(out, in, ui)
}

func (base *SABase) findInputOver(app *SAApp, ui *Ui, emptySlotOverNode bool) (*Node, int) {

	lv := ui.GetCall()
	touch := &ui.buff.win.io.touch

	var node_found *Node
	node_pos := -1

	for _, n := range app.nodes.nodes {
		fn := app.nodes.FindFn(n.FnName)
		if fn != nil {
			//inputs
			for i, in := range fn.ins {
				_, _, cq := base.getInputPos(i, n, app, ui)

				if lv.call.data.over && cq.Inside(touch.pos) {
					node_found = n
					node_pos = i

					ui.tile.Set(touch.pos, cq, false, in.name, OsCd{0, 0, 0, 255})
				}
			}
		}
	}

	if node_found == nil && emptySlotOverNode {
		for _, n := range app.nodes.nodes {
			cq := base.nodeToPixelsCoord(n, app, ui)
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
func (base *SABase) findOutputOver(app *SAApp, ui *Ui, emptySlotOverNode bool) (*Node, int) {

	lv := ui.GetCall()
	touch := &ui.buff.win.io.touch

	var node_found *Node
	node_pos := -1

	for _, n := range app.nodes.nodes {
		fn := app.nodes.FindFn(n.FnName)
		if fn != nil {
			//outputs
			for i, out := range fn.outs {
				_, _, cq := base.getOutputPos(i, n, app, ui)

				if lv.call.data.over && cq.Inside(touch.pos) {
					node_found = n
					node_pos = i

					ui.tile.Set(touch.pos, cq, false, out.name, OsCd{0, 0, 0, 255})
				}
			}
		}
	}

	if node_found == nil && emptySlotOverNode {
		for _, n := range app.nodes.nodes {
			cq := base.nodeToPixelsCoord(n, app, ui)
			if lv.call.data.over && cq.Inside(touch.pos) {
				//find empty slot

				//must go through all nodes which can connect to this one ..............
			}
		}
	}

	return node_found, node_pos
}

func (base *SABase) drawNetwork(app *SAApp, ui *Ui) {

	if len(app.nodes.nodes) == 0 {
		app.Cam_zoom = 1
	}

	pl := ui.buff.win.io.GetPalette()

	ui.Div_colMax(0, 100)
	ui.Div_rowMax(0, 100)

	//fade "press tab" centered in background
	lv := ui.GetCall()
	ui._compDrawText(lv.call.canvas, "press tab", "", pl.GetGrey(0.7), SKYALT_FONT_HEIGHT, false, false, 1, 1, true)

	//+
	base.drawCreateNode(app, ui)

	//draw connections
	for _, nodeIn := range app.nodes.nodes {
		for i, in := range nodeIn.Inputs {
			nodeOut := app.nodes.FindNode(in.Node)
			if nodeOut != nil {
				base.drawNodeConnection(nodeOut, nodeIn, in.Pos, i, app, ui)
			}
		}
	}

	//draw node bodies
	var clickedNode *Node
	for _, n := range app.nodes.nodes {
		if base.drawNode(n, app, ui) {
			clickedNode = n
		}
	}

	//keys actions
	//H - zoom out to all ...
	//G - zoom to selected ...

	//Node context menu: copy / cut / paste / delete ...

	//Node bypass flag ...

	//Node rename ...

	//touch actions
	touch := &ui.buff.win.io.touch
	keys := &ui.buff.win.io.keys
	if lv.call.data.over {

		//delete node connection ...

		//connect with 2x click ...
		//disconnect with 2x click ...

		//nodes: inputs/outputs
		{
			node_in, node_in_pos := base.findInputOver(app, ui, false)
			node_out, node_out_pos := base.findOutputOver(app, ui, false)
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
				p, _, _ := base.getInputPos(app.node_connect_in, app.node_connect, app, ui)
				base.drawConnection(touch.pos, p, ui)
			}

			if app.node_connect_out >= 0 {
				p, _, _ := base.getOutputPos(app.node_connect_out, app.node_connect, app, ui)
				base.drawConnection(p, touch.pos, ui)
			}
		}

		//nodes
		if clickedNode != nil && touch.start && !keys.shift && !keys.ctrl && app.node_connect == nil {
			app.node_move = true
			app.touch_start = touch.pos
			for _, n := range app.nodes.nodes {
				n.pos_start = n.Pos
			}
			app.node_move_selected = clickedNode

			//click on un-selected => de-select all & select only current
			if !clickedNode.selected {
				for _, n := range app.nodes.nodes {
					n.selected = false
				}
				clickedNode.selected = true
			}
		}
		if app.node_move {
			cell := ui.win.Cell()
			p := touch.pos.Sub(app.touch_start)
			var r OsV2f
			r.X = float32(p.X) / float32(app.Cam_zoom) / float32(cell)
			r.Y = float32(p.Y) / float32(app.Cam_zoom) / float32(cell)

			for _, n := range app.nodes.nodes {
				if n.selected {
					n.Pos = n.pos_start.Add(r)
				}
			}
		}

		//zoom
		if touch.wheel != 0 {
			app.Cam_zoom += float32(touch.wheel) * -0.1
			if app.Cam_zoom <= 0.1 {
				app.Cam_zoom = 0.1
			}
			if app.Cam_zoom >= 2.5 {
				app.Cam_zoom = 2.5
			}

			touch.wheel = 0
		}

		//cam
		if (clickedNode == nil || keys.shift || keys.ctrl) && touch.start && app.node_connect == nil {
			if keys.space || touch.rm {
				//start camera move
				app.cam_move = true
				app.touch_start = touch.pos
				app.cam_start = app.Cam
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
			r.X = float32(p.X) / float32(app.Cam_zoom) / float32(cell)
			r.Y = float32(p.Y) / float32(app.Cam_zoom) / float32(cell)

			app.Cam = app.cam_start.Sub(r)
		}

		if app.node_select {
			coord := InitOsV4ab(app.touch_start, touch.pos)
			coord.Size.X++
			coord.Size.Y++

			ui.buff.AddRect(coord, pl.P.SetAlpha(50), 0)     //back
			ui.buff.AddRect(coord, pl.P, ui.CellWidth(0.03)) //border

			//update
			for _, n := range app.nodes.nodes {
				n.selected_cover = coord.HasCover(base.nodeToPixelsCoord(n, app, ui))
			}
		}

	}

	if touch.end {
		//when it's clicked on selected node, but it's not moved => select only this node
		if app.node_move && app.node_move_selected != nil {
			if app.touch_start.Distance(touch.pos) < float32(ui.win.Cell())/5 {
				for _, n := range app.nodes.nodes {
					n.selected = false
				}
				app.node_move_selected.selected = true
			}
		}

		if app.node_select {
			for _, n := range app.nodes.nodes {
				n.selected = n.KeyProgessSelection(keys)
			}
		}

		if app.node_connect != nil {
			if app.node_connect_in >= 0 {
				node_out, node_out_pos := base.findOutputOver(app, ui, true)

				if node_out != nil {
					app.node_connect.SetInput(app.node_connect_in, node_out, node_out_pos)
				}
			}

			if app.node_connect_out >= 0 {
				node_in, node_in_pos := base.findInputOver(app, ui, true)

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
