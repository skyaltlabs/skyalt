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
	"math/rand"
	"strconv"
	"strings"

	"github.com/go-audio/audio"
)

type SACanvas struct { // widgets
	addGrid   OsV4
	addPos    OsV2f
	addParent SANodePath

	startClick    SANodePath
	startClickRel OsV2

	resize SANodePath

	addnode_search string
}

type SAApp struct {
	base *SABase

	Name string
	IDE  bool

	Cam_x, Cam_y, Cam_z float64 `json:",omitempty"`
	EnableExecution     bool
	ExePos              int

	Selected_canvas SANodePath

	root *SANode
	exe  *SANode //root.exe

	graph  *SAGraph
	canvas SACanvas

	mic_nodes []SANodePath

	all_nodes      []*SANode
	selected_nodes []*SANode

	last_trigger_ticks int64
}

func (a *SAApp) init(base *SABase) {
	a.base = base

	a.graph = NewSAGraph(a)
}

func NewSAApp(name string, base *SABase) *SAApp {
	app := &SAApp{}
	app.Name = name
	app.IDE = true
	app.EnableExecution = true

	app.init(base)
	return app
}
func (app *SAApp) Destroy() {
}

func SAApp_GetNewFolderPath(name string) string {
	return "apps/" + name + "/"
}

func (app *SAApp) GetFolderPath() string {
	return SAApp_GetNewFolderPath(app.Name)
}

func (app *SAApp) GetJsonPath() string {
	return app.GetFolderPath() + "app.json"
}

func (app *SAApp) AddMicNode(nodePath SANodePath) {
	app.mic_nodes = append(app.mic_nodes, nodePath)
}
func (app *SAApp) RemoveMicNode(nodePath SANodePath) bool {
	for i, pt := range app.mic_nodes {
		if pt.Cmp(nodePath) {
			app.mic_nodes = append(app.mic_nodes[:i], app.mic_nodes[i+1:]...) //remove
			return true
		}
	}
	return false
}
func (app *SAApp) IsMicNodeRecording(nodePath SANodePath) bool {
	for _, pt := range app.mic_nodes {
		if pt.Cmp(nodePath) {
			return true
		}
	}
	return false
}
func (app *SAApp) AddMic(data audio.IntBuffer) {
	for i := len(app.mic_nodes) - 1; i >= 0; i-- {
		nd := app.mic_nodes[i].FindPath(app.root)
		if nd != nil {
			nd.temp_mic_data.SourceBitDepth = data.SourceBitDepth
			nd.temp_mic_data.Format = data.Format
			nd.temp_mic_data.Data = append(nd.temp_mic_data.Data, data.Data...)

		} else {
			//un-register
			app.mic_nodes = append(app.mic_nodes[:i], app.mic_nodes[i+1:]...) //remove
		}
	}
}

func (app *SAApp) buildNodes(node *SANode, onlySelected bool) []*SANode {
	var list []*SANode

	for _, nd := range node.Subs {

		if !onlySelected || nd.Selected {
			list = append(list, nd)
		}
		list = append(list, app.buildNodes(nd, onlySelected)...)
	}
	return list
}

func (app *SAApp) rebuildLists() {
	app.all_nodes = app.buildNodes(app.root, false)
	app.selected_nodes = app.buildNodes(app.root, true)
}

func (app *SAApp) TryExecute() {
	ui := app.base.ui

	if !app.EnableExecution {
		app.ExePos = -1
		return
	}

	if !ui.win.io.touch.end && (ui.edit.IsActive() || !ui.win.io.keys.hasChanged) && (app.last_trigger_ticks > 0 && OsIsTicksIn(app.last_trigger_ticks, 500)) {
		return
	}

	if app.ExePos < 0 {
		for _, nd := range app.exe.Subs {
			if len(nd.Code.exes) > 0 {
				app.ExePos = 0
				break
			}
		}
	}

	_, progressProc := app.base.jobs.FindAppProgress(app)
	if progressProc >= 0 {
		return //running
	}

	//update "changed" for sqlite dbs
	for _, nd := range app.all_nodes {
		if nd.IsTypeDbFile() {
			path := nd.GetAttrString("path", "")
			db, _, err := app.base.ui.win.disk.OpenDb(path)
			if err == nil {
				tm := db.GetTime()
				if !tm.Cmp(&nd.db_time) {
					fmt.Printf("Db '%s' has changed\n", nd.Name)
					nd.db_time = tm
					nd.SetChange(nil)
				}
			}
		}
	}

	if app.ExePos < 0 {
		return
	}

	//reset
	if app.ExePos == 0 {
		for _, nd := range app.all_nodes {
			if nd.IsTypeList() {
				nd.listSubs = nil
			}
		}
	}

	if app.ExePos < len(app.exe.Subs) {
		nd := app.exe.Subs[app.ExePos]

		var exe_prms []SANodeCodeExePrm
		if len(nd.Code.exes) > 0 {
			exe_prms = nd.Code.exes[0].prms
			nd.Code.exes = nd.Code.exes[1:] //remove
		}

		nd.Code.Execute(exe_prms)
		app.last_trigger_ticks = 0 //test for new changes immidiatly

		app.ExePos++
	} else {
		app.ExePos = -1 //off
	}
}

