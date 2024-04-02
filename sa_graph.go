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
	"bytes"
	"encoding/json"
	"fmt"
	"math"
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

	//connect_in  *SANode
	//connect_out *SANode

	showNodeList          bool
	showNodeList_justOpen bool
	node_search           string

	history     [][]byte //JSONs
	history_pos int
}

func NewSAGraph(app *SAApp) *SAGraph {
	var gr SAGraph
	gr.app = app
	return &gr
}

/*func (gr *SAGraph) isConnecting() bool {
	return gr.connect_in != nil || gr.connect_out != nil
}

func (gr *SAGraph) resetConnect(tryResetIn bool) {
	if tryResetIn && gr.connect_in != nil {
		gr.connect_in.Code.Triggers = nil
	}

	gr.connect_in = nil
	gr.connect_out = nil
}

func (gr *SAGraph) tryConnect() {
	if gr.connect_out != nil && gr.connect_in != nil {
		gr.connect_in.Code.addTrigger(gr.connect_out.Name)
		gr.resetConnect(false)
	}
}

func (gr *SAGraph) SetConnectIn(attr *SANode) {
	gr.connect_in = attr
	gr.tryConnect()
}
func (gr *SAGraph) SetConnectOut(attr *SANode) {
	gr.connect_out = attr
	gr.tryConnect()
}*/

func (gr *SAGraph) drawCreateNode() {
	ui := gr.app.base.ui
	lvBaseDiv := ui.GetCall().call

	if !ui.edit.IsActive() {
		if ui.win.io.keys.tab && lvBaseDiv.IsOver(ui) {
			gr.app.canvas.addGrid = InitOsV4(0, 0, 1, 1)
			gr.app.canvas.addPos = gr.app.root.pixelsToNode(ui.win.io.touch.pos, lvBaseDiv)
			gr.app.canvas.addnode_search = ""
			gr.app.canvas.addParent = NewSANodePath(gr.app.root.FindInsideParent(ui.win.io.touch.pos, lvBaseDiv.canvas))

			ui.Dialog_open("nodes_list", 2)
		}
	}
}

/*func (gr *SAGraph) drawConnectionV(start OsV2, end OsV2, cellr float32, dash float32, cd OsCd) {
	ui := gr.app.base.ui
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
}*/

