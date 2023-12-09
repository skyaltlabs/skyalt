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
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Node struct {
	parent *Node

	FnName   string
	Id       int
	Bypass   bool
	Pos      OsV2f
	Selected bool

	Subs    []*Node
	Attrs   []*NodeParamOut
	Inputs  []*NodeParamIn
	outputs []*NodeParamOut

	pos_start      OsV2f
	selected_cover bool

	changed bool
	running bool
	done    bool

	err error
}

func NewNode(parent *Node, fnName string) *Node {
	var node Node
	node.parent = parent
	node.FnName = fnName

	if parent != nil {
		node.Id = parent.getUniqueId()
	} else {
		node.Id = 1
	}
	node.changed = true

	return &node
}

func (node *Node) Destroy() {
	for _, n := range node.Subs {
		n.Destroy()
	}
}

func (node *Node) getPath() string {

	var path string

	if node.parent != nil {
		path += node.parent.getPath()
	}

	path += node.GetAttr("name").Value + "/"

	return path
}

func (node *Node) IsGuiSub() bool {
	return node.FnName == "gui_sub"
}

func (a *Node) FindMirror(b *Node, b_act *Node) *Node {

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

func (src *Node) Copy() (*Node, error) {

	js, err := json.Marshal(src)
	if err != nil {
		return nil, err
	}

	dst := &Node{}
	err = json.Unmarshal(js, dst)
	if err != nil {
		return nil, err
	}

	dst.UpdateParents(nil)

	return dst, nil
}

func (a *Node) Cmp(b *Node) bool {

	if a.Id != b.Id || a.Bypass != b.Bypass || a.Pos.X != b.Pos.X || a.Pos.Y != b.Pos.Y || a.Selected != b.Selected || a.FnName != b.FnName {
		return false
	}

	if len(a.Subs) != len(b.Subs) || len(a.Attrs) != len(b.Attrs) || len(a.Inputs) != len(b.Inputs) {
		return false
	}

	for i, itA := range a.Attrs {
		itB := b.Attrs[i]
		if itA.Name != itB.Name || itA.Value != itB.Value || itA.Gui_type != itB.Gui_type || itA.Gui_options != itB.Gui_options {
			return false
		}
	}

	for i, itA := range a.Inputs {
		itB := b.Inputs[i]

		if itA.Name != itB.Name || itA.Value != itB.Value || itA.Gui_type != itB.Gui_type || itA.Gui_options != itB.Gui_options {
			return false
		}
		if itA.Wire_id != itB.Wire_id || itA.Wire_param != itB.Wire_param {
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

func (node *Node) SetChanged() {
	node.changed = true
}

func (node *Node) UpdateParents(parent *Node) {
	node.parent = parent
	node.SetChanged()

	for _, it := range node.Attrs {
		it.node = node
	}
	for _, it := range node.Inputs {
		it.node = node
	}
	for _, it := range node.outputs {
		it.node = node
	}

	for _, n := range node.Subs {
		n.UpdateParents(node)
	}
}

func (node *Node) getUniqueId() int {

	max := 0
	for _, n := range node.Subs {
		max = OsMax(max, n.Id)
	}
	return max + 1
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

func (node *Node) FindAttr(name string) *NodeParamOut {
	for _, attr := range node.Attrs {
		if attr.Name == name {
			return attr
		}
	}
	return nil
}
func (node *Node) FindInput(name string) *NodeParamIn {
	for _, in := range node.Inputs {
		if in.Name == name {
			return in
		}
	}
	return nil
}
func (node *Node) FindOutput(name string) *NodeParamOut {
	for _, out := range node.outputs {
		if out.Name == name {
			return out
		}
	}
	return nil
}

func (node *Node) GetAttr(name string) *NodeParamOut {
	attr := node.FindAttr(name)
	if attr == nil {
		//add
		attr = &NodeParamOut{Name: name, node: node}
		node.Attrs = append(node.Attrs, attr)
	}
	return attr
}

func (node *Node) GetInput(name string) *NodeParamIn {
	in := node.FindInput(name)
	if in == nil {
		//add
		in = &NodeParamIn{Name: name, node: node}
		node.Inputs = append(node.Inputs, in)
	}
	return in
}

func (node *Node) GetOutput(name string) *NodeParamOut {
	out := node.FindOutput(name)
	if out == nil {
		//add
		out = &NodeParamOut{Name: name, node: node}
		node.outputs = append(node.outputs, out)
	}
	return out
}

func (node *Node) GetInputString(name string) string {
	in := node.GetInput(name)

	out := in.FindWireOut()
	if out != nil {
		return out.Value
	}

	return in.Value
}
func (node *Node) GetInputInt(name string) int {
	v, _ := strconv.Atoi(node.GetInputString(name))
	return v
}

func (node *Node) FindNode(id int) *Node {
	for _, n := range node.Subs {
		if n.Id == id {
			return n
		}
	}
	return nil
}

func (node *Node) AddNodePtr(n *Node) {
	n.UpdateParents(node)
	node.Subs = append(node.Subs, n)
}

func (node *Node) AddNode(fnName string) *Node {
	n := NewNode(node, fnName)
	node.AddNodePtr(n)
	return n
}

func (node *Node) RemoveNode(id int) bool {

	if id == 1 {
		return false //can't delete main
	}

	found := false
	for i, n := range node.Subs {
		if n.Id == id {
			node.Subs = append(node.Subs[:i], node.Subs[i+1:]...)
			found = true
			break
		}
	}

	for _, n := range node.Subs {
		n.RemoveInputs(id)
	}

	return found
}

func (node *Node) RemoveInputs(id int) {
	for _, in := range node.Inputs {
		if in.Wire_id == id {
			in.Wire_id = 0
		}
	}
}

func (node *Node) IsExecuteIgnore() bool {
	return strings.HasPrefix(node.FnName, "gui_") || node.FnName == "constant"
}

func (node *Node) ExecuteConstant() {
	node.GetAttr("value")
}

func (node *Node) Execute(server *NodeServer) {

	node.err = nil

	//call
	if node.Bypass {
		//copy inputs to outputs??? ...
		//node.outputs = make([]NodeData, len(ins))
		//copy(node.outputs, ins)
	} else {

		if !node.IsExecuteIgnore() {
			nc := server.Start(node.FnName)
			if nc != nil {

				//add/update new
				for _, in := range nc.strct.Attrs {
					a := node.GetAttr(in.Name)
					a.Gui_type = in.Gui_type
					a.Gui_options = in.Gui_options
				}
				for _, in := range nc.strct.Inputs {
					i := node.GetInput(in.Name)
					i.Gui_type = in.Gui_type
					i.Gui_options = in.Gui_options
				}

				//set/remove
				for i := len(node.Attrs) - 1; i >= 0; i-- {
					src := node.Attrs[i]
					dst := nc.FindAttr(src.Name)
					if dst != nil {
						dst.Value = src.Value
					} else {
						node.Attrs = append(node.Attrs[:i], node.Attrs[i+1:]...) //remove
					}
				}
				for i := len(node.Inputs) - 1; i >= 0; i-- {
					src := node.Inputs[i]
					dst := nc.FindInput(src.Name)
					if dst != nil {
						out := src.FindWireOut()
						if out != nil {
							dst.Value = out.Value
						} else {
							dst.Value = src.Value
						}
					} else {
						node.Inputs = append(node.Inputs[:i], node.Inputs[i+1:]...) //remove
					}
				}

				//execute
				nc.Start()

				//copy back
				node.outputs = nil //reset
				for _, in := range nc.strct.Outputs {
					o := node.GetOutput(in.Name) //add
					o.Value = in.Value
					o.Gui_type = in.Gui_type
					o.Gui_options = in.Gui_options
				}

				if nc.progress.Error != "" {
					node.err = errors.New(nc.progress.Error)
				}
			} else {
				node.err = fmt.Errorf("can't find node exe(%s)", node.FnName)
			}
		}

		if node.err != nil {
			fmt.Printf("Node(%d) has error(%v)\n", node.Id, node.err)
		}
	}

	if !server.IsRunning() {
		node.err = fmt.Errorf("Interrupted")
		return
	}

	node.SetChanged() //child can update
	node.done = true
	node.running = false
}

func (node *Node) ExecuteSubs(server *NodeServer, max_threads int) {

	//prepare
	for _, n := range node.Subs {
		n.done = false
		n.running = false

		if n.FnName == "constant" {
			n.ExecuteConstant()
		}
	}

	//multi-thread executing
	var numActiveThreads atomic.Int64
	var wg sync.WaitGroup
	run := true
	for run && server.IsRunning() {
		run = false
		for _, n := range node.Subs {

			if !n.done {
				run = true

				if !n.running {
					if n.areInputsErrorFree() {
						if n.areInputsReadyToRun() {
							if n.areInputsDone() {

								if n.changed || n.isInputsChanged() { //|| n.err != nil {

									//maximum concurent threads
									if numActiveThreads.Load() >= int64(max_threads) {
										time.Sleep(10 * time.Millisecond)
									}

									//run it
									n.running = true
									wg.Add(1)
									go func(nn *Node) {
										numActiveThreads.Add(1)
										nn.Execute(server)
										wg.Done()
										numActiveThreads.Add(-1)
									}(n)
								} //else {
								n.done = true
								//}
							}
						}
					} else {
						n.done = true
					}
				}
			}
		}
	}

	wg.Wait()

	if server.IsRunning() {
		for _, n := range node.Subs {
			n.changed = false
		}

		//app.Maintenance()
	}
}

func (node *Node) areInputsErrorFree() bool {

	for _, in := range node.Inputs {
		out := in.FindWireOut()
		if out != nil {
			if out.node.err != nil {
				in.node.err = fmt.Errorf("incomming error from input(%s)", in.Name)
				return false
			}
		}
	}

	return true
}

func (node *Node) areInputsReadyToRun() bool {

	for _, in := range node.Inputs {
		out := in.FindWireOut()
		if out != nil && out.node.running {
			return false //still running
		}
	}

	return true
}

func (node *Node) areInputsDone() bool {

	for _, in := range node.Inputs {
		out := in.FindWireOut()
		if out != nil && !out.node.done {
			return false //not finished
		}
	}

	return true
}

func (node *Node) isInputsChanged() bool {

	for _, in := range node.Inputs {
		out := in.FindWireOut()
		if out != nil && out.node.changed {
			return true //changed
		}
	}

	return false
}