func SAApp_getYellow() OsCd {
	return OsCd{204, 204, 0, 255} //...
}

func (app *SAApp) checkSelectedCanvas() *SANode {
	node := app.Selected_canvas.FindPath(app.root)
	if node == nil {
		node = app.root
	}
	for node != nil && !node.HasNodeSubs() {
		node = node.parent
	}
	app.Selected_canvas = NewSANodePath(node)

	return node
}

func (app *SAApp) RenderApp() {
	app.root.renderLayout()
}

func (app *SAApp) renderAppWithColsRows() {

	ui := app.base.ui

	ui.Div_colMax(1, 100)
	ui.Div_rowMax(1, 100)

	var colDiv *UiLayoutDiv
	var rowDiv *UiLayoutDiv

	node := app.checkSelectedCanvas()

	//app
	var appID float64
	appDiv := ui.Div_start(1, 1, 1, 1)
	{
		appID = ui.DivInfo_get(SA_DIV_GET_uid, 0)

		node.renderLayout()
	}
	ui.Div_end()

	//size
	gridMax := appDiv.GetGridMax(OsV2{1, 1}) //app size
	gridMax.X = OsMax(gridMax.X, SANodeColRow_GetMaxPos(&node.Cols)+1)
	gridMax.Y = OsMax(gridMax.Y, SANodeColRow_GetMaxPos(&node.Rows)+1)

	//+
	if ui.Comp_buttonLight(0, 0, 1, 1, "+", Comp_buttonProp().Tooltip(ui.trns.ADD_COLUMNS_ROWS)) > 0 {
		ui.Dialog_open("add_cols_rows", 1)
	}
	if ui.Dialog_start("add_cols_rows") {
		ui.Div_colMax(0, 4)
		if ui.Comp_buttonMenu(0, 0, 1, 1, ui.trns.ADD_NEW_COLUMN, false, Comp_buttonProp()) > 0 {
			SANodeColRow_Insert(&node.Cols, nil, gridMax.X, true)
		}
		if ui.Comp_buttonMenu(0, 1, 1, 1, ui.trns.ADD_NEW_ROW, false, Comp_buttonProp()) > 0 {
			SANodeColRow_Insert(&node.Rows, nil, gridMax.Y, true)
		}

		ui.Dialog_end()
	}

	//cols header
	ui.Div_start(1, 0, 1, 1)
	{
		colDiv = ui.GetCall().call
		ui.DivInfo_set(SA_DIV_SET_scrollHshow, 0, 0)
		ui.DivInfo_set(SA_DIV_SET_scrollVshow, 0, 0)

		ui.DivInfo_set(SA_DIV_SET_copyCols, appID, 0)

		for i := 0; i < gridMax.X; i++ {
			dnm := fmt.Sprintf("col_details_%d", i)
			item := SANodeColRow_Find(&node.Cols, i)

			//drag & drop
			ui.Div_start(i, 0, 1, 1)
			{
				ui.Div_drag("cols", i)
				src, pos, done := ui.Div_drop("cols", false, true, false)
				if done {
					dst_i := Div_DropMoveElementIndex(src, i, pos)
					cp := SANodeColRow_Find(&node.Cols, src)
					if cp != nil {
						SANodeColRow_Remove(&node.Cols, src)
						SANodeColRow_Insert(&node.Cols, cp, dst_i, true)
					}
				}
			}
			ui.Div_end()

			click := false
			if item != nil {
				click = ui.Comp_buttonLight(i, 0, 1, 1, fmt.Sprintf("%d", i), Comp_buttonProp()) > 0
			} else {
				click = ui.Comp_buttonText(i, 0, 1, 1, fmt.Sprintf("%d", i), Comp_buttonProp().CdFade(true)) > 0
			}
			if click {
				if ui.win.io.keys.ctrl {
					SANodeColRow_GetOrCreate(&node.Cols, i).Max = 100
				} else {
					ui.Dialog_open(dnm, 1)
				}
			}

			app.drawColsRowsDialog(dnm, node, true, i)
		}
	}
	ui.Div_end()

	//rows header
	ui.Div_start(0, 1, 1, 1)
	{
		rowDiv = ui.GetCall().call
		ui.DivInfo_set(SA_DIV_SET_scrollHshow, 0, 0)
		ui.DivInfo_set(SA_DIV_SET_scrollVshow, 0, 0)

		ui.DivInfo_set(SA_DIV_SET_copyRows, appID, 0)

		for i := 0; i < gridMax.Y; i++ {
			dnm := fmt.Sprintf("row_details_%d", i)
			item := SANodeColRow_Find(&node.Rows, i)

			//drag & drop
			ui.Div_start(0, i, 1, 1)
			{
				ui.Div_drag("rows", i)
				src, pos, done := ui.Div_drop("rows", true, false, false)
				if done {
					dst_i := Div_DropMoveElementIndex(src, i, pos)
					cp := SANodeColRow_Find(&node.Rows, src)
					if cp != nil {
						SANodeColRow_Remove(&node.Rows, src)
						SANodeColRow_Insert(&node.Rows, cp, dst_i, true)
					}
				}
			}
			ui.Div_end()

			click := false
			if item != nil {
				click = ui.Comp_buttonLight(0, i, 1, 1, fmt.Sprintf("%d", i), Comp_buttonProp()) > 0
			} else {
				click = ui.Comp_buttonText(0, i, 1, 1, fmt.Sprintf("%d", i), Comp_buttonProp().CdFade(true)) > 0
			}
			if click {
				if ui.win.io.keys.ctrl {
					SANodeColRow_GetOrCreate(&node.Rows, i).Max = 100
				} else {
					ui.Dialog_open(dnm, 1)
				}
			}

			app.drawColsRowsDialog(dnm, node, false, i)
		}

	}
	ui.Div_end()

	if colDiv != nil {
		appDiv.data.scrollH.attach = &colDiv.data.scrollH
	}
	if rowDiv != nil {
		appDiv.data.scrollV.attach = &rowDiv.data.scrollV
	}

	touch := &ui.win.io.touch
	keys := &ui.win.io.keys
	//add node
	if (!ui.touch.IsAnyActive() || ui.touch.canvas == appDiv.GetHash()) && !app.canvas.startClick.Is() && !keys.alt {
		if appDiv.IsOver(ui) {
			grid := appDiv.GetCloseCell(ui.win.io.touch.pos)

			if appDiv.FindFromGridPos(grid.Start) == nil { //no node under touch
				rect := appDiv.data.Convert(ui.win.Cell(), grid)

				rect.Start = rect.Start.Add(appDiv.canvas.Start)
				ui.buff.AddRect(rect, SAApp_getYellow(), ui.CellWidth(0.03))
				ui.buff.AddText("+", InitWinFontPropsDef(ui.win), rect, SAApp_getYellow(), OsV2{1, 1}, 0, 1)

				if appDiv.IsTouchEnd(ui) {

					app.canvas.addGrid = grid
					app.canvas.addPos = OsV2f{float32(app.Cam_x + rand.Float64()*4 - 2), float32(app.Cam_y + rand.Float64()*4 - 2)}
					app.canvas.addParent = NewSANodePath(app.root)
					app.canvas.addnode_search = ""
					ui.Dialog_open("nodes_list_ui", 2)
				}
			}
		}
	}
	app.drawCreateNode()

	//select/move/resize node
	if appDiv.IsOver(ui) {
		touch_grid := appDiv.GetCloseCell(touch.pos)

		//find resizer
		for _, w := range app.all_nodes {
			if !w.CanBeRenderOnCanvas() {
				continue
			}
			if w.Selected && w.GetResizerCoord().Inside(touch.pos) {
				//resize start
				if touch.start && keys.alt {
					app.canvas.resize = NewSANodePath(w)
					break
				}
			}
		}

		//find select/move node
		if !app.canvas.resize.Is() {
			for _, w := range app.all_nodes {
				if !w.CanBeRenderOnCanvas() {
					continue
				}
				if w.GetGridShow() && w.GetGrid().Inside(touch_grid.Start) {
					//select start(go to inside)
					if keys.alt {
						if touch.start {
							app.canvas.startClick = NewSANodePath(w)
							app.canvas.startClickRel = touch_grid.Start.Sub(w.GetGrid().Start)
							w.SelectOnlyThis()
						}
					}
					break
				}
			}
		}

		//move
		if app.canvas.startClick.Is() {
			newPos := touch_grid.Start.Sub(app.canvas.startClickRel)
			stClick := app.canvas.startClick.FindPath(app.root)
			if stClick != nil {
				stClick.SetGridStart(newPos)
			}
		}

		//resize
		if app.canvas.resize.Is() {
			pos := appDiv.GetCloseCell(touch.pos)

			res := app.canvas.resize.FindPath(app.root)
			if res != nil {
				grid := res.GetGrid()
				grid.Size.X = OsMax(0, pos.Start.X-grid.Start.X) + 1
				grid.Size.Y = OsMax(0, pos.Start.Y-grid.Start.Y) + 1

				res.SetGrid(grid)
			}
		}

	}
	if touch.end {
		if appDiv.IsOver(ui) && keys.alt && !app.canvas.startClick.Is() && !app.canvas.resize.Is() { //click outside nodes
			app.root.DeselectAll()
		}
		app.canvas.startClick = SANodePath{}
		app.canvas.resize = SANodePath{}
	}

	//shortcuts
	if !ui.edit.IsActive() && appDiv.IsOver(ui) {
		keys := &ui.win.io.keys

		//delete
		if keys.delete {
			app.root.RemoveSelectedNodes()
		}
	}
}

