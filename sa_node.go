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

	"github.com/go-audio/audio"
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

	temp_mic_data audio.IntBuffer
}

func NewSANode(app *SAApp, parent *SANode, name string, exe string, grid OsV4, pos OsV2f) *SANode {
	w := &SANode{}
	w.parent = parent
	w.app = app
	w.Name = name
	w.Exe = exe
	w.Pos = pos
	w.Cam_z = 1

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

func (a *SANode) Distance(b *SANode) float32 {
	v := a.Pos.Sub(b.Pos)
	return v.toV2().Len()
}
func (node *SANode) GetDependDistance() float32 {
	sum := float32(0)
	for _, attr := range node.Attrs {
		for _, d := range attr.depends {
			sum += node.Distance(d.node)
		}
	}
	return sum
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

	return true
}

func (w *SANode) buildSubList(list *[]*SANode) {
	for _, it := range w.Subs {

		*list = append(*list, it)

		if !SAGroups_IsNodeFor(it.Exe) {
			it.buildSubList(list)
		}
	}
}

func (w *SANode) IsGuiLayout() bool {
	return SAGroups_HasNodeSub(w.Exe)
}
func (w *SANode) CanBeRenderOnCanvas() bool {
	return w.app.base.node_groups.IsUI(w.Exe)
}

func (w *SANode) markUnusedAttrs() {
	for _, a := range w.Attrs {
		a.exeMark = false
	}

	for _, n := range w.Subs {
		n.markUnusedAttrs()
	}
}

func (w *SANode) ExecuteGui() {

	w.z_depth = 1

	gnd := w.app.base.node_groups.FindNode(w.Exe)
	if gnd != nil && gnd.render != nil {
		gnd.render(w, true)
	}
}

func (w *SANode) Execute() bool {

	ok := true
	st := OsTime()

	gnd := w.app.base.node_groups.FindNode(w.Exe)
	if gnd != nil {
		if gnd.fn != nil {
			ok = gnd.fn(w)
		} else {
			gnd.render(w, false)
		}
	} else {
		fmt.Printf("Unknown node: %s", w.Exe)
		ok = false
	}

	w.exeTimeSec = OsTime() - st
	//fmt.Printf("'%s' done in %.2fs\n", w.Name, w.exeTimeSec)
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
	if w.Cam_z <= 0 {
		w.Cam_z = 1
	}

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

func (w *SANode) AddNode(grid OsV4, pos OsV2f, name string, exe string) *SANode {
	nw := NewSANode(w.app, w, name, exe, grid, pos)
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
	v.Ui = defValue.Ui
	v.exeMark = true
	return v
}

func (w *SANode) AddAttr(name string) *SANodeAttr {
	return w._getAttr(false, SANodeAttr{Name: name})
}
func (w *SANode) GetAttr(name string, value string) *SANodeAttr {
	return w._getAttr(true, SANodeAttr{Name: name, Value: value})
}
func (w *SANode) GetAttrUi(name string, value string, ui SAAttrUiValue) *SANodeAttr {
	return w._getAttr(true, SANodeAttr{Name: name, Value: value, Ui: ui})
}

func (w *SANode) GetGrid() OsV4 {
	grid := w.GetAttr("grid", "[0, 0, 1, 1]").GetV4()

	//check
	grid.Size.X = OsMax(grid.Size.X, 1)
	grid.Size.Y = OsMax(grid.Size.Y, 1)

	return grid

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
	w.SetGridSize(coord.Size)
	w.SetGridStart(coord.Start)
}

func (w *SANode) GetGridShow() bool {
	return w.GetAttrUi("grid_show", "1", SAAttrUi_SWITCH).GetBool()
}

func (w *SANode) Render() {

	ui := w.app.base.ui

	w.ExecuteGui()

	if w.app.IDE && w.CanBeRenderOnCanvas() {

		grid := w.GetGrid()
		grid.Size.Y = OsMax(grid.Size.Y, 1)

		//draw Select rectangle
		if w.app.act == w.parent {
			if w.HasExeError() || w.HasExpError() {
				pl := ui.win.io.GetPalette()
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
				if en.Inside(ui.win.io.touch.pos) && ui.win.io.keys.alt {
					pl := ui.win.io.GetPalette()
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

func _SANode_renderAttrValueHeight(attr *SANodeAttr, attr_instr *VmInstr, uiVal *SAAttrUiValue) int {
	height := 1

	if !attr.ShowExp {
		if attr_instr == nil {
			return height
		}

		instr := attr_instr.GetConst()

		if instr != nil && VmCallback_Cmp(instr.fn, VmBasic_InitArray) {
			item_pre_row := (uiVal.Fn == "map")
			n := instr.NumPrms()
			if item_pre_row {
				height = OsMax(height, n+1) //+1 => (add/sub row)
			}
		}
	}

	return height
}

func _SANode_renderAttrValue(x, y, w, h int, attr *SANodeAttr, attr_instr *VmInstr, isOutput bool, uiVal *SAAttrUiValue, ui *Ui) {

	if attr != nil && attr.ShowExp {
		ui.Comp_editbox(x, y, w, h, &attr.Value, 2, 0, nil, "", false, false, false, true) //show whole expression
	} else {

		if attr_instr == nil {
			return
		}

		instr := attr_instr.GetConst()

		var value string
		if instr != nil {
			value = instr.temp.String()
		} else {
			value = attr_instr.temp.String() //result(disabled) from expression
		}

		editable := !isOutput && instr != nil && !instr.pos_attr.IsOutput()

		if uiVal.Fn == SAAttrUi_SWITCH.Fn {
			if ui.Comp_switch(x, y, w, h, &value, false, "", "", editable) {
				instr.LineReplace(value, false)
			}
		} else if uiVal.Fn == SAAttrUi_CHECKBOX.Fn {
			if ui.Comp_checkbox(x, y, w, h, &value, false, "", "", editable) {
				instr.LineReplace(value, false)
			}
		} else if uiVal.Fn == SAAttrUi_DATE.Fn {
			ui.Div_start(x, y, w, h)
			val := int64(instr.temp.Number())
			if ui.Comp_CalendarDataPicker(&val, true, "attr_calendar", editable) {
				instr.LineReplace(strconv.Itoa(int(val)), false)
			}
			ui.Div_end()
		} else if uiVal.Fn == "combo" {
			if ui.Comp_combo(x, y, w, h, &value, strings.Split(uiVal.Prm, ";"), strings.Split(uiVal.Prm2, ";"), "", editable, false) {
				instr.LineReplace(value, false)
			}
		} else if instr != nil && uiVal.Fn == SAAttrUi_COLOR.Fn {
			cd := instr.temp.Cd()
			if ui.comp_colorPicker(x, y, w, h, &cd, "attr_cd", true) {
				if editable {
					instr.pos_attr.ReplaceCd(cd)
				}
			}
		} else if uiVal.Fn == SAAttrUi_DIR.Fn {
			if ui.comp_dirPicker(x, y, w, h, &value, false, "attr_folder", editable) {
				instr.LineReplace(value, false)
			}
		} else if uiVal.Fn == SAAttrUi_FILE.Fn {
			if ui.comp_dirPicker(x, y, w, h, &value, true, "attr_file", editable) {
				instr.LineReplace(value, false)
			}
		} else if instr != nil && uiVal.Fn == SAAttrUi_BLOB.Fn {
			blob := instr.temp.Blob()
			if ui.Comp_buttonIcon(x, y, w, h, InitWinMedia_blob(blob), 0, "", CdPalette_White, true, false) > 0 {
				ui.Dialog_open("attr_blob", 0)
			}
			if ui.Dialog_start("attr_blob") {
				ui.Div_colMax(0, 20)
				ui.Div_rowMax(0, 20)
				ui.Comp_image(0, 0, 1, 1, InitWinMedia_blob(blob), InitOsCdWhite(), 0, 1, 1, false)
				ui.Dialog_end()
			}
		} else if instr != nil && VmCallback_Cmp(instr.fn, VmBasic_InitArray) {
			ui.Div_start(x, y, w, h)
			{
				//show first 4, then "others" -> open dialog? .....

				item_pre_row := (uiVal.Fn == "map")
				n := instr.NumPrms()
				var addDelPos OsV4
				if n == 0 {
					//empty
					ui.Div_colMax(0, 100) //key
					ui.Comp_text(0, 0, 1, 1, "Empty Array []", 0)
					addDelPos = InitOsV4(1, 0, 2, 1)
				} else if item_pre_row {
					//multiple lines
					ui.Div_colMax(0, 100)
					for i := 0; i < n; i++ {
						item_instr := instr.GetConstArrayPrm(i)
						_SANode_renderAttrValue(0, i, 1, 1, nil, item_instr, isOutput, uiVal, ui)
					}

					//reorder rows .....
					//remove row .....

					addDelPos = InitOsV4(0, n, 1, 1)
				} else {
					//single line
					ui.Div_colMax(0, 100)
					ui.DivInfo_set(SA_DIV_SET_scrollHnarrow, 1, 0)
					ui.Div_row(0, 0.5)
					ui.Div_rowMax(0, 2)

					for i := 0; i < n; i++ {
						ui.Div_colMax(i, 100)
					}

					for i := 0; i < n; i++ {
						//item_instr := instr.GetConstArrayPrm(i)
						item_instr := instr.prms[i].value
						_SANode_renderAttrValue(i, 0, 1, 1, nil, item_instr, isOutput, uiVal, ui)
					}
					addDelPos = InitOsV4(n, 0, 2, 1)
				}

				//+/-
				if !isOutput {
					ui.Div_start(addDelPos.Start.X, addDelPos.Start.Y, addDelPos.Size.X, addDelPos.Size.Y)
					{
						ui.DivInfo_set(SA_DIV_SET_scrollHshow, 0, 0)
						ui.DivInfo_set(SA_DIV_SET_scrollVshow, 0, 0)

						if ui.Comp_buttonLight(0, 0, 1, 1, "+", "Add item", true) > 0 {
							if editable {

								newVal := "0"
								if uiVal.Fn == "map" {
									//create new value from map items
									newVal = "{"
									for k := range uiVal.Map {
										newVal += "\"" + k + "\":" + "\"\", "
									}
									newVal, _ = strings.CutSuffix(newVal, ",")
									newVal += "}"
								}

								instr.pos_attr.AddParamsItem(newVal, false)
							}
						}
						if ui.Comp_buttonLight(1, 0, 1, 1, "-", "Remove last item", n > 0) > 0 {
							if editable {
								instr.pos_attr.RemoveParamsItem(false)
							}
						}
					}
					ui.Div_end()
				}

			}
			ui.Div_end()
		} else if instr != nil && VmCallback_Cmp(instr.fn, VmBasic_InitMap) {
			ui.Div_start(x, y, w, h)
			{
				//show first 4, then "others" -> open dialog? .....

				n := instr.NumPrms()

				ui.Div_colMax(0, 100)
				ui.Div_start(0, 0, 1, 1)
				{
					ui.DivInfo_set(SA_DIV_SET_scrollHnarrow, 1, 0)
					ui.Div_row(0, 0.5)
					ui.Div_rowMax(0, 2)

					showKey := (uiVal.Fn != "map")

					for i := 0; i < n; i++ {
						ui.Div_colMax(i*3+0, OsTrnFloat(showKey, 100, 2)) //key
						ui.Div_colMax(i*3+1, 100)                         //value
						ui.Div_col(i*3+2, 0.2)                            //space
					}

					for i := 0; i < n; i++ {
						key_instr := instr.prms[i].key
						val_instr := instr.prms[i].value
						//key_instr, val_instr := instr.GetConstMapPrm(i)

						if showKey {
							_SANode_renderAttrValue(i*3+0, 0, 1, 1, nil, key_instr, isOutput, &SAAttrUiValue{}, ui) //key
							_SANode_renderAttrValue(i*3+1, 0, 1, 1, nil, val_instr, isOutput, &SAAttrUiValue{}, ui) //value
						} else {
							ui_key := key_instr.temp.String()
							ui_ui := uiVal.Map[ui_key]

							//drag & drop to re-order .....
							ui.Comp_text(i*3+0, 0, 1, 1, ui_key, 2)                                       //key
							_SANode_renderAttrValue(i*3+1, 0, 1, 1, nil, val_instr, isOutput, &ui_ui, ui) //value
						}
					}

					if n == 0 {
						ui.Div_colMax(0, 100) //key
						ui.Comp_text(0, 0, 1, 1, "Empty Map {}", 0)
					}
				}
				ui.Div_end()

				//+/-
				if !isOutput && !uiVal.HideAddDel {
					if ui.Comp_buttonLight(1, 0, 1, 1, "+", "Add item", true) > 0 {
						if editable {
							newKey := fmt.Sprintf("key_%d", n)
							instr.pos_attr.AddParamsItem("\""+newKey+"\": 0", true)
						}
					}
					if ui.Comp_buttonLight(2, 0, 1, 1, "-", "Remove last item", n > 0) > 0 {
						if editable {
							instr.pos_attr.RemoveParamsItem(true)
						}
					}
				}
			}
			ui.Div_end()
		} else {
			//VmBasic_Constant
			_, _, _, fnshd, _ := ui.Comp_editbox(x, y, w, h, &value, 2, 0, nil, "", false, false, false, editable)
			if fnshd {
				instr.LineReplace(value, false)
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
		_, _, _, fnshd, _ := ui.Comp_editbox_desc(ui.trns.NAME, 2, 2, 1, 0, 1, 1, &w.Name, 0, 0, nil, ui.trns.NAME, false, false, false, true)
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
			pl := ui.win.io.GetPalette()
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
		h := _SANode_renderAttrValueHeight(it, it.instr, &it.Ui)
		h = OsMin(h, 5)

		ui.Div_start(0, y, 1, h)
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
				keys := &ui.win.io.keys
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
					ui.Comp_editbox(0, 0, 1, 1, &it.Name, 0, 0, nil, "Name", false, false, false, true)
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
			ui.Div_start(x, 0, 1, h)
			{
				ui.Div_colMax(0, 100)
				ui.Div_row(0, float64(h))

				if it.errExp != nil {
					ui.Paint_tooltip(0, 0, 1, 1, it.errExp.Error())
					pl := ui.win.io.GetPalette()
					ui.Paint_rect(0, 0, 1, 1, 0, pl.E, 0.03)
				} else if it.ShowExp {
					ui.Paint_rect(0, 0, 1, 1, 0, SAApp_getYellow(), 0.03)
				}

				_SANode_renderAttrValue(0, 0, 1, 1, it, it.instr, it.IsOutput(), &it.Ui, ui)
			}
			ui.Div_end()
			x++

			//error
			if it.errExe != nil {
				ui.Div_start(x, 0, 1, 1)
				{
					ui.Paint_tooltip(0, 0, 1, 1, it.errExe.Error())
					pl := ui.win.io.GetPalette()
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
		y += h
	}
}

//maybe make dialog with expression like a dialog(bellow is active) and user can click on nodes and their path is insert into expression? .......
