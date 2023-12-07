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
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type NodeView struct {
	root *Node
	act  *Node //view

	history_act []*Node //JSONs
	history     []*Node //JSONs
	history_pos int
}

func NewNodeView(path string) (*NodeView, error) {
	view := &NodeView{}

	view.root = NewNode(nil, "net")
	view.act = view.root
	view.SetAttrCamZ(1) //default

	//load
	{
		js, err := os.ReadFile(path)
		if err == nil {
			err = json.Unmarshal([]byte(js), view.root)
			if err != nil {
				fmt.Printf("Unmarshal(%s) failed: %v\n", path, err)
			}
		}
		view.root.UpdateParents(nil)
	}

	view.act.GetAttr("name").Value = ""

	//init history
	view.addHistory()
	view.history_pos = 0

	return view, nil
}

func (view *NodeView) Destroy() {
	view.root.Destroy()
}

func (view *NodeView) GetAttrCols() []NodeRenderColRow {
	var cols []NodeRenderColRow
	json.Unmarshal([]byte(view.act.GetAttr("cols").Value), &cols) //err...
	return cols
}
func (view *NodeView) GetAttrRows() []NodeRenderColRow {
	var rows []NodeRenderColRow
	json.Unmarshal([]byte(view.act.GetAttr("rows").Value), &rows) //err...
	return rows
}

func (view *NodeView) SetAttrCols(cols []NodeRenderColRow) {
	jsCols, _ := json.Marshal(cols) //err ...
	view.act.GetAttr("cols").Value = string(jsCols)
}
func (view *NodeView) SetAttrRows(rows []NodeRenderColRow) {
	jsRows, _ := json.Marshal(rows) //err ...
	view.act.GetAttr("rows").Value = string(jsRows)
}

func (view *NodeView) GetAttrCam() OsV2f {
	x, _ := strconv.ParseFloat(view.act.GetAttr("cam_x").Value, 64)
	y, _ := strconv.ParseFloat(view.act.GetAttr("cam_y").Value, 64)
	return OsV2f{float32(x), float32(y)}
}
func (view *NodeView) SetAttrCam(v OsV2f) {
	view.act.GetAttr("cam_x").Value = strconv.FormatFloat(float64(v.X), 'f', 4, 64)
	view.act.GetAttr("cam_y").Value = strconv.FormatFloat(float64(v.Y), 'f', 4, 64)
}

func (view *NodeView) GetAttrCamZ() float32 {
	z, _ := strconv.ParseFloat(view.act.GetAttr("cam_z").Value, 64)
	return float32(z)
}
func (view *NodeView) SetAttrCamZ(v float32) {
	view.act.GetAttr("cam_z").Value = strconv.FormatFloat(float64(v), 'f', 4, 64)
}

func (view *NodeView) cmpAndAddHistory() bool {

	if len(view.history) > 0 {

		if view.act == view.root.FindMirror(view.history[view.history_pos], view.history_act[view.history_pos]) {
			if view.root.Cmp(view.history[view.history_pos]) {
				return false //same
			}
		}
	}

	view.addHistory()
	return true
}

func (view *NodeView) addHistory() {
	//cut newer history
	if view.history_pos < len(view.history) {
		view.history = view.history[:view.history_pos+1]
	}

	//add history
	root, _ := view.root.Copy() //err ...
	act := root.FindMirror(view.root, view.act)

	view.history = append(view.history, root)
	view.history_act = append(view.history_act, act)
	view.history_pos++
}

func (view *NodeView) recoverHistory() {
	view.root, _ = view.history[view.history_pos].Copy()
	view.act = view.root.FindMirror(view.history[view.history_pos], view.history_act[view.history_pos])
}

func (view *NodeView) canHistoryBack() bool {
	return view.history_pos > 0
}
func (view *NodeView) canHistoryForward() bool {
	return view.history_pos+1 < len(view.history)
}

func (view *NodeView) stepHistoryBack() bool {
	if !view.canHistoryBack() {
		return false
	}

	view.history_pos--
	view.recoverHistory()
	return true
}
func (view *NodeView) stepHistoryForward() bool {

	if !view.canHistoryForward() {
		return false
	}

	view.history_pos++
	view.recoverHistory()
	return true
}