func (app *SAApp) ComboListOfNodes(x, y, w, h int, act string) string {
	ui := app.base.ui

	fns_values := app.base.node_groups.getList()

	ui.Comp_combo(x, y, w, h, &act, fns_values, fns_values, "", true, true) //add icons ...
	return act
}

func SAApp_IsSearchedName(name string, search []string) bool {
	name = strings.ToLower(name)
	for _, se := range search {
		if !strings.Contains(name, se) {
			return false //must has all
		}
	}
	return true
}

func (app *SAApp) drawCreateNodeGroup(start OsV2, gr *SAGroup, searches []string, only_ui bool) OsV2 {
	ui := app.base.ui
	keys := &ui.win.io.keys

	ui.Div_start(start.X, start.Y, 1, 1+len(gr.nodes))
	{
		ui.Div_colMax(0, 100)

		ui.Comp_text(0, 0, 1, 1, gr.name, 1)

		y := 1
		for _, nd := range gr.nodes {
			if !only_ui || nd.render != nil {
				if app.canvas.addnode_search == "" || SAApp_IsSearchedName(nd.name, searches) {
					if keys.enter || ui.Comp_buttonMenuIcon(0, y, 1, 1, nd.name, gr.icon, 0.2, false, Comp_buttonProp()) > 0 {
						//add new node
						parent := app.canvas.addParent.FindPath(app.root)
						if only_ui {
							parent = app.Selected_canvas.FindPath(app.root)
						}

						nw := parent.AddNode(app.canvas.addGrid, app.canvas.addPos, nd.name, nd.name)
						nw.SelectOnlyThis()
						ui.CloseAll()
						keys.enter = false
					}
					y++
				}
			}
		}

		pl := ui.win.io.GetPalette()
		ui.Paint_rect(0, 0, 1, 1, 0.03, pl.GetGrey(0.2), 0.03)
	}
	ui.Div_end()
	start.Y += 1 + len(gr.nodes)
	return start
}

