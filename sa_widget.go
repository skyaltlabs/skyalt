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
	"sync"
	"sync/atomic"
	"time"
)

type SAWidgetValue struct {
	widget *SAWidget

	Name      string
	Value     string
	oldValue  string
	expValue  *SAWidgetValue
	expWidget *SAWidget
	err       error

	Gui_type     string `json:",omitempty"`
	Gui_options  string `json:",omitempty"`
	Gui_ReadOnly bool   `json:",omitempty"` //output

	editExp bool
}

type SAWidgetColRow struct {
	Min, Max, Resize float64 `json:",omitempty"`
	ResizeName       string  `json:",omitempty"`
}

func (a *SAWidgetColRow) Cmp(b *SAWidgetColRow) bool {
	return a.Min == b.Min && a.Max == b.Max && a.Resize == b.Resize && a.ResizeName == b.ResizeName
}

func InitSAWidgetColRow() SAWidgetColRow {
	return SAWidgetColRow{Min: 1, Max: 1, Resize: 1}
}

type SAWidget struct {
	parent *SAWidget

	Name     string
	Node     string
	Selected bool

	Values []*SAWidgetValue `json:",omitempty"`

	//sub-layout
	Cols []SAWidgetColRow `json:",omitempty"`
	Rows []SAWidgetColRow `json:",omitempty"`
	Subs []*SAWidget      `json:",omitempty"`

	//depends []*SAWidget
	//changed bool
	//outputs[]*SAWidget
	running bool
	done    bool
	err     error
	loop_id int
}

func NewSAWidget(parent *SAWidget, name string, node string, coord OsV4) *SAWidget {
	w := &SAWidget{}
	w.parent = parent
	w.Name = name
	w.Node = node
	w.SetGrid(coord)
	return w
}

func NewSAWidgetRoot(path string) (*SAWidget, error) {
	w := NewSAWidget(nil, "root", "gui_layout", OsV4{})

	//load
	{
		js, err := os.ReadFile(path)
		if err == nil {
			err = json.Unmarshal([]byte(js), w)
			if err != nil {
				fmt.Printf("Unmarshal(%s) failed: %v\n", path, err)
			}
		}
		w.UpdateParents(nil)
	}

	return w, nil
}

func (w *SAWidget) Save(path string) error {
	if path == "" {
		return nil
	}

	js, err := json.MarshalIndent(w, "", "")
	if err != nil {
		return fmt.Errorf("MarshalIndent() failed: %w", err)
	}

	err = os.WriteFile(path, js, 0644)
	if err != nil {
		return fmt.Errorf("WriteFile() failed: %w", err)
	}

	return nil
}

func (a *SAWidget) Cmp(b *SAWidget) bool {
	if a.Name != b.Name || a.Node != b.Node || !a.GetGrid().Cmp(b.GetGrid()) || a.Selected != b.Selected {
		return false
	}

	if len(a.Values) != len(b.Values) {
		return false
	}
	if len(a.Cols) != len(b.Cols) {
		return false
	}
	if len(a.Rows) != len(b.Rows) {
		return false
	}
	if len(a.Subs) != len(b.Subs) {
		return false
	}

	for i, itA := range a.Values {
		itB := b.Values[i]
		if itA.Name != itB.Name || itA.Value != itB.Value || itA.Gui_type != itB.Gui_type || itA.Gui_options != itB.Gui_options || itA.Gui_ReadOnly != itB.Gui_ReadOnly {
			return false
		}
	}

	for i, itA := range a.Cols {
		if !itA.Cmp(&b.Cols[i]) {
			return false
		}
	}
	for i, itA := range a.Rows {
		if !itA.Cmp(&b.Rows[i]) {
			return false
		}
	}

	for i, itA := range a.Subs {
		if !itA.Cmp(b.Subs[i]) {
			return false
		}
	}

	return true
}

func (w *SAWidget) HasError() bool {
	if w.err != nil {
		return true
	}

	for _, it := range w.Values {
		if it.err != nil {
			return true
		}
	}
	return false
}

func (w *SAWidget) UpdateExpresions() {

	for _, it := range w.Values {
		it.expWidget = nil
		it.expValue = nil
		it.err = nil

		var found bool
		for found {
			it.Value, found = strings.CutPrefix(it.Value, " ")
		}

		val, found := strings.CutPrefix(it.Value, "=")
		if found {
			val = strings.ReplaceAll(val, " ", "")
			vals := strings.Split(val, ".")

			if len(vals) == 2 {
				it.expWidget = w.parent.FindNode(vals[0])
				if it.expWidget != nil {
					it.expValue = it.expWidget.findValue(vals[1])
					if it.expValue == nil {
						it.err = fmt.Errorf("Value(%s) not found", vals[1])
					}
				} else {
					it.err = fmt.Errorf("Widget(%s) not found", vals[0])
				}
			} else {
				it.err = fmt.Errorf("too short, use: node.value")
			}
		}
	}

	for _, it := range w.Subs {
		it.UpdateExpresions()
	}
}

