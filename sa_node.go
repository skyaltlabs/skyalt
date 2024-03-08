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
	"reflect"
	"sort"
	"strings"

	"github.com/go-audio/audio"
)

const (
	SANode_STATE_WAITING = 0
	SANode_STATE_RUNNING = 1
	SANode_STATE_DONE    = 2
)

type SANode struct {
	app    *SAApp
	parent *SANode

	Pos       OsV2f
	pos_start OsV2f

	Name     string
	Exe      string
	Selected bool `json:",omitempty"`

	Code SANodeCode

	selected_cover  bool
	selected_canvas OsV4

	Attrs map[string]interface{} `json:",omitempty"` //only for Skyalt

	//sub-layout
	Cols []*SANodeColRow `json:",omitempty"`
	Rows []*SANodeColRow `json:",omitempty"`
	Subs []*SANode       `json:",omitempty"`

	//state         int //0=SANode_STATE_WAITING, 1=SANode_STATE_RUNNING, 2=SANode_STATE_DONE
	errExe        error
	progress      float64
	progress_desc string

	z_depth float64

	temp_mic_data audio.IntBuffer
}

func NewSANode(app *SAApp, parent *SANode, name string, exe string, grid OsV4, pos OsV2f) *SANode {
	w := &SANode{}
	w.parent = parent
	w.app = app
	w.Name = name
	w.Exe = exe
	w.Pos = pos

	w.Attrs = make(map[string]interface{})

	if w.CanBeRenderOnCanvas() {
		w.SetGrid(grid)
	}

	w.Code = InitSANodeCode(w)

	return w
}

