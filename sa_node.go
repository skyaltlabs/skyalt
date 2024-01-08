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

	if len(a.Cols) != len(b.Cols) {
		*historyDiff = true //no return!
	}
	if len(a.Rows) != len(b.Rows) {
		*historyDiff = true //no return!
	}

	if len(a.Attrs) != len(b.Attrs) {
		return false
	}
	if len(a.Subs) != len(b.Subs) {
		return false
	}

	for i, itA := range a.Attrs {
		itB := b.Attrs[i]
		if !itA.Cmp(itB) {
			return false
		}
	}

	for i, itA := range a.Cols {
		if !itA.Cmp(&b.Cols[i]) {
			*historyDiff = true //no return!
		}
	}
	for i, itA := range a.Rows {
		if !itA.Cmp(&b.Rows[i]) {
			*historyDiff = true //no return!
		}
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

func (w *SANode) ResetAttrsExeErrors() {
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
		a.useMark = false
	}

	for _, n := range w.Subs {
		n.markUnusedAttrs()
	}
}
func (w *SANode) removeUnusedAttrs() {

	if !w.Bypass {
		for i := len(w.Attrs) - 1; i >= 0; i-- {
			if !w.Attrs[i].useMark {
				//..... w.Attrs = append(w.Attrs[:i], w.Attrs[i+1:]...) //remove
			}
		}
	}
	for _, n := range w.Subs {
		n.removeUnusedAttrs()
	}
}