func (app *SAApp) drawCreateNode() {
	ui := app.base.ui

	mode_ui := ui.Dialog_start("nodes_list_ui")
	mode_graph := ui.Dialog_start("nodes_list_graph")
	mode_exe := ui.Dialog_start("nodes_list_exe")

	if mode_ui || mode_graph || mode_exe {
		ui.Div_colMax(0, 5)
		ui.Div_colMax(1, 5)

		ui.Comp_editbox(0, 0, 2, 1, &app.canvas.addnode_search, Comp_editboxProp().TempToValue(true).Ghost(ui.trns.SEARCH).Highlight(app.canvas.addnode_search != ""))

		//search
		searches := strings.Split(strings.ToLower(app.canvas.addnode_search), " ")

		//group: UI
		if mode_ui || mode_graph {
			app.drawCreateNodeGroup(OsV2{0, 1}, app.base.node_groups.groups[0], searches, mode_ui)
		}

		//group: Disk
		p := OsV2{1, 1}
		if mode_ui || mode_graph {
			p = app.drawCreateNodeGroup(p, app.base.node_groups.groups[1], searches, mode_ui)
		}

		//group: NN
		if mode_ui || mode_graph {
			app.drawCreateNodeGroup(p, app.base.node_groups.groups[2], searches, mode_ui)
		}

		//group: code
		if mode_exe {
			app.drawCreateNodeGroup(OsV2{0, 1}, app.base.node_groups.groups[3], searches, mode_ui)
		}

		if ui.win.io.keys.tab {
			ui.edit.uid = nil //non-standard(not save src) end of editbox
			ui.Dialog_close()
		}

		ui.Dialog_end()
	}
}

