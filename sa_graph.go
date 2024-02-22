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
	"math/rand"
	"strings"
)

type SAGraph struct {
	app *SAApp

	cam_move           bool
	node_move          bool
	node_select        bool
	cam_start          OsV2f
	touch_start        OsV2
	node_move_selected SANodePath

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

	if !ui.edit.IsActive() {
		if ui.win.io.keys.tab && lvBaseDiv.IsOver(ui) {
			gr.app.canvas.addGrid = InitOsV4(0, 0, 1, 1)
			gr.app.canvas.addPos = gr.app.root.pixelsToNode(ui.win.io.touch.pos, ui, lvBaseDiv)
			gr.app.canvas.addnode_search = ""
			gr.app.canvas.addParent = NewSANodePath(gr.app.root.FindInsideParent(ui.win.io.touch.pos, lvBaseDiv.canvas, ui))

			ui.Dialog_open("nodes_list", 2)
		}
	}
}

func _SAGraph_drawConnectionV(start OsV2, end OsV2, active bool, cellr float32, ui *Ui, dash float32) {
	cd := Node_connectionCd(ui)
	if active {
		cd = SAApp_getYellow()
	}
	t := cellr * 0.3
	end.Y -= int(t) //connect to top of arrow
	mid := start.Aprox(end, 0.5)
	wi := ui.CellWidth(0.03)

	if start.Y < end.Y {
		ui.buff.AddBezier(start, OsV2{start.X, mid.Y}, OsV2{end.X, mid.Y}, end, cd, wi, dash)
	} else {
		mv := int(cellr * 2)

		end2 := mid
		end2.Y = start.Y
		ui.buff.AddBezier(start,
			OsV2{start.X, start.Y + mv},
			OsV2{end2.X, end2.Y + mv},
			end2, //mid
			cd, wi, dash)

		end3 := mid
		end3.Y = end.Y
		ui.buff.AddBezier(end,
			OsV2{end.X, end.Y - mv},
			OsV2{end3.X, end3.Y - mv},
			end3, //mid
			cd, wi, dash)

		ui.buff.AddLine(end2, end3, cd, wi)
	}

	//arrow
	ui.buff.AddPoly(end.Add(OsV2{int(-t / 2), 0}), []OsV2f{{0, 0}, {-t / 2, -t}, {t / 2, -t}}, cd, 0)
}

func _SAGraph_drawConnectionH(start OsV2, end OsV2, active bool, cellr float32, ui *Ui, dash float32) {
	cd := Node_connectionCd(ui)
	if active {
		cd = SAApp_getYellow()
	}
	t := cellr * 0.3
	end.X -= int(t) //connect to left of arrow
	mid := start.Aprox(end, 0.5)
	wi := ui.CellWidth(0.03)

	if start.X < end.X {
		ui.buff.AddBezier(start, OsV2{mid.X, start.Y}, OsV2{mid.X, end.Y}, end, cd, wi, dash)
	} else {
		mv := int(cellr * 2)

		end2 := mid
		end2.X = start.X
		ui.buff.AddBezier(start,
			OsV2{start.X + mv, start.Y},
			OsV2{end2.X + mv, end2.Y},
			end2, //mid
			cd, wi, dash)

		end3 := mid
		end3.X = end.X
		ui.buff.AddBezier(end,
			OsV2{end.X - mv, end.Y},
			OsV2{end3.X - mv, end3.Y},
			end3, //mid
			cd, wi, dash)

		ui.buff.AddLine(end2, end3, cd, wi)
	}

	//arrow
	ui.buff.AddPoly(end.Add(OsV2{0, int(-t / 2)}), []OsV2f{{0, 0}, {-t, -t / 2}, {-t, t / 2}}, cd, 0)
}

func _reorder_layer(nodes_layer []*SANode, max_width int) []float32 {

	best_poses := make([]float32, len(nodes_layer))
	best_score := float32(-1)

	ids := make([]int, max_width)
	for i := range ids {
		ids[i] = i
	}

	for ii := 0; ii < 1000; ii++ {
		//prepare random position
		for s := 0; s < len(ids); s++ {
			d := rand.Int() % len(ids)
			ids[s], ids[d] = ids[d], ids[s] //swap
		}

		//set random positions
		for i, n := range nodes_layer {
			n.Pos.X = float32(ids[i])
		}

		//get score
		score := float32(0)
		for _, n := range nodes_layer {
			score += n.GetDependDistance()
		}

		//update best
		if best_score < 0 || score < best_score {
			p := 0
			for _, n := range nodes_layer {
				best_poses[p] = n.Pos.X
				p++
			}

			best_score = score
			if score == 0 {
				break //1st layer
			}
		}
	}

	//set nodes with best_poses
	for i, n := range nodes_layer {
		n.Pos.X = float32(best_poses[i])
	}

	return best_poses
}

