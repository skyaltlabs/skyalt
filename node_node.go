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
	"fmt"
	"strconv"
)

type NodeParam struct {
	Name  string
	Value string
}

type NodeColRow struct {
	Min, Max, Resize float64 `json:",omitempty"`
	ResizeName       string  `json:",omitempty"`
}

type Node struct {
	parent *Node
	app    *NodeApp

	Name string
	Subs []*Node `json:",omitempty"`

	Cam_x float32 `json:",omitempty"`
	Cam_y float32 `json:",omitempty"`
	Cam_z float32 `json:",omitempty"`

	Grid_x int `json:",omitempty"`
	Grid_y int `json:",omitempty"`
	Grid_w int `json:",omitempty"`
	Grid_h int `json:",omitempty"`

	Cols []NodeColRow `json:",omitempty"`
	Rows []NodeColRow `json:",omitempty"`

	Pos            OsV2f
	pos_start      OsV2f
	Selected       bool
	selected_cover bool

	FnName     string
	Parameters []NodeParam
	Inputs     []NodeIn

	Bypass bool

	outputs []NodeData

	changed bool
	running bool
	done    bool

	err error

	//GridCoord OsV4 //for output/render node

	//info_action   string
	//info_progress float64 //0-1

	db *DiskDb
}

func NewNode(parent *Node, app *NodeApp, name string, fnName string) *Node {
	var node Node
	node.parent = parent
	node.app = app

	node.Name = name
	//node.Parameters = make(map[string]string)
	node.FnName = fnName
	node.changed = true
	//node.GridCoord = InitOsV4(0, 0, 1, 1)

	return &node
}

func (node *Node) Destroy() {
	if node.db != nil {
		node.db.Destroy()
	}

	for _, n := range node.Subs {
		n.Destroy()
	}
}

func (node *Node) SetParentAndApp(parent *Node, app *NodeApp) {
	node.parent = parent
	node.app = app
	for _, n := range node.Subs {
		n.SetParentAndApp(node, app)
	}
}

func (node *Node) KeyProgessSelection(keys *WinKeys) bool {

	if keys.shift {
		if node.selected_cover {
			return true
		}
		return node.Selected
	} else if keys.ctrl {
		if node.selected_cover {
			return false
		}
		return node.Selected
	}

	return node.selected_cover
}

func (node *Node) GetParamString(name string) string {
	//find
	p := node.GetParam(name)
	return *p
}
func (node *Node) GetParamInt(name string) int {
	//find
	p := node.GetParam(name)
	v, _ := strconv.Atoi(*p)
	return v
}
func (node *Node) GetParamFloat(name string) float64 {

	//find
	p := node.GetParam(name)
	v, _ := strconv.ParseFloat(*p, 64)
	return v
}
func (node *Node) GetParamFloat32(name string) float32 {
	return float32(node.GetParamFloat(name))
}

func (node *Node) FindParam(name string) *NodeParam {
	for i, p := range node.Parameters {
		if p.Name == name {
			return &node.Parameters[i]
		}
	}

	return nil
}

func (node *Node) GetParam(name string) *string {
	//find
	f := node.FindParam(name)
	if f != nil {
		return &f.Value
	}

	//add
	p := NodeParam{Name: name}
	node.Parameters = append(node.Parameters, p)
	return node.GetParam(name)
}

func (node *Node) SetParam(name string, value string) {
	//find
	f := node.FindParam(name)
	if f != nil {
		f.Value = value
		return
	}

	//add
	p := NodeParam{Name: name, Value: value}
	node.Parameters = append(node.Parameters, p)
}

func (node *Node) SetParamFloat64(name string, value float64) {
	node.SetParam(name, strconv.FormatFloat(value, 'f', 30, 32))
}

func (node *Node) SetParamFloat32(name string, value float32) {
	node.SetParamFloat64(name, float64(value))
}

func (node *Node) SetParamInit(name string, defValue string) {
	//find
	f := node.FindParam(name)
	if f != nil {
		return
	}

	//add
	p := NodeParam{Name: name, Value: defValue}
	node.Parameters = append(node.Parameters, p)
}

func (node *Node) SetInput(out_pos int, src *Node, src_pos int) {

	for i := len(node.Inputs); i <= out_pos; i++ {
		node.Inputs = append(node.Inputs, NodeIn{})
	}

	node.Inputs[out_pos] = NodeIn{Node: src.Name, Pos: src_pos}
	node.changed = true
}