func (w *SAWidget) ResetLoopId() {
	w.loop_id = 0
	for _, it := range w.Subs {
		it.ResetLoopId()
	}
}

func (w *SAWidget) checkForLoopInner(loop_id int) {

	w.loop_id = loop_id

	for _, it := range w.Values {
		if it.expValue != nil {

			if it.expWidget != w && it.expWidget.loop_id == loop_id {
				it.err = fmt.Errorf("Loop")
				continue //avoid infinite recursion
			}

			it.expWidget.checkForLoopInner(loop_id)
		}
	}
}

func (w *SAWidget) CheckForLoops(loop_id int) {

	w.checkForLoopInner(loop_id)

	for _, it := range w.Subs {
		loop_id++
		it.CheckForLoops(loop_id)
	}
}

// inputs -> values ...
func (w *SAWidget) areInputsErrorFree() bool {

	for _, v := range w.Values {
		if v.expWidget != nil {
			if v.expWidget.err != nil {
				w.err = fmt.Errorf("incomming error for Value(%s)", v.Name)
				return false
			}
		}
	}

	return true
}

func (w *SAWidget) areInputsReadyToRun() bool {
	for _, v := range w.Values {
		if v.expWidget != nil && (v.expWidget.running || !v.expWidget.done) {
			return false //still running
		}
	}
	return true
}

func (w *SAWidget) areInputsChanged() bool {

	for _, it := range w.Values {
		value := it.Value
		if it.expValue != nil {
			value = it.expValue.oldValue
		}

		changed := (value != it.oldValue)

		it.oldValue = value
		if changed {
			return true
		}
	}
	return false
}

func (w *SAWidget) IsReadyToBeExe() bool {

	if w.done && !w.running {
		return false
	}

	if w.areInputsErrorFree() {
		if w.areInputsReadyToRun() {
			w.done = true
			return w.areInputsChanged()
		}
	} else {
		w.done = true
	}

	return false
}

func (w *SAWidget) ResetExecute() {
	w.done = false
	w.running = false
	w.err = nil

	for _, it := range w.Subs {
		it.ResetExecute()
	}
}

func (w *SAWidget) buildList(list *[]*SAWidget) {
	*list = append(*list, w)
	for _, it := range w.Subs {
		it.buildList(list)
	}
}
func (w *SAWidget) ExecuteSubs(server *NodeServer, max_threads int) {
	var list []*SAWidget
	w.buildList(&list)

	//multi-thread executing
	var numActiveThreads atomic.Int64
	var wg sync.WaitGroup
	run := true
	for run && server.IsRunning() {
		run = false
		for _, it := range list {

			if !it.done {
				run = true
				if it.IsReadyToBeExe() {

					//maximum concurent threads
					if numActiveThreads.Load() >= int64(max_threads) {
						time.Sleep(10 * time.Millisecond)
					}

					//run it
					it.running = true
					wg.Add(1)
					go func(ww *SAWidget) {
						numActiveThreads.Add(1)
						ww.execute(server)
						wg.Done()
						numActiveThreads.Add(-1)
					}(it)

				}
			}
		}
	}
}

func (w *SAWidget) CanSkipExecute() bool {
	return strings.HasPrefix(w.Node, "gui_")
}