func (gr *SAGraph) reorder(onlySelected bool, ui *Ui) {
	x_jump := float32(6)
	y_jump := float32(6)

	//create list
	var nodes []*SANode
	for _, n := range gr.app.root.Subs {
		if onlySelected && !n.Selected {
			continue //skip
		}
		nodes = append(nodes, n)
	}

	//vertical
	{
		//reset
		for _, n := range nodes {
			n.sort_depth = 0
		}
		//update depth
		for _, n := range nodes {
			n.UpdateDepth(n)
		}
		//set posY
		for _, n := range nodes {
			n.Pos.Y = float32(n.sort_depth)
		}
	}
	//horizontal
	{
		//get num layers
		num_depth := 0
		for _, n := range nodes {
			num_depth = OsMax(num_depth, n.sort_depth+1)
		}

		max_width := 0
		for depth := 0; depth < num_depth; depth++ {
			width := 0
			for _, n := range nodes {
				if n.sort_depth == depth {
					width++
				}
			}
			if width > max_width {
				max_width = width
			}
		}

		var best_poses [][]float32
		best_score := float32(-1)
		for i := 0; i < 100; i++ {

			poses := make([][]float32, num_depth)

			for depth := 0; depth < num_depth; depth++ {

				var nodes_layer []*SANode
				for _, n := range nodes {
					if n.sort_depth == depth {
						nodes_layer = append(nodes_layer, n)
					}
				}

				poses[depth] = _reorder_layer(nodes_layer, max_width)
			}

			score := float32(0)
			for _, n := range nodes {
				score += n.GetDependDistance()
			}

			if best_score < 0 || score < best_score {
				best_poses = poses
				best_score = score
			}
		}

		//set final x poses
		for depth := 0; depth < num_depth; depth++ {
			p := 0
			for _, n := range nodes {
				if n.sort_depth == depth {
					n.Pos.X = best_poses[depth][p] * x_jump
					n.Pos.Y *= y_jump
					p++
				}
			}
		}
	}
}

