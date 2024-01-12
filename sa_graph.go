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
	"strings"
)

type SAGraph struct {
	app *SAApp

	cam_move           bool
	node_move          bool
	node_select        bool
	cam_start          OsV2f
	touch_start        OsV2
	node_move_selected *SANode

	copiedNodes []*SANode

	showNodeList          bool
	showNodeList_justOpen bool
	node_search           string
}

func NewSAGraph(app *SAApp) *SAGraph {
	var gr SAGraph
	gr.app = app
	return &gr
}

func (gr *SAGraph) drawCreateNode(ui *Ui) {

	lvBaseDiv := ui.GetCall().call

	if ui.win.io.keys.tab && lvBaseDiv.IsOver(ui) {
		gr.app.canvas.addGrid = InitOsV4(0, 0, 1, 1)
		gr.app.canvas.addPos = gr.app.act.pixelsToNode(ui.buff.win.io.touch.pos, ui, lvBaseDiv)
		gr.app.canvas.addnode_search = ""
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
	var nodes []*SANode
	for _, n := range gr.app.act.Subs {
		if onlySelected && !n.Selected {
			continue //skip
		}
		nodes = append(nodes, n)
	}

	//vertical
	{
		//reset
		for _, n := range nodes {
			n.depth = 0
		}
		//update depth
		for _, n := range nodes {
			n.UpdateDepth(n)
		}
		//set posY
		for _, n := range nodes {
			n.Pos.Y = float32(n.depth * 2)
		}
	}
	//horizontal
	{
		x_jump := float32(6)

		//reset
		for _, n := range nodes {
			n.Pos.X = -1
		}

		//get num layers
		mx_depth := 0
		for _, n := range nodes {
			mx_depth = OsMax(mx_depth, n.depth)
		}

		//init bottom layer(depends go up)
		var last_layer []*SANode
		x := float32(0)
		for _, n := range nodes {
			if n.depth == mx_depth {
				n.Pos.X = x
				x += x_jump
				last_layer = append(last_layer, n)
			}
		}

		//order layer by layer
		for depth := mx_depth; depth >= 1; depth-- {
			x = float32(0)

			//depends
			var next_layer []*SANode
			for _, node := range last_layer {
				for _, attr := range node.Attrs {
					for _, dp := range attr.depends {
						if dp.node.depth == depth-1 && dp.node.Pos.X < 0 { //never set
							dp.node.Pos.X = x
							x += x_jump
							next_layer = append(next_layer, dp.node)
						}
					}
				}

			}
			//no depends
			for _, node := range nodes {
				if node.depth == depth-1 && node.Pos.X < 0 { //never set
					node.Pos.X = x
					x += x_jump
					next_layer = append(next_layer, node)
				}
			}
			last_layer = next_layer
		}
	}
}

func (gr *SAGraph) autoZoom(onlySelected bool, canvas OsV4, ui *Ui) {
	//zoom to all
	first := true
	var mn OsV2f
	var mx OsV2f
	num := 0
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
		num++
	}
	if num == 0 {
		return
	}
	gr.app.act.Cam_x = float64(mn.X+mx.X) / 2
	gr.app.act.Cam_y = float64(mn.Y+mx.Y) / 2

	gr.app.act.Cam_z = 1
	for gr.app.act.Cam_z > 0.1 {
		areIn := true
		for _, n := range gr.app.act.Subs {
			if onlySelected && !n.Selected {
				continue //skip
			}

			_, cq := n.nodeToPixelsCoord(canvas, ui)
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

	keyAllow := false
	touch := &ui.buff.win.io.touch
	keys := &ui.buff.win.io.keys
	var graphCanvas OsV4

	ui.Div_start(0, 0, 1, 1)
	{
		ui.Div_colMax(0, 100)
		ui.Div_rowMax(0, 100)
		if gr.showNodeList {
			ui.Div_col(1, 3)
			ui.Div_colResize(1, "node_list", 5, false)
		}

		ui.Div_start(0, 0, 1, 1)
		{
			graphCanvas = ui.GetCall().call.canvas

			ui.Div_colMax(0, 100)
			ui.Div_rowMax(0, 100)
			ui.Paint_rect(0, 0, 1, 1, 0, pl.GetGrey(0.8), 0)

			//fade "press tab" bottom - middle in background
			lv := ui.GetCall()
			ui._compDrawText(lv.call.canvas.AddSpace(ui.CellWidth(0.5)), "press tab", "", pl.GetGrey(1), SKYALT_FONT_HEIGHT, false, false, 1, 2, true)

			//+
			gr.drawCreateNode(ui)

			//draw connections
			for _, node := range gr.app.act.Subs {
				for _, in := range node.Attrs {
					for _, out := range in.depends {
						coordOut, selCoordOut := out.node.nodeToPixelsCoord(lv.call.canvas, ui)
						coordIn, selCoordIn := in.node.nodeToPixelsCoord(lv.call.canvas, ui)

						if out.node.Selected {
							coordOut = selCoordOut
						}
						if in.node.Selected {
							coordIn = selCoordIn
						}
						_SAGraph_drawConnection(OsV2{coordOut.Middle().X, coordOut.End().Y}, OsV2{coordIn.Middle().X, coordIn.Start.Y}, node.Selected || out.node.Selected, ui)
					}
				}
			}

			//draw node bodies
			var touchInsideNode *SANode
			for _, n := range gr.app.act.Subs {
				inside := n.drawNode(gr.node_select, gr.app)
				if inside {
					touchInsideNode = n
				}
			}

			over := lv.call.IsOver(ui)

			if touch.rm {
				touchInsideNode = nil
			}

			//keys actions
			keyAllow = (over && ui.edit.uid == nil)
			if keyAllow {

				//bypass
				if keys.text == "b" {
					gr.app.act.BypassReverseSelectedNodes()
				}

				//delete
				if keys.delete {
					gr.app.act.RemoveSelectedNodes()
				}

				//copy
				if keys.copy {
					//add selected into list
					gr.app.graph.copiedNodes = nil
					for _, n := range gr.app.act.Subs {
						if n.Selected {
							gr.app.graph.copiedNodes = append(gr.app.graph.copiedNodes, n)
						}
					}
				}

				//cut
				if keys.cut {
					//add selected into list
					gr.app.graph.copiedNodes = nil
					for _, n := range gr.app.act.Subs {
						if n.Selected {
							gr.app.graph.copiedNodes = append(gr.app.graph.copiedNodes, n)
						}
					}
					gr.app.act.RemoveSelectedNodes()
				}
				//paste
				if keys.paste {

					var newNodes []*SANode
					for _, src := range gr.copiedNodes {
						nw := gr.app.act.AddNodeCopy(src)
						newNodes = append(newNodes, nw)
					}

					//rename expressions access to keep links between copied nodes
					{
						for i := 0; i < len(newNodes); i++ {

							node := newNodes[i]
							node.ParseExpresions()

							for j := 0; j < len(newNodes); j++ {
								oldName := gr.copiedNodes[j].Name
								newName := newNodes[j].Name

								node.RenameExpressionAccess(oldName, newName)
							}
						}
					}

					//select and zoom
					gr.app.act.DeselectAll()
					for _, n := range newNodes {
						n.Selected = true
					}
					gr.autoZoom(true, graphCanvas, ui)
				}

				if keys.copy {
					for _, n := range gr.app.act.Subs {
						if n.Selected {
							keys.clipboard = n.Name //copy name into clipboard
							break
						}
					}
				}

				if keys.selectAll {
					for _, n := range gr.app.act.Subs {
						n.Selected = true
					}
				}
			}

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
					zoom := OsClampFloat(float64(gr.app.act.Cam_z)+float64(touch.wheel)*-0.1, 0.2, 2) //zoom += wheel
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
						cq, _ := n.nodeToPixelsCoord(lv.call.canvas, ui)
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

			if !gr.app.EnableExecution {
				ui.Paint_rect(0, 0, 1, 1, 0, pl.E, 0.03)
			}
		}
		ui.Div_end()

		//node list
		if gr.showNodeList {
			ui.Div_start(1, 0, 1, 1)
			{
				ui.Div_colMax(0, 100)

				ui.Div_start(0, 0, 1, 1)
				{
					//activate editbox
					if gr.showNodeList_justOpen {
						ui.edit.setFirstEditbox = true
						gr.showNodeList_justOpen = false
					}

					ui.Div_colMax(0, 100)
					ui.Comp_editbox(0, 0, 1, 1, &gr.node_search, 0, nil, ui.trns.SEARCH, gr.node_search != "", true, true)
					if ui.Comp_buttonText(1, 0, 1, 1, "X", "", ui.trns.CLOSE, true, false) > 0 {
						gr.showNodeList = false //hide
					}
				}
				ui.Div_end()

				y := 1
				search := strings.ToLower(gr.node_search)
				for _, n := range gr.app.act.Subs {
					if search == "" || strings.Contains(strings.ToLower(n.Name), search) {
						if ui.Comp_buttonMenu(0, y, 1, 1, n.Name, n.Exe, true, n.Selected) > 0 {
							n.SelectOnlyThis()
							gr.autoZoom(true, graphCanvas, ui)
						}
						y++
					}
				}

			}
			ui.Div_end()
		}
	}
	ui.Div_end()

	//panel
	ui.Div_start(1, 0, 1, 1)
	{
		ui.DivInfo_set(SA_DIV_SET_scrollVnarrow, 1, 0)
		ui.DivInfo_set(SA_DIV_SET_scrollHshow, 0, 0)

		ui.Div_rowMax(7, 100)

		path := "file:apps/base/resources/"

		y := 0
		if ui.Comp_buttonIcon(0, y, 1, 1, InitWinMedia_url(path+OsTrnString(gr.app.EnableExecution, "pause.png", "play.png")), 0.25, "Enable/Disable nodes execution", uint8(OsTrn(gr.app.EnableExecution, int(CdPalette_B), int(CdPalette_E))), true, false) > 0 {
			gr.app.EnableExecution = !gr.app.EnableExecution
			if gr.app.EnableExecution {
				gr.app.exeIt = true
			}
		}
		y++

		if ui.Comp_buttonIcon(0, y, 1, 1, InitWinMedia_url(path+"update.png"), 0.25, "Recompute all nodes", CdPalette_B, true, false) > 0 || (keyAllow && strings.EqualFold(keys.text, "h")) {
			gr.app.exeIt = true
		}
		y++

		if ui.Comp_buttonIcon(0, y, 1, 1, InitWinMedia_url(path+"home.png"), 0.3, "Zoom all nodes(H)", CdPalette_B, true, false) > 0 || (keyAllow && strings.EqualFold(keys.text, "h")) {
			gr.autoZoom(false, graphCanvas, ui) //zoom to all
		}
		y++

		if ui.Comp_buttonIcon(0, y, 1, 1, InitWinMedia_url(path+"home_select.png"), 0.2, "Zoom selected nodes(G)", CdPalette_B, true, false) > 0 || (keyAllow && strings.EqualFold(keys.text, "g")) {
			gr.autoZoom(true, graphCanvas, ui) //zoom to selected
		}
		y++

		if ui.Comp_buttonIcon(0, y, 1, 1, InitWinMedia_url(path+"hierarchy.png"), 0.25, "Reoder all nodes(L)", CdPalette_B, true, false) > 0 || (keyAllow && strings.EqualFold(keys.text, "l")) {
			gr.reorder(false, ui)               //reorder nodes
			gr.autoZoom(false, graphCanvas, ui) //zoom to all
		}
		y++

		if ui.Comp_buttonIcon(0, y, 1, 1, InitWinMedia_url(path+"hierarchy_select.png"), 0.2, "Reorder selected nodes(K)", CdPalette_B, true, false) > 0 || (keyAllow && strings.EqualFold(keys.text, "k")) {
			gr.reorder(true, ui)               //reorder only selected nodes
			gr.autoZoom(true, graphCanvas, ui) //zoom to selected
		}
		y++

		if ui.Comp_buttonIcon(0, y, 1, 1, InitWinMedia_url(path+"list.png"), 0.2, "Show list of all nodes(Ctrl+F)", CdPalette_P, true, gr.showNodeList) > 0 || strings.EqualFold(keys.ctrlChar, "f") {
			gr.showNodeList = !gr.showNodeList
			if gr.showNodeList {
				gr.showNodeList_justOpen = true
			}
		}
		y++

		y++ //space - adjust Div_rowMax()

		if gr.app.exe != nil {
			done := gr.app.exe.GetStatDone()
			ui.Comp_text(0, y, 1, 1, OsTrnString(done > 0, fmt.Sprintf("%.0f%%", done*100), "---"), 1)
		}

	}
	ui.Div_end()
}