func (gr *SAGraph) drawConnectionDirect(startRect OsV4, endRect OsV4, dash float32, selectedCd bool, move float32, cellr float32, depend_write *bool, name string) {

	//H-H nebo V-V

	stMid := startRect.Middle()
	stStart := startRect.Start
	stEnd := startRect.End()

	enMid := endRect.Middle()
	enStart := endRect.Start
	enEnd := endRect.End()

	var start, end OsV2
	isV := false

	v := endRect.Middle().Sub(startRect.Middle()) //opačně? ...
	rad := v.Angle()

	if rad <= math.Pi/4 && rad >= -math.Pi/4 {
		//right
		start = OsV2{stEnd.X, stMid.Y}
		end = OsV2{enStart.X, enMid.Y}
	}
	if rad > math.Pi/4 && rad < math.Pi*3/4 {
		//down
		isV = true
		start = OsV2{stMid.X, stEnd.Y}
		end = OsV2{enMid.X, enStart.Y}
	}
	if rad < -math.Pi/4 && rad >= -math.Pi*3/4 {
		//up
		isV = true
		start = OsV2{stMid.X, stStart.Y}
		end = OsV2{enMid.X, enEnd.Y}
	}
	if rad > math.Pi*3/4 || rad < -math.Pi*3/4 {
		//left
		start = OsV2{stStart.X, stMid.Y}
		end = OsV2{enEnd.X, enMid.Y}
	}

	ui := gr.app.base.ui
	wi := ui.CellWidth(0.03)

	mm := start.Aprox(end, 0.5)
	var mS, mE OsV2
	if isV {
		mS = OsV2{start.X, mm.Y}
		mE = OsV2{end.X, mm.Y}
	} else {
		mS = OsV2{mm.X, start.Y}
		mE = OsV2{mm.X, end.Y}
	}

	cd := Node_connectionCd(selectedCd, ui)
	ui.buff.AddBezier(start, mS, mE, end, cd, wi, dash, move)

	// arrow
	/*{
		//poly
		t := cellr * 0.3
		ppts := []OsV2f{{0, 0}, {-t / 2, t}, {t / 2, t}}
		poly := ui.buff.GetPoly(ppts, 0)

		//quad
		top_mid, dir := ui.win.GetBezier(start, mS, mE, end, OsTrnFloat(*depend_write, 0.1, 0.9))
		dir = dir.MulV(1 / dir.Len() * t) //normalize
		if !*depend_write {
			dir = OsV2f{-dir.X, -dir.Y}
		}

		bot_mid := top_mid.Add(dir)
		rev := OsV2f{-dir.Y / 2, dir.X / 2}

		pts := [4]OsV2f{top_mid.Add(rev), top_mid.Sub(rev), bot_mid.Sub(rev), bot_mid.Add(rev)}
		uvs := [4]OsV2f{{0, 0}, {1, 0}, {1, 1}, {0, 1}}

		if *depend_write && !selectedCd {
			pl := ui.win.io.GetPalette()
			cd = pl.P
		}

		ui.buff.AddPolyQuad(pts, uvs, poly, cd)
		//ui.buff.AddCircle(InitOsV4Mid(top_mid.toV2(), OsV2{int(cellr * 0.2), int(cellr * 0.2)}), cd, 0)
	}

	//button
	bck := ui.win.io.ini.Dpi
	ui.win.io.ini.Dpi = int(float32(ui.win.io.ini.Dpi)*float32(gr.app.Cam_z)) / 2
	{
		mid, _ := ui.win.GetBezier(start, mS, mE, end, 0.5)

		sz := int(cellr * 0.6)
		cq := InitOsV4Mid(mid.toV2(), OsV2{sz, sz})

		ui.Div_startCoord(0, 0, 1, 1, cq, name)
		ui.Div_col(0, 0.1)
		ui.Div_row(0, 0.1)
		ui.Div_colMax(0, 100)
		ui.Div_rowMax(0, 100)

		ui.DivInfo_set(SA_DIV_SET_scrollHshow, 0, 0)
		ui.DivInfo_set(SA_DIV_SET_scrollVshow, 0, 0)
		if ui.Comp_button(0, 0, 1, 1, "↔", Comp_buttonProp().Shape(1)) > 0 {
			*depend_write = !*depend_write
		}
		ui.Div_end()
	}
	ui.win.io.ini.Dpi = bck*/

}

/*func (gr *SAGraph) drawConnectionTrigger(start OsV2, end OsV2, cellr float32, dash float32, cd OsCd) {
	ui := gr.app.base.ui
	t := cellr * 0.3
	end.X -= int(t) //connect to left of arrow
	mid := start.Aprox(end, 0.5)
	wi := ui.CellWidth(0.03)

	if start.X < end.X {
		ui.buff.AddBezier(start, OsV2{mid.X, start.Y}, OsV2{mid.X, end.Y}, end, cd, wi, dash, 0)
	} else {
		mv := int(cellr * 2)

		end2 := mid
		end2.X = start.X
		ui.buff.AddBezier(start,
			OsV2{start.X + mv, start.Y},
			OsV2{end2.X + mv, end2.Y},
			end2, //mid
			cd, wi, dash, 0)

		end3 := mid
		end3.X = end.X
		ui.buff.AddBezier(end,
			OsV2{end.X - mv, end.Y},
			OsV2{end3.X - mv, end3.Y},
			end3, //mid
			cd, wi, dash, 0)

		ui.buff.AddLine(end2, end3, cd, wi)
	}

	//arrow
	ui.buff.AddPoly(end.Add(OsV2{0, int(-t / 2)}), []OsV2f{{0, 0}, {-t, -t / 2}, {-t, t / 2}}, cd, 0)
}*/

