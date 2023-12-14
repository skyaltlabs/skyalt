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

type SAApp struct {
	base *SABase

	Name string
	IDE  bool

	root *SAWidget
	act  *SAWidget

	history_act []*SAWidget //JSONs
	history     []*SAWidget //JSONs
	history_pos int

	saveIt bool
	exeIt  bool

	addWidgetCoord OsV4

	startClickWidget *SAWidget
	startClickRel    OsV2

	ops   *VmOps
	apis  *VmApis
	prior int
}

func NewSAApp(name string, base *SABase) *SAApp {
	var app SAApp
	app.base = base
	app.Name = name
	app.IDE = true

	return &app
}
func (app *SAApp) Destroy() {

}

func (app *SAApp) GetPath() string {
	return "apps/" + app.Name + "/app.json"
}

func (app *SAApp) renderIDE(ui *Ui) {

	ui.Div_colMax(1, 100)
	ui.Div_rowMax(1, 100)

	var colDiv *UiLayoutDiv
	var rowDiv *UiLayoutDiv

	lay := app.act

	//at least one
	if len(lay.Cols) == 0 {
		lay.Cols = append(lay.Cols, InitSAWidgetColRow())
	}
	if len(lay.Rows) == 0 {
		lay.Rows = append(lay.Rows, InitSAWidgetColRow())
	}

	if ui.Comp_button(0, 0, 1, 1, "+", "Add Column/Row", true) > 0 {
		ui.Dialog_open("add_col_row", 1)
	}
	if ui.Dialog_start("add_col_row") {
		ui.Div_col(0, 4)
		if ui.Comp_buttonMenu(0, 0, 1, 1, "Add new Column", "", true, false) > 0 {
			lay.Cols = append(lay.Cols, InitSAWidgetColRow())
		}
		if ui.Comp_buttonMenu(0, 1, 1, 1, "Add new Row", "", true, false) > 0 {
			lay.Rows = append(lay.Rows, InitSAWidgetColRow())

		}
		ui.Dialog_end()
	}

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
	appDiv := ui.Div_start(1, 1, 1, 1)
	{
		ui.GetCall().call.data.scrollH.attach = &colDiv.data.scrollH
		ui.GetCall().call.data.scrollV.attach = &rowDiv.data.scrollV

		app.act.RenderLayout(ui, app)
	}
	ui.Div_end()

	touch := &ui.buff.win.io.touch
	keys := &ui.buff.win.io.keys

	//add widget
	if appDiv.data.over && app.startClickWidget == nil && !keys.alt {
		touchPos := ui.win.io.touch.pos
		if appDiv.crop.Inside(touchPos) {
			grid := appDiv.GetCloseCell(touchPos)

			if appDiv.FindFromGridPos(grid.Start) == nil { //no widget under touch
				rect := appDiv.data.Convert(ui.win.Cell(), grid)

				rect.Start = rect.Start.Add(appDiv.canvas.Start)
				ui.buff.AddRect(rect, SAApp_getYellow(), ui.CellWidth(0.03))
				ui.buff.AddText("+", rect, ui.win.fonts.Get(SKYALT_FONT_PATH), SAApp_getYellow(), ui.win.io.GetDPI()/8, OsV2{1, 1}, nil, true)

				if ui.win.io.touch.end {
					app.addWidgetCoord = grid
					ui.Dialog_open("nodes_list", 2)
				}
			}
		}
	}
	app.drawCreateWidget(ui)

	//select/move widget
	if appDiv.data.over {
		grid := appDiv.GetCloseCell(touch.pos)

		var found *SAWidget
		for _, w := range app.act.Subs {
			if w.GetGrid().Inside(grid.Start) {
				found = w
				break
			}
		}

		if found != nil && keys.alt {
			if touch.start {
				foundStart := appDiv.crop.Start.Add(appDiv.data.Convert(ui.win.Cell(), found.GetGrid()).Start)
				app.startClickWidget = found
				app.startClickRel = touch.pos.Sub(foundStart)
				app.act.DeselectAll()
				found.Selected = true
			}

			if touch.end && touch.numClicks > 1 && found.IsGuiLayout() {
				app.act = found //goto layout
			}
		}

		if app.startClickWidget != nil {
			//move
			gridMove := appDiv.GetCloseCell(touch.pos.Sub(app.startClickRel).Add(OsV2{ui.CellWidth(0.5), ui.CellWidth(0.5)}))
			//gridMove := appDiv.GetCloseCell(touch.pos)
			app.startClickWidget.SetGridStart(gridMove.Start)
		}
		//}
	}
	if touch.end {
		if appDiv.data.over && keys.alt && app.startClickWidget == nil { //click outside widgets
			app.act.DeselectAll()
		}
		app.startClickWidget = nil
	}

	//shortcuts
	if ui.edit.uid == nil {
		keys := &ui.buff.win.io.keys

		//delete
		if appDiv.data.over && keys.delete {
			app.act.RemoveSelectedNodes()
		}

	}

}