func (view *NodeView) Save(path string) error {
	if path == "" {
		return nil
	}

	js, err := json.MarshalIndent(view.root, "", "")
	if err != nil {
		return fmt.Errorf("MarshalIndent() failed: %w", err)
	}

	err = os.WriteFile(path, js, 0644)
	if err != nil {
		return fmt.Errorf("WriteFile() failed: %w", err)
	}

	return nil
}

func (view *NodeView) RemoveSelectedNodes() {

	for i := len(view.act.Subs) - 1; i >= 0; i-- {
		if view.act.Subs[i].Selected {
			view.act.Subs = append(view.act.Subs[:i], view.act.Subs[i+1:]...)
		}
	}
}

func (view *NodeView) BypassSelectedNodes() {
	for _, n := range view.act.Subs {
		if n.Selected {
			n.Bypass = !n.Bypass
		}
	}
}

type NodeRenderColRow struct {
	Min, Max, Resize float64 `json:",omitempty"`
	ResizeName       string  `json:",omitempty"`
}

func InitNodeRenderColRow() NodeRenderColRow {
	return NodeRenderColRow{Min: 1, Max: 1, Resize: 1}
}

func (view *NodeView) renderApp(ui *Ui, renderRoot bool) {

	act_backup := view.act

	if renderRoot {
		view.act = view.root
	}

	cols := view.GetAttrCols()
	rows := view.GetAttrRows()

	changed := false
	for i, c := range cols {
		ui.Div_col(i, c.Min)
		ui.Div_colMax(i, c.Max)
		if c.ResizeName != "" {
			active, v := ui.Div_colResize(i, c.ResizeName, c.Resize, true)
			if active {
				changed = true
				cols[i].Resize = v
			}
		}
	}

	for i, r := range rows {
		ui.Div_row(i, r.Min)
		ui.Div_rowMax(i, r.Max)
		if r.ResizeName != "" {
			active, v := ui.Div_rowResize(i, r.ResizeName, r.Resize, true)
			if active {
				changed = true
				rows[i].Resize = v
			}
		}
	}

	for _, n := range view.act.Subs {
		if n.Bypass {
			continue
		}

		if strings.HasPrefix(n.FnName, "gui_") {
			grid := InitOsV4(n.GetInputInt("grid_x"), n.GetInputInt("grid_y"), n.GetInputInt("grid_w"), n.GetInputInt("grid_h"))

			switch n.FnName {

			case "gui_sub":
				ui.Div_start(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y)

				sub := NodeView{act: n}
				sub.renderApp(ui, false)

				ui.Div_end()

			case "gui_text":
				n.GetInput("align").Gui_type = "combo"
				n.GetInput("align").Gui_options = "Left|Center|Right"

				ui.Comp_text(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, n.GetInputString("label"), n.GetInputInt("align"))

			case "gui_edit":
				ui.Comp_editbox(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &n.GetAttr("value").Value, 3, "", "", false, false, true)

				//...................
				//1) separtors(+ labels: Attr/Inputs/Outputs) ...
				//2) měnit pořadí(d&d) attr/ins/outs ...

				//5) try node_sqlite? ...

				//case "gui_checkbox" ...
			}

			if !renderRoot && n.Selected && (view.root != nil && view.act == n.parent) {
				ui.Div_start(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y)
				ui.Paint_rect(0, 0, 1, 1, 0, Node_getYellow(), 0.03)
				ui.Div_end()
			}

			//alt+click => select node and zoom_in network ...
		}
	}

	if changed {
		view.SetAttrCols(cols)
		view.SetAttrRows(rows)
	}

	view.act = act_backup
}

