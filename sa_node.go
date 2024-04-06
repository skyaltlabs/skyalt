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

	listSubs []*SANode

	ShowCodeChat bool

	errExe        error
	progress      float64 //? ......
	progress_desc string  //? ......

	z_depth float64

	temp_mic_data audio.IntBuffer

	db_time DiskDbTime
}

func NewSANode(app *SAApp, parent *SANode, name string, exe string, grid OsV4, pos OsV2f) *SANode {
	node := &SANode{}
	node.parent = parent
	node.app = app
	node.Name = name
	node.Exe = exe
	node.Pos = pos

	node.Attrs = make(map[string]interface{})

	if node.CanBeRenderOnCanvas() {
		node.SetGrid(grid)
	}

	node.Code = InitSANodeCode(node)

	return node
}

func NewSANodeRoot(path string, app *SAApp) (*SANode, *SANode, error) {
	node := NewSANode(app, nil, "root", "layout", OsV4{}, OsV2f{})
	node.Exe = "layout"

	//load
	if path != "" {
		js, err := os.ReadFile(path)
		if err == nil {
			err = json.Unmarshal([]byte(js), node)
			if err != nil {
				fmt.Printf("Unmarshal(%s) failed: %v\n", path, err)
			}
		}
		node.updateLinks(nil, app)
		node.updateCodeLinks()
	}

	exe := node.FindNode("exe")
	if exe == nil {
		exe = node.AddNode(OsV4{}, OsV2f{}, "exe", "exe")
	}

	return node, exe, nil
}

func (node *SANode) SetError(err error) {
	node.errExe = err
}

func (node *SANode) addIntoDependedCodes(changedNode *SANode, prms []SANodeCodeExePrm) {
	for _, nd := range node.app.exe.Subs {
		if nd.IsTypeCode() && !nd.IsBypassed() && nd != changedNode {
			if nd.Code.findFuncDepend(changedNode) {
				nd.Code.AddExe(prms)
			}
		}
	}
}

func (node *SANode) SetChange(prms []SANodeCodeExePrm) {

	//convert copySubs
	if node.parent != nil && node.parent.parent != nil && node.parent.parent.IsTypeList() {
		for i := range prms {
			prms[i].Node = node.parent.parent.Name //Copy
			prms[i].ListNode = node.Name
			prms[i].ListPos = node.parent.parent.FindCopyPos(node.parent) //'node.parent' = layout
		}
	}

	//from sub -> copy node(aka base)
	for node.parent != nil && node.parent.parent != nil {
		node = node.parent
	}

	node.GetParentRoot().addIntoDependedCodes(node, prms)
}

func (node *SANode) SetStructChange() {

	node.SetChange(nil) //exe depending

	for node != nil {
		node.Code.AddExe(nil) //exe this and parents
		node = node.parent
	}
}

func (node *SANode) FindCopyPos(find *SANode) int {
	for i, nd := range node.listSubs {
		if nd == find {
			return i
		}
	}
	return -1
}

