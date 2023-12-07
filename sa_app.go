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

type SAApp struct {
	parent *SABase

	Name string
	IDE  bool

	view               *NodeView
	cam_move           bool
	node_move          bool
	node_select        bool
	cam_start          OsV2f
	touch_start        OsV2
	node_move_selected *Node

	node_connect     *Node
	node_connect_in  *NodeParamIn
	node_connect_out *NodeParamOut

	saveIt bool

	tab_touchPos OsV2
}

func NewSAApp(name string, parent *SABase) *SAApp {
	var app SAApp
	app.parent = parent
	app.Name = name
	app.IDE = true

	return &app
}

func (app *SAApp) GetPath() string {
	return "apps/" + app.Name + "/app.json"
}

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

		var fns []string
		fns = append(fns, "gui_sub")
		fns = append(fns, "gui_text")
		fns = append(fns, "gui_edit")
		//...

		keys := &ui.buff.win.io.keys

		for _, fn := range fns {
			if search == "" || strings.Contains(fn, search) {
				if keys.enter || ui.Comp_buttonMenu(0, y, 1, 1, fn, "", true, false) > 0 {
					n := app.view.act.AddNode(fn)
					n.Pos = app.view.pixelsToNode(app.tab_touchPos, ui, lvBaseDiv)

					if strings.HasPrefix(fn, "gui_") {
						n.GetInput("grid_x")
						n.GetInput("grid_y")
						n.GetInput("grid_w").Value = "1"
						n.GetInput("grid_h").Value = "1"
					}
					if n.IsGuiSub() {
						n.GetAttr("cam_z").Value = "1"
					}

					ui.Dialog_close()
					break
				}
				y++
			}
		}

		ui.Dialog_end()
	}
}

func (app *SAApp) drawGraphHeader(ui *Ui) {
	ui.Div_colMax(1, 100)

	//level up
	if ui.Comp_buttonIcon(0, 0, 1, 1, "file:apps/base/resources/levelup.png", 0.3, "One level up", app.view.act.parent != nil) > 0 {
		app.view.act = app.view.act.parent
	}

	//list
	{
		var listPathes string
		var listNodes []*Node
		app.view.root.buildSubsList(&listPathes, &listNodes)
		if len(listPathes) >= 1 {
			listPathes = listPathes[:len(listPathes)-1] //cut last '|'
		}
		combo := 0
		for i, n := range listNodes {
			if app.view.act == n {
				combo = i
			}
		}
		if ui.Comp_combo(1, 0, 1, 1, &combo, listPathes, "", true, true) {
			app.view.act = listNodes[combo]
		}
	}

	//short cuts
	if ui.Comp_buttonLight(2, 0, 1, 1, "←", "Back", app.view.canHistoryBack()) > 0 {
		app.view.stepHistoryBack()

	}
	if ui.Comp_buttonLight(3, 0, 1, 1, "→", "Forward", app.view.canHistoryForward()) > 0 {
		app.view.stepHistoryForward()
	}
}

func _SAApp_drawConnection(start OsV2, end OsV2, active bool, ui *Ui) {
	cd := Node_connectionCd(ui)
	if active {
		cd = Node_getYellow()
	}

	mid := start.Aprox(end, 0.5)
	ui.buff.AddBezier(start, OsV2{mid.X, start.Y}, OsV2{mid.X, end.Y}, end, cd, ui.CellWidth(0.03), false)
	//ui.buff.AddLine(start, end, cd, ui.CellWidth(0.03))

	//t := sp / 5
	//ui.buff.AddTringle(end, end.Add(OsV2{-t, t}), end.Add(OsV2{t, t}), pl.GetGrey(0.5)) //arrow
}

