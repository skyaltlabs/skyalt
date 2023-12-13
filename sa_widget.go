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
	"errors"
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

	Name    string
	Value   string
	ShowExp bool

	oldValue     string
	depends      []*SAWidgetValue
	isDirectLink bool
	err          error

	Gui_type     string `json:",omitempty"`
	Gui_options  string `json:",omitempty"`
	Gui_ReadOnly bool   `json:",omitempty"` //output

}

func (v *SAWidgetValue) GetDirectLink() (*SAWidgetValue, *string, bool) {

	for v.isDirectLink {
		return v.depends[0].GetDirectLink() //go to source
	}

	if len(v.depends) > 0 {
		return v, &v.oldValue, false //expression. oldValue = result
	}

	return v, &v.Value, true //this
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

	running bool
	done    bool
	err     error
	//loop_id int
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
		it.depends = nil
		it.isDirectLink = false
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
				ww := w.parent.FindNode(vals[0])
				if ww != nil {
					vv := ww.findValue(vals[1])
					if vv != nil {
						it.depends = append(it.depends, vv) //add
						it.isDirectLink = true              //...
					} else {
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

func (w *SAWidget) checkForLoopInner(find *SAWidget) {

	for _, v := range w.Values {
		for _, dep := range v.depends {
			if dep.widget == find {
				v.err = fmt.Errorf("Loop")
				continue //avoid infinite recursion
			}

			dep.widget.checkForLoopInner(find)
		}
	}
}

func (w *SAWidget) CheckForLoops() {

	w.checkForLoopInner(w)

	for _, it := range w.Subs {
		it.CheckForLoops()
	}
}

// inputs -> values ...
func (w *SAWidget) areInputsErrorFree() bool {

	for _, v := range w.Values {
		for _, dep := range v.depends {
			if dep.widget.HasError() {
				w.err = fmt.Errorf("incomming error for Value(%s)", v.Name)
				return false
			}
		}
	}

	return true
}

func (w *SAWidget) areInputsReadyToRun() bool {
	for _, v := range w.Values {
		for _, dep := range v.depends {
			if dep.widget.running || !dep.widget.done {
				return false //still running
			}
		}
	}
	return true
}

func (w *SAWidget) areInputsChangedAndUpdate() bool {

	changed := false
	for _, it := range w.Values {

		_, value, _ := it.GetDirectLink()

		if *value != it.oldValue {
			changed = true
		}

		it.oldValue = *value
	}
	return changed
}

func (w *SAWidget) IsReadyToBeExe() bool {

	if w.done || w.running {
		return false
	}

	if w.areInputsErrorFree() {
		if w.areInputsReadyToRun() {
			changed := w.areInputsChangedAndUpdate()
			if !changed {
				w.done = true
			}
			return changed
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

func (w *SAWidget) CanSkipExecute() bool {
	return strings.HasPrefix(w.Node, "gui_")
}

func (w *SAWidget) ExecuteSubs(server *SANodeServer, max_threads int) {
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

					if !it.CanSkipExecute() {

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
					} else {
						it.done = true
						it.running = false
					}
				}
			}
		}
	}
}

func (w *SAWidget) execute(server *SANodeServer) {

	fmt.Println("execute:", w.Name)

	nc := server.Start(w.Node)
	if nc != nil {

		//add/update
		for _, v := range nc.strct.Values {
			a := w.GetValue(v.Name, v.Value, v.Gui_type, v.Gui_options, v.Gui_ReadOnly)
			a.Gui_type = v.Gui_type
			a.Gui_options = v.Gui_options
			a.Gui_ReadOnly = v.Gui_ReadOnly
		}

		//set/remove
		for i := len(w.Values) - 1; i >= 0; i-- {
			src := w.Values[i]
			if strings.HasPrefix(src.Name, "grid_") {
				continue
			}
			dst := nc.FindValue(src.Name)
			if dst != nil {
				dst.Value = src.oldValue
			} else {
				w.Values = append(w.Values[:i], w.Values[i+1:]...) //remove
			}
		}

		//execute
		nc.Start()

		//copy back
		for _, v := range nc.strct.Values {
			if v.Gui_ReadOnly {
				a := w.GetValue(v.Name, v.Value, v.Gui_type, v.Gui_options, v.Gui_ReadOnly)
				a.Value = v.Value
				a.Gui_type = v.Gui_type
				a.Gui_options = v.Gui_options
				a.Gui_ReadOnly = v.Gui_ReadOnly
			}
		}

		if nc.progress.Error != "" {
			w.err = errors.New(nc.progress.Error)
		}
	} else {
		w.err = fmt.Errorf("can't find node exe(%s)", w.Node)
	}

	if w.err != nil {
		fmt.Printf("Node(%s) has error(%v)\n", w.Name, w.err)
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

	for _, v := range w.Values {
		v.widget = w
	}

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

// this should be use for Gui_ReadOnly=true
func (w *SAWidget) GetValue(name string, value string, gui_type string, gui_options string, gui_readOnly bool) *SAWidgetValue {
	return w._getValue(SAWidgetValue{Name: name, Value: value, Gui_type: gui_type, Gui_options: gui_options, Gui_ReadOnly: gui_readOnly})
}

func (w *SAWidget) GetValueStringPtrEdit(name string, defValue string) (*string, bool) {
	v := w._getValue(SAWidgetValue{Name: name, Value: defValue, Gui_type: "gui_edit"})
	_, val, editable := v.GetDirectLink()
	return val, editable
}

func (w *SAWidget) GetValueStringEdit(name string, defValue string) string {
	s, _ := w.GetValueStringPtrEdit(name, defValue)
	return *s
}

func (w *SAWidget) GetValueIntEdit(name string, defValue string) int {
	vv, _ := strconv.Atoi(w.GetValueStringEdit(name, defValue))
	return vv
}

func (w *SAWidget) GetValueIntCombo(name string, defValue string, defOptions string) int {
	v := w._getValue(SAWidgetValue{Name: name, Value: defValue, Gui_type: "gui_combo", Gui_options: defOptions})
	_, val, _ := v.GetDirectLink()
	vv, _ := strconv.Atoi(*val)
	return vv
}

func (w *SAWidget) GetValueBoolSwitch(name string, defValue string) bool {
	v := w._getValue(SAWidgetValue{Name: name, Value: defValue, Gui_type: "gui_switch"})
	_, val, _ := v.GetDirectLink()
	vv, _ := strconv.Atoi(*val)
	return vv != 0
}

func (w *SAWidget) SetGridStart(v OsV2) {
	str, edit := w.GetValueStringPtrEdit("grid_x", "0")
	if edit {
		*str = strconv.Itoa(v.X)
	}
	str, edit = w.GetValueStringPtrEdit("grid_y", "0")
	if edit {
		*str = strconv.Itoa(v.Y)
	}
}
func (w *SAWidget) SetGridSize(v OsV2) {
	str, edit := w.GetValueStringPtrEdit("grid_w", "0")
	if edit {
		*str = strconv.Itoa(v.X)
	}
	str, edit = w.GetValueStringPtrEdit("grid_h", "0")
	if edit {
		*str = strconv.Itoa(v.Y)
	}
}

func (w *SAWidget) SetGrid(coord OsV4) {
	w.SetGridStart(coord.Start)
	w.SetGridSize(coord.Size)
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

	//pokud je value expression, tak by mělo být disable, nebo nastavit link hodnotu(co když a*b?) ...

	//co když je: a.x + b.x? Dovoli obojí(disable nebo enable+changeOrigInTheLink) ...

	//=>expValue(expNode) nedávají moc smysl když tam bude a * b? ... 2x depends? ....................................

	case "gui_button":
		enable := w.GetValueBoolSwitch("enable", "1")
		ui.Comp_button(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, w.GetValueStringEdit("label", ""), "", enable)
		//outputs(readOnly): is clicked ...

	case "gui_text":
		ui.Comp_text(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, w.GetValueStringEdit("label", ""), w.GetValueIntCombo("align", "0", "Left|Center|Right"))

	case "gui_edit":
		value, editable := w.GetValueStringPtrEdit("value", "")
		enable := w.GetValueBoolSwitch("enable", "1") && editable
		ui.Comp_editbox(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, value, w.GetValueIntEdit("precision", "2"), "", w.GetValueStringEdit("ghost", ""), false, w.GetValueBoolSwitch("tempToValue", "0"), enable)

	case "gui_switch":
		value, editable := w.GetValueStringPtrEdit("value", "")
		enable := w.GetValueBoolSwitch("enable", "1") && editable
		ui.Comp_switch(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, value, false, w.GetValueStringEdit("label", ""), "", enable)

	case "gui_checkbox":
		value, editable := w.GetValueStringPtrEdit("value", "")
		enable := w.GetValueBoolSwitch("enable", "1") && editable
		ui.Comp_checkbox(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, value, false, w.GetValueStringEdit("label", ""), "", enable)

	case "gui_combo":
		value, editable := w.GetValueStringPtrEdit("value", "")
		enable := w.GetValueBoolSwitch("enable", "1") && editable
		ui.Comp_combo(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, value, w.GetValueStringEdit("options", "a|b"), "", enable, w.GetValueBoolSwitch("search", "0"))

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
			ui.Paint_rect(0, 0, 1, 1, 0, SAApp2_getYellow(), 0.03)
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
	if it.ShowExp {
		ui.Comp_editbox(x, y, w, h, &it.Value, 2, "", "", false, false, true)
	} else {
		_, value, editable := it.GetDirectLink()

		switch it.Gui_type {
		case "gui_checkbox":
			ui.Comp_checkbox(x, y, w, h, value, false, "", "", editable)

		case "gui_switch":
			ui.Comp_switch(x, y, w, h, value, false, "", "", editable)

		case "gui_edit":
			ui.Comp_editbox(x, y, w, h, value, 2, "", "", false, false, editable)

		case "gui_combo":
			ui.Comp_combo(x, y, w, h, value, it.Gui_options, "", editable, false)
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
		if ui.Comp_buttonMenu(0, 0, 1, 1, "Grid", "", true, xx.ShowExp) > 0 {
			xx.ShowExp = !xx.ShowExp
			yy.ShowExp = !yy.ShowExp
			ww.ShowExp = !ww.ShowExp
			hh.ShowExp = !hh.ShowExp
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

			//highlight because it has expression
			if len(it.depends) > 0 {
				ui.Paint_rect(0, 0, 1, 1, 0.03, SAApp2_getYellow().SetAlpha(50), 0)
			}

			//error
			if it.err != nil {
				ui.Paint_tooltip(0, 0, 1, 1, it.err.Error())
				pl := ui.buff.win.io.GetPalette()
				ui.Paint_rect(0, 0, 1, 1, 0, pl.E, 0.03)
			}

			//name
			if ui.Comp_buttonMenu(0, 0, 1, 1, it.Name, "", true, it.ShowExp) > 0 {
				it.ShowExp = !it.ShowExp
			}

			//value
			_SAWidget_renderParamsValue(1, 0, 1, 1, it, ui)
		}
		ui.Div_end()
		y++
	}
}