func (node *SANode) ResetDbs() {
	node.db_time = DiskDbTime{}
	for _, nd := range node.Subs {
		nd.ResetDbs()
	}
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

func (node *SANode) IsAttrDBValue() bool {
	_, found := node.Attrs["db_value"]
	if found {
		return node.GetAttrBool("db_value", false)
	}
	return false
}

func (node *SANode) IsTypeWhispercpp() bool {
	return node.Exe == "whispercpp"
}
func (node *SANode) IsTypeLLamacpp() bool {
	return node.Exe == "llamacpp"
}
func (node *SANode) IsTypeOpenAI() bool {
	return node.Exe == "openai"
}
func (node *SANode) IsTypeNet() bool {
	return node.Exe == "net"
}

func (node *SANode) IsTypeTables() bool {
	return strings.EqualFold(node.Exe, "tables")
}

func (node *SANode) IsTypeList() bool {
	return strings.EqualFold(node.Exe, "list")
}

func (node *SANode) IsTypeCode() bool {
	return strings.EqualFold(node.Exe, "code") || node.IsTypeList()
}

func (node *SANode) HasNodeSubs() bool {
	return strings.EqualFold(node.Exe, "layout") || strings.EqualFold(node.Exe, "dialog") || strings.EqualFold(node.Exe, "exe") || node.IsTypeList()
}

func (node *SANode) HasAttrNode() bool {
	return node.Exe == "whispercpp" || node.Exe == "llamacpp" || node.Exe == "openai" || node.Exe == "net"
}

func (node *SANode) IsTypeWithAttrLabel() bool {
	return strings.EqualFold(node.Exe, "text") || strings.EqualFold(node.Exe, "button") //|| strings.EqualFold(node.Exe, "checkbox") || strings.EqualFold(node.Exe, "switch")
}

func (node *SANode) IsBypassed() bool {
	return node.IsTypeCode() && node.GetAttrBool("bypass", false)
}

func (node *SANode) SetBypass() {
	if node.IsTypeCode() {
		node.Attrs["bypass"] = !node.GetAttrBool("bypass", false)
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

func (a *SANode) CmpListSub(b *SANode) bool {
	if a.Name != b.Name || a.Exe != b.Exe {
		return false
	}

	if len(a.Attrs) != len(b.Attrs) {
		return false
	}
	for key, itA := range a.Attrs {
		if strings.HasPrefix(key, "grid_") {
			continue
		}
		//if !reflect.DeepEqual(itA, itB) {
		if fmt.Sprintf("%v", itA) != fmt.Sprintf("%v", b.Attrs[key]) {
			return false
		}
	}

	if len(a.Subs) != len(b.Subs) {
		return false
	}
	for i, itA := range a.Subs {
		if !itA.CmpListSub(b.Subs[i]) {
			return false
		}
	}

	return true
}

/*func (a *SANode) Cmp(b *SANode) bool {
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
		if !itA.Cmp(b.Subs[i]) {
			return false
		}
	}

	return true
}*/

func (node *SANode) HasError() bool {
	return node.errExe != nil
}

func (node *SANode) CanBeRenderOnCanvas() bool {

	if node.GetGridShow() {
		gr := node.app.base.node_groups.FindNode(node.Exe)
		return gr != nil && gr.render != nil
	}
	return false
}

func (node *SANode) FindNodeOrig(name string) *SANode {
	for _, it := range node.Subs {
		if it.Name == name {
			return it
		}
	}

	//parent only, never deeper
	if node.parent != nil {
		return node.parent.FindNodeOrig(name)
	}

	return nil
}

func (node *SANode) _findNodeInner(name string) *SANode {
	if node.Name == name {
		return node
	}

	//subs
	for _, it := range node.Subs {
		nd := it._findNodeInner(name)
		if nd != nil {
			return nd
		}
	}

	return nil
}

func (node *SANode) FindNodeSubs(name string) *SANode {
	nd := node._findNodeInner(name)
	if nd == nil {
		fmt.Printf("Node '%s' not found ", name)
	}
	return nd
}

func (node *SANode) FindNode(name string) *SANode {
	nd := node.GetParentRoot()._findNodeInner(name)
	if nd == nil {
		fmt.Printf("Node '%s' not found ", name)
	}
	return nd
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

	for _, it := range node.Subs {
		it.updateLinks(node, app)
	}
}
func (node *SANode) updateCodeLinks() {
	node.Code.UpdateLinks(node)
	for _, it := range node.Subs {
		it.updateCodeLinks()
	}
}

func (node *SANode) Copy(updateCode bool) (*SANode, error) {
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
	if updateCode {
		dst.updateCodeLinks()
	}

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
	node.GetParentRoot().DeselectAll()
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
	if node.IsTypeCode() && node.Selected {
		node.SetBypass()
	}

	for _, n := range node.Subs {
		n.BypassSelectedCodeNodes()
	}
}

/*func (node *SANode) GetPath() string {

	var path string

	if node.parent != nil {
		path += node.parent.GetPath()
	}

	path += node.Name + "/"

	return path
}*/

/*func (node *SANode) GetPathSplit() string {

	var path string

	if node.parent != nil && node.parent.parent != nil {
		path += node.parent.GetPathSplit() + "___"
	}

	path += node.Name

	return path
}*/

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

func (node *SANode) GetParentRoot() *SANode {
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
	for node.GetParentRoot().NumSubNames(node.Name) >= 2 {
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
	nw, _ := src.Copy(true) //err ...
	nw.updateLinks(node, node.app)
	nw.updateCodeLinks()

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

			//rect
			div := ui.Div_startName(grid.Start.X, grid.Start.Y, grid.Size.X, grid.Size.Y, ".err.")
			div.touch_enabled = false
			ui.Paint_rect(0, 0, 1, 1, 0.06, pl.OnE, 0.06)
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
	node.Code.RenameNode(oldName, newName)
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