func (app *SAApp) History(ui *Ui) {
	//init history
	if len(app.history) == 0 {
		app.addHistory()
		app.history_pos = 0
		app.saveIt = false
	}

	lv := ui.GetCall()
	touch := &ui.buff.win.io.touch
	keys := &ui.buff.win.io.keys
	//over := lv.call.data.over

	if lv.call.data.over && ui.win.io.keys.backward {
		app.stepHistoryBack()

	}
	if lv.call.data.over && ui.win.io.keys.forward {
		app.stepHistoryForward()
	}

	if touch.end || keys.hasChanged {
		app.cmpAndAddHistory()
	}

}

var SAStandardWidgets = []string{"layout", "button", "text", "checkbox", "switch", "edit", "combo"}

func (app *SAApp) drawCreateWidget(ui *Ui) {

	if ui.Dialog_start("nodes_list") {
		ui.Div_colMax(0, 5)

		y := 0
		var search string
		ui.Comp_editbox(0, 0, 1, 1, &search, 0, "", app.base.trns.SAVE, search != "", true, true)
		y++

		fns := SAStandardWidgets                    //stds
		fns = append(fns, app.base.server.nodes...) //from /nodes dir

		keys := &ui.buff.win.io.keys

		for _, fn := range fns {
			if search == "" || strings.Contains(fn, search) {

				if keys.enter || ui.Comp_buttonMenu(0, y, 1, 1, fn, "", true, false) > 0 {
					//add new Widget
					nw := app.act.AddWidget(app.addWidgetCoord, fn)
					app.act.DeselectAll()
					nw.Selected = true

					ui.Dialog_close()
					break
				}
				y++
			}
		}

		ui.Dialog_end()
	}
}