func (w *SANode) ExecuteGui(renderIt bool) {

	//w.ResetAttrsExeErrors()

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

func (w *SANode) Execute(app *SAApp) bool {

	ok := true
	st := OsTime()

	switch strings.ToLower(w.Exe) {
	case "sqlite_select":
		ok = w.Sqlite_select()
	case "sqlite_insert":
		//ok = w.Sqlite_insert()
	case "sqlite_update":
		//...
	case "sqlite_delete":
		//...
	case "sqlite_execute":
		//...

	case "csv_select":
		ok = w.Csv_select()

	case "blob":
		ok = w.ConstBlob()
	case "array":
		ok = w.ConstArray()
	case "table":
		ok = w.ConstTable()

	default:
		if SAApp_IsExternal(w.Exe) {
			ok = w.executeProgram(app)
		}
	}

	w.exeTimeSec = OsTime() - st
	fmt.Printf("'%s' done in %.2fs\n", w.Name, w.exeTimeSec)
	return ok
}

func (w *SANode) executeProgram(app *SAApp) bool {

	fmt.Println("execute:", w.Name)

	w.errExe = nil
	w.progress = 0
	w.progress_desc = ""

	conn := app.base.server.Start(w.Exe)
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

func (w *SANode) Copy(app *SAApp) (*SANode, error) {

	js, err := json.Marshal(w)
	if err != nil {
		return nil, err
	}

	dst := NewSANode(app, nil, "", "", OsV4{}, OsV2f{})
	err = json.Unmarshal(js, dst)
	if err != nil {
		return nil, err
	}

	dst.updateLinks(nil, app)

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
	nw := NewSANode(w.app, w, exe, exe, grid, pos)
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
			it.useMark = true
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

	if v.Value == "" {
		v.Value = "\"\"" //edit
	}

	if v.instr == nil {
		v.ParseExpresion()
		v.ExecuteExpression() //right now, so default value is in v.finalValue
	}

	v.defaultValue = defValue.Value //update
	v.useMark = true
	return v
}

func (w *SANode) GetAttr(name string, value string) *SANodeAttr {
	return w._getAttr(SANodeAttr{Name: name, Value: value})
}
func (w *SANode) GetAttrOutput(name string, value string) *SANodeAttr {
	a := w.GetAttr(name, value)
	a.Output = true
	return a
}

func (w *SANode) GetGrid() OsV4 {
	return w.GetAttr("grid", "[0, 0, 1, 1]").result.Array().GetV4()
}

func (w *SANode) SetGridStart(v OsV2) {
	attr := w.GetAttr("grid", "[0, 0, 1, 1]")
	if attr == nil {
		return
	}

	attr.ReplaceArrayItemInt(1, v.Y) //y
	attr.ReplaceArrayItemInt(0, v.X) //x
}
func (w *SANode) SetGridSize(v OsV2) {
	attr := w.GetAttr("grid", "[0, 0, 1, 1]")
	if attr == nil {
		return
	}

	attr.ReplaceArrayItemInt(3, v.Y) //h
	attr.ReplaceArrayItemInt(2, v.X) //w
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

func (w *SANode) RenderLayout() {

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

	//items
	for _, it := range w.Subs {
		it.Render()
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
		value := ""
		if instr != nil {
			value = instr.pos_attr.result.String()
		}

		editable := (!attr.Output && instr != nil)

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
			options := attr.instr.temp.String() //instr is first parameter, GuiCombo() api is parent!
			if ui.Comp_combo(x, y, w, h, &value, options, "", editable, false) {
				instr.LineReplace(value)
			}
		} else if VmCallback_Cmp(fn, VmApi_UiColor) {
			cd := attr.GetCd()
			if ui.comp_colorPicker(x, y, w, h, &cd, attr.Name, true) {
				attr.ReplaceCd(cd)
			}
		} else if VmCallback_Cmp(fn, VmApi_UiBlob) {
			if attr.result.IsBlob() {
				blob := attr.result.Blob()
				dnm := "blob_" + attr.Name
				if ui.Comp_buttonIcon(x, y, w, h, InitWinMedia_blob(blob.data, blob.hash), 0, "", CdPalette_White, true, false) > 0 {
					ui.Dialog_open(dnm, 0)
				}
				if ui.Dialog_start(dnm) {
					ui.Div_colMax(0, 20)
					ui.Div_rowMax(0, 20)
					ui.Comp_image(0, 0, 1, 1, InitWinMedia_blob(blob.data, blob.hash), InitOsCd32(255, 255, 255, 255), 0, 1, 1, false)
					ui.Dialog_end()
				}
			} else {
				ui.Comp_text(x, y, w, h, "Error: Not Blob", 1)
			}
		} else if VmCallback_Cmp(fn, VmBasic_ConstArray) {
			ui.Div_start(x, y, w, h)
			{
				for i := range attr.instr.prms {
					ui.Div_colMax(i, 100)
				}

				arr := attr.result.Array()
				for i := range attr.instr.prms {
					if i < arr.Num() {
						value = arr.Get(i).String()
						instr2 := attr.instr.GetConstArrayPrm(i)
						_, _, _, fnshd, _ := ui.Comp_editbox(i, 0, 1, 1, &value, 2, nil, "", false, false, !attr.Output && instr2 != nil)
						if fnshd {
							attr.ReplaceArrayItem(i, value)
						}
					}
				}
			}
			ui.Div_end()

		} else if VmCallback_Cmp(fn, VmBasic_ConstTable) {
			tb := attr.result.Table()
			ui.Comp_button(x, y, w, h, fmt.Sprintf("Table(%dcols x %drow)", len(tb.names), tb.NumRows()), "", true) //......
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
	for _, it := range w.Subs {
		for _, attr := range it.Attrs {
			if attr.instr != nil {
				attr.Value = attr.instr.RenameAccessNode(attr.Value, oldName, newName)
			}
		}
	}
}

func (w *SANode) RenderAttrs(app *SAApp) {

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
		ui.Div_colMax(2, 3)
		ui.Div_colMax(3, 2)

		//Name
		oldName := w.Name
		_, _, _, fnshd, _ := ui.Comp_editbox_desc(ui.trns.NAME, 0, 2, 0, 0, 1, 1, &w.Name, 0, nil, ui.trns.NAME, false, false, true)
		if fnshd && w.parent != nil {
			w.CheckUniqueName()

			//rename access in other nodes expressions
			if w.parent != nil {
				w.parent.RenameExpressionAccess(oldName, w.Name)
			}
		}

		//type
		w.Exe = app.ComboListOfNodes(1, 0, 1, 1, w.Exe, ui)

		//context with duplicate(rename! + edit expressions to keep links between new nodes) / delete ......

		//bypass
		ui.Comp_switch(2, 0, 1, 1, &w.Bypass, false, ui.trns.BYPASS, "", true)

		//delete
		if ui.Comp_button(3, 0, 1, 1, ui.trns.REMOVE, "", true) > 0 {
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
		ui.Div_start(0, y, 1, 1)
		{
			ui.Div_colMax(1, 3)
			ui.Div_colMax(2, 100)

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
				nm := it.Name
				if it.Output {
					nm += "(OUT)"
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

			_SANode_renderAttrValue(x, 0, 1, 1, it, ui)
			x++

			//context
			{
				dnm := "context_" + it.Name
				if ui.Comp_buttonIcon(x, 0, 1, 1, InitWinMedia_url("file:apps/base/resources/context.png"), 0.3, "", CdPalette_B, true, false) > 0 {
					ui.Dialog_open(dnm, 1)
				}
				if ui.Dialog_start(dnm) {
					ui.Div_colMax(0, 5)
					if ui.Comp_buttonMenu(0, 0, 1, 1, "Reset value to default", "", true, false) > 0 {
						it.Value = it.defaultValue
						ui.Dialog_close()
					}
					ui.Dialog_end()
				}
				x++
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
				}
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