func (app *SAApp) drawColsRowsDialog(name string, node *SANode, isCol bool, pos int) bool {

	ui := app.base.ui

	items := &node.Rows
	if isCol {
		items = &node.Cols
	}

	changed := false
	if ui.Dialog_start(name) {

		ui.Div_col(0, 10)

		//add left/right
		ui.Div_start(0, 0, 1, 1)
		{
			ui.Div_colMax(0, 100)
			ui.Div_colMax(1, 100)
			ui.Div_colMax(2, 100)

			if ui.Comp_buttonLight(0, 0, 1, 1, ui.trns.ADD_BEFORE, Comp_buttonProp()) > 0 {
				SANodeColRow_Insert(items, nil, pos, true)
				node.MakeGridSpace(OsTrn(isCol, pos, 0), OsTrn(!isCol, pos, 0), OsTrn(isCol, 1, 0), OsTrn(!isCol, 1, 0))
				ui.Dialog_close()
				changed = true
			}

			ui.Comp_text(1, 0, 1, 1, strconv.Itoa(pos), 1) //description

			if ui.Comp_buttonLight(2, 0, 1, 1, ui.trns.ADD_AFTER, Comp_buttonProp()) > 0 {
				SANodeColRow_Insert(items, nil, pos+1, true)
				node.MakeGridSpace(OsTrn(isCol, pos+1, 0), OsTrn(!isCol, pos+1, 0), OsTrn(isCol, 1, 0), OsTrn(!isCol, 1, 0))
				ui.Dialog_close()
				changed = true
			}
		}
		ui.Div_end()

		item := SANodeColRow_Find(items, pos)
		not_exist := item == nil
		if not_exist {
			item = &SANodeColRow{Pos: pos, Min: 1, Max: 1, Resize: 1}
		}

		_, _, _, fnshd1, _ := ui.Comp_editbox_desc(ui.trns.MIN, 0, 2, 0, 1, 1, 1, &item.Min, Comp_editboxProp())
		_, _, _, fnshd2, _ := ui.Comp_editbox_desc(ui.trns.MAX, 0, 2, 0, 2, 1, 1, &item.Max, Comp_editboxProp())

		ui.Div_start(0, 3, 1, 1)
		{
			ui.Div_colMax(0, 100)
			ui.Div_colMax(1, 100)

			_, _, _, fnshd3, _ := ui.Comp_editbox_desc(ui.trns.RESIZE, 0, 2, 0, 0, 1, 1, &item.ResizeName, Comp_editboxProp().Ghost(ui.trns.NAME))
			ui.Comp_text(1, 0, 1, 1, strconv.FormatFloat(item.Resize, 'f', 2, 64), 0)

			if fnshd1 || fnshd2 || fnshd3 {
				if not_exist {
					SANodeColRow_Insert(items, item, pos, false)
				}
				changed = true
			}

		}
		ui.Div_end()

		//remove
		if ui.Comp_button(0, 5, 1, 1, ui.trns.REMOVE, Comp_buttonProp().Enable(item != nil)) > 0 {
			SANodeColRow_Remove(items, pos)
			ui.Dialog_close()
			changed = true
		}

		ui.Dialog_end()
	}

	return changed
}
