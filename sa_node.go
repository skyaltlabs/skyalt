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
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
)

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

const (
	SANode_STATE_WAITING = 0
	SANode_STATE_RUNNING = 1
	SANode_STATE_DONE    = 2
)

type SANode struct {
	app    *SAApp
	parent *SANode

	Pos                 OsV2f
	pos_start           OsV2f
	Cam_x, Cam_y, Cam_z float64 `json:",omitempty"`

	Name     string
	Exe      string
	Bypass   bool `json:",omitempty"`
	Selected bool `json:",omitempty"`

	selected_cover  bool
	selected_canvas OsV4

	Attrs []*SANodeAttr `json:",omitempty"`

	//sub-layout
	Cols []SANodeColRow `json:",omitempty"`
	Rows []SANodeColRow `json:",omitempty"`
	Subs []*SANode      `json:",omitempty"`

	state atomic.Uint32 //0=waiting, 1=running, 2=done

	errExe        error
	progress      float64
	progress_desc string
	exeTimeSec    float64

	z_depth float64

	sort_depth int //hierarchy
}

func NewSANode(app *SAApp, parent *SANode, name string, exe string, grid OsV4, pos OsV2f) *SANode {
	w := &SANode{}
	w.parent = parent
	w.app = app
	w.Name = name
	w.Exe = exe
	w.Pos = pos

	if w.CanBeRenderOnCanvas() {
		w.SetGrid(grid)
	}
	return w
}

func NewSANodeRoot(path string, app *SAApp) (*SANode, error) {
	w := NewSANode(app, nil, "root", "layout", OsV4{}, OsV2f{})
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
		w.updateLinks(nil, app)
	}

	return w, nil
}

func (w *SANode) UpdateDepth(orig *SANode) {
	for _, attr := range w.Attrs {
		for _, dp := range attr.depends {

			n := dp.node
			if n == w {
				continue //inner link
			}

			if n == orig {
				//loop err ...
				return
			}

			dp.node.UpdateDepth(orig)
			w.sort_depth = OsMax(w.sort_depth, n.sort_depth+1)
		}
	}
}