func (gr *SAGraph) autoZoom(onlySelected bool, canvas OsV4, ui *Ui) {
	//zoom to all
	first := true
	var mn OsV2f
	var mx OsV2f
	num := 0
	for _, n := range gr.app.root.Subs {
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
	gr.app.Cam_x = float64(mn.X+mx.X) / 2
	gr.app.Cam_y = float64(mn.Y+mx.Y) / 2

	gr.app.Cam_z = 1
	for gr.app.Cam_z > 0.1 {
		areIn := true
		for _, n := range gr.app.root.Subs {
			if onlySelected && !n.Selected {
				continue //skip
			}

			_, cq, _ := n.nodeToPixelsCoord(canvas, ui)
			if !canvas.GetIntersect(cq).Cmp(cq) {
				areIn = false
				break
			}
		}
		if areIn {
			break
		}
		gr.app.Cam_z -= 0.05
	}
}

func (gr *SAGraph) buildNodes(node *SANode) []*SANode {
	var list []*SANode

	for _, nd := range node.Subs {
		list = append(list, nd)
		list = append(list, gr.buildNodes(nd)...)
	}
	return list
}

func (gr *SAGraph) drawConnections(nodes []*SANode, ui *Ui) {

	lv := ui.GetCall()
	cellr := gr.app.root.cellZoom(ui)

	for _, node := range nodes {

		//attributtes connection
		{
			coordNode, selCoordNode, _ := node.nodeToPixelsCoord(lv.call.canvas, ui)
			if node.Selected {
				coordNode.Start.X = selCoordNode.Start.X
				coordNode.Size.X = selCoordNode.Size.X
			}

			for _, in := range node.Attrs {
				for _, out := range in.depends {

					if out.node == node {
						continue
					}

					coordOut, selCoordOut, _ := out.node.nodeToPixelsCoord(lv.call.canvas, ui)
					if out.node.Selected {
						coordOut.Start.X = selCoordOut.Start.X
						coordOut.Size.X = selCoordOut.Size.X
					}

					var outPos OsV2
					outPos.X = coordOut.End().X
					outPos.Y += coordOut.Start.Y + int(cellr*(float32(out.node.VisiblePos(out))+0.5))

					var inPos OsV2
					inPos.X = coordNode.Start.X
					inPos.Y += coordNode.Start.Y + int(cellr*(float32(node.VisiblePos(in))+0.5))

					_SAGraph_drawConnectionH(outPos, inPos, node.Selected || out.node.Selected, cellr, ui, 0)
				}
			}
		}

		//setter connections
		if SAGroups_IsNodeSetter(node.Exe) {
			dstNode, _ := SAExe_Setter_destNode(node)
			if dstNode != nil {
				coordOut, selCoordOut, _ := node.nodeToPixelsCoord(lv.call.canvas, ui)
				coordIn, selCoordIn, _ := dstNode.nodeToPixelsCoord(lv.call.canvas, ui)

				if node.Selected {
					coordOut = selCoordOut
				}
				if dstNode.Selected {
					coordIn = selCoordIn
				}
				_SAGraph_drawConnectionV(OsV2{coordOut.Middle().X, coordOut.End().Y}, OsV2{coordIn.Middle().X, coordIn.Start.Y}, node.Selected || dstNode.Selected, cellr, ui, cellr)
			}
		}
	}
}

func (gr *SAGraph) drawNodes(nodes []*SANode, rects bool, classic bool, ui *Ui) *SANode {

	var touchInsideNode *SANode
	for _, n := range nodes {
		if SAGroups_HasNodeSub(n.Exe) {
			if rects {
				if n.drawRectNode(gr.node_select, gr.app) { //inside
					touchInsideNode = n
				}
			}
		} else {
			if classic {
				if n.drawNode(gr.node_select) { //inside
					touchInsideNode = n
				}
			}
		}
	}

	return touchInsideNode
}

func (gr *SAGraph) drawGraph(root *SANode, ui *Ui) (OsV4, bool) {
	if !ui.IsStackTop() {
		//reset
		gr.cam_move = false
		gr.node_move = false
		gr.node_select = false
	}

	nodes := gr.buildNodes(root)

	root.ResetIsRead()
	root.UpdateIsRead()

	pl := ui.win.io.GetPalette()

	touch := &ui.win.io.touch
	keys := &ui.win.io.keys
	var graphCanvas OsV4

	over := ui.GetCall().call.IsOver(ui)
	keyAllow := (over && !ui.edit.IsActive())

	graphCanvas = ui.GetCall().call.canvas

	changed := false

	ui.Div_colMax(0, 100)
	ui.Div_rowMax(0, 100)

	//background
	ui.Paint_rect(0, 0, 1, 1, 0, pl.GetGrey(0.8), 0)

	//grid
	{
		lvBaseDiv := ui.GetCall().call
		st := root.pixelsToNode(graphCanvas.Start, ui, lvBaseDiv)
		en := root.pixelsToNode(graphCanvas.End(), ui, lvBaseDiv)

		wi := ui.CellWidth(0.03)
		cd := InitOsCdWhite().SetAlpha(70)
		for x := int(st.X); x < int(en.X+1); x++ {
			s := root.nodeToPixels(OsV2f{float32(x), st.Y}, graphCanvas, ui)
			e := root.nodeToPixels(OsV2f{float32(x), en.Y}, graphCanvas, ui)
			ui.buff.AddLine(s, e, cd, wi)
		}
		for y := int(st.Y); y < int(en.Y+1); y++ {
			s := root.nodeToPixels(OsV2f{st.X, float32(y)}, graphCanvas, ui)
			e := root.nodeToPixels(OsV2f{en.X, float32(y)}, graphCanvas, ui)
			ui.buff.AddLine(s, e, cd, wi)
		}
	}

	//fade "press tab" bottom - middle in background
	lv := ui.GetCall()
	ui._compDrawText(lv.call.canvas.AddSpace(ui.CellWidth(0.5)), "press tab", "", pl.GetGrey(1), InitWinFontPropsDef(ui.win), false, false, OsV2{1, 2}, false, false)

	//+
	gr.drawCreateNode(ui)

	touchInsideNode := gr.drawNodes(nodes, true, false, ui)

	gr.drawConnections(nodes, ui)
	tin := gr.drawNodes(nodes, false, true, ui)
	if tin != nil {
		touchInsideNode = tin
	}

	if touch.rm {
		touchInsideNode = nil
	}

	//keys actions
	if keyAllow {
		//delete
		if keys.delete {
			gr.app.root.RemoveSelectedNodes()
			changed = true
		}

		//copy
		if keys.copy {
			//add selected into list
			gr.app.base.copiedNodes = gr.app.root.BuildListOfSelected()
		}

		//cut
		if keys.cut {
			//add selected into list
			gr.app.base.copiedNodes = gr.app.root.BuildListOfSelected()
			gr.app.root.RemoveSelectedNodes()
			changed = true
		}
		//paste
		if keys.paste {

			//add new nodes
			origNodes := gr.app.base.copiedNodes
			var newNodes []*SANode
			for _, src := range origNodes {

				path := NewSANodePath(src.parent)
				dstParent := path.FindPath(gr.app.root)
				if dstParent == nil {
					dstParent = gr.app.root
				}

				nw := dstParent.AddNodeCopy(src)
				newNodes = append(newNodes, nw)

				//add subs(rename expressions later)
				origNodes = append(origNodes, src.Subs...)
				newNodes = append(newNodes, nw.Subs...)
			}

			//rename expressions access to keep links between copied nodes
			for i := 0; i < len(newNodes); i++ {

				node := newNodes[i]
				node.ParseExpresions()

				for j := 0; j < len(newNodes); j++ {
					oldName := origNodes[j].Name
					newName := newNodes[j].Name

					node.RenameExpressionAccessNode(oldName, newName)
				}
			}

			//select and zoom
			gr.app.root.DeselectAll()
			for _, n := range newNodes {
				n.Selected = true
			}
			gr.autoZoom(true, graphCanvas, ui)
			changed = true
		}

		if keys.copy {
			for _, n := range nodes {
				if n.Selected {
					keys.clipboard = n.Name //copy name into clipboard
					break
				}
			}
		}

		if keys.selectAll {
			for _, n := range nodes {
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
			gr.app.root.SetPosStart() //ALL nodes(not only selected)
			gr.node_move_selected = NewSANodePath(touchInsideNode)

			//click on un-selected => de-select all & select only current
			if !touchInsideNode.Selected {
				for _, n := range nodes {
					n.Selected = false
				}
				touchInsideNode.Selected = true
			}
		}
		if gr.node_move {
			cell := ui.win.Cell()
			p := touch.pos.Sub(gr.touch_start)
			var r OsV2f
			r.X = float32(p.X) / float32(gr.app.Cam_z) / float32(cell)
			r.Y = float32(p.Y) / float32(gr.app.Cam_z) / float32(cell)

			for _, n := range nodes {
				if n.Selected {
					n.AddPos(r)
				}
			}
		}

		//zoom
		if over && touch.wheel != 0 {
			zoom := OsClampFloat(float64(gr.app.Cam_z)+float64(touch.wheel)*-0.1, 0.2, 2) //zoom += wheel
			gr.app.Cam_z = zoom

			touch.wheel = 0
		}

		//cam & selection
		if (touchInsideNode == nil || keys.shift || keys.ctrl) && over && touch.start {
			if touch.rm {
				//start camera move
				gr.cam_move = true
				gr.touch_start = touch.pos
				gr.cam_start = OsV2f{float32(gr.app.Cam_x), float32(gr.app.Cam_y)}
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
			r.X = float32(p.X) / float32(gr.app.Cam_z) / float32(cell)
			r.Y = float32(p.Y) / float32(gr.app.Cam_z) / float32(cell)

			gr.app.Cam_x = float64(gr.cam_start.Sub(r).X)
			gr.app.Cam_y = float64(gr.cam_start.Sub(r).Y)
		}

		if gr.node_select {
			coord := InitOsV4ab(gr.touch_start, touch.pos)
			coord.Size.X++
			coord.Size.Y++

			ui.buff.AddRect(coord, pl.P.SetAlpha(50), 0)     //back
			ui.buff.AddRect(coord, pl.P, ui.CellWidth(0.03)) //border

			//update
			for _, n := range nodes {
				_, _, sel := n.nodeToPixelsCoord(lv.call.canvas, ui)
				n.selected_cover = coord.HasCover(sel)
			}
		}

	}

	if touch.end {
		//when it's clicked on selected node, but it's not moved => select only this node
		if gr.node_move && gr.node_move_selected.Is() {
			if gr.touch_start.Distance(touch.pos) < float32(ui.win.Cell())/5 {
				for _, n := range nodes {
					n.Selected = false
				}

				sn := gr.node_move_selected.FindPath(gr.app.root)
				if sn != nil {
					sn.Selected = true
				}
			}
		}

		if gr.node_select {
			for _, n := range nodes {
				n.Selected = n.KeyProgessSelection(keys)
			}
		}

		gr.cam_move = false
		gr.node_move = false
		gr.node_select = false
	}

	if gr.app.exeState == SANode_STATE_RUNNING {
		ui.Paint_rect(0, 0, 1, 1, 0, pl.P, 0.06) //exe rect
	} else if !gr.app.EnableExecution {
		ui.Paint_rect(0, 0, 1, 1, 0, pl.E, 0.03)
	}

	if changed {
		gr.app.SetExecute()
	}

	return graphCanvas, keyAllow
}

func (gr *SAGraph) drawNodeList(graphCanvas OsV4, act *SANode, ui *Ui) {

	//activate editbox
	if gr.showNodeList_justOpen {
		ui.edit.setFirstEditbox = true
		gr.showNodeList_justOpen = false
	}

	//search
	ui.Div_colMax(0, 100)
	ui.Comp_editbox(0, 0, 1, 1, &gr.node_search, Comp_editboxProp().TempToValue(true).Ghost(ui.trns.SEARCH).Highlight(gr.node_search != ""))

	//items
	y := 1
	searches := strings.Split(strings.ToLower(gr.node_search), " ")
	for _, n := range act.Subs {
		if gr.node_search == "" || SAApp_IsSearchedName(n.Name, searches) {
			if ui.Comp_buttonMenu(0, y, 1, 1, n.Name, n.Exe, true, n.Selected) > 0 {
				n.SelectOnlyThis()
				gr.autoZoom(true, graphCanvas, ui)
			}
			y++
		}
	}
}

func (gr *SAGraph) drawPanel(graphCanvas OsV4, keyAllow bool, ui *Ui) {

	keys := &ui.win.io.keys

	ui.DivInfo_set(SA_DIV_SET_scrollVnarrow, 1, 0)
	ui.DivInfo_set(SA_DIV_SET_scrollHshow, 0, 0)

	ui.Div_rowMax(7, 100)

	path := "file:apps/base/resources/"

	y := 0
	if ui.Comp_buttonIcon(0, y, 1, 1, InitWinMedia_url(path+OsTrnString(gr.app.EnableExecution, "pause.png", "play.png")), 0.25, "Enable/Disable nodes execution", uint8(OsTrn(gr.app.EnableExecution, int(CdPalette_B), int(CdPalette_E))), true, false) > 0 {
		gr.app.EnableExecution = !gr.app.EnableExecution
		if gr.app.EnableExecution {
			gr.app.SetExecute()
		}
	}
	y++

	if ui.Comp_buttonIcon(0, y, 1, 1, InitWinMedia_url(path+"update.png"), 0.25, "Recompute all nodes", CdPalette_B, true, false) > 0 || (keyAllow && strings.EqualFold(keys.text, "h")) {
		gr.app.SetExecute()
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

	if ui.Comp_buttonIcon(0, y, 1, 1, InitWinMedia_url(path+"list.png"), 0.2, "Show/Hide list of all nodes(Ctrl+F)", CdPalette_P, true, gr.showNodeList) > 0 || strings.EqualFold(keys.ctrlChar, "f") {
		gr.showNodeList = !gr.showNodeList
		if gr.showNodeList {
			gr.showNodeList_justOpen = true
		}
	}
	y++

	y++ //space - adjust Div_rowMax()

	if ui.Comp_buttonIcon(0, y, 1, 1, InitWinMedia_url(path+"code.png"), 0.2, "Show/Hide code", CdPalette_P, true, gr.app.ShowCode) > 0 || strings.EqualFold(keys.ctrlChar, "t") {
		gr.app.ShowCode = !gr.app.ShowCode
	}
	y++

	//if gr.app.exeState == SANode_STATE_RUNNING {
	//ui.Comp_text(0, y, 1, 1, fmt.Sprintf("**%d**", gr.app.jobs.Num()), 1)
	//ui.Comp_text(0, y, 1, 1, OsTrnString(done > 0, fmt.Sprintf("%.0f%%", done*100), "---"), 1)
	//}
}
