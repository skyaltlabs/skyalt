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

type SAWidgetAttr struct {
	widget *SAWidget

	Name    string
	Value   string `json:",omitempty"`
	ShowExp bool

	oldValue     string
	instr        *VmInstr
	depends      []*SAWidgetAttr
	isDirectLink bool
	errExp       error
	errExe       error

	Gui_type     string `json:",omitempty"`
	Gui_options  string `json:",omitempty"`
	Gui_ReadOnly bool   `json:",omitempty"` //output
}

func (attr *SAWidgetAttr) GetDirectLink() (*SAWidgetAttr, *string, bool) {

	for attr.isDirectLink {
		return attr.depends[0].GetDirectLink() //go to source
	}

	if len(attr.depends) > 0 {
		return attr, &attr.oldValue, false //expression. oldValue = result
	}

	return attr, &attr.Value, true //this
}
func (attr *SAWidgetAttr) SetString(value string) {
	_, val, editable := attr.GetDirectLink()
	if editable {
		*val = value
	}
}
func (attr *SAWidgetAttr) SetInt(value int) {
	_, val, editable := attr.GetDirectLink()
	if editable {
		*val = strconv.Itoa(value)
	}
}
func (attr *SAWidgetAttr) SetFloat(value float64) {
	_, val, editable := attr.GetDirectLink()
	if editable {
		*val = strconv.FormatFloat(value, 'f', -1, 64)
	}
}

func (attr *SAWidgetAttr) GetString() string {
	_, val, _ := attr.GetDirectLink()
	return *val
}
func (attr *SAWidgetAttr) GetInt() int {
	v, _ := strconv.Atoi(attr.GetString())
	return v
}
func (attr *SAWidgetAttr) GetInt64() int64 {
	v, _ := strconv.Atoi(attr.GetString())
	return int64(v)
}
func (attr *SAWidgetAttr) GetFloat() float64 {
	v, _ := strconv.ParseFloat(attr.GetString(), 64)
	return v
}
func (attr *SAWidgetAttr) GetBool() bool {
	return attr.GetInt() != 0
}
func (attr *SAWidgetAttr) GetByte() byte {
	return byte(attr.GetInt())
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
	Selected bool
	Exe      string

	Attrs []*SAWidgetAttr `json:",omitempty"`

	//sub-layout
	Cols []SAWidgetColRow `json:",omitempty"`
	Rows []SAWidgetColRow `json:",omitempty"`
	Subs []*SAWidget      `json:",omitempty"`

	running bool
	done    bool
	errExe  error
}

func NewSAWidget(parent *SAWidget, name string, exe string, coord OsV4) *SAWidget {
	w := &SAWidget{}
	w.parent = parent
	w.Name = name
	w.Exe = exe
	w.SetGrid(coord)
	return w
}

