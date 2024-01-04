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

func (w *SANode) Execute(app *SAApp) bool {

	ok := true
	st := OsTime()

	switch w.Exe {
	case "sqlite_select":
		w.Sqlite_select()
		//...
	case "sqlite_update":
		//...
	case "sqlite_delete":
		//...
	case "sqlite_execute":
		//...

	default:
		ok = w.executeProgram(app)
	}

	w.exeTimeSec = OsTime() - st

	fmt.Println(w.Name, "done")
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
			dst.Value = src.finalValue.String()
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

		v.ParseExpresion()
		v.ExecuteExpression() //right now, so default value is in v.finalValue
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

func (w *SANode) GetAttr(name string, value string) *SANodeAttr {
	if value == "" {
		value = "\"\"" //edit
	}
	return w._getAttr(SANodeAttr{Name: name, Value: value})
}

func (w *SANode) GetGrid() OsV4 {
	return w.GetAttr("grid", "[0, 0, 1, 1]").finalValue.Array().GetV4()
}

func (w *SANode) SetGridStart(v OsV2) {
	attr := w.GetAttr("grid", "[0, 0, 1, 1]")
	if attr == nil {
		return
	}
	//y
	a, instr := attr.GetArrayDirectLink(1)
	if instr != nil {
		a.LineReplace(instr, strconv.Itoa(v.Y))
	}

	//x
	a, instr = attr.GetArrayDirectLink(0)
	if instr != nil {
		a.LineReplace(instr, strconv.Itoa(v.X))
	}
}
func (w *SANode) SetGridSize(v OsV2) {
	attr := w.GetAttr("grid", "[0, 0, 1, 1]")
	if attr == nil {
		return
	}
	//h
	a, instr := attr.GetArrayDirectLink(3)
	if instr != nil {
		a.LineReplace(instr, strconv.Itoa(v.Y))
	}

	//w
	a, instr = attr.GetArrayDirectLink(2)
	if instr != nil {
		a.LineReplace(instr, strconv.Itoa(v.X))
	}
}
func (w *SANode) SetGrid(coord OsV4) {
	w.SetGridStart(coord.Start)
	w.SetGridSize(coord.Size)
}

func (w *SANode) GetGridShow() bool {
	return w.GetAttr("grid_show", "bool(1)").GetBool()
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

	switch strings.ToLower(w.Exe) {

	case "button":
		enable := w.GetAttr("enable", "bool(1)").GetBool()
		tp := w.GetAttr("type", "combo(0, \"Classic;Light;Menu;Segments\")").GetInt()

		clicked := false
		switch tp {
		case 0:
			clicked = ui.Comp_button(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, w.GetAttr("label", "").GetString(), "", enable) > 0
		case 1:
			clicked = ui.Comp_buttonLight(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, w.GetAttr("label", "").GetString(), "", enable) > 0
		case 2:
			selected := w.GetAttr("selected", "bool(0)").GetBool()
			clicked = ui.Comp_buttonMenu(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, w.GetAttr("label", "").GetString(), "", enable, selected) > 0
			if clicked {
				sel := w.findAttr("selected")
				sel.SetExpBool(!selected)
			}

		case 3:
			labels := w.GetAttr("label", "").GetString()
			butts := strings.Split(labels, ";")

			selected := w.GetAttr("selected", fmt.Sprintf("combo(0, %s)", labels)).GetInt()
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
						sel.SetExpInt(i)
					}
					if i+1 < len(butts) {
						ui.Div_SpacerCol(i*2+1, 0, 1, 1)
					}
				}
				//ui.Paint_rect(0, 0, 1, 1, 0, ui.buff.win.io.GetPalette().GetGrey(0.5), 0.03)
			}
			ui.Div_end()
		}
		w.GetAttr("clicked", "bool(0)").GetBool()
		cl := w.findAttr("clicked")
		cl.Value = OsTrnString(clicked, "1", "0")

	case "text":
		ui.Comp_text(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, w.GetAttr("label", "").GetString(), w.GetAttr("align", "combo(0, \"Left;Center;Right\")").GetInt())

	case "switch":
		a, instr := w.GetAttr("value", "").GetDirectLink()
		value := a.finalValue.String()
		enable := w.GetAttr("enable", "bool(1)").GetBool() && instr != nil
		if ui.Comp_switch(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &value, false, w.GetAttr("label", "").GetString(), "", enable) {
			if instr != nil {
				a.LineReplace(instr, value)
			}
		}

	case "checkbox":
		a, instr := w.GetAttr("value", "").GetDirectLink()
		value := a.finalValue.String()
		enable := w.GetAttr("enable", "bool(1)").GetBool() && instr != nil
		if ui.Comp_checkbox(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &value, false, w.GetAttr("label", "").GetString(), "", enable) {
			if instr != nil {
				a.LineReplace(instr, value)
			}
		}

	case "combo":
		a, instr := w.GetAttr("value", "").GetDirectLink()
		value := a.finalValue.String()
		enable := w.GetAttr("enable", "bool(1)").GetBool() && instr != nil
		if ui.Comp_combo(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &value, w.GetAttr("options", "\"a;b;c\")").GetString(), "", enable, w.GetAttr("search", "bool(0)").GetBool()) {
			if instr != nil {
				a.LineReplace(instr, value)
			}
		}

	case "editbox":
		a, instr := w.GetAttr("value", "").GetDirectLink()
		value := a.finalValue.String()
		enable := w.GetAttr("enable", "bool(1)").GetBool() && instr != nil
		tmpToValue := w.GetAttr("tempToValue", "bool(0)").GetBool()
		_, _, chngd, fnshd, _ := ui.Comp_editbox(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, &value, w.GetAttr("precision", "2").GetInt(), "", w.GetAttr("ghost", "").GetString(), false, tmpToValue, enable)
		if fnshd || (tmpToValue && chngd) {
			if instr != nil {
				a.LineReplace(instr, value)
			}
		}

	case "divider":
		tp := w.GetAttr("type", "combo(0, \"Column;Row\"").GetInt()
		switch tp {
		case 0:
			ui.Div_SpacerCol(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y)
		case 1:
			ui.Div_SpacerRow(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y)
		}

	case "color_palette":
		ui.Div_startName(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, w.Name)
		{
			cdAttr := w.GetAttr("cd", "color(0, 0, 0, 255)")
			cd := cdAttr.GetCd()
			if ui.comp_colorPalette(&cd) {
				cdAttr.SetCd(cd)
			}
		}
		ui.Div_end()

	case "color":
		ui.Div_startName(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, w.Name)
		{
			//........
			enable := w.GetAttr("enable", "bool(1)").GetBool()
			cdAttr := w.GetAttr("cd", "color(0, 0, 0, 255)")
			cd := cdAttr.GetCd()
			if ui.comp_colorPicker(&cd, w.Name, enable) {
				cdAttr.SetCd(cd)
			}
		}
		ui.Div_end()

	case "calendar":
		ui.Div_startName(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, w.Name)
		{
			value := w.GetAttr("value", "date(0)").GetInt64()
			page := w.GetAttr("page", "date(0)").GetInt64()

			ui.Comp_Calendar(&value, &page, 100, 100)

			w.GetAttr("value", "date(0)").SetExpInt(int(value))
			w.GetAttr("page", "date(0)").SetExpInt(int(page))
		}
		ui.Div_end()

	case "date":
		ui.Div_startName(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, w.Name)
		{
			enable := w.GetAttr("enable", "bool(1)").GetBool()
			value := w.GetAttr("value", "date(0)").GetInt64()
			show_time := w.GetAttr("show_time", "bool(0)").GetBool()
			if ui.Comp_CalendarDataPicker(&value, show_time, w.Name, enable) {
				w.GetAttr("value", "date(0)").SetExpInt(int(value))
			}
		}
		ui.Div_end()

	case "map":
		ui.Div_startName(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, w.Name)
		{
			file := w.GetAttr("file", "\"maps/osm\"").GetString()
			url := w.GetAttr("url", "\"https://tile.openstreetmap.org/{z}/{x}/{y}.png\"").GetString()
			copyright := w.GetAttr("copyright", "\"(c)OpenStreetMap contributors\"").GetString()
			copyright_url := w.GetAttr("copyright_url", "\"https://www.openstreetmap.org/copyright\"").GetString()

			file = "disk/" + file

			cam_lon := w.GetAttr("lon", "14.4071117049").GetFloat()
			cam_lat := w.GetAttr("lat", "50.0852013259").GetFloat()
			cam_zoom := w.GetAttr("zoom", "5").GetFloat()

			err := ui.comp_map(app.mapp, &cam_lon, &cam_lat, &cam_zoom, file, url, copyright, copyright_url)
			if err != nil {
				w.errExe = err
			}

			//set back
			w.GetAttr("lon", "0").SetExpFloat(cam_lon)
			w.GetAttr("lat", "0").SetExpFloat(cam_lat)
			w.GetAttr("zoom", "5").SetExpInt(int(cam_zoom))

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
			items := w.GetAttr("items", "\"[{\"lon\":14.4071117049, \"lat\":50.0852013259, \"label\":\"1\"}, {\"lon\":14, \"lat\":50, \"label\":\"2\"}]\"").GetString()

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

func _SANode_renderAttrValue(x, y, w, h int, attr *SANodeAttr, ui *Ui) {

	if attr.ShowExp {
		ui.Comp_editbox(x, y, w, h, &attr.Value, 2, "", "", false, false, true) //show whole expression
	} else {

		if attr.instr == nil {
			return
		}

		fn := attr.instr.fn

		//if VmCallback_Cmp(fn, VmApi_GuiBool) {
		//	attr.instr.prms[0].instr.GetDirectDirectAccess()
		//}

		//send data through link
		a, instr := attr.GetDirectLink()
		value := a.finalValue.String()

		if VmCallback_Cmp(fn, VmApi_GuiBool) {
			if ui.Comp_switch(x, y, w, h, &value, false, "", "", instr != nil) {
				a.LineReplace(instr, value)
			}
		} else if VmCallback_Cmp(fn, VmApi_GuiBool2) {
			if ui.Comp_checkbox(x, y, w, h, &value, false, "", "", instr != nil) {
				a.LineReplace(instr, value)
			}
		} else if VmCallback_Cmp(fn, VmApi_GuiDate) {
			ui.Div_start(x, y, w, h)
			val := int64(a.finalValue.Number())
			if ui.Comp_CalendarDataPicker(&val, true, attr.Name, instr != nil) {
				a.LineReplace(instr, strconv.Itoa(int(val)))
			}
			ui.Div_end()
		} else if VmCallback_Cmp(fn, VmApi_GuiCombo) {
			options := instr.parent.temp.value.String() //instr is first parameter, GuiCombo() api is parent!
			if ui.Comp_combo(x, y, w, h, &value, options, "", instr != nil, false) {
				a.LineReplace(instr, value)
			}
		} else if VmCallback_Cmp(fn, VmApi_GuiColor) {
			ui.Div_start(x, y, w, h)
			{
				cd := a.GetCd()
				if ui.comp_colorPicker(&cd, attr.Name, true) {
					if instr != nil {
						a.LineReplace(instr, value)
					}
				}
			}
			ui.Div_end()

		} else if VmCallback_Cmp(fn, VmBasic_ConstArray) {
			ui.Div_start(x, y, w, h)
			{
				for i := range attr.instr.prms {
					ui.Div_colMax(i, 100)
				}

				arr := attr.finalValue.Array()
				for i := range attr.instr.prms {

					a, instr = attr.GetArrayDirectLink(i)
					if i < arr.Num() {
						value = arr.Get(i).String()
						_, _, _, fnshd, _ := ui.Comp_editbox(i, 0, 1, 1, &value, 2, "", "", false, false, instr != nil)
						if fnshd {
							a.LineReplace(instr, value)
						}
					}
				}
			}
			ui.Div_end()

		} else if VmCallback_Cmp(fn, VmBasic_ConstTable) {
			tb := a.finalValue.Table()
			ui.Comp_button(x, y, w, h, fmt.Sprintf("Table(%dcols x %drow)", len(tb.names), tb.NumRows()), "", true) //......
		} else /*if VmCallback_Cmp(fn, VmBasic_Constant)*/ {
			_, _, _, fnshd, _ := ui.Comp_editbox(x, y, w, h, &value, 2, "", "", false, false, instr != nil)
			if fnshd {
				a.LineReplace(instr, value)
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
		_, _, _, fnshd, _ := ui.Comp_editbox_desc(ui.trns.NAME, 0, 2, 0, 0, 1, 1, &w.Name, 0, "", ui.trns.NAME, false, false, true)
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

			if ui.Comp_buttonIcon(x, 0, 1, 1, "file:apps/base/resources/copy.png", 0.3, ui.trns.COPY, CdPalette_B, true, false) > 0 {
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
