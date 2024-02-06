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
	"strings"
)

type SACanvas struct {
	addGrid OsV4
	addPos  OsV2f

	startClick    *SANode
	startClickRel OsV2

	resize *SANode

	addnode_search string
}

type SASetAttr struct {
	attr  *SANodeAttr
	value string
}

type SAApp struct {
	base *SABase

	Name string
	IDE  bool

	root *SANode
	act  *SANode

	history_act           []*SANode //JSONs
	history               []*SANode //JSONs
	history_pos           int
	history_divScroll     *UiLayoutDiv
	history_divSroll_time float64

	exeIt bool
	exe   *SAAppExe

	graph  *SAGraph
	canvas SACanvas

	ops   *VmOps
	apis  *VmApis
	prior int

	EnableExecution bool

	iconPath string
}

func (a *SAApp) init(base *SABase) {
	a.base = base

	a.ops = NewVmOps()
	a.apis = NewVmApis()
	a.prior = 100

	a.graph = NewSAGraph(a)

	a.exe = NewSAAppExe(a)

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
	app.exe.Destroy()
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

func (app *SAApp) RenderApp(ide bool) {

	node := app.root
	if ide {
		node = app.act
	}

	node.renderLayout()
}

func (app *SAApp) renderIDE(ui *Ui) {

	ui.Div_colMax(1, 100)
	ui.Div_rowMax(1, 100)

	var colDiv *UiLayoutDiv
	var rowDiv *UiLayoutDiv

	lay := app.act

	//at least one
	if len(lay.Cols) == 0 {
		lay.Cols = append(lay.Cols, InitSANodeColRow())
	}
	if len(lay.Rows) == 0 {
		lay.Rows = append(lay.Rows, InitSANodeColRow())
	}

	if ui.Comp_button(0, 0, 1, 1, "+", ui.trns.ADD_COLUMNS_ROWS, true) > 0 {
		ui.Dialog_open("add_col_row", 1)
	}
	if ui.Dialog_start("add_col_row") {
		ui.Div_col(0, 4)
		if ui.Comp_buttonMenu(0, 0, 1, 1, ui.trns.ADD_NEW_COLUMN, "", true, false) > 0 {
			lay.Cols = append(lay.Cols, InitSANodeColRow())
		}
		if ui.Comp_buttonMenu(0, 1, 1, 1, ui.trns.ADD_NEW_ROW, "", true, false) > 0 {
			lay.Rows = append(lay.Rows, InitSANodeColRow())

		}
		ui.Dialog_end()
	}

	//size
	appDiv := ui.Div_start(1, 1, 1, 1)
	gridMax := appDiv.GetGridMax(OsV2{1, 1})
	ui.Div_end()

	//cols header
	ui.Div_start(1, 0, 1, 1)
	{
		colDiv = ui.GetCall().call
		ui.DivInfo_set(SA_DIV_SET_scrollHshow, 0, 0)
		ui.DivInfo_set(SA_DIV_SET_scrollVshow, 0, 0)

		for i, c := range lay.Cols {
			ui.Div_col(i, c.Min)
			ui.Div_colMax(i, c.Max)
			if c.ResizeName != "" {
				active, v := ui.Div_colResize(i, c.ResizeName, c.Resize, true)
				if active {
					lay.Cols[i].Resize = v
				}
			}
		}
		//add fake
		for i := len(lay.Cols); i < gridMax.X; i++ {
			ui.Div_col(i, 1)
		}

		for i := range lay.Cols {
			nm := fmt.Sprintf("col_details_%d", i)

			//drag & drop
			ui.Div_start(i, 0, 1, 1)
			{
				ui.Div_drag("cols", i)
				src, pos, done := ui.Div_drop("cols", false, true, false)
				if done {
					Div_DropMoveElement(&lay.Cols, &lay.Cols, src, i, pos)
				}
			}
			ui.Div_end()

			if ui.Comp_buttonLight(i, 0, 1, 1, fmt.Sprintf("%d", i), "", true) > 0 {
				ui.Dialog_open(nm, 1)
			}

			_SAApp_drawColsRowsDialog(nm, &lay.Cols, i, ui)

		}
	}
	ui.Div_end()

	//rows header
	ui.Div_start(0, 1, 1, 1)
	{
		rowDiv = ui.GetCall().call
		ui.DivInfo_set(SA_DIV_SET_scrollHshow, 0, 0)
		ui.DivInfo_set(SA_DIV_SET_scrollVshow, 0, 0)

		for i, r := range lay.Rows {
			ui.Div_row(i, r.Min)
			ui.Div_rowMax(i, r.Max)

			if r.ResizeName != "" {
				active, v := ui.Div_rowResize(i, r.ResizeName, r.Resize, true)
				if active {
					lay.Rows[i].Resize = v
				}
			}
		}
		//add fake
		for i := len(lay.Rows); i < gridMax.Y; i++ {
			ui.Div_col(i, 1)
		}

		for i := range lay.Rows {

			nm := fmt.Sprintf("row_details_%d", i)

			//drag & drop
			ui.Div_start(0, i, 1, 1)
			{
				ui.Div_drag("rows", i)
				src, pos, done := ui.Div_drop("rows", true, false, false)
				if done {
					Div_DropMoveElement(&lay.Rows, &lay.Rows, src, i, pos)
				}
			}
			ui.Div_end()

			if ui.Comp_buttonLight(0, i, 1, 1, fmt.Sprintf("%d", i), "", true) > 0 {
				ui.Dialog_open(nm, 1)
			}
			_SAApp_drawColsRowsDialog(nm, &lay.Rows, i, ui)
		}

	}
	ui.Div_end()

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
	if (!ui.touch.IsAnyActive() || ui.touch.canvas == appDiv) && app.canvas.startClick == nil && !keys.alt {
		touchPos := ui.win.io.touch.pos
		if appDiv.IsOver(ui) { // appDiv.crop.Inside(touchPos)
			grid := appDiv.GetCloseCell(touchPos)

			if appDiv.FindFromGridPos(grid.Start) == nil { //no node under touch
				rect := appDiv.data.Convert(ui.win.Cell(), grid)

				rect.Start = rect.Start.Add(appDiv.canvas.Start)
				ui.buff.AddRect(rect, SAApp_getYellow(), ui.CellWidth(0.03))
				ui.buff.AddText("+", InitWinFontPropsDef(ui.win), rect, SAApp_getYellow(), OsV2{1, 1}, 0, 1)

				if appDiv.IsTouchEnd(ui) {
					app.canvas.addGrid = grid
					app.canvas.addPos = OsV2f{}
					app.canvas.addnode_search = ""
					ui.Dialog_open("nodes_list", 2)
				}
			}
		}
	}
	app.drawCreateNode(ui)

	//select/move/resize node
	if appDiv.IsOver(ui) {
		grid := appDiv.GetCloseCell(touch.pos)

		//find resizer
		for _, w := range app.act.Subs {
			if !w.CanBeRenderOnCanvas() {
				continue
			}
			if w.Selected && w.GetResizerCoord(ui).Inside(touch.pos) {
				//resize start
				if touch.start && keys.alt {
					app.canvas.resize = w
					break
				}
			}
		}

		//find select/move node
		if app.canvas.resize == nil {
			for _, w := range app.act.Subs {
				if !w.CanBeRenderOnCanvas() {
					continue
				}
				if w.GetGridShow() && w.GetGrid().Inside(grid.Start) {

					//select start(go to inside)
					if keys.alt {
						if touch.start {
							wStart := appDiv.crop.Start.Add(appDiv.data.Convert(ui.win.Cell(), w.GetGrid()).Start)
							app.canvas.startClick = w
							app.canvas.startClickRel = touch.pos.Sub(wStart)
							w.SelectOnlyThis()
						}

						if touch.end && touch.numClicks > 1 && w.IsGuiLayout() {
							app.act = w //goto layout
						}
					}

					break
				}
			}
		}

		//move
		if app.canvas.startClick != nil {
			gridMove := appDiv.GetCloseCell(touch.pos.Sub(app.canvas.startClickRel).Add(OsV2{ui.CellWidth(0.5), ui.CellWidth(0.5)}))
			//gridMove := appDiv.GetCloseCell(touch.pos)
			app.canvas.startClick.SetGridStart(gridMove.Start)
		}

		//resize
		if app.canvas.resize != nil {
			pos := appDiv.GetCloseCell(touch.pos)

			grid := app.canvas.resize.GetGrid()
			grid.Size.X = OsMax(0, pos.Start.X-grid.Start.X) + 1
			grid.Size.Y = OsMax(0, pos.Start.Y-grid.Start.Y) + 1

			app.canvas.resize.SetGrid(grid)
		}

	}
	if touch.end {
		if appDiv.IsOver(ui) && keys.alt && app.canvas.startClick == nil && app.canvas.resize == nil { //click outside nodes
			app.act.DeselectAll()
		}
		app.canvas.startClick = nil
		app.canvas.resize = nil
	}

	//shortcuts
	if ui.edit.uid == nil && appDiv.IsOver(ui) {
		keys := &ui.win.io.keys

		//bypass
		if keys.text == "b" {
			app.act.BypassReverseSelectedNodes()
		}

		//delete
		if keys.delete {
			app.act.RemoveSelectedNodes()
		}
	}
}

func (app *SAApp) History(ui *Ui) {

	//init history
	if len(app.history) == 0 {
		app.addHistory(true, false)
		app.history_pos = 0
	}

	lv := ui.GetCall()
	touch := &ui.win.io.touch
	keys := &ui.win.io.keys

	if !ui.edit.IsActive() {
		if lv.call.IsOver(ui) && ui.win.io.keys.backward {
			app.stepHistoryBack()

		}
		if lv.call.IsOver(ui) && ui.win.io.keys.forward {
			app.stepHistoryForward()
		}
	}

	if touch.end || keys.hasChanged || app.base.ui.touch.scrollWheel != nil || touch.drop_path != "" {
		app.cmpAndAddHistory()
	}
}

func (app *SAApp) ComboListOfNodes(x, y, w, h int, act string, ui *Ui) string {
	//fns_names := app.getListOfNodesTranslated()
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
		ui.Comp_editbox(0, 0, 1, 1, &app.canvas.addnode_search, 0, OsV2{0, 1}, nil, ui.trns.SEARCH, app.canvas.addnode_search != "", true, false, true)
		y++

		if app.canvas.addnode_search != "" {

			//search
			keys := &ui.win.io.keys
			searches := strings.Split(strings.ToLower(app.canvas.addnode_search), " ")
		out1:
			for _, gr := range app.base.node_groups.groups {
				for _, nd := range gr.nodes {
					if app.canvas.addnode_search == "" || SAApp_IsSearchedName(nd.name, searches) {
						if keys.enter || ui.Comp_buttonMenuIcon(0, y, 1, 1, nd.name, gr.icon, 0.2, "", true, false) > 0 {
							//add new node
							nw := app.act.AddNode(app.canvas.addGrid, app.canvas.addPos, nd.name, nd.name)
							nw.SelectOnlyThis()

							ui.Dialog_close()
							break out1
						}
						y++
					}
				}
			}
		} else {

		out2:
			for _, gr := range app.base.node_groups.groups {

				//folders
				dnm := "node_group_" + gr.name
				if ui.Comp_buttonMenuIcon(0, y, 1, 1, gr.name, gr.icon, 0.2, "", true, false) > 0 {
					ui.Dialog_open(dnm, 1)
				}
				ui.Comp_text(1, y, 1, 1, "►", 1)

				if ui.Dialog_start(dnm) {
					ui.Div_colMax(0, 5)

					for i, nd := range gr.nodes {
						if ui.Comp_buttonMenuIcon(0, i, 1, 1, nd.name, gr.icon, 0.2, "", true, false) > 0 {
							//add new node
							nw := app.act.AddNode(app.canvas.addGrid, app.canvas.addPos, nd.name, nd.name)
							nw.SelectOnlyThis()

							ui.CloseAll()
							break out2
						}
					}

					ui.Dialog_end()
				}

				y++
			}

		}

		if ui.win.io.keys.tab {
			ui.Dialog_close()
		}

		ui.Dialog_end()
	}
}