func NewSAWidgetRoot(path string) (*SAWidget, error) {
	w := NewSAWidget(nil, "root", "layout", OsV4{})
	w.Exe = "layout"

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
	if a.Name != b.Name || a.Exe != b.Exe || a.Selected != b.Selected {
		return false
	}

	if len(a.Attrs) != len(b.Attrs) {
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

	for i, itA := range a.Attrs {
		itB := b.Attrs[i]
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

func (w *SAWidget) HasExpError() bool {

	for _, it := range w.Attrs {
		if it.errExp != nil {
			return true
		}
	}
	return false
}

func (w *SAWidget) HasExeError() bool {

	if w.errExe != nil {
		return true
	}

	for _, it := range w.Attrs {
		if it.errExe != nil {
			return true
		}
	}
	return false
}

func (w *SAWidget) UpdateExpresions(app *SAApp) {

	for _, it := range w.Attrs {
		it.instr = nil
		it.depends = nil
		it.isDirectLink = false
		it.errExp = nil

		var found bool
		for found {
			it.Value, found = strings.CutPrefix(it.Value, " ")
		}

		val, found := strings.CutPrefix(it.Value, "=")
		if found {
			ln, err := InitVmLine(val, app.ops, app.apis, app.prior, w)
			if err == nil {
				it.instr = ln.Parse()
				if it.instr != nil {
					it.depends = ln.depends
					it.isDirectLink = it.instr.IsDirectLink()
				}
				if len(ln.errs) > 0 {
					it.errExp = errors.New(ln.errs[0])
				}
			} else {
				it.errExp = err
			}
		}
	}

	for _, it := range w.Subs {
		it.UpdateExpresions(app)
	}
}

func (w *SAWidget) checkForLoopInner(find *SAWidget) {

	for _, v := range w.Attrs {
		for _, dep := range v.depends {
			if dep.widget == w {
				continue //skip self-depends
			}

			if dep.widget == find {
				v.errExp = fmt.Errorf("Loop")
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

func (w *SAWidget) areAttrsErrorFree() bool {

	for _, v := range w.Attrs {
		for _, dep := range v.depends {
			if dep.widget.HasExpError() {
				v.errExp = fmt.Errorf("incomming error")
				return false
			}
		}
	}

	return true
}

func (w *SAWidget) areAttrsReadyToRun() bool {
	for _, v := range w.Attrs {
		for _, dep := range v.depends {
			if dep.widget == w {
				continue //skip self-depends
			}

			if dep.widget.running || !dep.widget.done {
				return false //still running
			}
		}
	}
	return true
}

func (w *SAWidget) areAttrsChangedAndUpdate() bool {

	changed := false

	for i := 0; i < 2; i++ { //run 2x, because attributes can depend in same node - dirty hack, need to find better solution ...

		for _, it := range w.Attrs {

			if it.errExp != nil {
				return true
			}

			var val string
			if it.instr != nil && !it.isDirectLink {
				st := InitVmST()
				rec := it.instr.Exe(nil, &st)
				val = rec.GetString()
			} else {
				_, value, _ := it.GetDirectLink()
				val = *value
			}

			if val != it.oldValue {
				changed = true
			}
			it.oldValue = val

		}

	}

	return changed
}

func (w *SAWidget) IsReadyToBeExe() bool {

	if w.done || w.running {
		return false
	}

	if w.areAttrsErrorFree() {
		if w.areAttrsReadyToRun() {
			changed := w.areAttrsChangedAndUpdate()
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

func (w *SAWidget) IsGuiLayout() bool {
	return w.Exe == "layout" || !w.IsGui() //every exe is layout
}
func (w *SAWidget) IsGui() bool {
	if w.Exe == "" {
		return true
	}
	return SAApp_IsStdPrimitive(w.Exe)
}
func (w *SAWidget) IsExe() bool {
	if w.Exe == "" {
		return false
	}
	if SAApp_IsStdPrimitive(w.Exe) {
		return false
	}
	if SAApp_IsStdComponent(w.Exe) {
		return false
	}
	return true
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
					if it.IsExe() {

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

	nc := server.Start(w.Exe)
	if nc != nil {
		//add/update
		for _, v := range nc.Attrs {
			v.Error = ""

			a := w.GetAttr(v.Name, v.Value, v.Gui_type, v.Gui_options, v.Gui_ReadOnly)
			a.Gui_type = v.Gui_type
			a.Gui_options = v.Gui_options
			a.Gui_ReadOnly = v.Gui_ReadOnly
			a.errExe = nil
		}

		//set/remove
		for i := len(w.Attrs) - 1; i >= 0; i-- {
			src := w.Attrs[i]
			if strings.HasPrefix(src.Name, "grid_") {
				continue
			}
			dst := nc.FindAttr(src.Name)
			if dst != nil {
				dst.Value = src.oldValue
			} else {
				w.Attrs = append(w.Attrs[:i], w.Attrs[i+1:]...) //remove
			}
		}

		//execute
		nc.Start()

		//copy back
		for _, v := range nc.Attrs {
			a := w.GetAttr(v.Name, v.Value, v.Gui_type, v.Gui_options, v.Gui_ReadOnly)
			if v.Gui_ReadOnly {
				a.Value = v.Value
				a.Gui_type = v.Gui_type
				a.Gui_options = v.Gui_options
				a.Gui_ReadOnly = v.Gui_ReadOnly
			}
			a.errExe = nil
			if v.Error != "" {
				a.errExe = errors.New(v.Error)
			}
		}

		if nc.progress.Error != "" {
			w.errExe = errors.New(nc.progress.Error)
		}
	} else {
		w.errExe = fmt.Errorf("can't find node exe(%s)", w.Exe)
	}

	//if w.HasExeError() {
	//	fmt.Printf("Node(%s) has error(%v)\n", w.Name, w.errExe)
	//}

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

	for _, v := range w.Attrs {
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

func (w *SAWidget) buildSubsList(listPathes *string, listNodes *[]*SAWidget) {
	nm := w.getPath()
	if len(nm) > 2 {
		nm = nm[:len(nm)-1] //cut last ';'
	}

	*listPathes += nm + ";"
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

func (w *SAWidget) CheckUniqueName() {
	w.Name = strings.ReplaceAll(w.Name, ".", "") //remove all '.'
	for w.parent.NumNames(w.Name) >= 2 {
		w.Name += "1"
	}
}

func (w *SAWidget) AddWidget(coord OsV4, exe string) *SAWidget {
	nw := NewSAWidget(w, exe, exe, coord)
	w.Subs = append(w.Subs, nw)
	nw.CheckUniqueName()
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

func (w *SAWidget) findAttr(name string) *SAWidgetAttr {
	for _, it := range w.Attrs {
		if it.Name == name {
			return it
		}
	}
	return nil
}
func (w *SAWidget) _getAttr(defValue SAWidgetAttr) *SAWidgetAttr {

	v := w.findAttr(defValue.Name)
	if v == nil {
		v = &SAWidgetAttr{}
		*v = defValue
		v.widget = w
		w.Attrs = append(w.Attrs, v)
	}

	return v
}

func (w *SAWidget) findAttrFloat(name string) (*SAWidgetAttr, float64) {
	for _, it := range w.Attrs {
		if it.Name == name {
			vv, _ := strconv.ParseFloat(it.Value, 64)
			return it, vv
		}
	}
	return nil, 0
}

func (w *SAWidget) GetAttr(name string, value string, gui_type string, gui_options string, gui_readOnly bool) *SAWidgetAttr {
	return w._getAttr(SAWidgetAttr{Name: name, Value: value, Gui_type: gui_type, Gui_options: gui_options, Gui_ReadOnly: gui_readOnly})
}

func (w *SAWidget) GetAttrEdit(name string, defValue string) *SAWidgetAttr {
	return w._getAttr(SAWidgetAttr{Name: name, Value: defValue, Gui_type: "edit"})
}

func (w *SAWidget) GetAttrCombo(name string, defValue string, defOptions string) *SAWidgetAttr {
	return w._getAttr(SAWidgetAttr{Name: name, Value: defValue, Gui_type: "combo", Gui_options: defOptions})
}

func (w *SAWidget) GetAttrCheckbox(name string, defValue string) *SAWidgetAttr {
	return w._getAttr(SAWidgetAttr{Name: name, Value: defValue, Gui_type: "checkbox"})
}

func (w *SAWidget) GetAttrSwitch(name string, defValue string) *SAWidgetAttr {
	return w._getAttr(SAWidgetAttr{Name: name, Value: defValue, Gui_type: "switch"})
}

func (w *SAWidget) GetAttrDate(name string, defValue string) *SAWidgetAttr {
	return w._getAttr(SAWidgetAttr{Name: name, Value: defValue, Gui_type: "date"})
}

func (w *SAWidget) GetAttrColor(prefix_name string) OsCd {
	var cd OsCd
	cd.R = w.GetAttrEdit(prefix_name+"r", "0").GetByte()
	cd.G = w.GetAttrEdit(prefix_name+"g", "0").GetByte()
	cd.B = w.GetAttrEdit(prefix_name+"b", "0").GetByte()
	cd.A = w.GetAttrEdit(prefix_name+"a", "255").GetByte()
	return cd
}
func (w *SAWidget) SetAttrColor(prefix_name string, cd OsCd) {
	w.GetAttrEdit(prefix_name+"r", "0").SetInt(int(cd.R))
	w.GetAttrEdit(prefix_name+"g", "0").SetInt(int(cd.G))
	w.GetAttrEdit(prefix_name+"b", "0").SetInt(int(cd.B))
	w.GetAttrEdit(prefix_name+"a", "255").SetInt(int(cd.A))
}

func (w *SAWidget) GetGrid() OsV4 {
	var v OsV4
	v.Start.X = w.GetAttrEdit("grid_x", "0").GetInt()
	v.Start.Y = w.GetAttrEdit("grid_y", "0").GetInt()
	v.Size.X = w.GetAttrEdit("grid_w", "1").GetInt()
	v.Size.Y = w.GetAttrEdit("grid_h", "1").GetInt()
	return v
}
func (w *SAWidget) SetGridStart(v OsV2) {
	w.GetAttrEdit("grid_x", "0").SetInt(v.X)
	w.GetAttrEdit("grid_y", "0").SetInt(v.Y)
}
func (w *SAWidget) SetGridSize(v OsV2) {
	w.GetAttrEdit("grid_w", "1").SetInt(v.X)
	w.GetAttrEdit("grid_h", "1").SetInt(v.X)
}
func (w *SAWidget) SetGrid(coord OsV4) {
	w.SetGridStart(coord.Start)
	w.SetGridSize(coord.Size)
}

func (w *SAWidget) GetGridShow() bool {
	return w.GetAttrSwitch("grid_show", "1").GetBool()
}

func (w *SAWidget) Render(ui *Ui, app *SAApp) {

	if !w.GetGridShow() {
		return
	}

	grid := w.GetGrid()
	grid.Size.X = OsMax(grid.Size.X, 1)
	grid.Size.Y = OsMax(grid.Size.Y, 1)

	switch w.Exe {

	case "button":
		enable := w.GetAttrSwitch("enable", "1").GetBool()
		tp := w.GetAttrCombo("type", "0", "Classic;Light;Menu;Segments").GetInt()

		clicked := false
		switch tp {
		case 0:
			clicked = ui.Comp_button(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, w.GetAttrEdit("label", "").GetString(), "", enable) > 0
		case 1:
			clicked = ui.Comp_buttonLight(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, w.GetAttrEdit("label", "").GetString(), "", enable) > 0
		case 2:
			selected := w.GetAttrSwitch("selected", "0").GetBool()
			clicked = ui.Comp_buttonMenu(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, w.GetAttrEdit("label", "").GetString(), "", enable, selected) > 0
			if clicked {
				sel := w.findAttr("selected")
				_, val, editable := sel.GetDirectLink()
				if editable {
					*val = OsTrnString(selected, "0", "1")
				}
			}

		case 3:
			labels := w.GetAttrEdit("label", "").GetString()
			butts := strings.Split(labels, ";")

			selected := w.GetAttrCombo("selected", "0", labels).GetInt()
			ui.Div_start(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y)
			{
				for i := range butts {
					ui.Div_colMax(i*2+0, 100)
					if i+1 < len(butts) {
						ui.Div_col(i*2+1, 0.1)
					}
				}
				for i, it := range butts {
					clicked = ui.Comp_buttonText(i*2+0, 0, 1, 1, it, "", "", enable, selected == i) > 0
					if clicked {
						sel := w.findAttr("selected")
						_, val, editable := sel.GetDirectLink()
						if editable {
							*val = strconv.Itoa(i)
						}
					}
					if i+1 < len(butts) {
						ui.Div_SpacerCol(i*2+1, 0, 1, 1)
					}
				}
				//ui.Paint_rect(0, 0, 1, 1, 0, ui.buff.win.io.GetPalette().GetGrey(0.5), 0.03)
			}
			ui.Div_end()
		}
		w.GetAttrSwitch("clicked", "0").GetBool()
		cl := w.findAttr("clicked")
		cl.Gui_ReadOnly = true
		cl.Value = OsTrnString(clicked, "1", "0")

	case "text":
		ui.Comp_text(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, w.GetAttrEdit("label", "").GetString(), w.GetAttrCombo("align", "0", "Left;Center;Right").GetInt())

	case "switch":
		_, value, editable := w.GetAttrEdit("value", "").GetDirectLink()
		enable := w.GetAttrSwitch("enable", "1").GetBool() && editable
		ui.Comp_switch(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, value, false, w.GetAttrEdit("label", "").GetString(), "", enable)

	case "checkbox":
		_, value, editable := w.GetAttrEdit("value", "").GetDirectLink()
		enable := w.GetAttrSwitch("enable", "1").GetBool() && editable
		ui.Comp_checkbox(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, value, false, w.GetAttrEdit("label", "").GetString(), "", enable)

	case "combo":
		_, value, editable := w.GetAttrEdit("value", "").GetDirectLink()
		enable := w.GetAttrSwitch("enable", "1").GetBool() && editable
		ui.Comp_combo(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, value, w.GetAttrEdit("options", "a;b;c").GetString(), "", enable, w.GetAttrSwitch("search", "0").GetBool())

	case "edit":
		_, value, editable := w.GetAttrEdit("value", "").GetDirectLink()
		enable := w.GetAttrSwitch("enable", "1").GetBool() && editable
		ui.Comp_editbox(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, value, w.GetAttrEdit("precision", "2").GetInt(), "", w.GetAttrEdit("ghost", "").GetString(), false, w.GetAttrSwitch("tempToValue", "0").GetBool(), enable)

	case "divider":
		tp := w.GetAttrCombo("type", "0", "Column;Row").GetInt()
		switch tp {
		case 0:
			ui.Div_SpacerCol(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y)
		case 1:
			ui.Div_SpacerRow(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y)
		}

	case "color_palette":
		ui.Div_startName(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, w.Name)
		{
			cd := w.GetAttrColor("cd_")
			if ui.comp_colorPalette(&cd) {
				w.SetAttrColor("cd_", cd)
			}
		}
		ui.Div_end()

	case "color_picker":
		ui.Div_startName(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, w.Name)
		{
			enable := w.GetAttrSwitch("enable", "1").GetBool()
			cd := w.GetAttrColor("cd_")
			if ui.comp_colorPicker(&cd, w.Name, enable) {
				w.SetAttrColor("cd_", cd)
			}
		}
		ui.Div_end()

	case "calendar":
		ui.Div_startName(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, w.Name)
		{
			value := w.GetAttrDate("value", "0").GetInt64()
			page := w.GetAttrDate("page", "0").GetInt64()

			ui.Comp_Calendar(&value, &page, 100, 100)

			w.GetAttrDate("value", "0").SetInt(int(value))
			w.GetAttrDate("page", "0").SetInt(int(page))
		}
		ui.Div_end()

	case "date_picker":
		ui.Div_startName(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, w.Name)
		{
			enable := w.GetAttrSwitch("enable", "1").GetBool()
			value := w.GetAttrDate("value", "0").GetInt64()
			show_time := w.GetAttrSwitch("show_time", "0").GetBool()
			if ui.Comp_CalendarDataPicker(&value, show_time, w.Name, enable) {
				w.GetAttrDate("value", "0").SetInt(int(value))
			}
		}
		ui.Div_end()

	case "map":
		ui.Div_startName(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, w.Name)
		{
			file := w.GetAttrEdit("file", "maps/osm").GetString()
			url := w.GetAttrEdit("url", "https://tile.openstreetmap.org/{z}/{x}/{y}.png").GetString()
			copyright := w.GetAttrEdit("copyright", "(c)OpenStreetMap contributors").GetString()
			copyright_url := w.GetAttrEdit("copyright_url", "https://www.openstreetmap.org/copyright").GetString()

			file = "disk/" + file

			cam_lon := w.GetAttrEdit("lon", "14.4071117049").GetFloat()
			cam_lat := w.GetAttrEdit("lat", "50.0852013259").GetFloat()
			cam_zoom := w.GetAttrEdit("zoom", "5").GetFloat()

			err := ui.comp_map(app.mapp, &cam_lon, &cam_lat, &cam_zoom, file, url, copyright, copyright_url)
			if err != nil {
				w.errExe = err
			}

			//set back
			w.GetAttrEdit("lon", "0").SetFloat(cam_lon)
			w.GetAttrEdit("lat", "0").SetFloat(cam_lat)
			w.GetAttrEdit("zoom", "5").SetInt(int(cam_zoom))

			//app.mapp.comp_map(w, ui)
			ui.Div_end()
		}
		div := ui.Div_startName(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, ".subs.")
		{
			div.touch_enabled = false
			w.RenderLayout(ui, app)
		}
		ui.Div_end()

	case "map_locators":
		ui.Div_startName(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, w.Name)
		{
			lonAttr, cam_lon := w.parent.findAttrFloat("lon")
			latAttr, cam_lat := w.parent.findAttrFloat("lat")
			zoomAttr, cam_zoom := w.parent.findAttrFloat("zoom")
			if lonAttr == nil || latAttr == nil || zoomAttr == nil {
				w.errExe = fmt.Errorf("parent widget is not 'Map' type")
				return
			}
			items := w.GetAttrEdit("items", "[{\"lon\":14.4071117049, \"lat\":50.0852013259, \"label\":\"1\"}, {\"lon\":14, \"lat\":50, \"label\":\"2\"}]").GetString()

			err := ui.comp_mapLocators(app.mapp, cam_lon, cam_lat, cam_zoom, items)
			if err != nil {
				w.findAttr("items").errExe = err
			}
		}
		ui.Div_end()

	case "layout":
		ui.Div_startName(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, w.Name)
		w.RenderLayout(ui, app)
		ui.Div_end()

	default: //layout
		ui.Div_startName(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, w.Name)
		w.RenderLayout(ui, app)
		ui.Div_end()
	}

	if app.IDE {
		//draw Select rectangle
		if app.act == w.parent {
			if w.HasExeError() || w.HasExpError() {
				pl := ui.buff.win.io.GetPalette()
				cd := pl.E
				cd.A = 150

				//rect
				div := ui.Div_startName(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, ".err.")
				div.touch_enabled = false
				ui.Paint_rect(0, 0, 1, 1, 0, cd, 0)
				ui.Div_end()
			}

			if w.Selected {
				cd := SAApp_getYellow()

				//rect
				div := ui.Div_startName(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, ".sel.")
				div.touch_enabled = false
				ui.Paint_rect(0, 0, 1, 1, 0, cd, 0.06)
				ui.Div_end()

				//resizer
				s := ui.CellWidth(0.2)
				ui.buff.AddRect(InitOsV4Mid(div.crop.End(), OsV2{s, s}), cd, 0)
			}
		}
		//when editbox with expression is active - match colors between access(text) and widgets(coords) ...
	}
}

func (w *SAWidget) RenderLayout(ui *Ui, app *SAApp) {

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

func _SAWidget_renderParamsValue(x, y, w, h int, attr *SAWidgetAttr, ui *Ui, gui_type string) {

	if attr.ShowExp {
		switch gui_type {

		case "xywh":
			ui.Div_start(x, y, w, h)
			{
				prefix := attr.Name[:len(attr.Name)-1]
				attrX := attr.widget.GetAttrEdit(prefix+"x", "0")
				attrY := attr.widget.GetAttrEdit(prefix+"y", "0")
				attrW := attr.widget.GetAttrEdit(prefix+"w", "1")
				attrH := attr.widget.GetAttrEdit(prefix+"h", "1")

				ui.Div_colMax(0, 100)
				ui.Div_colMax(1, 100)
				ui.Div_colMax(2, 100)
				ui.Div_colMax(3, 100)

				ui.Comp_editbox(0, 0, 1, 1, &attrX.Value, 2, "", "", false, false, true)
				ui.Comp_editbox(1, 0, 1, 1, &attrY.Value, 2, "", "", false, false, true)
				ui.Comp_editbox(2, 0, 1, 1, &attrW.Value, 2, "", "", false, false, true)
				ui.Comp_editbox(3, 0, 1, 1, &attrH.Value, 2, "", "", false, false, true)
			}
			ui.Div_end()

		case "rgba":
			ui.Div_start(x, y, w, h)
			{
				prefix := attr.Name[:len(attr.Name)-1]
				attrR := attr.widget.GetAttrEdit(prefix+"r", "0")
				attrG := attr.widget.GetAttrEdit(prefix+"g", "0")
				attrB := attr.widget.GetAttrEdit(prefix+"b", "0")
				attrA := attr.widget.GetAttrEdit(prefix+"a", "255")

				ui.Div_colMax(0, 100)
				ui.Div_colMax(1, 100)
				ui.Div_colMax(2, 100)
				ui.Div_colMax(3, 100)

				ui.Comp_editbox(0, 0, 1, 1, &attrR.Value, 2, "", "", false, false, true)
				ui.Comp_editbox(1, 0, 1, 1, &attrG.Value, 2, "", "", false, false, true)
				ui.Comp_editbox(2, 0, 1, 1, &attrB.Value, 2, "", "", false, false, true)
				ui.Comp_editbox(3, 0, 1, 1, &attrA.Value, 2, "", "", false, false, true)
			}
			ui.Div_end()

		default:
			ui.Comp_editbox(x, y, w, h, &attr.Value, 2, "", "", false, false, true)
		}

	} else {
		_, value, editable := attr.GetDirectLink()

		switch gui_type {
		case "checkbox":
			ui.Comp_checkbox(x, y, w, h, value, false, "", "", editable)

		case "switch":
			ui.Comp_switch(x, y, w, h, value, false, "", "", editable)

		case "date":
			ui.Div_start(x, y, w, h)
			value := attr.GetInt64()
			if ui.Comp_CalendarDataPicker(&value, true, attr.Name, editable) {
				attr.SetInt(int(value))
			}
			ui.Div_end()

		case "combo":
			ui.Comp_combo(x, y, w, h, value, attr.Gui_options, "", editable, false)

		case "edit":
			ui.Comp_editbox(x, y, w, h, value, 2, "", "", false, false, editable)

		case "xywh":
			ui.Div_start(x, y, w, h)
			{
				prefix := attr.Name[:len(attr.Name)-1]
				_, valueX, editableX := attr.widget.GetAttrEdit(prefix+"x", "0").GetDirectLink()
				_, valueY, editableY := attr.widget.GetAttrEdit(prefix+"y", "0").GetDirectLink()
				_, valueW, editableW := attr.widget.GetAttrEdit(prefix+"w", "1").GetDirectLink()
				_, valueH, editableH := attr.widget.GetAttrEdit(prefix+"h", "1").GetDirectLink()

				ui.Div_colMax(0, 100)
				ui.Div_colMax(1, 100)
				ui.Div_colMax(2, 100)
				ui.Div_colMax(3, 100)

				ui.Comp_editbox(0, 0, 1, 1, valueX, 2, "", "", false, false, editableX)
				ui.Comp_editbox(1, 0, 1, 1, valueY, 2, "", "", false, false, editableY)
				ui.Comp_editbox(2, 0, 1, 1, valueW, 2, "", "", false, false, editableW)
				ui.Comp_editbox(3, 0, 1, 1, valueH, 2, "", "", false, false, editableH)
			}
			ui.Div_end()

		case "rgba":
			ui.Div_start(x, y, w, h)

			prefix := attr.Name[:len(attr.Name)-1]
			cd := attr.widget.GetAttrColor(prefix)
			if ui.comp_colorPicker(&cd, attr.Name, editable) {
				attr.widget.SetAttrColor("cd_", cd)
			}

			ui.Div_end()

		default: //edit
			ui.Comp_editbox(x, y, w, h, value, 2, "", "", false, false, editable)
		}
	}
}

func (widget *SAWidget) IsAttrGroup(find *SAWidgetAttr) (string, bool, string) {

	if len(find.Name) <= 2 {
		return "", false, ""
	}
	prefix := find.Name[:len(find.Name)-1]

	x := widget.findAttr(prefix + "x")
	y := widget.findAttr(prefix + "y")
	z := widget.findAttr(prefix + "z")
	w := widget.findAttr(prefix + "w")
	h := widget.findAttr(prefix + "h")
	if x != nil && y != nil && w != nil && h != nil {
		return "xywh", find == x, prefix
	}

	if x != nil && y != nil && z != nil {
		return "xyz", find == x, prefix
	}
	if x != nil && y != nil {
		return "xy", find == x, prefix
	}

	r := widget.findAttr(prefix + "r")
	g := widget.findAttr(prefix + "g")
	b := widget.findAttr(prefix + "b")
	a := widget.findAttr(prefix + "a")
	if r != nil && g != nil && b != nil && a != nil {
		return "rgba", find == r, prefix
	}

	if len(find.Name) <= 4 {
		return "", false, ""
	}
	prefix = find.Name[:len(find.Name)-3]

	lon := widget.findAttr(prefix + "lon")
	lat := widget.findAttr(prefix + "lat")
	if lon != nil && lat != nil {
		return "lonlat", find == lon, prefix
	}

	return "", false, "" //not found
}

func (w *SAWidget) RenderParams(app *SAApp, ui *Ui) {

	ui.Div_colMax(0, 100)
	if w.IsGuiLayout() {
		ui.Div_row(2, 0.5) //spacer
	} else {
		ui.Div_row(1, 0.5) //spacer
	}

	y := 0

	if w.IsGuiLayout() {
		if ui.Comp_buttonLight(0, y, 1, 1, ui.trns.OPEN, "", true) > 0 {
			app.act = w
		}
		y++
	}

	ui.Div_start(0, y, 1, 1)
	{
		ui.Div_colMax(0, 100)
		ui.Div_colMax(1, 2)
		ui.Div_colMax(2, 2)

		//Name
		_, _, _, fnshd, _ := ui.Comp_editbox_desc(ui.trns.NAME, 0, 2, 0, 0, 1, 1, &w.Name, 0, "", ui.trns.NAME, false, false, true)
		if fnshd && w.parent != nil {
			w.CheckUniqueName()
		}

		//type
		w.Exe = app.ComboListOfWidgets(1, 0, 1, 1, w.Exe, ui)

		//delete
		if ui.Comp_button(2, 0, 1, 1, ui.trns.REMOVE, "", true) > 0 {
			w.Remove()
		}
	}
	ui.Div_end()
	y++

	ui.Div_SpacerRow(0, y, 1, 1)
	y++

	if w.errExe != nil {
		ui.Div_start(0, y, 1, 1)
		{
			pl := ui.buff.win.io.GetPalette()
			ui.Paint_rect(0, 0, 1, 1, 0, pl.E, 0) //red rect
		}
		ui.Div_end()
		ui.Comp_text(0, y, 1, 1, "Error: "+w.errExe.Error(), 0)
		y++
	}

	for i, it := range w.Attrs {
		group, isGroupFirst, prefixName := w.IsAttrGroup(it)
		if group != "" && !isGroupFirst {
			continue //skip 2nd, 3rd in group
		}

		ui.Div_start(0, y, 1, 1)
		{
			ui.Div_colMax(0, 3)
			ui.Div_colMax(1, 100)

			//highlight because it has expression
			if len(it.depends) > 0 {
				ui.Paint_rect(0, 0, 1, 1, 0.03, SAApp_getYellow().SetAlpha(50), 0)
			}

			//name: drag & drop
			ui.Div_start(0, 0, 1, 1)
			{
				ui.Div_drag("attr", i)
				src, pos, done := ui.Div_drop("attr", true, false, false)
				if done {
					Div_DropMoveElement(&w.Attrs, &w.Attrs, src, i, pos)
				}
			}
			ui.Div_end()

			//name
			nm := it.Name
			if prefixName != "" {
				nm = prefixName[:len(prefixName)-1] + "[" + group + "]"
			}
			if ui.Comp_buttonMenu(0, 0, 1, 1, nm, "", true, it.ShowExp) > 0 {
				it.ShowExp = !it.ShowExp
			}

			//value
			if it.errExp != nil {
				ui.Div_start(1, 0, 1, 1)
				{
					ui.Paint_tooltip(0, 0, 1, 1, it.errExp.Error())
					pl := ui.buff.win.io.GetPalette()
					ui.Paint_rect(0, 0, 1, 1, 0, pl.E, 0.03)
				}
				ui.Div_end()
			}

			gr := it.Gui_type
			if group != "" && isGroupFirst {
				gr = group
			}
			_SAWidget_renderParamsValue(1, 0, 1, 1, it, ui, gr)

			//error
			if it.errExe != nil {
				ui.Div_start(2, 0, 1, 1)
				{
					ui.Paint_tooltip(0, 0, 1, 1, it.errExe.Error())
					pl := ui.buff.win.io.GetPalette()
					ui.Paint_rect(0, 0, 1, 1, 0, pl.E, 0) //red rect
				}
				ui.Div_end()
			}

		}
		ui.Div_end()
		y++
	}
}

// expression language ... => show num rows after SELECT COUNT(*) FROM ...
//- práce s json? ...

// resize widget ...
// test history re-execute()? ...

// execute in 2nd thread and copy back when done ...
// change Node.Exe + Attr.Gui_type,.Gui_options ...