func drawColsRowsDialog(name string, items *[]NodeRenderColRow, i int, ui *Ui) bool {

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
				*items = append(*items, NodeRenderColRow{})
				copy((*items)[i+1:], (*items)[i:])
				(*items)[i] = InitNodeRenderColRow()
				ui.Dialog_close()
				changed = true
			}

			ui.Comp_text(1, 0, 1, 1, strconv.Itoa(i), 1) //description

			if ui.Comp_buttonLight(2, 0, 1, 1, "Add after", "", true) > 0 {
				*items = append(*items, NodeRenderColRow{})
				copy((*items)[i+2:], (*items)[i+1:])
				(*items)[i+1] = InitNodeRenderColRow()
				ui.Dialog_close()
				changed = true
			}
		}
		ui.Div_end()

		_, _, _, fnshd1 := ui.Comp_editbox_desc("Min", 0, 2, 0, 1, 1, 1, &(*items)[i].Min, 1, "", "", false, false, true)
		_, _, _, fnshd2 := ui.Comp_editbox_desc("Max", 0, 2, 0, 2, 1, 1, &(*items)[i].Max, 1, "", "", false, false, true)

		ui.Div_start(0, 3, 1, 1)
		{
			ui.Div_colMax(0, 100)
			ui.Div_colMax(1, 100)

			_, _, _, fnshd3 := ui.Comp_editbox_desc("Resize", 0, 2, 0, 0, 1, 1, &(*items)[i].ResizeName, 1, "", "Name", false, false, true)
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

func (view *NodeView) renderAppIDE(ui *Ui) {

	ui.Div_colMax(1, 100)
	ui.Div_rowMax(1, 100)

	var colDiv *UiLayoutDiv
	var rowDiv *UiLayoutDiv

	cols := view.GetAttrCols()
	rows := view.GetAttrRows()
	changed := false

	//at least one
	if len(cols) == 0 {
		cols = append(cols, InitNodeRenderColRow())
		changed = true
	}
	if len(rows) == 0 {
		rows = append(rows, InitNodeRenderColRow())
		changed = true
	}

	if ui.Comp_button(0, 0, 1, 1, "+", "Add Column/Row", true) > 0 {
		ui.Dialog_open("add_col_row", 1)
	}
	if ui.Dialog_start("add_col_row") {
		ui.Div_col(0, 4)
		if ui.Comp_buttonMenu(0, 0, 1, 1, "Add new Column", "", true, false) > 0 {
			cols = append(cols, InitNodeRenderColRow())
			changed = true
		}
		if ui.Comp_buttonMenu(0, 1, 1, 1, "Add new Row", "", true, false) > 0 {
			rows = append(rows, InitNodeRenderColRow())
			changed = true
		}
		ui.Dialog_end()
	}

	//cols header
	ui.Div_start(1, 0, 1, 1)
	{
		colDiv = ui.GetCall().call
		ui.DivInfo_set(SA_DIV_SET_scrollHshow, 0, 0)
		ui.DivInfo_set(SA_DIV_SET_scrollVshow, 0, 0)

		for i, c := range cols {
			ui.Div_col(i, c.Min)
			ui.Div_colMax(i, c.Max)
			if c.ResizeName != "" {
				active, v := ui.Div_colResize(i, c.ResizeName, c.Resize, true)
				if active {
					cols[i].Resize = v
					changed = true
				}
			}
		}

		for i := range cols {
			nm := fmt.Sprintf("col_details_%d", i)

			//drag & drop
			ui.Div_start(i, 0, 1, 1)
			{
				ui.Div_drag("cols", i)
				src, pos, done := ui.Div_drop("cols", false, true, false)
				if done {
					Div_DropMoveElement(&cols, &cols, src, i, pos)
					changed = true
				}
			}
			ui.Div_end()

			if ui.Comp_buttonLight(i, 0, 1, 1, fmt.Sprintf("%d", i), "", true) > 0 {
				ui.Dialog_open(nm, 1)
			}

			if drawColsRowsDialog(nm, &cols, i, ui) {
				changed = true
			}
		}
	}
	ui.Div_end()

	//rows header
	ui.Div_start(0, 1, 1, 1)
	{
		rowDiv = ui.GetCall().call
		ui.DivInfo_set(SA_DIV_SET_scrollHshow, 0, 0)
		ui.DivInfo_set(SA_DIV_SET_scrollVshow, 0, 0)

		for i, r := range rows {
			ui.Div_row(i, r.Min)
			ui.Div_rowMax(i, r.Max)

			if r.ResizeName != "" {
				active, v := ui.Div_rowResize(i, r.ResizeName, r.Resize, true)
				if active {
					rows[i].Resize = v
					changed = true
				}
			}
		}

		for i := range rows {

			nm := fmt.Sprintf("row_details_%d", i)

			//drag & drop
			ui.Div_start(0, i, 1, 1)
			{
				ui.Div_drag("rows", i)
				src, pos, done := ui.Div_drop("rows", true, false, false)
				if done {
					Div_DropMoveElement(&rows, &rows, src, i, pos)
					changed = true
				}
			}
			ui.Div_end()

			if ui.Comp_buttonLight(0, i, 1, 1, fmt.Sprintf("%d", i), "", true) > 0 {
				ui.Dialog_open(nm, 1)
			}
			if drawColsRowsDialog(nm, &rows, i, ui) {
				changed = true
			}
		}

	}
	ui.Div_end()

	//app
	ui.Div_start(1, 1, 1, 1)
	{
		ui.GetCall().call.data.scrollH.attach = &colDiv.data.scrollH
		ui.GetCall().call.data.scrollV.attach = &rowDiv.data.scrollV

		view.renderApp(ui, false)
	}
	ui.Div_end()

	if changed {
		view.SetAttrCols(cols)
		view.SetAttrRows(rows)
	}
}

func (view *NodeView) pixelsToNode(touchPos OsV2, ui *Ui, lvDiv *UiLayoutDiv) OsV2f {

	cell := ui.win.Cell()

	p := touchPos.Sub(lvDiv.canvas.Start).Sub(lvDiv.canvas.Size.MulV(0.5))

	zoom := view.GetAttrCamZ()
	var r OsV2f
	r.X = float32(p.X) / zoom / float32(cell)
	r.Y = float32(p.Y) / zoom / float32(cell)

	r = r.Add(view.GetAttrCam())

	return r
}

func (view *NodeView) nodeToPixels(p OsV2f, ui *Ui) OsV2 {

	lv := ui.GetCall()
	cell := ui.win.Cell()

	p = p.Sub(view.GetAttrCam())

	zoom := view.GetAttrCamZ()
	var r OsV2
	r.X = int(p.X * float32(cell) * zoom)
	r.Y = int(p.Y * float32(cell) * zoom)

	return r.Add(lv.call.canvas.Start).Add(lv.call.canvas.Size.MulV(0.5))
}

func (view *NodeView) cellZoom(ui *Ui) int {
	return int(float32(ui.win.Cell()) * view.GetAttrCamZ() * 1)
}

func (view *NodeView) findInputOver(end bool, ui *Ui) (*Node, *NodeParamIn) {

	lv := ui.GetCall()
	if !lv.call.data.over {
		return nil, nil
	}

	touch := &ui.buff.win.io.touch

	var node_found *Node
	var param_found *NodeParamIn

	for _, n := range view.act.Subs {

		//inputs
		for _, in := range n.Inputs {
			if in.coordDot.Inside(touch.pos) || (end && in.coordLabel.Inside(touch.pos)) {
				node_found = n
				param_found = in
			}
		}
	}

	return node_found, param_found
}
func (view *NodeView) findOutputOver(end bool, ui *Ui) (*Node, *NodeParamOut) {

	lv := ui.GetCall()
	if !lv.call.data.over {
		return nil, nil
	}

	touch := &ui.buff.win.io.touch

	var node_found *Node
	var param_found *NodeParamOut

	for _, n := range view.act.Subs {
		//attributes
		for _, attr := range n.Attrs {
			if attr.coordDot.Inside(touch.pos) || (end && attr.coordLabel.Inside(touch.pos)) {
				node_found = n
				param_found = attr
			}
		}

		//outputs
		for _, out := range n.outputs {
			if out.coordDot.Inside(touch.pos) || (end && out.coordLabel.Inside(touch.pos)) {
				node_found = n
				param_found = out
			}
		}
	}

	return node_found, param_found
}