func (app *SAApp) drawGraph(ui *Ui) {
	pl := ui.buff.win.io.GetPalette()

	ui.Div_colMax(0, 100)
	ui.Div_rowMax(0, 100)

	ui.Paint_rect(0, 0, 1, 1, 0, pl.GetGrey(0.8), 0)

	//fade "press tab" centered in background
	lv := ui.GetCall()
	ui._compDrawText(lv.call.canvas, "press tab", "", pl.GetGrey(1), SKYALT_FONT_HEIGHT, false, false, 1, 1, true)

	//+
	app.drawCreateNode(ui)

	//draw connections
	for _, nodeIn := range app.view.act.Subs {
		for _, paramIn := range nodeIn.Inputs {

			_, paramOut := paramIn.FindWireOut(nodeIn)

			if paramOut != nil {
				_SAApp_drawConnection(paramOut.coordDot.Middle(), paramIn.coordDot.Middle(), false, ui)
			}
		}
	}

	//draw node bodies
	var touchInsideNode *Node
	var touchMoveNode *Node
	for _, n := range app.view.act.Subs {
		inside, move := n.drawNode(app.node_select, ui, app.view)
		if inside {
			touchInsideNode = n
		}
		if move {
			touchMoveNode = n
		}
	}

	touch := &ui.buff.win.io.touch
	keys := &ui.buff.win.io.keys
	over := lv.call.data.over

	//keys actions
	{
		//delete
		if over && keys.delete {
			app.view.RemoveSelectedNodes()
		}

		//bypass
		if over && strings.EqualFold(keys.text, "b") {
			app.view.BypassSelectedNodes()
		}
		//H - zoom out to all ...
		//G - zoom to selected ...
	}

	//touch actions
	{
		//nodes: inputs/outputs
		{
			node_in, node_in_param := app.view.findInputOver(false, ui)
			node_out, node_out_param := app.view.findOutputOver(false, ui)
			if over && touch.start {
				if node_in != nil && node_in_param != nil {
					app.node_connect = node_in
					app.node_connect_in = node_in_param
					app.node_connect_out = nil
				}
				if node_out != nil && node_out_param != nil {
					app.node_connect = node_out
					app.node_connect_in = nil
					app.node_connect_out = node_out_param
				}
			}
		}
		if app.node_connect != nil {
			if app.node_connect_in != nil {
				_SAApp_drawConnection(touch.pos, app.node_connect_in.coordDot.Middle(), true, ui)
			}
			if app.node_connect_out != nil {
				_SAApp_drawConnection(app.node_connect_out.coordDot.Middle(), touch.pos, true, ui)
			}
		}

		//nodes
		if touchMoveNode != nil && over && touch.start && !keys.shift && !keys.ctrl && app.node_connect == nil {
			app.node_move = true
			app.touch_start = touch.pos
			for _, n := range app.view.act.Subs {
				n.pos_start = n.Pos
			}
			app.node_move_selected = touchMoveNode

			//click on un-selected => de-select all & select only current
			if !touchMoveNode.Selected {
				for _, n := range app.view.act.Subs {
					n.Selected = false
				}
				touchMoveNode.Selected = true
			}
		}
		if app.node_move {
			cell := ui.win.Cell()
			p := touch.pos.Sub(app.touch_start)
			var r OsV2f
			zoom := app.view.GetAttrCamZ()
			r.X = float32(p.X) / zoom / float32(cell)
			r.Y = float32(p.Y) / zoom / float32(cell)

			for _, n := range app.view.act.Subs {
				if n.Selected {
					n.Pos = n.pos_start.Add(r)
				}
			}
		}

		//zoom
		if over && touch.wheel != 0 {
			zoom := OsClampFloat(float64(app.view.GetAttrCamZ())+float64(touch.wheel)*-0.1, 0.4, 2.5) //zoom += wheel

			app.view.SetAttrCamZ(float32(zoom))

			touch.wheel = 0
		}

		//cam
		if (touchMoveNode == nil || keys.shift || keys.ctrl) && over && touch.start && app.node_connect == nil {
			if keys.space || touch.rm {
				//start camera move
				app.cam_move = true
				app.touch_start = touch.pos
				app.cam_start = app.view.GetAttrCam()
			} else if touchInsideNode == nil {
				//start selection
				app.node_select = true
				app.touch_start = touch.pos
			}
		}
		if app.cam_move {
			cell := ui.win.Cell()
			p := touch.pos.Sub(app.touch_start)
			var r OsV2f
			zoom := app.view.GetAttrCamZ()
			r.X = float32(p.X) / zoom / float32(cell)
			r.Y = float32(p.Y) / zoom / float32(cell)

			app.view.SetAttrCam(app.cam_start.Sub(r))
		}

		if app.node_select {
			coord := InitOsV4ab(app.touch_start, touch.pos)
			coord.Size.X++
			coord.Size.Y++

			ui.buff.AddRect(coord, pl.P.SetAlpha(50), 0)     //back
			ui.buff.AddRect(coord, pl.P, ui.CellWidth(0.03)) //border

			//update
			for _, n := range app.view.act.Subs {
				n.selected_cover = coord.HasCover(n.nodeToPixelsCoord(ui, app.view))
			}
		}

	}

	if touch.end {
		//when it's clicked on selected node, but it's not moved => select only this node
		if app.node_move && app.node_move_selected != nil {
			if app.touch_start.Distance(touch.pos) < float32(ui.win.Cell())/5 {
				for _, n := range app.view.act.Subs {
					n.Selected = false
				}
				app.node_move_selected.Selected = true
			}
		}

		if app.node_select {
			for _, n := range app.view.act.Subs {
				n.Selected = n.KeyProgessSelection(keys)
			}
		}

		if app.node_connect != nil {

			if app.node_connect_in != nil {
				node_out, node_out_param := app.view.findOutputOver(true, ui)
				app.node_connect_in.SetWire(node_out, node_out_param) //connet & disconnect
			}

			if app.node_connect_out != nil {
				node_in, node_in_param := app.view.findInputOver(true, ui)
				if node_in != nil {
					node_in_param.SetWire(app.node_connect, app.node_connect_out)
				}
			}
		}

		app.cam_move = false
		app.node_move = false
		app.node_select = false
		app.node_connect = nil
	}

	//short cuts
	if lv.call.data.over && ui.win.io.keys.backward {
		app.view.stepHistoryBack()

	}
	if lv.call.data.over && ui.win.io.keys.forward {
		app.view.stepHistoryForward()
	}

	if touch.end || keys.hasChanged {
		app.view.cmpAndAddHistory()
	}
}
