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

type SASetAttr struct {
	node       SANodePath
	attr       string
	value      string
	mapOrArray bool
	exeIt      bool
}

func (sa *SASetAttr) Write(root *SANode) bool {
	if sa.exeIt {
		return true
	}

	node := sa.node.FindPath(root)
	if node == nil {
		return false
	}

	attr := node.findAttr(sa.attr)
	if attr == nil {
		return false
	}

	attr.SetExpString(sa.value, sa.mapOrArray)
	return true
}

type SAApp struct {
	base *SABase

	Name string
	IDE  bool

	Cam_x, Cam_y, Cam_z float64 `json:",omitempty"`
	EnableExecution     bool

	ShowCode bool

	historyIt bool
	exeIt     bool
	root      *SANode

	history     [][]byte //JSONs
	history_pos int

	setAttrs []SASetAttr
	exeNode  *SANode //copy, not pointer
	exeState int     //0=SANode_STATE_WAITING, 1=SANode_STATE_RUNNING, 2=SANode_STATE_DONE

	graph  *SAGraph
	canvas SACanvas

	ops   *VmOps
	apis  *VmApis
	prior int

	iconPath string

	mic_nodes []SANodePath

	code string
}

func (a *SAApp) init(base *SABase) {
	a.base = base

	a.ops = NewVmOps()
	a.apis = NewVmApis()
	a.prior = 100

	a.graph = NewSAGraph(a)

	a.Cam_z = 1

	ic := a.GetFolderPath() + "icon.png"
	if OsFileExists(ic) {
		a.iconPath = "file:" + ic
	}
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

func (app *SAApp) AddSetAttr(attr *SANodeAttr, value string, mapOrArray bool, exeIt bool) {

	if len(app.setAttrs) > 0 && exeIt && app.setAttrs[len(app.setAttrs)-1].exeIt {
		return
	}

	var node SANodePath
	var name string
	if attr != nil {
		node = NewSANodePath(attr.node)
		name = attr.Name
	}

	app.setAttrs = append(app.setAttrs, SASetAttr{node: node, attr: name, value: value, mapOrArray: mapOrArray, exeIt: exeIt})
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

func (app *SAApp) Tick() {

	switch app.exeState {
	case SANode_STATE_WAITING:

		if app.EnableExecution {
			//save changes
			n := 0
			for _, sa := range app.setAttrs {
				sa.Write(app.root) //if different, app.exeIt = true

				n++
				if sa.exeIt {
					break
				}
			}

			if n > 0 {
				app.setAttrs = app.setAttrs[n:]
			}

			if n == 0 && app.historyIt && !app.base.ui.touch.IsAnyActive() {
				app.historyIt = false
				app.addHistory()
			}

			if app.exeIt {
				app.exeIt = false
				app.historyIt = true

				var err error
				app.exeNode, err = app.root.Copy()
				if err != nil {
					fmt.Println("Error Copy()", err)
					return
				}

				app.code = app.ExportCode()

				app.exeState = SANode_STATE_RUNNING
				go app.executeThread()
			}

		}

	case SANode_STATE_RUNNING:
		//nothing ...

	case SANode_STATE_DONE:
		//copy result back
		app.exeNode.CopyPoses(app.root)
		app.root = app.exeNode
		app.exeNode = nil

		app.exeState = SANode_STATE_WAITING

		app.base.ui.win.SetRedraw()
	}
}

func (app *SAApp) executeThread() {

	app.exeNode.PrepareExe() //.state = WAITING(to be executed)
	app.exeNode.ParseExpresions()
	app.exeNode.CheckForLoops()

	app.exeNode.ExecuteSubs()
	app.exeState = SANode_STATE_DONE
}

func (app *SAApp) RenderApp(ide bool) {
	app.root.renderLayout()
}

func (app *SAApp) renderIDE(ui *Ui) {

	ui.Div_colMax(1, 100)
	ui.Div_rowMax(1, 100)

	var colDiv *UiLayoutDiv
	var rowDiv *UiLayoutDiv

	node := app.root

	//size
	appDiv := ui.Div_start(1, 1, 1, 1)
	gridMax := appDiv.GetGridMax(OsV2{1, 1}) //app size
	ui.Div_end()

	gridMax.X = OsMax(gridMax.X, SANodeColRow_GetMaxPos(&node.Cols)+1)
	gridMax.Y = OsMax(gridMax.Y, SANodeColRow_GetMaxPos(&node.Rows)+1)

	changed := false

	//cols header
	ui.Div_start(1, 0, 1, 1)
	{
		ui.DivInfo_set(SA_DIV_SET_scrollHshow, 0, 0)
		ui.DivInfo_set(SA_DIV_SET_scrollVshow, 0, 0)

		node.renderLayoutCols()

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
						changed = true
					}
				}
			}
			ui.Div_end()

			if item != nil {
				if ui.Comp_buttonLight(i, 0, 1, 1, fmt.Sprintf("%d", i), "", true) > 0 {
					ui.Dialog_open(dnm, 1)
				}
			} else {
				if ui.Comp_buttonTextFade(i, 0, 1, 1, fmt.Sprintf("%d", i), "", "", true, false, true) > 0 {
					ui.Dialog_open(dnm, 1)
				}
			}

			if _SAApp_drawColsRowsDialog(dnm, &node.Cols, i, ui) {
				changed = true
			}
		}
	}
	ui.Div_end()

	//+
	if ui.Comp_buttonLight(2, 0, 1, 1, "+", ui.trns.ADD_NEW_COLUMN, true) > 0 {
		SANodeColRow_Insert(&node.Cols, nil, gridMax.X, true)
		changed = true
	}

	//rows header
	ui.Div_start(0, 1, 1, 1)
	{
		rowDiv = ui.GetCall().call
		ui.DivInfo_set(SA_DIV_SET_scrollHshow, 0, 0)
		ui.DivInfo_set(SA_DIV_SET_scrollVshow, 0, 0)

		node.renderLayoutRows()

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
						changed = true
					}
				}
			}
			ui.Div_end()

			if item != nil {
				if ui.Comp_buttonLight(0, i, 1, 1, fmt.Sprintf("%d", i), "", true) > 0 {
					ui.Dialog_open(dnm, 1)
				}
			} else {
				if ui.Comp_buttonTextFade(0, i, 1, 1, fmt.Sprintf("%d", i), "", "", true, false, true) > 0 {
					ui.Dialog_open(dnm, 1)
				}
			}

			if _SAApp_drawColsRowsDialog(dnm, &node.Rows, i, ui) {
				changed = true
			}
		}

	}
	ui.Div_end()

	//+
	if ui.Comp_buttonLight(0, 2, 1, 1, "+", ui.trns.ADD_NEW_ROW, true) > 0 {
		SANodeColRow_Insert(&node.Rows, nil, gridMax.Y, true)
		changed = true
	}

	//app
	appDiv = ui.Div_start(1, 1, 1, 1)
	{
		if colDiv != nil {
			ui.GetCall().call.data.scrollH.attach = &colDiv.data.scrollH
		}
		if rowDiv != nil {
			ui.GetCall().call.data.scrollV.attach = &rowDiv.data.scrollV
		}

		app.RenderApp(true)
	}
	ui.Div_end()

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
					app.canvas.addPos = OsV2f{}
					app.canvas.addParent = NewSANodePath(app.root)
					app.canvas.addnode_search = ""
					ui.Dialog_open("nodes_list", 2)
				}
			}
		}
	}
	app.drawCreateNode(ui)

	//select/move/resize node
	if appDiv.IsOver(ui) {
		touch_grid := appDiv.GetCloseCell(touch.pos)

		//find resizer
		for _, w := range app.root.Subs {
			if !w.CanBeRenderOnCanvas() {
				continue
			}
			if w.Selected && w.GetResizerCoord(ui).Inside(touch.pos) {
				//resize start
				if touch.start && keys.alt {
					app.canvas.resize = NewSANodePath(w)
					break
				}
			}
		}

		//find select/move node
		if !app.canvas.resize.Is() {
			for _, w := range app.root.Subs {
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
				changed = true
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
				changed = true
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
	if ui.edit.uid == nil && appDiv.IsOver(ui) {
		keys := &ui.win.io.keys

		//delete
		if keys.delete {
			app.root.RemoveSelectedNodes()
			changed = true
		}
	}

	if changed {
		app.SetExecute()
	}
}

func (app *SAApp) History(ui *Ui) {
	if len(app.history) == 0 {
		return
	}

	lv := ui.GetCall()
	if !ui.edit.IsActive() {
		if lv.call.IsOver(ui) && ui.win.io.keys.backward {
			app.stepHistoryBack()

		}
		if lv.call.IsOver(ui) && ui.win.io.keys.forward {
			app.stepHistoryForward()
		}
	}
}

func (app *SAApp) ComboListOfNodes(x, y, w, h int, act string, ui *Ui) string {
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

func (app *SAApp) drawCreateNode(ui *Ui) {

	if ui.Dialog_start("nodes_list") {
		ui.Div_colMax(0, 5)

		y := 0
		ui.Comp_editbox(0, 0, 1, 1, &app.canvas.addnode_search, Comp_editboxProp().TempToValue(true).Ghost(ui.trns.SEARCH).Highlight(app.canvas.addnode_search != ""))
		y++

		if app.canvas.addnode_search != "" {
			//search
			keys := &ui.win.io.keys
			searches := strings.Split(strings.ToLower(app.canvas.addnode_search), " ")
			for _, gr := range app.base.node_groups.groups {
				for _, nd := range gr.nodes {
					if app.canvas.addnode_search == "" || SAApp_IsSearchedName(nd.name, searches) {
						if keys.enter || ui.Comp_buttonMenuIcon(0, y, 1, 1, nd.name, gr.icon, 0.2, "", true, false) > 0 {
							//add new node
							parent := app.canvas.addParent.FindPath(app.root)
							nw := parent.AddNode(app.canvas.addGrid, app.canvas.addPos, nd.name, nd.name)
							nw.SelectOnlyThis()

							ui.Dialog_close()
							keys.enter = false
						}
						y++
					}
				}
			}
		} else {
			for _, gr := range app.base.node_groups.groups {
				//folders
				dnm := "node_group_" + gr.name
				if ui.Comp_buttonMenuIcon(0, y, 1, 1, gr.name, gr.icon, 0.2, "", true, false) > 0 {
					ui.Dialog_open(dnm, 1)
				}
				//ui.Comp_text(1, y, 1, 1, "►", 1)

				if ui.Dialog_start(dnm) {
					ui.Div_colMax(0, 5)

					for i, nd := range gr.nodes {
						if ui.Comp_buttonMenuIcon(0, i, 1, 1, nd.name, gr.icon, 0.2, "", true, false) > 0 {
							//add new node
							nw := app.root.AddNode(app.canvas.addGrid, app.canvas.addPos, nd.name, nd.name)
							nw.SelectOnlyThis()

							ui.CloseAll()
						}
					}

					ui.Dialog_end()
				}

				y++
			}

		}

		if ui.win.io.keys.tab {
			ui.edit.uid = nil //non-standard(not save src) end of editbox
			ui.Dialog_close()
		}

		ui.Dialog_end()
	}
}

func _SAApp_drawColsRowsDialog(name string, items *[]*SANodeColRow, pos int, ui *Ui) bool {

	changed := false
	if ui.Dialog_start(name) {

		ui.Div_col(0, 10)

		//add left/right
		ui.Div_start(0, 0, 1, 1)
		{
			ui.Div_colMax(0, 100)
			ui.Div_colMax(1, 100)
			ui.Div_colMax(2, 100)

			if ui.Comp_buttonLight(0, 0, 1, 1, ui.trns.ADD_BEFORE, "", pos > 0) > 0 {
				SANodeColRow_Insert(items, nil, pos, true)
				ui.Dialog_close()
				changed = true
			}

			ui.Comp_text(1, 0, 1, 1, strconv.Itoa(pos), 1) //description

			if ui.Comp_buttonLight(2, 0, 1, 1, ui.trns.ADD_AFTER, "", true) > 0 {
				SANodeColRow_Insert(items, nil, pos+1, true)
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
		if ui.Comp_button(0, 5, 1, 1, ui.trns.REMOVE, "", item != nil) > 0 {
			SANodeColRow_Remove(items, pos)
			ui.Dialog_close()
			changed = true
		}

		ui.Dialog_end()
	}

	return changed
}

func (app *SAApp) RenderHeader(ui *Ui) {
	ui.Div_colMax(1, 100)
	ui.Div_colMax(2, 6)

	ui.Comp_text(2, 0, 1, 1, "Press Alt-key to select nodes", 1)

	if ui.Comp_buttonLight(3, 0, 1, 1, "←", fmt.Sprintf("%s(%d)", ui.trns.BACKWARD, app.history_pos), app.canHistoryBack()) > 0 {
		app.stepHistoryBack()

	}
	if ui.Comp_buttonLight(4, 0, 1, 1, "→", fmt.Sprintf("%s(%d)", ui.trns.FORWARD, len(app.history)-app.history_pos-1), app.canHistoryForward()) > 0 {
		app.stepHistoryForward()
	}
}

func (app *SAApp) addHistory() {

	js, err := json.Marshal(app.root)
	if err != nil {
		return
	}
	if len(app.history) > 0 && bytes.Equal(js, app.history[app.history_pos]) {
		return //same as current history
	}

	//cut newer history
	if app.history_pos+1 < len(app.history) {
		app.history = app.history[:app.history_pos+1]
	}

	app.history = append(app.history, js)
	app.history_pos = len(app.history) - 1
}

func (app *SAApp) recoverHistory() {

	dst := NewSANode(app, nil, "", "", OsV4{}, OsV2f{})
	err := json.Unmarshal(app.history[app.history_pos], dst)
	if err != nil {
		return
	}
	dst.updateLinks(nil, app)
	app.root = dst

	app.SetExecute()
}

func (app *SAApp) canHistoryBack() bool {
	return app.history_pos > 0
}
func (app *SAApp) canHistoryForward() bool {
	return app.history_pos+1 < len(app.history)
}

func (app *SAApp) stepHistoryBack() bool {
	if !app.canHistoryBack() {
		return false
	}

	app.history_pos--
	app.recoverHistory()
	return true
}
func (app *SAApp) stepHistoryForward() bool {

	if !app.canHistoryForward() {
		return false
	}

	app.history_pos++
	app.recoverHistory()
	return true
}

func (app *SAApp) SetExecute() {
	app.exeIt = true
}

func SAApp_getYellow() OsCd {
	return OsCd{204, 204, 0, 255} //...
}

type SALine struct {
	line string
	subs []*SALine
}

func (node *SANode) _importCode(line *SALine, ops *VmOps) {

	for i, ln := range line.subs {
		if ln.line == "" {
			continue //skip empty
		}

		lex, err := ParseLine(ln.line, 0, ops)
		if err != nil {
			fmt.Printf("Line(%d: %s) has parsing error: %v\n", i, ln, err)
			continue
		}

		if len(lex.subs) >= 3 &&
			lex.subs[0].tp == VmLexerWord &&
			lex.subs[1].tp == VmLexerOp &&
			lex.subs[1].GetString(ln.line) == "=" &&
			lex.subs[2].tp == VmLexerWord &&
			lex.subs[3].tp == VmLexerBracketRound {

			nd := node.AddNode(OsV4{}, OsV2f{}, lex.subs[0].GetString(ln.line), lex.subs[2].GetString(ln.line)) //grid ... pos ...

			//parameters
			prms := lex.subs[3]
			prm_i := 0
			for {
				prm := prms.ExtractParam(prm_i)
				if prm == nil {
					break
				}

				prm_n := len(prm.subs)
				if prm_n >= 3 && prm.subs[0].tp == VmLexerWord && prm.subs[1].tp == VmLexerDiv {
					key := prm.Extract(0, 1).GetString(ln.line)
					value := prm.Extract(2, -1).GetString(ln.line)

					attr := nd.findAttr(key)
					if attr == nil {
						attr = nd.AddAttr(key)
					}
					attr.Value = value
				} else {
					fmt.Printf("Line(%d: %s) has param(%d) error\n", i, ln, prm_i)
				}

				prm_i++
			}

			//subs
			nd._importCode(ln, ops)
		} else {
			fmt.Printf("Line(%d: %s) has base error\n", i, ln)
		}
	}
}

func _SALine_countTabs(line string) int {
	tabs := 0
	for _, ch := range line {
		if ch == '\t' {
			tabs++
		} else {
			break
		}
	}
	return tabs

}

func InitSALine(code string) *SALine {
	root_ln := &SALine{}

	var list []*SALine
	list = append(list, root_ln) //keep depth
	code_lines := strings.Split(code, "\n")
	for i := 0; i < len(code_lines); i++ {

		line := code_lines[i]

		tabs := _SALine_countTabs(line)
		line = line[tabs:] //cut

		depth := len(list) - 1
		if tabs == depth {
			list[depth].subs = append(list[depth].subs, &SALine{line: line})
		} else if tabs < depth {
			list = list[:tabs+1]
			i-- //do the line again
		} else if tabs == depth+1 {

			if len(list[depth].subs) > 0 {
				parent := list[depth].subs[len(list[depth].subs)-1]
				//go deep
				list = append(list, parent)
				i-- //do the line again
			} else {
				fmt.Printf("Error on line(%d): First line must be 0 tabs\n", i)
			}
		} else if tabs > depth+1 {
			fmt.Printf("Error on line(%d): Too many tabs\n", i)
		}
	}

	return root_ln
}

func (app *SAApp) ImportCode(code string) {

	ln := InitSALine(code)

	ops := *app.ops
	ops.ops = append(ops.ops, VmOp{100, false, "=", nil})

	app.root, _ = NewSANodeRoot("", app)
	app.root._importCode(ln, &ops)
}

func (node *SANode) _exportCode(depth int) string {
	tabs := ""
	for i := 0; i < depth; i++ {
		tabs += "\t"
	}

	str := ""
	for _, nd := range node.Subs {
		//params
		params := ""
		for _, attr := range nd.Attrs {
			if !attr.IsOutput() {
				params += fmt.Sprintf("%s:%s,", attr.Name, OsTrnString(attr.Value == "", `""`, attr.Value))
			}
		}
		params, _ = strings.CutSuffix(params, ",")

		//line
		str += fmt.Sprintf("%s%s=%s(%s)\n", tabs, nd.Name, nd.Exe, params)

		//subs
		if len(nd.Subs) > 0 {
			str += nd._exportCode(depth + 1)
		}
	}

	return str
}

func (app *SAApp) ExportCode() string {
	str := app.root._exportCode(0)
	str, _ = strings.CutSuffix(str, "\n")
	return str
}