func NewSANodeRoot(path string, app *SAApp) (*SANode, error) {
	w := NewSANode(app, nil, "root", "layout", OsV4{}, OsV2f{})
	w.Exe = "layout"

	//load
	if path != "" {
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

func (node *SANode) SetError(err error) {
	node.errExe = err
}

func (node *SANode) SetPosStart() {
	node.pos_start = node.Pos
	for _, nd := range node.Subs {
		nd.SetPosStart()
	}
}

func (node *SANode) AddPos(r OsV2f) {
	node.Pos = node.pos_start.Add(r)

	if node.HasNodeSubs() {
		for _, nd := range node.Subs {
			nd.AddPos(r)
		}
	}
}

func (node *SANode) HasAttrNode() bool {
	return node.Exe == "whisper_cpp" || node.Exe == "llamaCpp" || node.Exe == "g4f"
}

func (node *SANode) HasNodeSubs() bool {
	return strings.EqualFold(node.Exe, "layout") || strings.EqualFold(node.Exe, "dialog")
}

func (node *SANode) IsTypeCode() bool {
	return strings.EqualFold(node.Exe, "code_go") || strings.EqualFold(node.Exe, "code_python")
}

func (node *SANode) IsWithChangedAttr() bool {
	//Button and Editbox NOT here, because the have 'clicked' and 'finished'

	grnd := node.app.base.node_groups.FindNode(node.Exe)
	return grnd != nil && grnd.changedAttr
}

func (node *SANode) IsTypeTrigger() bool {
	return node.IsWithChangedAttr() || strings.EqualFold(node.Exe, "button") || strings.EqualFold(node.Exe, "editbox") || strings.EqualFold(node.Exe, "timer")
}

func (node *SANode) IsTriggered() bool {
	if strings.EqualFold(node.Exe, "button") {
		return node.GetAttrBool("clicked", false)
	}
	if strings.EqualFold(node.Exe, "editbox") {
		return node.GetAttrBool("finished", false)
	}
	if strings.EqualFold(node.Exe, "timer") {
		return node.GetAttrBool("done", false)
	}
	if node.IsWithChangedAttr() {
		return node.GetAttrBool("changed", false)
	}

	return false
}

func (node *SANode) ResetTriggers() {
	if strings.EqualFold(node.Exe, "button") {
		node.Attrs["clicked"] = false
	}
	if strings.EqualFold(node.Exe, "editbox") {
		node.Attrs["finished"] = false
	}
	if strings.EqualFold(node.Exe, "timer") {
		node.Attrs["done"] = false
	}
	if node.IsWithChangedAttr() {
		node.Attrs["changed"] = false
	}
}

func (node *SANode) MakeGridSpace(colStart, rowStart, colMove, rowMove int) {

	for _, it := range node.Subs {

		grid := it.GetGrid()
		changed := false

		if grid.Start.X >= colStart {
			grid.Start.X += colMove
			if colMove != 0 {
				changed = true
			}
		}

		if grid.Start.Y >= rowStart {
			grid.Start.Y += rowMove
			if rowMove != 0 {
				changed = true
			}
		}

		if changed {
			it.SetGrid(grid)
			//node.app.SetExecute()
		}
	}
}

func (node *SANode) Save(path string) error {
	if path == "" {
		return nil
	}

	js, err := json.MarshalIndent(node, "", "")
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
	if a.Name != b.Name || a.Exe != b.Exe {
		return false
	}

	if a.Selected != b.Selected {
		*historyDiff = true //no return!
	}

	if !a.Pos.Cmp(b.Pos) {
		*historyDiff = true //no return!
	}

	if len(a.Attrs) != len(b.Attrs) {
		return false
	}
	for i, itA := range a.Attrs {
		itB := b.Attrs[i]
		if !reflect.DeepEqual(itA, itB) {
			//if !itA.Cmp(itB) {
			return false
		}
	}

	if len(a.Cols) == len(b.Cols) {
		for i, itA := range a.Cols {
			if !itA.Cmp(b.Cols[i]) {
				*historyDiff = true //no return!
			}
		}
	} else {
		*historyDiff = true //no return!
	}

	if len(a.Rows) == len(b.Rows) {
		for i, itA := range a.Rows {
			if !itA.Cmp(b.Rows[i]) {
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

func (node *SANode) HasError() bool {
	return node.errExe != nil
}

func (node *SANode) CanBeRenderOnCanvas() bool {
	return node.app.base.node_groups.IsUI(node.Exe) && node.GetGridShow()
}

func (node *SANode) FindNode(name string) *SANode {
	for _, it := range node.Subs {
		if it.Name == name {
			return it
		}
	}

	//parent only, never deeper
	if node.parent != nil {
		return node.parent.FindNode(name)
	}

	return nil
}

func (node *SANode) FindParent(parent *SANode) bool {
	for node != nil {
		if node == parent {
			return true
		}
		node = node.parent
	}
	return false
}

func (node *SANode) updateLinks(parent *SANode, app *SAApp) {
	node.parent = parent
	node.app = app

	if node.Attrs == nil {
		node.Attrs = make(map[string]interface{})
	}

	err := node.Code.updateLinks(node)
	if err != nil {
		fmt.Printf("updateLinks() for node '%s' failed: %v\n", node.Name, err)
	}

	for _, it := range node.Subs {
		it.updateLinks(node, app)
	}
}

func (node *SANode) Copy() (*SANode, error) {
	js, err := json.Marshal(node)
	if err != nil {
		return nil, err
	}

	dst := NewSANode(node.app, nil, "", "", OsV4{}, OsV2f{})
	err = json.Unmarshal(js, dst)
	if err != nil {
		return nil, err
	}
	dst.updateLinks(nil, node.app)

	return dst, nil
}

func (dst *SANode) CopyPoses(src *SANode) {
	dst.Pos = src.Pos
	dst.Cols = src.Cols
	dst.Rows = src.Rows
	dst.Selected = src.Selected

	for _, dstIt := range dst.Subs {
		srcIt := src.FindNode(dstIt.Name)
		if srcIt != nil {
			dstIt.CopyPoses(srcIt)
		}
	}
}

func (node *SANode) SelectOnlyThis() {
	node.GetAbsoluteRoot().DeselectAll()
	node.Selected = true
}

func (node *SANode) SelectAll() {
	node.Selected = true
	for _, n := range node.Subs {
		n.SelectAll()
	}
}

func (node *SANode) NumSelected() int {
	sum := OsTrn(node.Selected, 1, 0)
	for _, nd := range node.Subs {
		sum += nd.NumSelected()
	}
	return sum
}

func (node *SANode) DeselectAll() {
	for _, n := range node.Subs {
		n.Selected = false
		n.DeselectAll()
	}
}

func (node *SANode) isParentSelected() bool {
	node = node.parent
	for node != nil {
		if node.Selected {
			return true
		}
		node = node.parent
	}
	return false
}

func (node *SANode) _buildListOfSelected(list *[]*SANode) {
	for _, n := range node.Subs {
		if n.Selected && !n.isParentSelected() {
			*list = append(*list, n)
		}
		n._buildListOfSelected(list)
	}
}

func (node *SANode) BuildListOfSelected() []*SANode {
	var list []*SANode
	node._buildListOfSelected(&list)
	return list
}

func (node *SANode) FindSelected() *SANode {
	for _, it := range node.Subs {
		if it.Selected {
			return it
		}
	}

	for _, nd := range node.Subs {
		//if nd.IsGuiLayout() {
		fd := nd.FindSelected()
		if fd != nil {
			return fd
		}
		//}
	}

	return nil
}

func (node *SANode) RemoveSelectedNodes() {
	for _, n := range node.Subs {
		n.RemoveSelectedNodes()
	}

	for i := len(node.Subs) - 1; i >= 0; i-- {
		if node.Subs[i].Selected {
			node.Subs = append(node.Subs[:i], node.Subs[i+1:]...) //remove
		} else {
			node.Subs[i].RemoveSelectedNodes() //go deeper
		}
	}
}

func (node *SANode) BypassSelectedCodeNodes() {

	if node.IsTypeCode() {
		node.SetBypass()
	}

	for _, n := range node.Subs {
		n.BypassSelectedCodeNodes()
	}
}

func (node *SANode) GetPath() string {

	var path string

	if node.parent != nil {
		path += node.parent.GetPath()
	}

	path += node.Name + "/"

	return path
}

func (node *SANode) NumAttrNames(name string) int {
	n := 0
	for nm := range node.Attrs {
		if nm == name {
			n++
		}
	}
	return n
}

func (node *SANode) NumSubNames(name string) int {
	n := 0
	for _, nd := range node.Subs {
		if nd.Name == name {
			n++
		}
		n += nd.NumSubNames(name)
	}
	return n
}

func (node *SANode) GetAbsoluteRoot() *SANode {
	for node.parent != nil {
		node = node.parent
	}
	return node
}

func (node *SANode) CheckUniqueName() {
	//check
	if node.Name == "" {
		node.Name = "node"
	}
	node.Name = strings.ReplaceAll(node.Name, ".", "") //remove all '.'

	//set unique
	for node.GetAbsoluteRoot().NumSubNames(node.Name) >= 2 {
		node.Name += "1"
	}

	//check subs as well
	for _, nd := range node.Subs {
		nd.CheckUniqueName()
	}
}

func (node *SANode) AddNode(grid OsV4, pos OsV2f, name string, exe string) *SANode {
	nw := NewSANode(node.app, node, name, exe, grid, pos)
	node.Subs = append(node.Subs, nw)
	nw.CheckUniqueName()
	return nw
}

func (node *SANode) AddNodeCopy(src *SANode) *SANode { // note: 'w' can be root graph
	nw, _ := src.Copy() //err ...
	nw.updateLinks(node, node.app)

	//move Pos
	nw.Pos = nw.Pos.Add(OsV2f{1, 1})
	for _, nd := range nw.Subs {
		nd.Pos = nd.Pos.Add(OsV2f{1, 1})
	}

	node.Subs = append(node.Subs, nw)
	nw.CheckUniqueName()
	return nw
}

func (node *SANode) Remove() bool {
	if node.parent != nil {
		for i, it := range node.parent.Subs {
			if it == node {
				node.parent.Subs = append(node.parent.Subs[:i], node.parent.Subs[i+1:]...)
				return true
			}
		}
	}
	return false
}

func (node *SANode) RenderCanvas() {
	ui := node.app.base.ui

	node.z_depth = 1

	if node.CanBeRenderOnCanvas() {
		gnd := node.app.base.node_groups.FindNode(node.Exe)
		if gnd != nil && gnd.render != nil {
			gnd.render(node)
		}
	}

	if node.app.IDE && node.CanBeRenderOnCanvas() {

		grid := node.GetGrid()
		grid.Size.Y = OsMax(grid.Size.Y, 1)

		//draw Select rectangle
		if node.HasError() { //|| w.HasExpError() {
			pl := ui.win.io.GetPalette()
			cd := pl.E
			cd.A = 150

			//rect
			div := ui.Div_startName(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, ".err.")
			div.touch_enabled = false
			ui.Paint_rect(0, 0, 1, 1, 0, cd, 0)
			ui.Div_end()
		}

		if node.Selected {
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

			node.selected_canvas = div.canvas //copy coord size
		}
	}
}

func (node *SANode) GetResizerCoord() OsV4 {
	s := node.app.base.ui.CellWidth(0.3)
	return InitOsV4Mid(node.selected_canvas.End(), OsV2{s, s})
}

func (node *SANode) renderLayoutCols() {
	SANodeColRow_Check(&node.Cols)

	ui := node.app.base.ui
	for _, it := range node.Cols {
		ui.Div_col(it.Pos, it.Min)
		ui.Div_colMax(it.Pos, it.Max)
		if it.ResizeName != "" {
			active, v := ui.Div_colResize(it.Pos, it.ResizeName, it.Resize, true)
			if active {
				it.Resize = v
			}
		}
	}
}
func (node *SANode) renderLayoutRows() {
	SANodeColRow_Check(&node.Rows)

	ui := node.app.base.ui
	for _, it := range node.Rows {
		ui.Div_row(it.Pos, it.Min)
		ui.Div_rowMax(it.Pos, it.Max)
		if it.ResizeName != "" {
			active, v := ui.Div_rowResize(it.Pos, it.ResizeName, it.Resize, true)
			if active {
				it.Resize = v
			}
		}
	}
}

func (node *SANode) renderLayout() {

	//cols/rows
	node.renderLayoutCols()
	node.renderLayoutRows()

	//sort z-depth
	sort.Slice(node.Subs, func(i, j int) bool {
		return node.Subs[i].z_depth < node.Subs[j].z_depth
	})

	//other items
	for _, it := range node.Subs {
		it.RenderCanvas()
	}
}

func (node *SANode) RenameDepends(oldName string, newName string) {
	node.Code.renameNode(oldName, newName)
}

// use node.GetAbsoluteRoot().RenameSubDepends()
func (node *SANode) RenameSubDepends(oldName string, newName string) {
	//expressions
	for _, it := range node.Subs {
		it.RenameDepends(oldName, newName)
	}

	//Subs
	for _, it := range node.Subs {
		it.RenameSubDepends(oldName, newName)
	}
}