func (w *SAWidget) execute(server *NodeServer) {

	if !w.CanSkipExecute() {

		fmt.Println(w.Name)

		nc := server.Start(w.Node)
		if nc != nil {

			//add/update new
			/*for _, in := range nc.strct.Attrs {
				a := node.GetAttr(in.Name)
				a.Gui_type = in.Gui_type
				a.Gui_options = in.Gui_options
				a.Hide = in.Hide
			}

			//set/remove
			for i := len(node.Attrs) - 1; i >= 0; i-- {
				src := node.Attrs[i]
				dst := nc.FindAttr(src.Name)
				if dst != nil {
					dst.Value = src.Value
				} else {
					node.Attrs = append(node.Attrs[:i], node.Attrs[i+1:]...) //remove
				}
			}
			for i := len(node.Inputs) - 1; i >= 0; i-- {
				src := node.Inputs[i]
				dst := nc.FindInput(src.Name)
				if dst != nil {
					out := src.FindWireOut()
					if out != nil {
						dst.Value = out.Value
					} else {
						dst.Value = src.Value
					}
				} else {
					node.Inputs = append(node.Inputs[:i], node.Inputs[i+1:]...) //remove
				}
			}

			//execute
			nc.Start()

			//copy back
			node.outputs = nil //reset
			for _, in := range nc.strct.Outputs {
				o := node.GetOutput(in.Name) //add
				o.Value = in.Value
				o.Gui_type = in.Gui_type
				o.Gui_options = in.Gui_options
			}

			if nc.progress.Error != "" {
				node.err = errors.New(nc.progress.Error)
			}*/
		} else {
			w.err = fmt.Errorf("can't find node exe(%s)", w.Node)
		}

		if w.err != nil {
			fmt.Printf("Node(%s) has error(%v)\n", w.Name, w.err)
		}
	}

	w.done = true
	w.running = false
}

func (w *SAWidget) FindNode(name string) *SAWidget {
	for _, it := range w.Subs {
		if it.Name == name {
			return it
		}
	}
	return nil
}

func (w *SAWidget) UpdateParents(parent *SAWidget) {
	w.parent = parent

	for _, it := range w.Subs {
		it.UpdateParents(w)
	}
}

func (w *SAWidget) Copy() (*SAWidget, error) {

	js, err := json.Marshal(w)
	if err != nil {
		return nil, err
	}

	dst := NewSAWidget(nil, "", "", OsV4{})
	err = json.Unmarshal(js, dst)
	if err != nil {
		return nil, err
	}

	dst.UpdateParents(nil)

	return dst, nil
}

func (a *SAWidget) FindMirror(b *SAWidget, b_act *SAWidget) *SAWidget {

	if b == b_act {
		return a
	}
	for i, na := range a.Subs {
		ret := na.FindMirror(b.Subs[i], b_act)
		if ret != nil {
			return ret
		}
	}
	return nil
}

func (w *SAWidget) DeselectAll() {
	for _, n := range w.Subs {
		n.Selected = false
	}
}

func (w *SAWidget) RemoveSelectedNodes() {
	for i := len(w.Subs) - 1; i >= 0; i-- {
		if w.Subs[i].Selected {
			w.Subs = append(w.Subs[:i], w.Subs[i+1:]...)
		}
	}
}

func (w *SAWidget) FindSelected() *SAWidget {

	for _, it := range w.Subs {
		if it.Selected {
			return it
		}
	}
	return nil
}

func (w *SAWidget) IsGuiLayout() bool {
	return w.Node == "gui_layout"
}

func (w *SAWidget) buildSubsList(listPathes *string, listNodes *[]*SAWidget) {
	nm := w.getPath()
	if len(nm) > 2 {
		nm = nm[:len(nm)-1] //cut last '/'
	}

	*listPathes += nm + "|"
	*listNodes = append(*listNodes, w)

	for _, n := range w.Subs {
		if n.IsGuiLayout() {
			n.buildSubsList(listPathes, listNodes)
		}
	}
}

func (w *SAWidget) getPath() string {

	var path string

	if w.parent != nil {
		path += w.parent.getPath()
	}

	path += w.Name + "/"

	return path
}

func (w *SAWidget) AddWidget(coord OsV4, name string, node string) *SAWidget {
	nw := NewSAWidget(w, name, node, coord)
	w.Subs = append(w.Subs, nw)
	return nw
}

func (w *SAWidget) Remove() bool {

	if w.parent != nil {
		for i, it := range w.parent.Subs {
			if it == w {
				w.parent.Subs = append(w.parent.Subs[:i], w.parent.Subs[i+1:]...)
				return true
			}
		}
	}
	return false
}

func (w *SAWidget) findValue(name string) *SAWidgetValue {
	for _, it := range w.Values {
		if it.Name == name {
			return it
		}
	}
	return nil
}
func (w *SAWidget) _getValue(defValue SAWidgetValue) *SAWidgetValue {

	v := w.findValue(defValue.Name)
	if v == nil {
		v = &SAWidgetValue{}
		*v = defValue
		v.widget = w
		w.Values = append(w.Values, v)
	}

	return v
}

func (w *SAWidget) GetValueStringEdit(name string, defValue string) string {
	return w._getValue(SAWidgetValue{Name: name, Value: defValue, Gui_type: "gui_edit"}).oldValue
}

func (w *SAWidget) GetValueStringPtrEdit(name string, defValue string) *string {
	return &w._getValue(SAWidgetValue{Name: name, Value: defValue, Gui_type: "gui_edit"}).oldValue
}

