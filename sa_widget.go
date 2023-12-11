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
)

type SAWidgetValue struct {
	widget *SAWidget

	Name  string
	Value string

	Gui_type     string `json:",omitempty"`
	Gui_options  string `json:",omitempty"`
	Gui_ReadOnly bool   `json:",omitempty"` //output
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
	Coord    OsV4
	Selected bool

	Values []*SAWidgetValue `json:",omitempty"`

	//sub-layout
	Cols []SAWidgetColRow `json:",omitempty"`
	Rows []SAWidgetColRow `json:",omitempty"`
	Subs []*SAWidget      `json:",omitempty"`
}

func NewSAWidget(parent *SAWidget, name string, node string, coord OsV4) *SAWidget {
	w := &SAWidget{}
	w.parent = parent
	w.Name = name
	w.Node = node
	w.Coord = coord
	//w.Values = make(map[string]*SAWidgetValue)
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
	if a.Name != b.Name || a.Node != b.Node || !a.Coord.Cmp(b.Coord) || a.Selected != b.Selected {
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

func (w *SAWidget) UpdateParents(parent *SAWidget) {
	w.parent = parent

	//if w.Values == nil {
	//	w.Values = make(map[string]*SAWidgetValue)
	//}

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
	return w._getValue(SAWidgetValue{Name: name, Value: defValue, Gui_type: "gui_edit"}).Value
}

func (w *SAWidget) GetValueStringPtrEdit(name string, defValue string) *string {
	return &w._getValue(SAWidgetValue{Name: name, Value: defValue, Gui_type: "gui_edit"}).Value
}

func (w *SAWidget) GetValueIntEdit(name string, defValue string) int {
	vv, _ := strconv.Atoi(w.GetValueStringEdit(name, defValue))
	return vv
}

func (w *SAWidget) GetValueIntCombo(name string, defValue string, defOptions string) int {
	vv, _ := strconv.Atoi(w._getValue(SAWidgetValue{Name: name, Value: defValue, Gui_type: "gui_combo", Gui_options: defOptions}).Value)
	return vv
}

func (w *SAWidget) GetValueBoolSwitch(name string, defValue string) bool {
	vv, _ := strconv.Atoi(w._getValue(SAWidgetValue{Name: name, Value: defValue, Gui_type: "gui_switch"}).Value)
	return vv != 0
}

func (w *SAWidget) Render(ui *Ui) {

	switch w.Node {
	case "gui_button":
		ui.Comp_button(w.Coord.Start.X, w.Coord.Start.Y, w.Coord.Size.X, w.Coord.Size.Y, w.GetValueStringEdit("value", ""), "", w.GetValueBoolSwitch("enable", "1"))

		//outputs(readOnly): is clicked ...

	case "gui_text":
		ui.Comp_text(w.Coord.Start.X, w.Coord.Start.Y, w.Coord.Size.X, w.Coord.Size.Y, w.GetValueStringEdit("value", ""), w.GetValueIntCombo("align", "0", "Left|Center|Right"))

	case "gui_edit":
		ui.Comp_editbox(w.Coord.Start.X, w.Coord.Start.Y, w.Coord.Size.X, w.Coord.Size.Y, w.GetValueStringPtrEdit("value", ""), w.GetValueIntEdit("precision", "2"), "", w.GetValueStringEdit("ghost", ""), false, w.GetValueBoolSwitch("tempToValue", "0"), w.GetValueBoolSwitch("enable", "1"))

	case "gui_combo":
		ui.Comp_combo(w.Coord.Start.X, w.Coord.Start.Y, w.Coord.Size.X, w.Coord.Size.Y, w.GetValueStringPtrEdit("value", ""), w.GetValueStringEdit("options", "a|b"), "", w.GetValueBoolSwitch("enable", "1"), w.GetValueBoolSwitch("search", "0"))

	case "gui_layout":
		ui.Div_start(w.Coord.Start.X, w.Coord.Start.Y, w.Coord.Size.X, w.Coord.Size.Y)
		w.RenderLayout(ui)
		ui.Div_end()
	}

	if w.Selected {
		div := ui.Div_start(w.Coord.Start.X, w.Coord.Start.Y, w.Coord.Size.X, w.Coord.Size.Y)
		div.enableInput = false
		ui.Paint_rect(0, 0, 1, 1, 0, Node_getYellow(), 0.03)
		ui.Div_end()
	}

}

func (w *SAWidget) RenderLayout(ui *Ui) {

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
		it.Render(ui)
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

func (w *SAWidget) RenderParams(ui *Ui) {

	ui.Div_colMax(0, 100)

	ui.Div_colMax(1, 0.1) //spacer

	y := 0

	//Name
	_, _, _, fnshd, _ := ui.Comp_editbox_desc("Name", 0, 2, 0, y, 1, 1, &w.Name, 0, "", "Name", false, false, true)
	if fnshd && w.parent != nil {
		//create unique name
		for w.parent.NumNames(w.Name) >= 2 {
			w.Name += "1"
		}
	}
	y++

	ui.Div_SpacerRow(0, y, 1, 1)
	y++

	//Grid
	ui.Div_start(0, y, 1, 1)
	{
		ui.Div_colMax(0, 2)
		ui.Div_colMax(1, 100)
		ui.Div_colMax(2, 100)
		ui.Div_colMax(3, 100)
		ui.Div_colMax(4, 100)
		ui.Comp_text(0, 0, 1, 1, "Grid", 0)

		ui.Comp_editbox(1, 0, 1, 1, &w.Coord.Start.X, 0, "", "x", false, false, true)
		ui.Comp_editbox(2, 0, 1, 1, &w.Coord.Start.Y, 0, "", "y", false, false, true)
		ui.Comp_editbox(3, 0, 1, 1, &w.Coord.Size.X, 0, "", "w", false, false, true)
		ui.Comp_editbox(4, 0, 1, 1, &w.Coord.Size.Y, 0, "", "h", false, false, true)
	}
	ui.Div_end()
	y++

	for _, it := range w.Values {

		//name_width := ui.Paint_textWidth(name, -1, 0, "", true)

		switch it.Gui_type {

		case "gui_switch":
			//ui.Comp_switch(0, y, 1, 1, &it.Value, 2, "", "", false, false, true)

		case "gui_edit":
			ui.Comp_editbox_desc(it.Name, 0, 2, 0, y, 1, 1, &it.Value, 2, "", "", false, false, true)

		case "gui_combo":
			ui.Comp_combo_desc(it.Name, 0, 2, 0, y, 1, 1, &it.Value, it.Gui_options, "", true, false)
		}

		y++
	}
}

//delete / select(2xclick - goto sub) / resize / move ...

//= expressions + switch between value/formula ...
//server + execute graph ...
//find circles? ...