func (w *SANode) SetError(err string) {
	w.errExe = errors.New(err)
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

func (a *SANode) Cmp(b *SANode, historyDiff *bool) bool {
	if a.Name != b.Name || a.Exe != b.Exe || a.Bypass != b.Bypass {
		return false
	}

	if a.Selected != b.Selected {
		*historyDiff = true //no return!
	}

	if !a.Pos.Cmp(b.Pos) {
		*historyDiff = true //no return!
	}

	if a.Cam_x != b.Cam_x || a.Cam_y != b.Cam_y || a.Cam_z != b.Cam_z {
		*historyDiff = true //no return!
	}

	if len(a.Attrs) != len(b.Attrs) {
		return false
	}
	for i, itA := range a.Attrs {
		itB := b.Attrs[i]
		if !itA.Cmp(itB) {
			return false
		}
	}

	if len(a.Cols) == len(b.Cols) {
		for i, itA := range a.Cols {
			if !itA.Cmp(&b.Cols[i]) {
				*historyDiff = true //no return!
			}
		}
	} else {
		*historyDiff = true //no return!
	}

	if len(a.Rows) == len(b.Rows) {
		for i, itA := range a.Rows {
			if !itA.Cmp(&b.Rows[i]) {
				*historyDiff = true //no return!
			}
		}
	} else {
		*historyDiff = true //no return!
	}

	if len(a.Subs) != len(b.Subs) {
		return false
	}
	for i, itA := range a.Subs {
		if !itA.Cmp(b.Subs[i], historyDiff) {
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

func (w *SANode) ResetExeErrors() {
	w.errExe = nil
	for _, v := range w.Attrs {
		v.errExe = nil
	}
}

func (w *SANode) PrepareExe() {
	w.state.Store(SANode_STATE_WAITING)

	for _, v := range w.Attrs {
		v.errExe = nil
		v.errExp = nil
	}

	for _, it := range w.Subs {
		it.PrepareExe()
	}
}

func (w *SANode) ParseExpresions() {

	for _, it := range w.Attrs {
		it.ParseExpresion()
	}

	for _, it := range w.Subs {
		it.ParseExpresions()
	}
}

func (w *SANode) checkForLoopNode(find *SANode) {
	for _, v := range w.Attrs {
		for _, dep := range v.depends {
			if dep.node == w {
				dep.CheckForLoopAttr(v)
				continue //skip self-depends
			}

			if dep.node == find {
				v.errExp = fmt.Errorf("Loop")
				continue //avoid infinite recursion
			}

			dep.node.checkForLoopNode(find)
		}
	}
}

func (w *SANode) CheckForLoops() {

	w.checkForLoopNode(w)

	for _, it := range w.Subs {
		it.CheckForLoops()
	}
}

func (w *SANode) IsReadyToBeExe() bool {

	//areAttrsErrorFree
	for _, v := range w.Attrs {
		if v.errExp != nil {
			w.state.Store(SANode_STATE_DONE)
			return false
		}
	}

	//areDependsErrorFree
	for _, v := range w.Attrs {
		for _, dep := range v.depends {
			if dep.node.HasExpError() {
				w.state.Store(SANode_STATE_DONE)
				return false //has error
			}
		}
	}

	//areDependsReadyToRun
	for _, v := range w.Attrs {
		for _, dep := range v.depends {
			if dep.node == w {
				continue //skip self-depends
			}

			if dep.node.state.Load() != SANode_STATE_DONE {
				return false //still running
			}
		}
	}

	//execute expression
	for _, v := range w.Attrs {
		if v.errExp != nil {
			continue
		}
		v.ExecuteExpression()
	}

	return true
}

func (w *SANode) buildList(list *[]*SANode) {
	*list = append(*list, w)
	for _, it := range w.Subs {
		it.buildList(list)
	}
}

func (w *SANode) IsGuiLayout() bool {
	return strings.EqualFold(w.Exe, "layout") || !w.IsGuiPrimitive() //every exe is layout
}
func (w *SANode) IsGuiPrimitive() bool {
	if w.Exe == "" {
		return true
	}
	return SAApp_IsStdPrimitive(w.Exe)
}
func (w *SANode) CanBeRenderOnCanvas() bool {
	return (SAApp_IsStdPrimitive(w.Exe) || SAApp_IsStdComponent(w.Exe))
}

func (w *SANode) markUnusedAttrs() {
	for _, a := range w.Attrs {
		a.exeMark = false
	}

	for _, n := range w.Subs {
		n.markUnusedAttrs()
	}
}

func (w *SANode) ExecuteGui(renderIt bool) {

	if !renderIt {
		w.ResetExeErrors()
	}

	w.z_depth = 1

	switch strings.ToLower(w.Exe) {

	case "dialog":
		w.SARender_Dialog(renderIt)

	case "button":
		w.SARender_Button(renderIt)
	case "text":
		w.SARender_Text(renderIt)
	case "switch":
		w.SARender_Switch(renderIt)
	case "checkbox":
		w.SARender_Checkbox(renderIt)
	case "combo":
		w.SARender_Combo(renderIt)
	case "editbox":
		w.SARender_Editbox(renderIt)
	case "divider":
		w.SARender_Divider(renderIt)
	case "color_palette":
		w.SARender_ColorPalette(renderIt)
	case "color":
		w.SARender_Color(renderIt)
	case "calendar":
		w.SARender_Calendar(renderIt)
	case "date":
		w.SARender_Date(renderIt)
	case "map":
		w.SARender_Map(renderIt)
	case "image":
		w.SARender_Image(renderIt)
	case "file":
		w.SARender_File(renderIt)

	case "layout":
		w.SARender_Layout(renderIt)
	}
}

func (w *SANode) Execute() bool {

	ok := true
	st := OsTime()

	switch strings.ToLower(w.Exe) {
	case "sqlite_select":
		ok = w.Sqlite_select()
	case "sqlite_insert":
		//ok = w.Sqlite_insert()	//.......
	case "sqlite_update":
		//...
	case "sqlite_delete":
		//...
	case "sqlite_execute":
		//...

	case "csv_select":
		ok = w.Csv_select()

	case "gpx_to_json":
		ok = w.SAConvert_GpxToJson()

	case "write_file":
		ok = w.SA_WriteFile()

	case "blob":
		ok = w.ConstBlob()
	case "medium":
		ok = w.ConstMedium()

	default:
		if SAApp_IsExternal(w.Exe) {
			ok = w.executeProgram()
		}
	}

	w.exeTimeSec = OsTime() - st
	fmt.Printf("'%s' done in %.2fs\n", w.Name, w.exeTimeSec)
	return ok
}

func (w *SANode) executeProgram() bool {

	fmt.Println("execute:", w.Name)

	w.errExe = nil
	w.progress = 0
	w.progress_desc = ""

	conn := w.app.base.server.Start(w.Exe)
	if conn == nil {
		w.errExe = fmt.Errorf("can't find node program(%s)", w.Exe)
		return false
	}

	conn.Lock()
	defer conn.Unlock()

	//add/update attributes
	for _, v := range conn.Attrs {
		v.Error = ""

		a := w.GetAttr(v.Name, v.Value)
		a.errExe = nil
	}

	//set/remove attributes
	for i := len(w.Attrs) - 1; i >= 0; i-- {
		src := w.Attrs[i]
		dst := conn.FindAttr(src.Name)
		if dst != nil {
			dst.Value = src.result.String()
		} else {
			w.Attrs = append(w.Attrs[:i], w.Attrs[i+1:]...) //remove
		}
	}

	//execute
	ok := conn.Run(w)

	//copy back
	for _, v := range conn.Attrs {
		a := w.GetAttr(v.Name, v.Value)
		a.Value = v.Value
		a.errExe = nil
		if v.Error != "" {
			a.errExe = errors.New(v.Error)
		}
	}

	fmt.Println(w.Name, "done")
	return ok
}

func (w *SANode) FindNode(name string) *SANode {
	for _, it := range w.Subs {
		if it.Name == name {
			return it
		}
	}
	return nil
}

func (w *SANode) updateLinks(parent *SANode, app *SAApp) {
	w.parent = parent
	w.app = app
	for _, v := range w.Attrs {
		v.node = w
	}

	for _, it := range w.Subs {
		it.updateLinks(w, app)
	}
}

func (w *SANode) Copy() (*SANode, error) {

	js, err := json.Marshal(w)
	if err != nil {
		return nil, err
	}

	dst := NewSANode(w.app, nil, "", "", OsV4{}, OsV2f{})
	err = json.Unmarshal(js, dst)
	if err != nil {
		return nil, err
	}

	dst.updateLinks(nil, w.app)

	return dst, nil
}

func (a *SANode) FindMirror(b *SANode, b_find *SANode) *SANode {

	if b == b_find {
		return a
	}
	for i, na := range a.Subs {
		ret := na.FindMirror(b.Subs[i], b_find)
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

func (w *SANode) BypassReverseSelectedNodes() {
	for _, n := range w.Subs {
		if n.Selected {
			n.Bypass = !n.Bypass
		}
	}
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

func (w *SANode) buildSubsList(listPathes *[]string, listNodes *[]*SANode) {
	*listPathes = append(*listPathes, w.getPath())
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

func (w *SANode) NumAttrNames(name string) int {
	n := 0
	for _, attr := range w.Attrs {
		if attr.Name == name {
			n++
		}
	}
	return n
}

func (w *SANode) NumSubNames(name string) int {
	n := 0
	for _, nd := range w.Subs {
		if nd.Name == name {
			n++
		}
	}
	return n
}

func (w *SANode) CheckUniqueName() {

	if w.Name == "" {
		w.Name = "node"
	}

	w.Name = strings.ReplaceAll(w.Name, ".", "") //remove all '.'

	for w.parent.NumSubNames(w.Name) >= 2 {
		w.Name += "1"
	}
}

func (w *SANode) AddNode(grid OsV4, pos OsV2f, exe string) *SANode {
	nw := NewSANode(w.app, w, exe, exe, grid, pos)
	w.Subs = append(w.Subs, nw)
	nw.CheckUniqueName()
	return nw
}

func (w *SANode) AddNodeCopy(src *SANode) *SANode { // note: 'w' can be root graph
	nw, _ := src.Copy() //err ...
	nw.updateLinks(w, w.app)
	nw.Pos = nw.Pos.Add(OsV2f{1, 1})

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
			it.exeMark = true
			return it
		}
	}
	return nil
}
func (w *SANode) _getAttr(find bool, defValue SANodeAttr) *SANodeAttr {

	var v *SANodeAttr
	if find {
		v = w.findAttr(defValue.Name)
	}

	if v == nil {
		v = &SANodeAttr{}
		*v = defValue
		v.node = w
		w.Attrs = append(w.Attrs, v)
		v.CheckUniqueName()
	}
	if v.Value == "" {
		v.Value = "\"\"" //edit
	}

	if v.instr == nil {
		v.ParseExpresion()
		v.ExecuteExpression() //right now, so default value is in v.finalValue
	}

	v.defaultValue = defValue.Value //update
	v.exeMark = true
	return v
}

func (w *SANode) AddAttr(name string) *SANodeAttr {
	return w._getAttr(false, SANodeAttr{Name: name})
}
func (w *SANode) GetAttr(name string, value string) *SANodeAttr {
	return w._getAttr(true, SANodeAttr{Name: name, Value: value})
}

func (w *SANode) GetGrid() OsV4 {
	return w.GetAttr("grid", "[0, 0, 1, 1]").result.GetV4()
}

func (w *SANode) SetGridStart(v OsV2) {
	attr := w.GetAttr("grid", "[0, 0, 1, 1]")
	if attr == nil {
		return
	}

	attr.ReplaceArrayItemValueInt(1, v.Y) //y
	attr.ReplaceArrayItemValueInt(0, v.X) //x
}
func (w *SANode) SetGridSize(v OsV2) {
	attr := w.GetAttr("grid", "[0, 0, 1, 1]")
	if attr == nil {
		return
	}

	attr.ReplaceArrayItemValueInt(3, v.Y) //h
	attr.ReplaceArrayItemValueInt(2, v.X) //w
}
func (w *SANode) SetGrid(coord OsV4) {
	w.SetGridStart(coord.Start)
	w.SetGridSize(coord.Size)
}

func (w *SANode) GetGridShow() bool {
	return w.GetAttr("grid_show", "uiSwitch(1)").GetBool()
}

func (w *SANode) Render() {

	ui := w.app.base.ui

	w.ExecuteGui(true)

	if w.app.IDE && w.CanBeRenderOnCanvas() {

		grid := w.GetGrid()
		grid.Size.X = OsMax(grid.Size.X, 1)
		grid.Size.Y = OsMax(grid.Size.Y, 1)

		//draw Select rectangle
		if w.app.act == w.parent {
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
				en := InitOsV4Mid(div.canvas.End(), OsV2{s, s})
				if en.Inside(ui.buff.win.io.touch.pos) && ui.buff.win.io.keys.alt {
					pl := ui.buff.win.io.GetPalette()
					cd = pl.P
				}
				ui.buff.AddRect(en, cd, 0)

				w.selected_canvas = div.canvas //copy coord size
			}
		}
		//when editbox with expression is active - match colors between access(text) and nodes(coords) ...
	}
}

func (w *SANode) GetResizerCoord(ui *Ui) OsV4 {
	s := ui.CellWidth(0.3)
	return InitOsV4Mid(w.selected_canvas.End(), OsV2{s, s})
}

func (w *SANode) renderLayout() {

	ui := w.app.base.ui

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

	//sort z-depth
	sort.Slice(w.Subs, func(i, j int) bool {
		return w.Subs[i].z_depth < w.Subs[j].z_depth
	})

	//other items
	for _, it := range w.Subs {
		it.Render()
	}
}

func _SANode_isEditable(attr *SANodeAttr, instr *VmInstr) bool {
	return !attr.IsOutput() && instr != nil && !instr.pos_attr.IsOutput()
}

func _SANode_renderAttrValue(x, y, w, h int, attr *SANodeAttr, ui *Ui) {

	if attr.ShowExp {
		ui.Comp_editbox(x, y, w, h, &attr.Value, 2, nil, "", false, false, true) //show whole expression
	} else {

		if attr.instr == nil {
			return
		}

		var fn VmInstr_callbackExecute
		if attr.instr != nil {
			fn = attr.instr.fn
		}

		instr := attr.instr.GetConst()
		var value string
		if instr != nil {
			value = instr.pos_attr.result.String()
		} else {
			value = attr.result.String()
		}

		editable := _SANode_isEditable(attr, instr)

		if VmCallback_Cmp(fn, VmApi_UiSwitch) {
			if ui.Comp_switch(x, y, w, h, &value, false, "", "", editable) {
				instr.LineReplace(value)
			}
		} else if VmCallback_Cmp(fn, VmApi_UiCheckbox) {
			if ui.Comp_checkbox(x, y, w, h, &value, false, "", "", editable) {
				instr.LineReplace(value)
			}
		} else if VmCallback_Cmp(fn, VmApi_UiDate) {
			ui.Div_start(x, y, w, h)
			val := int64(instr.pos_attr.result.Number())
			if ui.Comp_CalendarDataPicker(&val, true, attr.Name, editable) {
				instr.LineReplace(strconv.Itoa(int(val)))
			}
			ui.Div_end()
		} else if VmCallback_Cmp(fn, VmApi_UiCombo) {
			if len(attr.instr.prms) >= 3 {
				options_names := attr.instr.prms[1].value.temp.String()  //instr is first parameter, GuiCombo() api is parent!
				options_values := attr.instr.prms[2].value.temp.String() //instr is first parameter, GuiCombo() api is parent!
				if ui.Comp_combo(x, y, w, h, &value, strings.Split(options_names, ";"), strings.Split(options_values, ";"), "", editable, false) {
					instr.LineReplace(value)
				}
			}
		} else if VmCallback_Cmp(fn, VmApi_UiColor) {
			cd := attr.result.GetCd()
			if ui.comp_colorPicker(x, y, w, h, &cd, attr.Name, true) {
				if editable {
					attr.ReplaceCd(cd)
				}
			}
		} else if VmCallback_Cmp(fn, VmApi_UiBlob) {
			blob := attr.result.Blob()
			dnm := "blob_" + attr.Name
			if ui.Comp_buttonIcon(x, y, w, h, InitWinMedia_blob(blob), 0, "", CdPalette_White, true, false) > 0 {
				ui.Dialog_open(dnm, 0)
			}
			if ui.Dialog_start(dnm) {
				ui.Div_colMax(0, 20)
				ui.Div_rowMax(0, 20)
				ui.Comp_image(0, 0, 1, 1, InitWinMedia_blob(blob), InitOsCd32(255, 255, 255, 255), 0, 1, 1, false)
				ui.Dialog_end()
			}
		} else if VmCallback_Cmp(fn, VmBasic_InitArray) {
			ui.Div_start(x, y, w, h)
			{
				//show first 4, then "others" -> open dialog? .....

				n := attr.result.NumArrayItems()

				ui.Div_colMax(0, 100)
				ui.Div_start(0, 0, 1, 1)
				{
					ui.DivInfo_set(SA_DIV_SET_scrollHnarrow, 1, 0)
					ui.Div_row(0, 0.5)
					ui.Div_rowMax(0, 2)

					for i := 0; i < n; i++ {
						ui.Div_colMax(i, 100)
					}

					for i := 0; i < n; i++ {
						it := attr.result.GetArrayItem(i)
						value := ""
						if it != nil {
							value = it.String()
						}
						//value = attr.instr.prms[i].instr.temp.String()	//if attribute is Output then there is no instr

						item_instr := attr.instr.GetConstArrayPrm(i)
						_, _, _, fnshd, _ := ui.Comp_editbox(i, 0, 1, 1, &value, 2, nil, "", false, false, _SANode_isEditable(attr, item_instr))
						if fnshd {
							attr.ReplaceArrayItemValue(i, value)
						}
					}

					if n == 0 {
						ui.Div_colMax(0, 100) //key
						ui.Comp_text(0, 0, 1, 1, "Empty Array []", 0)
					}
				}
				ui.Div_end()

				if !attr.IsOutput() {
					if ui.Comp_buttonLight(1, 0, 1, 1, "+", "Add item", true) > 0 {
						attr.AddParamsItem("0", false)
					}

					if ui.Comp_buttonLight(2, 0, 1, 1, "-", "Remove last item", n > 0) > 0 {
						attr.RemoveParamsItem(false)
					}
				}
			}
			ui.Div_end()
		} else if VmCallback_Cmp(fn, VmBasic_InitMap) {
			ui.Div_start(x, y, w, h)
			{
				//show first 4, then "others" -> open dialog? .....

				n := attr.result.NumMapItems()

				ui.Div_colMax(0, 100)
				ui.Div_start(0, 0, 1, 1)
				{
					ui.DivInfo_set(SA_DIV_SET_scrollHnarrow, 1, 0)
					ui.Div_row(0, 0.5)
					ui.Div_rowMax(0, 2)

					for i := 0; i < n; i++ {
						ui.Div_colMax(i*3+0, 100) //key
						ui.Div_colMax(i*3+1, 100) //value
						ui.Div_col(i*3+2, 0.2)    //space
					}

					for i := 0; i < n; i++ {
						key, it := attr.result.GetMapItem(i)

						value := ""
						if it != nil {
							value = it.String()
						}
						key_instr, value_instr := attr.instr.GetConstMapPrm(i)

						_, _, _, fnshd1, _ := ui.Comp_editbox(i*3+0, 0, 1, 1, &key, 2, nil, "", false, false, _SANode_isEditable(attr, key_instr))
						if fnshd1 {
							attr.ReplaceMapItemKey(i, key)
						}

						_, _, _, fnshd2, _ := ui.Comp_editbox(i*3+1, 0, 1, 1, &value, 2, nil, "", false, false, _SANode_isEditable(attr, value_instr))
						if fnshd2 {
							attr.ReplaceMapItemValue(i, value)
						}
					}

					if n == 0 {
						ui.Div_colMax(0, 100) //key
						ui.Comp_text(0, 0, 1, 1, "Empty Map {}", 0)
					}
				}
				ui.Div_end()

				if !attr.IsOutput() {
					if ui.Comp_buttonLight(1, 0, 1, 1, "+", "Add item", true) > 0 {
						newKey := fmt.Sprintf("key_%d", n)
						attr.AddParamsItem("\""+newKey+"\": 0", true)
					}
					if ui.Comp_buttonLight(2, 0, 1, 1, "-", "Remove last item", n > 0) > 0 {
						attr.RemoveParamsItem(true)
					}
				}
			}
			ui.Div_end()
		} else {
			//VmBasic_Constant
			_, _, _, fnshd, _ := ui.Comp_editbox(x, y, w, h, &value, 2, nil, "", false, false, editable)
			if fnshd {
				instr.LineReplace(value)
			}
		}
	}
}

func (w *SANode) RenameExpressionAccess(oldName string, newName string) {
	for _, attr := range w.Attrs {
		if attr.instr != nil {
			attr.Value = attr.instr.RenameAccessNode(attr.Value, oldName, newName)
		}
	}

}

func (w *SANode) RenameSubsExpressionAccess(oldName string, newName string) {
	for _, it := range w.Subs {
		it.RenameExpressionAccess(oldName, newName)
	}
}

func (w *SANode) RenderAttrs() {

	ui := w.app.base.ui

	ui.Div_colMax(0, 100)
	if w.IsGuiLayout() {
		ui.Div_row(2, 0.5) //spacer
	} else {
		ui.Div_row(1, 0.5) //spacer
	}

	y := 0

	if w.IsGuiLayout() {
		if ui.Comp_buttonLight(0, y, 1, 1, ui.trns.OPEN, "", true) > 0 {
			w.app.act = w
		}
		y++
	}

	ui.Div_start(0, y, 1, 1)
	{
		ui.Div_colMax(1, 100)
		ui.Div_colMax(2, 3)

		//create new attribute
		if ui.Comp_button(0, 0, 1, 1, "+", "Add attribute", true) > 0 {
			w.AddAttr("attr")
		}

		//Name
		oldName := w.Name
		_, _, _, fnshd, _ := ui.Comp_editbox_desc(ui.trns.NAME, 2, 2, 1, 0, 1, 1, &w.Name, 0, nil, ui.trns.NAME, false, false, true)
		if fnshd && w.parent != nil {
			w.CheckUniqueName()

			//rename access in other nodes expressions
			if w.parent != nil {
				w.parent.RenameSubsExpressionAccess(oldName, w.Name)
			}
		}

		//type
		w.Exe = w.app.ComboListOfNodes(2, 0, 1, 1, w.Exe, ui)

		//context
		{
			dnm := "node_" + w.Name
			if ui.Comp_buttonIcon(3, 0, 1, 1, InitWinMedia_url("file:apps/base/resources/context.png"), 0.3, "", CdPalette_B, true, false) > 0 {
				ui.Dialog_open(dnm, 1)
			}
			if ui.Dialog_start(dnm) {
				ui.Div_colMax(0, 5)
				y := 0

				if ui.Comp_buttonMenu(0, y, 1, 1, ui.trns.DUPLICATE, "", true, false) > 0 {
					nw := w.parent.AddNodeCopy(w)
					nw.SelectOnlyThis()
					ui.Dialog_close()
				}
				y++

				if ui.Comp_buttonMenu(0, y, 1, 1, ui.trns.BYPASS, "", true, false) > 0 {
					w.Bypass = !w.Bypass
					ui.Dialog_close()
				}
				y++

				if ui.Comp_buttonMenu(0, y, 1, 1, ui.trns.REMOVE, "", true, false) > 0 {
					w.Remove()
					ui.Dialog_close()
				}
				y++

				ui.Dialog_end()
			}
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

	hasAttrWith_goto := false
	hasAttrWith_err := false
	for _, it := range w.Attrs {
		if len(it.depends) == 1 {
			hasAttrWith_goto = true
		}
		if it.errExe != nil {
			hasAttrWith_err = true
		}

	}

	for i, it := range w.Attrs {
		ui.Div_start(0, y, 1, 1)
		{
			ui.Div_colMax(2, 3)
			ui.Div_colMax(3, 100)

			if hasAttrWith_goto || hasAttrWith_err {
				ui.Div_col(5, 1)
				if hasAttrWith_goto && hasAttrWith_err {
					ui.Div_col(6, 1)
				}
			}

			//highlight because it has expression
			if len(it.depends) > 0 {
				ui.Paint_rect(0, 0, 1, 1, 0.03, SAApp_getYellow().SetAlpha(50), 0)
			}

			x := 0

			if ui.Comp_buttonIcon(x, 0, 1, 1, InitWinMedia_url("file:apps/base/resources/copy.png"), 0.3, ui.trns.COPY, CdPalette_B, true, false) > 0 {
				keys := &ui.buff.win.io.keys
				keys.clipboard = w.Name + "." + it.Name
			}
			x++

			//edit name
			if !it.exeMark {
				dnm := "rename_" + it.Name

				if ui.Comp_buttonIcon(x, 0, 1, 1, InitWinMedia_url("file:apps/base/resources/edit.png"), 0.22, "Rename", CdPalette_B, true, false) > 0 {
					ui.Dialog_open(dnm, 1)
				}
				if ui.Dialog_start(dnm) {
					ui.Div_colMax(0, 5)
					ui.Comp_editbox(0, 0, 1, 1, &it.Name, 0, nil, "Name", false, false, true)
					it.CheckUniqueName()
					ui.Dialog_end()
				}
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
				//switch: value or expression
				if ui.Comp_buttonMenu(x, 0, 1, 1, it.Name, "", true, it.ShowExp) > 0 {
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

			_SANode_renderAttrValue(x, 0, 1, 1, it, ui)
			x++

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
			} else {
				if hasAttrWith_err {
					x++
				}
			}

			//goto
			{
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
				} else {
					if hasAttrWith_goto {
						x++
					}
				}
			}

			//context
			{
				dnm := "context_" + it.Name
				if ui.Comp_buttonIcon(x, 0, 1, 1, InitWinMedia_url("file:apps/base/resources/context.png"), 0.3, "", CdPalette_B, true, false) > 0 {
					ui.Dialog_open(dnm, 1)
				}
				if ui.Dialog_start(dnm) {
					ui.Div_colMax(0, 5)
					y := 0

					//default
					if ui.Comp_buttonMenu(0, y, 1, 1, "Set default value", "", true, false) > 0 {
						it.Value = it.defaultValue
						ui.Dialog_close()
					}
					y++

					//remove
					if !it.exeMark {
						if ui.Comp_buttonMenu(0, y, 1, 1, "Delete", "", true, false) > 0 {
							w.Attrs = append(w.Attrs[:i], w.Attrs[i+1:]...) //remove
						}
					}
					y++

					ui.Dialog_end()
				}
				x++
			}

		}
		ui.Div_end()
		y++
	}
}

//maybe make dialog with expression like a dialog(bellow is active) and user can click on nodes and their path is insert into expression? .......
