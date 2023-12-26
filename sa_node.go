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

type SANodeAttr struct {
	node *SANode

	Name    string
	Value   string `json:",omitempty"`
	ShowExp bool

	oldValue     string
	instr        *VmInstr
	depends      []*SANodeAttr
	isDirectLink bool
	errExp       error
	errExe       error

	Gui_type     string `json:",omitempty"`
	Gui_options  string `json:",omitempty"`
	Gui_ReadOnly bool   `json:",omitempty"` //output
}

func (attr *SANodeAttr) GetDirectLink() (*SANodeAttr, *string, bool) {

	for attr.isDirectLink {
		return attr.depends[0].GetDirectLink() //go to source
	}

	if len(attr.depends) > 0 {
		return attr, &attr.oldValue, false //expression. oldValue = result
	}

	return attr, &attr.Value, true //this
}
func (attr *SANodeAttr) SetString(value string) {
	_, val, editable := attr.GetDirectLink()
	if editable {
		*val = value
	}
}
func (attr *SANodeAttr) SetInt(value int) {
	_, val, editable := attr.GetDirectLink()
	if editable {
		*val = strconv.Itoa(value)
	}
}
func (attr *SANodeAttr) SetFloat(value float64) {
	_, val, editable := attr.GetDirectLink()
	if editable {
		*val = strconv.FormatFloat(value, 'f', -1, 64)
	}
}

func (attr *SANodeAttr) GetString() string {
	_, val, _ := attr.GetDirectLink()
	return *val
}
func (attr *SANodeAttr) GetInt() int {
	v, _ := strconv.Atoi(attr.GetString())
	return v
}
func (attr *SANodeAttr) GetInt64() int64 {
	v, _ := strconv.Atoi(attr.GetString())
	return int64(v)
}
func (attr *SANodeAttr) GetFloat() float64 {
	v, _ := strconv.ParseFloat(attr.GetString(), 64)
	return v
}
func (attr *SANodeAttr) GetBool() bool {
	return attr.GetInt() != 0
}
func (attr *SANodeAttr) GetByte() byte {
	return byte(attr.GetInt())
}

type SANodeColRow struct {
	Min, Max, Resize float64 `json:",omitempty"`
	ResizeName       string  `json:",omitempty"`
}

func (a *SANodeColRow) Cmp(b *SANodeColRow) bool {
	return a.Min == b.Min && a.Max == b.Max && a.Resize == b.Resize && a.ResizeName == b.ResizeName
}

func InitSANodeColRow() SANodeColRow {
	return SANodeColRow{Min: 1, Max: 1, Resize: 1}
}

type SANode struct {
	parent *SANode

	Pos                 OsV2f
	pos_start           OsV2f
	Cam_x, Cam_y, Cam_z float64 `json:",omitempty"`

	Name           string
	Exe            string
	Selected       bool
	selected_cover bool

	Attrs []*SANodeAttr `json:",omitempty"`

	//sub-layout
	Cols []SANodeColRow `json:",omitempty"`
	Rows []SANodeColRow `json:",omitempty"`
	Subs []*SANode      `json:",omitempty"`

	//Bypass bool

	running bool
	done    bool
	errExe  error
}

func NewSANode(parent *SANode, name string, exe string, grid OsV4, pos OsV2f) *SANode {
	w := &SANode{}
	w.parent = parent
	w.Name = name
	w.Exe = exe
	w.Pos = pos

	if w.CanBeRenderOnCanvas() {
		w.SetGrid(grid)
	}
	return w
}