func _SAApp_drawColsRowsDialog(name string, items *[]SANodeColRow, i int, ui *Ui) bool {

	changed := false
	if ui.Dialog_start(name) {

		ui.Div_col(0, 10)

		//add left/right
		ui.Div_start(0, 0, 1, 1)
		{
			ui.Div_colMax(0, 100)
			ui.Div_colMax(1, 100)
			ui.Div_colMax(2, 100)

			if ui.Comp_buttonLight(0, 0, 1, 1, ui.trns.ADD_BEFORE, "", i > 0) > 0 {
				*items = append(*items, SANodeColRow{})
				copy((*items)[i+1:], (*items)[i:])
				(*items)[i] = InitSANodeColRow()
				ui.Dialog_close()
				changed = true
			}

			ui.Comp_text(1, 0, 1, 1, strconv.Itoa(i), 1) //description

			if ui.Comp_buttonLight(2, 0, 1, 1, ui.trns.ADD_AFTER, "", true) > 0 {
				*items = append(*items, SANodeColRow{})
				copy((*items)[i+2:], (*items)[i+1:])
				(*items)[i+1] = InitSANodeColRow()
				ui.Dialog_close()
				changed = true
			}
		}
		ui.Div_end()

		_, _, _, fnshd1, _ := ui.Comp_editbox_desc(ui.trns.MIN, 0, 2, 0, 1, 1, 1, &(*items)[i].Min, 1, OsV2{0, 1}, nil, "", false, false, false, true)
		_, _, _, fnshd2, _ := ui.Comp_editbox_desc(ui.trns.MAX, 0, 2, 0, 2, 1, 1, &(*items)[i].Max, 1, OsV2{0, 1}, nil, "", false, false, false, true)

		ui.Div_start(0, 3, 1, 1)
		{
			ui.Div_colMax(0, 100)
			ui.Div_colMax(1, 100)

			_, _, _, fnshd3, _ := ui.Comp_editbox_desc(ui.trns.RESIZE, 0, 2, 0, 0, 1, 1, &(*items)[i].ResizeName, 1, OsV2{0, 1}, nil, "Name", false, false, false, true)
			ui.Comp_text(1, 0, 1, 1, strconv.FormatFloat((*items)[i].Resize, 'f', 2, 64), 0)

			if fnshd1 || fnshd2 || fnshd3 {
				changed = true
			}

		}
		ui.Div_end()

		//remove
		if ui.Comp_button(0, 5, 1, 1, ui.trns.REMOVE, "", len(*items) > 1) > 0 {
			*items = append((*items)[:i], (*items)[i+1:]...)
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
	//ui.Div_colMax(5, 2)

	//level up
	if ui.Comp_buttonIcon(0, 0, 1, 1, InitWinMedia_url("file:apps/base/resources/levelup.png"), 0.3, "One level up", CdPalette_P, app.act.parent != nil, false) > 0 {
		app.act = app.act.parent
	}
	if !ui.edit.IsActive() {
		keys := &ui.win.io.keys
		if strings.EqualFold(keys.text, "u") {
			if app.act.parent != nil {
				app.act = app.act.parent
			}
		}
	}

	//list
	{
		var listPathes []string
		var listNodes []*SANode
		app.root.buildSubsList(&listPathes, &listNodes)

		val := ""
		for i, nd := range listNodes {
			if app.act == nd {
				val = listPathes[i]
			}
		}
		if ui.Comp_combo(1, 0, 1, 1, &val, listPathes, listPathes, "", true, true) {
			for i, lp := range listPathes {
				if val == lp {
					app.act = listNodes[i]
					break
				}
			}
		}
	}

	ui.Comp_text(2, 0, 1, 1, "Press Alt-key to select nodes", 1)

	//shortcuts
	if ui.Comp_buttonLight(3, 0, 1, 1, "←", fmt.Sprintf("%s(%d)", ui.trns.BACKWARD, app.history_pos), app.canHistoryBack()) > 0 {
		app.stepHistoryBack()

	}
	if ui.Comp_buttonLight(4, 0, 1, 1, "→", fmt.Sprintf("%s(%d)", ui.trns.FORWARD, len(app.history)-app.history_pos-1), app.canHistoryForward()) > 0 {
		app.stepHistoryForward()
	}
}

func (app *SAApp) cmpAndAddHistory() {
	if len(app.history) > 0 {
		historyDiff := false
		exeDiff := !app.root.Cmp(app.history[app.history_pos], &historyDiff)
		if exeDiff || historyDiff {

			//scroll - update last item in history or add new item
			rewrite := (app.history_divScroll != nil && app.history_divScroll == app.base.ui.touch.scrollWheel)
			if OsTime()-app.history_divSroll_time > 1 { //1sec
				rewrite = false
			}

			app.addHistory(exeDiff, rewrite)

			app.history_divScroll = app.base.ui.touch.scrollWheel
			if app.base.ui.touch.scrollWheel != nil {
				app.history_divSroll_time = OsTime()
			}
		}
	}
}

func (app *SAApp) addHistory(exeIt bool, rewriteLast bool) {
	//cut newer history
	if app.history_pos+1 < len(app.history) {
		app.history = app.history[:app.history_pos+1]
		app.history_act = app.history_act[:app.history_pos+1]
	}

	cp_root, _ := app.root.Copy() //err ...
	cp_act := cp_root.FindMirror(app.root, app.act)

	if rewriteLast {
		app.history[app.history_pos] = cp_root
		app.history_act[app.history_pos] = cp_act
	} else {

		if cp_root == nil || cp_act == nil { //delete ........
			fmt.Print("df")
		}

		//add history
		app.history = append(app.history, cp_root)
		app.history_act = append(app.history_act, cp_act)
		app.history_pos++
	}
	if exeIt {
		app.SetExecute()
	}
}

func (app *SAApp) recoverHistory() {
	app.root, _ = app.history[app.history_pos].Copy()
	app.act = app.root.FindMirror(app.history[app.history_pos], app.history_act[app.history_pos])

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

func (app *SAApp) ImportCode(code string) {
	lines := strings.Split(code, "\n")

	ops := *app.ops
	ops.ops = append(ops.ops, VmOp{100, false, "=", nil})

	for i, ln := range lines {
		if ln == "" {
			continue //skip empty
		}

		lex, err := ParseLine(ln, 0, &ops)
		if err != nil {
			fmt.Printf("Line(%d: %s) has parsing error: %v\n", i, ln, err)
			continue
		}

		if len(lex.subs) >= 3 &&
			lex.subs[0].tp == VmLexerWord &&
			lex.subs[1].tp == VmLexerOp &&
			lex.subs[1].GetString(ln) == "=" &&
			lex.subs[2].tp == VmLexerWord &&
			lex.subs[3].tp == VmLexerBracketRound {

			nd := app.root.AddNode(OsV4{}, OsV2f{}, lex.subs[0].GetString(ln), lex.subs[2].GetString(ln)) //grid ... pos ...

			//parameters
			prms := lex.subs[3]
			prm_i := 0
			for {
				prm := prms.ExtractParam(prm_i)
				if prm == nil {
					break
				}

				if len(prm.subs) >= 3 && prm.subs[0].tp == VmLexerWord && prm.subs[1].tp == VmLexerDiv {
					attr := nd.AddAttr(prm.subs[0].GetString(ln))
					attr.Value = ln[:prm.subs[1].end]
				} else {
					fmt.Printf("Line(%d: %s) has param(%d) error\n", i, ln, prm_i)
				}

				prm_i++
			}
		} else {
			fmt.Printf("Line(%d: %s) has base error\n", i, ln)
		}
	}
}

func (app *SAApp) ExportCode() string {
	str := ""

	for _, nd := range app.root.Subs {
		//params
		params := ""
		for _, attr := range nd.Attrs {
			if !attr.IsOutput() {
				params += fmt.Sprintf("%s:%s,", attr.Name, OsTrnString(attr.Value == "", `""`, attr.Value))
			}
		}
		params, _ = strings.CutSuffix(params, ",")

		//whole line
		str += fmt.Sprintf("%s=%s(%s)\n", nd.Name, nd.Exe, params)

		//nd.Subs? ...
	}
	str, _ = strings.CutSuffix(str, "\n")

	return str
}