func (gr *SAGraph) autoZoom(onlySelected bool, canvas OsV4) {
	gr.app.rebuildLists()

	//zoom to all
	first := true
	var mn OsV2f
	var mx OsV2f
	num := 0

	//list
	nodes := gr.app.all_nodes
	if onlySelected {
		nodes = gr.app.selected_nodes
	}

	for _, n := range nodes {
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
		for _, n := range nodes {
			_, cq, _ := n.nodeToPixelsCoord(canvas)
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

func (gr *SAGraph) drawConnections() {

	ui := gr.app.base.ui
	lv := ui.GetCall()
	cellr := gr.app.root.cellZoom(ui)

	for _, out := range gr.app.all_nodes {

		coordOut, selCoordOut, _ := out.nodeToPixelsCoord(lv.call.canvas)
		if out.Selected {
			coordOut = selCoordOut
		}

		//attributtes connection
		for i, in := range out.Code.func_depends {
			if in == out {
				continue
			}

			coordIn, selCoordIn, _ := in.nodeToPixelsCoord(lv.call.canvas)
			if in.Selected {
				coordIn = selCoordIn
			}

			//gr.drawConnectionDirect(OsV2{coordIn.Middle().X, coordIn.End().Y}, OsV2{coordOut.Middle().X, coordOut.Start.Y}, cellr, 0, Node_connectionCd(in.Selected || out.Selected, ui))
			gr.drawConnectionDirect(coordOut, coordIn, 0, in.Selected || out.Selected, 0, cellr, &out.Code.GetArg(in.Name).Write, fmt.Sprintf("%s_%d", in.Name, i))
		}

		/*cellr := gr.app.root.cellZoom(ui)
		for _, inName := range out.Code.Triggers {

			in := out.FindNode(inName)
			if in == nil {
				continue
			}
			if in == out {
				continue
			}

			coordIn, selCoordIn, _ := in.nodeToPixelsCoord(lv.call.canvas)
			if in.Selected {
				coordIn = selCoordIn
			}

			//coordOut = selCoordOut //move by button_circle_rad
			//gr.drawConnectionTrigger(OsV2{selCoordIn.End().X, selCoordIn.Middle().Y}, OsV2{coordOut.Start.X, coordOut.Middle().Y}, cellr, cellr, Node_connectionCd(in.Selected || out.Selected, ui))
			gr.drawConnectionDirect(coordOut, coordIn, cellr, Node_connectionCd(in.Selected || out.Selected, ui), cellr/10)
		}*/
	}
}

func (gr *SAGraph) drawNodes(rects bool, classic bool) *SANode {

	var touchInsideNode *SANode
	for _, n := range gr.app.all_nodes {
		if n.HasNodeSubs() {
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

func (gr *SAGraph) drawGraph(root *SANode) (OsV4, bool) {
	ui := gr.app.base.ui
	pl := ui.win.io.GetPalette()

	touch := &ui.win.io.touch
	keys := &ui.win.io.keys
	var graphCanvas OsV4

	over := ui.GetCall().call.IsOver(ui)
	keyAllow := (over && !ui.edit.IsActive() && ui.GetCall().call.enableInput)

	graphCanvas = ui.GetCall().call.canvas

	ui.Div_colMax(0, 100)
	ui.Div_rowMax(0, 100)

	//background
	ui.Paint_rect(0, 0, 1, 1, 0, pl.GetGrey(0.8), 0)

	//grid
	{
		lvBaseDiv := ui.GetCall().call
		st := root.pixelsToNode(graphCanvas.Start, lvBaseDiv)
		en := root.pixelsToNode(graphCanvas.End(), lvBaseDiv)

		wi := ui.CellWidth(0.03)
		cd := InitOsCdWhite().SetAlpha(70)
		for x := int(st.X); x < int(en.X+1); x++ {
			s := root.nodeToPixels(OsV2f{float32(x), st.Y}, graphCanvas)
			e := root.nodeToPixels(OsV2f{float32(x), en.Y}, graphCanvas)
			ui.buff.AddLine(s, e, cd, wi)
		}
		for y := int(st.Y); y < int(en.Y+1); y++ {
			s := root.nodeToPixels(OsV2f{st.X, float32(y)}, graphCanvas)
			e := root.nodeToPixels(OsV2f{en.X, float32(y)}, graphCanvas)
			ui.buff.AddLine(s, e, cd, wi)
		}
	}

	//fade "press tab" bottom - middle in background
	lv := ui.GetCall()
	ui._compDrawText(lv.call.canvas.AddSpace(ui.CellWidth(0.5)), "press tab", "", pl.GetGrey(1), InitWinFontPropsDef(ui.win), false, false, OsV2{1, 2}, false, false)

	//+
	gr.drawCreateNode()

	touchInsideNode := gr.drawNodes(true, false)

	gr.drawConnections()
	tin := gr.drawNodes(false, true)
	if tin != nil {
		touchInsideNode = tin
	}
	if touch.rm {
		touchInsideNode = nil
	}

	//making connection
	/*{
		cellr := gr.app.root.cellZoom(ui)
		cd := pl.P

		if gr.connect_out != nil {
			coordIn, selCoordIn, _ := gr.connect_out.nodeToPixelsCoord(lv.call.canvas)
			if gr.connect_out.Selected {
				coordIn = selCoordIn
			}
			gr.drawConnectionTrigger(OsV2{coordIn.End().X, coordIn.Middle().Y}, ui.win.io.touch.pos, cellr, 0, cd)
		}

		if gr.connect_in != nil {
			coordOut, selCoordOut, _ := gr.connect_in.nodeToPixelsCoord(lv.call.canvas)
			if gr.connect_in.Selected {
				coordOut = selCoordOut
			}
			gr.drawConnectionTrigger(ui.win.io.touch.pos, OsV2{coordOut.Start.X, coordOut.Middle().Y}, cellr, 0, cd)
		}
	}*/

	//must be below dialog!
	if !ui.GetCall().call.enableInput {
		//reset
		gr.cam_move = false
		gr.node_move = false
		gr.node_select = false
	}

	//keys actions
	if keyAllow {
		//reset connecting
		/*if keys.esc {
			gr.resetConnect(false)
		}*/

		//delete
		if keys.delete {
			gr.app.root.RemoveSelectedNodes()
		}

		//delete
		if keys.text == "b" {
			gr.app.root.BypassSelectedCodeNodes()
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
				//node.ParseExpresions()

				for j := 0; j < len(newNodes); j++ {
					oldName := origNodes[j].Name
					newName := newNodes[j].Name

					node.RenameDepends(oldName, newName)
				}
			}

			//select and zoom
			gr.app.root.DeselectAll()
			for _, n := range newNodes {
				n.Selected = true
			}
			gr.autoZoom(true, graphCanvas)
		}

		if keys.copy {
			for _, n := range gr.app.selected_nodes {
				keys.clipboard = n.Name //copy name into clipboard
				break
			}
		}

		if keys.selectAll {
			for _, n := range gr.app.all_nodes {
				n.Selected = true
			}
		}
	}

	//touch actions
	{
		//nodes
		if touchInsideNode != nil && over && touch.start && !keys.shift && !keys.ctrl /*&& !gr.isConnecting()*/ {
			gr.node_move = true
			gr.touch_start = touch.pos
			gr.app.root.SetPosStart() //ALL nodes(not only selected)
			gr.node_move_selected = NewSANodePath(touchInsideNode)

			//click on un-selected => de-select all & select only current
			if !touchInsideNode.Selected {
				touchInsideNode.SelectOnlyThis()
			}
		}
		if gr.node_move {
			cell := ui.win.Cell()
			p := touch.pos.Sub(gr.touch_start)
			var r OsV2f
			r.X = float32(p.X) / float32(gr.app.Cam_z) / float32(cell)
			r.Y = float32(p.Y) / float32(gr.app.Cam_z) / float32(cell)

			for _, n := range gr.app.selected_nodes {
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

				//gr.resetConnect(true)
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
			for _, n := range gr.app.all_nodes {
				_, _, sel := n.nodeToPixelsCoord(lv.call.canvas)
				n.selected_cover = coord.HasCover(sel)
			}
		}

	}

	if touch.end {
		//when it's clicked on selected node, but it's not moved => select only this node
		if gr.node_move && gr.node_move_selected.Is() {
			if gr.touch_start.Distance(touch.pos) < float32(ui.win.Cell())/5 {
				for _, n := range gr.app.all_nodes {
					n.Selected = false
				}

				sn := gr.node_move_selected.FindPath(gr.app.root)
				if sn != nil {
					sn.Selected = true
				}
			}
		}

		if gr.node_select {
			for _, n := range gr.app.all_nodes {
				n.Selected = n.KeyProgessSelection(keys)
			}
		}

		if over && touch.numClicks > 1 {
			sel := gr.app.root.FindSelected()
			if sel == nil {
				gr.app.Selected_canvas = SANodePath{}
			} else {
				gr.app.Selected_canvas = NewSANodePath(sel)
			}
		}

		gr.cam_move = false
		gr.node_move = false
		gr.node_select = false
	}

	//if gr.app.exeState == SANode_STATE_RUNNING {
	//	ui.Paint_rect(0, 0, 1, 1, 0, pl.P, 0.06) //exe rect
	//} else if !gr.app.EnableExecution {
	//	ui.Paint_rect(0, 0, 1, 1, 0, pl.E, 0.03)
	//}

	return graphCanvas, keyAllow
}

func (gr *SAGraph) drawNodeList(graphCanvas OsV4) {
	ui := gr.app.base.ui

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
	for _, n := range gr.app.all_nodes {
		if gr.node_search == "" || OsIsSearchedName(n.Name, searches) {
			if ui.Comp_buttonMenu(0, y, 1, 1, n.Name, n.Selected, Comp_buttonProp().Tooltip(n.Exe)) > 0 {
				n.SelectOnlyThis()
				gr.autoZoom(true, graphCanvas)
			}
			y++
		}
	}
}

func (gr *SAGraph) drawPanel(graphCanvas OsV4, keyAllow bool) {

	ui := gr.app.base.ui
	keys := &ui.win.io.keys

	ui.DivInfo_set(SA_DIV_SET_scrollVnarrow, 1, 0)
	ui.DivInfo_set(SA_DIV_SET_scrollHshow, 0, 0)

	ui.Div_colMax(3, 100)

	path := "file:apps/base/resources/"

	//if ui.Comp_buttonIcon(0, y, 1, 1, InitWinMedia_url(path+OsTrnString(gr.app.EnableExecution, "pause.png", "play.png")), 0.25, "Enable/Disable nodes execution", uint8(OsTrn(gr.app.EnableExecution, int(CdPalette_B), int(CdPalette_E))), true, false) > 0 {
	//	gr.app.EnableExecution = !gr.app.EnableExecution
	//	if gr.app.EnableExecution {
	//		gr.app.SetExecute()
	//	}
	//}
	//y++

	//if ui.Comp_buttonIcon(0, y, 1, 1, InitWinMedia_url(path+"update.png"), 0.25, "Recompute all nodes", CdPalette_B, true, false) > 0 || (keyAllow && strings.EqualFold(keys.text, "h")) {
	//	gr.app.SetExecute()
	//}
	//y++

	if ui.Comp_buttonLight(0, 0, 1, 1, "←", Comp_buttonProp().Enable(gr.canHistoryBack()).Tooltip(fmt.Sprintf("%s(%d)", ui.trns.BACKWARD, gr.history_pos))) > 0 {
		gr.stepHistoryBack()

	}

	if ui.Comp_buttonLight(1, 0, 1, 1, "→", Comp_buttonProp().Enable(gr.canHistoryForward()).Tooltip(fmt.Sprintf("%s(%d)", ui.trns.FORWARD, len(gr.history)-gr.history_pos-1))) > 0 {
		gr.stepHistoryForward()
	}

	//progress
	{
		progressStr, progressProc := gr.app.base.jobs.FindAppProgress(gr.app)
		if progressProc >= 0 {
			ui.Div_start(3, 0, 1, 1)
			{
				dnm := "progress"
				ui.Div_colMax(0, 100)
				if ui.Comp_button(0, 0, 1, 1, fmt.Sprintf("%s ... %.1f%%", progressStr, progressProc*100), Comp_buttonProp()) > 0 {
					ui.Dialog_open(dnm, 0)
				}
				if ui.Dialog_start(dnm) {
					if !gr.app.base.jobs.RenderAppProgress(gr.app) {
						ui.Dialog_close()
					}
					ui.Dialog_end()
				}

				ui.win.SetRedraw()
				//time.Sleep(500 * time.Millisecond)	//.....
			}
			ui.Div_end()
		}
	}

	if ui.Comp_buttonIcon(5, 0, 1, 1, InitWinMedia_url(path+"home.png"), 0.3, "Zoom all nodes(H)", Comp_buttonProp().Cd(CdPalette_B)) > 0 || (keyAllow && strings.EqualFold(keys.text, "h")) {
		gr.autoZoom(false, graphCanvas) //zoom to all
	}

	if ui.Comp_buttonIcon(6, 0, 1, 1, InitWinMedia_url(path+"home_select.png"), 0.2, "Zoom selected nodes(G)", Comp_buttonProp().Cd(CdPalette_B)) > 0 || (keyAllow && strings.EqualFold(keys.text, "g")) {
		gr.autoZoom(true, graphCanvas) //zoom to selected
	}

	//if ui.Comp_buttonIcon(0, y, 1, 1, InitWinMedia_url(path+"hierarchy.png"), 0.25, "Reoder all nodes(L)", Comp_buttonProp().Cd(CdPalette_B)) > 0 || (keyAllow && strings.EqualFold(keys.text, "l")) {
	//	gr.reorder(false)               //reorder nodes
	//	gr.autoZoom(false, graphCanvas) //zoom to all
	//}
	//y++

	//if ui.Comp_buttonIcon(0, y, 1, 1, InitWinMedia_url(path+"hierarchy_select.png"), 0.2, "Reorder selected nodes(K)", Comp_buttonProp().Cd(CdPalette_B)) > 0 || (keyAllow && strings.EqualFold(keys.text, "k")) {
	//	gr.reorder(true)               //reorder only selected nodes
	//	gr.autoZoom(true, graphCanvas) //zoom to selected
	//}
	//y++

	if ui.Comp_buttonIcon(7, 0, 1, 1, InitWinMedia_url(path+"list.png"), 0.2, "Show/Hide list of all nodes(Ctrl+F)", Comp_buttonProp().DrawBack(gr.showNodeList)) > 0 || strings.EqualFold(keys.ctrlChar, "f") {
		gr.showNodeList = !gr.showNodeList
		if gr.showNodeList {
			gr.showNodeList_justOpen = true
		}
	}

}

func (gr *SAGraph) History() {

	ui := gr.app.base.ui

	if len(gr.history) == 0 {
		gr.checkAndAddHistory()
		return
	}

	lv := ui.GetCall()
	if !ui.edit.IsActive() {
		if lv.call.IsOver(ui) && ui.win.io.keys.backward {
			gr.stepHistoryBack()

		}
		if lv.call.IsOver(ui) && ui.win.io.keys.forward {
			gr.stepHistoryForward()
		}
	}
}

func (gr *SAGraph) compare() ([]byte, bool) {

	js, err := json.Marshal(gr.app.root)
	if err != nil {
		return nil, false
	}
	if len(gr.history) > 0 && bytes.Equal(js, gr.history[gr.history_pos]) {
		return js, false //same as current history
	}
	return js, true
}

func (gr *SAGraph) checkAndAddHistory() {

	js, changed := gr.compare()
	if !changed {
		return //same as current history
	}

	//cut newer history
	if gr.history_pos+1 < len(gr.history) {
		gr.history = gr.history[:gr.history_pos+1]
	}

	gr.history = append(gr.history, js)
	gr.history_pos = len(gr.history) - 1
}

func (gr *SAGraph) recoverHistory() {
	dst, _ := NewSANodeRoot("", gr.app)
	err := json.Unmarshal(gr.history[gr.history_pos], dst)
	if err != nil {
		return
	}
	dst.updateLinks(nil, gr.app)
	dst.updateCodeLinks()
	gr.app.root = dst
}

func (gr *SAGraph) canHistoryBack() bool {
	return gr.history_pos > 0
}
func (gr *SAGraph) canHistoryForward() bool {
	return gr.history_pos+1 < len(gr.history)
}

func (gr *SAGraph) stepHistoryBack() bool {
	if !gr.canHistoryBack() {
		return false
	}

	gr.history_pos--
	gr.recoverHistory()
	return true
}
func (gr *SAGraph) stepHistoryForward() bool {

	if !gr.canHistoryForward() {
		return false
	}

	gr.history_pos++
	gr.recoverHistory()
	return true
}