func (w *SAWidget) GetValueIntEdit(name string, defValue string) int {
	vv, _ := strconv.Atoi(w.GetValueStringEdit(name, defValue))
	return vv
}

func (w *SAWidget) GetValueIntCombo(name string, defValue string, defOptions string) int {
	vv, _ := strconv.Atoi(w._getValue(SAWidgetValue{Name: name, Value: defValue, Gui_type: "gui_combo", Gui_options: defOptions}).oldValue)
	return vv
}

func (w *SAWidget) GetValueBoolSwitch(name string, defValue string) bool {
	vv, _ := strconv.Atoi(w._getValue(SAWidgetValue{Name: name, Value: defValue, Gui_type: "gui_switch"}).oldValue)
	return vv != 0
}

func (w *SAWidget) SetGridStart(v OsV2) {
	*w.GetValueStringPtrEdit("grid_x", "0") = strconv.Itoa(v.X)
	*w.GetValueStringPtrEdit("grid_y", "0") = strconv.Itoa(v.Y)
}
func (w *SAWidget) SetGridSize(v OsV2) {
	*w.GetValueStringPtrEdit("grid_w", "1") = strconv.Itoa(v.X)
	*w.GetValueStringPtrEdit("grid_h", "1") = strconv.Itoa(v.Y)
}

func (w *SAWidget) SetGrid(coord OsV4) {
	w.SetGridStart(coord.Start)
	w.SetGridStart(coord.Size)
}
func (w *SAWidget) GetGrid() OsV4 {
	var v OsV4
	v.Start.X = w.GetValueIntEdit("grid_x", "0")
	v.Start.Y = w.GetValueIntEdit("grid_y", "0")
	v.Size.X = w.GetValueIntEdit("grid_w", "1")
	v.Size.Y = w.GetValueIntEdit("grid_h", "1")
	return v
}

func (w *SAWidget) Render(ui *Ui, app *SAApp2) {

	grid := w.GetGrid()

	switch w.Node {
	case "gui_button":
		ui.Comp_button(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, w.GetValueStringEdit("label", ""), "", w.GetValueBoolSwitch("enable", "1"))

		//outputs(readOnly): is clicked ...

	case "gui_text":
		ui.Comp_text(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, w.GetValueStringEdit("label", ""), w.GetValueIntCombo("align", "0", "Left|Center|Right"))

	case "gui_edit":
		ui.Comp_editbox(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, w.GetValueStringPtrEdit("value", ""), w.GetValueIntEdit("precision", "2"), "", w.GetValueStringEdit("ghost", ""), false, w.GetValueBoolSwitch("tempToValue", "0"), w.GetValueBoolSwitch("enable", "1"))

	case "gui_switch":
		ui.Comp_switch(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, w.GetValueStringPtrEdit("value", ""), false, w.GetValueStringEdit("label", ""), "", w.GetValueBoolSwitch("enable", "1"))

	case "gui_checkbox":
		ui.Comp_checkbox(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, w.GetValueStringPtrEdit("value", ""), false, w.GetValueStringEdit("label", ""), "", w.GetValueBoolSwitch("enable", "1"))

	case "gui_combo":
		ui.Comp_combo(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, w.GetValueStringPtrEdit("value", ""), w.GetValueStringEdit("options", "a|b"), "", w.GetValueBoolSwitch("enable", "1"), w.GetValueBoolSwitch("search", "0"))

	case "gui_layout":
		ui.Div_start(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y)
		w.RenderLayout(ui, app)
		ui.Div_end()
	}

	if app.IDE {
		//draw Select rectangle
		if w.Selected && app.act == w.parent {
			div := ui.Div_start(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y)
			div.enableInput = false
			ui.Paint_rect(0, 0, 1, 1, 0, Node_getYellow(), 0.03)
			ui.Div_end()
		}

	}
}

func (w *SAWidget) RenderLayout(ui *Ui, app *SAApp2) {

	//columns
	for i, c := range w.Cols {
		ui.Div_col(i, c.Min)
		ui.Div_colMax(i, c.Max)
		if c.ResizeName != "" {
			active, v := ui.Div_colResize(i, c.ResizeName, c.Resize, true)
			if active {
				w.Cols[i].Resize = v
			}
		}
	}

	//rows
	for i, r := range w.Rows {
		ui.Div_row(i, r.Min)
		ui.Div_rowMax(i, r.Max)
		if r.ResizeName != "" {
			active, v := ui.Div_rowResize(i, r.ResizeName, r.Resize, true)
			if active {
				w.Rows[i].Resize = v
			}
		}
	}

	//items
	for _, it := range w.Subs {
		it.Render(ui, app)
	}
}