func NewSANodeRoot(path string) (*SANode, error) {
	w := NewSANode(nil, "root", "layout", OsV4{}, OsV2f{})
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

func (w *SANode) Save(path string) error {
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

func (a *SANode) Cmp(b *SANode) bool {
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

func (w *SANode) HasExpError() bool {

	for _, it := range w.Attrs {
		if it.errExp != nil {
			return true
		}
	}
	return false
}

func (w *SANode) HasExeError() bool {

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

func (w *SANode) HasError() bool {
	return w.HasExeError() || w.HasExpError()
}

func (w *SANode) UpdateExpresions(app *SAApp) {

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

func (w *SANode) checkForLoopInner(find *SANode) {

	for _, v := range w.Attrs {
		for _, dep := range v.depends {
			if dep.node == w {
				continue //skip self-depends
			}

			if dep.node == find {
				v.errExp = fmt.Errorf("Loop")
				continue //avoid infinite recursion
			}

			dep.node.checkForLoopInner(find)
		}
	}
}

func (w *SANode) CheckForLoops() {

	w.checkForLoopInner(w)

	for _, it := range w.Subs {
		it.CheckForLoops()
	}
}

func (w *SANode) areAttrsErrorFree() bool {

	for _, v := range w.Attrs {
		for _, dep := range v.depends {
			if dep.node.HasExpError() {
				v.errExp = fmt.Errorf("incomming error")
				return false
			}
		}
	}

	return true
}

func (w *SANode) areAttrsReadyToRun() bool {
	for _, v := range w.Attrs {
		for _, dep := range v.depends {
			if dep.node == w {
				continue //skip self-depends
			}

			if dep.node.running || !dep.node.done {
				return false //still running
			}
		}
	}
	return true
}

func (w *SANode) areAttrsChangedAndUpdate() bool {

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

func (w *SANode) IsReadyToBeExe() bool {

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

func (w *SANode) ResetExecute() {
	w.done = false
	w.running = false

	for _, it := range w.Subs {
		it.ResetExecute()
	}
}

func (w *SANode) buildList(list *[]*SANode) {
	*list = append(*list, w)
	for _, it := range w.Subs {
		it.buildList(list)
	}
}

func (w *SANode) IsGuiLayout() bool {
	return w.Exe == "layout" || !w.IsGui() //every exe is layout
}
func (w *SANode) IsGui() bool {
	if w.Exe == "" {
		return true
	}
	return SAApp_IsStdPrimitive(w.Exe)
}
func (w *SANode) IsExe() bool {
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

func (w *SANode) ExecuteSubs(server *SANodeServer, max_threads int) {
	var list []*SANode
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
							time.Sleep(20 * time.Millisecond)
						}

						//run it
						it.running = true
						wg.Add(1)
						go func(ww *SANode) {
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

func (w *SANode) execute(server *SANodeServer) {

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

func (w *SANode) FindNode(name string) *SANode {
	for _, it := range w.Subs {
		if it.Name == name {
			return it
		}
	}
	return nil
}

func (w *SANode) UpdateParents(parent *SANode) {
	w.parent = parent
	for _, v := range w.Attrs {
		v.node = w
	}

	for _, it := range w.Subs {
		it.UpdateParents(w)
	}
}

func (w *SANode) Copy() (*SANode, error) {

	js, err := json.Marshal(w)
	if err != nil {
		return nil, err
	}

	dst := NewSANode(nil, "", "", OsV4{}, OsV2f{})
	err = json.Unmarshal(js, dst)
	if err != nil {
		return nil, err
	}

	dst.UpdateParents(nil)

	return dst, nil
}

func (a *SANode) FindMirror(b *SANode, b_act *SANode) *SANode {

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

func (w *SANode) DeselectAll() {
	for _, n := range w.Subs {
		n.Selected = false
	}
}

func (w *SANode) SelectOnlyThis() {
	w.parent.DeselectAll()
	w.Selected = true
}

func (w *SANode) RemoveSelectedNodes() {
	for i := len(w.Subs) - 1; i >= 0; i-- {
		if w.Subs[i].Selected {
			w.Subs = append(w.Subs[:i], w.Subs[i+1:]...)
		}
	}
}

func (w *SANode) FindSelected() *SANode {
	for _, it := range w.Subs {
		if it.Selected {
			return it
		}
	}
	return nil
}

func (w *SANode) buildSubsList(listPathes *string, listNodes *[]*SANode) {
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

func (w *SANode) getPath() string {

	var path string

	if w.parent != nil {
		path += w.parent.getPath()
	}

	path += w.Name + "/"

	return path
}

func (w *SANode) CheckUniqueName() {
	w.Name = strings.ReplaceAll(w.Name, ".", "") //remove all '.'
	for w.parent.NumNames(w.Name) >= 2 {
		w.Name += "1"
	}
}

func (w *SANode) AddNode(grid OsV4, pos OsV2f, exe string) *SANode {
	nw := NewSANode(w, exe, exe, grid, pos)
	w.Subs = append(w.Subs, nw)
	nw.CheckUniqueName()
	return nw
}

func (w *SANode) Remove() bool {

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

func (w *SANode) findAttr(name string) *SANodeAttr {
	for _, it := range w.Attrs {
		if it.Name == name {
			return it
		}
	}
	return nil
}
func (w *SANode) _getAttr(defValue SANodeAttr) *SANodeAttr {

	v := w.findAttr(defValue.Name)
	if v == nil {
		v = &SANodeAttr{}
		*v = defValue
		v.node = w
		w.Attrs = append(w.Attrs, v)
	}

	return v
}

func (w *SANode) findAttrFloat(name string) (*SANodeAttr, float64) {
	for _, it := range w.Attrs {
		if it.Name == name {
			vv, _ := strconv.ParseFloat(it.Value, 64)
			return it, vv
		}
	}
	return nil, 0
}

func (w *SANode) GetAttr(name string, value string, gui_type string, gui_options string, gui_readOnly bool) *SANodeAttr {
	return w._getAttr(SANodeAttr{Name: name, Value: value, Gui_type: gui_type, Gui_options: gui_options, Gui_ReadOnly: gui_readOnly})
}

func (w *SANode) GetAttrEdit(name string, defValue string) *SANodeAttr {
	return w._getAttr(SANodeAttr{Name: name, Value: defValue, Gui_type: "edit"})
}

func (w *SANode) GetAttrCombo(name string, defValue string, defOptions string) *SANodeAttr {
	return w._getAttr(SANodeAttr{Name: name, Value: defValue, Gui_type: "combo", Gui_options: defOptions})
}

func (w *SANode) GetAttrCheckbox(name string, defValue string) *SANodeAttr {
	return w._getAttr(SANodeAttr{Name: name, Value: defValue, Gui_type: "checkbox"})
}

func (w *SANode) GetAttrSwitch(name string, defValue string) *SANodeAttr {
	return w._getAttr(SANodeAttr{Name: name, Value: defValue, Gui_type: "switch"})
}

func (w *SANode) GetAttrDate(name string, defValue string) *SANodeAttr {
	return w._getAttr(SANodeAttr{Name: name, Value: defValue, Gui_type: "date"})
}

func (w *SANode) GetAttrColor(prefix_name string) OsCd {
	var cd OsCd
	cd.R = w.GetAttrEdit(prefix_name+"r", "0").GetByte()
	cd.G = w.GetAttrEdit(prefix_name+"g", "0").GetByte()
	cd.B = w.GetAttrEdit(prefix_name+"b", "0").GetByte()
	cd.A = w.GetAttrEdit(prefix_name+"a", "255").GetByte()
	return cd
}
func (w *SANode) SetAttrColor(prefix_name string, cd OsCd) {
	w.GetAttrEdit(prefix_name+"r", "0").SetInt(int(cd.R))
	w.GetAttrEdit(prefix_name+"g", "0").SetInt(int(cd.G))
	w.GetAttrEdit(prefix_name+"b", "0").SetInt(int(cd.B))
	w.GetAttrEdit(prefix_name+"a", "255").SetInt(int(cd.A))
}

func (w *SANode) GetGrid() OsV4 {
	var v OsV4
	v.Start.X = w.GetAttrEdit("grid_x", "0").GetInt()
	v.Start.Y = w.GetAttrEdit("grid_y", "0").GetInt()
	v.Size.X = w.GetAttrEdit("grid_w", "1").GetInt()
	v.Size.Y = w.GetAttrEdit("grid_h", "1").GetInt()
	return v
}
func (w *SANode) SetGridStart(v OsV2) {
	w.GetAttrEdit("grid_x", "0").SetInt(v.X)
	w.GetAttrEdit("grid_y", "0").SetInt(v.Y)
}
func (w *SANode) SetGridSize(v OsV2) {
	w.GetAttrEdit("grid_w", "1").SetInt(v.X)
	w.GetAttrEdit("grid_h", "1").SetInt(v.Y)
}
func (w *SANode) SetGrid(coord OsV4) {
	w.SetGridStart(coord.Start)
	w.SetGridSize(coord.Size)
}

func (w *SANode) GetGridShow() bool {
	return w.GetAttrSwitch("grid_show", "1").GetBool()
}

func (w *SANode) CanBeRenderOnCanvas() bool {
	return (SAApp_IsStdPrimitive(w.Exe) || SAApp_IsStdComponent(w.Exe))
}

func (w *SANode) Render(ui *Ui, app *SAApp) {

	if !w.CanBeRenderOnCanvas() {
		return
	}

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
				w.errExe = fmt.Errorf("parent node is not 'Map' type")
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
				s := ui.CellWidth(0.3)
				rs := InitOsV4Mid(div.crop.End(), OsV2{s, s})
				ui.buff.AddRect(rs, cd, 0)

				//musím stisknout alt-key + ignorovat select + highlight when over ... ................
				touch := &ui.buff.win.io.touch
				if touch.start && rs.Inside(touch.pos) {
					app.canvas.resize = w
				}

				if app.canvas.resize != nil {
					pos := ui.GetCall().call.GetCloseCell(touch.pos)
					fmt.Println(pos.Start)

					grid := app.canvas.resize.GetGrid()
					grid.Size.X = OsMax(0, pos.Start.X-grid.Start.X) + 1
					grid.Size.Y = OsMax(0, pos.Start.Y-grid.Start.Y) + 1

					app.canvas.resize.SetGrid(grid)
				}
				if touch.end {
					app.canvas.resize = nil
				}
			}
		}
		//when editbox with expression is active - match colors between access(text) and nodes(coords) ...
	}
}

func (w *SANode) RenderLayout(ui *Ui, app *SAApp) {

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

func (w *SANode) NumNames(name string) int {

	n := 0
	for _, it := range w.Subs {
		if it.Name == name {
			n++
		}
	}
	return n

}

func _SANode_renderParamsValue(x, y, w, h int, attr *SANodeAttr, ui *Ui, gui_type string) {

	if attr.ShowExp {
		switch gui_type {

		case "xywh":
			ui.Div_start(x, y, w, h)
			{
				prefix := attr.Name[:len(attr.Name)-1]
				attrX := attr.node.GetAttrEdit(prefix+"x", "0")
				attrY := attr.node.GetAttrEdit(prefix+"y", "0")
				attrW := attr.node.GetAttrEdit(prefix+"w", "1")
				attrH := attr.node.GetAttrEdit(prefix+"h", "1")

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
				attrR := attr.node.GetAttrEdit(prefix+"r", "0")
				attrG := attr.node.GetAttrEdit(prefix+"g", "0")
				attrB := attr.node.GetAttrEdit(prefix+"b", "0")
				attrA := attr.node.GetAttrEdit(prefix+"a", "255")

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
				_, valueX, editableX := attr.node.GetAttrEdit(prefix+"x", "0").GetDirectLink()
				_, valueY, editableY := attr.node.GetAttrEdit(prefix+"y", "0").GetDirectLink()
				_, valueW, editableW := attr.node.GetAttrEdit(prefix+"w", "1").GetDirectLink()
				_, valueH, editableH := attr.node.GetAttrEdit(prefix+"h", "1").GetDirectLink()

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
			cd := attr.node.GetAttrColor(prefix)
			if ui.comp_colorPicker(&cd, attr.Name, editable) {
				attr.node.SetAttrColor("cd_", cd)
			}

			ui.Div_end()

		default: //edit
			ui.Comp_editbox(x, y, w, h, value, 2, "", "", false, false, editable)
		}
	}
}

func (node *SANode) IsAttrGroup(find *SANodeAttr) (string, bool, string) {

	if len(find.Name) <= 2 {
		return "", false, ""
	}
	prefix := find.Name[:len(find.Name)-1]

	x := node.findAttr(prefix + "x")
	y := node.findAttr(prefix + "y")
	z := node.findAttr(prefix + "z")
	w := node.findAttr(prefix + "w")
	h := node.findAttr(prefix + "h")
	if x != nil && y != nil && w != nil && h != nil {
		return "xywh", find == x, prefix
	}

	if x != nil && y != nil && z != nil {
		return "xyz", find == x, prefix
	}
	if x != nil && y != nil {
		return "xy", find == x, prefix
	}

	r := node.findAttr(prefix + "r")
	g := node.findAttr(prefix + "g")
	b := node.findAttr(prefix + "b")
	a := node.findAttr(prefix + "a")
	if r != nil && g != nil && b != nil && a != nil {
		return "rgba", find == r, prefix
	}

	if len(find.Name) <= 4 {
		return "", false, ""
	}
	prefix = find.Name[:len(find.Name)-3]

	lon := node.findAttr(prefix + "lon")
	lat := node.findAttr(prefix + "lat")
	if lon != nil && lat != nil {
		return "lonlat", find == lon, prefix
	}

	return "", false, "" //not found
}

func (w *SANode) RenderParams(app *SAApp) {

	ui := app.base.ui

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
		ui.Div_colMax(1, 3)
		ui.Div_colMax(2, 2)

		//Name
		_, _, _, fnshd, _ := ui.Comp_editbox_desc(ui.trns.NAME, 0, 2, 0, 0, 1, 1, &w.Name, 0, "", ui.trns.NAME, false, false, true)
		if fnshd && w.parent != nil {
			w.CheckUniqueName()
		}

		//type
		w.Exe = app.ComboListOfNodes(1, 0, 1, 1, w.Exe, ui)

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
			ui.Div_colMax(1, 3)
			ui.Div_colMax(2, 100)

			//highlight because it has expression
			if len(it.depends) > 0 {
				ui.Paint_rect(0, 0, 1, 1, 0.03, SAApp_getYellow().SetAlpha(50), 0)
			}

			x := 0

			if ui.Comp_buttonIcon(x, 0, 1, 1, "file:apps/base/resources/copy.png", 0.3, ui.trns.COPY, CdPalette_B, true) > 0 {
				keys := &ui.buff.win.io.keys
				keys.clipboard = w.Name + "." + it.Name
			}
			x++

			//name: drag & drop
			ui.Div_start(x, 0, 1, 1)
			{
				ui.Div_drag("attr", i)
				src, pos, done := ui.Div_drop("attr", true, false, false)
				if done {
					Div_DropMoveElement(&w.Attrs, &w.Attrs, src, i, pos)
				}
			}
			ui.Div_end()

			//name
			{
				nm := it.Name
				if prefixName != "" {
					nm = prefixName[:len(prefixName)-1] + "[" + group + "]"
				}
				if ui.Comp_buttonMenu(x, 0, 1, 1, nm, "", true, it.ShowExp) > 0 {
					it.ShowExp = !it.ShowExp
				}
				x++
			}

			//value - error/title
			if it.errExp != nil {
				ui.Div_start(x, 0, 1, 1)
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
			_SANode_renderParamsValue(x, 0, 1, 1, it, ui, gr)
			x++

			if len(it.depends) == 1 {
				if ui.Comp_buttonLight(x, 0, 1, 1, ui.trns.GOTO, "", true) > 0 {
					it.depends[0].node.SelectOnlyThis()
				}
				x++
			} else if len(it.depends) > 1 {
				nm := "goto_" + it.Name
				if ui.Comp_buttonLight(x, 0, 1, 1, ui.trns.GOTO, "", true) > 0 {
					ui.Dialog_open(nm, 1)
				}
				if ui.Dialog_start(nm) {
					ui.Div_colMax(0, 5)
					for i, dp := range it.depends {
						if ui.Comp_buttonMenu(0, i, 1, 1, dp.node.Name+"."+dp.Name, "", true, false) > 0 {
							dp.node.SelectOnlyThis()
						}
					}
					ui.Dialog_end()
				}

				x++
			}

			//error
			if it.errExe != nil {
				ui.Div_start(x, 0, 1, 1)
				{
					ui.Paint_tooltip(0, 0, 1, 1, it.errExe.Error())
					pl := ui.buff.win.io.GetPalette()
					ui.Paint_rect(0, 0, 1, 1, 0, pl.E, 0) //red rect
				}
				ui.Div_end()
				x++
			}
		}
		ui.Div_end()
		y++
	}
}

// node copy/paste ...
// node Context menu? ...

// expression language ... => show num rows after SELECT COUNT(*) FROM ...
//- práce s json? ...

// execute in 2nd thread and copy back when done ... + some cache, less computing ...
// change Node.Exe + Attr.Gui_type,.Gui_options ...