func _SAApp_drawColsRowsDialog(name string, items *[]SAWidgetColRow, i int, ui *Ui) bool {

	changed := false
	if ui.Dialog_start(name) {

		ui.Div_col(0, 10)

		//add left/right
		ui.Div_start(0, 0, 1, 1)
		{
			ui.Div_colMax(0, 100)
			ui.Div_colMax(1, 100)
			ui.Div_colMax(2, 100)

			if ui.Comp_buttonLight(0, 0, 1, 1, "Add before", "", i > 0) > 0 {
				*items = append(*items, SAWidgetColRow{})
				copy((*items)[i+1:], (*items)[i:])
				(*items)[i] = InitSAWidgetColRow()
				ui.Dialog_close()
				changed = true
			}

			ui.Comp_text(1, 0, 1, 1, strconv.Itoa(i), 1) //description

			if ui.Comp_buttonLight(2, 0, 1, 1, "Add after", "", true) > 0 {
				*items = append(*items, SAWidgetColRow{})
				copy((*items)[i+2:], (*items)[i+1:])
				(*items)[i+1] = InitSAWidgetColRow()
				ui.Dialog_close()
				changed = true
			}
		}
		ui.Div_end()

		_, _, _, fnshd1, _ := ui.Comp_editbox_desc("Min", 0, 2, 0, 1, 1, 1, &(*items)[i].Min, 1, "", "", false, false, true)
		_, _, _, fnshd2, _ := ui.Comp_editbox_desc("Max", 0, 2, 0, 2, 1, 1, &(*items)[i].Max, 1, "", "", false, false, true)

		ui.Div_start(0, 3, 1, 1)
		{
			ui.Div_colMax(0, 100)
			ui.Div_colMax(1, 100)

			_, _, _, fnshd3, _ := ui.Comp_editbox_desc("Resize", 0, 2, 0, 0, 1, 1, &(*items)[i].ResizeName, 1, "", "Name", false, false, true)
			ui.Comp_text(1, 0, 1, 1, strconv.FormatFloat((*items)[i].Resize, 'f', 2, 64), 0)

			if fnshd1 || fnshd2 || fnshd3 {
				changed = true
			}

		}
		ui.Div_end()

		//remove
		if ui.Comp_button(0, 5, 1, 1, "Remove", "", len(*items) > 1) > 0 {
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

	//level up
	if ui.Comp_buttonIcon(0, 0, 1, 1, "file:apps/base/resources/levelup.png", 0.3, "One level up", app.act.parent != nil) > 0 {
		app.act = app.act.parent
	}

	//list
	{
		var listPathes string
		var listNodes []*SAWidget
		app.root.buildSubsList(&listPathes, &listNodes)
		if len(listPathes) >= 1 {
			listPathes = listPathes[:len(listPathes)-1] //cut last ';'
		}
		combo := 0
		for i, n := range listNodes {
			if app.act == n {
				combo = i
			}
		}
		if ui.Comp_combo(1, 0, 1, 1, &combo, listPathes, "", true, true) {
			app.act = listNodes[combo]
		}
	}

	ui.Comp_text(2, 0, 1, 1, "Press Alt-key to select widgets", 1)

	//short cuts
	if ui.Comp_buttonLight(3, 0, 1, 1, "←", "Back", app.canHistoryBack()) > 0 {
		app.stepHistoryBack()

	}
	if ui.Comp_buttonLight(4, 0, 1, 1, "→", "Forward", app.canHistoryForward()) > 0 {
		app.stepHistoryForward()
	}
}

func (app *SAApp) cmpAndAddHistory() bool {
	if len(app.history) > 0 {

		if app.act == app.root.FindMirror(app.history[app.history_pos], app.history_act[app.history_pos]) {
			if app.root.Cmp(app.history[app.history_pos]) {
				return false //same
			}
		}
	}

	app.addHistory()
	return true
}

func (app *SAApp) addHistory() {
	//cut newer history
	if app.history_pos+1 < len(app.history) {
		app.history = app.history[:app.history_pos+1]
		app.history_act = app.history_act[:app.history_pos+1]
	}

	//add history
	root, _ := app.root.Copy() //err ...
	act := root.FindMirror(app.root, app.act)

	app.history = append(app.history, root)
	app.history_act = append(app.history_act, act)
	app.history_pos++

	app.saveIt = true
	app.exeIt = true
}

func (app *SAApp) recoverHistory() {
	app.root, _ = app.history[app.history_pos].Copy()
	app.act = app.root.FindMirror(app.history[app.history_pos], app.history_act[app.history_pos])
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

func (app *SAApp) Execute(numThreads int) {

	if !app.exeIt {
		return
	}

	app.ops = NewVmOps()
	app.apis = NewVmApis()
	app.prior = 100

	app.root.UpdateExpresions(app)

	app.root.CheckForLoops()

	app.root.ResetExecute()
	app.root.ExecuteSubs(app.base.server, OsMax(numThreads, 1))

	app.exeIt = false
}

func SAApp_getYellow() OsCd {
	return OsCd{204, 204, 0, 255} //...
}