func (w *SAWidget) NumNames(name string) int {

	n := 0
	for _, it := range w.Subs {
		if it.Name == name {
			n++
		}
	}
	return n

}

func _SAWidget_renderParamsValue(x, y, w, h int, it *SAWidgetValue, ui *Ui) {
	if it.editExp {
		ui.Comp_editbox(x, y, w, h, &it.Value, 2, "", "", false, false, true)
	} else {
		value := &it.Value
		if it.expValue != nil {
			value = &it.oldValue
		}
		enable := it.expValue == nil

		switch it.Gui_type {
		case "gui_checkbox":
			ui.Comp_checkbox(x, y, w, h, value, false, "", "", enable)

		case "gui_switch":
			ui.Comp_switch(x, y, w, h, value, false, "", "", enable)

		case "gui_edit":
			ui.Comp_editbox(x, y, w, h, value, 2, "", "", false, false, enable)

		case "gui_combo":
			ui.Comp_combo(x, y, w, h, value, it.Gui_options, "", enable, false)
		}
	}
}

func (w *SAWidget) RenderParams(ui *Ui) {

	ui.Div_colMax(0, 100)
	ui.Div_row(2, 0.5) //spacer

	y := 0

	//Name
	ui.Div_start(0, y, 1, 1)
	{
		ui.Div_colMax(0, 100)
		ui.Div_colMax(1, 2)
		_, _, _, fnshd, _ := ui.Comp_editbox_desc("Name", 0, 2, 0, 0, 1, 1, &w.Name, 0, "", "Name", false, false, true)
		if fnshd && w.parent != nil {
			//create unique name
			for w.parent.NumNames(w.Name) >= 2 {
				w.Name += "1"
			}
		}

		if ui.Comp_button(1, 0, 1, 1, "Delete", "", true) > 0 {
			w.Remove()
		}
	}
	ui.Div_end()
	y++

	//Grid
	ui.Div_start(0, y, 1, 1)
	{
		ui.Div_colMax(0, 2)
		ui.Div_colMax(1, 100)
		ui.Div_colMax(2, 100)
		ui.Div_colMax(3, 100)
		ui.Div_colMax(4, 100)

		//label
		w.GetValueIntEdit("grid_x", "0") //create if not exist
		w.GetValueIntEdit("grid_y", "0")
		w.GetValueIntEdit("grid_w", "1")
		w.GetValueIntEdit("grid_h", "1")
		//find
		xx := w.findValue("grid_x")
		yy := w.findValue("grid_y")
		ww := w.findValue("grid_w")
		hh := w.findValue("grid_h")
		if ui.Comp_buttonMenu(0, 0, 1, 1, "Grid", "", true, xx.editExp) > 0 {
			xx.editExp = !xx.editExp
			yy.editExp = !yy.editExp
			ww.editExp = !ww.editExp
			hh.editExp = !hh.editExp
		}

		//values
		_SAWidget_renderParamsValue(1, 0, 1, 1, xx, ui)
		_SAWidget_renderParamsValue(2, 0, 1, 1, yy, ui)
		_SAWidget_renderParamsValue(3, 0, 1, 1, ww, ui)
		_SAWidget_renderParamsValue(4, 0, 1, 1, hh, ui)
	}
	ui.Div_end()
	y++

	ui.Div_SpacerRow(0, y, 1, 1)
	y++

	for _, it := range w.Values {

		if it.Name == "grid_x" || it.Name == "grid_y" || it.Name == "grid_w" || it.Name == "grid_h" {
			continue
		}

		//name_width := ui.Paint_textWidth(name, -1, 0, "", true)

		ui.Div_start(0, y, 1, 1)
		{
			ui.Div_colMax(0, 3)
			ui.Div_colMax(1, 100)

			//error
			if it.err != nil {
				ui.Paint_tooltip(0, 0, 1, 1, it.err.Error())
				pl := ui.buff.win.io.GetPalette()
				ui.Paint_rect(0, 0, 1, 1, 0, pl.E, 0.03)
			}

			//name
			if ui.Comp_buttonMenu(0, 0, 1, 1, it.Name, "", true, it.editExp) > 0 {
				it.editExp = !it.editExp
			}

			//value
			_SAWidget_renderParamsValue(1, 0, 1, 1, it, ui)
		}
		ui.Div_end()
		y++
	}
}

// reoder(d & d) Values ...
// server + execute graph ...
// resize widget ...