func (node *Node) RenameInputs(src, dst string) {
	for _, in := range node.Inputs {
		if in.Node == src {
			in.Node = dst
		}
	}
}
func (node *Node) RemoveInputs(name string) {
	for _, in := range node.Inputs {
		if in.Node == name {
			in.Node = ""
		}
	}
}

func (node *Node) FindNode(name string) *Node {
	for _, n := range node.Subs {
		if n.Name == name {
			return n
		}
	}
	return nil
}

func (node *Node) Execute() {
	//function
	fn := node.FindFn(node.FnName)
	if fn == nil {
		node.err = fmt.Errorf("Function(%s) not found", node.FnName)
		return
	}

	//free previous db
	if node.db != nil {
		node.db.Destroy()
		node.db = nil
	}

	//inputs
	var ins []NodeData
	for i, in := range node.Inputs {
		n := node.FindNode(in.Node)
		if n == nil {
			node.err = fmt.Errorf("Node(%s) for %d input not found", in.Node, i+1)
			return
		}

		//resize
		if in.Pos >= len(n.outputs) {
			t := n.outputs
			n.outputs = make([]NodeData, in.Pos+1)
			copy(n.outputs, t)
		}

		//set
		ins = append(ins, n.outputs[in.Pos])
	}

	//call
	if node.Bypass {
		//copy inputs to outputs
		node.outputs = make([]NodeData, len(ins))
		copy(node.outputs, ins)
	} else {
		node.outputs, node.err = fn.exe(ins, node)
		if node.err != nil {
			fmt.Printf("Node(%s) has error(%v)\n", node.Name, node.err)
		}
	}

	if !node.app.IsRunning() {
		node.err = fmt.Errorf("Interrupted")
		return
	}

	if node.db != nil {
		node.db.Commit()
	}

	node.changed = true //child can update
	node.done = true
	node.running = false
}

func (node *Node) areInputsErrorFree() bool {

	for i, in := range node.Inputs {
		n := node.FindNode(in.Node)
		if n == nil {
			node.err = fmt.Errorf("Node(%s) for %d input not found", in.Node, i+1)
			return false
		}

		if n.err != nil {
			n.err = fmt.Errorf("incomming error from %d input", i+1)
			return false
		}
	}

	return true
}

func (node *Node) areInputsReadyToRun() bool {

	for _, in := range node.Inputs {
		n := node.FindNode(in.Node)
		if n == nil || n.running {
			return false
		}
	}

	return true
}

func (node *Node) areInputsDone() bool {
	for _, in := range node.Inputs {
		n := node.FindNode(in.Node)
		if n == nil || !n.done {
			return false
		}
	}
	return true
}

func (node *Node) isInputsChanged() bool {
	for _, in := range node.Inputs {
		n := node.FindNode(in.Node)
		if n == nil || n.changed {
			return true
		}
	}
	return false
}

func (node *Node) getUniqueName(name string) string {

	if node.FindNode(name) == nil {
		return name
	}

	i := 1
	for {
		nm := fmt.Sprintf("%s_%d", name, i)
		if node.FindNode(nm) == nil {
			return nm
		}
		i++
	}
}

func (node *Node) AddNode(name string, fnDef *NodeFnDef) *Node {

	if name == "" {
		name = node.getUniqueName(fnDef.name)
	}

	n := node.FindNode(name)
	if n != nil {
		return nil //already exist
	}

	n = NewNode(node, node.app, name, fnDef.name)
	if fnDef.init != nil {
		fnDef.init(n)
	}
	node.Subs = append(node.Subs, n)
	return n
}

func (node *Node) RemoveNode(name string) bool {

	found := false
	for i, n := range node.Subs {
		if n.Name == name {
			node.Subs = append(node.Subs[:i], node.Subs[i+1:]...)
			found = true
			break
		}
	}

	for _, n := range node.Subs {
		n.RemoveInputs(name)
	}

	return found
}

func (node *Node) RenameNode(src, dst string) bool {

	n := node.FindNode(dst)
	if n != nil {
		return false //newName already exist
	}

	found := false
	for _, n := range node.Subs {
		if n.Name == src {
			n.Name = dst
		}
		n.RenameInputs(src, dst)
	}

	return found
}

func (node *Node) FindFn(name string) *NodeFnDef {
	return node.app.FindFn(name)
}
